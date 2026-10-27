package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		RecordRequest()

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		reqID := GetRequestID(c)

		errMsg := ""
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}

		if status >= 400 {
			RecordError()
		}

		if errMsg != "" {
			log.Printf("request_id=%s method=%s path=%s status=%d duration=%s error=%s",
				reqID, c.Request.Method, c.Request.URL.Path, status, duration, errMsg)
		} else {
			log.Printf("request_id=%s method=%s path=%s status=%d duration=%s",
				reqID, c.Request.Method, c.Request.URL.Path, status, duration)
		}
	}
}
