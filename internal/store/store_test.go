package store

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	var err error
	DB, err = sql.Open("sqlite3", ":memory:?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := CreateTables(); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}
	if err := RunMigrations(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	if err := CreateViews(); err != nil {
		t.Fatalf("failed to create views: %v", err)
	}

	t.Cleanup(func() {
		DB.Close()
	})
}

func createTestUser(t *testing.T, provider, providerID, displayName string) *User {
	t.Helper()
	user, err := UpsertUser(provider, providerID, displayName, "")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func TestCreateTables(t *testing.T) {
	setupTestDB(t)

	tables := []string{"users", "game_results", "user_stats", "game_progress"}
	for _, table := range tables {
		var name string
		err := DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	setupTestDB(t)

	if err := RunMigrations(); err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}
	if err := RunMigrations(); err != nil {
		t.Fatalf("third migration run failed: %v", err)
	}
}

func TestUpsertUser_New(t *testing.T) {
	setupTestDB(t)

	user, err := UpsertUser("github", "12345", "testuser", "https://avatars.githubusercontent.com/u/12345")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
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

	user1, _ := UpsertUser("github", "12345", "oldname", "")
	user2, _ := UpsertUser("github", "12345", "newname", "https://avatars.githubusercontent.com/u/12345")

	if user1.ID != user2.ID {
		t.Errorf("expected same user ID, got %d and %d", user1.ID, user2.ID)
	}

	fetched, err := GetUserByID(user1.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if fetched.DisplayName != "newname" {
		t.Errorf("expected updated display_name newname, got %s", fetched.DisplayName)
	}
}

func TestGetUserByID(t *testing.T) {
	setupTestDB(t)

	created := createTestUser(t, "discord", "999", "discorduser")
	fetched, err := GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
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

	_, err := GetUserByID(99999)
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestBanUnban(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t, "github", "1", "banme")

	if err := BanUser(user.ID); err != nil {
		t.Fatalf("BanUser failed: %v", err)
	}
	fetched, _ := GetUserByID(user.ID)
	if !fetched.Banned {
		t.Error("expected user to be banned")
	}

	if err := UnbanUser(user.ID); err != nil {
		t.Fatalf("UnbanUser failed: %v", err)
	}
	fetched, _ = GetUserByID(user.ID)
	if fetched.Banned {
		t.Error("expected user to be unbanned")
	}
}

func TestAdminBootstrap(t *testing.T) {
	setupTestDB(t)

	createTestUser(t, "github", "first", "admin")
	RunMigrations()

	admin, _ := GetUserByID(1)
	if !admin.IsAdmin {
		t.Error("expected user ID 1 to be admin")
	}

	user2 := createTestUser(t, "github", "second", "regular")
	RunMigrations()
	regular, _ := GetUserByID(user2.ID)
	if regular.IsAdmin {
		t.Error("expected user ID 2 to not be admin")
	}
}

func TestInsertGameResult(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	guesses := 4
	err := InsertGameResult(user.ID, "2024-01-15", true, &guesses, false)
	if err != nil {
		t.Fatalf("InsertGameResult failed: %v", err)
	}

	var won bool
	var g int
	err = DB.QueryRow("SELECT won, guesses FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&won, &g)
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
	InsertGameResult(user.ID, "2024-01-15", true, &guesses, false)

	guesses2 := 2
	err := InsertGameResult(user.ID, "2024-01-15", true, &guesses2, false)
	if err != nil {
		t.Fatalf("duplicate InsertGameResult should not error: %v", err)
	}

	var g int
	DB.QueryRow("SELECT guesses FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&g)
	if g != 4 {
		t.Errorf("expected original guesses=4 preserved, got %d", g)
	}
}

func TestUpdateCustomName(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	err := UpdateCustomName(user.ID, "CoolName")
	if err != nil {
		t.Fatalf("UpdateCustomName failed: %v", err)
	}

	fetched, _ := GetUserByID(user.ID)
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
	user2, _ := UpsertUser("github", "same-id", "user2", "")

	var count int
	DB.QueryRow("SELECT COUNT(*) FROM users WHERE provider = 'github' AND provider_id = 'same-id'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 user row, got %d", count)
	}
	_ = user2
}

func TestUniqueConstraint_GameResults(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	guesses := 3
	InsertGameResult(user.ID, "2024-01-15", true, &guesses, false)
	InsertGameResult(user.ID, "2024-01-15", true, &guesses, false)

	var count int
	DB.QueryRow("SELECT COUNT(*) FROM game_results WHERE user_id = ? AND date = '2024-01-15'", user.ID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 game result row, got %d", count)
	}
}
