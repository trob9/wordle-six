package auth

import (
	"encoding/json"
	"net/http"
	"os"

	pkgauth "github.com/trob9/pkg/auth"
	"wordle-six/internal/middleware"
	"wordle-six/internal/netutil"
	"wordle-six/internal/store"
)

// Handler is the shared auth handler, initialised by Init.
var Handler *pkgauth.Handler

// Init wires up OAuth providers and must be called once after store.InitDB.
func Init() {
	Handler = pkgauth.New(pkgauth.Config{
		JWTSecret:   os.Getenv("JWT_SECRET"),
		CallbackURL: os.Getenv("OAUTH_CALLBACK_URL"),
		Providers: []pkgauth.Provider{
			pkgauth.GitHub(os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET")),
			pkgauth.Discord(os.Getenv("DISCORD_CLIENT_ID"), os.Getenv("DISCORD_CLIENT_SECRET")),
			pkgauth.Google(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET")),
		},
		OnLogin: func(r *http.Request, id pkgauth.Identity) (int64, error) {
			user, err := store.UpsertUser(id.Provider, id.ProviderID, id.DisplayName, id.AvatarURL)
			if err != nil {
				return 0, err
			}
			return user.ID, nil
		},
	})
}

// GetUserFromRequest extracts the session JWT and returns the store.User,
// or nil if unauthenticated. Used by game, middleware, and admin handlers.
func GetUserFromRequest(r *http.Request) *store.User {
	userID, ok := Handler.GetUserID(r)
	if !ok {
		return nil
	}
	user, _ := store.GetUserByID(userID)
	return user
}

// ValidateAvatarURL is re-exported so tests that import this package still compile.
var ValidateAvatarURL = pkgauth.ValidateAvatarURL

// OAuthHTTPClient is re-exported for tests that substitute a mock client.
var OAuthHTTPClient = pkgauth.OAuthHTTPClient

// TestJWTSecret returns the JWT signing key for tests.
func TestJWTSecret() []byte {
	return Handler.TestJWTSecret()
}

// HandleAuthStart, HandleAuthCallback, HandleAuthLogout are the HTTP handlers
// registered in main.go. HandleAuthMe lives here because it needs store access.
func HandleAuthStart(w http.ResponseWriter, r *http.Request) {
	Handler.Start(w, r)
}

func HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	Handler.Callback(w, r)
}

func HandleAuthMe(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromRequest(r)
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

func HandleAuthLogout(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromRequest(r)
	if user != nil {
		middleware.LogAuthEvent("auth_logout", user.Provider, "success", netutil.ClientIP(r), map[string]interface{}{
			"user_id": user.ID,
		})
	}
	Handler.Logout(w, r)
}
