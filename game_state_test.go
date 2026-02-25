package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Save/Load Progress ---

func TestSaveAndLoadProgress(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	// Save progress
	body := map[string]interface{}{
		"date":    "2024-01-15",
		"guesses": []string{"BRIDGE", "SNATCH"},
		"hardMode": false,
		"gameOver": false,
		"won":     false,
	}
	req := jsonRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	handleSaveProgress(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("save progress: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Load progress
	req = httptest.NewRequest("GET", "/api/game-state?date=2024-01-15", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	handleGetGameState(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("get game state: expected 200, got %d", rr.Code)
	}

	var progress GameProgress
	json.NewDecoder(rr.Body).Decode(&progress)
	if len(progress.Guesses) != 2 {
		t.Errorf("expected 2 guesses, got %d", len(progress.Guesses))
	}
	if progress.Guesses[0] != "BRIDGE" {
		t.Errorf("expected first guess BRIDGE, got %s", progress.Guesses[0])
	}
}

func TestSaveProgress_Upsert(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	// First save
	body := map[string]interface{}{
		"date":    "2024-01-15",
		"guesses": []string{"BRIDGE"},
		"hardMode": false,
		"gameOver": false,
		"won":     false,
	}
	req := jsonRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	handleSaveProgress(rr, req)

	// Second save with more guesses
	body["guesses"] = []string{"BRIDGE", "SNATCH"}
	req = jsonRequest(t, "POST", "/api/save-progress", body, cookie)
	rr = httptest.NewRecorder()
	handleSaveProgress(rr, req)

	// Load and verify
	req = httptest.NewRequest("GET", "/api/game-state?date=2024-01-15", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	handleGetGameState(rr, req)

	var progress GameProgress
	json.NewDecoder(rr.Body).Decode(&progress)
	if len(progress.Guesses) != 2 {
		t.Errorf("expected 2 guesses after upsert, got %d", len(progress.Guesses))
	}
}

func TestSaveProgress_GameOverAutoInsertsResult(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"date":     "2024-01-15",
		"guesses":  []string{"BRIDGE", "SNATCH", "WINNER"},
		"hardMode": true,
		"gameOver": true,
		"won":      true,
	}
	req := jsonRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	handleSaveProgress(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify game_results was auto-inserted
	var won bool
	var guesses int
	var hardMode bool
	err := db.QueryRow("SELECT won, guesses, hard_mode FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&won, &guesses, &hardMode)
	if err != nil {
		t.Fatalf("expected game_results row: %v", err)
	}
	if !won || guesses != 3 || !hardMode {
		t.Errorf("expected won=true guesses=3 hard_mode=true, got won=%v guesses=%d hard_mode=%v", won, guesses, hardMode)
	}
}

func TestSaveProgress_TooManyGuesses(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"date":    "2024-01-15",
		"guesses": []string{"A", "B", "C", "D", "E", "F", "G"},
		"hardMode": false,
		"gameOver": false,
		"won":     false,
	}
	req := jsonRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	handleSaveProgress(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for too many guesses, got %d", rr.Code)
	}
}

func TestSaveProgress_MissingDate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"guesses": []string{"BRIDGE"},
		"hardMode": false,
		"gameOver": false,
		"won":     false,
	}
	req := jsonRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	handleSaveProgress(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing date, got %d", rr.Code)
	}
}

func TestSaveProgress_NotAuthenticated(t *testing.T) {
	setupTestDB(t)

	body := map[string]interface{}{
		"date":    "2024-01-15",
		"guesses": []string{"BRIDGE"},
	}
	req := jsonRequest(t, "POST", "/api/save-progress", body, nil)
	rr := httptest.NewRecorder()
	handleSaveProgress(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetGameState_NoProgress(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	req := httptest.NewRequest("GET", "/api/game-state?date=2024-01-15", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleGetGameState(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var progress GameProgress
	json.NewDecoder(rr.Body).Decode(&progress)
	if len(progress.Guesses) != 0 {
		t.Errorf("expected empty guesses, got %d", len(progress.Guesses))
	}
}

func TestGetGameState_MissingDate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	req := httptest.NewRequest("GET", "/api/game-state", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleGetGameState(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing date, got %d", rr.Code)
	}
}

// --- Stats ---

func TestSaveAndLoadStats(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	// Insert 5 consecutive wins so streak_view returns 5.
	for i, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13", "2024-01-12", "2024-01-11"} {
		g := i + 2
		insertGameResult(user.ID, date, true, &g, false)
	}

	body := UserStats{
		Played:       10,
		Won:          8,
		PlayedHard:   3,
		WonHard:      2,
		Distribution: []int{0, 1, 2, 3, 1, 1},
		LastDate:     "2024-01-15",
		HardMode:     true,
	}
	req := jsonRequest(t, "POST", "/api/user-stats", body, cookie)
	rr := httptest.NewRecorder()
	handleSaveUserStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("save stats: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Load stats
	req = httptest.NewRequest("GET", "/api/user-stats", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	handleGetUserStats(rr, req)

	var stats UserStats
	json.NewDecoder(rr.Body).Decode(&stats)
	if stats.Played != 10 {
		t.Errorf("expected played=10, got %d", stats.Played)
	}
	if stats.Won != 8 {
		t.Errorf("expected won=8, got %d", stats.Won)
	}
	if stats.CurrentStreak != 5 {
		t.Errorf("expected streak=5, got %d", stats.CurrentStreak)
	}
	if stats.MaxStreak != 5 {
		t.Errorf("expected max_streak=5, got %d", stats.MaxStreak)
	}
	if !stats.HardMode {
		t.Error("expected hardMode=true")
	}
}

func TestGetUserStats_Default(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	req := httptest.NewRequest("GET", "/api/user-stats", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleGetUserStats(rr, req)

	var stats UserStats
	json.NewDecoder(rr.Body).Decode(&stats)
	if stats.Played != 0 {
		t.Errorf("expected default played=0, got %d", stats.Played)
	}
	if len(stats.Distribution) != 6 {
		t.Errorf("expected 6-element distribution, got %d", len(stats.Distribution))
	}
}

// --- Display Name ---

func TestUpdateDisplayName_Valid(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]string{"name": "CoolPlayer"}
	req := jsonRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	handleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	fetched, _ := getUserByID(user.ID)
	if fetched.CustomName == nil || *fetched.CustomName != "CoolPlayer" {
		t.Errorf("expected custom_name CoolPlayer, got %v", fetched.CustomName)
	}
}

func TestUpdateDisplayName_TooLong(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]string{"name": "ThisNameIsWayTooLongForTheLimit"}
	req := jsonRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	handleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for too long name, got %d", rr.Code)
	}
}

func TestUpdateDisplayName_Empty(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]string{"name": ""}
	req := jsonRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	handleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", rr.Code)
	}
}

func TestUpdateDisplayName_InvalidChars(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]string{"name": "player<script>"}
	req := jsonRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	handleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid chars, got %d", rr.Code)
	}
}

func TestUpdateDisplayName_BannedUser(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	banUser(user.ID)
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]string{"name": "NewName"}
	req := jsonRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	handleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for banned user, got %d", rr.Code)
	}
}

