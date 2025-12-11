package views

import (
	"context"
	"fmt"
	"io"
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
)

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

// Icon renders an SVG icon from the sprite sheet.
// Size utilities (Tailwind):
//   - icon-xs → w-3 h-3 (12px)
//   - icon-sm → w-4 h-4 (16px)
//   - icon (default) → w-5 h-5 (20px)
//   - icon-lg → w-6 h-6 (24px)
//   - icon-xl → w-8 h-8 (32px)
//   - icon-2xl → w-10 h-10 (40px)
//   - icon-3xl → w-12 h-12 (48px)
func Icon(name string, class string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// Convert custom icon size classes to Tailwind utilities
		class = convertIconSizeToTailwind(class)

		_, err := fmt.Fprintf(w,
			`<svg class="inline-block %s" aria-hidden="true"><use href="/static/icons.svg#%s"></use></svg>`,
			class, name)
		return err
	})
}

// convertIconSizeToTailwind converts custom icon size classes to Tailwind utilities.
func convertIconSizeToTailwind(class string) string {
	// Replace icon size classes with Tailwind utilities
	class = strings.ReplaceAll(class, "icon-3xl", "w-12 h-12")
	class = strings.ReplaceAll(class, "icon-2xl", "w-10 h-10")
	class = strings.ReplaceAll(class, "icon-xl", "w-8 h-8")
	class = strings.ReplaceAll(class, "icon-lg", "w-6 h-6")
	class = strings.ReplaceAll(class, "icon-sm", "w-4 h-4")
	class = strings.ReplaceAll(class, "icon-xs", "w-3 h-3")

	// Replace standalone "icon" with default size (but preserve icon-* variants)
	// Only replace if it's the whole word "icon" not part of another class
	if class == "icon" {
		class = "w-5 h-5"
	} else if strings.HasPrefix(class, "icon ") {
		class = "w-5 h-5 " + strings.TrimPrefix(class, "icon ")
	} else if strings.HasSuffix(class, " icon") {
		class = strings.TrimSuffix(class, " icon") + " w-5 h-5"
	} else if strings.Contains(class, " icon ") {
		class = strings.ReplaceAll(class, " icon ", " w-5 h-5 ")
	}

	return class
}

// IconWithLabel renders icon with accessible label.
func IconWithLabel(name string, label string, class string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		// Convert custom icon size classes to Tailwind utilities
		class = convertIconSizeToTailwind(class)

		_, err := fmt.Fprintf(w,
			`<span class="inline-flex items-center gap-1">
                <svg class="inline-block %s" aria-label="%s"><use href="/static/icons.svg#%s"></use></svg>
                <span class="sr-only">%s</span>
            </span>`,
			class, label, name, label)
		return err
	})
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

// StatusBadge renders a status badge component with Tailwind utilities.
// Badge pattern: inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium
//   - Success: bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300
//   - Error: bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300
func StatusBadge(status string, message string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		switch status {
		case "accessible":
			_, err := fmt.Fprintf(w,
				`<span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300" role="status">
					<svg class="inline-block w-4 h-4" aria-hidden="true"><use href="/static/icons.svg#check-circle"></use></svg>
					<span>Accessible</span>
				</span>`)
			return err
		case "inaccessible":
			html := `<span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300" role="status">
				<svg class="inline-block w-4 h-4" aria-hidden="true"><use href="/static/icons.svg#x-circle"></use></svg>
				<span>Inaccessible</span>
			</span>`

			if message != "" {
				html += fmt.Sprintf(`<div class="mt-1 text-xs text-gray-600 dark:text-gray-400" title="%s">
					<small>%s</small>
				</div>`, message, truncateETag(message, 40))
			}

			_, err := fmt.Fprintf(w, "%s", html)
			return err
		default:
			return nil
		}
	})
}

// SkipToContent renders a skip to content link for accessibility.
// The link is visually hidden but becomes visible when focused via keyboard.
func SkipToContent() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w,
			`<a href="#main-content" class="sr-only focus:not-sr-only focus:absolute focus:top-0 focus:left-0 focus:z-50 focus:px-4 focus:py-2 focus:bg-blue-600 focus:text-white focus:font-medium">
                Skip to main content
            </a>`)
		return err
	})
}

