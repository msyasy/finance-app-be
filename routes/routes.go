package routes

import (
	"finance-app-be/controllers"
	"finance-app-be/middlewares"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware menangani preflight request (OPTIONS) dan memberikan izin CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		// Jika request berupa OPTIONS (Preflight), langsung hentikan dengan status 200 OK
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		c.Next()
	}
}

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Pasang Middleware CORS paling atas
	r.Use(CORSMiddleware())

	api := r.Group("/api")
	{
		// Public Routes
		api.POST("/register", controllers.Register)
		api.POST("/login", controllers.Login)

		// Protected Routes
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