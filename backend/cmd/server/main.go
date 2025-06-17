package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"utunnel-pro/internal/api/handlers"
	"utunnel-pro/internal/config"
	"utunnel-pro/internal/middleware"
	"utunnel-pro/internal/models"
	"utunnel-pro/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize Redis
	redisClient := initRedis(cfg)

	// Initialize services
	authService := services.NewAuthService(db, redisClient, cfg)
	tunnelService := services.NewTunnelService(db, redisClient, cfg)
	monitoringService := services.NewMonitoringService(db, redisClient, cfg)

	// Start monitoring service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := monitoringService.Start(ctx); err != nil {
		log.Fatalf("Failed to start monitoring service: %v", err)
	}

	// Initialize handlers
	tunnelHandler := handlers.NewTunnelHandler(tunnelService, nil)
	authHandler := handlers.NewAuthHandler(authService)

	// Setup Gin router
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(middleware.CORSMiddleware(cfg))

	// Add rate limiting middleware
	if cfg.Security.RateLimitEnabled {
		router.Use(middleware.RateLimitMiddleware(cfg))
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   cfg.App.Version,
		})
	})

	// API routes
	setupAPIRoutes(router, authService, tunnelHandler, authHandler)

	// Prometheus metrics endpoint
	if cfg.Monitoring.PrometheusEnabled {
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if cfg.Server.TLS.Enabled {
			if err := server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start HTTPS server: %v", err)
			}
		} else {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start HTTP server: %v", err)
			}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cancel monitoring service context
	cancel()

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	// Build DSN
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Configure GORM logger
	var gormLogger logger.Interface
	if cfg.App.Debug {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.MaxLifetime)

	// Auto-migrate database schema
	if err := db.AutoMigrate(
		&models.User{},
		&models.Tunnel{},
		&models.TunnelLog{},
		&models.TunnelMetric{},
		&models.UserSession{},
		&models.AuditLog{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database initialized successfully")
	return db, nil
}

func initRedis(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis initialized successfully")
	return client
}

func setupAPIRoutes(router *gin.Engine, authService *services.AuthService, tunnelHandler *handlers.TunnelHandler, authHandler *handlers.AuthHandler) {
	api := router.Group("/api/v1")

	// Public routes
	public := api.Group("/")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/refresh", authHandler.RefreshToken)
		public.POST("/auth/forgot-password", authHandler.ForgotPassword)
		public.POST("/auth/reset-password", authHandler.ResetPassword)
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(authService))
	{
		// Auth routes
		auth := protected.Group("/auth")
		{
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/change-password", authHandler.ChangePassword)
			auth.GET("/profile", authHandler.GetProfile)
			auth.PUT("/profile", authHandler.UpdateProfile)
		}

		// Tunnel routes
		tunnels := protected.Group("/tunnels")
		{
			tunnels.GET("/", tunnelHandler.GetTunnels)
			tunnels.POST("/", tunnelHandler.CreateTunnel)
			tunnels.GET("/:id", tunnelHandler.GetTunnel)
			tunnels.PUT("/:id", tunnelHandler.UpdateTunnel)
			tunnels.DELETE("/:id", tunnelHandler.DeleteTunnel)
			tunnels.POST("/:id/start", tunnelHandler.StartTunnel)
			tunnels.POST("/:id/stop", tunnelHandler.StopTunnel)
			tunnels.GET("/:id/status", tunnelHandler.GetTunnelStatus)
			tunnels.GET("/:id/metrics", tunnelHandler.GetTunnelMetrics)
			tunnels.GET("/:id/logs", tunnelHandler.GetTunnelLogs)
		}

		// Dashboard routes
		dashboard := protected.Group("/dashboard")
		{
			dashboard.GET("/stats", tunnelHandler.GetDashboardStats)
			dashboard.GET("/activity", tunnelHandler.GetRecentActivity)
		}
	}

	// Admin routes
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(authService))
	admin.Use(middleware.AdminOnlyMiddleware())
	{
		admin.GET("/users", authHandler.GetUsers)
		admin.GET("/users/:id", authHandler.GetUser)
		admin.PUT("/users/:id", authHandler.UpdateUser)
		admin.DELETE("/users/:id", authHandler.DeleteUser)
		admin.GET("/system/stats", tunnelHandler.GetSystemStats)
		admin.GET("/audit-logs", authHandler.GetAuditLogs)
	}
}
