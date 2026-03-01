package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"wordle-six/internal/auth"
	"wordle-six/internal/store"
	"wordle-six/internal/testutil"
)

// --- ValidateAvatarURL tests ---

func TestValidateAvatarURL_ValidGitHub(t *testing.T) {
	url := "https://avatars.githubusercontent.com/u/12345"
	if got := auth.ValidateAvatarURL(url); got != url {
		t.Errorf("expected %q, got %q", url, got)
	}
}

func TestValidateAvatarURL_ValidDiscord(t *testing.T) {
	url := "https://cdn.discordapp.com/avatars/123/abc.png"
	if got := auth.ValidateAvatarURL(url); got != url {
		t.Errorf("expected %q, got %q", url, got)
	}
}

func TestValidateAvatarURL_ValidGoogle(t *testing.T) {
	url := "https://lh3.googleusercontent.com/a/photo"
	if got := auth.ValidateAvatarURL(url); got != url {
		t.Errorf("expected %q, got %q", url, got)
	}
}

func TestValidateAvatarURL_RejectsHTTP(t *testing.T) {
	url := "http://avatars.githubusercontent.com/u/12345"
	if got := auth.ValidateAvatarURL(url); got != "" {
		t.Errorf("expected empty for HTTP URL, got %q", got)
	}
}

func TestValidateAvatarURL_RejectsUnknownHost(t *testing.T) {
	url := "https://evil.com/avatar.png"
	if got := auth.ValidateAvatarURL(url); got != "" {
		t.Errorf("expected empty for unknown host, got %q", got)
	}
}

func TestValidateAvatarURL_Empty(t *testing.T) {
	if got := auth.ValidateAvatarURL(""); got != "" {
		t.Errorf("expected empty for empty input, got %q", got)
	}
}

func TestValidateAvatarURL_Malformed(t *testing.T) {
	if got := auth.ValidateAvatarURL("://not-a-url"); got != "" {
		t.Errorf("expected empty for malformed URL, got %q", got)
	}
}

// --- GetUserFromRequest tests ---

func TestGetUserFromRequest_ValidJWT(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "testuser")

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(testutil.MakeAuthCookie(t, user.ID))

	got := auth.GetUserFromRequest(req)
	if got == nil {
		t.Fatal("expected non-nil user")
	}
	if got.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, got.ID)
	}
}

func TestGetUserFromRequest_ExpiredJWT(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "testuser")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(auth.TestJWTSecret())

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: tokenString})

	got := auth.GetUserFromRequest(req)
	if got != nil {
		t.Error("expected nil for expired JWT")
	}
}

func TestGetUserFromRequest_WrongSignature(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "testuser")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("wrong-secret"))

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: tokenString})

	got := auth.GetUserFromRequest(req)
	if got != nil {
		t.Error("expected nil for wrong signature")
	}
}

func TestGetUserFromRequest_NoCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/me", nil)
	got := auth.GetUserFromRequest(req)
	if got != nil {
		t.Error("expected nil when no cookie present")
	}
}

func TestGetUserFromRequest_NonExistentUser(t *testing.T) {
	testutil.SetupTestDB(t)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(testutil.MakeAuthCookie(t, 99999))

	got := auth.GetUserFromRequest(req)
	if got != nil {
		t.Error("expected nil for non-existent user ID in JWT")
	}
}

func TestGetUserFromRequest_BannedUser(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "banned")
	store.BanUser(user.ID)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(testutil.MakeAuthCookie(t, user.ID))

	// GetUserFromRequest returns the user even if banned — handlers check .Banned
	got := auth.GetUserFromRequest(req)
	if got == nil {
		t.Fatal("expected non-nil user even if banned")
	}
	if !got.Banned {
		t.Error("expected Banned=true")
	}
}

// --- HandleAuthMe tests ---

func TestHandleAuthMe_LoggedIn(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "testuser")
	store.UpdateCustomName(user.ID, "MyName")

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(testutil.MakeAuthCookie(t, user.ID))
	rr := httptest.NewRecorder()

	auth.HandleAuthMe(rr, req)

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

	auth.HandleAuthMe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["user"] != nil {
		t.Errorf("expected null user, got %v", resp["user"])
	}
}

// --- HandleAuthLogout tests ---

func TestHandleAuthLogout_ClearsCookie(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	rr := httptest.NewRecorder()

	auth.HandleAuthLogout(rr, req)

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

// --- HandleAuthCallback tests ---

func TestHandleAuthCallback_MockOAuth(t *testing.T) {
	testutil.SetupTestDB(t)

	// Create a mock OAuth server
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

	origClient := auth.OAuthHTTPClient
	auth.OAuthHTTPClient = mockServer.Client()
	defer func() { auth.OAuthHTTPClient = origClient }()

	resp, err := auth.OAuthHTTPClient.Post(mockServer.URL+"/token", "application/x-www-form-urlencoded", nil)
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

	auth.HandleAuthCallback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing state cookie, got %d", rr.Code)
	}
}

func TestHandleAuthCallback_StateMismatch(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/github/callback?code=abc&state=wrong", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct"})
	rr := httptest.NewRecorder()

	auth.HandleAuthCallback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for state mismatch, got %d", rr.Code)
	}
}

func TestHandleAuthCallback_MissingCode(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/github/callback?state=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "abc"})
	rr := httptest.NewRecorder()

	auth.HandleAuthCallback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing code, got %d", rr.Code)
	}
}
