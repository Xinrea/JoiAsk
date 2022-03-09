package router

import (
	"joiask-backend/internal/database"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func Run() {
	r := gin.Default()
	r.Use(gin.Recovery())
	if os.Getenv("GIN_MODE") == "release" {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"https://ask.vjoi.cn"},
			AllowCredentials: true,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		}))
	} else {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"https://ask.vjoi.cn", "http://localhost:5500"},
			AllowCredentials: true,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		}))
	}

	store := cookie.NewStore([]byte("WhyJoiIsSoCute"))
	store.Options(sessions.Options{
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})
	r.Use(sessions.Sessions("session", store))
	r.POST("/login", login)
	r.GET("/tag", getTag)
	r.GET("/questions", func(c *gin.Context) {
		if c.Query("tag_id") == "" {
			getAllQuestions(c)
		} else {
			getAllQuestionsByTag(c)
		}
	})
	r.POST("/questions", addQuestion)
	r.GET("/rainbows", getAllRainbowQuestions)
	r.GET("/like", likes)
	r.GET("/tags", getAllTags)
	r.GET("/config", func(c *gin.Context) {
		config, err := database.GetConfig()
		if err != nil {
			c.JSON(200, gin.H{
				"code":    3,
				"message": "获取配置失败",
			})
			return
		}
		c.JSON(200, gin.H{
			"code":    0,
			"message": "获取成功",
			"data":    config,
		})
	})
	authed := r.Group("/auth")
	authed.Use(authMiddleware)
	{
		authed.GET("/logout", logout)
		authed.GET("/login", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"code":    0,
				"message": "已登录",
			})
		})
		authed.GET("/tags", authGetAllTags)
		authed.POST("/questions", authGetAllQuestions)
		authed.PUT("/questions", updateQuestion)
		authed.DELETE("/questions", deleteQuesion)
		authed.POST("/config", func(c *gin.Context) {
			announcement := c.PostForm("announcement")
			database.SetConfig(announcement)
			c.JSON(200, gin.H{
				"code":    0,
				"message": "设置成功",
			})
		})
	}
	address := viper.GetString("server.host") + ":" + strconv.Itoa(viper.GetInt("server.port"))
	logrus.Info(address)
	r.Run(address)
}
