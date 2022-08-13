package controller

import (
	"joiask-backend/internal/database"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
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
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	var user database.Admin
	err := database.DB.Where("username = ?", request.Username).First(&user).Error
	if err != nil {
		Fail(c, 404, "用户不存在")
		return
	}
	if request.Password != user.Password {
		Fail(c, 401, "密码错误")
		return
	}
	session := sessions.Default(c)
	session.Set("user", user.ID)
	session.Set("authed", true)
	err = session.Save()
	if err != nil {
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, nil)
}

func (*UserController) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("user")
	session.Delete("authed")
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
	var user database.Admin
	database.DB.First(&user, c.Param("id"))
	if user.ID == 0 {
		Fail(c, 404, "用户不存在")
		return
	}
	var request UserModifyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求无效")
		return
	}
	if request.Username != "" {
		user.Username = request.Username
	}
	user.Password = request.Password
	if err := database.DB.Save(&user).Error; err != nil {
		log.Errorf("failed to save user: %v", err)
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
	var user database.Admin
	user.Username = request.Username
	user.Password = request.Password
	if err := database.DB.Create(&user).Error; err != nil {
		log.Errorf("failed to create user: %v", err)
		Fail(c, 501, "创建用户失败")
		return
	}
	Success(c, user)
}

func (*UserController) Delete(c *gin.Context) {
	userModel := c.MustGet("user").(database.Admin)
	if userModel.Username != "admin" {
		Fail(c, 403, "没有权限")
		return
	}
	var user database.Admin
	database.DB.First(&user, c.Param("id"))
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
