package controller

import (
	"errors"
	"joiask-backend/internal/adminpassword"
	"joiask-backend/internal/database"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserController struct{}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserModifyRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserAddRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (*UserController) Info(c *gin.Context) {
	Success(c, c.MustGet("user"))
}

func (*UserController) Login(c *gin.Context) {
	if !checkAdminLoginRate(c) {
		return
	}
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	if request.Username == "" || request.Password == "" {
		Fail(c, 400, "用户名和密码不能为空")
		return
	}
	var user database.Admin
	err := database.DB.Where("username = ?", request.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Fail(c, 401, "用户名或密码错误")
		} else {
			Fail(c, 500, "内部错误")
		}
		return
	}
	matches, err := verifyAndUpgradeAdminPassword(&user, request.Password)
	if err != nil {
		log.Errorf("failed to migrate admin password: %v", err)
		Fail(c, 500, "内部错误")
		return
	}
	if !matches {
		Fail(c, 401, "用户名或密码错误")
		return
	}
	session := sessions.Default(c)
	session.Set("user", user.ID)
	session.Set("authed", true)
	// Bind to the version read with the verified password, not a fresh version
	// that could belong to a concurrent password reset.
	session.Set(adminSessionVersionKey, user.SessionVersion)
	err = session.Save()
	if err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, nil)
}

func verifyAndUpgradeAdminPassword(user *database.Admin, password string) (bool, error) {
	matches, needsMigration := adminpassword.Verify(user.Password, password)
	if !matches || !needsMigration {
		return matches, nil
	}
	hash, err := adminpassword.Hash(password)
	if err != nil {
		return false, err
	}
	// Do not overwrite a concurrent password reset, deletion, or migration.
	result := database.DB.Model(&database.Admin{}).
		Where("id = ? AND password = ?", user.ID, user.Password).
		Update("password", hash)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		if err := database.DB.First(user, user.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		matches, _ := adminpassword.Verify(user.Password, password)
		return matches, nil
	}
	user.Password = hash
	return true, nil
}

func (*UserController) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("user")
	session.Delete("authed")
	session.Delete(adminSessionVersionKey)
	err := session.Save()
	if err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, nil)
}

func (*UserController) Get(c *gin.Context) {
	var users []database.Admin
	database.DB.Find(&users)
	Success(c, users)
}

func (*UserController) Put(c *gin.Context) {
	userModel := c.MustGet("user").(database.Admin)
	if userModel.Username != "admin" {
		Fail(c, 403, "没有权限")
		return
	}
	id, ok := parseRecordID(c, c.Param("id"))
	if !ok {
		return
	}
	var user database.Admin
	database.DB.First(&user, id)
	if user.ID == 0 {
		Fail(c, 404, "用户不存在")
		return
	}
	var request UserModifyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	updates := map[string]interface{}{}
	if request.Username != "" {
		user.Username = request.Username
		updates["username"] = request.Username
	}
	// An omitted or empty password leaves the existing credential unchanged.
	if request.Password != "" {
		hash, err := adminpassword.Hash(request.Password)
		if err != nil {
			adminPasswordError(c, err)
			return
		}
		user.Password = hash
		updates["password"] = hash
		// Revoke every previously issued session in the same write as the
		// password change. A database increment also preserves concurrent resets.
		updates["session_version"] = gorm.Expr("session_version + 1")
	}
	// Only update requested fields: saving a stale full record could revert a
	// concurrent password migration/reset or recreate a deleted administrator.
	result := database.DB.Model(&database.Admin{}).Where("id = ?", user.ID).Updates(updates)
	if result.Error != nil {
		log.Errorf("failed to save user: %v", result.Error)
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, user)
}

func (*UserController) Post(c *gin.Context) {
	userModel := c.MustGet("user").(database.Admin)
	if userModel.Username != "admin" {
		Fail(c, 403, "权限不足")
		return
	}
	var request UserAddRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	if request.Username == "" {
		Fail(c, 400, "用户名不能为空")
		return
	}
	hash, err := adminpassword.Hash(request.Password)
	if err != nil {
		adminPasswordError(c, err)
		return
	}
	var user database.Admin
	user.Username = request.Username
	user.Password = hash
	if err := database.DB.Create(&user).Error; err != nil {
		log.Errorf("failed to create user: %v", err)
		Fail(c, 501, "创建用户失败")
		return
	}
	Success(c, user)
}

func adminPasswordError(c *gin.Context, err error) {
	if errors.Is(err, adminpassword.ErrEmptyPassword) {
		Fail(c, 400, err.Error())
		return
	}
	log.Errorf("failed to hash admin password: %v", err)
	Fail(c, 500, "内部错误")
}

func (*UserController) Delete(c *gin.Context) {
	userModel := c.MustGet("user").(database.Admin)
	if userModel.Username != "admin" {
		Fail(c, 403, "没有权限")
		return
	}
	id, ok := parseRecordID(c, c.Param("id"))
	if !ok {
		return
	}
	var user database.Admin
	database.DB.First(&user, id)
	if user.ID == 0 {
		Fail(c, 404, "用户不存在")
		return
	}
	if user.Username == "admin" {
		Fail(c, 403, "不能删除默认管理员")
		return
	}
	if err := database.DB.Delete(&user).Error; err != nil {
		log.Errorf("failed to delete user: %v", err)
		Fail(c, 502, "删除用户失败")
		return
	}
	Success(c, nil)
}