/*
─────────────────────────────────────────────────────────────────────────────
TAILWIND CSS COMPONENT PATTERNS
─────────────────────────────────────────────────────────────────────────────

This section documents reusable Tailwind CSS patterns for consistent styling
across the application. Apply these classes directly in .templ templates.

BUTTON PATTERNS
───────────────

Primary Button (Call-to-action):
  Base classes:
    bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium
    transition-colors duration-200

  With focus ring:
    bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium
    focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2
    transition-colors duration-200

  Dark mode variant:
    bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600
    text-white px-4 py-2 rounded-lg font-medium

  Example usage:
    <button class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium">
      Submit
    </button>

Secondary Button (Neutral action):
  Base classes:
    bg-gray-100 hover:bg-gray-200 text-gray-900 px-4 py-2 rounded-lg font-medium
    transition-colors duration-200

  Dark mode variant:
    bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700
    text-gray-900 dark:text-gray-100 px-4 py-2 rounded-lg font-medium

  Example usage:
    <button class="bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100 px-4 py-2 rounded-lg font-medium">
      Cancel
    </button>

Action Button (Icon + text in lists):
  Base classes:
    inline-flex items-center gap-2 bg-gray-100 hover:bg-gray-200
    dark:bg-gray-800 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100
    px-3 py-1.5 rounded-md text-sm font-medium transition-colors duration-200

  With icon example:
    <button class="inline-flex items-center gap-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700 px-3 py-1.5 rounded-md text-sm font-medium">
      @Icon("download", "w-4 h-4")
      <span>Download</span>
    </button>

Danger Button (Destructive action):
  Base classes:
    bg-red-600 hover:bg-red-700 dark:bg-red-500 dark:hover:bg-red-600
    text-white px-4 py-2 rounded-lg font-medium transition-colors duration-200

  Example usage:
    <button class="bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-lg font-medium">
      Delete
    </button>

Ghost Button (Minimal style):
  Base classes:
    hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-900 dark:text-gray-100
    px-3 py-1.5 rounded-md text-sm font-medium transition-colors duration-200

  Example usage:
    <button class="hover:bg-gray-100 dark:hover:bg-gray-800 px-3 py-1.5 rounded-md text-sm font-medium">
      View Details
    </button>

Disabled Button:
  Add to any button pattern:
    opacity-50 cursor-not-allowed pointer-events-none

  Example:
    <button class="bg-blue-600 text-white px-4 py-2 rounded-lg font-medium opacity-50 cursor-not-allowed" disabled>
      Submit
    </button>

LINK PATTERNS
─────────────

Primary Link:
  text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300
  underline decoration-1 underline-offset-2

Secondary Link (No underline by default):
  text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300
  hover:underline

FORM INPUT PATTERNS
───────────────────

Text Input:
  w-full px-3 py-2 border border-gray-300 dark:border-gray-600
  rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100
  focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors

Search Input:
  w-full px-4 py-2 pl-10 border border-gray-300 dark:border-gray-600
  rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100
  focus:ring-2 focus:ring-blue-500 focus:border-blue-500

CARD PATTERNS
─────────────

Basic Card:
  bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200
  dark:border-gray-700 p-4

Card with Hover:
  bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200
  dark:border-gray-700 p-4 hover:shadow-md transition-shadow duration-200

BADGE PATTERNS
──────────────

See StatusBadge() function for success/error badge implementation.

Info Badge:
  inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium
  bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300

Warning Badge:
  inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium
  bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300

SPACING & LAYOUT
────────────────

Container padding:
  px-4 py-6 md:px-6 md:py-8 lg:px-8 lg:py-10

Section spacing:
  mb-6 md:mb-8 lg:mb-10

Gap between items:
  gap-2    (8px)  - Tight spacing
  gap-4    (16px) - Default spacing
  gap-6    (24px) - Relaxed spacing

RESPONSIVE BREAKPOINTS
──────────────────────

sm:  640px  - Small tablets
md:  768px  - Tablets
lg:  1024px - Small desktops
xl:  1280px - Large desktops
2xl: 1536px - Extra large screens

Example responsive button:
  <button class="px-3 py-1.5 text-sm md:px-4 md:py-2 md:text-base lg:px-6 lg:py-3">
    Responsive Button
  </button>

ACCESSIBILITY
─────────────

Focus rings (required for keyboard navigation):
  focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2

Screen reader only text:
  sr-only (utility class that hides content visually but keeps it for screen readers)

Skip to content link pattern (already implemented in SkipToContent()):
  See SkipToContent() function

─────────────────────────────────────────────────────────────────────────────
*/