// --- sanitizeName ---

func TestSanitizeName_TrimsWhitespace(t *testing.T) {
	if got := sanitizeName("  hello  "); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestSanitizeName_CollapsesSpaces(t *testing.T) {
	if got := sanitizeName("hello   world"); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestSanitizeName_EmptyInput(t *testing.T) {
	if got := sanitizeName("   "); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// --- Admin: ban/unban ---

func TestHandleBanUser_AdminCanBan(t *testing.T) {
	setupTestDB(t)
	admin := createTestUser(t, "github", "admin1", "admin")
	runMigrations() // makes ID 1 admin
	target := createTestUser(t, "github", "target1", "target")
	cookie := makeAuthCookie(t, admin.ID)

	body := map[string]interface{}{"user_id": target.ID, "ban": true}
	req := jsonRequest(t, "POST", "/api/admin/ban", body, cookie)
	rr := httptest.NewRecorder()
	handleBanUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	fetched, _ := getUserByID(target.ID)
	if !fetched.Banned {
		t.Error("expected target to be banned")
	}
}

func TestHandleBanUser_NonAdminForbidden(t *testing.T) {
	setupTestDB(t)
	createTestUser(t, "github", "admin1", "admin") // ID 1 = admin
	nonAdmin := createTestUser(t, "github", "regular1", "regular")
	cookie := makeAuthCookie(t, nonAdmin.ID)

	body := map[string]interface{}{"user_id": 1, "ban": true}
	req := jsonRequest(t, "POST", "/api/admin/ban", body, cookie)
	rr := httptest.NewRecorder()
	handleBanUser(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestHandleBanUser_CannotBanSelf(t *testing.T) {
	setupTestDB(t)
	admin := createTestUser(t, "github", "admin1", "admin")
	runMigrations()
	cookie := makeAuthCookie(t, admin.ID)

	body := map[string]interface{}{"user_id": admin.ID, "ban": true}
	req := jsonRequest(t, "POST", "/api/admin/ban", body, cookie)
	rr := httptest.NewRecorder()
	handleBanUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for self-ban, got %d", rr.Code)
	}
}

func TestHandleListUsers_AdminOnly(t *testing.T) {
	setupTestDB(t)
	admin := createTestUser(t, "github", "admin1", "admin")
	runMigrations()
	createTestUser(t, "github", "user2", "user2")
	cookie := makeAuthCookie(t, admin.ID)

	req := httptest.NewRequest("GET", "/api/admin/users", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleListUsers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	users := resp["users"].([]interface{})
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestHandleListUsers_NonAdminForbidden(t *testing.T) {
	setupTestDB(t)
	createTestUser(t, "github", "admin1", "admin")
	nonAdmin := createTestUser(t, "github", "regular1", "regular")
	cookie := makeAuthCookie(t, nonAdmin.ID)

	req := httptest.NewRequest("GET", "/api/admin/users", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handleListUsers(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
