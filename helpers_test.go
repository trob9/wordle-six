package main

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
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) {
	t.Helper()

	var err error
	db, err = sql.Open("sqlite3", ":memory:?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := createTables(); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}
	if err := runMigrations(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	if err := createViews(); err != nil {
		t.Fatalf("failed to create views: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})
}

// createTestUser creates a user via upsertUser and returns the user.
func createTestUser(t *testing.T, provider, providerID, displayName string) *User {
	t.Helper()
	user, err := upsertUser(provider, providerID, displayName, "")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// makeAuthCookie creates a valid JWT session cookie for the given user ID.
func makeAuthCookie(t *testing.T, userID int64) *http.Cookie {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}
	return &http.Cookie{
		Name:  "session",
		Value: tokenString,
	}
}

// jsonRequest builds an *http.Request with a JSON body and optional session cookie.
func jsonRequest(t *testing.T, method, url string, body interface{}, cookie *http.Cookie) *http.Request {
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
