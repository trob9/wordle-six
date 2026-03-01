package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"wordle-six/internal/auth"
	"wordle-six/internal/store"
)

// --- Game progress ---

type GameProgress struct {
	Guesses  []string `json:"guesses"`
	HardMode bool     `json:"hardMode"`
	GameOver bool     `json:"gameOver"`
	Won      bool     `json:"won"`
}

func HandleGetGameState(w http.ResponseWriter, r *http.Request) {
	log.Printf("GET /api/game-state called, date=%s", r.URL.Query().Get("date"))
	user := auth.GetUserFromRequest(r)
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

	err := store.DB.QueryRow(
		"SELECT guesses, hard_mode, game_over, won FROM game_progress WHERE user_id = ? AND date = ?",
		user.ID, date,
	).Scan(&guessesJSON, &hardMode, &gameOver, &won)

	if err != nil {
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

func HandleSaveProgress(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)

	var body struct {
		Date     string   `json:"date"`
		Guesses  []string `json:"guesses"`
		HardMode bool     `json:"hardMode"`
		GameOver bool     `json:"gameOver"`
		Won      bool     `json:"won"`
	}
	decodeErr := json.NewDecoder(r.Body).Decode(&body)

	if user == nil {
		if decodeErr == nil && len(body.Guesses) > 0 && body.Date != "" {
			logGuessEvent(0, "", body.Date, body.Guesses, body.HardMode, body.GameOver, body.Won, false, r.Header.Get("CF-Connecting-IP"))
		}
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	if decodeErr != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Date == "" {
		http.Error(w, "Date is required", http.StatusBadRequest)
		return
	}

	if len(body.Guesses) > 6 {
		http.Error(w, "Too many guesses", http.StatusBadRequest)
		return
	}

	var existingCount int
	store.DB.QueryRow("SELECT COUNT(*) FROM game_progress WHERE user_id = ? AND date = ?", user.ID, body.Date).Scan(&existingCount)

	guessesJSON, _ := json.Marshal(body.Guesses)

	_, err := store.DB.Exec(`
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

	if body.GameOver {
		guessCount := len(body.Guesses)
		var guessPtr *int
		if body.Won && guessCount > 0 {
			guessPtr = &guessCount
		}
		if err := store.InsertGameResult(user.ID, body.Date, body.Won, guessPtr, body.HardMode); err != nil {
			log.Printf("POST /api/save-progress: auto-insert game_result failed: %v", err)
		}
	}

	anonSync := existingCount == 0 && len(body.Guesses) > 1
	name := user.DisplayName
	if user.CustomName != nil {
		name = *user.CustomName
	}
	logGuessEvent(user.ID, name, body.Date, body.Guesses, body.HardMode, body.GameOver, body.Won, anonSync, r.Header.Get("CF-Connecting-IP"))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func logGuessEvent(userID int64, displayName string, date string, guesses []string, hardMode, gameOver, won, anonSync bool, ip string) {
	if len(guesses) == 0 {
		return
	}
	loggedIn := userID != 0
	entry := map[string]interface{}{
		"ts":        time.Now().UTC().Format(time.RFC3339),
		"event":     "guess",
		"logged_in": loggedIn,
		"date":      date,
		"guess":     guesses[len(guesses)-1],
		"guess_num": len(guesses),
		"game_over": gameOver,
		"won":       won,
		"hard_mode": hardMode,
		"ip":        ip,
	}
	if loggedIn {
		entry["user_id"] = userID
		entry["display_name"] = displayName
	}
	if anonSync {
		entry["anon_sync"] = true
		entry["synced_guess_count"] = len(guesses)
	}
	b, _ := json.Marshal(entry)
	fmt.Println(string(b))
}

// --- User stats ---

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

func HandleGetUserStats(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var played, won, playedHard, wonHard, currentStreak, maxStreak int
	var distributionJSON string
	var lastDate *string
	var hardModeVal bool

	err := store.DB.QueryRow(
		"SELECT played, won, played_hard, won_hard, distribution, last_date, hard_mode FROM user_stats WHERE user_id = ?",
		user.ID,
	).Scan(&played, &won, &playedHard, &wonHard, &distributionJSON, &lastDate, &hardModeVal)
	store.DB.QueryRow("SELECT COALESCE(current_streak, 0), COALESCE(max_streak, 0) FROM streak_view WHERE user_id = ?", user.ID).Scan(&currentStreak, &maxStreak)

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

func HandleSaveUserStats(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
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

	_, err := store.DB.Exec(`
		INSERT INTO user_stats (user_id, played, won, played_hard, won_hard, distribution, last_date, hard_mode)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			played = excluded.played,
			won = excluded.won,
			played_hard = excluded.played_hard,
			won_hard = excluded.won_hard,
			distribution = excluded.distribution,
			last_date = excluded.last_date,
			hard_mode = excluded.hard_mode
	`, user.ID, body.Played, body.Won, body.PlayedHard, body.WonHard, string(distributionJSON), body.LastDate, body.HardMode)

	if err != nil {
		log.Printf("POST /api/user-stats: db error: %v", err)
		http.Error(w, "Failed to save stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// --- Display name ---

// Only allow alphanumeric, spaces, hyphens, underscores
var validNamePattern = regexp.MustCompile(`^[\w\s\-]+$`)

// SanitizeName trims whitespace and collapses internal spaces.
func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	return name
}

func checkProfanity(text string) (bool, error) {
	payload, _ := json.Marshal(map[string]string{"message": text})
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post("https://vector.profanity.dev", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("Profanity API error (rejecting name): %v", err)
		return false, fmt.Errorf("profanity check unavailable")
	}
	defer resp.Body.Close()

	var result struct {
		IsProfanity bool `json:"isProfanity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("profanity check decode error")
	}
	return result.IsProfanity, nil
}

func HandleUpdateDisplayName(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
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

	name := SanitizeName(body.Name)
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
	if err != nil {
		http.Error(w, "Unable to verify name, please try again later", http.StatusServiceUnavailable)
		return
	}
	if isProfane {
		http.Error(w, "That name is not allowed", http.StatusBadRequest)
		return
	}

	if err := store.UpdateCustomName(user.ID, name); err != nil {
		http.Error(w, "Failed to update name", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// --- Admin ---

func HandleBanUser(w http.ResponseWriter, r *http.Request) {
	admin := auth.GetUserFromRequest(r)
	if admin == nil || !admin.IsAdmin {
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
		err = store.BanUser(body.UserID)
	} else {
		err = store.UnbanUser(body.UserID)
	}
	if err != nil {
		http.Error(w, "Failed to update ban status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func HandleListUsers(w http.ResponseWriter, r *http.Request) {
	admin := auth.GetUserFromRequest(r)
	if admin == nil || !admin.IsAdmin {
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

	rows, err := store.DB.Query("SELECT id, provider, display_name, COALESCE(custom_name, ''), COALESCE(avatar_url, ''), banned FROM users ORDER BY id LIMIT ?", limit)
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

// --- Leaderboard ---

type LeaderboardEntry struct {
	Rank         int     `json:"rank"`
	UserID       int64   `json:"user_id"`
	DisplayName  string  `json:"display_name"`
	AvatarURL    string  `json:"avatar_url,omitempty"`
	WeightedAvg  float64 `json:"weighted_avg"`
	TrueAvg      float64 `json:"true_avg"`
	WinRate      float64 `json:"win_rate"`
	GamesPlayed  int     `json:"games_played"`
	Streak       int     `json:"current_streak"`
	HardModeWins int     `json:"hard_mode_wins"`
}

func HandleGetLeaderboard(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Bayesian weighted average: pulls players with few games toward the global mean.
	// Formula: bayesian_avg = (C * global_mean + player_sum) / (C + games_played)
	// C = 10. Hard mode wins get 10% bonus (guesses * 0.9). Losses count as 8.
	rows, err := store.DB.Query(`
		WITH global AS (
			SELECT SUM(
				CASE
					WHEN gr.won AND gr.hard_mode THEN gr.guesses * 0.9
					WHEN gr.won THEN gr.guesses
					ELSE 8.0
				END
			) / COUNT(*) AS mean
			FROM game_results gr
			JOIN users u ON u.id = gr.user_id
			WHERE u.banned = FALSE
		),
		player AS (
			SELECT
				u.id AS user_id,
				COALESCE(u.custom_name, u.display_name) AS display_name,
				COALESCE(u.avatar_url, '') AS avatar_url,
				SUM(
					CASE
						WHEN gr.won AND gr.hard_mode THEN gr.guesses * 0.9
						WHEN gr.won THEN gr.guesses
						ELSE 8.0
					END
				) AS score_sum,
				CAST(SUM(CASE WHEN gr.won THEN gr.guesses ELSE 8 END) AS REAL) / COUNT(*) AS true_avg,
				CAST(SUM(CASE WHEN gr.won THEN 1 ELSE 0 END) AS REAL) / COUNT(*) AS win_rate,
				COUNT(*) AS games_played,
				SUM(CASE WHEN gr.won AND gr.hard_mode THEN 1 ELSE 0 END) AS hard_mode_wins
			FROM users u
			JOIN game_results gr ON gr.user_id = u.id
			WHERE u.banned = FALSE
			GROUP BY u.id
			HAVING COUNT(*) >= 1
		)
		SELECT
			p.user_id,
			p.display_name,
			p.avatar_url,
			(10.0 * g.mean + p.score_sum) / (10.0 + p.games_played) AS weighted_avg,
			p.true_avg,
			p.win_rate,
			p.games_played,
			p.hard_mode_wins,
			COALESCE(sv.current_streak, 0) AS current_streak
		FROM player p
		CROSS JOIN global g
		LEFT JOIN streak_view sv ON sv.user_id = p.user_id
		ORDER BY weighted_avg ASC, win_rate DESC, games_played DESC
		LIMIT ?
	`, limit)
	if err != nil {
		http.Error(w, "Failed to query leaderboard", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e LeaderboardEntry
		err := rows.Scan(&e.UserID, &e.DisplayName, &e.AvatarURL, &e.WeightedAvg, &e.TrueAvg, &e.WinRate, &e.GamesPlayed, &e.HardModeWins, &e.Streak)
		if err != nil {
			continue
		}
		e.Rank = rank
		rank++
		entries = append(entries, e)
	}

	if entries == nil {
		entries = []LeaderboardEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"leaderboard": entries})
}

func HandleSubmitResult(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var body struct {
		Date     string `json:"date"`
		Won      bool   `json:"won"`
		Guesses  *int   `json:"guesses"`
		HardMode bool   `json:"hard_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Date == "" {
		http.Error(w, "Date is required", http.StatusBadRequest)
		return
	}

	if body.Won && (body.Guesses == nil || *body.Guesses < 1 || *body.Guesses > 6) {
		http.Error(w, "Invalid guess count", http.StatusBadRequest)
		return
	}

	if err := store.InsertGameResult(user.ID, body.Date, body.Won, body.Guesses, body.HardMode); err != nil {
		http.Error(w, "Failed to save result", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

// --- Fingerprint ---

func HandleFingerprint(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Hash      string `json:"hash"`
		UserAgent string `json:"user_agent"`
		Timezone  string `json:"timezone"`
		Screen    string `json:"screen"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user := auth.GetUserFromRequest(r)
	var userID int64
	if user != nil {
		userID = user.ID
	}

	entry := map[string]interface{}{
		"ts":      time.Now().UTC().Format(time.RFC3339),
		"event":   "fingerprint",
		"fp_hash": body.Hash,
		"tz":      body.Timezone,
		"screen":  body.Screen,
		"ip":      r.Header.Get("CF-Connecting-IP"),
		"user_id": userID,
	}
	b, _ := json.Marshal(entry)
	log.Println(string(b))

	w.WriteHeader(http.StatusNoContent)
}
