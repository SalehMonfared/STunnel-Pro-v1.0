package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"utunnel-pro/internal/models"
	"utunnel-pro/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TunnelService handles tunnel operations
type TunnelService struct {
	db          *gorm.DB
	redis       *redis.Client
	config      *config.Config
	activeTunnels map[string]*TunnelProcess
	tunnelsMux    sync.RWMutex
}

// TunnelProcess represents an active tunnel process
type TunnelProcess struct {
	ID          string
	Tunnel      *models.Tunnel
	Process     *exec.Cmd
	Status      models.TunnelStatus
	StartedAt   time.Time
	LastPing    time.Time
	Metrics     *TunnelMetrics
	StopChannel chan bool
}

// TunnelMetrics represents tunnel performance metrics
type TunnelMetrics struct {
	BytesIn         int64   `json:"bytes_in"`
	BytesOut        int64   `json:"bytes_out"`
	ConnectionCount int     `json:"connection_count"`
	Latency         float64 `json:"latency"`
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     int64   `json:"memory_usage"`
	ErrorCount      int     `json:"error_count"`
	LastUpdated     time.Time `json:"last_updated"`
}

// NewTunnelService creates a new tunnel service
func NewTunnelService(db *gorm.DB, redis *redis.Client, config *config.Config) *TunnelService {
	return &TunnelService{
		db:            db,
		redis:         redis,
		config:        config,
		activeTunnels: make(map[string]*TunnelProcess),
	}
}

// CreateTunnel creates a new tunnel
func (s *TunnelService) CreateTunnel(tunnel *models.Tunnel) (*models.Tunnel, error) {
	// Validate tunnel configuration
	if err := s.validateTunnelConfig(tunnel); err != nil {
		return nil, fmt.Errorf("invalid tunnel configuration: %w", err)
	}

	// Check if tunnel name already exists for this user
	var existingTunnel models.Tunnel
	if err := s.db.Where("name = ? AND user_id = ?", tunnel.Name, tunnel.UserID).First(&existingTunnel).Error; err == nil {
		return nil, fmt.Errorf("tunnel with name '%s' already exists", tunnel.Name)
	}

	// Create tunnel in database
	if err := s.db.Create(tunnel).Error; err != nil {
		return nil, fmt.Errorf("failed to create tunnel: %w", err)
	}

	// Cache tunnel configuration in Redis
	tunnelJSON, _ := json.Marshal(tunnel)
	s.redis.Set(context.Background(), fmt.Sprintf("tunnel:config:%s", tunnel.ID), tunnelJSON, 0)

	log.Printf("Tunnel created: %s (%s)", tunnel.Name, tunnel.ID)
	return tunnel, nil
}

// GetTunnels retrieves tunnels with pagination and filtering
func (s *TunnelService) GetTunnels(filters map[string]interface{}, page, limit int) ([]models.Tunnel, int64, error) {
	var tunnels []models.Tunnel
	var total int64

	query := s.db.Model(&models.Tunnel{})

	// Apply filters
	for key, value := range filters {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}

	// Get total count
	query.Count(&total)

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Preload("User").Find(&tunnels).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve tunnels: %w", err)
	}

	return tunnels, total, nil
}

// GetTunnelByID retrieves a tunnel by ID
func (s *TunnelService) GetTunnelByID(id uuid.UUID) (*models.Tunnel, error) {
	var tunnel models.Tunnel
	if err := s.db.Preload("User").First(&tunnel, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("tunnel not found: %w", err)
	}
	return &tunnel, nil
}

// UpdateTunnel updates a tunnel configuration
func (s *TunnelService) UpdateTunnel(id uuid.UUID, updates map[string]interface{}) (*models.Tunnel, error) {
	var tunnel models.Tunnel
	if err := s.db.First(&tunnel, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("tunnel not found: %w", err)
	}

	// Stop tunnel if it's running
	if tunnel.Status == models.TunnelStatusActive {
		if err := s.StopTunnel(id); err != nil {
			log.Printf("Warning: failed to stop tunnel before update: %v", err)
		}
	}

	// Update tunnel
	if err := s.db.Model(&tunnel).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update tunnel: %w", err)
	}

	// Update cache
	tunnelJSON, _ := json.Marshal(&tunnel)
	s.redis.Set(context.Background(), fmt.Sprintf("tunnel:config:%s", tunnel.ID), tunnelJSON, 0)

	log.Printf("Tunnel updated: %s (%s)", tunnel.Name, tunnel.ID)
	return &tunnel, nil
}

