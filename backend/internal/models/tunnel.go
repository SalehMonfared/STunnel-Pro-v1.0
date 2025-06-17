package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TunnelStatus represents the status of a tunnel
type TunnelStatus string

const (
	TunnelStatusActive    TunnelStatus = "active"
	TunnelStatusInactive  TunnelStatus = "inactive"
	TunnelStatusError     TunnelStatus = "error"
	TunnelStatusConnecting TunnelStatus = "connecting"
)

// TunnelProtocol represents supported tunnel protocols
type TunnelProtocol string

const (
	ProtocolTCP      TunnelProtocol = "tcp"
	ProtocolUDP      TunnelProtocol = "udp"
	ProtocolWS       TunnelProtocol = "ws"
	ProtocolWSS      TunnelProtocol = "wss"
	ProtocolTCPMux   TunnelProtocol = "tcpmux"
	ProtocolWSMux    TunnelProtocol = "wsmux"
	ProtocolWSSMux   TunnelProtocol = "wssmux"
	ProtocolUTCPMux  TunnelProtocol = "utcpmux"
	ProtocolUWSMux   TunnelProtocol = "uwsmux"
)

// Tunnel represents a tunnel configuration
type Tunnel struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string         `json:"name" gorm:"uniqueIndex;not null" validate:"required,min=3,max=50"`
	Description string         `json:"description" gorm:"type:text"`
	Protocol    TunnelProtocol `json:"protocol" gorm:"not null" validate:"required"`
	Status      TunnelStatus   `json:"status" gorm:"default:'inactive'"`
	
	// Server Configuration
	ServerIP     string `json:"server_ip" gorm:"not null" validate:"required,ip"`
	ServerPort   int    `json:"server_port" gorm:"not null" validate:"required,min=1,max=65535"`
	
	// Client Configuration  
	ClientIP     string `json:"client_ip" validate:"omitempty,ip"`
	ClientPort   int    `json:"client_port" validate:"omitempty,min=1,max=65535"`
	
	// Target Configuration
	TargetIP     string `json:"target_ip" gorm:"not null" validate:"required,ip"`
	TargetPort   int    `json:"target_port" gorm:"not null" validate:"required,min=1,max=65535"`
	
	// Authentication
	Token        string `json:"token" gorm:"not null" validate:"required,min=16"`
	
	// Advanced Configuration
	MuxConfig    MuxConfig `json:"mux_config" gorm:"embedded"`
	TLSConfig    TLSConfig `json:"tls_config" gorm:"embedded"`
	
	// Monitoring
	LastSeen     *time.Time `json:"last_seen"`
	BytesIn      int64      `json:"bytes_in" gorm:"default:0"`
	BytesOut     int64      `json:"bytes_out" gorm:"default:0"`
	ConnectionCount int     `json:"connection_count" gorm:"default:0"`
	
	// Metadata
	UserID       uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	User         User       `json:"user" gorm:"foreignKey:UserID"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	Logs         []TunnelLog     `json:"logs,omitempty" gorm:"foreignKey:TunnelID"`
	Metrics      []TunnelMetric  `json:"metrics,omitempty" gorm:"foreignKey:TunnelID"`
}

// MuxConfig represents multiplexing configuration
type MuxConfig struct {
	Enabled         bool `json:"enabled" gorm:"default:true"`
	Connections     int  `json:"connections" gorm:"default:8" validate:"min=1,max=100"`
	FrameSize       int  `json:"frame_size" gorm:"default:32768" validate:"min=1024,max=65536"`
	ReceiveBuffer   int  `json:"receive_buffer" gorm:"default:4194304" validate:"min=65536"`
	StreamBuffer    int  `json:"stream_buffer" gorm:"default:65536" validate:"min=32768"`
	Version         int  `json:"version" gorm:"default:2" validate:"min=1,max=2"`
	ChannelSize     int  `json:"channel_size" gorm:"default:2048" validate:"min=512,max=32768"`
	ConnectionPool  int  `json:"connection_pool" gorm:"default:8" validate:"min=1,max=50"`
	Heartbeat       int  `json:"heartbeat" gorm:"default:30" validate:"min=5,max=300"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled         bool   `json:"enabled" gorm:"default:false"`
	CertFile        string `json:"cert_file"`
	KeyFile         string `json:"key_file"`
	CAFile          string `json:"ca_file"`
	InsecureSkipVerify bool `json:"insecure_skip_verify" gorm:"default:false"`
	MinVersion      string `json:"min_version" gorm:"default:'1.2'"`
	MaxVersion      string `json:"max_version" gorm:"default:'1.3'"`
}

// TunnelLog represents tunnel activity logs
type TunnelLog struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TunnelID  uuid.UUID `json:"tunnel_id" gorm:"type:uuid;not null"`
	Level     string    `json:"level" gorm:"not null"` // INFO, WARN, ERROR
	Message   string    `json:"message" gorm:"type:text;not null"`
	Timestamp time.Time `json:"timestamp" gorm:"not null"`
	Metadata  string    `json:"metadata" gorm:"type:jsonb"` // Additional context as JSON
}

// TunnelMetric represents tunnel performance metrics
type TunnelMetric struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TunnelID        uuid.UUID `json:"tunnel_id" gorm:"type:uuid;not null"`
	Timestamp       time.Time `json:"timestamp" gorm:"not null"`
	BytesIn         int64     `json:"bytes_in"`
	BytesOut        int64     `json:"bytes_out"`
	ConnectionCount int       `json:"connection_count"`
	Latency         float64   `json:"latency"` // in milliseconds
	CPUUsage        float64   `json:"cpu_usage"` // percentage
	MemoryUsage     int64     `json:"memory_usage"` // in bytes
	ErrorCount      int       `json:"error_count"`
}

// BeforeCreate hook to generate UUID and token
func (t *Tunnel) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Token == "" {
		t.Token = generateSecureToken(32)
	}
	return nil
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// parseUUID safely parses a UUID string
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// GetOptimalMuxConfig returns optimal MUX configuration based on expected user count
func GetOptimalMuxConfig(userCount int) MuxConfig {
	switch {
	case userCount <= 50:
		return MuxConfig{
			Enabled:        true,
			Connections:    8,
			FrameSize:      16384,
			ReceiveBuffer:  4194304,
			StreamBuffer:   32768,
			Version:        2,
			ChannelSize:    2048,
			ConnectionPool: 8,
			Heartbeat:      40,
		}
	case userCount <= 200:
		return MuxConfig{
			Enabled:        true,
			Connections:    24,
			FrameSize:      32768,
			ReceiveBuffer:  16777216,
			StreamBuffer:   65536,
			Version:        2,
			ChannelSize:    8192,
			ConnectionPool: 16,
			Heartbeat:      25,
		}
	case userCount <= 500:
		return MuxConfig{
			Enabled:        true,
			Connections:    48,
			FrameSize:      65536,
			ReceiveBuffer:  50331648,
			StreamBuffer:   262144,
			Version:        2,
			ChannelSize:    24576,
			ConnectionPool: 32,
			Heartbeat:      10,
		}
	default:
		return MuxConfig{
			Enabled:        true,
			Connections:    64,
			FrameSize:      65536,
			ReceiveBuffer:  67108864,
			StreamBuffer:   524288,
			Version:        2,
			ChannelSize:    32768,
			ConnectionPool: 48,
			Heartbeat:      5,
		}
	}
}
