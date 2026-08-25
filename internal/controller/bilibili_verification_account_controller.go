package controller

import (
	"context"
	"joiask-backend/internal/bilibili"
	"joiask-backend/internal/database"
	"joiask-backend/internal/secret"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BilibiliVerificationAccountController struct {
	Client *bilibili.Client
}

var verificationAccountMutex sync.Mutex

type verificationAccountRequest struct {
	BilibiliUID string `json:"bilibili_uid"`
	Cookie      string `json:"cookie"`
}

func verificationAccountResponse(account database.BilibiliVerificationAccount) gin.H {
	return gin.H{
		"id":                 account.ID,
		"bilibili_uid":       strconv.FormatInt(account.BilibiliUID, 10),
		"cookie_configured":  account.EncryptedCookie != "",
		"last_checked_at":    account.LastCheckedAt,
		"last_successful_at": account.LastSuccessfulAt,
		"last_error":         account.LastError,
		"created_at":         account.CreatedAt,
		"updated_at":         account.UpdatedAt,
	}
}

func (ctl *BilibiliVerificationAccountController) Get(c *gin.Context) {
	var account database.BilibiliVerificationAccount
	if err := database.DB.First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			Success(c, nil)
			return
		}
		Fail(c, 500, "读取验证账号失败")
		return
	}
	Success(c, verificationAccountResponse(account))
}

func (ctl *BilibiliVerificationAccountController) Put(c *gin.Context) {
	var body verificationAccountRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	uid, _, err := normalizeBilibiliUID(body.BilibiliUID)
	if err != nil {
		Fail(c, 400, err.Error())
		return
	}
	verificationAccountMutex.Lock()
	defer verificationAccountMutex.Unlock()
	var account database.BilibiliVerificationAccount
	findErr := database.DB.First(&account).Error
	cookieValue := strings.TrimSpace(body.Cookie)
	if cookieValue == "" && findErr == nil {
		cookieValue, err = secret.DecryptCookie(account.EncryptedCookie)
	}
	if cookieValue == "" || err != nil {
		Fail(c, 400, "请输入有效的 B 站 Cookie")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	loggedInUID, err := ctl.Client.ValidateAccount(ctx, cookieValue)
	if err != nil {
		Fail(c, 400, err.Error())
		return
	}
	if loggedInUID != uid {
		Fail(c, 400, "Cookie 登录账号与填写的 UID 不一致")
		return
	}
	encryptedCookie, err := secret.EncryptCookie(cookieValue)
	if err != nil {
		Fail(c, 500, err.Error())
		return
	}
	now := time.Now().UTC()
	if findErr == gorm.ErrRecordNotFound {
		account = database.BilibiliVerificationAccount{
			BilibiliUID:      uid,
			EncryptedCookie:  encryptedCookie,
			LastCheckedAt:    &now,
			LastSuccessfulAt: &now,
		}
		if err := database.DB.Create(&account).Error; err != nil {
			Fail(c, 500, "保存验证账号失败")
			return
		}
	} else if findErr != nil {
		Fail(c, 500, "读取验证账号失败")
		return
	} else {
		account.BilibiliUID = uid
		account.EncryptedCookie = encryptedCookie
		account.LastCheckedAt = &now
		account.LastSuccessfulAt = &now
		account.LastError = ""
		if err := database.DB.Save(&account).Error; err != nil {
			Fail(c, 500, "保存验证账号失败")
			return
		}
	}
	Success(c, verificationAccountResponse(account))
}

func (ctl *BilibiliVerificationAccountController) Test(c *gin.Context) {
	var account database.BilibiliVerificationAccount
	if err := database.DB.First(&account).Error; err != nil {
		Fail(c, 404, "尚未配置验证账号")
		return
	}
	cookieValue, err := secret.DecryptCookie(account.EncryptedCookie)
	if err != nil {
		Fail(c, 500, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	uid, err := ctl.Client.ValidateAccount(ctx, cookieValue)
	now := time.Now().UTC()
	updates := map[string]any{"last_checked_at": now}
	if err != nil || uid != account.BilibiliUID {
		message := "Cookie 登录账号与配置 UID 不一致"
		if err != nil {
			message = err.Error()
		}
		updates["last_error"] = message
		database.DB.Model(&account).Updates(updates)
		Fail(c, 400, message)
		return
	}
	updates["last_successful_at"] = now
	updates["last_error"] = ""
	database.DB.Model(&account).Updates(updates)
	Success(c, gin.H{"valid": true, "checked_at": now})
}

func (*BilibiliVerificationAccountController) Delete(c *gin.Context) {
	verificationAccountMutex.Lock()
	defer verificationAccountMutex.Unlock()
	if err := database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&database.BilibiliVerificationAccount{}).Error; err != nil {
		Fail(c, 500, "删除验证账号失败")
		return
	}
	Success(c, nil)
}
