package main

import (
	"testing"
)

func TestCreateTables(t *testing.T) {
	setupTestDB(t)

	// Verify all tables exist by querying them
	tables := []string{"users", "game_results", "user_stats", "game_progress", "tz_events"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	setupTestDB(t)

	// Running migrations again should not error
	if err := runMigrations(); err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}
	if err := runMigrations(); err != nil {
		t.Fatalf("third migration run failed: %v", err)
	}
}

func TestUpsertUser_New(t *testing.T) {
	setupTestDB(t)

	user, err := upsertUser("github", "12345", "testuser", "https://avatars.githubusercontent.com/u/12345")
	if err != nil {
		t.Fatalf("upsertUser failed: %v", err)
	}

	if user.ID == 0 {
		t.Error("expected non-zero user ID")
	}
	if user.Provider != "github" {
		t.Errorf("expected provider github, got %s", user.Provider)
	}
	if user.ProviderID != "12345" {
		t.Errorf("expected provider_id 12345, got %s", user.ProviderID)
	}
	if user.DisplayName != "testuser" {
		t.Errorf("expected display_name testuser, got %s", user.DisplayName)
	}
}

func TestUpsertUser_Existing(t *testing.T) {
	setupTestDB(t)

	user1, _ := upsertUser("github", "12345", "oldname", "")
	user2, _ := upsertUser("github", "12345", "newname", "https://avatars.githubusercontent.com/u/12345")

	if user1.ID != user2.ID {
		t.Errorf("expected same user ID, got %d and %d", user1.ID, user2.ID)
	}

	// Verify display_name was updated
	fetched, err := getUserByID(user1.ID)
	if err != nil {
		t.Fatalf("getUserByID failed: %v", err)
	}
	if fetched.DisplayName != "newname" {
		t.Errorf("expected updated display_name newname, got %s", fetched.DisplayName)
	}
}

func TestGetUserByID(t *testing.T) {
	setupTestDB(t)

	created := createTestUser(t, "discord", "999", "discorduser")
	fetched, err := getUserByID(created.ID)
	if err != nil {
		t.Fatalf("getUserByID failed: %v", err)
	}

	if fetched.Provider != "discord" {
		t.Errorf("expected provider discord, got %s", fetched.Provider)
	}
	if fetched.DisplayName != "discorduser" {
		t.Errorf("expected display_name discorduser, got %s", fetched.DisplayName)
	}
	if !fetched.IsNew {
		t.Error("expected IsNew=true for user without custom_name")
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	setupTestDB(t)

	_, err := getUserByID(99999)
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestBanUnban(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t, "github", "1", "banme")

	if err := banUser(user.ID); err != nil {
		t.Fatalf("banUser failed: %v", err)
	}
	fetched, _ := getUserByID(user.ID)
	if !fetched.Banned {
		t.Error("expected user to be banned")
	}

	if err := unbanUser(user.ID); err != nil {
		t.Fatalf("unbanUser failed: %v", err)
	}
	fetched, _ = getUserByID(user.ID)
	if fetched.Banned {
		t.Error("expected user to be unbanned")
	}
}

func TestAdminBootstrap(t *testing.T) {
	setupTestDB(t)

	// First user (ID 1) should get admin after migration
	createTestUser(t, "github", "first", "admin")
	runMigrations()

	admin, _ := getUserByID(1)
	if !admin.IsAdmin {
		t.Error("expected user ID 1 to be admin")
	}

	// Second user should not be admin
	user2 := createTestUser(t, "github", "second", "regular")
	runMigrations()
	regular, _ := getUserByID(user2.ID)
	if regular.IsAdmin {
		t.Error("expected user ID 2 to not be admin")
	}
}

func TestInsertGameResult(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	guesses := 4
	err := insertGameResult(user.ID, "2024-01-15", true, &guesses, false)
	if err != nil {
		t.Fatalf("insertGameResult failed: %v", err)
	}

	// Verify it was inserted
	var won bool
	var g int
	err = db.QueryRow("SELECT won, guesses FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&won, &g)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if !won || g != 4 {
		t.Errorf("expected won=true guesses=4, got won=%v guesses=%d", won, g)
	}
}

func TestInsertGameResult_Duplicate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	guesses := 4
	insertGameResult(user.ID, "2024-01-15", true, &guesses, false)

	// Duplicate insert should silently succeed (ON CONFLICT DO NOTHING)
	guesses2 := 2
	err := insertGameResult(user.ID, "2024-01-15", true, &guesses2, false)
	if err != nil {
		t.Fatalf("duplicate insertGameResult should not error: %v", err)
	}

	// Original value should be preserved
	var g int
	db.QueryRow("SELECT guesses FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&g)
	if g != 4 {
		t.Errorf("expected original guesses=4 preserved, got %d", g)
	}
}

func TestUpdateCustomName(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	err := updateCustomName(user.ID, "CoolName")
	if err != nil {
		t.Fatalf("updateCustomName failed: %v", err)
	}

	fetched, _ := getUserByID(user.ID)
	if fetched.CustomName == nil || *fetched.CustomName != "CoolName" {
		t.Errorf("expected custom_name CoolName, got %v", fetched.CustomName)
	}
	if fetched.IsNew {
		t.Error("expected IsNew=false after setting custom_name")
	}
}

func TestUniqueConstraint_ProviderProviderID(t *testing.T) {
	setupTestDB(t)

	createTestUser(t, "github", "same-id", "user1")
	// Same provider+provider_id should upsert, not create duplicate
	user2, _ := upsertUser("github", "same-id", "user2", "")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE provider = 'github' AND provider_id = 'same-id'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 user row, got %d", count)
	}
	_ = user2
}

func TestUniqueConstraint_GameResults(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	guesses := 3
	insertGameResult(user.ID, "2024-01-15", true, &guesses, false)
	insertGameResult(user.ID, "2024-01-15", true, &guesses, false)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM game_results WHERE user_id = ? AND date = '2024-01-15'", user.ID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 game result row, got %d", count)
	}
}
