package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- computeStreak ---

func TestComputeStreak_AllWins(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "streaker")

	for i, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13"} {
		guesses := i + 2
		insertGameResult(user.ID, date, true, &guesses, false)
	}

	streak := computeStreak(user.ID)
	if streak != 3 {
		t.Errorf("expected streak=3, got %d", streak)
	}
}

func TestComputeStreak_BrokenByLoss(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")

	g3 := 3
	insertGameResult(user.ID, "2024-01-15", true, &g3, false)  // most recent: win
	insertGameResult(user.ID, "2024-01-14", true, &g3, false)  // win
	insertGameResult(user.ID, "2024-01-13", false, nil, false)  // loss breaks streak
	insertGameResult(user.ID, "2024-01-12", true, &g3, false)  // older win

	streak := computeStreak(user.ID)
	if streak != 2 {
		t.Errorf("expected streak=2, got %d", streak)
	}
}

func TestComputeStreak_NoResults(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "newbie")

	streak := computeStreak(user.ID)
	if streak != 0 {
		t.Errorf("expected streak=0, got %d", streak)
	}
}

func TestComputeStreak_AllLosses(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "loser")

	insertGameResult(user.ID, "2024-01-15", false, nil, false)
	insertGameResult(user.ID, "2024-01-14", false, nil, false)

	streak := computeStreak(user.ID)
	if streak != 0 {
		t.Errorf("expected streak=0, got %d", streak)
	}
}

// --- handleSubmitResult ---

func TestSubmitResult_BasicWin(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	guesses := 4
	body := map[string]interface{}{
		"date":      "2024-01-15",
		"won":       true,
		"guesses":   guesses,
		"hard_mode": false,
	}
	req := jsonRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	handleSubmitResult(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
}

func TestSubmitResult_Loss(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"date":      "2024-01-15",
		"won":       false,
		"hard_mode": false,
	}
	req := jsonRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	handleSubmitResult(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify it was stored
	var won bool
	db.QueryRow("SELECT won FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&won)
	if won {
		t.Error("expected won=false")
	}
}

func TestSubmitResult_HardMode(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	guesses := 3
	body := map[string]interface{}{
		"date":      "2024-01-15",
		"won":       true,
		"guesses":   guesses,
		"hard_mode": true,
	}
	req := jsonRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	handleSubmitResult(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var hardMode bool
	db.QueryRow("SELECT hard_mode FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&hardMode)
	if !hardMode {
		t.Error("expected hard_mode=true")
	}
}

func TestSubmitResult_DuplicateDate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	guesses := 4
	body := map[string]interface{}{
		"date":      "2024-01-15",
		"won":       true,
		"guesses":   guesses,
		"hard_mode": false,
	}
	req := jsonRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	handleSubmitResult(rr, req)

	// Submit again — should succeed (ON CONFLICT DO NOTHING)
	guesses2 := 2
	body["guesses"] = guesses2
	req = jsonRequest(t, "POST", "/api/result", body, cookie)
	rr = httptest.NewRecorder()
	handleSubmitResult(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for duplicate, got %d", rr.Code)
	}

	// Original value preserved
	var g int
	db.QueryRow("SELECT guesses FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&g)
	if g != 4 {
		t.Errorf("expected original guesses=4, got %d", g)
	}
}

func TestSubmitResult_MissingDate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"won":     true,
		"guesses": 3,
	}
	req := jsonRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	handleSubmitResult(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing date, got %d", rr.Code)
	}
}

func TestSubmitResult_InvalidGuessCount(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "player")
	cookie := makeAuthCookie(t, user.ID)

	guesses := 7
	body := map[string]interface{}{
		"date":    "2024-01-15",
		"won":     true,
		"guesses": guesses,
	}
	req := jsonRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	handleSubmitResult(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid guess count, got %d", rr.Code)
	}
}

func TestSubmitResult_NotAuthenticated(t *testing.T) {
	setupTestDB(t)

	body := map[string]interface{}{
		"date":    "2024-01-15",
		"won":     true,
		"guesses": 3,
	}
	req := jsonRequest(t, "POST", "/api/result", body, nil)
	rr := httptest.NewRecorder()
	handleSubmitResult(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// --- handleGetLeaderboard ---

func TestLeaderboard_Empty(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	handleGetLeaderboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})
	if len(entries) != 0 {
		t.Errorf("expected empty leaderboard, got %d entries", len(entries))
	}
}

func TestLeaderboard_SinglePlayer(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "github", "1", "solo")

	g := 3
	insertGameResult(user.ID, "2024-01-15", true, &g, false)

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	handleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0].(map[string]interface{})
	if entry["display_name"] != "solo" {
		t.Errorf("expected display_name solo, got %v", entry["display_name"])
	}
	if entry["rank"] != float64(1) {
		t.Errorf("expected rank 1, got %v", entry["rank"])
	}
}

