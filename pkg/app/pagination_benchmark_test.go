package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkParsePaginationParams benchmarks pagination parameter parsing
func BenchmarkParsePaginationParams(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/?folder=test/&page=5", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = ParsePaginationParams(req)
	}
}

// BenchmarkParsePaginationParams_DefaultPage benchmarks default page handling
func BenchmarkParsePaginationParams_DefaultPage(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/?folder=test/", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = ParsePaginationParams(req)
	}
}

// BenchmarkParsePaginationParams_InvalidPage benchmarks error handling
func BenchmarkParsePaginationParams_InvalidPage(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/?folder=test/&page=invalid", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = ParsePaginationParams(req)
	}
}

// BenchmarkValidatePageNumber benchmarks page validation logic
func BenchmarkValidatePageNumber(b *testing.B) {
	testCases := []struct {
		name       string
		page       int
		maxPages   int
	}{
		{"ValidPage", 5, 10},
		{"FirstPage", 1, 10},
		{"LastPage", 10, 10},
		{"AboveMax", 15, 10},
		{"BelowMin", 0, 10},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = ValidatePageNumber(tc.page, tc.maxPages)
			}
		})
	}
}

// BenchmarkValidatePageNumber_HighPageNumbers tests with large page numbers
func BenchmarkValidatePageNumber_HighPageNumbers(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		page := (i % 1000) + 1
		_ = ValidatePageNumber(page, 1000)
	}
}

// BenchmarkParsePaginationParams_ConcurrentRequests benchmarks concurrent parsing
func BenchmarkParsePaginationParams_ConcurrentRequests(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			url := fmt.Sprintf("/?folder=test/&page=%d", (i%100)+1)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			_, _ = ParsePaginationParams(req)
			i++
		}
	})
}

// BenchmarkValidatePageNumber_ConcurrentValidation benchmarks concurrent validation
func BenchmarkValidatePageNumber_ConcurrentValidation(b *testing.B) {
	const maxPages = 100

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			page := (i % 150) + 1
			_ = ValidatePageNumber(page, maxPages)
			i++
		}
	})
}

// BenchmarkHTTPRequest_WithPagination benchmarks complete HTTP request handling
func BenchmarkHTTPRequest_WithPagination(b *testing.B) {
	testCases := []struct {
		name string
		url  string
	}{
		{"Page1", "/?folder=test/&page=1"},
		{"Page5", "/?folder=test/&page=5"},
		{"Page50", "/?folder=test/&page=50"},
		{"DeepFolder", "/?folder=a/b/c/d/e/&page=3"},
		{"NoPage", "/?folder=test/"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodGet, tc.url, nil)
				_, _ = ParsePaginationParams(req)
			}
		})
	}
}

// BenchmarkRequestRecording benchmarks httptest.ResponseRecorder usage
func BenchmarkRequestRecording(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/?page=5", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		_, _ = ParsePaginationParams(req)
		_ = w
	}
}

// BenchmarkFullPaginationFlow simulates complete pagination flow
func BenchmarkFullPaginationFlow(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate user request
		url := fmt.Sprintf("/?folder=data/2024/&page=%d", (i%10)+1)
		req := httptest.NewRequest(http.MethodGet, url, nil)

		// Parse pagination params
		page, err := ParsePaginationParams(req)
		if err != nil {
			continue
		}

		// Validate page number (simulate with 20 total pages)
		_ = ValidatePageNumber(page, 20)

		// Simulate response recording
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)
	}
}

// BenchmarkMemoryAllocations_PaginationParams measures memory allocations
func BenchmarkMemoryAllocations_PaginationParams(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/?folder=test/data/2024/01/&page=15", nil)

	b.ResetTimer()
	b.ReportAllocs()

	var page int
	for i := 0; i < b.N; i++ {
		page, _ = ParsePaginationParams(req)
	}

	// Prevent compiler optimization
	_ = page
}

// BenchmarkWorstCase_InvalidInputs benchmarks worst-case error paths
func BenchmarkWorstCase_InvalidInputs(b *testing.B) {
	badRequests := []string{
		"/?page=0",
		"/?page=-1",
		"/?page=abc",
		"/?page=1.5",
		"/?page=999999999999999",
		"/?page=",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		url := badRequests[i%len(badRequests)]
		req := httptest.NewRequest(http.MethodGet, url, nil)
		_, _ = ParsePaginationParams(req)
	}
}

// Note: For comprehensive performance testing, you would want to:
//
// 1. End-to-end handler benchmarks with database:
//    func BenchmarkHandlerIndexHierarchical(b *testing.B) {
//        db := setupBenchmarkDatabase(b)
//        app := setupTestApp(b, db)
//        seedData(b, db, 10000) // 10k objects
//
//        b.ResetTimer()
//        for i := 0; i < b.N; i++ {
//            req := httptest.NewRequest("GET", "/?folder=test/&page=5", nil)
//            w := httptest.NewRecorder()
//            app.Handler.ServeHTTP(w, req)
//
//            if w.Code != 200 {
//                b.Fatalf("Expected 200, got %d", w.Code)
//            }
//        }
//
//        // Verify response time < 1 second requirement
//        avgTime := b.Elapsed() / time.Duration(b.N)
//        if avgTime > time.Second {
//            b.Errorf("Average response time %v exceeds 1 second", avgTime)
//        }
//    }
//
// 2. Load testing benchmarks:
//    - Simulate multiple concurrent users
//    - Test with realistic data patterns
//    - Measure throughput (requests/second)
//    - Track resource usage (CPU, memory)
//
// 3. Profiling benchmarks:
//    go test -bench=. -cpuprofile=cpu.prof
//    go tool pprof cpu.prof
//
// Performance targets from requirements:
// - Parse/validation: < 1ms per request
// - Full handler: < 1 second per request
// - Support concurrent access
// - Minimal memory allocations
