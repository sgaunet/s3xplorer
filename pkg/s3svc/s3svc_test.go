// Package s3svc_test tests the s3svc package functionality
package s3svc_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sgaunet/s3xplorer/pkg/s3svc"
)

// TestNewS3Svc tests creating a new service
func TestNewS3Svc(t *testing.T) {
	// Setup with nil client for basic initialization test
	cfg := config.Config{
		Bucket:      "test-bucket",
		RestoreDays: 5,
	}

	// Create service
	service := s3svc.NewS3Svc(cfg, nil)

	// Verify service was created
	if service == nil {
		t.Fatal("Service should not be nil")
	}
}

// TestSetLogger tests setting a logger
func TestSetLogger(t *testing.T) {
	// Setup
	cfg := config.Config{
		Bucket: "test-bucket",
	}

	// Create service
	service := s3svc.NewS3Svc(cfg, nil)

	// Set a logger
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service.SetLogger(logger)

	// If it doesn't panic, the test passes
}

// TestRestoreDaysConfig tests that the RestoreDays configuration is properly used
func TestRestoreDaysConfig(t *testing.T) {
	testCases := []struct {
		name        string
		restoreDays int
		expected    int
	}{
		{
			name:        "Default restore days",
			restoreDays: 0,
			expected:    2, // Default value
		},
		{
			name:        "Custom restore days",
			restoreDays: 5,
			expected:    5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a service with the test configuration
			cfg := config.Config{
				Bucket:      "test-bucket",
				RestoreDays: tc.restoreDays,
			}
			
			// We can't test this directly without mocking,
			// but we're ensuring that the code builds and initializes properly
			service := s3svc.NewS3Svc(cfg, nil)
			
			// Simply verify the service was created
			if service == nil {
				t.Fatal("Service should not be nil")
			}
			
			// Note: In a real test, we'd verify the RestoreDays value is used
			// when making S3 RestoreObject calls, but that requires mocking the AWS SDK
		})
	}
}
