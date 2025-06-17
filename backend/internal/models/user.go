package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
)

// UserRole represents user roles in the system
type UserRole string

const (
	RoleAdmin     UserRole = "admin"
	RoleModerator UserRole = "moderator"
	RoleUser      UserRole = "user"
	RoleGuest     UserRole = "guest"
)

// UserStatus represents user account status
type UserStatus string

const (
	StatusActive    UserStatus = "active"
	StatusInactive  UserStatus = "inactive"
	StatusSuspended UserStatus = "suspended"
	StatusBanned    UserStatus = "banned"
)

// User represents a system user
type User struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username    string     `json:"username" gorm:"uniqueIndex;not null" validate:"required,min=3,max=30,alphanum"`
	Email       string     `json:"email" gorm:"uniqueIndex;not null" validate:"required,email"`
	Password    string     `json:"-" gorm:"not null" validate:"required,min=8"`
	FirstName   string     `json:"first_name" validate:"required,min=2,max=50"`
	LastName    string     `json:"last_name" validate:"required,min=2,max=50"`
	
	// Account Information
	Role        UserRole   `json:"role" gorm:"default:'user'" validate:"required"`
	Status      UserStatus `json:"status" gorm:"default:'active'"`
	
	// Profile
	Avatar      string     `json:"avatar"`
	Phone       string     `json:"phone" validate:"omitempty,e164"`
	Company     string     `json:"company"`
	Department  string     `json:"department"`
	
	// Preferences
	Language    string     `json:"language" gorm:"default:'en'" validate:"len=2"`
	Timezone    string     `json:"timezone" gorm:"default:'UTC'"`
	Theme       string     `json:"theme" gorm:"default:'light'" validate:"oneof=light dark auto"`
	
	// Security
	TwoFactorEnabled    bool       `json:"two_factor_enabled" gorm:"default:false"`
	TwoFactorSecret     string     `json:"-"`
	LastLoginAt         *time.Time `json:"last_login_at"`
	LastLoginIP         string     `json:"last_login_ip"`
	PasswordChangedAt   time.Time  `json:"password_changed_at"`
	FailedLoginAttempts int        `json:"failed_login_attempts" gorm:"default:0"`
	LockedUntil         *time.Time `json:"locked_until"`
	
	// Limits and Quotas
	Limits      UserLimits `json:"limits" gorm:"embedded"`
	
	// API Access
	APIKey      string     `json:"api_key" gorm:"uniqueIndex"`
	APIKeyCreatedAt *time.Time `json:"api_key_created_at"`
	
	// Metadata
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	Tunnels     []Tunnel       `json:"tunnels,omitempty" gorm:"foreignKey:UserID"`
	Sessions    []UserSession  `json:"sessions,omitempty" gorm:"foreignKey:UserID"`
	AuditLogs   []AuditLog     `json:"audit_logs,omitempty" gorm:"foreignKey:UserID"`
}

// UserLimits represents user resource limits
type UserLimits struct {
	MaxTunnels          int   `json:"max_tunnels" gorm:"default:10" validate:"min=0,max=1000"`
	MaxBandwidthMBps    int   `json:"max_bandwidth_mbps" gorm:"default:100" validate:"min=0"`
	MaxConnections      int   `json:"max_connections" gorm:"default:1000" validate:"min=0"`
	MaxStorageGB        int   `json:"max_storage_gb" gorm:"default:10" validate:"min=0"`
	DailyTransferGB     int   `json:"daily_transfer_gb" gorm:"default:100" validate:"min=0"`
	MonthlyTransferGB   int   `json:"monthly_transfer_gb" gorm:"default:1000" validate:"min=0"`
	CanCreatePublicTunnels bool `json:"can_create_public_tunnels" gorm:"default:false"`
	CanUseCustomDomains    bool `json:"can_use_custom_domains" gorm:"default:false"`
	CanAccessAPI           bool `json:"can_access_api" gorm:"default:true"`
}

