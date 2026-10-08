package controller

import (
	"errors"
	"fmt"
	"joiask-backend/internal/database"
	"joiask-backend/internal/mailer"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	emailPurposeBind  = "bind"
	emailPurposeReset = "reset"
)

var errEmailLinkInvalid = errors.New("邮件链接已失效或已使用，请重新请求")
var errEmailRateLimited = errors.New("请求过于频繁，请稍后重试")

type emailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type emailConfirmRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func enabledMailSettings(c *gin.Context) (database.Config, mailer.Settings, bool) {
	var config database.Config
	if err := database.DB.First(&config).Error; err != nil {
		Fail(c, 503, "邮件服务暂不可用")
		return config, mailer.Settings{}, false
	}
	if !config.EmailEnabled {
		Fail(c, 404, "邮件功能未开启")
		return config, mailer.Settings{}, false
	}
	settings := mailSettings(config)
	if err := settings.Validate(); err != nil {
		Fail(c, 503, "邮件服务配置不完整，请联系管理员")
		return config, settings, false
	}
	return config, settings, true
}

// Shared, persistent limits apply even to unknown reset addresses. Atomic updates
// bound attempts across concurrent requests and multiple backend instances.
func emailRateLimit(keys ...string) error {
	now := time.Now().UTC()
	return database.DB.Transaction(func(tx *gorm.DB) error {
		for i, key := range keys {
			limit, window := 1, time.Minute
			if i == 0 {
				limit = 5
			}
			if strings.HasPrefix(key, "daily:") {
				limit, window = 10, 24*time.Hour
			}
			bucket := database.EmailRateLimit{Key: hashVerificationToken(key), WindowStart: now}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&bucket).Error; err != nil {
				return err
			}
			if err := tx.Model(&database.EmailRateLimit{}).Where("`key` = ? AND window_start <= ?", bucket.Key, now.Add(-window)).Updates(map[string]any{"window_start": now, "count": 0}).Error; err != nil {
				return err
			}
			result := tx.Model(&database.EmailRateLimit{}).Where("`key` = ? AND count < ?", bucket.Key, limit).UpdateColumn("count", gorm.Expr("count + 1"))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errEmailRateLimited
			}
		}
		return nil
	})
}

func checkEmailRate(c *gin.Context, identity string) bool {
	err := emailRateLimit("ip:"+c.ClientIP(), "recipient:"+identity, "daily:"+identity)
	if err == nil {
		return true
	}
	if errors.Is(err, errEmailRateLimited) {
		Fail(c, 429, err.Error())
	} else {
		Fail(c, 500, "内部错误")
	}
	return false
}

func emailValue(user database.User) string {
	if user.Email == nil {
		return ""
	}
	return *user.Email
}

func (ctl *AccountController) sendAccountEmail(c *gin.Context, config database.Config, settings mailer.Settings, user database.User, email, purpose string) error {
	token, digest, err := newVerificationToken()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	request := database.EmailToken{BilibiliUID: user.BilibiliUID, Purpose: purpose, Email: email, PreviousEmail: emailValue(user), PasswordHash: user.PasswordHash, CredentialHash: digest, ExpiresAt: now.Add(30 * time.Minute)}
	// Prune old credentials and limiter buckets without retaining submitted addresses forever.
	if err := database.DB.Where("expires_at < ?", now.Add(-24*time.Hour)).Delete(&database.EmailToken{}).Error; err != nil {
		return err
	}
	if err := database.DB.Where("window_start < ?", now.Add(-48*time.Hour)).Delete(&database.EmailRateLimit{}).Error; err != nil {
		return err
	}
	if err := database.DB.Create(&request).Error; err != nil {
		return err
	}
	subject, action := "确认绑定邮箱", "确认绑定邮箱"
	if purpose == emailPurposeReset {
		subject, action = "重置密码", "重新设置密码"
	}
	body := fmt.Sprintf("请打开以下链接%s：\n\n%s\n\n链接在 30 分钟内有效，且只能使用一次。\n如果你没有发起此操作，请忽略此邮件。", action, settings.Link(purpose, token))
	sender := ctl.MailSender
	if sender == nil {
		sender = mailer.SMTPSender{}
	}
	if err := sender.Send(c.Request.Context(), settings, email, config.SiteName+" - "+subject, body); err != nil {
		database.DB.Delete(&request)
		return err
	}
	return nil
}

