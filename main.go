package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wordle-six/internal/auth"
	"wordle-six/internal/game"
	"wordle-six/internal/middleware"
	"wordle-six/internal/store"
)

func main() {
	if err := store.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	auth.Init()

	mux := http.NewServeMux()

	// Auth routes (5 req/min per IP)
	mux.HandleFunc("GET /auth/{provider}", middleware.RateLimit(5.0/60, 5, auth.HandleAuthStart))
	mux.HandleFunc("GET /auth/{provider}/callback", middleware.RateLimit(5.0/60, 5, auth.HandleAuthCallback))
	mux.HandleFunc("GET /auth/me", middleware.RateLimit(30.0/60, 10, auth.HandleAuthMe))
	mux.HandleFunc("POST /auth/logout", middleware.RateLimit(5.0/60, 3, auth.HandleAuthLogout))

	// API routes: rate limited + body limited + CSRF (RequireJSON) on POST
	mux.HandleFunc("POST /api/result", middleware.RateLimit(10.0/60, 5, middleware.RequireJSON(middleware.LimitBody(10*1024, game.HandleSubmitResult))))
	mux.HandleFunc("GET /api/leaderboard", middleware.RateLimit(30.0/60, 10, game.HandleGetLeaderboard))
	mux.HandleFunc("GET /api/game-state", middleware.RateLimit(30.0/60, 10, game.HandleGetGameState))
	mux.HandleFunc("POST /api/save-progress", middleware.RateLimit(30.0/60, 10, middleware.RequireJSON(middleware.LimitBody(10*1024, game.HandleSaveProgress))))
	mux.HandleFunc("GET /api/user-stats", middleware.RateLimit(30.0/60, 10, game.HandleGetUserStats))
	mux.HandleFunc("POST /api/user-stats", middleware.RateLimit(10.0/60, 5, middleware.RequireJSON(middleware.LimitBody(10*1024, game.HandleSaveUserStats))))
	mux.HandleFunc("POST /api/fingerprint", middleware.RateLimit(10.0/60, 5, middleware.RequireJSON(middleware.LimitBody(10*1024, game.HandleFingerprint))))
	mux.HandleFunc("POST /api/display-name", middleware.RateLimit(5.0/60, 3, middleware.RequireJSON(middleware.LimitBody(1024, game.HandleUpdateDisplayName))))
	mux.HandleFunc("POST /api/admin/ban", middleware.RateLimit(5.0/60, 3, middleware.RequireJSON(middleware.LimitBody(1024, game.HandleBanUser))))
	mux.HandleFunc("GET /api/admin/users", middleware.RateLimit(10.0/60, 5, game.HandleListUsers))

	// Redirect /favicon.ico to the app icon so browsers don't generate 404s
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/icon-192.png", http.StatusMovedPermanently)
	})

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

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           middleware.AccessLog(mux),
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	log.Printf("Wordle Six server starting on :%s", port)
	log.Fatal(server.ListenAndServe())
}
