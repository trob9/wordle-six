const WORD_LENGTH = 6;
const MAX_GUESSES = 6;

let targetWord = '';
let currentGuess = '';
let currentRow = 0;
let gameOver = false;
let gameState = null;

// Word validation cache
const validationCache = JSON.parse(localStorage.getItem('wordCache')) || {};

// Initialize game
function init() {
    loadGameState();
    createBoard();
    attachKeyboard();
    updateStatsDisplay();

    // Check if we need a new game (new day)
    const today = getDateString();
    if (!gameState || gameState.date !== today) {
        startNewGame();
    } else {
        // Restore existing game
        targetWord = getTodaysWord();
        currentRow = gameState.guesses.length;
        gameOver = gameState.gameOver;

        // Restore board
        gameState.guesses.forEach((guess, i) => {
            restoreGuess(i, guess);
        });

        // Restore keyboard
        gameState.guesses.forEach(guess => {
            const result = checkGuess(guess);
            updateKeyboard(guess, result);
        });

        if (gameOver) {
            if (gameState.won) {
                showMessage('🎉 You already won today!');
                showShareButton();
            } else {
                showMessage(`Today's word was ${targetWord}`);
            }
        }
    }

    // Update timer
    updateTimer();
    setInterval(updateTimer, 1000);
}

function startNewGame() {
    targetWord = getTodaysWord();
    currentGuess = '';
    currentRow = 0;
    gameOver = false;

    gameState = {
        date: getDateString(),
        guesses: [],
        gameOver: false,
        won: false
    };

    saveGameState();
    createBoard();
    resetKeyboard();
}

function createBoard() {
    const board = document.getElementById('board');
    board.innerHTML = '';

    for (let i = 0; i < MAX_GUESSES; i++) {
        const row = document.createElement('div');
        row.className = 'row';
        row.dataset.row = i;

        for (let j = 0; j < WORD_LENGTH; j++) {
            const tile = document.createElement('div');
            tile.className = 'tile';
            tile.id = `tile-${i}-${j}`;
            row.appendChild(tile);
        }

        board.appendChild(row);
    }
}

function restoreGuess(rowIndex, word) {
    const result = checkGuess(word);
    for (let i = 0; i < WORD_LENGTH; i++) {
        const tile = document.getElementById(`tile-${rowIndex}-${i}`);
        tile.textContent = word[i];
        tile.className = `tile ${result[i]}`;
    }
}

function attachKeyboard() {
    document.querySelectorAll('.key').forEach(key => {
        key.addEventListener('click', () => handleKey(key.dataset.key));
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') handleKey('ENTER');
        else if (e.key === 'Backspace') handleKey('BACKSPACE');
        else if (e.key.match(/^[a-zA-Z]$/)) handleKey(e.key.toUpperCase());
    });
}

function handleKey(key) {
    if (gameOver) return;

    if (key === 'ENTER') {
        if (currentGuess.length === WORD_LENGTH) {
            submitGuess();
        } else {
            showMessage('Not enough letters!');
            shakeRow();
        }
    } else if (key === 'BACKSPACE') {
        if (currentGuess.length > 0) {
            currentGuess = currentGuess.slice(0, -1);
            updateCurrentRow();
        }
    } else if (currentGuess.length < WORD_LENGTH) {
        currentGuess += key;
        updateCurrentRow();
    }
}

function updateCurrentRow() {
    for (let i = 0; i < WORD_LENGTH; i++) {
        const tile = document.getElementById(`tile-${currentRow}-${i}`);
        tile.textContent = currentGuess[i] || '';
        tile.className = currentGuess[i] ? 'tile filled' : 'tile';
    }
}

