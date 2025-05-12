package app

import (
	"log/slog"
	"net/http"

	"github.com/sgaunet/s3xplorer/pkg/views"
)

// BucketListingHandler handles the request to list available buckets.
func (s *App) BucketListingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Check if bucket changes are allowed
	if s.cfg.BucketLocked {
		// If bucket is locked (specified in config), don't allow bucket selection
		s.log.Warn("Attempted to access bucket selection when bucket is locked in config", 
			slog.String("current", s.cfg.Bucket))
		
		// Render an error page explaining that bucket is locked
		s.renderErrorPage(ctx, w, "Bucket changes are not permitted when a bucket is explicitly defined in configuration. Please update your configuration file if you need to access a different bucket.")
		return
	}
	
	// Get list of available buckets
	buckets, err := s.s3svc.ListBuckets(ctx)
	if err != nil {
		s.log.Error("Error listing buckets", slog.String("error", err.Error()))
		s.renderErrorPage(ctx, w, "Failed to retrieve bucket list: "+err.Error())
		return
	}
	
	// Generate the bucket selection template
	template := views.BucketSelection(buckets, s.s3svc.GetBucketName(), s.cfg)
	
	// Render the bucket selection page
	err = template.Render(ctx, w)
	if err != nil {
		s.log.Error("Error rendering bucket selection page", slog.String("error", err.Error()))
		http.Error(w, "Failed to render bucket selection page", http.StatusInternalServerError)
		return
	}
}
