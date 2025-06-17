package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"utunnel-pro/internal/models"
	"utunnel-pro/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

// MonitoringService handles real-time monitoring and alerting
type MonitoringService struct {
	db          *gorm.DB
	redis       *redis.Client
	config      *config.Config
	clients     map[string]*websocket.Conn
	clientsMux  sync.RWMutex
	tunnelStats map[string]*TunnelStats
	statsMux    sync.RWMutex
	
	// Prometheus metrics
	tunnelConnections prometheus.Gauge
	tunnelBandwidth   prometheus.CounterVec
	tunnelLatency     prometheus.HistogramVec
	tunnelErrors      prometheus.CounterVec
	tunnelUptime      prometheus.GaugeVec
}

// TunnelStats represents real-time tunnel statistics
type TunnelStats struct {
	TunnelID        string    `json:"tunnel_id"`
	Status          string    `json:"status"`
	IsOnline        bool      `json:"is_online"`
	LastPing        time.Time `json:"last_ping"`
	ConnectionCount int       `json:"connection_count"`
	BytesIn         int64     `json:"bytes_in"`
	BytesOut        int64     `json:"bytes_out"`
	Latency         float64   `json:"latency"`
	CPUUsage        float64   `json:"cpu_usage"`
	MemoryUsage     int64     `json:"memory_usage"`
	ErrorCount      int       `json:"error_count"`
	Timestamp       time.Time `json:"timestamp"`
}

// AlertRule represents monitoring alert rules
type AlertRule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TunnelID    string    `json:"tunnel_id"`
	Metric      string    `json:"metric"`      // latency, cpu_usage, error_rate, etc.
	Operator    string    `json:"operator"`    // >, <, >=, <=, ==
	Threshold   float64   `json:"threshold"`
	Duration    int       `json:"duration"`    // seconds
	Enabled     bool      `json:"enabled"`
	LastTriggered *time.Time `json:"last_triggered"`
}

