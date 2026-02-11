# CLAUDE.md — Wordle Six

## Project Overview

Wordle Six is a daily 6-letter word puzzle game. Go backend + vanilla JS frontend. SQLite database. OAuth authentication with cross-device game state sync.

**Live URL:** https://wordle-six.tomtom.fyi
**Deployment:** Docker container on vibe-deploy (`vibe-deploy_web` network), Cloudflare Tunnel via Caddy reverse proxy.

## File Map

### Backend (Go 1.23)

| File | Purpose |
|------|---------|
| `main.go` | HTTP server, route registration, static file serving. Port 8080. Blocks `.go`/`.mod`/`.sum` from being served. |
| `auth.go` | OAuth2 flow (GitHub, Discord, Google). Manual implementation — no oauth2 library. JWT session cookies (30-day, HttpOnly, SameSite=Lax). `getUserFromRequest()` is the auth helper used by all protected endpoints. |
| `db.go` | SQLite init (WAL mode), table creation, migrations, user CRUD. DB path: `/data/wordle-six.db` (Docker volume). |
| `game-state.go` | Game progress sync (save/load guesses), user stats sync, display name update. |
| `leaderboard.go` | Leaderboard query (weighted avg with hard mode bonus), game result submission, streak computation. |
| `go.mod` | Dependencies: `go-sqlite3`, `golang-jwt/jwt/v5` |
| `Dockerfile` | Multi-stage: `golang:1.23-alpine` builder → `alpine:3.20` runtime. Static files copied to `/app/static/`. |

### Frontend (Vanilla JS, no build step)

| File | Purpose |
|------|---------|
| `index.html` | All HTML + all CSS (inline `<style>`). Modals: help, settings, stats, leaderboard, win, loss, welcome (first login name prompt). |
| `game.js` | Core game logic. Board rendering, keyboard input, guess validation, hard mode enforcement, animations, stats tracking, server sync. Exports `initGame()` (called by auth-ui.js after auth check). |
| `auth-ui.js` | Auth UI (login dropdown, avatar), leaderboard rendering, display name management, result submission. Runs an async IIFE on load that fetches `/auth/me`, sets `currentUser`, then calls `initGame()`. |
| `words.js` | 743 curated daily words (`DAILY_WORDS` array). Deterministic selection via seeded PRNG based on date. |
| `valid-words.js` | 14,404 valid 6-letter words (`VALID_WORDS` Set). Client-side validation, no server round-trip. |

### Static Assets

`manifest.json`, `icon.svg`, `icon-192.png`, `icon-512.png`, `og-preview.png`

## Database Schema (SQLite)

### `users`
- `id` INTEGER PK — internal user ID
- `provider` TEXT — "github", "discord", or "google"
- `provider_id` TEXT — provider's user ID
- `display_name` TEXT — from OAuth provider (GitHub username, Discord username, Google name)
- `custom_name` TEXT nullable — user-chosen display name, set on first login
- `avatar_url` TEXT nullable
- Unique constraint: `(provider, provider_id)`
- Leaderboard uses `COALESCE(custom_name, display_name)`

### `game_results`
- Final game outcomes only. One row per user per day.
- `won` BOOLEAN, `guesses` INTEGER (null if lost), `hard_mode` BOOLEAN
- Unique: `(user_id, date)`, insert via `ON CONFLICT DO NOTHING` (no updates)
- Powers leaderboard rankings

### `game_progress`
- Live in-progress game state. Upserted after every guess.
- `guesses` TEXT — JSON array of guess strings, e.g. `["SNATCH","BRIDGE"]`
- `hard_mode` BOOLEAN, `game_over` BOOLEAN, `won` BOOLEAN
- Unique: `(user_id, date)`
- Enables cross-device resume

### `user_stats`
- Cumulative stats per user. One row per user.
- `played`, `won`, `played_hard`, `won_hard` — game counts
- `current_streak`, `max_streak` — win streaks
- `distribution` TEXT — JSON array of 6 ints (wins by guess count)
- `last_date` TEXT — for streak calculation
- `hard_mode` BOOLEAN — user's hard mode preference
- Updated after game completion and hard mode toggle

