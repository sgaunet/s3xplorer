package views

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"

	"github.com/a-h/templ"
)

// versionedAssets are hashed to produce the cache-buster. StaticHandler serves
// /static with a one-year immutable Cache-Control, so the URL must change
// whenever the bytes do.
var versionedAssets = []string{"static/bulma.min.css", "static/theme.css", "static/app.js"}

// assetVersion is the cache-buster appended to every /static asset URL.
// It is derived from the embedded asset bytes, so editing any of them
// invalidates the cached URL automatically — there is nothing to remember to bump.
var assetVersion = computeAssetVersion()

// assetVersionLength is the number of hex characters kept from the digest.
const assetVersionLength = 12

// computeAssetVersion hashes the embedded assets into a short, stable token.
func computeAssetVersion() string {
	h := sha256.New()
	for _, name := range versionedAssets {
		data, err := fs.ReadFile(staticCSS, name)
		if err != nil {
			// The files are embedded at compile time; a miss means the
			// //go:embed directive and this list have drifted apart.
			panic("views: missing embedded asset " + name + ": " + err.Error())
		}
		_, _ = h.Write([]byte(name))
		_, _ = h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))[:assetVersionLength]
}

// Container width modifiers for the layout's content wrapper.
const (
	// ContainerWide suits data tables (Bulma's widescreen breakpoint, 1152px).
	ContainerWide = "is-max-widescreen"
	// ContainerMedium suits status and detail pages (960px).
	ContainerMedium = "is-max-desktop"
	// ContainerNarrow suits single-message pages such as errors (768px).
	ContainerNarrow = "is-max-tablet"
)

// PageOpts configures the shared page shell rendered by Layout.
type PageOpts struct {
	// Title is the browser tab title.
	Title string
	// ActivePage marks the current nav item: "home", "search", "buckets" or "".
	ActivePage string
	// Container is a Bulma container modifier; defaults to ContainerWide.
	Container string
	// RefreshSecs adds a meta refresh when greater than zero.
	RefreshSecs int
}

// container returns the container modifier to apply, defaulting to ContainerWide.
func (o PageOpts) container() string {
	if o.Container == "" {
		return ContainerWide
	}
	return o.Container
}

// asset appends the cache-buster to a static asset path.
func asset(p string) templ.SafeURL {
	return templ.SafeURL(p + "?v=" + assetVersion)
}
