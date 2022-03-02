package router

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func authMiddleware(c *gin.Context) {
	session := sessions.Default(c)
	authed := session.Get("authed")
	if authed == nil || authed == false {
		c.AbortWithStatusJSON(200, gin.H{
			"code":    2,
			"message": "请先登录",
		})
	}
	c.Next()
}