### Migrations
`runMigrations()` in `db.go` handles schema evolution via idempotent `ALTER TABLE` statements (errors from duplicate columns are silently ignored).

## Key Behaviors

### Init Ordering
1. `words.js` and `valid-words.js` load first (word data)
2. `game.js` loads — defines `initGame()` but does NOT call it
3. `auth-ui.js` loads — IIFE fetches `/auth/me`, sets `currentUser`, then calls `initGame()`
4. `initGame()` loads localStorage, then if logged in, fetches server state and uses whichever has more guesses

### Game State Sync
- **After each guess:** `saveGameState()` (localStorage) + `saveProgressToServer()` (fire-and-forget POST)
- **On game end:** stats updated locally + `saveStatsToServer()` + `submitResultToAPI()` (leaderboard)
- **On page load (logged in):** server state fetched; if server has more guesses than local, server state wins and board is re-rendered
- **Logged out:** localStorage only, no server calls

### Hard Mode
- Toggle in settings. Can be enabled mid-game if `canEnableHardMode()` passes (replays all prior guesses through `checkHardMode()` to verify compliance)
- Disabled (greyed out) when game is over
- `checkHardMode(guess)` validates against `gameState.guesses` (all previous guesses), checking: green positions fixed, yellow letters present but not in excluded positions, absent letters not reused
- Hard mode preference persisted in `user_stats.hard_mode` and synced on login

### Custom Display Names
- `users.custom_name` is NULL on first OAuth login
- `/auth/me` returns `is_new: true` when `custom_name` is NULL
- `auth-ui.js` shows welcome modal on first login to prompt for name
- Name can be changed later in settings modal (visible only when logged in)
- `POST /api/display-name` with `{ name: "..." }` (1-20 chars, trimmed)

### Leaderboard
- Weighted average: hard mode wins score `guesses * 0.9`, normal wins score `guesses`, losses score `7.0`
- Sorted by weighted avg ASC, then win rate DESC, then games played DESC
- Top 3 get gold/silver/bronze trophy SVG icons
- Streak computed in Go by iterating `game_results` DESC
- Compact top-3 shown below game board; full list in modal

### Stats Display
- Stats modal shows: Played (with hard mode count), Won (with hard mode count), Streak, Best
- Hard mode sub-counts displayed as "(3H)" suffix in present/yellow color
- Guess distribution chart with highlighted bar for current game

## Environment Variables

| Var | Required | Purpose |
|-----|----------|---------|
| `GITHUB_CLIENT_ID` | Yes | GitHub OAuth app |
| `GITHUB_CLIENT_SECRET` | Yes | GitHub OAuth app |
| `DISCORD_CLIENT_ID` | Yes | Discord OAuth app |
| `DISCORD_CLIENT_SECRET` | Yes | Discord OAuth app |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth app |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth app |
| `JWT_SECRET` | No | Session signing key (random if unset, sessions lost on restart) |
| `PORT` | No | Server port (default: 8080) |
| `DB_PATH` | No | SQLite path (default: `/data/wordle-six.db`) |

## Common Tasks

### Rebuild and redeploy
```bash
cd /home/tom/projects/vibe-deploy/deployments/wordle-six
go build ./...  # verify
docker build -t vibe-deploy-wordle-six:latest .
docker stop wordle-six && docker rm wordle-six
docker run -d --name wordle-six --network vibe-deploy_web -v wordle-six-data:/data \
  -e GITHUB_CLIENT_ID=... -e ... --restart unless-stopped vibe-deploy-wordle-six:latest
```
After deploying, purge Cloudflare cache for JS/CSS changes to take effect.

### Access the database
```bash
sudo sqlite3 /var/lib/docker/volumes/wordle-six-data/_data/wordle-six.db
```

### Check logs
```bash
docker logs wordle-six --tail 50
```

## Rules / Constraints

- Words are 6 letters, 6 max guesses
- One word per day, same for all players (deterministic from `DAILY_WORDS` array + date seed)
- Game results are insert-once — no re-submitting for the same date
- Stats are server-authoritative when logged in (server stats loaded on login, not merged from local)
- localStorage is the fallback for logged-out play
- No email addresses are stored or exposed — Google OAuth uses `userinfo.profile` scope only