// DeleteTunnel deletes a tunnel
func (s *TunnelService) DeleteTunnel(id uuid.UUID) error {
	var tunnel models.Tunnel
	if err := s.db.First(&tunnel, "id = ?", id).Error; err != nil {
		return fmt.Errorf("tunnel not found: %w", err)
	}

	// Stop tunnel if it's running
	if tunnel.Status == models.TunnelStatusActive {
		if err := s.StopTunnel(id); err != nil {
			log.Printf("Warning: failed to stop tunnel before deletion: %v", err)
		}
	}

	// Delete from database
	if err := s.db.Delete(&tunnel).Error; err != nil {
		return fmt.Errorf("failed to delete tunnel: %w", err)
	}

	// Remove from cache
	s.redis.Del(context.Background(), fmt.Sprintf("tunnel:config:%s", tunnel.ID))

	log.Printf("Tunnel deleted: %s (%s)", tunnel.Name, tunnel.ID)
	return nil
}

// StartTunnel starts a tunnel
func (s *TunnelService) StartTunnel(id uuid.UUID) error {
	tunnel, err := s.GetTunnelByID(id)
	if err != nil {
		return err
	}

	s.tunnelsMux.Lock()
	defer s.tunnelsMux.Unlock()

	// Check if tunnel is already running
	if _, exists := s.activeTunnels[tunnel.ID.String()]; exists {
		return fmt.Errorf("tunnel is already running")
	}

	// Create tunnel process
	process, err := s.createTunnelProcess(tunnel)
	if err != nil {
		return fmt.Errorf("failed to create tunnel process: %w", err)
	}

	// Start the process
	if err := process.Process.Start(); err != nil {
		return fmt.Errorf("failed to start tunnel process: %w", err)
	}

	// Update tunnel status
	s.db.Model(tunnel).Update("status", models.TunnelStatusActive)

	// Store active tunnel
	s.activeTunnels[tunnel.ID.String()] = process

	// Start monitoring
	go s.monitorTunnel(process)

	log.Printf("Tunnel started: %s (%s)", tunnel.Name, tunnel.ID)
	return nil
}

// StopTunnel stops a tunnel
func (s *TunnelService) StopTunnel(id uuid.UUID) error {
	tunnel, err := s.GetTunnelByID(id)
	if err != nil {
		return err
	}

	s.tunnelsMux.Lock()
	defer s.tunnelsMux.Unlock()

	// Check if tunnel is running
	process, exists := s.activeTunnels[tunnel.ID.String()]
	if !exists {
		return fmt.Errorf("tunnel is not running")
	}

	// Stop the process
	if process.Process != nil && process.Process.Process != nil {
		if err := process.Process.Process.Kill(); err != nil {
			log.Printf("Warning: failed to kill tunnel process: %v", err)
		}
	}

	// Signal stop
	close(process.StopChannel)

	// Update tunnel status
	s.db.Model(tunnel).Update("status", models.TunnelStatusInactive)

	// Remove from active tunnels
	delete(s.activeTunnels, tunnel.ID.String())

	log.Printf("Tunnel stopped: %s (%s)", tunnel.Name, tunnel.ID)
	return nil
}

// GetTunnelStatus returns tunnel status and last ping
func (s *TunnelService) GetTunnelStatus(id uuid.UUID) (bool, *time.Time) {
	s.tunnelsMux.RLock()
	defer s.tunnelsMux.RUnlock()

	if process, exists := s.activeTunnels[id.String()]; exists {
		return true, &process.LastPing
	}
	return false, nil
}

