package middlewares

import (
	"net/http"
	"time"

	"finance-app-be/utils"

	"github.com/gin-gonic/gin"
)

// RateLimiter membatasi setiap IP maksimal `maxRequests` request dalam jendela `window`
// (jendela geser, disimpan di memori). Contoh: RateLimiter(5, time.Minute) berarti
// 5 request per menit per IP.
//
// Versi lama memakai tollbooth, yang angka pertamanya berarti request PER DETIK,
// jadi RateLimiter(5, time.Minute) sebenarnya mengizinkan 5 request per detik.
//
// IP diambil dari c.ClientIP(). Agar nilainya IP asli pengunjung (bukan alamat proxy),
// SetupRouter harus mengatur r.TrustedPlatform = gin.PlatformCloudflare.
//
// Setiap pemanggilan membuat penghitung sendiri, jadi limiter "global" dan "strict"
// di routes.go tidak saling mengganggu.
func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	limiter := utils.NewMemoryRateLimiter()

	return func(c *gin.Context) {
		if !limiter.Allow(c.ClientIP(), maxRequests, window) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Terlalu banyak permintaan. Silakan coba beberapa saat lagi.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}