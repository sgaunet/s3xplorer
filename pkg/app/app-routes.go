package app

import "github.com/sgaunet/s3xplorer/pkg/views"

// initRouter initializes the router of the App.
func (s *App) initRouter() {
	s.router.PathPrefix("/static").Handler(views.StaticHandler)
	s.router.HandleFunc("/favicon.ico", views.FaviconHandler)
	s.router.HandleFunc("/", s.IndexBucket)
	s.router.HandleFunc("/download", s.DownloadFile)
	s.router.HandleFunc("/restore", s.RestoreHandler)
	s.router.HandleFunc("/search", s.SearchHandler)
	s.router.HandleFunc("/buckets", s.BucketListingHandler)
	s.srv.Handler = s.router
}
