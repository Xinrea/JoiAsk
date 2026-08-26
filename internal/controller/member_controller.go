package controller

import (
	"context"
	"joiask-backend/internal/avatar"
	"joiask-backend/internal/bilibili"
	"joiask-backend/internal/database"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type BilibiliProfileClient interface {
	Profile(context.Context, int64) (bilibili.Profile, error)
}

type MemberController struct {
	Client        BilibiliProfileClient
	AvatarStorage AvatarStorage
}

type memberListRequest struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type memberStatusRequest struct {
	IsDisabled *bool `json:"is_disabled"`
}

type memberCreateRequest struct {
	BilibiliUID string `json:"bilibili_uid"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

func (*MemberController) Get(c *gin.Context) {
	var query memberListRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	page, pageSize := getPage(query.Page), getPageSize(query.PageSize)
	var users []database.User
	var total int64
	if err := database.DB.Model(&database.User{}).Count(&total).Scopes(paginate(page, pageSize)).Order("created_at desc").Find(&users).Error; err != nil {
		Fail(c, 500, "读取注册用户失败")
		return
	}
	items := make([]gin.H, 0, len(users))
	for _, user := range users {
		items = append(items, publicUser(user))
	}
	Success(c, gin.H{"users": items, "total": total, "page": page, "page_size": pageSize})
}

func (ctl *MemberController) Post(c *gin.Context) {
	var body memberCreateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	uid, _, err := normalizeBilibiliUID(body.BilibiliUID)
	if err != nil {
		Fail(c, 400, err.Error())
		return
	}
	username := strings.TrimSpace(body.Username)
	if utf8.RuneCountInString(username) < 2 || utf8.RuneCountInString(username) > 32 || strings.IndexFunc(username, unicode.IsSpace) >= 0 {
		Fail(c, 400, "登录名需为 2 至 32 个字符且不能包含空格")
		return
	}
	if utf8.RuneCountInString(body.Password) < 8 || len([]byte(body.Password)) > 72 {
		Fail(c, 400, "密码长度需为 8 至 72 个字符")
		return
	}
	if ctl.Client == nil {
		Fail(c, 503, "B 站用户信息服务暂不可用")
		return
	}
	profileContext, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	profile, err := ctl.Client.Profile(profileContext, uid)
	if err != nil {
		log.Warnf("failed to fetch Bilibili profile for manual member uid %d: %v", uid, err)
		Fail(c, 400, "获取 B 站账号信息失败，请确认 UID 有效")
		return
	}
	var count int64
	if err := database.DB.Model(&database.User{}).Where("bilibili_uid = ?", uid).Count(&count).Error; err != nil {
		Fail(c, 500, "检查用户失败")
		return
	}
	if count > 0 {
		Fail(c, 409, "该 B 站 UID 已注册")
		return
	}
	if err := database.DB.Model(&database.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		Fail(c, 500, "检查用户失败")
		return
	}
	if count > 0 {
		Fail(c, 409, "该登录名已被使用")
		return
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		Fail(c, 500, "创建用户失败")
		return
	}
	now := time.Now().UTC()
	avatarStorage := ctl.AvatarStorage
	if avatarStorage == nil {
		avatarStorage = avatar.NewStore()
	}
	storedAvatar := ""
	if profile.Face != "" {
		storedAvatar, err = avatarStorage.Save(profileContext, uid, profile.Face)
		if err != nil {
			Fail(c, 502, "保存 B 站头像失败，请重试")
			return
		}
	}
	user := database.User{
		BilibiliUID:    uid,
		Username:       username,
		PasswordHash:   string(passwordHash),
		BilibiliName:   profile.Name,
		BilibiliAvatar: storedAvatar,
		VerifiedAt:     now,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		Fail(c, 409, "创建失败，该 UID 或登录名可能已被使用")
		return
	}
	Success(c, publicUser(user))
}

func (*MemberController) Put(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil || uid <= 0 {
		Fail(c, 400, "B 站 UID 无效")
		return
	}
	var body memberStatusRequest
	if err := c.ShouldBindJSON(&body); err != nil || body.IsDisabled == nil {
		Fail(c, 400, "请求无效")
		return
	}
	var user database.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		Fail(c, 404, "用户不存在")
		return
	}
	if err := database.DB.Model(&user).Update("is_disabled", *body.IsDisabled).Error; err != nil {
		Fail(c, 500, "更新用户状态失败")
		return
	}
	user.IsDisabled = *body.IsDisabled
	Success(c, publicUser(user))
}

func (*MemberController) Delete(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil || uid <= 0 {
		Fail(c, 400, "B 站 UID 无效")
		return
	}
	var user database.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		Fail(c, 404, "用户不存在")
		return
	}
	if err := database.DB.Delete(&user).Error; err != nil {
		Fail(c, 500, "删除用户失败")
		return
	}
	Success(c, nil)
}
