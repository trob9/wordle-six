package main

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() error {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/data/wordle-six.db"
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}

	var err error
	db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return err
	}

	if err := createTables(); err != nil {
		return err
	}
	return runMigrations()
}

func createTables() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			provider TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			custom_name TEXT,
			avatar_url TEXT,
			banned BOOLEAN NOT NULL DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider, provider_id)
		);

		CREATE TABLE IF NOT EXISTS game_results (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			date TEXT NOT NULL,
			won BOOLEAN NOT NULL,
			guesses INTEGER,
			hard_mode BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, date)
		);

		CREATE TABLE IF NOT EXISTS user_stats (
			user_id INTEGER PRIMARY KEY REFERENCES users(id),
			played INTEGER NOT NULL DEFAULT 0,
			won INTEGER NOT NULL DEFAULT 0,
			played_hard INTEGER NOT NULL DEFAULT 0,
			won_hard INTEGER NOT NULL DEFAULT 0,
			current_streak INTEGER NOT NULL DEFAULT 0,
			max_streak INTEGER NOT NULL DEFAULT 0,
			distribution TEXT NOT NULL DEFAULT '[0,0,0,0,0,0]',
			last_date TEXT,
			hard_mode BOOLEAN NOT NULL DEFAULT FALSE
		);

		CREATE TABLE IF NOT EXISTS game_progress (
			user_id INTEGER NOT NULL REFERENCES users(id),
			date TEXT NOT NULL,
			guesses TEXT NOT NULL DEFAULT '[]',
			hard_mode BOOLEAN NOT NULL DEFAULT FALSE,
			game_over BOOLEAN NOT NULL DEFAULT FALSE,
			won BOOLEAN NOT NULL DEFAULT FALSE,
			UNIQUE(user_id, date)
		);
	`)
	return err
}

type User struct {
	ID          int64   `json:"id"`
	Provider    string  `json:"provider"`
	ProviderID  string  `json:"-"`
	DisplayName string  `json:"display_name"`
	CustomName  *string `json:"custom_name,omitempty"`
	AvatarURL   string  `json:"avatar_url,omitempty"`
	Banned      bool    `json:"-"`
	IsNew       bool    `json:"is_new,omitempty"`
}

func upsertUser(provider, providerID, displayName, avatarURL string) (*User, error) {
	result, err := db.Exec(`
		INSERT INTO users (provider, provider_id, display_name, avatar_url)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(provider, provider_id)
		DO UPDATE SET display_name = excluded.display_name, avatar_url = excluded.avatar_url
	`, provider, providerID, displayName, avatarURL)
	if err != nil {
		return nil, err
	}

	// Get the user ID (last insert or existing)
	id, err := result.LastInsertId()
	if err != nil || id == 0 {
		row := db.QueryRow("SELECT id FROM users WHERE provider = ? AND provider_id = ?", provider, providerID)
		if err := row.Scan(&id); err != nil {
			return nil, err
		}
	}

	return &User{
		ID:          id,
		Provider:    provider,
		ProviderID:  providerID,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}, nil
}

func getUserByID(id int64) (*User, error) {
	u := &User{}
	var customName *string
	err := db.QueryRow("SELECT id, provider, provider_id, display_name, custom_name, COALESCE(avatar_url, ''), banned FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Provider, &u.ProviderID, &u.DisplayName, &customName, &u.AvatarURL, &u.Banned)
	if err != nil {
		return nil, err
	}
	u.CustomName = customName
	u.IsNew = customName == nil
	return u, nil
}

func banUser(userID int64) error {
	_, err := db.Exec("UPDATE users SET banned = TRUE WHERE id = ?", userID)
	return err
}

func unbanUser(userID int64) error {
	_, err := db.Exec("UPDATE users SET banned = FALSE WHERE id = ?", userID)
	return err
}

func runMigrations() error {
	db.Exec("ALTER TABLE users ADD COLUMN custom_name TEXT")
	db.Exec("ALTER TABLE users ADD COLUMN banned BOOLEAN NOT NULL DEFAULT FALSE")
	db.Exec("ALTER TABLE user_stats ADD COLUMN played_hard INTEGER NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE user_stats ADD COLUMN won_hard INTEGER NOT NULL DEFAULT 0")
	return nil
}

func updateCustomName(userID int64, name string) error {
	_, err := db.Exec("UPDATE users SET custom_name = ? WHERE id = ?", name, userID)
	return err
}

func insertGameResult(userID int64, date string, won bool, guesses *int, hardMode bool) error {
	_, err := db.Exec(`
		INSERT INTO game_results (user_id, date, won, guesses, hard_mode)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, date) DO NOTHING
	`, userID, date, won, guesses, hardMode)
	return err
}