// Alert represents a triggered alert
type Alert struct {
	ID          string    `json:"id"`
	RuleID      string    `json:"rule_id"`
	TunnelID    string    `json:"tunnel_id"`
	TunnelName  string    `json:"tunnel_name"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"`    // info, warning, critical
	Status      string    `json:"status"`      // active, resolved
	TriggeredAt time.Time `json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(db *gorm.DB, redis *redis.Client, config *config.Config) *MonitoringService {
	// Initialize Prometheus metrics
	tunnelConnections := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "utunnel_active_connections_total",
		Help: "Total number of active tunnel connections",
	})

	tunnelBandwidth := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "utunnel_bandwidth_bytes_total",
		Help: "Total bandwidth usage in bytes",
	}, []string{"tunnel_id", "direction"})

	tunnelLatency := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "utunnel_latency_seconds",
		Help:    "Tunnel latency in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"tunnel_id"})

	tunnelErrors := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "utunnel_errors_total",
		Help: "Total number of tunnel errors",
	}, []string{"tunnel_id", "error_type"})

	tunnelUptime := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "utunnel_uptime_seconds",
		Help: "Tunnel uptime in seconds",
	}, []string{"tunnel_id"})

	return &MonitoringService{
		db:                db,
		redis:             redis,
		config:            config,
		clients:           make(map[string]*websocket.Conn),
		tunnelStats:       make(map[string]*TunnelStats),
		tunnelConnections: tunnelConnections,
		tunnelBandwidth:   tunnelBandwidth,
		tunnelLatency:     tunnelLatency,
		tunnelErrors:      tunnelErrors,
		tunnelUptime:      tunnelUptime,
	}
}

// Start begins the monitoring service
func (m *MonitoringService) Start(ctx context.Context) error {
	log.Println("Starting monitoring service...")

	// Start metrics collection
	go m.collectMetrics(ctx)
	
	// Start alert processing
	go m.processAlerts(ctx)
	
	// Start cleanup routine
	go m.cleanup(ctx)

	return nil
}

// RegisterWebSocketClient registers a new WebSocket client for real-time updates
func (m *MonitoringService) RegisterWebSocketClient(clientID string, conn *websocket.Conn) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	
	m.clients[clientID] = conn
	log.Printf("WebSocket client registered: %s", clientID)
}

// UnregisterWebSocketClient removes a WebSocket client
func (m *MonitoringService) UnregisterWebSocketClient(clientID string) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	
	if conn, exists := m.clients[clientID]; exists {
		conn.Close()
		delete(m.clients, clientID)
		log.Printf("WebSocket client unregistered: %s", clientID)
	}
}

// UpdateTunnelStats updates tunnel statistics
func (m *MonitoringService) UpdateTunnelStats(stats *TunnelStats) {
	m.statsMux.Lock()
	defer m.statsMux.Unlock()
	
	stats.Timestamp = time.Now()
	m.tunnelStats[stats.TunnelID] = stats
	
	// Update Prometheus metrics
	m.tunnelConnections.Set(float64(stats.ConnectionCount))
	m.tunnelBandwidth.WithLabelValues(stats.TunnelID, "in").Add(float64(stats.BytesIn))
	m.tunnelBandwidth.WithLabelValues(stats.TunnelID, "out").Add(float64(stats.BytesOut))
	m.tunnelLatency.WithLabelValues(stats.TunnelID).Observe(stats.Latency / 1000) // Convert to seconds
	
	if stats.IsOnline {
		m.tunnelUptime.WithLabelValues(stats.TunnelID).SetToCurrentTime()
	}
	
	// Store in Redis for real-time access
	statsJSON, _ := json.Marshal(stats)
	m.redis.Set(context.Background(), fmt.Sprintf("tunnel:stats:%s", stats.TunnelID), statsJSON, 5*time.Minute)
	
	// Broadcast to WebSocket clients
	m.broadcastStats(stats)
	
	// Store in database for historical analysis
	go m.storeMetricInDB(stats)
}

// GetTunnelStats retrieves current tunnel statistics
func (m *MonitoringService) GetTunnelStats(tunnelID string) (*TunnelStats, error) {
	m.statsMux.RLock()
	defer m.statsMux.RUnlock()
	
	if stats, exists := m.tunnelStats[tunnelID]; exists {
		return stats, nil
	}
	
	// Try to get from Redis
	statsJSON, err := m.redis.Get(context.Background(), fmt.Sprintf("tunnel:stats:%s", tunnelID)).Result()
	if err != nil {
		return nil, err
	}
	
	var stats TunnelStats
	if err := json.Unmarshal([]byte(statsJSON), &stats); err != nil {
		return nil, err
	}
	
	return &stats, nil
}

// GetHistoricalMetrics retrieves historical metrics for a tunnel
func (m *MonitoringService) GetHistoricalMetrics(tunnelID string, from, to time.Time) ([]models.TunnelMetric, error) {
	var metrics []models.TunnelMetric
	
	err := m.db.Where("tunnel_id = ? AND timestamp BETWEEN ? AND ?", tunnelID, from, to).
		Order("timestamp ASC").
		Find(&metrics).Error
	
	return metrics, err
}

// CreateAlertRule creates a new alert rule
func (m *MonitoringService) CreateAlertRule(rule *AlertRule) error {
	ruleJSON, _ := json.Marshal(rule)
	return m.redis.Set(context.Background(), fmt.Sprintf("alert:rule:%s", rule.ID), ruleJSON, 0).Err()
}

// GetAlertRules retrieves all alert rules for a tunnel
func (m *MonitoringService) GetAlertRules(tunnelID string) ([]*AlertRule, error) {
	keys, err := m.redis.Keys(context.Background(), "alert:rule:*").Result()
	if err != nil {
		return nil, err
	}
	
	var rules []*AlertRule
	for _, key := range keys {
		ruleJSON, err := m.redis.Get(context.Background(), key).Result()
		if err != nil {
			continue
		}
		
		var rule AlertRule
		if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
			continue
		}
		
		if rule.TunnelID == tunnelID || tunnelID == "" {
			rules = append(rules, &rule)
		}
	}
	
	return rules, nil
}

// Private methods

func (m *MonitoringService) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collectSystemMetrics()
		}
	}
}

func (m *MonitoringService) collectSystemMetrics() {
	// Collect system-wide metrics
	var tunnels []models.Tunnel
	m.db.Where("status = ?", models.TunnelStatusActive).Find(&tunnels)
	
	for _, tunnel := range tunnels {
		// Simulate metric collection (in real implementation, this would collect actual metrics)
		stats := &TunnelStats{
			TunnelID:        tunnel.ID.String(),
			Status:          string(tunnel.Status),
			IsOnline:        tunnel.Status == models.TunnelStatusActive,
			LastPing:        time.Now(),
			ConnectionCount: tunnel.ConnectionCount,
			BytesIn:         tunnel.BytesIn,
			BytesOut:        tunnel.BytesOut,
			Latency:         float64(time.Now().UnixNano()%100) + 10, // Simulated latency
			CPUUsage:        float64(time.Now().UnixNano()%50) + 10,  // Simulated CPU usage
			MemoryUsage:     int64(time.Now().UnixNano()%1000000) + 1000000, // Simulated memory usage
			ErrorCount:      0,
		}
		
		m.UpdateTunnelStats(stats)
	}
}

func (m *MonitoringService) processAlerts(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkAlertRules()
		}
	}
}

func (m *MonitoringService) checkAlertRules() {
	rules, err := m.GetAlertRules("")
	if err != nil {
		log.Printf("Error getting alert rules: %v", err)
		return
	}
	
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		
		stats, err := m.GetTunnelStats(rule.TunnelID)
		if err != nil {
			continue
		}
		
		if m.evaluateRule(rule, stats) {
			m.triggerAlert(rule, stats)
		}
	}
}

func (m *MonitoringService) evaluateRule(rule *AlertRule, stats *TunnelStats) bool {
	var value float64
	
	switch rule.Metric {
	case "latency":
		value = stats.Latency
	case "cpu_usage":
		value = stats.CPUUsage
	case "memory_usage":
		value = float64(stats.MemoryUsage)
	case "error_count":
		value = float64(stats.ErrorCount)
	default:
		return false
	}
	
	switch rule.Operator {
	case ">":
		return value > rule.Threshold
	case "<":
		return value < rule.Threshold
	case ">=":
		return value >= rule.Threshold
	case "<=":
		return value <= rule.Threshold
	case "==":
		return value == rule.Threshold
	default:
		return false
	}
}

func (m *MonitoringService) triggerAlert(rule *AlertRule, stats *TunnelStats) {
	// Check if alert was recently triggered to avoid spam
	if rule.LastTriggered != nil && time.Since(*rule.LastTriggered) < time.Duration(rule.Duration)*time.Second {
		return
	}
	
	alert := &Alert{
		ID:          fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		RuleID:      rule.ID,
		TunnelID:    rule.TunnelID,
		Message:     fmt.Sprintf("Alert: %s %s %f", rule.Metric, rule.Operator, rule.Threshold),
		Severity:    m.getSeverity(rule.Metric),
		Status:      "active",
		TriggeredAt: time.Now(),
		Metadata: map[string]interface{}{
			"current_value": m.getCurrentValue(rule.Metric, stats),
			"threshold":     rule.Threshold,
		},
	}
	
	// Store alert
	alertJSON, _ := json.Marshal(alert)
	m.redis.Set(context.Background(), fmt.Sprintf("alert:active:%s", alert.ID), alertJSON, 24*time.Hour)
	
	// Update rule last triggered time
	now := time.Now()
	rule.LastTriggered = &now
	m.CreateAlertRule(rule)
	
	// Send notifications (Telegram, email, etc.)
	go m.sendNotification(alert)
	
	log.Printf("Alert triggered: %s", alert.Message)
}

func (m *MonitoringService) broadcastStats(stats *TunnelStats) {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()
	
	message, _ := json.Marshal(map[string]interface{}{
		"type": "tunnel_stats",
		"data": stats,
	})
	
	for clientID, conn := range m.clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error broadcasting to client %s: %v", clientID, err)
			conn.Close()
			delete(m.clients, clientID)
		}
	}
}

func (m *MonitoringService) storeMetricInDB(stats *TunnelStats) {
	tunnelID, _ := parseUUID(stats.TunnelID)
	
	metric := &models.TunnelMetric{
		TunnelID:        tunnelID,
		Timestamp:       stats.Timestamp,
		BytesIn:         stats.BytesIn,
		BytesOut:        stats.BytesOut,
		ConnectionCount: stats.ConnectionCount,
		Latency:         stats.Latency,
		CPUUsage:        stats.CPUUsage,
		MemoryUsage:     stats.MemoryUsage,
		ErrorCount:      stats.ErrorCount,
	}
	
	m.db.Create(metric)
}

func (m *MonitoringService) cleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Clean up old metrics (keep last 30 days)
			cutoff := time.Now().AddDate(0, 0, -30)
			m.db.Where("timestamp < ?", cutoff).Delete(&models.TunnelMetric{})
			
			// Clean up resolved alerts (keep last 7 days)
			alertCutoff := time.Now().AddDate(0, 0, -7)
			keys, _ := m.redis.Keys(context.Background(), "alert:active:*").Result()
			for _, key := range keys {
				alertJSON, _ := m.redis.Get(context.Background(), key).Result()
				var alert Alert
				if json.Unmarshal([]byte(alertJSON), &alert) == nil {
					if alert.ResolvedAt != nil && alert.ResolvedAt.Before(alertCutoff) {
						m.redis.Del(context.Background(), key)
					}
				}
			}
		}
	}
}

func (m *MonitoringService) getSeverity(metric string) string {
	switch metric {
	case "error_count":
		return "critical"
	case "cpu_usage", "memory_usage":
		return "warning"
	default:
		return "info"
	}
}

func (m *MonitoringService) getCurrentValue(metric string, stats *TunnelStats) interface{} {
	switch metric {
	case "latency":
		return stats.Latency
	case "cpu_usage":
		return stats.CPUUsage
	case "memory_usage":
		return stats.MemoryUsage
	case "error_count":
		return stats.ErrorCount
	default:
		return nil
	}
}

func (m *MonitoringService) sendNotification(alert *Alert) {
	// Send Telegram notification if configured
	if m.config.Telegram.Enabled {
		go m.sendTelegramNotification(alert)
	}

	// Send email notification if configured
	go m.sendEmailNotification(alert)

	log.Printf("Sending notification for alert: %s", alert.Message)
}

func (m *MonitoringService) sendTelegramNotification(alert *Alert) {
	if m.config.Telegram.BotToken == "" || m.config.Telegram.ChatID == "" {
		return
	}

	message := fmt.Sprintf(`🚨 *UTunnel Pro Alert*

*Alert:* %s
*Tunnel:* %s
*Severity:* %s
*Time:* %s

*Details:*
%s`, alert.Message, alert.TunnelName, alert.Severity, alert.TriggeredAt.Format("2006-01-02 15:04:05"), alert.Metadata)

	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", m.config.Telegram.BotToken)

	payload := map[string]interface{}{
		"chat_id":    m.config.Telegram.ChatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(telegramURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Failed to send Telegram notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Telegram API returned status: %d", resp.StatusCode)
	}
}

func (m *MonitoringService) sendEmailNotification(alert *Alert) {
	// Email notification implementation would go here
	// This would use SMTP to send email alerts
	log.Printf("Email notification for alert: %s", alert.Message)
}
