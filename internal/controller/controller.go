package controller

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{
		"code":    200,
		"message": "success",
		"data":    data,
	})
}

func Fail(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(200, gin.H{
		"code":    code,
		"message": message,
		"data":    nil,
	})
}

func getOrderBy(orderBy string) string {
	switch orderBy {
	case "id":
		return "id"
	case "created_at":
		return "created_at"
	case "is_hide":
		return "is_hide"
	case "is_rainbow":
		return "is_rainbow"
	case "is_archive":
		return "is_archive"
	case "is_publish":
		return "is_publish"
	default:
		return "id"
	}
}

func getOrder(order string) string {
	if order == "asc" || order == "desc" {
		return order
	}
	return "desc"
}

func getPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func getPageSize(pageSize int) int {
	if pageSize < 5 {
		return 5
	}
	if pageSize > 100 {
		return 100
	}
	return pageSize
}

func paginate(page int, size int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page == 0 {
			page = 1
		}
		switch {
		case size > 100:
			size = 100
		case size <= 0:
			size = 10
		}

		offset := (page - 1) * size
		return db.Offset(offset).Limit(size)
	}
}