async function submitGuess() {
    const guess = currentGuess;

    // Show loading state
    showMessage('<span class="loading"></span>');

    // Validate word using API
    const isValid = await validateWord(guess);

    if (!isValid) {
        showMessage('Not a valid word!');
        shakeRow();
        return;
    }

    // Save guess
    gameState.guesses.push(guess);

    const result = checkGuess(guess);

    // Animate tiles
    for (let i = 0; i < WORD_LENGTH; i++) {
        const tile = document.getElementById(`tile-${currentRow}-${i}`);
        setTimeout(() => {
            tile.className = `tile ${result[i]}`;
        }, i * 200);
    }

    // Update keyboard
    setTimeout(() => {
        updateKeyboard(guess, result);
    }, WORD_LENGTH * 200);

    // Check win/lose
    if (guess === targetWord) {
        setTimeout(() => {
            gameOver = true;
            gameState.gameOver = true;
            gameState.won = true;
            saveGameState();
            updateStats(true, currentRow + 1);
            showMessage('🎉 Congratulations!');
            bounceRow();
            showShareButton();
        }, WORD_LENGTH * 200 + 500);
    } else if (currentRow === MAX_GUESSES - 1) {
        setTimeout(() => {
            gameOver = true;
            gameState.gameOver = true;
            gameState.won = false;
            saveGameState();
            updateStats(false);
            showMessage(`The word was ${targetWord}`);
        }, WORD_LENGTH * 200 + 500);
    }

    currentRow++;
    currentGuess = '';
    saveGameState();
}

async function validateWord(word) {
    // Check cache first
    if (validationCache[word] !== undefined) {
        return validationCache[word];
    }

    // Fast local check against dwordly word list
    if (VALID_WORDS.has(word)) {
        validationCache[word] = true;
        localStorage.setItem('wordCache', JSON.stringify(validationCache));
        return true;
    }

    // Not in local list - fallback to dictionary API for obscure words
    try {
        const response = await fetch(`https://api.dictionaryapi.dev/api/v2/entries/en/${word.toLowerCase()}`);
        const isValid = response.ok;

        // Cache the result
        validationCache[word] = isValid;
        localStorage.setItem('wordCache', JSON.stringify(validationCache));

        return isValid;
    } catch (error) {
        console.error('Validation error:', error);
        // If API fails, reject since it wasn't in local list either
        return false;
    }
}

function checkGuess(guess) {
    const result = Array(WORD_LENGTH).fill('absent');
    const targetLetters = targetWord.split('');
    const guessLetters = guess.split('');

    // First pass: mark correct letters
    for (let i = 0; i < WORD_LENGTH; i++) {
        if (guessLetters[i] === targetLetters[i]) {
            result[i] = 'correct';
            targetLetters[i] = null;
        }
    }

    // Second pass: mark present letters
    for (let i = 0; i < WORD_LENGTH; i++) {
        if (result[i] === 'correct') continue;

        const index = targetLetters.indexOf(guessLetters[i]);
        if (index !== -1) {
            result[i] = 'present';
            targetLetters[index] = null;
        }
    }

    return result;
}

function updateKeyboard(guess, result) {
    for (let i = 0; i < WORD_LENGTH; i++) {
        const key = document.querySelector(`[data-key="${guess[i]}"]`);
        if (!key) continue;

        const currentClass = key.className.split(' ').find(c =>
            c === 'correct' || c === 'present' || c === 'absent'
        );

        if (!currentClass ||
            (result[i] === 'correct') ||
            (result[i] === 'present' && currentClass !== 'correct')) {
            key.classList.remove('correct', 'present', 'absent');
            key.classList.add(result[i]);
        }
    }
}

function resetKeyboard() {
    document.querySelectorAll('.key').forEach(key => {
        key.classList.remove('correct', 'present', 'absent');
    });
}

function shakeRow() {
    const row = document.querySelector(`[data-row="${currentRow}"]`);
    row.classList.add('shake');
    setTimeout(() => row.classList.remove('shake'), 500);
}

function bounceRow() {
    const row = document.querySelector(`[data-row="${currentRow}"]`);
    row.classList.add('bounce');
    setTimeout(() => row.classList.remove('bounce'), 600);
}

function showMessage(msg) {
    const messageEl = document.getElementById('message');
    messageEl.innerHTML = msg;

    if (!msg.includes('loading')) {
        setTimeout(() => {
            if (messageEl.innerHTML === msg) messageEl.innerHTML = '';
        }, 3000);
    }
}

// Stats management
function loadGameState() {
    const saved = localStorage.getItem('gameState');
    if (saved) {
        gameState = JSON.parse(saved);
    }
}

