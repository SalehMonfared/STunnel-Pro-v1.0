package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// SuccessResponse sends a successful response
func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	response := APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	c.JSON(statusCode, response)
}

// ErrorResponse sends an error response
func ErrorResponse(c *gin.Context, statusCode int, message string, err error) {
	errorInfo := &ErrorInfo{
		Code:    getErrorCode(statusCode),
		Message: message,
	}

	if err != nil {
		errorInfo.Details = err.Error()
	}

	response := APIResponse{
		Success: false,
		Message: message,
		Error:   errorInfo,
	}

	c.JSON(statusCode, response)
}

// ValidationErrorResponse sends a validation error response
func ValidationErrorResponse(c *gin.Context, errors map[string]string) {
	errorInfo := &ErrorInfo{
		Code:    "VALIDATION_ERROR",
		Message: "Validation failed",
		Details: errors,
	}

	response := APIResponse{
		Success: false,
		Message: "Validation failed",
		Error:   errorInfo,
	}

	c.JSON(http.StatusBadRequest, response)
}

// PaginatedResponse sends a paginated response
func PaginatedResponse(c *gin.Context, statusCode int, message string, data interface{}, total int64, page, limit int) {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	
	pagination := Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	response := PaginatedResponse{
		Success:    true,
		Message:    message,
		Data:       data,
		Pagination: pagination,
	}

	c.JSON(statusCode, response)
}

// NotFoundResponse sends a not found response
func NotFoundResponse(c *gin.Context, resource string) {
	ErrorResponse(c, http.StatusNotFound, resource+" not found", nil)
}

// UnauthorizedResponse sends an unauthorized response
func UnauthorizedResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Unauthorized access"
	}
	ErrorResponse(c, http.StatusUnauthorized, message, nil)
}

// ForbiddenResponse sends a forbidden response
func ForbiddenResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Access forbidden"
	}
	ErrorResponse(c, http.StatusForbidden, message, nil)
}

// InternalServerErrorResponse sends an internal server error response
func InternalServerErrorResponse(c *gin.Context, err error) {
	ErrorResponse(c, http.StatusInternalServerError, "Internal server error", err)
}

// BadRequestResponse sends a bad request response
func BadRequestResponse(c *gin.Context, message string, err error) {
	if message == "" {
		message = "Bad request"
	}
	ErrorResponse(c, http.StatusBadRequest, message, err)
}

// ConflictResponse sends a conflict response
func ConflictResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Resource conflict"
	}
	ErrorResponse(c, http.StatusConflict, message, nil)
}

// TooManyRequestsResponse sends a rate limit exceeded response
func TooManyRequestsResponse(c *gin.Context) {
	ErrorResponse(c, http.StatusTooManyRequests, "Rate limit exceeded", nil)
}

// ServiceUnavailableResponse sends a service unavailable response
func ServiceUnavailableResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	ErrorResponse(c, http.StatusServiceUnavailable, message, nil)
}

// getErrorCode returns error code based on HTTP status code
func getErrorCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusTooManyRequests:
		return "RATE_LIMIT_EXCEEDED"
	case http.StatusInternalServerError:
		return "INTERNAL_SERVER_ERROR"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return "UNKNOWN_ERROR"
	}
}
