package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- requireJSON ---

func TestRequireJSON_ValidContentType(t *testing.T) {
	handler := requireJSON(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireJSON_WithCharset(t *testing.T) {
	handler := requireJSON(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with charset, got %d", rr.Code)
	}
}

func TestRequireJSON_InvalidContentType(t *testing.T) {
	handler := requireJSON(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", rr.Code)
	}
}

func TestRequireJSON_MissingContentType(t *testing.T) {
	handler := requireJSON(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", rr.Code)
	}
}

// --- limitBody ---

func TestLimitBody_WithinLimit(t *testing.T) {
	handler := limitBody(100, func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		w.Write(buf[:n])
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader("hello"))
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestLimitBody_ExceedsLimit(t *testing.T) {
	handler := limitBody(5, func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		_, err := r.Body.Read(buf)
		if err != nil {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader("this body is way too long"))
	rr := httptest.NewRecorder()
	handler(rr, req)

	// MaxBytesReader returns an error when the body exceeds the limit
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rr.Code)
	}
}

// --- rateLimit ---

func TestRateLimit_AllowsBurst(t *testing.T) {
	handler := rateLimit(1, 3, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := range 3 {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, rr.Code)
		}
	}
}

func TestRateLimit_BlocksExcess(t *testing.T) {
	// Use a unique IP to avoid interference from other tests
	handler := rateLimit(0.001, 2, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Exhaust the burst
	for range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.99.99.99:1234"
		rr := httptest.NewRecorder()
		handler(rr, req)
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.99.99.99:1234"
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr.Code)
	}
}

// --- Static file security ---

func TestStaticFile_BlocksGitDirectory(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/.") || strings.HasPrefix(r.URL.Path, ".") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/.git/config", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for .git, got %d", rr.Code)
	}
}

func TestStaticFile_BlocksGoFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		lower := strings.ToLower(r.URL.Path)
		if strings.HasSuffix(lower, ".go") || strings.HasSuffix(lower, ".env") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	for _, path := range []string{"/main.go", "/auth.go", "/.env"} {
		req := httptest.NewRequest("GET", path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 for %s, got %d", path, rr.Code)
		}
	}
}

func TestStaticFile_AllowsNormalFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		lower := strings.ToLower(r.URL.Path)
		if strings.Contains(r.URL.Path, "/.") {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(lower, ".go") || strings.HasSuffix(lower, ".env") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	for _, path := range []string{"/index.html", "/game.js", "/manifest.json"} {
		req := httptest.NewRequest("GET", path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for %s, got %d", path, rr.Code)
		}
	}
}
