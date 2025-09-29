// Package health provides database health monitoring and status tracking functionality.
package health

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"
)

// Status represents the current health status.
type Status string

const (
	// StatusHealthy indicates the service is functioning normally.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indicates the service is experiencing issues.
	StatusUnhealthy Status = "unhealthy"
	// StatusUnknown indicates the health status hasn't been determined yet.
	StatusUnknown Status = "unknown"
)

// DatabaseHealth tracks database connectivity health.
type DatabaseHealth struct {
	mu                 sync.RWMutex
	db                 *sql.DB
	status             Status
	lastCheck          time.Time
	lastError          error
	consecutiveFailures int
	logger             *slog.Logger
	checkInterval      time.Duration
	maxRetries         int
	cancel             context.CancelFunc
}

// Info contains current health information.
type Info struct {
	Status              Status    `json:"status"`
	LastCheck           time.Time `json:"last_check"`
	LastError           string    `json:"last_error,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	IsConnected         bool      `json:"is_connected"`
}

// NewDatabaseHealth creates a new database health monitor.
func NewDatabaseHealth(db *sql.DB, logger *slog.Logger) *DatabaseHealth {
	const (
		defaultCheckInterval = 30 * time.Second
		defaultMaxRetries    = 3
	)
	return &DatabaseHealth{
		db:            db,
		status:        StatusUnknown,
		logger:        logger,
		checkInterval: defaultCheckInterval,
		maxRetries:    defaultMaxRetries,
	}
}

// Start begins health monitoring in the background.
func (h *DatabaseHealth) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	h.cancel = cancel

	// Perform initial health check
	h.checkHealth(ctx)

	// Start periodic health checks
	go h.healthCheckLoop(ctx)
}

// Stop stops the health monitoring.
func (h *DatabaseHealth) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
}

// GetHealthInfo returns current health information.
func (h *DatabaseHealth) GetHealthInfo() Info {
	h.mu.RLock()
	defer h.mu.RUnlock()

	errorMsg := ""
	if h.lastError != nil {
		errorMsg = h.lastError.Error()
	}

	return Info{
		Status:              h.status,
		LastCheck:           h.lastCheck,
		LastError:           errorMsg,
		ConsecutiveFailures: h.consecutiveFailures,
		IsConnected:         h.status == StatusHealthy,
	}
}

// IsHealthy returns true if the database is currently healthy.
func (h *DatabaseHealth) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status == StatusHealthy
}

// UpdateDatabase updates the database connection being monitored.
func (h *DatabaseHealth) UpdateDatabase(db *sql.DB) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.db = db
	if db != nil {
		h.logger.Info("Database connection updated - performing immediate health check")
		go func() {
			// Perform immediate health check in background
			const healthCheckDelay = 100 * time.Millisecond
			time.Sleep(healthCheckDelay)
			h.checkHealth(context.Background())
		}()
	}
}

// healthCheckLoop runs periodic health checks.
func (h *DatabaseHealth) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkHealth(ctx)
		}
	}
}

// checkHealth performs a health check against the database.
func (h *DatabaseHealth) checkHealth(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastCheck = time.Now()

	if h.db == nil {
		h.status = StatusUnhealthy
		h.lastError = nil
		h.consecutiveFailures++
		return
	}

	// Ping with timeout
	const pingTimeout = 5 * time.Second
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	err := h.db.PingContext(pingCtx)
	if err != nil {
		h.status = StatusUnhealthy
		h.lastError = err
		h.consecutiveFailures++

		h.logger.Debug("Database health check failed",
			slog.String("error", err.Error()),
			slog.Int("consecutive_failures", h.consecutiveFailures))
	} else {
		wasUnhealthy := h.status == StatusUnhealthy
		h.status = StatusHealthy
		h.lastError = nil
		h.consecutiveFailures = 0

		if wasUnhealthy {
			h.logger.Info("Database health restored")
		}
	}
}