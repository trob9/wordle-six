package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a random secret if not set (sessions won't survive restarts)
		b := make([]byte, 32)
		rand.Read(b)
		secret = hex.EncodeToString(b)
	}
	jwtSecret = []byte(secret)
}

type oauthConfig struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	ClientID    string
	ClientSecret string
	Scopes      string
}

func getOAuthConfig(provider string) (*oauthConfig, error) {
	switch provider {
	case "github":
		return &oauthConfig{
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			Scopes:       "read:user",
		}, nil
	case "discord":
		return &oauthConfig{
			AuthURL:      "https://discord.com/api/oauth2/authorize",
			TokenURL:     "https://discord.com/api/oauth2/token",
			UserInfoURL:  "https://discord.com/api/users/@me",
			ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
			ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
			Scopes:       "identify",
		}, nil
	case "google":
		return &oauthConfig{
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			Scopes:       "https://www.googleapis.com/auth/userinfo.profile",
		}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

func handleAuthStart(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	cfg, err := getOAuthConfig(provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if cfg.ClientID == "" {
		http.Error(w, provider+" OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Generate state token for CSRF protection
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	callbackURL := fmt.Sprintf("https://wordle-six.tomtom.fyi/auth/%s/callback", provider)

	params := url.Values{
		"client_id":    {cfg.ClientID},
		"redirect_uri": {callbackURL},
		"scope":        {cfg.Scopes},
		"state":        {state},
	}

	if provider == "google" {
		params.Set("response_type", "code")
	} else {
		params.Set("response_type", "code")
	}

	http.Redirect(w, r, cfg.AuthURL+"?"+params.Encode(), http.StatusTemporaryRedirect)
}

func handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	log.Printf("OAuth callback: provider=%s", provider)

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		log.Printf("OAuth callback: oauth_state cookie missing: %v", err)
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	if stateCookie.Value != r.URL.Query().Get("state") {
		log.Printf("OAuth callback: state mismatch: cookie=%s url=%s", stateCookie.Value, r.URL.Query().Get("state"))
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		log.Printf("OAuth callback: no code parameter in URL")
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	cfg, err := getOAuthConfig(provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	callbackURL := fmt.Sprintf("https://wordle-six.tomtom.fyi/auth/%s/callback", provider)

	// Exchange code for token
	tokenData := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {callbackURL},
		"grant_type":    {"authorization_code"},
	}

	req, _ := http.NewRequest("POST", cfg.TokenURL, strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("OAuth callback: token exchange HTTP error: %v", err)
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var tokenResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Printf("OAuth callback: failed to decode token response (status %d): %v", resp.StatusCode, err)
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("OAuth callback: token endpoint returned %d: %v", resp.StatusCode, tokenResp)
	}

	accessToken, _ := tokenResp["access_token"].(string)
	if accessToken == "" {
		log.Printf("OAuth callback: no access_token in response: %v", tokenResp)
		http.Error(w, "No access token received", http.StatusInternalServerError)
		return
	}

	// Fetch user info
	userReq, _ := http.NewRequest("GET", cfg.UserInfoURL, nil)
	userReq.Header.Set("Authorization", "Bearer "+accessToken)
	userReq.Header.Set("Accept", "application/json")

	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		log.Printf("OAuth callback: user info fetch error: %v", err)
		http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
		return
	}
	if userResp.StatusCode != http.StatusOK {
		log.Printf("OAuth callback: user info endpoint returned %d", userResp.StatusCode)
	}
	defer userResp.Body.Close()

	body, _ := io.ReadAll(userResp.Body)
	var userInfo map[string]interface{}
	json.Unmarshal(body, &userInfo)

	// Extract user details based on provider
	var providerID, displayName, avatarURL string
	switch provider {
	case "github":
		providerID = fmt.Sprintf("%.0f", userInfo["id"].(float64))
		displayName, _ = userInfo["login"].(string)
		avatarURL, _ = userInfo["avatar_url"].(string)
	case "discord":
		providerID, _ = userInfo["id"].(string)
		displayName, _ = userInfo["username"].(string)
		avatar, _ := userInfo["avatar"].(string)
		if avatar != "" {
			avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", providerID, avatar)
		}
	case "google":
		providerID, _ = userInfo["id"].(string)
		displayName, _ = userInfo["name"].(string)
		avatarURL, _ = userInfo["picture"].(string)
	}

	log.Printf("OAuth callback: provider=%s user=%s id=%s", provider, displayName, providerID)

	user, err := upsertUser(provider, providerID, displayName, avatarURL)
	if err != nil {
		log.Printf("OAuth callback: upsertUser failed: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Create JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("OAuth callback: JWT signing failed: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	log.Printf("OAuth callback: success, setting session cookie for user %d (%s)", user.ID, displayName)

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    tokenString,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear oauth state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func handleAuthMe(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"user":null}`))
		return
	}

	resp := map[string]interface{}{
		"id":           user.ID,
		"provider":     user.Provider,
		"display_name": user.DisplayName,
		"avatar_url":   user.AvatarURL,
		"is_new":       user.IsNew,
		"banned":       user.Banned,
	}
	if user.CustomName != nil {
		resp["display_name"] = *user.CustomName
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"user": resp})
}

func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusOK)
}

func getUserFromRequest(r *http.Request) *User {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		log.Printf("Session: invalid JWT: %v", err)
		return nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil
	}

	userID, ok := claims["sub"].(float64)
	if !ok {
		return nil
	}

	user, err := getUserByID(int64(userID))
	if err != nil {
		return nil
	}

	return user
}
