package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// limitBody wraps a handler to enforce a max request body size.
func limitBody(maxBytes int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}

// --- Per-IP rate limiting ---

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters   = make(map[string]*ipLimiter)
	limitersMu sync.Mutex
)

func init() {
	// Clean up stale entries every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limitersMu.Lock()
			for ip, l := range limiters {
				if time.Since(l.lastSeen) > 10*time.Minute {
					delete(limiters, ip)
				}
			}
			limitersMu.Unlock()
		}
	}()
}

func getLimiter(ip string, rps rate.Limit, burst int) *rate.Limiter {
	limitersMu.Lock()
	defer limitersMu.Unlock()

	key := ip
	if l, ok := limiters[key]; ok {
		l.lastSeen = time.Now()
		return l.limiter
	}
	l := rate.NewLimiter(rps, burst)
	limiters[key] = &ipLimiter{limiter: l, lastSeen: time.Now()}
	return l
}

func clientIP(r *http.Request) string {
	// CF-Connecting-IP is set by Cloudflare and is the most authoritative source
	// in this stack (Cloudflare Tunnel → Caddy → wordle-six). Caddy forwards all
	// incoming headers to the backend, so this header arrives unchanged.
	if cf := r.Header.Get("CF-Connecting-IP"); cf != "" {
		return strings.TrimSpace(cf)
	}
	// Fall back to X-Forwarded-For (set by intermediate proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	// Last resort: direct connection address (will be Caddy's Docker IP in prod)
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}

// rateLimit wraps a handler with per-IP rate limiting.
func rateLimit(rps rate.Limit, burst int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !getLimiter(ip, rps, burst).Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// requireJSON rejects POST requests that don't have Content-Type: application/json.
// This provides CSRF protection since HTML forms cannot send application/json.
func requireJSON(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		next(w, r)
	}
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	mux := http.NewServeMux()

	// Auth routes (5 req/min per IP)
	mux.HandleFunc("GET /auth/{provider}", rateLimit(5.0/60, 5, handleAuthStart))
	mux.HandleFunc("GET /auth/{provider}/callback", rateLimit(5.0/60, 5, handleAuthCallback))
	mux.HandleFunc("GET /auth/me", rateLimit(30.0/60, 10, handleAuthMe))
	mux.HandleFunc("POST /auth/logout", rateLimit(5.0/60, 3, handleAuthLogout))

	// API routes: rate limited + body limited + CSRF (requireJSON) on POST
	mux.HandleFunc("POST /api/result", rateLimit(10.0/60, 5, requireJSON(limitBody(10*1024, handleSubmitResult))))
	mux.HandleFunc("GET /api/leaderboard", rateLimit(30.0/60, 10, handleGetLeaderboard))
	mux.HandleFunc("GET /api/game-state", rateLimit(30.0/60, 10, handleGetGameState))
	mux.HandleFunc("POST /api/save-progress", rateLimit(30.0/60, 10, requireJSON(limitBody(10*1024, handleSaveProgress))))
	mux.HandleFunc("GET /api/user-stats", rateLimit(30.0/60, 10, handleGetUserStats))
	mux.HandleFunc("POST /api/user-stats", rateLimit(10.0/60, 5, requireJSON(limitBody(10*1024, handleSaveUserStats))))
	mux.HandleFunc("POST /api/fingerprint", rateLimit(10.0/60, 5, requireJSON(limitBody(10*1024, handleFingerprint))))
	mux.HandleFunc("POST /api/display-name", rateLimit(5.0/60, 3, requireJSON(limitBody(1024, handleUpdateDisplayName))))
	mux.HandleFunc("POST /api/admin/ban", rateLimit(5.0/60, 3, requireJSON(limitBody(1024, handleBanUser))))
	mux.HandleFunc("GET /api/admin/users", rateLimit(10.0/60, 5, handleListUsers))

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
		Handler:           accessLog(mux),
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	log.Printf("Wordle Six server starting on :%s", port)
	log.Fatal(server.ListenAndServe())
}