// UserSession represents an active user session
type UserSession struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	Token       string    `json:"token" gorm:"uniqueIndex;not null"`
	RefreshToken string   `json:"refresh_token" gorm:"uniqueIndex"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	DeviceInfo  string    `json:"device_info"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
}

// AuditLog represents user activity audit log
type AuditLog struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	Action      string    `json:"action" gorm:"not null"` // login, logout, create_tunnel, etc.
	Resource    string    `json:"resource"`               // tunnel, user, etc.
	ResourceID  string    `json:"resource_id"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Success     bool      `json:"success" gorm:"default:true"`
	ErrorMessage string   `json:"error_message"`
	Metadata    string    `json:"metadata" gorm:"type:jsonb"` // Additional context as JSON
	Timestamp   time.Time `json:"timestamp" gorm:"not null"`
}

// BeforeCreate hook to generate UUID and API key
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.APIKey == "" {
		u.APIKey = generateAPIKey()
		now := time.Now()
		u.APIKeyCreatedAt = &now
	}
	u.PasswordChangedAt = time.Now()
	return nil
}

// BeforeUpdate hook to track password changes
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement.Changed("Password") {
		u.PasswordChangedAt = time.Now()
	}
	return nil
}

// HashPassword hashes the user's password
func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifies the provided password against the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// IsLocked checks if the user account is locked
func (u *User) IsLocked() bool {
	return u.LockedUntil != nil && u.LockedUntil.After(time.Now())
}

// CanPerformAction checks if user can perform a specific action based on role
func (u *User) CanPerformAction(action string) bool {
	switch u.Role {
	case RoleAdmin:
		return true // Admin can do everything
	case RoleModerator:
		moderatorActions := []string{
			"view_all_tunnels", "manage_users", "view_logs", "manage_tunnels",
		}
		return contains(moderatorActions, action)
	case RoleUser:
		userActions := []string{
			"create_tunnel", "manage_own_tunnels", "view_own_logs", "update_profile",
		}
		return contains(userActions, action)
	case RoleGuest:
		guestActions := []string{
			"view_public_info", "update_profile",
		}
		return contains(guestActions, action)
	default:
		return false
	}
}

// GetDefaultLimitsByRole returns default limits based on user role
func GetDefaultLimitsByRole(role UserRole) UserLimits {
	switch role {
	case RoleAdmin:
		return UserLimits{
			MaxTunnels:             1000,
			MaxBandwidthMBps:       10000,
			MaxConnections:         100000,
			MaxStorageGB:           1000,
			DailyTransferGB:        10000,
			MonthlyTransferGB:      100000,
			CanCreatePublicTunnels: true,
			CanUseCustomDomains:    true,
			CanAccessAPI:           true,
		}
	case RoleModerator:
		return UserLimits{
			MaxTunnels:             100,
			MaxBandwidthMBps:       1000,
			MaxConnections:         10000,
			MaxStorageGB:           100,
			DailyTransferGB:        1000,
			MonthlyTransferGB:      10000,
			CanCreatePublicTunnels: true,
			CanUseCustomDomains:    true,
			CanAccessAPI:           true,
		}
	case RoleUser:
		return UserLimits{
			MaxTunnels:             10,
			MaxBandwidthMBps:       100,
			MaxConnections:         1000,
			MaxStorageGB:           10,
			DailyTransferGB:        100,
			MonthlyTransferGB:      1000,
			CanCreatePublicTunnels: false,
			CanUseCustomDomains:    false,
			CanAccessAPI:           true,
		}
	case RoleGuest:
		return UserLimits{
			MaxTunnels:             1,
			MaxBandwidthMBps:       10,
			MaxConnections:         100,
			MaxStorageGB:           1,
			DailyTransferGB:        10,
			MonthlyTransferGB:      100,
			CanCreatePublicTunnels: false,
			CanUseCustomDomains:    false,
			CanAccessAPI:           false,
		}
	default:
		return UserLimits{}
	}
}

// generateAPIKey generates a secure API key
func generateAPIKey() string {
	return "utpro_" + uuid.New().String()
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
