package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
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
		Date     string   `json:"date"`
		Guesses  []string `json:"guesses"`
		HardMode bool     `json:"hardMode"`
		GameOver bool     `json:"gameOver"`
		Won      bool     `json:"won"`
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

	log.Printf("POST /api/save-progress: saved for user %d, date=%s, %d guesses", user.ID, body.Date, len(body.Guesses))
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
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

func handleUpdateDisplayName(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" || utf8.RuneCountInString(name) > 20 {
		http.Error(w, "Name must be 1-20 characters", http.StatusBadRequest)
		return
	}

	if err := updateCustomName(user.ID, name); err != nil {
		http.Error(w, "Failed to update name", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}
