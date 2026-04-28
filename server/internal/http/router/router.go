package router

import (
	"forlittle/server/internal/config"
	"forlittle/server/internal/http/handlers"
	"forlittle/server/internal/http/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(cfg config.Config, database *gorm.DB) *gin.Engine {
	engine := gin.Default()
	engine.Use(middleware.CORS(cfg))

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
			admin.POST("/auth/login", adminHandler.Login)

			secured := admin.Group("")
			secured.Use(middleware.AdminAuth(database, cfg))
			secured.GET("/auth/me", adminHandler.Me)
			secured.POST("/auth/logout", adminHandler.Logout)
			secured.GET("/little-monks", adminHandler.ListLittleMonks)
			secured.POST("/little-monks", adminHandler.CreateLittleMonk)
			secured.GET("/machines", adminHandler.ListMachines)
			secured.POST("/machines/:machineId/assign", adminHandler.AssignMachine)
			secured.GET("/policy-config", adminHandler.GetPolicyConfig)
			secured.PUT("/policy-config", adminHandler.UpdatePolicyConfig)
			secured.GET("/rules", adminHandler.ListRules)
			secured.POST("/rules", adminHandler.CreateRule)
			secured.PATCH("/rules/:ruleId", adminHandler.UpdateRule)
			secured.DELETE("/rules/:ruleId", adminHandler.DeleteRule)
			secured.GET("/logs", adminHandler.ListLogs)
			secured.GET("/log-groups", adminHandler.ListLogGroups)
		}
	}

	return engine
}
