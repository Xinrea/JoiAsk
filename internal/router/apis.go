package router

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"joiask-backend/internal/database"
	"path"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func login(c *gin.Context) {
	user := c.PostForm("username")
	password := c.PostForm("password")
	passHash, err := database.GetUserPasswordMD5(user)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    1,
			"message": "用户或密码错误",
		})
		return
	}
	if passHash != md5v(password) {
		c.JSON(200, gin.H{
			"code":    1,
			"message": "用户或密码错误",
		})
		logrus.Info(md5v(password))
		return
	}
	session := sessions.Default(c)
	session.Set("authed", true)
	session.Save()
	c.JSON(200, gin.H{
		"code":    0,
		"message": "登录成功",
	})
}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Set("authed", false)
	session.Save()
	c.JSON(200, gin.H{
		"code":    0,
		"message": "注销成功",
	})
}

func getTag(c *gin.Context) {
	tagid, err := strconv.Atoi(c.Query("tag_id"))
	if err != nil {
		tagid = 1
	}
	tag, err := database.GetTag(uint(tagid))
	if err != nil {
		c.JSON(200, gin.H{
			"code":    3,
			"message": "获取话题失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    tag,
	})
}

func getAllTags(c *gin.Context) {
	tags, err := database.GetAllTags(true)
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

func getAllQuestions(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	qs, err := database.GetAllQuestions(true, page, size)
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
		"data":    qs,
	})
}

func getAllQuestionsByTag(c *gin.Context) {
	tagID, _ := strconv.Atoi(c.Query("tag_id"))
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	qs, err := database.GetAllQuestionsByTag(true, tagID, page, size)
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
		"data":    qs,
	})
}

func getAllRainbowQuestions(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	qs, err := database.GetAllRainbowQuestions(true, page, size)
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
		"data":    qs,
	})
}

func likes(c *gin.Context) {
	if c.Query("id") == "" {
		c.JSON(200, gin.H{
			"code":    9,
			"message": "参数错误",
		})
		return
	}
	id, _ := strconv.Atoi(c.Query("id"))
	err := database.AddLikeRecord(c.ClientIP(), uint(id))
	if err != nil {
		c.JSON(200, gin.H{
			"code":    4,
			"message": "点赞失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "点赞成功",
	})
}

func addQuestion(c *gin.Context) {
	tagName := c.PostForm("tag")
	if tagName == "" {
		tagName = "轴问箱"
	}
	tagID, err := database.AddNewTag(tagName)
	if err != nil {
		logrus.Error(err)
		c.JSON(200, gin.H{
			"code":    3,
			"message": "话题创建失败",
		})
		return
	}
	var q database.Question
	q.TagID = int(tagID)
	q.Content = strings.Trim(c.PostForm("content"), " \r\n\t")
	q.IsHide = c.PostForm("hide") == "true"
	q.IsRainbow = c.PostForm("rainbow") == "true"
	mp, _ := c.MultipartForm()
	for _, v := range mp.File["files[]"] {
		f, err := v.Open()
		if err != nil {
			logrus.Error(err)
			continue
		}
		fileContent, err := ioutil.ReadAll(f)
		if err != nil {
			logrus.Error(err)
			continue
		}
		newFileName := md5v(string(fileContent)) + path.Ext(v.Filename)
		url, err := storageBackend.Upload(newFileName, bytes.NewReader(fileContent))
		if err != nil {
			logrus.Error(err)
			c.JSON(200, gin.H{
				"code":    3,
				"message": "提交问题失败",
			})
			return
		}
		q.ImagesNum++
		if q.Images == "" {
			q.Images = url
		} else {
			q.Images += ";" + url
		}
	}
	err = database.AddNewQuestion(q)
	if err != nil {
		logrus.Error(err)
		c.JSON(200, gin.H{
			"code":    3,
			"message": "提交问题失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code":    0,
		"message": "问题提交成功",
	})
}

func md5v(v string) string {
	d := []byte(v)
	m := md5.New()
	m.Write(d)
	return hex.EncodeToString(m.Sum(nil))
}
