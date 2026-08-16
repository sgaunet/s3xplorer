package views

import (
	"context"
	"fmt"
	"html"
	"io"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/a-h/templ"
)

const (
	hoursPerDay   = 24
	hoursPerWeek  = hoursPerDay * 7
	hoursPerMonth = hoursPerDay * 30
	hoursPerYear  = hoursPerDay * 365

	// etagDisplayLength is the maximum length for displaying ETags.
	etagDisplayLength = 40

	// etagCellLength is the truncation length for ETags shown in table cells.
	etagCellLength = 8

	// sizeBase is the divisor between successive human-readable size units.
	sizeBase = 1024
)

// sizeUnits are the human-readable size units, in ascending order.
var sizeUnits = []string{"Bytes", "KB", "MB", "GB", "TB", "PB"}

// formatSize converts a size in bytes to a human-readable string.
func formatSize(sizeInBytes int64) string {
	if sizeInBytes <= 0 {
		return "0 Bytes"
	}

	// Pick the unit by dividing by sizeBase repeatedly, capped at the largest unit.
	i := math.Floor(math.Log(float64(sizeInBytes)) / math.Log(float64(sizeBase)))
	if i >= float64(len(sizeUnits)) {
		i = float64(len(sizeUnits) - 1)
	}

	size := float64(sizeInBytes) / math.Pow(float64(sizeBase), i)

	// Whole numbers for bytes, one decimal place for everything larger.
	if i == 0 {
		return fmt.Sprintf("%d %s", int64(size), sizeUnits[int(i)])
	}
	return fmt.Sprintf("%.1f %s", size, sizeUnits[int(i)])
}

// formatRelativeTime converts a time.Time to a human-readable relative time string.
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	// Future dates
	if duration < 0 {
		return "in the future"
	}

	// Less than a minute
	if duration < time.Minute {
		return "just now"
	}

	// Minutes
	if duration < time.Hour {
		return formatMinutes(duration)
	}

	// Hours
	if duration < hoursPerDay*time.Hour {
		return formatHours(duration)
	}

	// Days
	if duration < hoursPerWeek*time.Hour {
		return formatDays(duration)
	}

	// Weeks
	if duration < hoursPerMonth*time.Hour {
		return formatWeeks(duration)
	}

	// Months
	if duration < hoursPerYear*time.Hour {
		return formatMonths(duration)
	}

	// Years
	return formatYears(duration)
}

func formatMinutes(d time.Duration) string {
	minutes := int(d.Minutes())
	if minutes == 1 {
		return "1 minute ago"
	}
	return fmt.Sprintf("%d minutes ago", minutes)
}

func formatHours(d time.Duration) string {
	hours := int(d.Hours())
	if hours == 1 {
		return "1 hour ago"
	}
	return fmt.Sprintf("%d hours ago", hours)
}

func formatDays(d time.Duration) string {
	days := int(d.Hours() / hoursPerDay)
	if days == 1 {
		return "yesterday"
	}
	return fmt.Sprintf("%d days ago", days)
}

func formatWeeks(d time.Duration) string {
	weeks := int(d.Hours() / hoursPerWeek)
	if weeks == 1 {
		return "1 week ago"
	}
	return fmt.Sprintf("%d weeks ago", weeks)
}

func formatMonths(d time.Duration) string {
	months := int(d.Hours() / hoursPerMonth)
	if months == 1 {
		return "1 month ago"
	}
	return fmt.Sprintf("%d months ago", months)
}

