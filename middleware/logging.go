package middleware

import (
	"github.com/gin-gonic/gin"
)

// Logging returns a logging middleware for HTTP requests
func Logging() gin.HandlerFunc {
	return gin.LoggerWithFormatter(gin.LogFormatter(func(params gin.LogFormatterParams) string {
		return ""
	}))
}