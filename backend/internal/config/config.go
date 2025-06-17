package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Server Configuration
	Server ServerConfig `mapstructure:"server"`
	
	// Database Configuration
	Database DatabaseConfig `mapstructure:"database"`
	
	// Redis Configuration
	Redis RedisConfig `mapstructure:"redis"`
	
	// Security Configuration
	Security SecurityConfig `mapstructure:"security"`
	
	// Telegram Configuration
	Telegram TelegramConfig `mapstructure:"telegram"`
	
	// Monitoring Configuration
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	
	// Application Configuration
	App AppConfig `mapstructure:"app"`
	
	// JWT Configuration
	JWTSecret string `mapstructure:"jwt_secret"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	TLS          TLSConfig     `mapstructure:"tls"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	PasswordMinLength    int           `mapstructure:"password_min_length"`
	MaxLoginAttempts     int           `mapstructure:"max_login_attempts"`
	LockoutDuration      time.Duration `mapstructure:"lockout_duration"`
	SessionTimeout       time.Duration `mapstructure:"session_timeout"`
	TwoFactorEnabled     bool          `mapstructure:"two_factor_enabled"`
	RateLimitEnabled     bool          `mapstructure:"rate_limit_enabled"`
	RateLimitRequests    int           `mapstructure:"rate_limit_requests"`
	RateLimitWindow      time.Duration `mapstructure:"rate_limit_window"`
	CORSAllowedOrigins   []string      `mapstructure:"cors_allowed_origins"`
	CORSAllowedMethods   []string      `mapstructure:"cors_allowed_methods"`
	CORSAllowedHeaders   []string      `mapstructure:"cors_allowed_headers"`
	CORSAllowCredentials bool          `mapstructure:"cors_allow_credentials"`
}

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	BotToken       string `mapstructure:"bot_token"`
	ChatID         string `mapstructure:"chat_id"`
	Enabled        bool   `mapstructure:"enabled"`
	WebhookURL     string `mapstructure:"webhook_url"`
	WebhookSecret  string `mapstructure:"webhook_secret"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	PrometheusEnabled bool          `mapstructure:"prometheus_enabled"`
	PrometheusPort    int           `mapstructure:"prometheus_port"`
	MetricsInterval   time.Duration `mapstructure:"metrics_interval"`
	HealthCheckPath   string        `mapstructure:"health_check_path"`
	LogLevel          string        `mapstructure:"log_level"`
	LogFormat         string        `mapstructure:"log_format"`
	LogOutput         string        `mapstructure:"log_output"`
}

// AppConfig holds application configuration
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	Debug       bool   `mapstructure:"debug"`
	TimeZone    string `mapstructure:"timezone"`
	Language    string `mapstructure:"language"`
}

// LoadConfig loads configuration from environment variables and config files
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Set default values
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")
	
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.max_lifetime", "5m")
	
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.min_idle_conns", 2)
	viper.SetDefault("redis.dial_timeout", "5s")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")
	
	viper.SetDefault("security.password_min_length", 8)
	viper.SetDefault("security.max_login_attempts", 5)
	viper.SetDefault("security.lockout_duration", "30m")
	viper.SetDefault("security.session_timeout", "24h")
	viper.SetDefault("security.rate_limit_enabled", true)
	viper.SetDefault("security.rate_limit_requests", 100)
	viper.SetDefault("security.rate_limit_window", "1m")
	viper.SetDefault("security.cors_allowed_origins", []string{"*"})
	viper.SetDefault("security.cors_allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("security.cors_allowed_headers", []string{"*"})
	viper.SetDefault("security.cors_allow_credentials", true)
	
	viper.SetDefault("monitoring.enabled", true)
	viper.SetDefault("monitoring.prometheus_enabled", true)
	viper.SetDefault("monitoring.prometheus_port", 9090)
	viper.SetDefault("monitoring.metrics_interval", "30s")
	viper.SetDefault("monitoring.health_check_path", "/health")
	viper.SetDefault("monitoring.log_level", "info")
	viper.SetDefault("monitoring.log_format", "json")
	viper.SetDefault("monitoring.log_output", "stdout")
	
	viper.SetDefault("app.name", "UTunnel Pro")
	viper.SetDefault("app.version", "2.0.0")
	viper.SetDefault("app.environment", "production")
	viper.SetDefault("app.debug", false)
	viper.SetDefault("app.timezone", "UTC")
	viper.SetDefault("app.language", "en")

	// Bind environment variables
	viper.BindEnv("server.host", "SERVER_HOST")
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.mode", "GIN_MODE")
	viper.BindEnv("server.tls.enabled", "TLS_ENABLED")
	viper.BindEnv("server.tls.cert_file", "TLS_CERT_FILE")
	viper.BindEnv("server.tls.key_file", "TLS_KEY_FILE")
	
	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("database.name", "DB_NAME")
	viper.BindEnv("database.ssl_mode", "DB_SSL_MODE")
	
	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.db", "REDIS_DB")
	
	viper.BindEnv("telegram.bot_token", "TELEGRAM_BOT_TOKEN")
	viper.BindEnv("telegram.chat_id", "TELEGRAM_CHAT_ID")
	
	viper.BindEnv("monitoring.log_level", "LOG_LEVEL")
	viper.BindEnv("app.environment", "ENVIRONMENT")
	viper.BindEnv("app.debug", "DEBUG")

	// Set config file paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/stunnel-pro")
	viper.AddConfigPath(".")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		log.Println("No config file found, using defaults and environment variables")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Override with environment variables
	config.JWTSecret = getEnvOrDefault("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production")
	
	// Enable Telegram if token is provided
	if config.Telegram.BotToken != "" && config.Telegram.ChatID != "" {
		config.Telegram.Enabled = true
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if config.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if config.JWTSecret == "" || config.JWTSecret == "your-super-secret-jwt-key-change-this-in-production" {
		log.Println("WARNING: Using default JWT secret. Please set JWT_SECRET environment variable in production!")
	}
	return nil
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsIntOrDefault gets environment variable as int or returns default value
func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBoolOrDefault gets environment variable as bool or returns default value
func getEnvAsBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