func formatYears(d time.Duration) string {
	years := int(d.Hours() / hoursPerYear)
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// formatDateTime formats a time.Time to a readable date and time string.
func formatDateTime(t time.Time) string {
	return t.Format("Jan 2, 2006 15:04")
}

// truncateETag truncates an ETag to the first N characters for display.
func truncateETag(etag string, length int) string {
	// Remove quotes if present
	etag = strings.Trim(etag, "\"")

	if len(etag) <= length {
		return etag
	}
	return etag[:length] + "..."
}

// getFileTypeLabel returns a human-readable label for file type.
func getFileTypeLabel(filename string) string {
	ext := strings.ToLower(filename)
	lastDot := strings.LastIndex(ext, ".")
	if lastDot == -1 {
		return "File"
	}
	ext = ext[lastDot+1:]
	return strings.ToUpper(ext)
}

// getFileIconName returns an appropriate Lucide icon name for a file based on its extension.
func getFileIconName(filename string) string {
	ext := strings.ToLower(filename)
	lastDot := strings.LastIndex(ext, ".")
	if lastDot == -1 {
		return "file" // Generic file
	}
	ext = ext[lastDot+1:]

	// Image files
	if slices.Contains([]string{"jpg", "jpeg", "png", "gif", "bmp", "svg", "webp", "ico"}, ext) {
		return "file-image"
	}

	// Spreadsheets
	if slices.Contains([]string{"xls", "xlsx", "ods", "csv"}, ext) {
		return "file-spreadsheet"
	}

	// Archives
	if slices.Contains([]string{"zip", "rar", "7z", "tar", "gz", "bz2"}, ext) {
		return "file-archive"
	}

	// Text/documents
	if slices.Contains([]string{"txt", "md", "doc", "docx", "pdf", "rtf"}, ext) {
		return "file-text"
	}

	// Default
	return "file"
}

// Icon renders an SVG sprite icon inside a Bulma icon wrapper.
//
// class carries Bulma modifiers, any combination of:
//   - size:  "" (default, 1.5rem), "is-small", "is-medium", "is-large"
//   - color: "has-text-link", "has-text-weak", "has-text-danger", "has-text-success", ...
//   - state: "is-spinning" (defined in theme.css)
//
// The sprite uses stroke="currentColor", so colour is inherited from the wrapper.
func Icon(name string, class string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w,
			`<span class="icon %s"><svg aria-hidden="true"><use href="/static/icons.svg#%s"></use></svg></span>`,
			class, name)
		return err
	})
}

// Bucket accessibility states understood by StatusBadge.
const (
	statusAccessible   = "accessible"
	statusInaccessible = "inaccessible"
)

// StatusBadge renders a bucket accessibility status as a Bulma tag.
// Both variants are theme-aware, so no dark-mode handling is needed.
func StatusBadge(status string, message string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		switch status {
		case statusAccessible:
			return writeStatusTag(w, "is-success", "check-circle", "Accessible")
		case statusInaccessible:
			if err := writeStatusTag(w, "is-danger", "x-circle", "Inaccessible"); err != nil {
				return err
			}
			if message == "" {
				return nil
			}
			// The message comes from the S3 API, so it must be escaped.
			_, err := fmt.Fprintf(w,
				`<p class="help has-text-weak mt-1" title="%s">%s</p>`,
				html.EscapeString(message),
				html.EscapeString(truncateETag(message, etagDisplayLength)))
			return err
		default:
			return nil
		}
	})
}

// writeStatusTag writes a single Bulma tag with a leading sprite icon.
func writeStatusTag(w io.Writer, tone string, icon string, label string) error {
	_, err := fmt.Fprintf(w,
		`<span class="tag %s is-light" role="status">`+
			`<span class="icon is-small"><svg aria-hidden="true">`+
			`<use href="/static/icons.svg#%s"></use></svg></span>`+
			`<span class="ml-1">%s</span></span>`,
		tone, icon, label)
	return err
}

// SkipToContent renders a skip link for keyboard users.
// It is visually hidden until focused; theme.css handles the reveal.
func SkipToContent() templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := fmt.Fprint(w,
			`<a href="#main-content" class="skip-link is-sr-only">Skip to main content</a>`)
		return err
	})
}

/*
─────────────────────────────────────────────────────────────────────────────
BULMA COMPONENT PATTERNS
─────────────────────────────────────────────────────────────────────────────

The UI is styled with Bulma 1.x (https://bulma.io/documentation/). Bulma is a
classes-only framework: there is no build step and no purge pass, so any class
in the Bulma docs is available immediately. See BULMA_VERSION for the pinned
release, and pkg/views/static/theme.css for the handful of rules Bulma lacks.

Dark mode is native: `data-theme="dark"` on <html> (set before first paint by
the inline script in layout.templ). Never write theme-conditional classes —
Bulma's semantic colours already adapt.

Layout      @Layout(PageOpts{...}, cfg) — the shared page shell.
            Containers: is-max-widescreen (tables), is-max-desktop, is-max-tablet.
Cards       .box
Tables      .table-container > .table.is-fullwidth.is-hoverable.is-striped
Buttons     .button.is-link (primary), .is-danger, .is-ghost (icon-only).
            Use the native `disabled` attribute — Bulma styles it.
Badges      .tag, .tag.is-rounded, .tag.is-success.is-light
Alerts      .notification.is-danger.is-light, .is-info.is-light
Inputs      .field > .control.has-icons-left > .input + .icon.is-left
Helpers     is-hidden, is-sr-only, is-flex, is-align-items-center, is-gap-N,
            is-hidden-mobile / is-hidden-tablet, and margin/padding helpers
            (mt-4, mb-5, px-4, ...) on a 0–6 spacing scale.
*/
