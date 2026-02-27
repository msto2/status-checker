package models

import "time"

// Service represents a service to monitor
type Service struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	IPAddress   string    `json:"ip_address"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StatusHistory represents a historical status check
type StatusHistory struct {
	ID             int       `json:"id"`
	ServiceID      int       `json:"service_id"`
	Status         string    `json:"status"` // "up" or "down"
	ResponseTimeMs int       `json:"response_time_ms"`
	CheckedAt      time.Time `json:"checked_at"`
}

// AlertRecipient represents an email recipient for alerts
type AlertRecipient struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// AlertLog represents a log entry for sent alerts
type AlertLog struct {
	ID             int       `json:"id"`
	ServiceID      int       `json:"service_id"`
	RecipientEmail string    `json:"recipient_email"`
	Status         string    `json:"status"` // "sent" or "failed"
	ErrorMessage   string    `json:"error_message,omitempty"`
	SentAt         time.Time `json:"sent_at"`
}

// Config represents a configuration key-value pair
type Config struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ServiceStatus represents the current status of a service for frontend display
type ServiceStatus struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	IPAddress      string `json:"ip_address"`
	Status         string `json:"status"` // "up", "down", or "unknown"
	ResponseTimeMs int    `json:"response_time_ms,omitempty"`
}

// FrontendPayload represents the data sent to the frontend
type FrontendPayload struct {
	Services       []ServiceStatus `json:"services"`
	LastUpdateTime time.Time       `json:"last_update_time"`
}

// AppConfig represents the application configuration
type AppConfig struct {
	AdminPort            int    `json:"admin_port"`
	AdminBind            string `json:"admin_bind"`
	DatabasePath         string `json:"database_path"`
	LogFile              string `json:"log_file"`
	LogLevel             string `json:"log_level"`
	FrontendURL          string `json:"frontend_url"`
	APIKey               string `json:"api_key"`
	CheckIntervalSeconds int    `json:"check_interval_seconds"`
	PingTimeoutSeconds   int    `json:"ping_timeout_seconds"`
	PingCount            int    `json:"ping_count"`
	MaxConcurrentPings   int    `json:"max_concurrent_pings"`
	PushTimeoutSeconds   int    `json:"push_timeout_seconds"`
	PushRetryAttempts    int    `json:"push_retry_attempts"`
	AlertCooldownMinutes int    `json:"alert_cooldown_minutes"`
}
