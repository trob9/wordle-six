package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

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

	// API routes
	mux.HandleFunc("POST /api/result", handleSubmitResult)
	mux.HandleFunc("GET /api/leaderboard", handleGetLeaderboard)
	mux.HandleFunc("GET /api/game-state", handleGetGameState)
	mux.HandleFunc("POST /api/save-progress", handleSaveProgress)
	mux.HandleFunc("GET /api/user-stats", handleGetUserStats)
	mux.HandleFunc("POST /api/user-stats", handleSaveUserStats)
	mux.HandleFunc("POST /api/display-name", handleUpdateDisplayName)

	// Static files - serve from current directory
	staticDir := "./static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = "."
	}
	fs := http.FileServer(http.Dir(staticDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Don't serve Go source files
		if strings.HasSuffix(r.URL.Path, ".go") || strings.HasSuffix(r.URL.Path, ".mod") || strings.HasSuffix(r.URL.Path, ".sum") {
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
