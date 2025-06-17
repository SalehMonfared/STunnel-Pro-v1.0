package services

import (
	"testing"
	"time"

	"utunnel-pro/internal/config"
	"utunnel-pro/internal/models"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type AuthServiceTestSuite struct {
	suite.Suite
	db          *gorm.DB
	redis       *redis.Client
	authService *AuthService
	config      *config.Config
}

func (suite *AuthServiceTestSuite) SetupSuite() {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)
	
	// Auto-migrate
	err = db.AutoMigrate(
		&models.User{},
		&models.UserSession{},
		&models.AuditLog{},
	)
	suite.Require().NoError(err)
	
	suite.db = db
	
	// Setup Redis mock (using miniredis for testing)
	suite.redis = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use different DB for testing
	})
	
	// Setup config
	suite.config = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
		Security: config.SecurityConfig{
			PasswordMinLength: 8,
			MaxLoginAttempts:  5,
			LockoutDuration:   30 * time.Minute,
			SessionTimeout:    24 * time.Hour,
		},
	}
	
	// Create auth service
	suite.authService = NewAuthService(suite.db, suite.redis, suite.config)
}

func (suite *AuthServiceTestSuite) TearDownSuite() {
	// Clean up
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
	suite.redis.Close()
}

func (suite *AuthServiceTestSuite) SetupTest() {
	// Clean up data before each test
	suite.db.Exec("DELETE FROM users")
	suite.db.Exec("DELETE FROM user_sessions")
	suite.db.Exec("DELETE FROM audit_logs")
}

func (suite *AuthServiceTestSuite) TestRegister() {
	// Test successful registration
	req := &RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}
	
	user, err := suite.authService.Register(req)
	
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	assert.Equal(suite.T(), req.Username, user.Username)
	assert.Equal(suite.T(), req.Email, user.Email)
	assert.Equal(suite.T(), req.FirstName, user.FirstName)
	assert.Equal(suite.T(), req.LastName, user.LastName)
	assert.Equal(suite.T(), models.RoleUser, user.Role)
	assert.Equal(suite.T(), models.StatusActive, user.Status)
	assert.Empty(suite.T(), user.Password) // Password should be cleared from response
	
	// Verify user exists in database
	var dbUser models.User
	err = suite.db.First(&dbUser, "username = ?", req.Username).Error
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), dbUser.Password) // Password should be hashed in DB
}

func (suite *AuthServiceTestSuite) TestRegisterDuplicateUsername() {
	// Create first user
	req1 := &RegisterRequest{
		Username:  "testuser",
		Email:     "test1@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}
	
	_, err := suite.authService.Register(req1)
	assert.NoError(suite.T(), err)
	
	// Try to create user with same username
	req2 := &RegisterRequest{
		Username:  "testuser",
		Email:     "test2@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User2",
	}
	
	_, err = suite.authService.Register(req2)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "already exists")
}

func (suite *AuthServiceTestSuite) TestRegisterDuplicateEmail() {
	// Create first user
	req1 := &RegisterRequest{
		Username:  "testuser1",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}
	
	_, err := suite.authService.Register(req1)
	assert.NoError(suite.T(), err)
	
	// Try to create user with same email
	req2 := &RegisterRequest{
		Username:  "testuser2",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User2",
	}
	
	_, err = suite.authService.Register(req2)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "already exists")
}

func (suite *AuthServiceTestSuite) TestLogin() {
	// Create user first
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Status:    models.StatusActive,
	}
	user.HashPassword()
	suite.db.Create(user)
	
	// Test successful login
	req := &LoginRequest{
		Username: "testuser",
		Password: "password123",
		Remember: false,
	}
	
	response, err := suite.authService.Login(req, "127.0.0.1", "test-agent")
	
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.NotEmpty(suite.T(), response.AccessToken)
	assert.NotEmpty(suite.T(), response.RefreshToken)
	assert.Greater(suite.T(), response.ExpiresIn, int64(0))
	assert.Equal(suite.T(), user.Username, response.User.Username)
	assert.Empty(suite.T(), response.User.Password) // Password should be cleared
}

