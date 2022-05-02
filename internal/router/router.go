package router

import (
	"joiask-backend/internal/database"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/contrib/static"
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
		Secure:   false,
		SameSite: http.SameSiteDefaultMode,
	})
	r.Use(sessions.Sessions("session", store))
	r.Use(static.Serve("/", static.LocalFile("./frontend/public/", true)))
	r.Use(static.Serve("/admin", static.LocalFile("./frontend/public/", true)))
	r.Use(static.Serve("/tags", static.LocalFile("./frontend/public/", true)))
	r.Use(static.Serve("/rainbow", static.LocalFile("./frontend/public/", true)))
	api := r.Group("/api")
	{
		api.POST("/login", login)
		api.GET("/tag", getTag)
		api.GET("/questions", func(c *gin.Context) {
			if c.Query("tag_id") == "" {
				getAllQuestions(c)
			} else {
				getAllQuestionsByTag(c)
			}
		})
		api.POST("/questions", addQuestion)
		api.GET("/rainbows", getAllRainbowQuestions)
		api.GET("/like", likes)
		api.GET("/tags", getAllTags)
		api.GET("/config", func(c *gin.Context) {
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
		authed := api.Group("/auth")
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
	}
	address := viper.GetString("server.host") + ":" + strconv.Itoa(viper.GetInt("server.port"))
	logrus.Info(address)
	r.Run(address)
}
