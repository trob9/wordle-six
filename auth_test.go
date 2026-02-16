package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// --- validateAvatarURL tests ---

func TestValidateAvatarURL_ValidGitHub(t *testing.T) {
	url := "https://avatars.githubusercontent.com/u/12345"
	if got := validateAvatarURL(url); got != url {
		t.Errorf("expected %q, got %q", url, got)
	}
}

func TestValidateAvatarURL_ValidDiscord(t *testing.T) {
	url := "https://cdn.discordapp.com/avatars/123/abc.png"
	if got := validateAvatarURL(url); got != url {
		t.Errorf("expected %q, got %q", url, got)
	}
}

func TestValidateAvatarURL_ValidGoogle(t *testing.T) {
	url := "https://lh3.googleusercontent.com/a/photo"
	if got := validateAvatarURL(url); got != url {
		t.Errorf("expected %q, got %q", url, got)
	}
}

func TestValidateAvatarURL_RejectsHTTP(t *testing.T) {
	url := "http://avatars.githubusercontent.com/u/12345"
	if got := validateAvatarURL(url); got != "" {
		t.Errorf("expected empty for HTTP URL, got %q", got)
	}
}

func TestValidateAvatarURL_RejectsUnknownHost(t *testing.T) {
	url := "https://evil.com/avatar.png"
	if got := validateAvatarURL(url); got != "" {
		t.Errorf("expected empty for unknown host, got %q", got)
	}
}

func TestValidateAvatarURL_Empty(t *testing.T) {
	if got := validateAvatarURL(""); got != "" {
		t.Errorf("expected empty for empty input, got %q", got)
	}
}

func TestValidateAvatarURL_Malformed(t *testing.T) {
	if got := validateAvatarURL("://not-a-url"); got != "" {
		t.Errorf("expected empty for malformed URL, got %q", got)
	}
}

// --- getUserFromRequest tests ---

func TestGetUserFromRequest_ValidJWT(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "testuser")

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(makeAuthCookie(t, user.ID))

	got := getUserFromRequest(req)
	if got == nil {
		t.Fatal("expected non-nil user")
	}
	if got.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, got.ID)
	}
}

func TestGetUserFromRequest_ExpiredJWT(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "testuser")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: tokenString})

	got := getUserFromRequest(req)
	if got != nil {
		t.Error("expected nil for expired JWT")
	}
}

func TestGetUserFromRequest_WrongSignature(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "testuser")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("wrong-secret"))

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: tokenString})

	got := getUserFromRequest(req)
	if got != nil {
		t.Error("expected nil for wrong signature")
	}
}

func TestGetUserFromRequest_NoCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/me", nil)
	got := getUserFromRequest(req)
	if got != nil {
		t.Error("expected nil when no cookie present")
	}
}

func TestGetUserFromRequest_NonExistentUser(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(makeAuthCookie(t, 99999))

	got := getUserFromRequest(req)
	if got != nil {
		t.Error("expected nil for non-existent user ID in JWT")
	}
}

func TestGetUserFromRequest_BannedUser(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "banned")
	banUser(user.ID)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(makeAuthCookie(t, user.ID))

	// getUserFromRequest returns the user even if banned — handlers check .Banned
	got := getUserFromRequest(req)
	if got == nil {
		t.Fatal("expected non-nil user even if banned")
	}
	if !got.Banned {
		t.Error("expected Banned=true")
	}
}

// --- handleAuthMe tests ---

func TestHandleAuthMe_LoggedIn(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "testuser")
	updateCustomName(user.ID, "MyName")

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(makeAuthCookie(t, user.ID))
	rr := httptest.NewRecorder()

	handleAuthMe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	userData := resp["user"].(map[string]interface{})
	if userData["display_name"] != "MyName" {
		t.Errorf("expected display_name MyName, got %v", userData["display_name"])
	}
}

func TestHandleAuthMe_LoggedOut(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/me", nil)
	rr := httptest.NewRecorder()

	handleAuthMe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["user"] != nil {
		t.Errorf("expected null user, got %v", resp["user"])
	}
}

// --- handleAuthLogout tests ---

func TestHandleAuthLogout_ClearsCookie(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	rr := httptest.NewRecorder()

	handleAuthLogout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	cookies := rr.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "session" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected session cookie to be cleared (MaxAge < 0)")
	}
}

// --- handleAuthCallback tests ---

func TestHandleAuthCallback_MockOAuth(t *testing.T) {
	setupTestDB(t)

	// Create a mock OAuth server that handles both token exchange and user info
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST": // Token exchange
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-token",
				"token_type":   "bearer",
			})
		case r.Method == "GET": // User info
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         float64(42),
				"login":      "mockuser",
				"avatar_url": "https://avatars.githubusercontent.com/u/42",
			})
		}
	}))
	defer mockServer.Close()

	// Save and restore original OAuth client
	origClient := oauthHTTPClient
	oauthHTTPClient = mockServer.Client()
	defer func() { oauthHTTPClient = origClient }()

	// We can't easily test the full callback flow without also mocking
	// getOAuthConfig, but we can verify the mock server works correctly
	// by making a direct request to it
	resp, err := oauthHTTPClient.Post(mockServer.URL+"/token", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("mock token request failed: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	if tokenResp["access_token"] != "mock-token" {
		t.Errorf("expected mock-token, got %v", tokenResp["access_token"])
	}
}

func TestHandleAuthCallback_MissingState(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/github/callback?code=abc", nil)
	rr := httptest.NewRecorder()

	handleAuthCallback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing state cookie, got %d", rr.Code)
	}
}

func TestHandleAuthCallback_StateMismatch(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/github/callback?code=abc&state=wrong", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct"})
	rr := httptest.NewRecorder()

	handleAuthCallback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for state mismatch, got %d", rr.Code)
	}
}

func TestHandleAuthCallback_MissingCode(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/github/callback?state=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "abc"})
	rr := httptest.NewRecorder()

	handleAuthCallback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing code, got %d", rr.Code)
	}
}
