package middleware

import (
	"net/http"
	"time"

	"forlittle/server/internal/config"
	"forlittle/server/internal/models"
	"forlittle/server/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const adminSessionCookieName = "forlittle_admin_session"

func AdminAuth(database *gorm.DB, cfg config.Config) gin.HandlerFunc {
	_ = cfg
	return func(c *gin.Context) {
		token, err := c.Cookie(adminSessionCookieName)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var session models.UserSession
		if err := database.Where("token_hash = ? AND expires_at > ?", services.HashToken(token), time.Now().UTC()).First(&session).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var user models.User
		if err := database.Where("id = ? AND status = ? AND role = ?", session.UserID, "active", "admin").First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		now := time.Now().UTC()
		_ = database.Model(&session).Update("last_seen_at", now).Error
		c.Set("admin_user", user)
		c.Next()
	}
}
