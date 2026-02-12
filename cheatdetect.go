package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type cachedGeoResult struct {
	timezone    string
	countryCode string
	fetchedAt   time.Time
}

var (
	geoCache   = make(map[string]cachedGeoResult)
	geoCacheMu sync.Mutex
)

// checkTimezone logs timezone data and returns true if suspicious activity is detected.
func checkTimezone(userID int64, clientTime string, tzOffset int, ip string, endpoint string) bool {
	now := time.Now().UTC()

	// Insert event
	_, err := db.Exec(`
		INSERT INTO tz_events (user_id, server_utc, client_time, tz_offset, ip, endpoint)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, now.Format(time.RFC3339), clientTime, tzOffset, ip, endpoint)
	if err != nil {
		log.Printf("cheatdetect: failed to insert tz_event: %v", err)
	}

	var reasons []string

	// 1. Impossible date check
	ct, err := time.Parse(time.RFC3339, clientTime)
	if err == nil {
		diffHours := math.Abs(ct.Sub(now).Hours())
		if diffHours > 26 { // no timezone is >14h offset, so >26h diff is impossible
			reasons = append(reasons, fmt.Sprintf("impossible_date_diff=%.1fh", diffHours))
		}
	}

	// 2. Timezone drift detection: check if offset changed within 30 min
	var prevOffset int
	var prevServerUTC string
	err = db.QueryRow(`
		SELECT tz_offset, server_utc FROM tz_events
		WHERE user_id = ? AND id != (SELECT MAX(id) FROM tz_events WHERE user_id = ?)
		ORDER BY server_utc DESC LIMIT 1
	`, userID, userID).Scan(&prevOffset, &prevServerUTC)
	if err == nil {
		prevTime, parseErr := time.Parse(time.RFC3339, prevServerUTC)
		if parseErr == nil && now.Sub(prevTime) < 30*time.Minute {
			if prevOffset != tzOffset {
				reasons = append(reasons, fmt.Sprintf("tz_drift=%d->%d_in_%s", prevOffset, tzOffset, now.Sub(prevTime).Round(time.Second)))
			}
		}
	}

	// 3. IP geolocation mismatch
	if ip != "" && !strings.HasPrefix(ip, "127.") && ip != "::1" {
		geoTZ := getGeoTimezone(ip)
		if geoTZ != "" {
			loc, err := time.LoadLocation(geoTZ)
			if err == nil {
				_, expectedOffset := time.Now().In(loc).Zone()
				expectedOffsetMin := -(expectedOffset / 60) // JS getTimezoneOffset is inverted
				diff := math.Abs(float64(tzOffset - expectedOffsetMin))
				if diff > 120 { // more than 2 hours off
					reasons = append(reasons, fmt.Sprintf("ip_tz_mismatch: ip=%s geo_tz=%s expected_offset=%d got=%d", ip, geoTZ, expectedOffsetMin, tzOffset))
				}
			}
		}
	}

	if len(reasons) > 0 {
		// Look up display name for logging
		var displayName string
		db.QueryRow("SELECT COALESCE(custom_name, display_name) FROM users WHERE id = ?", userID).Scan(&displayName)

		logEntry := fmt.Sprintf("[%s] user_id=%d name=%q reasons=[%s] client_time=%s tz_offset=%d ip=%s endpoint=%s\n",
			now.Format(time.RFC3339), userID, displayName, strings.Join(reasons, "; "), clientTime, tzOffset, ip, endpoint)

		log.Printf("cheatdetect: %s", logEntry)

		f, err := os.OpenFile("/data/cheatlog.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(logEntry)
			f.Close()
		}

		return true
	}

	return false
}

// getGeoTimezone returns the IANA timezone for an IP, using a 24h cache.
func getGeoTimezone(ip string) string {
	geoCacheMu.Lock()
	cached, ok := geoCache[ip]
	geoCacheMu.Unlock()

	if ok && time.Since(cached.fetchedAt) < 24*time.Hour {
		return cached.timezone
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://ip-api.com/json/%s?fields=status,timezone", ip))
	if err != nil {
		log.Printf("cheatdetect: geo lookup failed for %s: %v", ip, err)
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Status   string `json:"status"`
		Timezone string `json:"timezone"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Status != "success" {
		return ""
	}

	geoCacheMu.Lock()
	geoCache[ip] = cachedGeoResult{timezone: result.Timezone, fetchedAt: time.Now()}
	geoCacheMu.Unlock()

	return result.Timezone
}

// getClientIP extracts the client IP from the request, preferring Cf-Connecting-Ip.
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("Cf-Connecting-Ip"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	// RemoteAddr is "ip:port"
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
