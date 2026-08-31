package router

import (
	"forlittle/server/internal/config"
	"forlittle/server/internal/http/handlers"
	"forlittle/server/internal/http/middleware"
	"forlittle/server/internal/timecontrol"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(cfg config.Config, database *gorm.DB) *gin.Engine {
	engine := gin.Default()
	engine.Use(middleware.CORS(cfg))

	engine.GET("/healthz", handlers.Health)

	extensionReleaseHandler := handlers.ExtensionReleaseHandler{ReleasesDir: cfg.ExtensionReleasesDir}
	engine.GET("/extensions/:slug/update.xml", extensionReleaseHandler.ServeUpdateManifest)
	engine.GET("/extensions/:slug/:filename", extensionReleaseHandler.ServeAsset)

	agentHandler := handlers.AgentHandler{DB: database}
	adminHandler := handlers.AdminHandler{DB: database, Cfg: cfg}
	commandBroker := timecontrol.NewCommandBroker()
	timeControlHandler := handlers.TimeControlHandler{DB: database, Cfg: cfg, Broker: commandBroker}

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

		devices := api.Group("/devices")
		{
			devices.POST("/enroll", timeControlHandler.Enroll)

			secured := devices.Group("")
			secured.Use(middleware.DeviceAuth(database))
			secured.GET("/time-policy", timeControlHandler.GetPolicy)
			secured.GET("/commands", timeControlHandler.GetCommands)
			secured.GET("/ws", timeControlHandler.WebSocket)
			secured.POST("/heartbeat", timeControlHandler.Heartbeat)
			secured.POST("/usage", timeControlHandler.Usage)
			secured.POST("/commands/:commandId/ack", timeControlHandler.AckCommand)
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
			secured.GET("/service-machines", adminHandler.ListServiceMachines)
			secured.PATCH("/service-machines/:machineId", adminHandler.UpdateServiceMachine)
			secured.POST("/service-machines/:machineId/deactivate", adminHandler.DeactivateServiceMachine)
			secured.POST("/service-machines/:machineId/reactivate", adminHandler.ReactivateServiceMachine)
			secured.DELETE("/service-machines/:machineId", adminHandler.DeleteServiceMachine)
			secured.GET("/extension-users", adminHandler.ListExtensionUsers)
			secured.PATCH("/extension-users/:machineId", adminHandler.UpdateExtensionUser)
			secured.POST("/extension-users/:machineId/deactivate", adminHandler.DeactivateExtensionUser)
			secured.POST("/extension-users/:machineId/reactivate", adminHandler.ReactivateExtensionUser)
			secured.DELETE("/extension-users/:machineId", adminHandler.DeleteExtensionUser)
			secured.GET("/policy-config", adminHandler.GetPolicyConfig)
			secured.PUT("/policy-config", adminHandler.UpdatePolicyConfig)
			secured.GET("/rules", adminHandler.ListRules)
			secured.POST("/rules", adminHandler.CreateRule)
			secured.PATCH("/rules/:ruleId", adminHandler.UpdateRule)
			secured.DELETE("/rules/:ruleId", adminHandler.DeleteRule)
			secured.GET("/logs", adminHandler.ListLogs)
			secured.GET("/log-groups", adminHandler.ListLogGroups)
			secured.GET("/time-control/little-monks/:littleMonkId/policy", timeControlHandler.GetPolicyAdmin)
			secured.PUT("/time-control/little-monks/:littleMonkId/policy", timeControlHandler.PutPolicyAdmin)
			secured.GET("/time-control/machines", timeControlHandler.ListManagedMachinesAdmin)
			secured.GET("/time-control/shared-policies", timeControlHandler.ListSharedPoliciesAdmin)
			secured.POST("/time-control/shared-policies", timeControlHandler.CreateSharedPolicyAdmin)
			secured.PUT("/time-control/shared-policies/:policyId", timeControlHandler.UpdateSharedPolicyAdmin)
			secured.DELETE("/time-control/shared-policies/:policyId", timeControlHandler.DeleteSharedPolicyAdmin)
			secured.GET("/time-control/machines/:machineId/policy", timeControlHandler.GetMachinePolicyAdmin)
			secured.PUT("/time-control/machines/:machineId/shared-policy", timeControlHandler.PutMachineSharedPolicyAdmin)
			secured.PUT("/time-control/machines/:machineId/override-policy", timeControlHandler.PutMachineOverridePolicyAdmin)
			secured.DELETE("/time-control/machines/:machineId/override-policy", timeControlHandler.DeleteMachineOverridePolicyAdmin)
			secured.POST("/time-control/machines/:machineId/commands", timeControlHandler.CreateCommandAdmin)
			secured.POST("/time-control/machines/:machineId/sync-policy", timeControlHandler.SyncMachinePolicyAdmin)
			secured.GET("/time-control/machines/:machineId/state", timeControlHandler.GetMachineStateAdmin)
			secured.GET("/time-control/machines/:machineId/usage", timeControlHandler.ListUsageAdmin)
		}
	}

	return engine
}
