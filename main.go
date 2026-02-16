package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// limitBody wraps a handler to enforce a max request body size.
func limitBody(maxBytes int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("GET /auth/{provider}", handleAuthStart)
	mux.HandleFunc("GET /auth/{provider}/callback", handleAuthCallback)
	mux.HandleFunc("GET /auth/me", handleAuthMe)
	mux.HandleFunc("POST /auth/logout", handleAuthLogout)

	// API routes (10KB body limit for all POST endpoints)
	mux.HandleFunc("POST /api/result", limitBody(10*1024, handleSubmitResult))
	mux.HandleFunc("GET /api/leaderboard", handleGetLeaderboard)
	mux.HandleFunc("GET /api/game-state", handleGetGameState)
	mux.HandleFunc("POST /api/save-progress", limitBody(10*1024, handleSaveProgress))
	mux.HandleFunc("GET /api/user-stats", handleGetUserStats)
	mux.HandleFunc("POST /api/user-stats", limitBody(10*1024, handleSaveUserStats))
	mux.HandleFunc("POST /api/display-name", limitBody(1024, handleUpdateDisplayName))
	mux.HandleFunc("POST /api/admin/ban", limitBody(1024, handleBanUser))
	mux.HandleFunc("GET /api/admin/users", handleListUsers)

	// Static files - serve from current directory
	staticDir := "./static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = "."
	}
	fs := http.FileServer(http.Dir(staticDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Block hidden files and directories (.git, .env, etc.)
		if strings.Contains(r.URL.Path, "/.") || strings.HasPrefix(filepath.Base(r.URL.Path), ".") {
			http.NotFound(w, r)
			return
		}
		// Don't serve Go source files or sensitive extensions
		lower := strings.ToLower(r.URL.Path)
		if strings.HasSuffix(lower, ".go") || strings.HasSuffix(lower, ".mod") || strings.HasSuffix(lower, ".sum") ||
			strings.HasSuffix(lower, ".env") || strings.HasSuffix(lower, ".bak") || strings.HasSuffix(lower, ".old") {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Wordle Six server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
