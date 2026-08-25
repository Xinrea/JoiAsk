package controller

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"joiask-backend/internal/avatar"
	"joiask-backend/internal/database"
	"joiask-backend/internal/secret"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	memberSessionKey              = "member_bilibili_uid"
	verificationRequestSessionKey = "bilibili_verification_request_id"
	verificationTokenSessionKey   = "bilibili_verification_token"
)

type AvatarStorage interface {
	Save(context.Context, int64, string) (string, error)
	Delete(string) error
}

type AccountController struct {
	AvatarStorage AvatarStorage
}

var startVerificationMutex sync.Mutex

type verificationStartRequest struct {
	BilibiliUID string `json:"bilibili_uid"`
}

type accountLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type accountRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func normalizeBilibiliUID(value string) (int64, string, error) {
	value = strings.TrimSpace(value)
	uid, err := strconv.ParseInt(value, 10, 64)
	if err != nil || uid <= 0 {
		return 0, "", errors.New("B 站 UID 无效")
	}
	return uid, strconv.FormatInt(uid, 10), nil
}

func newVerificationToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw)
	return token, hashVerificationToken(token), nil
}

func hashVerificationToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func publicUser(user database.User) gin.H {
	return gin.H{
		"username":        user.Username,
		"bilibili_uid":    strconv.FormatInt(user.BilibiliUID, 10),
		"bilibili_name":   user.BilibiliName,
		"bilibili_avatar": user.BilibiliAvatar,
		"verified_at":     user.VerifiedAt,
		"is_disabled":     user.IsDisabled,
		"created_at":      user.CreatedAt,
		"updated_at":      user.UpdatedAt,
	}
}

func (*AccountController) StartVerification(c *gin.Context) {
	var body verificationStartRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	uid, _, err := normalizeBilibiliUID(body.BilibiliUID)
	if err != nil {
		Fail(c, 400, err.Error())
		return
	}
	var account database.BilibiliVerificationAccount
	if err := database.DB.First(&account).Error; err != nil {
		Fail(c, 503, "注册验证暂未配置")
		return
	}
	if _, err := secret.DecryptCookie(account.EncryptedCookie); err != nil {
		Fail(c, 503, "注册验证暂不可用，请联系管理员")
		return
	}
	var user database.User
	if database.DB.Where("bilibili_uid = ?", uid).First(&user).Error == nil {
		Fail(c, 409, "该 B 站 UID 已注册，请直接登录")
		return
	}
	now := time.Now().UTC().Truncate(time.Second)
	startVerificationMutex.Lock()
	defer startVerificationMutex.Unlock()
	var active database.BilibiliVerificationRequest
	if err := database.DB.Where("bilibili_uid = ? AND ((status = ? AND expires_at >= ?) OR (status = ? AND confirmation_until >= ?))",
		uid, database.VerificationPending, now, database.VerificationVerified, now).First(&active).Error; err == nil {
		Fail(c, 409, "该 UID 正在验证中，请稍后再试")
		return
	}
	token, credentialHash, err := newVerificationToken()
	if err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	request := database.BilibiliVerificationRequest{
		BilibiliUID:    uid,
		CredentialHash: credentialHash,
		Status:         database.VerificationPending,
		RequestedAt:    now,
		ExpiresAt:      now.Add(3 * time.Minute),
	}
	if err := database.DB.Create(&request).Error; err != nil {
		Fail(c, 500, "创建验证请求失败")
		return
	}
	session := sessions.Default(c)
	session.Set(verificationRequestSessionKey, request.ID)
	session.Set(verificationTokenSessionKey, token)
	if err := session.Save(); err != nil {
		database.DB.Delete(&request)
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, gin.H{
		"status":       request.Status,
		"target_uid":   strconv.FormatInt(account.BilibiliUID, 10),
		"requested_at": request.RequestedAt,
		"expires_at":   request.ExpiresAt,
	})
}

func loadSessionVerification(c *gin.Context) (database.BilibiliVerificationRequest, error) {
	session := sessions.Default(c)
	id := session.Get(verificationRequestSessionKey)
	token, ok := session.Get(verificationTokenSessionKey).(string)
	if id == nil || !ok || token == "" {
		return database.BilibiliVerificationRequest{}, gorm.ErrRecordNotFound
	}
	var request database.BilibiliVerificationRequest
	if err := database.DB.First(&request, id).Error; err != nil {
		return request, err
	}
	if request.CredentialHash != hashVerificationToken(token) {
		return request, errors.New("验证凭据无效")
	}
	return request, nil
}

