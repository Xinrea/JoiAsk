package router

import (
	"joiask-backend/internal/database"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func authGetAllTags(c *gin.Context) {
	tags, err := database.GetAllTags(false)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    3,
			"message": "获取标签失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    tags,
	})
}

func authGetAllQuestions(c *gin.Context) {
	var config database.TableConfig
	c.ShouldBindJSON(&config)
	qs, total, err := database.AuthAllQuestions(config)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    3,
			"message": "获取问题失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "获取成功",
		"data": gin.H{
			"questions": qs,
			"total":     total,
		},
	})
}

func updateQuestion(c *gin.Context) {
	var params map[string]interface{}
	c.ShouldBindJSON(&params)
	logrus.Info(params)
	id, ok := params["id"].(float64)
	if !ok {
		c.JSON(200, gin.H{
			"code":    9,
			"message": "ID错误",
		})
		return
	}
	row, ok := params["key"].(string)
	if !ok {
		c.JSON(200, gin.H{
			"code":    9,
			"message": "Key错误",
		})
		return
	}
	if params["value"] == nil {
		c.JSON(200, gin.H{
			"code":    9,
			"message": "Value错误",
		})
		return
	}
	err := database.UpdateQuestion(uint(id), row, params["value"])
	if err != nil {
		c.JSON(200, gin.H{
			"code":    3,
			"message": "更新失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

func deleteQuesion(c *gin.Context) {
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		c.JSON(200, gin.H{
			"code":    9,
			"message": "ID错误",
		})
		return
	}
	err = database.DeleteQuestion(uint(id))
	if err != nil {
		c.JSON(200, gin.H{
			"code":    3,
			"message": "删除失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}
