package router

import (
	"forlittle/server/internal/config"
	"forlittle/server/internal/http/handlers"
	"forlittle/server/internal/http/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(cfg config.Config, database *gorm.DB) *gin.Engine {
	_ = cfg
	engine := gin.Default()
	engine.Use(middleware.CORS())

	engine.GET("/healthz", handlers.Health)

	agentHandler := handlers.AgentHandler{DB: database}
	adminHandler := handlers.AdminHandler{DB: database, Cfg: cfg}

	api := engine.Group("/api/v1")
	{
		agents := api.Group("/agents")
		{
			agents.POST("/register", agentHandler.Register)

			secured := agents.Group("")
			secured.Use(middleware.AgentAuth(database))
			secured.POST("/heartbeat", agentHandler.Heartbeat)
			secured.GET("/policy", agentHandler.Policy)
			secured.POST("/logs/batch", agentHandler.LogsBatch)
		}

		admin := api.Group("/admin")
		{
			admin.POST("/login", adminHandler.Login)

			secured := admin.Group("")
			secured.Use(middleware.AdminAuth(cfg))
			secured.GET("/little-monks", adminHandler.ListLittleMonks)
			secured.POST("/little-monks", adminHandler.CreateLittleMonk)
			secured.GET("/machines", adminHandler.ListMachines)
			secured.POST("/machines/:machineId/assign", adminHandler.AssignMachine)
			secured.GET("/rules", adminHandler.ListRules)
			secured.POST("/rules", adminHandler.CreateRule)
			secured.GET("/logs", adminHandler.ListLogs)
		}
	}

	return engine
}
