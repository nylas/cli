// Package domain contains core business types.
package domain

import (
	"time"
)

// AuditStatus represents the status of a command execution.
type AuditStatus string

const (
	// AuditStatusSuccess indicates successful command execution.
	AuditStatusSuccess AuditStatus = "success"
	// AuditStatusError indicates command execution failed.
	AuditStatusError AuditStatus = "error"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	ID         string        `json:"id"`
	Timestamp  time.Time     `json:"timestamp"`
	Command    string        `json:"command"`            // e.g., "email list"
	Args       []string      `json:"args,omitempty"`     // Sanitized args
	GrantID    string        `json:"grant_id,omitempty"` // Grant used for command
	GrantEmail string        `json:"grant_email,omitempty"`
	Status     AuditStatus   `json:"status"`
	Duration   time.Duration `json:"duration"`
	Error      string        `json:"error,omitempty"`

	// Nylas API tracking
	RequestID  string `json:"request_id,omitempty"`  // Nylas request ID
	HTTPStatus int    `json:"http_status,omitempty"` // Response status code

	// Invocation tracking
	Invoker       string `json:"invoker,omitempty"`        // Username: "alice", "dependabot[bot]"
	InvokerSource string `json:"invoker_source,omitempty"` // Source: "claude-code", "github-actions", "terminal"
}

// AuditConfig contains all audit logging configuration.
type AuditConfig struct {
	// Core settings
	Enabled     bool `json:"enabled"`
	Initialized bool `json:"initialized"` // Has init been run?

	// Storage settings
	Path          string `json:"path"`           // Log directory
	RetentionDays int    `json:"retention_days"` // Days to keep logs
	MaxSizeMB     int    `json:"max_size_mb"`    // Max storage in MB
	Format        string `json:"format"`         // jsonl or json

	// Logging options
	LogAPIDetails bool `json:"log_api_details"` // Include endpoint/status
	LogRequestID  bool `json:"log_request_id"`  // Include Nylas request ID

	// Rotation settings
	RotateDaily bool `json:"rotate_daily"` // Create new file each day
	CompressOld bool `json:"compress_old"` // Gzip files older than 7 days
}

// DefaultAuditConfig returns the default audit configuration.
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		Enabled:       false,
		Initialized:   false,
		Path:          "", // Will be set to ~/.config/nylas/audit
		RetentionDays: 90,
		MaxSizeMB:     100,
		Format:        "jsonl",
		LogAPIDetails: true,
		LogRequestID:  true,
		RotateDaily:   true,
		CompressOld:   false,
	}
}

// AuditSummary contains aggregate statistics for audit logs.
type AuditSummary struct {
	// Time range
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Days      int       `json:"days"`

	// Totals
	TotalCommands  int     `json:"total_commands"`
	SuccessCount   int     `json:"success_count"`
	ErrorCount     int     `json:"error_count"`
	SuccessPercent float64 `json:"success_percent"`

	// Most used commands
	CommandCounts map[string]int `json:"command_counts"`

	// Account usage
	AccountCounts map[string]int `json:"account_counts"`

	// Invoker breakdown
	InvokerCounts map[string]int `json:"invoker_counts"`

	// API statistics
	TotalAPICalls   int           `json:"total_api_calls"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	APIErrorRate    float64       `json:"api_error_rate"`
}

// AuditQueryOptions defines filters for querying audit logs.
type AuditQueryOptions struct {
	Limit         int       `json:"limit,omitempty"`
	Since         time.Time `json:"since,omitempty"`
	Until         time.Time `json:"until,omitempty"`
	Command       string    `json:"command,omitempty"`
	Status        string    `json:"status,omitempty"`
	GrantID       string    `json:"grant_id,omitempty"`
	RequestID     string    `json:"request_id,omitempty"`
	Invoker       string    `json:"invoker,omitempty"`        // Filter by username
	InvokerSource string    `json:"invoker_source,omitempty"` // Filter by source platform
}
