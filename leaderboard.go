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

	// Bayesian weighted average: pulls players with few games toward the global mean.
	// Formula: bayesian_avg = (C * global_mean + player_sum) / (C + games_played)
	// C = 10 (confidence parameter). Higher C = more games needed to diverge from mean.
	// Hard mode wins get 10% bonus (guesses * 0.9). Losses count as 7.
	// Streak computed in Go since SQL window-based streak is complex in SQLite.
	rows, err := db.Query(`
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
			p.hard_mode_wins
		FROM player p, global g
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
		Date       string `json:"date"`
		Won        bool   `json:"won"`
		Guesses    *int   `json:"guesses"`
		HardMode   bool   `json:"hard_mode"`
		ClientTime string `json:"client_time"`
		TzOffset   *int   `json:"tz_offset"`
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

	// Timezone manipulation check
	tzWarning := false
	if body.ClientTime != "" && body.TzOffset != nil {
		ip := getClientIP(r)
		tzWarning = checkTimezone(user.ID, body.ClientTime, *body.TzOffset, ip, "result")
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{"ok": true}
	if tzWarning {
		resp["tz_warning"] = true
	}
	json.NewEncoder(w).Encode(resp)
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
