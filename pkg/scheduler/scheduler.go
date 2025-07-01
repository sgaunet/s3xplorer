package scheduler

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/robfig/cron/v3"
	"github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sgaunet/s3xplorer/pkg/scanner"
)

// Scheduler manages background jobs for S3 scanning
type Scheduler struct {
	cron    *cron.Cron
	scanner *scanner.Service
	cfg     config.Config
	log     *slog.Logger
	db      *sql.DB
}

// NewScheduler creates a new scheduler instance
func NewScheduler(cfg config.Config, db *sql.DB, scannerSvc *scanner.Service) *Scheduler {
	c := cron.New()
	return &Scheduler{
		cron:    c,
		scanner: scannerSvc,
		cfg:     cfg,
		log:     slog.New(slog.DiscardHandler),
		db:      db,
	}
}

// SetLogger sets the logger for the scheduler
func (s *Scheduler) SetLogger(log *slog.Logger) {
	s.log = log
}

// Start starts the scheduler and adds the scan job
func (s *Scheduler) Start(ctx context.Context) error {
	if !s.cfg.EnableBackgroundScan {
		s.log.Info("Background scanning is disabled")
		return nil
	}

	// Add the scanning job
	_, err := s.cron.AddFunc(s.cfg.ScanCronSchedule, func() {
		s.log.Info("Starting scheduled S3 scan")
		if err := s.scanner.ScanBucket(ctx, s.cfg.Bucket); err != nil {
			s.log.Error("Scheduled scan failed", slog.String("error", err.Error()))
		} else {
			s.log.Info("Scheduled scan completed successfully")
		}
	})
	if err != nil {
		return err
	}

	s.log.Info("Starting scheduler", slog.String("schedule", s.cfg.ScanCronSchedule))
	s.cron.Start()
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.log.Info("Stopping scheduler")
	s.cron.Stop()
}
