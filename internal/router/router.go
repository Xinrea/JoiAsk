package router

import (
	"joiask-backend/internal/database"
	"net/http"
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
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5500", "http://192.168.50.198:5500"},
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
	}))
	store := cookie.NewStore([]byte("WhyJoiIsSoCute"))
	store.Options(sessions.Options{
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})
	r.Use(sessions.Sessions("session", store))
	normal := r.Group("/v1")
	{
		normal.POST("/login", login)
		normal.GET("/tag", getTag)
		normal.GET("/questions", func(c *gin.Context) {
			if c.Query("tag_id") == "" {
				getAllQuestions(c)
			} else {
				getAllQuestionsByTag(c)
			}
		})
		normal.POST("/questions", addQuestion)
		normal.GET("/rainbows", getAllRainbowQuestions)
		normal.GET("/like", likes)
		normal.GET("/tags", getAllTags)
		normal.GET("/config", func(c *gin.Context) {
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
	}
	authed := r.Group("/v1")
	authed.Use(authMiddleware)
	{
		authed.GET("/logout", logout)
		authed.GET("/auth", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"code":    0,
				"message": "已登录",
			})
		})
		authed.GET("/auth/tags", authGetAllTags)
		authed.POST("/auth/questions", authGetAllQuestions)
		authed.PUT("/auth/questions", updateQuestion)
		authed.DELETE("/auth/questions", deleteQuesion)
		authed.POST("/config", func(c *gin.Context) {
			a := c.PostForm("announcement")
			database.SetConfig(a)
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
