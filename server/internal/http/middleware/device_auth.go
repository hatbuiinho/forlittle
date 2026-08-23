package middleware

import (
	"net/http"
	"strings"
	"time"

	"forlittle/server/internal/models"
	"forlittle/server/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func DeviceAuth(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing device token"})
			return
		}

		var client models.DeviceClient
		if err := database.Where("token_hash = ? AND client_type = ? AND revoked_at IS NULL", services.HashToken(token), "windows_service").First(&client).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid device token"})
			return
		}

		now := time.Now().UTC()
		_ = database.Model(&client).Update("last_seen_at", now).Error
		c.Set("machine_id", client.MachineID)
		c.Next()
	}
}
