package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"order-management/config"
	"order-management/database"
	"order-management/internal/gateway"
	orderPkg "order-management/internal/order"
	productPkg "order-management/internal/product"
	"order-management/middleware"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

func main() {
	cfg := config.LoadConfig()

	// 1. Initialize Database with GORM & connection pooling
	var orderRepo orderPkg.Repository
	var productRepo productPkg.Repository

	gormDB, err := database.NewGormDB(cfg)
	if err != nil {
		log.Printf("[WARN] PostgreSQL/GORM connection failed: %v. Running in-memory mode for offline use.", err)
		orderRepo = orderPkg.NewMemoryRepository()
		productRepo = productPkg.NewMemoryRepository()
	} else {
		orderRepo = orderPkg.NewGormRepository(gormDB)
		productRepo = productPkg.NewGormRepository(gormDB)
	}

	// 2. Initialize Product Microservice with Resilient Client (Circuit Breaker & Backoff)
	localProductSvc := productPkg.NewService(productRepo)
	resilientProductSvc := productPkg.NewResilientProductClient(cfg.ProductServiceURL, localProductSvc)

	// 3. Initialize Order Microservice
	orderSvc := orderPkg.NewService(orderRepo, resilientProductSvc)

	// 4. Initialize Gateway Controller
	orderCtrl := gateway.NewOrderController(orderSvc)

	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global Middlewares
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.LoggingMiddleware())
	r.Use(func(c *gin.Context) {
		// Secure HTTP Headers & HTTPS enforcement
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	})

	// Redirect root to interactive GraphQL UI
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/graphql")
	})

	// Observability Endpoints
	r.GET("/health/live", orderCtrl.HealthLive)
	r.GET("/health/ready", orderCtrl.HealthReady)
	r.GET("/metrics", middleware.MetricsHandler())

	// Public Authentication Endpoint
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":     "Invalid input: valid email and password are required",
				"requestId": middleware.GetRequestID(c),
			})
			return
		}

		role := "customer"
		if req.Role == "admin" || req.Email == "admin@company.com" {
			role = "admin"
		}

		token, err := middleware.GenerateToken(101, req.Email, role, cfg.JWTSecret, 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":     "Failed to generate authentication token",
				"requestId": middleware.GetRequestID(c),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":      token,
			"token_type": "Bearer",
			"expires_in": 86400,
			"role":       role,
		})
	})

	// Protected REST APIs for Orders
	apiV1 := r.Group("/api/v1")
	apiV1.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		apiV1.POST("/orders", orderCtrl.CreateOrder)
		apiV1.GET("/orders/:id", orderCtrl.GetOrder)
		apiV1.GET("/orders", orderCtrl.ListOrders)
		apiV1.PUT("/orders/:id/status", middleware.RequireRole("admin"), orderCtrl.UpdateOrderStatus)
	}

	// Mount GraphQL Gateway (Orders & Products)
	r.Any("/graphql", gateway.NewGraphQLHandler(orderSvc, resilientProductSvc))

	serverAddr := fmt.Sprintf(":%s", cfg.AppPort)
	graphqlURL := fmt.Sprintf("http://localhost:%s/graphql", cfg.AppPort)

	log.Println("==================================================================")
	log.Printf("[INFO] Gateway and Microservices running on port %s", cfg.AppPort)
	log.Printf("[INFO] Interactive GraphQL Web UI: %s", graphqlURL)
	log.Printf("[INFO] REST Endpoints available at /api/v1/orders")
	log.Println("==================================================================")

	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}
