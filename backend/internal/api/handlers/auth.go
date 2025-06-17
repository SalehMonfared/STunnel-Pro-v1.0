package handlers

import (
	"net/http"
	"strconv"

	"utunnel-pro/internal/models"
	"utunnel-pro/internal/services"
	"utunnel-pro/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param user body services.RegisterRequest true "User registration data"
// @Success 201 {object} models.User
// @Failure 400 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body", err)
		return
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		if err.Error() == "username or email already exists" {
			utils.ConflictResponse(c, err.Error())
		} else {
			utils.InternalServerErrorResponse(c, err)
		}
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "User registered successfully", user)
}

// Login handles user login
// @Summary User login
// @Description Authenticate user and return tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body services.LoginRequest true "Login credentials"
// @Success 200 {object} services.LoginResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body", err)
		return
	}

	// Get client info
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	response, err := h.authService.Login(&req, ipAddress, userAgent)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Login successful", response)
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Refresh access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param token body object{refresh_token=string} true "Refresh token"
// @Success 200 {object} services.LoginResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body", err)
		return
	}

	response, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Token refreshed successfully", response)
}

// Logout handles user logout
// @Summary User logout
// @Description Logout user and invalidate session
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} utils.APIResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	token, exists := c.Get("token")
	if !exists {
		utils.UnauthorizedResponse(c, "Token not found")
		return
	}

	tokenString := token.(string)
	if err := h.authService.Logout(tokenString); err != nil {
		utils.InternalServerErrorResponse(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Logout successful", nil)
}

// ChangePassword handles password change
// @Summary Change user password
// @Description Change current user's password
// @Tags auth
// @Accept json
// @Produce json
// @Param passwords body object{old_password=string,new_password=string} true "Password change data"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body", err)
		return
	}

	// Get user from context
	user, exists := c.Get("user")
	if !exists {
		utils.UnauthorizedResponse(c, "User not found in context")
		return
	}
	currentUser := user.(*models.User)

	if err := h.authService.ChangePassword(currentUser.ID, req.OldPassword, req.NewPassword); err != nil {
		if err.Error() == "invalid current password" {
			utils.BadRequestResponse(c, err.Error(), nil)
		} else {
			utils.InternalServerErrorResponse(c, err)
		}
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Password changed successfully", nil)
}

// ForgotPassword handles password reset request
// @Summary Request password reset
// @Description Send password reset email
// @Tags auth
// @Accept json
// @Produce json
// @Param email body object{email=string} true "Email address"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body", err)
		return
	}

	if err := h.authService.ResetPassword(req.Email); err != nil {
		utils.InternalServerErrorResponse(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Password reset email sent", nil)
}

// ResetPassword handles password reset confirmation
// @Summary Reset password with token
// @Description Reset password using reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param reset body object{token=string,new_password=string} true "Reset data"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body", err)
		return
	}

	if err := h.authService.ConfirmPasswordReset(req.Token, req.NewPassword); err != nil {
		if err.Error() == "invalid or expired reset token" {
			utils.BadRequestResponse(c, err.Error(), nil)
		} else {
			utils.InternalServerErrorResponse(c, err)
		}
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Password reset successfully", nil)
}

// GetProfile returns current user profile
// @Summary Get user profile
// @Description Get current user's profile information
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} models.User
// @Failure 401 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		utils.UnauthorizedResponse(c, "User not found in context")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Profile retrieved successfully", user)
}

// UpdateProfile updates current user profile
// @Summary Update user profile
// @Description Update current user's profile information
// @Tags auth
// @Accept json
// @Produce json
// @Param profile body object true "Profile update data"
// @Success 200 {object} models.User
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req struct {
		FirstName  *string `json:"first_name,omitempty"`
		LastName   *string `json:"last_name,omitempty"`
		Email      *string `json:"email,omitempty" binding:"omitempty,email"`
		Phone      *string `json:"phone,omitempty"`
		Company    *string `json:"company,omitempty"`
		Department *string `json:"department,omitempty"`
		Language   *string `json:"language,omitempty"`
		Timezone   *string `json:"timezone,omitempty"`
		Theme      *string `json:"theme,omitempty" binding:"omitempty,oneof=light dark auto"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "Invalid request body", err)
		return
	}

	// Get user from context
	user, exists := c.Get("user")
	if !exists {
		utils.UnauthorizedResponse(c, "User not found in context")
		return
	}
	currentUser := user.(*models.User)

	// TODO: Implement profile update logic
	// This would involve updating the user in the database

	utils.SuccessResponse(c, http.StatusOK, "Profile updated successfully", currentUser)
}

// Admin-only handlers

// GetUsers returns all users (admin only)
func (h *AuthHandler) GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// TODO: Implement user listing logic
	utils.SuccessResponse(c, http.StatusOK, "Users retrieved successfully", []models.User{})
}

// GetUser returns a specific user (admin only)
func (h *AuthHandler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", err)
		return
	}

	// TODO: Implement get user logic
	_ = userID
	utils.SuccessResponse(c, http.StatusOK, "User retrieved successfully", nil)
}

// UpdateUser updates a user (admin only)
func (h *AuthHandler) UpdateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", err)
		return
	}

	// TODO: Implement user update logic
	_ = userID
	utils.SuccessResponse(c, http.StatusOK, "User updated successfully", nil)
}

// DeleteUser deletes a user (admin only)
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequestResponse(c, "Invalid user ID", err)
		return
	}

	// TODO: Implement user deletion logic
	_ = userID
	utils.SuccessResponse(c, http.StatusOK, "User deleted successfully", nil)
}

// GetAuditLogs returns audit logs (admin only)
func (h *AuthHandler) GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// TODO: Implement audit logs logic
	_ = page
	_ = limit
	utils.SuccessResponse(c, http.StatusOK, "Audit logs retrieved successfully", []models.AuditLog{})
}
