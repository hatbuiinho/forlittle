package middleware

import (
	"net/http"

	"forlittle/server/internal/config"

	"github.com/gin-gonic/gin"
)

func AdminAuth(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer "+cfg.AdminToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Next()
	}
}
