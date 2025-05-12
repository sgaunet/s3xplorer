package s3svc

import (
	"context"
	"fmt"
	"math"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// DefaultRetentionPolicyInDays is the default number of days that objects will be
// restored for if not specified in the config.
const DefaultRetentionPolicyInDays int32 = 2

// IsDownloadable returns true if the object is downloadable.
func (s *Service) IsDownloadable(ctx context.Context, key string) (bool, bool, error) {
	var isDownloadable, isRestoring bool
	hi := s3.HeadObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &key,
	}
	o, err := s.awsS3Client.HeadObject(ctx, &hi)
	if err != nil {
		isDownloadable = false
		isRestoring = false
		return isDownloadable, isRestoring, fmt.Errorf("IsDownloadable: error when called HeadObject: %w", err)
	}

	// fmt.Printf("%+v\n", o)
	// fmt.Printf("%+v\n", *o.Restore)
	// fmt.Printf("%+v\n", o.StorageClass)
	if o.StorageClass == "" || o.StorageClass == "STANDARD" {
		isDownloadable = true
		isRestoring = false
		return isDownloadable, isRestoring, nil
	}

	// If the object is in Glacier, we check if it is downloadable
	if o.Restore != nil {
		return s.checkRestoreStatus(*o.Restore)
	}
	return isDownloadable, isRestoring, nil
}

// checkRestoreStatus analyzes the Restore header to determine object availability.
func (s *Service) checkRestoreStatus(restoreHeader string) (bool, bool, error) {
	var isDownloadable, isRestoring bool
	res := conv(strings.ReplaceAll(restoreHeader, ", ", " "))
	
	// Check if restoration is in progress
	if vv, ok := res["ongoing-request"]; ok {
		switch vv {
		case "\"false\"":
			isRestoring = false
		case "\"true\"":
			isRestoring = true
			return isDownloadable, isRestoring, nil
		}
	}
	
	// Check expiry date if present
	if vv, ok := res["expiry-date"]; ok {
		const layout = "\"Mon 2 Jan 2006 15:04:06 MST\""
		tm, err2 := time.Parse(layout, vv)
		if err2 != nil {
			s.log.Error("checkRestoreStatus: error when parsing time", slog.String("error", err2.Error()))
			return false, isRestoring, nil
		}
		
		// If expiry date is in the future, object is downloadable
		if time.Now().After(tm) {
			isDownloadable = true
			return isDownloadable, isRestoring, nil
		}
	}
	
	return isDownloadable, isRestoring, nil
}

// RestoreObject restores an object.
func (s *Service) RestoreObject(ctx context.Context, key string) error {
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3@v1.26.0/types#RestoreRequest
	tt := types.GlacierJobParameters{
		Tier: "Standard",
	}
	
	// Use configured RestoreDays if set, otherwise use the default
	var restoreDays int32
	// Check if the RestoreDays is within int32 bounds to prevent overflow
	switch {
	case s.cfg.RestoreDays <= 0:
		restoreDays = DefaultRetentionPolicyInDays
		s.log.Debug("Using default restore days", slog.Int("days", int(DefaultRetentionPolicyInDays)))
	case s.cfg.RestoreDays > int(math.MaxInt32):
		// If RestoreDays exceeds int32 max value, use the maximum value
		restoreDays = math.MaxInt32
		s.log.Warn("RestoreDays exceeds maximum allowed value, capping at maximum", 
			slog.Int("requested", s.cfg.RestoreDays), 
			slog.Int("maximum", int(math.MaxInt32)))
	default:
		// This case should only be reached when RestoreDays is greater than 0 and less than MaxInt32,
		// but we'll still handle it safely to satisfy the linter and prevent any possible issues.
		
		// Create a safe function to convert int to int32 without risk of overflow
		safeInt32Conversion := func(value int) int32 {
			// Define the int32 range boundaries
			const maxInt32Value = 2147483647 // math.MaxInt32
			
			// Ensure the value is within int32 bounds
			if value > maxInt32Value {
				return math.MaxInt32
			}
			if value < 0 {
				return 0 // or DefaultRetentionPolicyInDays if you prefer
			}
			
			// Now we can safely convert
			return int32(value)
		}
		
		// Convert using our safe function
		restoreDays = safeInt32Conversion(s.cfg.RestoreDays)
		s.log.Debug("Using configured restore days", slog.Int("days", s.cfg.RestoreDays))
	}
	
	r := types.RestoreRequest{
		Days: aws.Int32(restoreDays),
		// Type:           "SELECT",
		GlacierJobParameters: &tt,
		// Tier: "Standard",
		// OutputLocation: &x,
		// Description:    &i,
	}
	p := s3.RestoreObjectInput{
		Bucket:         &s.cfg.Bucket,
		Key:            &key,
		RestoreRequest: &r,
	}
	o, err := s.awsS3Client.RestoreObject(ctx, &p)
	if err != nil {
		return fmt.Errorf("RestoreObject: error when called RestoreObject: %w", err)
	}
	s.log.Debug("RestoreObject", slog.String("key", key), slog.String("output", fmt.Sprintf("%+v", o)))
	return nil
}

// conv converts a string to a map.
func conv(str string) map[string]string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}
	// splitting string by space but considering quoted section
	items := strings.FieldsFunc(str, f)
	// create and fill the map
	m := make(map[string]string)
	for _, item := range items {
		x := strings.Split(item, "=")
		m[x[0]] = x[1]
	}
	// print the map
	// for k, v := range m {
	// 	fmt.Printf("%s: %s\n", k, v)
	// }
	return m
}