function saveGameState() {
    localStorage.setItem('gameState', JSON.stringify(gameState));
}

function updateStats(won, guesses = null) {
    const stats = JSON.parse(localStorage.getItem('stats')) || {
        played: 0,
        won: 0,
        currentStreak: 0,
        maxStreak: 0,
        distribution: [0, 0, 0, 0, 0, 0],
        lastDate: null
    };

    stats.played++;

    if (won) {
        stats.won++;
        stats.distribution[guesses - 1]++;

        // Update streak
        const today = getDateString();
        const yesterday = new Date();
        yesterday.setDate(yesterday.getDate() - 1);
        const yesterdayStr = yesterday.toISOString().split('T')[0];

        if (stats.lastDate === yesterdayStr) {
            stats.currentStreak++;
        } else if (stats.lastDate !== today) {
            stats.currentStreak = 1;
        }

        stats.maxStreak = Math.max(stats.maxStreak, stats.currentStreak);
    } else {
        stats.currentStreak = 0;
    }

    stats.lastDate = getDateString();
    localStorage.setItem('stats', JSON.stringify(stats));
    updateStatsDisplay();
}

function updateStatsDisplay() {
    const stats = JSON.parse(localStorage.getItem('stats')) || {
        played: 0,
        won: 0,
        currentStreak: 0,
        maxStreak: 0,
        distribution: [0, 0, 0, 0, 0, 0]
    };

    document.getElementById('gamesPlayed').textContent = stats.played;
    document.getElementById('currentStreak').textContent = stats.currentStreak;
    document.getElementById('maxStreak').textContent = stats.maxStreak;

    const winRate = stats.played > 0 ? Math.round((stats.won / stats.played) * 100) : 0;
    document.getElementById('winRate').textContent = winRate;

    // Update distribution chart
    const chart = document.getElementById('distributionChart');
    chart.innerHTML = '';

    const maxCount = Math.max(...stats.distribution, 1);
    stats.distribution.forEach((count, i) => {
        const bar = document.createElement('div');
        bar.className = 'dist-bar';

        const label = document.createElement('div');
        label.className = 'dist-label';
        label.textContent = i + 1;

        const fill = document.createElement('div');
        fill.className = 'dist-bar-fill';
        const width = (count / maxCount) * 100;
        fill.style.width = `${Math.max(width, 7)}%`;
        fill.textContent = count;

        // Highlight current game's guess count
        if (gameState && gameState.won && gameState.guesses.length === i + 1) {
            fill.classList.add('current');
        }

        bar.appendChild(label);
        bar.appendChild(fill);
        chart.appendChild(bar);
    });
}

function showShareButton() {
    document.getElementById('shareSection').style.display = 'block';
}

function shareResults() {
    if (!gameState || !gameState.gameOver) return;

    const guessCount = gameState.won ? gameState.guesses.length : 'X';
    const emoji = gameState.guesses.map(guess => {
        const result = checkGuess(guess);
        return result.map(r => {
            if (r === 'correct') return '🟩';
            if (r === 'present') return '🟨';
            return '⬛';
        }).join('');
    }).join('\n');

    const text = `Wordle Six ${guessCount}/${MAX_GUESSES}\n\n${emoji}\n\nhttps://wordle-six.tomtom.fyi`;

    if (navigator.share) {
        navigator.share({ text });
    } else {
        navigator.clipboard.writeText(text);
        showMessage('Copied to clipboard!');
    }
}

// Modal functions
function showStats() {
    updateStatsDisplay();
    document.getElementById('statsModal').classList.add('show');
}

function showHelp() {
    document.getElementById('helpModal').classList.add('show');
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('show');
}

// Close modals on background click
document.querySelectorAll('.modal').forEach(modal => {
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            closeModal(modal.id);
        }
    });
});

// Timer update
function updateTimer() {
    const ms = getTimeUntilNextWord();
    const timeStr = formatTime(ms);
    const timer = document.getElementById('nextWordTimer');
    if (timer) {
        timer.textContent = `Next word in ${timeStr}`;
    }
}

// Initialize on load
document.addEventListener('DOMContentLoaded', init);
