package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				reqID := GetRequestID(c)
				log.Printf("[PANIC RECOVERED] RequestID=%s error=%v", reqID, rec)
				RecordError()
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":     "Internal server error",
					"requestId": reqID,
				})
			}
		}()
		c.Next()
	}
}
