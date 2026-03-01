// Package testutil provides shared test helpers for wordle-six internal packages.
// It is a regular (non-_test) package so it can be imported by multiple test packages.
package testutil

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"wordle-six/internal/auth"
	"wordle-six/internal/store"
)

// SetupTestDB creates an in-memory SQLite database and assigns it to store.DB.
func SetupTestDB(t *testing.T) {
	t.Helper()

	var err error
	store.DB, err = sql.Open("sqlite3", ":memory:?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := store.CreateTables(); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}
	if err := store.RunMigrations(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	if err := store.CreateViews(); err != nil {
		t.Fatalf("failed to create views: %v", err)
	}

	t.Cleanup(func() {
		store.DB.Close()
	})
}

// CreateTestUser creates a user via store.UpsertUser and returns it.
func CreateTestUser(t *testing.T, provider, providerID, displayName string) *store.User {
	t.Helper()
	user, err := store.UpsertUser(provider, providerID, displayName, "")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// MakeAuthCookie creates a valid JWT session cookie for the given user ID.
func MakeAuthCookie(t *testing.T, userID int64) *http.Cookie {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(auth.TestJWTSecret())
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}
	return &http.Cookie{
		Name:  "session",
		Value: tokenString,
	}
}

// JSONRequest builds an *http.Request with a JSON body and optional session cookie.
func JSONRequest(t *testing.T, method, url string, body interface{}, cookie *http.Cookie) *http.Request {
	t.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, url, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return req
}
