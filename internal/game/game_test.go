package game_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wordle-six/internal/game"
	"wordle-six/internal/store"
	"wordle-six/internal/testutil"
)

// --- Save/Load Progress ---

func TestSaveAndLoadProgress(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"date":     "2024-01-15",
		"guesses":  []string{"BRIDGE", "SNATCH"},
		"hardMode": false,
		"gameOver": false,
		"won":      false,
	}
	req := testutil.JSONRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSaveProgress(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("save progress: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/game-state?date=2024-01-15", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	game.HandleGetGameState(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("get game state: expected 200, got %d", rr.Code)
	}

	var progress game.GameProgress
	json.NewDecoder(rr.Body).Decode(&progress)
	if len(progress.Guesses) != 2 {
		t.Errorf("expected 2 guesses, got %d", len(progress.Guesses))
	}
	if progress.Guesses[0] != "BRIDGE" {
		t.Errorf("expected first guess BRIDGE, got %s", progress.Guesses[0])
	}
}

func TestSaveProgress_Upsert(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"date":     "2024-01-15",
		"guesses":  []string{"BRIDGE"},
		"hardMode": false,
		"gameOver": false,
		"won":      false,
	}
	req := testutil.JSONRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSaveProgress(rr, req)

	body["guesses"] = []string{"BRIDGE", "SNATCH"}
	req = testutil.JSONRequest(t, "POST", "/api/save-progress", body, cookie)
	rr = httptest.NewRecorder()
	game.HandleSaveProgress(rr, req)

	req = httptest.NewRequest("GET", "/api/game-state?date=2024-01-15", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	game.HandleGetGameState(rr, req)

	var progress game.GameProgress
	json.NewDecoder(rr.Body).Decode(&progress)
	if len(progress.Guesses) != 2 {
		t.Errorf("expected 2 guesses after upsert, got %d", len(progress.Guesses))
	}
}

func TestSaveProgress_GameOverAutoInsertsResult(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"date":     "2024-01-15",
		"guesses":  []string{"BRIDGE", "SNATCH", "WINNER"},
		"hardMode": true,
		"gameOver": true,
		"won":      true,
	}
	req := testutil.JSONRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSaveProgress(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var won bool
	var guesses int
	var hardMode bool
	err := store.DB.QueryRow("SELECT won, guesses, hard_mode FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&won, &guesses, &hardMode)
	if err != nil {
		t.Fatalf("expected game_results row: %v", err)
	}
	if !won || guesses != 3 || !hardMode {
		t.Errorf("expected won=true guesses=3 hard_mode=true, got won=%v guesses=%d hard_mode=%v", won, guesses, hardMode)
	}
}

func TestSaveProgress_TooManyGuesses(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"date":     "2024-01-15",
		"guesses":  []string{"A", "B", "C", "D", "E", "F", "G"},
		"hardMode": false,
		"gameOver": false,
		"won":      false,
	}
	req := testutil.JSONRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSaveProgress(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for too many guesses, got %d", rr.Code)
	}
}

func TestSaveProgress_MissingDate(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"guesses":  []string{"BRIDGE"},
		"hardMode": false,
		"gameOver": false,
		"won":      false,
	}
	req := testutil.JSONRequest(t, "POST", "/api/save-progress", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSaveProgress(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing date, got %d", rr.Code)
	}
}

func TestSaveProgress_NotAuthenticated(t *testing.T) {
	testutil.SetupTestDB(t)

	body := map[string]interface{}{
		"date":    "2024-01-15",
		"guesses": []string{"BRIDGE"},
	}
	req := testutil.JSONRequest(t, "POST", "/api/save-progress", body, nil)
	rr := httptest.NewRecorder()
	game.HandleSaveProgress(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetGameState_NoProgress(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	req := httptest.NewRequest("GET", "/api/game-state?date=2024-01-15", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	game.HandleGetGameState(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var progress game.GameProgress
	json.NewDecoder(rr.Body).Decode(&progress)
	if len(progress.Guesses) != 0 {
		t.Errorf("expected empty guesses, got %d", len(progress.Guesses))
	}
}

func TestGetGameState_MissingDate(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	req := httptest.NewRequest("GET", "/api/game-state", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	game.HandleGetGameState(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing date, got %d", rr.Code)
	}
}

// --- Stats ---

func TestSaveAndLoadStats(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	for i, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13", "2024-01-12", "2024-01-11"} {
		g := i + 2
		store.InsertGameResult(user.ID, date, true, &g, false)
	}

	body := game.UserStats{
		Played:       10,
		Won:          8,
		PlayedHard:   3,
		WonHard:      2,
		Distribution: []int{0, 1, 2, 3, 1, 1},
		LastDate:     "2024-01-15",
		HardMode:     true,
	}
	req := testutil.JSONRequest(t, "POST", "/api/user-stats", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSaveUserStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("save stats: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/user-stats", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	game.HandleGetUserStats(rr, req)

	var stats game.UserStats
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
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	req := httptest.NewRequest("GET", "/api/user-stats", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	game.HandleGetUserStats(rr, req)

	var stats game.UserStats
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
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]string{"name": "CoolPlayer"}
	req := testutil.JSONRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	fetched, _ := store.GetUserByID(user.ID)
	if fetched.CustomName == nil || *fetched.CustomName != "CoolPlayer" {
		t.Errorf("expected custom_name CoolPlayer, got %v", fetched.CustomName)
	}
}

func TestUpdateDisplayName_TooLong(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]string{"name": "ThisNameIsWayTooLongForTheLimit"}
	req := testutil.JSONRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for too long name, got %d", rr.Code)
	}
}

func TestUpdateDisplayName_Empty(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]string{"name": ""}
	req := testutil.JSONRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", rr.Code)
	}
}

func TestUpdateDisplayName_InvalidChars(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]string{"name": "player<script>"}
	req := testutil.JSONRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid chars, got %d", rr.Code)
	}
}

func TestUpdateDisplayName_BannedUser(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	store.BanUser(user.ID)
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]string{"name": "NewName"}
	req := testutil.JSONRequest(t, "POST", "/api/display-name", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleUpdateDisplayName(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for banned user, got %d", rr.Code)
	}
}

// --- SanitizeName ---

func TestSanitizeName_TrimsWhitespace(t *testing.T) {
	if got := game.SanitizeName("  hello  "); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestSanitizeName_CollapsesSpaces(t *testing.T) {
	if got := game.SanitizeName("hello   world"); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestSanitizeName_EmptyInput(t *testing.T) {
	if got := game.SanitizeName("   "); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// --- Admin: ban/unban ---

func TestHandleBanUser_AdminCanBan(t *testing.T) {
	testutil.SetupTestDB(t)
	admin := testutil.CreateTestUser(t, "github", "admin1", "admin")
	store.RunMigrations()
	target := testutil.CreateTestUser(t, "github", "target1", "target")
	cookie := testutil.MakeAuthCookie(t, admin.ID)

	body := map[string]interface{}{"user_id": target.ID, "ban": true}
	req := testutil.JSONRequest(t, "POST", "/api/admin/ban", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleBanUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	fetched, _ := store.GetUserByID(target.ID)
	if !fetched.Banned {
		t.Error("expected target to be banned")
	}
}

func TestHandleBanUser_NonAdminForbidden(t *testing.T) {
	testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, "github", "admin1", "admin")
	nonAdmin := testutil.CreateTestUser(t, "github", "regular1", "regular")
	cookie := testutil.MakeAuthCookie(t, nonAdmin.ID)

	body := map[string]interface{}{"user_id": 1, "ban": true}
	req := testutil.JSONRequest(t, "POST", "/api/admin/ban", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleBanUser(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestHandleBanUser_CannotBanSelf(t *testing.T) {
	testutil.SetupTestDB(t)
	admin := testutil.CreateTestUser(t, "github", "admin1", "admin")
	store.RunMigrations()
	cookie := testutil.MakeAuthCookie(t, admin.ID)

	body := map[string]interface{}{"user_id": admin.ID, "ban": true}
	req := testutil.JSONRequest(t, "POST", "/api/admin/ban", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleBanUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for self-ban, got %d", rr.Code)
	}
}

func TestHandleListUsers_AdminOnly(t *testing.T) {
	testutil.SetupTestDB(t)
	admin := testutil.CreateTestUser(t, "github", "admin1", "admin")
	store.RunMigrations()
	testutil.CreateTestUser(t, "github", "user2", "user2")
	cookie := testutil.MakeAuthCookie(t, admin.ID)

	req := httptest.NewRequest("GET", "/api/admin/users", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	game.HandleListUsers(rr, req)

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
	testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, "github", "admin1", "admin")
	nonAdmin := testutil.CreateTestUser(t, "github", "regular1", "regular")
	cookie := testutil.MakeAuthCookie(t, nonAdmin.ID)

	req := httptest.NewRequest("GET", "/api/admin/users", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	game.HandleListUsers(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// --- streak_view ---

func queryStreak(t *testing.T, userID int64) int {
	t.Helper()
	var streak int
	store.DB.QueryRow("SELECT COALESCE(current_streak, 0) FROM streak_view WHERE user_id = ?", userID).Scan(&streak)
	return streak
}

func TestStreakView_AllWins(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "streaker")

	for i, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13"} {
		guesses := i + 2
		store.InsertGameResult(user.ID, date, true, &guesses, false)
	}

	if streak := queryStreak(t, user.ID); streak != 3 {
		t.Errorf("expected streak=3, got %d", streak)
	}
}

func TestStreakView_BrokenByLoss(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")

	g3 := 3
	store.InsertGameResult(user.ID, "2024-01-15", true, &g3, false)
	store.InsertGameResult(user.ID, "2024-01-14", true, &g3, false)
	store.InsertGameResult(user.ID, "2024-01-13", false, nil, false)
	store.InsertGameResult(user.ID, "2024-01-12", true, &g3, false)

	if streak := queryStreak(t, user.ID); streak != 2 {
		t.Errorf("expected streak=2, got %d", streak)
	}
}

func TestStreakView_NoResults(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "newbie")

	if streak := queryStreak(t, user.ID); streak != 0 {
		t.Errorf("expected streak=0, got %d", streak)
	}
}

func TestStreakView_AllLosses(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "loser")

	store.InsertGameResult(user.ID, "2024-01-15", false, nil, false)
	store.InsertGameResult(user.ID, "2024-01-14", false, nil, false)

	if streak := queryStreak(t, user.ID); streak != 0 {
		t.Errorf("expected streak=0, got %d", streak)
	}
}

// --- HandleSubmitResult ---

func TestSubmitResult_BasicWin(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	guesses := 4
	body := map[string]interface{}{
		"date":      "2024-01-15",
		"won":       true,
		"guesses":   guesses,
		"hard_mode": false,
	}
	req := testutil.JSONRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSubmitResult(rr, req)

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
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"date":      "2024-01-15",
		"won":       false,
		"hard_mode": false,
	}
	req := testutil.JSONRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSubmitResult(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var won bool
	store.DB.QueryRow("SELECT won FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&won)
	if won {
		t.Error("expected won=false")
	}
}

func TestSubmitResult_HardMode(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	guesses := 3
	body := map[string]interface{}{
		"date":      "2024-01-15",
		"won":       true,
		"guesses":   guesses,
		"hard_mode": true,
	}
	req := testutil.JSONRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSubmitResult(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var hardMode bool
	store.DB.QueryRow("SELECT hard_mode FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&hardMode)
	if !hardMode {
		t.Error("expected hard_mode=true")
	}
}

func TestSubmitResult_DuplicateDate(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	guesses := 4
	body := map[string]interface{}{
		"date":      "2024-01-15",
		"won":       true,
		"guesses":   guesses,
		"hard_mode": false,
	}
	req := testutil.JSONRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSubmitResult(rr, req)

	guesses2 := 2
	body["guesses"] = guesses2
	req = testutil.JSONRequest(t, "POST", "/api/result", body, cookie)
	rr = httptest.NewRecorder()
	game.HandleSubmitResult(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for duplicate, got %d", rr.Code)
	}

	var g int
	store.DB.QueryRow("SELECT guesses FROM game_results WHERE user_id = ? AND date = ?", user.ID, "2024-01-15").Scan(&g)
	if g != 4 {
		t.Errorf("expected original guesses=4, got %d", g)
	}
}

func TestSubmitResult_MissingDate(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	body := map[string]interface{}{
		"won":     true,
		"guesses": 3,
	}
	req := testutil.JSONRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSubmitResult(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing date, got %d", rr.Code)
	}
}

func TestSubmitResult_InvalidGuessCount(t *testing.T) {
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "player")
	cookie := testutil.MakeAuthCookie(t, user.ID)

	guesses := 7
	body := map[string]interface{}{
		"date":    "2024-01-15",
		"won":     true,
		"guesses": guesses,
	}
	req := testutil.JSONRequest(t, "POST", "/api/result", body, cookie)
	rr := httptest.NewRecorder()
	game.HandleSubmitResult(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid guess count, got %d", rr.Code)
	}
}

func TestSubmitResult_NotAuthenticated(t *testing.T) {
	testutil.SetupTestDB(t)

	body := map[string]interface{}{
		"date":    "2024-01-15",
		"won":     true,
		"guesses": 3,
	}
	req := testutil.JSONRequest(t, "POST", "/api/result", body, nil)
	rr := httptest.NewRecorder()
	game.HandleSubmitResult(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// --- HandleGetLeaderboard ---

func TestLeaderboard_Empty(t *testing.T) {
	testutil.SetupTestDB(t)

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	game.HandleGetLeaderboard(rr, req)

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
	testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, "github", "1", "solo")

	g := 3
	store.InsertGameResult(user.ID, "2024-01-15", true, &g, false)

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	game.HandleGetLeaderboard(rr, req)

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
	testutil.SetupTestDB(t)
	good := testutil.CreateTestUser(t, "github", "1", "GoodPlayer")
	bad := testutil.CreateTestUser(t, "github", "2", "BadPlayer")

	for _, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13", "2024-01-12", "2024-01-11"} {
		g := 2
		store.InsertGameResult(good.ID, date, true, &g, false)
	}

	for _, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13", "2024-01-12", "2024-01-11"} {
		g := 6
		store.InsertGameResult(bad.ID, date, true, &g, false)
	}

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	game.HandleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	first := entries[0].(map[string]interface{})
	if first["display_name"] != "GoodPlayer" {
		t.Errorf("expected GoodPlayer first, got %v", first["display_name"])
	}
}

func TestLeaderboard_BannedExcluded(t *testing.T) {
	testutil.SetupTestDB(t)
	good := testutil.CreateTestUser(t, "github", "1", "GoodPlayer")
	banned := testutil.CreateTestUser(t, "github", "2", "BannedPlayer")
	store.BanUser(banned.ID)

	g := 3
	store.InsertGameResult(good.ID, "2024-01-15", true, &g, false)
	store.InsertGameResult(banned.ID, "2024-01-15", true, &g, false)

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	game.HandleGetLeaderboard(rr, req)

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
	testutil.SetupTestDB(t)

	for i := range 5 {
		user := testutil.CreateTestUser(t, "github", string(rune('A'+i)), "Player"+string(rune('A'+i)))
		g := 3
		store.InsertGameResult(user.ID, "2024-01-15", true, &g, false)
	}

	req := httptest.NewRequest("GET", "/api/leaderboard?limit=2", nil)
	rr := httptest.NewRecorder()
	game.HandleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with limit=2, got %d", len(entries))
	}
}

func TestLeaderboard_HardModeBonus(t *testing.T) {
	testutil.SetupTestDB(t)
	hardPlayer := testutil.CreateTestUser(t, "github", "1", "HardMode")
	normalPlayer := testutil.CreateTestUser(t, "github", "2", "Normal")

	for _, date := range []string{"2024-01-15", "2024-01-14", "2024-01-13", "2024-01-12", "2024-01-11"} {
		g := 4
		store.InsertGameResult(hardPlayer.ID, date, true, &g, true)
		store.InsertGameResult(normalPlayer.ID, date, true, &g, false)
	}

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	game.HandleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})

	first := entries[0].(map[string]interface{})
	if first["display_name"] != "HardMode" {
		t.Errorf("expected HardMode player first, got %v", first["display_name"])
	}
}

func TestLeaderboard_LossPenalty(t *testing.T) {
	testutil.SetupTestDB(t)
	winner := testutil.CreateTestUser(t, "github", "1", "Winner")
	loser := testutil.CreateTestUser(t, "github", "2", "Loser")

	g := 4
	store.InsertGameResult(winner.ID, "2024-01-15", true, &g, false)
	store.InsertGameResult(loser.ID, "2024-01-15", false, nil, false)

	req := httptest.NewRequest("GET", "/api/leaderboard", nil)
	rr := httptest.NewRecorder()
	game.HandleGetLeaderboard(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	entries := resp["leaderboard"].([]interface{})

	first := entries[0].(map[string]interface{})
	if first["display_name"] != "Winner" {
		t.Errorf("expected Winner first, got %v", first["display_name"])
	}
}
