package middleware

import (
	"net/http"
	"strings"

	"forlittle/server/internal/models"
	"forlittle/server/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AgentAuth(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" || token == auth {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		var client models.ExtensionClient
		if err := database.Where("token_hash = ? AND status IN ?", services.HashToken(token), []string{"active", "pending"}).First(&client).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid device token"})
			return
		}

		c.Set("machine_id", client.MachineID)
		c.Next()
	}
}
