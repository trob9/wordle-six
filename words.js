// API-based word selection - no word lists
// Uses random-word-api for daily word selection

// Get date string for tracking (YYYY-MM-DD)
function getDateString() {
    const now = new Date();
    return now.toISOString().split('T')[0];
}

// Calculate time until next word (midnight)
function getTimeUntilNextWord() {
    const now = new Date();
    const tomorrow = new Date(now);
    tomorrow.setDate(tomorrow.getDate() + 1);
    tomorrow.setHours(0, 0, 0, 0);
    return tomorrow - now;
}

// Format milliseconds to HH:MM:SS
function formatTime(ms) {
    const hours = Math.floor(ms / (1000 * 60 * 60));
    const minutes = Math.floor((ms % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((ms % (1000 * 60)) / 1000);
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
}

// Seeded random number generator for deterministic daily words
function seededRandom(seed) {
    const x = Math.sin(seed) * 10000;
    return x - Math.floor(x);
}

// Get seed from today's date
function getDailySeed() {
    const now = new Date();
    const start = new Date(2024, 0, 1);
    return Math.floor((now - start) / (1000 * 60 * 60 * 24)) + 42069;
}

// Word game module
const WordGame = (function() {
    'use strict';

    let _word = null;
    let _gameOver = false;
    let _ready = false;
    let _readyPromise = null;

    // Fetch today's word from API
    async function fetchDailyWord() {
        const cacheKey = 'dailyWord_' + getDateString();
        const cached = localStorage.getItem(cacheKey);

        if (cached) {
            return cached;
        }

        // Use seed to get deterministic word
        const seed = getDailySeed();

        try {
            // Try random-word-api with length parameter
            const response = await fetch(`https://random-word-api.herokuapp.com/word?length=6&number=10`);
            if (response.ok) {
                const words = await response.json();

                // Use seeded random to pick one deterministically
                const index = Math.floor(seededRandom(seed) * words.length);
                let word = words[index].toUpperCase();

                // Validate with dictionary API
                const validResponse = await fetch(`https://api.dictionaryapi.dev/api/v2/entries/en/${word.toLowerCase()}`);
                if (validResponse.ok) {
                    localStorage.setItem(cacheKey, word);
                    return word;
                }

                // If first pick invalid, try others
                for (let i = 0; i < words.length; i++) {
                    if (i === index) continue;
                    word = words[i].toUpperCase();
                    const check = await fetch(`https://api.dictionaryapi.dev/api/v2/entries/en/${word.toLowerCase()}`);
                    if (check.ok) {
                        localStorage.setItem(cacheKey, word);
                        return word;
                    }
                }
            }
        } catch (e) {
            console.error('Word API error:', e);
        }

        // Fallback: generate from seed (deterministic fallback list of common words)
        const fallbackWords = [
            'FRIEND', 'PLANET', 'GARDEN', 'BRIDGE', 'PURPLE', 'ORANGE', 'CASTLE', 'TEMPLE',
            'SHADOW', 'WINTER', 'SUMMER', 'SPRING', 'AUTUMN', 'FOREST', 'DESERT', 'STREAM',
            'COFFEE', 'BUTTER', 'CHEESE', 'COOKIE', 'MARKET', 'DOLLAR', 'TRAVEL', 'HEALTH',
            'BEAUTY', 'SILVER', 'GOLDEN', 'BRONZE', 'MARBLE', 'FABRIC', 'RIBBON', 'BUTTON'
        ];
        const fallbackIndex = Math.floor(seededRandom(seed) * fallbackWords.length);
        const fallbackWord = fallbackWords[fallbackIndex];
        localStorage.setItem(cacheKey, fallbackWord);
        return fallbackWord;
    }

    // Initialize
    async function init() {
        _word = await fetchDailyWord();
        _ready = true;
    }

    // Start init immediately
    _readyPromise = init();

    return {
        ready: function() {
            return _readyPromise;
        },

        reset: async function() {
            _gameOver = false;
            _word = await fetchDailyWord();
        },

        checkWin: function(guess) {
            return guess === _word;
        },

        getLetterFeedback: function(guess) {
            const result = Array(6).fill('absent');
            const targetLetters = _word.split('');
            const guessLetters = guess.split('');

            // First pass: correct positions
            for (let i = 0; i < 6; i++) {
                if (guessLetters[i] === targetLetters[i]) {
                    result[i] = 'correct';
                    targetLetters[i] = null;
                }
            }

            // Second pass: present but wrong position
            for (let i = 0; i < 6; i++) {
                if (result[i] === 'correct') continue;
                const idx = targetLetters.indexOf(guessLetters[i]);
                if (idx !== -1) {
                    result[i] = 'present';
                    targetLetters[idx] = null;
                }
            }

            return result;
        },

        setGameOver: function() {
            _gameOver = true;
        },

        revealWord: function() {
            if (_gameOver) return _word;
            return null;
        }
    };
})();
