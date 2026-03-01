package netutil

import (
	"net/http"
	"strings"
)

// ClientIP extracts the real client IP from a request, respecting proxy headers.
// CF-Connecting-IP is set by Cloudflare and is the most authoritative source
// in this stack (Cloudflare Tunnel → Caddy → wordle-six). Caddy forwards all
// incoming headers to the backend, so this header arrives unchanged.
func ClientIP(r *http.Request) string {
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
