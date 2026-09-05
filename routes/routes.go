package routes

import (
	"net/http"
	"os"
	"strings"
	"time"

	"finance-app-be/controllers"
	"finance-app-be/middlewares"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedEnvOrigin := os.Getenv("FRONTEND_URL")

		// Daftar domain
		allowedOrigins := map[string]bool{
			"http://localhost:5173":            true,
			"https://lapkeu-msyasy.vercel.app": true,
			"https://lapkeu.zone.id":           true,
			"https://www.lapkeu.zone.id":       true,
		}

		if allowedEnvOrigin != "" {
			allowedOrigins[strings.TrimSuffix(allowedEnvOrigin, "/")] = true
		}

		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if allowedEnvOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedEnvOrigin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Next()
	}
}

// MaxBodySizeMiddleware membatasi ukuran request body (mencegah DoS payload raksasa)
func MaxBodySizeMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.SetTrustedProxies(nil)

	// Middlewares Keamanan Utama
	r.Use(CORSMiddleware())
	r.Use(SecurityHeadersMiddleware())
	r.Use(MaxBodySizeMiddleware(2 << 20))

	authRateLimiter := middlewares.RateLimiter(5, 1*time.Minute)

	api := r.Group("/api")
	{
		// Public Routes
		api.POST("/register", controllers.Register)
		api.POST("/login", authRateLimiter, controllers.Login)
		api.POST("/forgot-password", authRateLimiter, controllers.ForgotPassword)
		api.POST("/reset-password", controllers.ResetPassword)

		// Protected Routes (Wajib Token JWT)
		protected := api.Group("/")
		protected.Use(middlewares.AuthMiddleware())
		{
			// Category Routes
			protected.POST("/categories", controllers.CreateCategory)
			protected.GET("/categories", controllers.GetCategories)
			protected.PUT("/categories/:id/budget", controllers.SetCategoryBudget) 

			// Wallet Routes
			protected.POST("/wallets", controllers.CreateWallet)
			protected.GET("/wallets", controllers.GetWallets)
			protected.POST("/wallets/transfer", controllers.TransferWallet) 
			protected.DELETE("/wallets/:id", controllers.DeleteWallet)

			// Transaction Routes
			protected.POST("/transactions", controllers.CreateTransaction)
			protected.GET("/transactions", controllers.GetTransactions)
			protected.DELETE("/transactions/:id", controllers.DeleteTransaction)
		}
	}

	return r
}