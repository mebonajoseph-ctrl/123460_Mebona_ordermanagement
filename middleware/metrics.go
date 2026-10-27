package middleware

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

var (
	requestCountTotal int64
	errorCountTotal   int64
	dbErrorsTotal     int64
)

func RecordRequest() {
	atomic.AddInt64(&requestCountTotal, 1)
}

func RecordError() {
	atomic.AddInt64(&errorCountTotal, 1)
}

func RecordDBError() {
	atomic.AddInt64(&dbErrorsTotal, 1)
}

func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"request_count":   atomic.LoadInt64(&requestCountTotal),
			"error_count":     atomic.LoadInt64(&errorCountTotal),
			"database_errors": atomic.LoadInt64(&dbErrorsTotal),
		})
	}
}