func (ctl *AccountController) RequestEmailBinding(c *gin.Context) {
	config, settings, ok := enabledMailSettings(c)
	if !ok {
		return
	}
	user, ok := currentMember(c)
	if !ok {
		Fail(c, 408, "请先登录")
		return
	}
	if user.Email != nil {
		Fail(c, 409, "账号已绑定邮箱，不支持更换")
		return
	}
	var body emailRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	email, err := mailer.NormalizeEmail(body.Email)
	if err != nil {
		Fail(c, 400, err.Error())
		return
	}
	if !checkEmailRate(c, email) {
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)) != nil {
		Fail(c, 403, "当前密码错误")
		return
	}
	var count int64
	if err := database.DB.Model(&database.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	if count > 0 {
		Fail(c, 409, "该邮箱已被绑定")
		return
	}
	if err := ctl.sendAccountEmail(c, config, settings, user, email, emailPurposeBind); err != nil {
		log.Warn("failed to send email binding message")
		Fail(c, 502, "邮件发送失败，请稍后重试或联系管理员")
		return
	}
	Success(c, gin.H{"message": "确认邮件已发送，请打开邮件中的链接完成绑定"})
}

func (ctl *AccountController) RequestPasswordReset(c *gin.Context) {
	config, settings, ok := enabledMailSettings(c)
	if !ok {
		return
	}
	var body emailRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	email, err := mailer.NormalizeEmail(body.Email)
	if err != nil {
		Fail(c, 400, err.Error())
		return
	}
	if !checkEmailRate(c, email) {
		return
	}
	var user database.User
	err = database.DB.Where("email = ? AND is_disabled = ?", email, false).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Fail(c, 500, "内部错误")
		return
	}
	if err == nil {
		// Do not disclose whether an address is registered, disabled, or failed to deliver.
		if err := ctl.sendAccountEmail(c, config, settings, user, email, emailPurposeReset); err != nil {
			log.Warn("failed to send password reset message")
		}
	}
	Success(c, gin.H{"message": "如果该邮箱已绑定可用账号，你将收到重置密码邮件，请检查收件箱和垃圾邮件"})
}

func consumeEmailToken(raw, purpose string, memberUID int64, passwordHash string) (database.User, error) {
	var user database.User
	if len(raw) != 64 {
		return user, errEmailLinkInvalid
	}
	now := time.Now().UTC()
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var token database.EmailToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("credential_hash = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > ?", hashVerificationToken(raw), purpose, now).First(&token).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errEmailLinkInvalid
			}
			return err
		}
		if purpose == emailPurposeBind && token.BilibiliUID != memberUID {
			return errors.New("请登录发起绑定的账号后再确认")
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, token.BilibiliUID).Error; err != nil {
			return errEmailLinkInvalid
		}
		if purpose == emailPurposeBind && (user.Email != nil || token.PreviousEmail != "") {
			return errEmailLinkInvalid
		}
		if user.IsDisabled || token.PasswordHash != user.PasswordHash || token.PreviousEmail != emailValue(user) {
			return errEmailLinkInvalid
		}
		if purpose == emailPurposeReset && emailValue(user) != token.Email {
			return errEmailLinkInvalid
		}
		result := tx.Model(&token).Where("consumed_at IS NULL AND expires_at > ?", now).Update("consumed_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errEmailLinkInvalid
		}
		updates := map[string]any{}
		if purpose == emailPurposeBind {
			updates["email"] = token.Email
		} else {
			updates["password_hash"] = passwordHash
			updates["session_version"] = gorm.Expr("session_version + 1")
		}
		// Snapshot conditions guard account changes even on databases without row locks.
		query := tx.Model(&user).Where("password_hash = ? AND is_disabled = ?", token.PasswordHash, false)
		if token.PreviousEmail == "" {
			query = query.Where("email IS NULL")
		} else {
			query = query.Where("email = ?", token.PreviousEmail)
		}
		result = query.Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errEmailLinkInvalid
		}
		if purpose == emailPurposeBind {
			user.Email = &token.Email
		}
		return nil
	})
	return user, err
}

func (*AccountController) ConfirmEmailBinding(c *gin.Context) {
	if _, _, ok := enabledMailSettings(c); !ok {
		return
	}
	user, ok := currentMember(c)
	if !ok {
		Fail(c, 408, "请先登录发起绑定的账号")
		return
	}
	var body emailConfirmRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	updated, err := consumeEmailToken(body.Token, emailPurposeBind, user.BilibiliUID, "")
	if err != nil {
		Fail(c, 400, "绑定失败，链接可能已失效、已使用或邮箱已被绑定，请登录原账号重试")
		return
	}
	Success(c, publicUser(updated))
}

func (*AccountController) ResetPassword(c *gin.Context) {
	if _, _, ok := enabledMailSettings(c); !ok {
		return
	}
	var body emailConfirmRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	if utf8.RuneCountInString(body.Password) < 8 || len([]byte(body.Password)) > 72 {
		Fail(c, 400, "密码至少需要 8 个字符，且不能超过 72 字节")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	if _, err := consumeEmailToken(body.Token, emailPurposeReset, 0, string(hash)); err != nil {
		Fail(c, 400, errEmailLinkInvalid.Error())
		return
	}
	Success(c, gin.H{"message": "密码已重置，请使用新密码登录"})
}
