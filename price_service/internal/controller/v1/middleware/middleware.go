package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func Log() gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Debug("request", slog.String("method", c.Request.Method), slog.String("uri", c.Request.URL.Path))
		c.Next()
	}
}
