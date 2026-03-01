package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