// GetUserTunnelCount returns the number of tunnels for a user
func (s *TunnelService) GetUserTunnelCount(userID uuid.UUID) (int, error) {
	var count int64
	if err := s.db.Model(&models.Tunnel{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetPerformanceMetrics returns performance metrics for a tunnel
func (s *TunnelService) GetPerformanceMetrics(id uuid.UUID) (*PerformanceMetrics, error) {
	s.tunnelsMux.RLock()
	defer s.tunnelsMux.RUnlock()

	if process, exists := s.activeTunnels[id.String()]; exists && process.Metrics != nil {
		return &PerformanceMetrics{
			AvgLatency:        process.Metrics.Latency,
			TotalBytes:        process.Metrics.BytesIn + process.Metrics.BytesOut,
			BytesPerSec:       float64(process.Metrics.BytesIn+process.Metrics.BytesOut) / time.Since(process.StartedAt).Seconds(),
			ConnectionsPerSec: float64(process.Metrics.ConnectionCount) / time.Since(process.StartedAt).Seconds(),
			ErrorRate:         float64(process.Metrics.ErrorCount) / float64(process.Metrics.ConnectionCount) * 100,
		}, nil
	}
	return nil, fmt.Errorf("tunnel not running or metrics not available")
}

// PerformanceMetrics represents tunnel performance data
type PerformanceMetrics struct {
	AvgLatency        float64 `json:"avg_latency"`
	TotalBytes        int64   `json:"total_bytes"`
	BytesPerSec       float64 `json:"bytes_per_sec"`
	ConnectionsPerSec float64 `json:"connections_per_sec"`
	ErrorRate         float64 `json:"error_rate"`
}

// Private methods

func (s *TunnelService) validateTunnelConfig(tunnel *models.Tunnel) error {
	if tunnel.Name == "" {
		return fmt.Errorf("tunnel name is required")
	}
	if tunnel.ServerIP == "" {
		return fmt.Errorf("server IP is required")
	}
	if tunnel.ServerPort <= 0 || tunnel.ServerPort > 65535 {
		return fmt.Errorf("invalid server port")
	}
	if tunnel.TargetIP == "" {
		return fmt.Errorf("target IP is required")
	}
	if tunnel.TargetPort <= 0 || tunnel.TargetPort > 65535 {
		return fmt.Errorf("invalid target port")
	}
	return nil
}

func (s *TunnelService) createTunnelProcess(tunnel *models.Tunnel) (*TunnelProcess, error) {
	// Build command based on protocol
	var cmd *exec.Cmd
	
	switch tunnel.Protocol {
	case models.ProtocolTCP:
		cmd = exec.Command("stunnel-core",
			"--mode", "server",
			"--protocol", "tcp",
			"--listen", fmt.Sprintf("%s:%d", tunnel.ServerIP, tunnel.ServerPort),
			"--target", fmt.Sprintf("%s:%d", tunnel.TargetIP, tunnel.TargetPort),
			"--token", tunnel.Token,
		)
	case models.ProtocolUDP:
		cmd = exec.Command("stunnel-core",
			"--mode", "server",
			"--protocol", "udp",
			"--listen", fmt.Sprintf("%s:%d", tunnel.ServerIP, tunnel.ServerPort),
			"--target", fmt.Sprintf("%s:%d", tunnel.TargetIP, tunnel.TargetPort),
			"--token", tunnel.Token,
		)
	case models.ProtocolWSS:
		cmd = exec.Command("stunnel-core",
			"--mode", "server",
			"--protocol", "wss",
			"--listen", fmt.Sprintf("%s:%d", tunnel.ServerIP, tunnel.ServerPort),
			"--target", fmt.Sprintf("%s:%d", tunnel.TargetIP, tunnel.TargetPort),
			"--token", tunnel.Token,
			"--cert", tunnel.TLSConfig.CertFile,
			"--key", tunnel.TLSConfig.KeyFile,
		)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", tunnel.Protocol)
	}

	return &TunnelProcess{
		ID:          tunnel.ID.String(),
		Tunnel:      tunnel,
		Process:     cmd,
		Status:      models.TunnelStatusConnecting,
		StartedAt:   time.Now(),
		LastPing:    time.Now(),
		StopChannel: make(chan bool),
		Metrics: &TunnelMetrics{
			LastUpdated: time.Now(),
		},
	}, nil
}

func (s *TunnelService) monitorTunnel(process *TunnelProcess) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-process.StopChannel:
			return
		case <-ticker.C:
			// Update metrics
			s.updateTunnelMetrics(process)
			
			// Check if process is still running
			if process.Process.ProcessState != nil && process.Process.ProcessState.Exited() {
				log.Printf("Tunnel process exited: %s", process.ID)
				s.handleTunnelExit(process)
				return
			}
			
			process.LastPing = time.Now()
		}
	}
}

func (s *TunnelService) updateTunnelMetrics(process *TunnelProcess) {
	// Simulate metrics collection (in real implementation, this would collect actual metrics)
	process.Metrics.ConnectionCount++
	process.Metrics.BytesIn += int64(1000 + (time.Now().UnixNano() % 5000))
	process.Metrics.BytesOut += int64(800 + (time.Now().UnixNano() % 3000))
	process.Metrics.Latency = float64(10 + (time.Now().UnixNano() % 50))
	process.Metrics.LastUpdated = time.Now()

	// Update database
	s.db.Model(process.Tunnel).Updates(map[string]interface{}{
		"bytes_in":         process.Metrics.BytesIn,
		"bytes_out":        process.Metrics.BytesOut,
		"connection_count": process.Metrics.ConnectionCount,
		"last_seen":        time.Now(),
	})
}

func (s *TunnelService) handleTunnelExit(process *TunnelProcess) {
	s.tunnelsMux.Lock()
	defer s.tunnelsMux.Unlock()

	// Update tunnel status
	s.db.Model(process.Tunnel).Update("status", models.TunnelStatusError)

	// Remove from active tunnels
	delete(s.activeTunnels, process.ID)

	log.Printf("Tunnel process exited and cleaned up: %s", process.ID)
}
