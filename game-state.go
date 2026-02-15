package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type GameProgress struct {
	Guesses  []string `json:"guesses"`
	HardMode bool     `json:"hardMode"`
	GameOver bool     `json:"gameOver"`
	Won      bool     `json:"won"`
}

func handleGetGameState(w http.ResponseWriter, r *http.Request) {
	log.Printf("GET /api/game-state called, date=%s", r.URL.Query().Get("date"))
	user := getUserFromRequest(r)
	if user == nil {
		log.Printf("GET /api/game-state: not authenticated")
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, "Date is required", http.StatusBadRequest)
		return
	}

	var guessesJSON string
	var hardMode, gameOver, won bool

	err := db.QueryRow(
		"SELECT guesses, hard_mode, game_over, won FROM game_progress WHERE user_id = ? AND date = ?",
		user.ID, date,
	).Scan(&guessesJSON, &hardMode, &gameOver, &won)

	if err != nil {
		// No progress found — return empty state
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GameProgress{Guesses: []string{}})
		return
	}

	var guesses []string
	if err := json.Unmarshal([]byte(guessesJSON), &guesses); err != nil {
		guesses = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GameProgress{
		Guesses:  guesses,
		HardMode: hardMode,
		GameOver: gameOver,
		Won:      won,
	})
}

func handleSaveProgress(w http.ResponseWriter, r *http.Request) {
	log.Printf("POST /api/save-progress called")
	user := getUserFromRequest(r)
	if user == nil {
		log.Printf("POST /api/save-progress: not authenticated")
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var body struct {
		Date       string   `json:"date"`
		Guesses    []string `json:"guesses"`
		HardMode   bool     `json:"hardMode"`
		GameOver   bool     `json:"gameOver"`
		Won        bool     `json:"won"`
		ClientTime string   `json:"client_time"`
		TzOffset   *int     `json:"tz_offset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Date == "" {
		http.Error(w, "Date is required", http.StatusBadRequest)
		return
	}

	guessesJSON, _ := json.Marshal(body.Guesses)

	_, err := db.Exec(`
		INSERT INTO game_progress (user_id, date, guesses, hard_mode, game_over, won)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, date) DO UPDATE SET
			guesses = excluded.guesses,
			hard_mode = excluded.hard_mode,
			game_over = excluded.game_over,
			won = excluded.won
	`, user.ID, body.Date, string(guessesJSON), body.HardMode, body.GameOver, body.Won)

	if err != nil {
		log.Printf("POST /api/save-progress: db error: %v", err)
		http.Error(w, "Failed to save progress", http.StatusInternalServerError)
		return
	}

	// If game is over, ensure a game_results entry exists (don't rely on client)
	if body.GameOver {
		guessCount := len(body.Guesses)
		var guessPtr *int
		if body.Won && guessCount > 0 {
			guessPtr = &guessCount
		}
		if err := insertGameResult(user.ID, body.Date, body.Won, guessPtr, body.HardMode); err != nil {
			log.Printf("POST /api/save-progress: auto-insert game_result failed: %v", err)
		} else {
			log.Printf("POST /api/save-progress: auto-inserted game_result for user %d, date=%s, won=%v, guesses=%d", user.ID, body.Date, body.Won, guessCount)
		}
	}

	// Timezone manipulation check
	tzWarning := false
	if body.ClientTime != "" && body.TzOffset != nil {
		ip := getClientIP(r)
		tzWarning = checkTimezone(user.ID, body.ClientTime, *body.TzOffset, ip, "save-progress")
	}

	log.Printf("POST /api/save-progress: saved for user %d, date=%s, %d guesses", user.ID, body.Date, len(body.Guesses))
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{"ok": true}
	if tzWarning {
		resp["tz_warning"] = true
	}
	json.NewEncoder(w).Encode(resp)
}

type UserStats struct {
	Played        int    `json:"played"`
	Won           int    `json:"won"`
	PlayedHard    int    `json:"playedHard"`
	WonHard       int    `json:"wonHard"`
	CurrentStreak int    `json:"currentStreak"`
	MaxStreak     int    `json:"maxStreak"`
	Distribution  []int  `json:"distribution"`
	LastDate      string `json:"lastDate"`
	HardMode      bool   `json:"hardMode"`
}

func handleGetUserStats(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var played, won, playedHard, wonHard, currentStreak, maxStreak int
	var distributionJSON string
	var lastDate *string
	var hardModeVal bool

	err := db.QueryRow(
		"SELECT played, won, played_hard, won_hard, current_streak, max_streak, distribution, last_date, hard_mode FROM user_stats WHERE user_id = ?",
		user.ID,
	).Scan(&played, &won, &playedHard, &wonHard, &currentStreak, &maxStreak, &distributionJSON, &lastDate, &hardModeVal)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserStats{Distribution: []int{0, 0, 0, 0, 0, 0}})
		return
	}

	var distribution []int
	if err := json.Unmarshal([]byte(distributionJSON), &distribution); err != nil {
		distribution = []int{0, 0, 0, 0, 0, 0}
	}

	ld := ""
	if lastDate != nil {
		ld = *lastDate
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UserStats{
		Played:        played,
		Won:           won,
		PlayedHard:    playedHard,
		WonHard:       wonHard,
		CurrentStreak: currentStreak,
		MaxStreak:     maxStreak,
		Distribution:  distribution,
		LastDate:      ld,
		HardMode:      hardModeVal,
	})
}

func handleSaveUserStats(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var body UserStats
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	distributionJSON, _ := json.Marshal(body.Distribution)

	_, err := db.Exec(`
		INSERT INTO user_stats (user_id, played, won, played_hard, won_hard, current_streak, max_streak, distribution, last_date, hard_mode)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			played = excluded.played,
			won = excluded.won,
			played_hard = excluded.played_hard,
			won_hard = excluded.won_hard,
			current_streak = excluded.current_streak,
			max_streak = excluded.max_streak,
			distribution = excluded.distribution,
			last_date = excluded.last_date,
			hard_mode = excluded.hard_mode
	`, user.ID, body.Played, body.Won, body.PlayedHard, body.WonHard, body.CurrentStreak, body.MaxStreak, string(distributionJSON), body.LastDate, body.HardMode)

	if err != nil {
		log.Printf("POST /api/user-stats: db error: %v", err)
		http.Error(w, "Failed to save stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// Only allow alphanumeric, spaces, hyphens, underscores
var validNamePattern = regexp.MustCompile(`^[\w\s\-]+$`)

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	// Collapse multiple spaces
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	return name
}

func checkProfanity(text string) (bool, error) {
	payload, _ := json.Marshal(map[string]string{"message": text})
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post("https://vector.profanity.dev", "application/json", bytes.NewReader(payload))
	if err != nil {
		// If the API is down, allow the name (fail open)
		log.Printf("Profanity API error: %v", err)
		return false, nil
	}
	defer resp.Body.Close()

	var result struct {
		IsProfanity bool `json:"isProfanity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, nil
	}
	return result.IsProfanity, nil
}