func (*AccountController) VerificationStatus(c *gin.Context) {
	request, err := loadSessionVerification(c)
	if err != nil {
		Fail(c, 404, "没有进行中的验证")
		return
	}
	now := time.Now().UTC()
	status := request.Status
	if status == database.VerificationPending && now.After(request.ExpiresAt) {
		status = database.VerificationExpired
	}
	if status == database.VerificationVerified && (request.ConfirmationUntil == nil || now.After(*request.ConfirmationUntil)) {
		status = database.VerificationExpired
		database.DB.Model(&request).Where("status = ?", database.VerificationVerified).Update("status", database.VerificationExpired)
	}
	data := gin.H{
		"status":       status,
		"bilibili_uid": strconv.FormatInt(request.BilibiliUID, 10),
		"requested_at": request.RequestedAt,
		"expires_at":   request.ExpiresAt,
	}
	var account database.BilibiliVerificationAccount
	if database.DB.First(&account).Error == nil {
		data["target_uid"] = strconv.FormatInt(account.BilibiliUID, 10)
	}
	if status == database.VerificationVerified {
		data["bilibili_name"] = request.BilibiliName
		data["bilibili_avatar"] = request.BilibiliAvatar
		data["confirmation_until"] = request.ConfirmationUntil
	}
	Success(c, data)
}

func (ctl *AccountController) Register(c *gin.Context) {
	var body accountRegisterRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	if utf8.RuneCountInString(body.Password) < 8 || len([]byte(body.Password)) > 72 {
		Fail(c, 400, "密码长度需为 8 至 72 个字符")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if utf8.RuneCountInString(body.Username) < 2 || utf8.RuneCountInString(body.Username) > 32 || strings.IndexFunc(body.Username, unicode.IsSpace) >= 0 {
		Fail(c, 400, "登录名需为 2 至 32 个字符且不能包含空格")
		return
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	request, err := loadSessionVerification(c)
	if err != nil {
		Fail(c, 403, "请先完成关注验证")
		return
	}
	now := time.Now().UTC()
	if request.Status != database.VerificationVerified || request.ConfirmationUntil == nil || now.After(*request.ConfirmationUntil) {
		Fail(c, 410, "验证已失效，请重新验证")
		return
	}
	avatarStorage := ctl.AvatarStorage
	if avatarStorage == nil {
		avatarStorage = avatar.NewStore()
	}
	avatarContext, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	storedAvatar, err := avatarStorage.Save(avatarContext, request.BilibiliUID, request.BilibiliAvatar)
	if err != nil {
		log.Warnf("failed to persist Bilibili avatar for uid %d: %v", request.BilibiliUID, err)
		Fail(c, 502, "保存 B 站头像失败，请重试")
		return
	}
	var user database.User
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var locked database.BilibiliVerificationRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, request.ID).Error; err != nil {
			return err
		}
		if locked.Status != database.VerificationVerified || locked.ConfirmationUntil == nil || now.After(*locked.ConfirmationUntil) {
			return errors.New("verification unavailable")
		}
		if locked.VerifiedAt == nil {
			return errors.New("verification timestamp missing")
		}
		user = database.User{
			Username:       body.Username,
			PasswordHash:   string(passwordHash),
			BilibiliUID:    locked.BilibiliUID,
			BilibiliName:   locked.BilibiliName,
			BilibiliAvatar: storedAvatar,
			VerifiedAt:     *locked.VerifiedAt,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		result := tx.Model(&locked).Where("status = ?", database.VerificationVerified).Updates(map[string]any{
			"status":      database.VerificationConsumed,
			"consumed_at": now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("verification already consumed")
		}
		return nil
	})
	if err != nil {
		if deleteErr := avatarStorage.Delete(storedAvatar); deleteErr != nil {
			log.Warnf("failed to clean up unused user avatar: %v", deleteErr)
		}
		Fail(c, 409, "注册失败，该 UID 可能已注册或验证已被使用")
		return
	}
	session := sessions.Default(c)
	session.Set(memberSessionKey, user.BilibiliUID)
	session.Delete(verificationRequestSessionKey)
	session.Delete(verificationTokenSessionKey)
	if err := session.Save(); err != nil {
		Fail(c, 500, "账号已创建，请重新登录")
		return
	}
	Success(c, publicUser(user))
}

func (*AccountController) Login(c *gin.Context) {
	var body accountLoginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	username := strings.TrimSpace(body.Username)
	if username == "" || utf8.RuneCountInString(username) > 32 {
		Fail(c, 401, "用户名或密码错误")
		return
	}
	var user database.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)) != nil {
		Fail(c, 401, "用户名或密码错误")
		return
	}
	if user.IsDisabled {
		Fail(c, 403, "账号已被禁用")
		return
	}
	session := sessions.Default(c)
	session.Set(memberSessionKey, user.BilibiliUID)
	if err := session.Save(); err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, publicUser(user))
}

func (*AccountController) Info(c *gin.Context) {
	user, ok := currentMember(c)
	if !ok {
		Fail(c, 408, "请先登录")
		return
	}
	Success(c, publicUser(user))
}

func (*AccountController) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete(memberSessionKey)
	if err := session.Save(); err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, nil)
}

func currentMember(c *gin.Context) (database.User, bool) {
	uid := sessions.Default(c).Get(memberSessionKey)
	if uid == nil {
		return database.User{}, false
	}
	var user database.User
	if err := database.DB.First(&user, uid).Error; err != nil || user.IsDisabled {
		return database.User{}, false
	}
	return user, true
}
