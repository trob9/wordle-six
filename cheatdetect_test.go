package main

import (
	"net/http/httptest"
	"testing"
	"time"
)

// --- isPrivateIP ---

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
		desc     string
	}{
		// Private ranges
		{"10.0.0.1", true, "10.x RFC1918"},
		{"10.255.255.255", true, "10.x high end"},
		{"172.16.0.1", true, "172.16.x RFC1918"},
		{"172.31.255.255", true, "172.31.x high end"},
		{"192.168.0.1", true, "192.168.x RFC1918"},
		{"192.168.1.100", true, "192.168.x common LAN"},
		{"127.0.0.1", true, "loopback"},
		{"127.0.0.2", true, "loopback range"},
		{"169.254.1.1", true, "link-local"},
		{"100.64.0.1", true, "CGNAT/Tailscale"},
		{"100.127.255.254", true, "CGNAT high end"},
		{"0.0.0.0", true, "zero address"},
		{"240.0.0.1", true, "reserved range"},
		{"::1", true, "IPv6 loopback"},
		{"fc00::1", true, "IPv6 ULA"},
		{"fe80::1", true, "IPv6 link-local"},

		// Public IPs
		{"8.8.8.8", false, "Google DNS"},
		{"1.1.1.1", false, "Cloudflare DNS"},
		{"203.0.113.50", false, "public IP"},
		{"104.16.0.1", false, "Cloudflare IP"},

		// Edge cases
		{"not-an-ip", true, "unparseable treated as private"},
		{"", true, "empty treated as private"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := isPrivateIP(tt.ip)
			if got != tt.expected {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

// --- checkTimezone ---

func TestCheckTimezone_NormalRequest(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	clientTime := time.Now().UTC().Format(time.RFC3339)
	suspicious := checkTimezone(user.ID, clientTime, -600, "1.2.3.4", "test")
	if suspicious {
		t.Error("expected normal request to not be suspicious")
	}
}

func TestCheckTimezone_ImpossibleDate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	// Client time 48 hours in the future
	futureTime := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	suspicious := checkTimezone(user.ID, futureTime, -600, "1.2.3.4", "test")
	if !suspicious {
		t.Error("expected impossible date to be suspicious")
	}
}

func TestCheckTimezone_TzDrift(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	clientTime := time.Now().UTC().Format(time.RFC3339)

	// First request establishes offset -300
	checkTimezone(user.ID, clientTime, -300, "1.2.3.4", "test")

	// Second request immediately with different offset — should detect drift
	// (within 30 min window, offset changed)
	suspicious := checkTimezone(user.ID, clientTime, -600, "1.2.3.4", "test")
	if !suspicious {
		t.Error("expected tz drift to be detected")
	}
}

func TestCheckTimezone_NoDriftAfter30Min(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	now := time.Now().UTC()
	clientTime := now.Format(time.RFC3339)

	// Insert an old event with different offset (>30 min ago)
	oldTime := now.Add(-45 * time.Minute).Format(time.RFC3339)
	db.Exec(`INSERT INTO tz_events (user_id, server_utc, client_time, tz_offset, ip, endpoint)
		VALUES (?, ?, ?, ?, ?, ?)`, user.ID, oldTime, clientTime, -300, "1.2.3.4", "test")

	// New request with different offset — should NOT be suspicious (>30 min gap)
	suspicious := checkTimezone(user.ID, clientTime, -600, "1.2.3.4", "test")
	if suspicious {
		t.Error("expected no drift detection after 30 min gap")
	}
}

func TestCheckTimezone_AuditTrailInserted(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	clientTime := time.Now().UTC().Format(time.RFC3339)
	checkTimezone(user.ID, clientTime, -600, "1.2.3.4", "test-endpoint")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM tz_events WHERE user_id = ? AND endpoint = ?", user.ID, "test-endpoint").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 tz_event, got %d", count)
	}
}

// --- getClientIP ---

func TestGetClientIP_CfConnectingIp(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cf-Connecting-Ip", "1.2.3.4")
	if got := getClientIP(req); got != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", got)
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "5.6.7.8,proxy1,proxy2")
	if got := getClientIP(req); got != "5.6.7.8" {
		t.Errorf("expected 5.6.7.8, got %s", got)
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "9.8.7.6:12345"
	if got := getClientIP(req); got != "9.8.7.6" {
		t.Errorf("expected 9.8.7.6, got %s", got)
	}
}