func handleUpdateDisplayName(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}
	if user.Banned {
		http.Error(w, "Account is banned", http.StatusForbidden)
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	name := sanitizeName(body.Name)
	nameLen := utf8.RuneCountInString(name)
	if name == "" || nameLen < 1 || nameLen > 20 {
		http.Error(w, "Name must be 1-20 characters", http.StatusBadRequest)
		return
	}

	if !validNamePattern.MatchString(name) {
		http.Error(w, "Name can only contain letters, numbers, spaces, hyphens, and underscores", http.StatusBadRequest)
		return
	}

	isProfane, err := checkProfanity(name)
	if err == nil && isProfane {
		http.Error(w, "That name is not allowed", http.StatusBadRequest)
		return
	}

	if err := updateCustomName(user.ID, name); err != nil {
		http.Error(w, "Failed to update name", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// Admin: ban/unban a user. Only user ID 1 (first registered user) can do this.
func handleBanUser(w http.ResponseWriter, r *http.Request) {
	admin := getUserFromRequest(r)
	if admin == nil || admin.ID != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		UserID int64 `json:"user_id"`
		Ban    bool  `json:"ban"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.UserID == admin.ID {
		http.Error(w, "Cannot ban yourself", http.StatusBadRequest)
		return
	}

	var err error
	if body.Ban {
		err = banUser(body.UserID)
	} else {
		err = unbanUser(body.UserID)
	}
	if err != nil {
		http.Error(w, "Failed to update ban status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	admin := getUserFromRequest(r)
	if admin == nil || admin.ID != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	rows, err := db.Query("SELECT id, provider, display_name, COALESCE(custom_name, ''), COALESCE(avatar_url, ''), banned FROM users ORDER BY id LIMIT ?", limit)
	if err != nil {
		http.Error(w, "Failed to query users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type userEntry struct {
		ID          int64  `json:"id"`
		Provider    string `json:"provider"`
		DisplayName string `json:"display_name"`
		CustomName  string `json:"custom_name,omitempty"`
		AvatarURL   string `json:"avatar_url,omitempty"`
		Banned      bool   `json:"banned"`
	}

	var users []userEntry
	for rows.Next() {
		var u userEntry
		rows.Scan(&u.ID, &u.Provider, &u.DisplayName, &u.CustomName, &u.AvatarURL, &u.Banned)
		users = append(users, u)
	}
	if users == nil {
		users = []userEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": users})
}
