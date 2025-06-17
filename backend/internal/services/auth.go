package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"utunnel-pro/internal/models"
	"utunnel-pro/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService handles authentication and authorization
type AuthService struct {
	db     *gorm.DB
	redis  *redis.Client
	config *config.Config
}

// LoginRequest represents login request data
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Remember bool   `json:"remember"`
}

// LoginResponse represents login response data
type LoginResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
}

// RegisterRequest represents registration request data
type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=30"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required,min=2,max=50"`
	LastName  string `json:"last_name" binding:"required,min=2,max=50"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID   uuid.UUID       `json:"user_id"`
	Username string          `json:"username"`
	Role     models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// NewAuthService creates a new auth service
func NewAuthService(db *gorm.DB, redis *redis.Client, config *config.Config) *AuthService {
	return &AuthService{
		db:     db,
		redis:  redis,
		config: config,
	}
}

// Register creates a new user account
func (s *AuthService) Register(req *RegisterRequest) (*models.User, error) {
	// Check if username already exists
	var existingUser models.User
	if err := s.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("username or email already exists")
	}

	// Create new user
	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      models.RoleUser,
		Status:    models.StatusActive,
		Language:  "en",
		Timezone:  "UTC",
		Theme:     "light",
		Limits:    models.GetDefaultLimitsByRole(models.RoleUser),
	}

	// Hash password
	if err := user.HashPassword(); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Save to database
	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Remove password from response
	user.Password = ""

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// Find user
	var user models.User
	if err := s.db.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is locked
	if user.IsLocked() {
		return nil, fmt.Errorf("account is locked until %v", user.LockedUntil)
	}

	// Check if user is active
	if user.Status != models.StatusActive {
		return nil, fmt.Errorf("account is not active")
	}

	// Verify password
	if !user.CheckPassword(req.Password) {
		// Increment failed login attempts
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= 5 {
			lockUntil := time.Now().Add(30 * time.Minute)
			user.LockedUntil = &lockUntil
		}
		s.db.Save(&user)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset failed login attempts
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = ipAddress
	s.db.Save(&user)

	// Generate tokens
	accessToken, refreshToken, expiresIn, err := s.generateTokens(&user, req.Remember)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	session := &models.UserSession{
		UserID:       user.ID,
		Token:        accessToken,
		RefreshToken: refreshToken,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsActive:     true,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
	s.db.Create(session)

	// Store session in Redis
	sessionData := map[string]interface{}{
		"user_id":    user.ID.String(),
		"username":   user.Username,
		"role":       user.Role,
		"ip_address": ipAddress,
		"user_agent": userAgent,
	}
	s.redis.HMSet(context.Background(), fmt.Sprintf("session:%s", accessToken), sessionData)
	s.redis.Expire(context.Background(), fmt.Sprintf("session:%s", accessToken), time.Duration(expiresIn)*time.Second)

	// Remove password from response
	user.Password = ""

	return &LoginResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// RefreshToken generates new tokens using refresh token
func (s *AuthService) RefreshToken(refreshToken string) (*LoginResponse, error) {
	// Verify refresh token
	claims, err := s.verifyToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Find user
	var user models.User
	if err := s.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if user is active
	if user.Status != models.StatusActive {
		return nil, fmt.Errorf("account is not active")
	}

	// Generate new tokens
	accessToken, newRefreshToken, expiresIn, err := s.generateTokens(&user, true)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update session
	s.db.Model(&models.UserSession{}).Where("refresh_token = ?", refreshToken).Updates(map[string]interface{}{
		"token":         accessToken,
		"refresh_token": newRefreshToken,
		"expires_at":    time.Now().Add(time.Duration(expiresIn) * time.Second),
		"last_used_at":  time.Now(),
	})

	// Remove password from response
	user.Password = ""

	return &LoginResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// Logout invalidates user session
func (s *AuthService) Logout(token string) error {
	// Remove from Redis
	s.redis.Del(context.Background(), fmt.Sprintf("session:%s", token))

	// Deactivate session in database
	s.db.Model(&models.UserSession{}).Where("token = ?", token).Update("is_active", false)

	return nil
}

// ValidateToken validates JWT token and returns user
func (s *AuthService) ValidateToken(tokenString string) (*models.User, error) {
	// Check if token exists in Redis (for quick validation)
	sessionData := s.redis.HMGetAll(context.Background(), fmt.Sprintf("session:%s", tokenString))
	if len(sessionData.Val()) == 0 {
		return nil, fmt.Errorf("invalid or expired token")
	}

	// Verify JWT token
	claims, err := s.verifyToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Find user
	var user models.User
	if err := s.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if user is active
	if user.Status != models.StatusActive {
		return nil, fmt.Errorf("account is not active")
	}

	// Update last used time
	s.redis.HSet(context.Background(), fmt.Sprintf("session:%s", tokenString), "last_used", time.Now().Unix())

	return &user, nil
}

// ChangePassword changes user password
func (s *AuthService) ChangePassword(userID uuid.UUID, oldPassword, newPassword string) error {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify old password
	if !user.CheckPassword(oldPassword) {
		return fmt.Errorf("invalid current password")
	}

	// Update password
	user.Password = newPassword
	if err := user.HashPassword(); err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Save to database
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all user sessions
	s.invalidateUserSessions(userID)

	return nil
}

// ResetPassword initiates password reset process
func (s *AuthService) ResetPassword(email string) error {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		// Don't reveal if email exists or not
		return nil
	}

	// Generate reset token
	resetToken := s.generateResetToken()

	// Store reset token in Redis (expires in 1 hour)
	s.redis.Set(context.Background(), fmt.Sprintf("reset:%s", resetToken), user.ID.String(), time.Hour)

	// TODO: Send email with reset link
	// emailService.SendPasswordResetEmail(user.Email, resetToken)

	return nil
}

// ConfirmPasswordReset confirms password reset with token
func (s *AuthService) ConfirmPasswordReset(token, newPassword string) error {
	// Get user ID from reset token
	userIDStr, err := s.redis.Get(context.Background(), fmt.Sprintf("reset:%s", token)).Result()
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	// Find user
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	// Update password
	user.Password = newPassword
	if err := user.HashPassword(); err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Save to database
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Delete reset token
	s.redis.Del(context.Background(), fmt.Sprintf("reset:%s", token))

	// Invalidate all user sessions
	s.invalidateUserSessions(userID)

	return nil
}

// Private methods

func (s *AuthService) generateTokens(user *models.User, remember bool) (string, string, int64, error) {
	// Set expiration time
	var expiresIn int64 = 3600 // 1 hour
	if remember {
		expiresIn = 3600 * 24 * 30 // 30 days
	}

	// Create access token claims
	accessClaims := &TokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "utunnel-pro",
			Subject:   user.ID.String(),
		},
	}

	// Create refresh token claims (longer expiration)
	refreshClaims := &TokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn*2) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "utunnel-pro",
			Subject:   user.ID.String(),
		},
	}

	// Generate tokens
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	// Sign tokens
	accessTokenString, err := accessToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", "", 0, err
	}

	refreshTokenString, err := refreshToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", "", 0, err
	}

	return accessTokenString, refreshTokenString, expiresIn, nil
}

func (s *AuthService) verifyToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *AuthService) generateResetToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func (s *AuthService) invalidateUserSessions(userID uuid.UUID) {
	// Get all user sessions
	var sessions []models.UserSession
	s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&sessions)

	// Remove from Redis and deactivate in database
	for _, session := range sessions {
		s.redis.Del(context.Background(), fmt.Sprintf("session:%s", session.Token))
	}

	// Deactivate all sessions
	s.db.Model(&models.UserSession{}).Where("user_id = ?", userID).Update("is_active", false)
}
