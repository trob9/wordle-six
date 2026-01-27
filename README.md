# Wordle Six

A daily 6-letter word puzzle game inspired by Wordle.

## Features

- **Daily Challenge**: New word every day at midnight, same for everyone
- **Offline Ready**: Local word list for instant validation, no API needed
- **Mobile Optimized**: Fully responsive with touch-friendly controls
- **Progressive Web App**: Install on mobile for app-like experience
- **Stats Tracking**: Win rate, streaks, and guess distribution
- **Share Results**: Share your daily results with emoji grid

## How It Works

### Deterministic Daily Words
- Uses seeded random based on date to ensure everyone gets the same word
- Curated list of 700+ quality 6-letter words (no plurals, common words)
- No server needed - fully static and client-side

### Word Validation
- Local word list with ~5,700 valid 6-letter words
- Instant validation with no network requests
- Works fully offline
- Daily words selected from curated non-plural list

### Mobile First
- Responsive CSS with clamp() for perfect scaling
- Touch-optimized keyboard
- No zoom/scroll issues on mobile
- PWA manifest for installability

## Tech Stack

- Pure HTML/CSS/JavaScript
- No build process or dependencies
- Fully offline capable
- localStorage for state management

## Deployment

Hosted at https://wordle-six.tomtom.fyi via VibeDeploy.

---

Built with ❤️ by Tom
