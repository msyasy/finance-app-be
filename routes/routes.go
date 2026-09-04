package routes

import (
	"os"
	"time"

	"finance-app-be/controllers"
	"finance-app-be/middlewares"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowedOrigin := os.Getenv("FRONTEND_URL")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:5173" 
		}

		origin := c.Request.Header.Get("Origin")
		if origin == allowedOrigin || origin == "http://localhost:5173" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
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

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.SetTrustedProxies(nil)

	r.Use(CORSMiddleware())
	r.Use(SecurityHeadersMiddleware())

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

			// Wallet Routes
			protected.POST("/wallets", controllers.CreateWallet)
			protected.GET("/wallets", controllers.GetWallets)
			protected.DELETE("/wallets/:id", controllers.DeleteWallet)

			// Transaction Routes
			protected.POST("/transactions", controllers.CreateTransaction)
			protected.GET("/transactions", controllers.GetTransactions)
			protected.DELETE("/transactions/:id", controllers.DeleteTransaction)
		}
	}

	return r
}