package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func handleFingerprint(w http.ResponseWriter, r *http.Request) {
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

	user := getUserFromRequest(r)
	var userID int64
	if user != nil {
		userID = user.ID
	}

	entry := map[string]interface{}{
		"ts":       time.Now().UTC().Format(time.RFC3339),
		"event":    "fingerprint",
		"fp_hash":  body.Hash,
		"tz":       body.Timezone,
		"screen":   body.Screen,
		"ip":       r.Header.Get("CF-Connecting-IP"),
		"user_id":  userID,
	}
	b, _ := json.Marshal(entry)
	log.Println(string(b))

	w.WriteHeader(http.StatusNoContent)
}
