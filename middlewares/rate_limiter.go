package middlewares

import (
	"time"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	"github.com/gin-gonic/gin"
)

func RateLimiter(maxRequests float64, ttl time.Duration) gin.HandlerFunc {
	lmt := tollbooth.NewLimiter(maxRequests, &limiter.ExpirableOptions{DefaultExpirationTTL: ttl})
	lmt.SetMessage(`{"error": "Terlalu banyak permintaan. Silakan coba beberapa saat lagi."}`)
	lmt.SetMessageContentType("application/json")

	return func(c *gin.Context) {
		httpError := tollbooth.LimitByRequest(lmt, c.Writer, c.Request)
		if httpError != nil {
			c.JSON(httpError.StatusCode, gin.H{"error": "Terlalu banyak permintaan. Silakan coba beberapa saat lagi."})
			c.Abort()
			return
		}
		c.Next()
	}
}