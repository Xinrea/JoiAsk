package router

import (
	"joiask-backend/internal/database"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func authMiddleware(c *gin.Context) {
	session := sessions.Default(c)
	authed := session.Get("authed")
	userID := session.Get("user")
	if authed == nil || authed == false || userID == nil {
		c.AbortWithStatusJSON(200, gin.H{
			"code":    408,
			"message": "请先登录",
		})
		return
	}
	var userModel database.Admin
	if database.DB.Where("id = ?", userID).First(&userModel).RowsAffected == 0 {
		session.Delete("user")
		session.Delete("authed")
		c.AbortWithStatusJSON(200, gin.H{
			"code":    408,
			"message": "请先登录",
		})
		return
	}
	c.Set("user", userModel)
	c.Set("authed", true)
	c.Next()
}
