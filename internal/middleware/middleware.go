package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"wordle-six/internal/netutil"
)

// --- Body limiting ---

// LimitBody wraps a handler to enforce a max request body size.
func LimitBody(maxBytes int64, next http.HandlerFunc) http.HandlerFunc {
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

	if l, ok := limiters[ip]; ok {
		l.lastSeen = time.Now()
		return l.limiter
	}
	l := rate.NewLimiter(rps, burst)
	limiters[ip] = &ipLimiter{limiter: l, lastSeen: time.Now()}
	return l
}

// RateLimit wraps a handler with per-IP rate limiting.
func RateLimit(rps rate.Limit, burst int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := netutil.ClientIP(r)
		if !getLimiter(ip, rps, burst).Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// RequireJSON rejects POST requests that don't have Content-Type: application/json.
// This provides CSRF protection since HTML forms cannot send application/json.
func RequireJSON(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		next(w, r)
	}
}

// --- Access logging ---

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// scannerUAs are known scanner/bot user agent substrings (lowercased).
var scannerUAs = []string{
	"nikto", "sqlmap", "masscan", "nmap", "zgrab", "nuclei",
	"gobuster", "dirb", "dirbuster", "wfuzz", "ffuf", "hydra",
	"metasploit", "acunetix", "nessus", "openvas", "skipfish",
	"appscan", "webinspect", "libwww-perl", "scrapy",
	"semrushbot", "ahrefsbot", "mj12bot", "dotbot", "petalbot",
	"bytespider", "gptbot",
}

// scannerPaths are paths commonly probed by scanners and exploit scripts.
var scannerPaths = []string{
	"/.env", "/.git/", "/wp-admin", "/wp-login.php", "/wp-content/",
	"/phpmyadmin", "/.htaccess", "/config.php", "/backup",
	"/.ds_store", "/etc/passwd", "/cgi-bin/", "/xmlrpc.php",
	"/actuator", "/.aws/", "/.ssh/", "/shell.php", "/cmd.php",
	"/eval.php", "/.bash_history", "/id_rsa", "/console",
	"/jmx-console", "/manager/html", "/solr/", "/jenkins",
	"/.well-known/security.txt",
}

// unusualMethods are HTTP methods beyond standard REST operations.
var unusualMethods = map[string]bool{
	"TRACE": true, "CONNECT": true, "DEBUG": true,
	"PROPFIND": true, "PROPPATCH": true, "MKCOL": true,
	"COPY": true, "MOVE": true, "LOCK": true, "UNLOCK": true,
	"SEARCH": true,
}

// AccessLog is an HTTP middleware that logs every request as structured JSON.
// It detects scanner patterns and flags suspicious requests for easy monitoring.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		latency := time.Since(start)

		ip := netutil.ClientIP(r)
		ua := r.UserAgent()
		path := r.URL.Path
		method := r.Method

		flags := detectFlags(method, path, ua, rw.status)

		entry := map[string]interface{}{
			"ts":         time.Now().UTC().Format(time.RFC3339),
			"method":     method,
			"path":       path,
			"status":     rw.status,
			"latency_ms": latency.Milliseconds(),
			"ip":         ip,
			"ua":         ua,
			"size":       rw.size,
		}
		if len(flags) > 0 {
			entry["flags"] = flags
		}
		if ref := r.Referer(); ref != "" {
			entry["referer"] = ref
		}

		b, _ := json.Marshal(entry)
		fmt.Println(string(b))
	})
}

func detectFlags(method, path, ua string, status int) []string {
	var flags []string

	uaLower := strings.ToLower(ua)
	for _, s := range scannerUAs {
		if strings.Contains(uaLower, s) {
			flags = append(flags, "scanner_ua")
			break
		}
	}

	pathLower := strings.ToLower(path)
	for _, p := range scannerPaths {
		if strings.HasPrefix(pathLower, p) || strings.Contains(pathLower, p) {
			flags = append(flags, "scanner_path")
			break
		}
	}

	if unusualMethods[strings.ToUpper(method)] {
		flags = append(flags, "unusual_method")
	}

	if status >= 400 && status < 500 {
		flags = append(flags, "client_error")
	}
	if status >= 500 {
		flags = append(flags, "server_error")
	}
	return flags
}

// LogAuthEvent writes a structured JSON auth event log entry.
func LogAuthEvent(event, provider, result, ip string, extra map[string]interface{}) {
	entry := map[string]interface{}{
		"ts":       time.Now().UTC().Format(time.RFC3339),
		"event":    event,
		"provider": provider,
		"result":   result,
		"ip":       ip,
	}
	for k, v := range extra {
		entry[k] = v
	}
	b, _ := json.Marshal(entry)
	fmt.Println(string(b))
}
