package controller

import (
	"joiask-backend/internal/database"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// checkAdminLoginRate must run before looking up the admin or checking passwords,
// so every attempt consumes the same persistent, per-IP budget.
func checkAdminLoginRate(c *gin.Context) bool {
	now := time.Now().UTC()
	allowed := false
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Indexed pruning bounds retained IP digests to the active minute, including
		// when the current request is rejected. No process-local state is needed.
		if err := tx.Where("window_start <= ?", now.Add(-time.Minute)).Delete(&database.AdminLoginRateLimit{}).Error; err != nil {
			return err
		}
		bucket := database.AdminLoginRateLimit{Key: hashVerificationToken(c.ClientIP()), WindowStart: now}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&bucket).Error; err != nil {
			return err
		}
		result := tx.Model(&database.AdminLoginRateLimit{}).
			Where("`key` = ? AND count < ?", bucket.Key, 10).
			UpdateColumn("count", gorm.Expr("count + 1"))
		if result.Error != nil {
			return result.Error
		}
		allowed = result.RowsAffected == 1
		return nil
	})
	if err != nil {
		Fail(c, 500, "内部错误")
		return false
	}
	if !allowed {
		c.Header("Retry-After", "60")
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"code":    429,
			"message": "请求过于频繁，请稍后重试",
			"data":    nil,
		})
		return false
	}
	return true
}
