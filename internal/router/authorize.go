package router

import (
	"joiask-backend/internal/controller"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func authMiddleware(c *gin.Context) {
	checkAdminSession(c, true)
}

// Public endpoints may use a validated administrator identity, but invalid or
// revoked sessions must have exactly the same permissions as a guest.
func optionalAuthMiddleware(c *gin.Context) {
	checkAdminSession(c, false)
}

func checkAdminSession(c *gin.Context, required bool) {
	admin, authed, err := controller.CurrentAdmin(c)
	if err != nil {
		log.Errorf("failed to validate admin session: %v", err)
		c.AbortWithStatusJSON(200, gin.H{
			"code":    500,
			"message": "内部错误",
		})
		return
	}
	if !authed && required {
		c.AbortWithStatusJSON(200, gin.H{
			"code":    408,
			"message": "请先登录",
		})
		return
	}
	if authed {
		c.Set("user", admin)
		c.Set("authed", true)
	}
	c.Next()
}
