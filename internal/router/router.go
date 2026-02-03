package router

import (
	"joiask-backend/internal/controller"
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
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "*"},
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))
	store := cookie.NewStore([]byte("WhyJoiIsSoCute"))
	store.Options(sessions.Options{
		Path:     "/",
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("session", store))
	api := r.Group("/api")
	tagController := new(controller.TagController)
	userController := new(controller.UserController)
	questionController := controller.NewQuestionController()
	configController := new(controller.ConfigController)
	statisticsController := new(controller.StatisticsController)
	{
		// User
		{
			api.POST("/login", userController.Login)
			api.GET("/info", authMiddleware, userController.Info)
			api.GET("/logout", authMiddleware, userController.Logout)

			api.GET("/user", authMiddleware, userController.Get)
			api.POST("/user", authMiddleware, userController.Post)
			api.PUT("/user/:id", authMiddleware, userController.Put)
			api.DELETE("/user/:id", authMiddleware, userController.Delete)
		}
		// Tag
		{
			api.GET("/tag", tagController.Get)
			api.PUT("/tag/:id", authMiddleware, tagController.Put)
			api.DELETE("/tag/:id", authMiddleware, tagController.Delete)
			api.POST("/tag", authMiddleware, tagController.Post)
		}
		// Question
		{
			api.GET("/question", questionController.Get)
			api.POST("/question", questionController.Post)
			api.PUT("/question/:id", authMiddleware, questionController.Put)
			api.POST("/question/:id/emoji", questionController.Emoji)
			api.GET("/sse", questionController.SSE)
			api.GET("/ws", questionController.WebSocket)
			api.DELETE("/question/:id", authMiddleware, questionController.Delete)
		}
		// Config
		{
			api.GET("/config", configController.Get)
			api.PUT("/config", authMiddleware, configController.Put)
		}
		// Statistics
		{
			api.GET("/statistics", authMiddleware, statisticsController.Get)
		}
	}
	address := viper.GetString("server.host") + ":" + strconv.Itoa(viper.GetInt("server.port"))
	logrus.Info(address)
	logrus.Error(r.Run(address))
}