func (suite *AuthServiceTestSuite) TestLoginInvalidCredentials() {
	// Create user first
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Status:    models.StatusActive,
	}
	user.HashPassword()
	suite.db.Create(user)
	
	// Test login with wrong password
	req := &LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
		Remember: false,
	}
	
	_, err := suite.authService.Login(req, "127.0.0.1", "test-agent")
	
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid credentials")
}

func (suite *AuthServiceTestSuite) TestLoginInactiveUser() {
	// Create inactive user
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Status:    models.StatusInactive,
	}
	user.HashPassword()
	suite.db.Create(user)
	
	// Test login with inactive user
	req := &LoginRequest{
		Username: "testuser",
		Password: "password123",
		Remember: false,
	}
	
	_, err := suite.authService.Login(req, "127.0.0.1", "test-agent")
	
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not active")
}

func (suite *AuthServiceTestSuite) TestLoginAccountLockout() {
	// Create user
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Status:    models.StatusActive,
	}
	user.HashPassword()
	suite.db.Create(user)
	
	// Make multiple failed login attempts
	req := &LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
		Remember: false,
	}
	
	for i := 0; i < 5; i++ {
		_, err := suite.authService.Login(req, "127.0.0.1", "test-agent")
		assert.Error(suite.T(), err)
	}
	
	// Account should now be locked
	req.Password = "password123" // Correct password
	_, err := suite.authService.Login(req, "127.0.0.1", "test-agent")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "locked")
}

func (suite *AuthServiceTestSuite) TestValidateToken() {
	// Create user and login to get token
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Status:    models.StatusActive,
	}
	user.HashPassword()
	suite.db.Create(user)
	
	req := &LoginRequest{
		Username: "testuser",
		Password: "password123",
		Remember: false,
	}
	
	response, err := suite.authService.Login(req, "127.0.0.1", "test-agent")
	assert.NoError(suite.T(), err)
	
	// Test token validation
	validatedUser, err := suite.authService.ValidateToken(response.AccessToken)
	
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), validatedUser)
	assert.Equal(suite.T(), user.Username, validatedUser.Username)
}

func (suite *AuthServiceTestSuite) TestValidateInvalidToken() {
	// Test with invalid token
	_, err := suite.authService.ValidateToken("invalid-token")
	
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid")
}

func (suite *AuthServiceTestSuite) TestChangePassword() {
	// Create user
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "oldpassword123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Status:    models.StatusActive,
	}
	user.HashPassword()
	suite.db.Create(user)
	
	// Test password change
	err := suite.authService.ChangePassword(user.ID, "oldpassword123", "newpassword123")
	
	assert.NoError(suite.T(), err)
	
	// Verify old password no longer works
	req := &LoginRequest{
		Username: "testuser",
		Password: "oldpassword123",
		Remember: false,
	}
	
	_, err = suite.authService.Login(req, "127.0.0.1", "test-agent")
	assert.Error(suite.T(), err)
	
	// Verify new password works
	req.Password = "newpassword123"
	_, err = suite.authService.Login(req, "127.0.0.1", "test-agent")
	assert.NoError(suite.T(), err)
}

func (suite *AuthServiceTestSuite) TestChangePasswordWrongOldPassword() {
	// Create user
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "oldpassword123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Status:    models.StatusActive,
	}
	user.HashPassword()
	suite.db.Create(user)
	
	// Test password change with wrong old password
	err := suite.authService.ChangePassword(user.ID, "wrongoldpassword", "newpassword123")
	
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid current password")
}

func (suite *AuthServiceTestSuite) TestLogout() {
	// Create user and login
	user := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Status:    models.StatusActive,
	}
	user.HashPassword()
	suite.db.Create(user)
	
	req := &LoginRequest{
		Username: "testuser",
		Password: "password123",
		Remember: false,
	}
	
	response, err := suite.authService.Login(req, "127.0.0.1", "test-agent")
	assert.NoError(suite.T(), err)
	
	// Test logout
	err = suite.authService.Logout(response.AccessToken)
	assert.NoError(suite.T(), err)
	
	// Token should no longer be valid
	_, err = suite.authService.ValidateToken(response.AccessToken)
	assert.Error(suite.T(), err)
}

// Run the test suite
func TestAuthServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}