func TestLeaderboard_MultiPlayerRanking(t *testing.T) {
	setupTestDB(t)
	good := createTestUser(t, "github", "1", "GoodPlayer")
	bad := createTestUser(t, "github", "2", "BadPlayer")

	// GoodPlayer wins in 2 guesses on multiple days
	for _, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13", "2024-01-12", "2024-01-11"} {
		g := 2
		insertGameResult(good.ID, date, true, &g, false)
	}

	// BadPlayer wins in 6 guesses
	for _, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13", "2024-01-12", "2024-01-11"} {
		g := 6
		insertGameResult(bad.ID, date, true, &g, false)
	}

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	handleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// GoodPlayer (avg 2) should rank above BadPlayer (avg 6)
	first := entries[0].(map[string]interface{})
	if first["display_name"] != "GoodPlayer" {
		t.Errorf("expected GoodPlayer first, got %v", first["display_name"])
	}
}

func TestLeaderboard_BannedExcluded(t *testing.T) {
	setupTestDB(t)
	good := createTestUser(t, "github", "1", "GoodPlayer")
	banned := createTestUser(t, "github", "2", "BannedPlayer")
	banUser(banned.ID)

	g := 3
	insertGameResult(good.ID, "2024-01-15", true, &g, false)
	insertGameResult(banned.ID, "2024-01-15", true, &g, false)

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	handleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (banned excluded), got %d", len(entries))
	}

	entry := entries[0].(map[string]interface{})
	if entry["display_name"] != "GoodPlayer" {
		t.Errorf("expected GoodPlayer, got %v", entry["display_name"])
	}
}

func TestLeaderboard_LimitParam(t *testing.T) {
	setupTestDB(t)

	// Create 5 players with results
	for i := range 5 {
		user := createTestUser(t, "github", string(rune('A'+i)), "Player"+string(rune('A'+i)))
		g := 3
		insertGameResult(user.ID, "2024-01-15", true, &g, false)
	}

	req := httptest.NewRequest("GET", "/api/leaderboard?limit=2", nil)
	rr := httptest.NewRecorder()
	handleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with limit=2, got %d", len(entries))
	}
}

func TestLeaderboard_HardModeBonus(t *testing.T) {
	setupTestDB(t)
	hardPlayer := createTestUser(t, "github", "1", "HardMode")
	normalPlayer := createTestUser(t, "github", "2", "Normal")

	// Both win in 4 guesses, but hard mode gets 10% bonus
	for _, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13", "2024-01-12", "2024-01-11"} {
		g := 4
		insertGameResult(hardPlayer.ID, date, true, &g, true)  // hard mode
		insertGameResult(normalPlayer.ID, date, true, &g, false) // normal
	}

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	handleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})

	// Hard mode player should rank higher (lower weighted avg: 4*0.9=3.6 vs 4.0)
	first := entries[0].(map[string]interface{})
	if first["display_name"] != "HardMode" {
		t.Errorf("expected HardMode player first, got %v", first["display_name"])
	}
}

func TestLeaderboard_LossPenalty(t *testing.T) {
	setupTestDB(t)
	winner := createTestUser(t, "github", "1", "Winner")
	loser := createTestUser(t, "github", "2", "Loser")

	g := 4
	insertGameResult(winner.ID, "2024-01-15", true, &g, false)
	insertGameResult(loser.ID, "2024-01-15", false, nil, false) // loss counts as 8.0

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	handleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})

	first := entries[0].(map[string]interface{})
	if first["display_name"] != "Winner" {
		t.Errorf("expected Winner first, got %v", first["display_name"])
	}
}
