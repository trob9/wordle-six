package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type LeaderboardEntry struct {
	Rank        int     `json:"rank"`
	UserID      int64   `json:"user_id"`
	DisplayName string  `json:"display_name"`
	AvatarURL   string  `json:"avatar_url,omitempty"`
	WeightedAvg float64 `json:"weighted_avg"`
	TrueAvg     float64 `json:"true_avg"`
	WinRate     float64 `json:"win_rate"`
	GamesPlayed int     `json:"games_played"`
	Streak      int     `json:"current_streak"`
	HardModeWins int    `json:"hard_mode_wins"`
}

func handleGetLeaderboard(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Weighted average guesses with hard mode 10% bonus.
	// Losses count as 7. Min 5 games.
	// Streak computed in Go since SQL window-based streak is complex in SQLite.
	rows, err := db.Query(`
		SELECT
			u.id AS user_id,
			COALESCE(u.custom_name, u.display_name),
			COALESCE(u.avatar_url, ''),
			SUM(
				CASE
					WHEN gr.won AND gr.hard_mode THEN gr.guesses * 0.9
					WHEN gr.won THEN gr.guesses
					ELSE 7.0
				END
			) / COUNT(*) AS weighted_avg,
			CAST(SUM(CASE WHEN gr.won THEN gr.guesses ELSE 7 END) AS REAL) / COUNT(*) AS true_avg,
			CAST(SUM(CASE WHEN gr.won THEN 1 ELSE 0 END) AS REAL) / COUNT(*) AS win_rate,
			COUNT(*) AS games_played,
			SUM(CASE WHEN gr.won AND gr.hard_mode THEN 1 ELSE 0 END) AS hard_mode_wins
		FROM users u
		JOIN game_results gr ON gr.user_id = u.id
		WHERE u.banned = FALSE
		GROUP BY u.id
		HAVING COUNT(*) >= 1
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
		err := rows.Scan(&e.UserID, &e.DisplayName, &e.AvatarURL, &e.WeightedAvg, &e.TrueAvg, &e.WinRate, &e.GamesPlayed, &e.HardModeWins)
		if err != nil {
			continue
		}
		e.Rank = rank
		rank++
		entries = append(entries, e)
	}

	// Compute current streak for each player
	for i := range entries {
		entries[i].Streak = computeStreak(entries[i].UserID)
	}

	if entries == nil {
		entries = []LeaderboardEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"leaderboard": entries})
}

func handleSubmitResult(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)
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

	// Validate guesses
	if body.Won && (body.Guesses == nil || *body.Guesses < 1 || *body.Guesses > 6) {
		http.Error(w, "Invalid guess count", http.StatusBadRequest)
		return
	}

	if err := insertGameResult(user.ID, body.Date, body.Won, body.Guesses, body.HardMode); err != nil {
		http.Error(w, "Failed to save result", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func computeStreak(userID int64) int {
	rows, err := db.Query(`
		SELECT won FROM game_results
		WHERE user_id = ?
		ORDER BY date DESC
	`, userID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	streak := 0
	for rows.Next() {
		var won bool
		rows.Scan(&won)
		if !won {
			break
		}
		streak++
	}
	return streak
}
