package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	HeaderRequestID = "X-Request-ID"
	RequestIDKey    = "request_id"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderRequestID)
		if reqID == "" {
			reqID = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}
		c.Header(HeaderRequestID, reqID)
		c.Set(RequestIDKey, reqID)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	if val, exists := c.Get(RequestIDKey); exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
