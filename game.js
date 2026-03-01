const WORD_LENGTH = 6;
const MAX_GUESSES = 6;

let targetWord = '';
let currentGuess = '';
let currentRow = 0;
let gameOver = false;
let gameState = null;
let hardMode = false;

// --- Browser fingerprinting ---
// Collects everything the browser exposes for session correlation.
// The hash is included in every guess event so anon and logged-in sessions
// from the same device can be linked even across login boundaries.

async function collectFingerprint() {
    // Canvas fingerprint: rendering differences reveal GPU/font/OS combination.
    let canvas_fp = '';
    try {
        const c = document.createElement('canvas');
        c.width = 240; c.height = 50;
        const ctx = c.getContext('2d');
        ctx.textBaseline = 'alphabetic';
        ctx.fillStyle = '#f0f';
        ctx.fillRect(0, 0, 240, 50);
        ctx.font = '15px Arial';
        ctx.fillStyle = '#069';
        ctx.fillText('Wordle Six \uD83C\uDF10 <canvas>', 2, 20);
        ctx.font = '11px "Times New Roman"';
        ctx.fillStyle = 'rgba(102,204,0,0.85)';
        ctx.fillText('fingerprint \u2764 1234567890', 4, 38);
        canvas_fp = c.toDataURL().slice(-48);
    } catch (e) {}

    // WebGL: exposes GPU vendor/renderer string — very stable per device.
    let webgl_vendor = '', webgl_renderer = '';
    try {
        const gl = document.createElement('canvas').getContext('webgl') ||
                   document.createElement('canvas').getContext('experimental-webgl');
        if (gl) {
            const ext = gl.getExtension('WEBGL_debug_renderer_info');
            if (ext) {
                webgl_vendor   = gl.getParameter(ext.UNMASKED_VENDOR_WEBGL)   || '';
                webgl_renderer = gl.getParameter(ext.UNMASKED_RENDERER_WEBGL) || '';
            } else {
                webgl_vendor   = gl.getParameter(gl.VENDOR)   || '';
                webgl_renderer = gl.getParameter(gl.RENDERER) || '';
            }
        }
    } catch (e) {}

    const fp = {
        // Locale & timezone
        lang:         navigator.language       || '',
        langs:        (navigator.languages     || []).join(','),
        tz:           Intl.DateTimeFormat().resolvedOptions().timeZone || '',
        tz_offset:    new Date().getTimezoneOffset(),
        // Screen & display
        screen_w:     screen.width,
        screen_h:     screen.height,
        avail_w:      screen.availWidth,
        avail_h:      screen.availHeight,
        color_depth:  screen.colorDepth,
        dpr:          window.devicePixelRatio || 1,
        // Hardware hints
        hw_concurrency: navigator.hardwareConcurrency || 0,
        device_memory:  navigator.deviceMemory        || 0,
        // Platform & input
        platform:     navigator.platform      || '',
        touch_points: navigator.maxTouchPoints || 0,
        // Browser settings
        do_not_track:    navigator.doNotTrack   || '',
        cookies_enabled: navigator.cookieEnabled,
        // UA (also present in server-side HTTP headers, useful for cross-check)
        ua: navigator.userAgent,
        // Rendering fingerprints
        canvas_fp,
        webgl_vendor,
        webgl_renderer,
    };

    // A short stable hash over fields that are constant for a given device/browser.
    // Deliberately excludes anything that changes with window state (avail_w/h, dpr)
    // or user action, so the hash stays the same across a session even if the
    // window is resized or moved between monitors.
    const hashInput = [
        fp.ua, fp.screen_w, fp.screen_h, fp.color_depth,
        fp.hw_concurrency, fp.device_memory, fp.tz, fp.platform,
        fp.touch_points, fp.canvas_fp, fp.webgl_renderer
    ].join('|');
    fp.hash = hashInput.split('').reduce((h, ch) => (Math.imul(31, h) + ch.charCodeAt(0)) | 0, 0)
                       .toString(16).replace('-', 'n');

    return fp;
}

async function sendFingerprint() {
    try {
        const fp = await collectFingerprint();
        sessionStorage.setItem('fp_hash', fp.hash);
        fetch('/api/fingerprint', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(fp)
        }).catch(() => {});
    } catch (e) {}
}

// Initialize game
async function initGame() {
    loadGameState();
    loadHardMode();
    createBoard();
    attachKeyboard();
    updateStatsDisplay();
    sendFingerprint(); // fire-and-forget; also stashes fp_hash in sessionStorage

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
                showMessage('You already solved today\'s puzzle', true);
                showShareButton();
            } else {
                // Show the answer persistently for returning users who lost
                showMessage(`The word was ${targetWord}`, true);
                showShareButton();
            }
        }
    }

    // If signed in, server state is authoritative — replace local state
    if (typeof currentUser !== 'undefined' && currentUser) {
        await syncStatsFromServer();
        try {
            const resp = await fetch(`/api/game-state?date=${getDateString()}`);
            if (resp.ok) {
                const serverState = await resp.json();
                // Always apply server state for logged-in users
                const serverGuesses = serverState.guesses || [];
                gameState.guesses = serverGuesses;
                gameState.gameOver = serverState.gameOver || false;
                gameState.won = serverState.won || false;
                // Hard mode preference is owned by user_stats (loaded in syncStatsFromServer above).
                // Never read it from game-state — that would create a second source of truth.
                currentRow = gameState.guesses.length;
                gameOver = gameState.gameOver;
                targetWord = getTodaysWord();
                saveGameState();

                // Re-render board
                createBoard();
                gameState.guesses.forEach((guess, i) => {
                    restoreGuess(i, guess);
                });
                resetKeyboard();
                gameState.guesses.forEach(guess => {
                    const result = checkGuess(guess);
                    updateKeyboard(guess, result);
                });

                // Update hard mode toggle UI
                const toggle = document.getElementById('hardModeToggle');
                if (toggle) toggle.classList.toggle('active', hardMode);

                if (gameOver) {
                    if (gameState.won) {
                        showMessage('You already solved today\'s puzzle', true);
                        showShareButton();
                    } else {
                        showMessage(`The word was ${targetWord}`, true);
                        showShareButton();
                    }
                }
            }
        } catch (e) {
            // Silent fail — localStorage state is the fallback
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
    board.textContent = '';

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
        // Don't capture keys when an input or modal is focused
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
        if (document.querySelector('.modal.show')) return;
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

function submitGuess() {
    const guess = currentGuess;

    // Validate word against local dictionary
    if (!VALID_WORDS.has(guess)) {
        showMessage('Not a valid word!');
        shakeRow();
        return;
    }

    // Check hard mode constraints
    const hardCheck = checkHardMode(guess);
    if (!hardCheck.valid) {
        showMessage(hardCheck.message);
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
    const guessNumber = currentRow + 1; // Capture before currentRow gets incremented
    if (guess === targetWord) {
        setTimeout(() => {
            gameOver = true;
            gameState.gameOver = true;
            gameState.won = true;
            saveGameState();
            saveProgressToServer();
            updateStats(true, guessNumber);
            bounceRow();
            showShareButton();
            showWinModal(guessNumber);
            if (typeof submitResultToAPI === 'function') submitResultToAPI(true, guessNumber, hardMode);
        }, WORD_LENGTH * 200 + 500);
    } else if (currentRow === MAX_GUESSES - 1) {
        setTimeout(() => {
            gameOver = true;
            gameState.gameOver = true;
            gameState.won = false;
            saveGameState();
            saveProgressToServer();
            updateStats(false);
            showShareButton();
            showLossModal(targetWord);
            if (typeof submitResultToAPI === 'function') submitResultToAPI(false, null, hardMode);
        }, WORD_LENGTH * 200 + 500);
    }

    currentRow++;
    currentGuess = '';
    saveGameState();
    saveProgressToServer();
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

function showMessage(msg, persistent = false) {
    const messageEl = document.getElementById('message');
    messageEl.textContent = msg;

    if (!msg.includes('loading') && !persistent) {
        setTimeout(() => {
            if (messageEl.textContent === msg) messageEl.textContent = '';
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

let tzWarningShown = false;

function saveProgressToServer() {
    if (!gameState) return;
    fetch('/api/save-progress', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            date: gameState.date,
            guesses: gameState.guesses,
            hardMode: hardMode,
            gameOver: gameState.gameOver,
            won: gameState.won,
            client_time: new Date().toISOString(),
            tz_offset: new Date().getTimezoneOffset(),
            fp_hash: sessionStorage.getItem('fp_hash') || ''
        })
    }).then(r => r.json()).then(data => {
        if (data.tz_warning) showTzWarning();
    }).catch(() => {});
}

function showTzWarning() {
    if (tzWarningShown) return;
    tzWarningShown = true;
    showMessage('Timezone manipulation detected and logged for review.', true);
}

async function syncStatsFromServer() {
    try {
        const resp = await fetch('/api/user-stats');
        if (!resp.ok) return;
        const serverStats = await resp.json();

        // Load hard mode preference from server
        if (serverStats.hardMode !== undefined) {
            hardMode = serverStats.hardMode;
            localStorage.setItem('hardMode', hardMode);
            const toggle = document.getElementById('hardModeToggle');
            if (toggle) toggle.classList.toggle('active', hardMode);
        }

        // Server stats are authoritative — use them if any games recorded
        if (serverStats.played > 0) {
            const stats = {
                played: serverStats.played,
                won: serverStats.won,
                playedHard: serverStats.playedHard || 0,
                wonHard: serverStats.wonHard || 0,
                currentStreak: serverStats.currentStreak,
                maxStreak: serverStats.maxStreak,
                distribution: serverStats.distribution || [0, 0, 0, 0, 0, 0],
                lastDate: serverStats.lastDate || null
            };
            localStorage.setItem('stats', JSON.stringify(stats));
            updateStatsDisplay();
        }
    } catch (e) {
        // Silent fail
    }
}

function saveStatsToServer() {
    if (typeof currentUser === 'undefined' || !currentUser) return;
    const stats = JSON.parse(localStorage.getItem('stats'));
    if (!stats) return;
    fetch('/api/user-stats', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            played: stats.played,
            won: stats.won,
            playedHard: stats.playedHard || 0,
            wonHard: stats.wonHard || 0,
            currentStreak: stats.currentStreak,
            maxStreak: stats.maxStreak,
            distribution: stats.distribution,
            lastDate: stats.lastDate || '',
            hardMode: hardMode
        })
    }).catch(() => {});
}

function getDefaultStats() {
    return {
        played: 0, won: 0, playedHard: 0, wonHard: 0,
        currentStreak: 0, maxStreak: 0,
        distribution: [0, 0, 0, 0, 0, 0], lastDate: null
    };
}

function updateStats(won, guesses = null) {
    const stats = JSON.parse(localStorage.getItem('stats')) || getDefaultStats();
    if (stats.playedHard === undefined) { stats.playedHard = 0; stats.wonHard = 0; }

    stats.played++;
    if (hardMode) stats.playedHard++;

    if (won) {
        stats.won++;
        if (hardMode) stats.wonHard++;
        stats.distribution[guesses - 1]++;

        // Update streak
        const today = getDateString();
        // Compute yesterday using local date arithmetic (same method as getDateString)
        // to avoid timezone mismatch with toISOString() which is always UTC.
        const yd = new Date();
        yd.setDate(yd.getDate() - 1);
        const yesterdayStr = `${yd.getFullYear()}-${String(yd.getMonth() + 1).padStart(2, '0')}-${String(yd.getDate()).padStart(2, '0')}`;

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
    saveStatsToServer();
}

function updateStatsDisplay() {
    const stats = JSON.parse(localStorage.getItem('stats')) || getDefaultStats();
    if (stats.playedHard === undefined) { stats.playedHard = 0; stats.wonHard = 0; }

    const playedEl = document.getElementById('gamesPlayed');
    playedEl.textContent = stats.played;
    if (stats.playedHard > 0) {
        const sub = document.createElement('span');
        sub.className = 'stat-hard-sub';
        sub.textContent = ` (${stats.playedHard}H)`;
        playedEl.appendChild(sub);
    }

    document.getElementById('currentStreak').textContent = stats.currentStreak;
    document.getElementById('maxStreak').textContent = stats.maxStreak;

    const gamesWonEl = document.getElementById('gamesWon');
    gamesWonEl.textContent = stats.won;
    if (stats.wonHard > 0) {
        const sub = document.createElement('span');
        sub.className = 'stat-hard-sub';
        sub.textContent = ` (${stats.wonHard}H)`;
        gamesWonEl.appendChild(sub);
    }

    // Update distribution chart
    const chart = document.getElementById('distributionChart');
    chart.textContent = '';

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

function shareResults(button) {
    if (!gameState || !gameState.gameOver) return;

    const guessCount = gameState.won ? gameState.guesses.length : 'X';
    const hardIndicator = hardMode ? ' (Hard Mode)' : '';
    const emoji = gameState.guesses.map(guess => {
        const result = checkGuess(guess);
        return result.map(r => {
            if (r === 'correct') return '🟩';
            if (r === 'present') return '🟨';
            return '⬛';
        }).join('');
    }).join('\n');

    const text = `Wordle Six ${guessCount}/${MAX_GUESSES}${hardIndicator}\n\n${emoji}\n\nhttps://wordle.xxoo.ooo`;

    // Only use native share on mobile (touch devices), clipboard everywhere else
    const isMobile = 'ontouchstart' in window || navigator.maxTouchPoints > 0;

    if (isMobile && navigator.share) {
        navigator.share({ text });
    } else {
        navigator.clipboard.writeText(text);
        // Change button text to show feedback
        if (button) {
            const originalText = button.textContent;
            button.textContent = 'Copied!';
            setTimeout(() => {
                button.textContent = originalText;
            }, 2000);
        }
    }
}

// Hard mode functions
function loadHardMode() {
    hardMode = localStorage.getItem('hardMode') === 'true';
    const toggle = document.getElementById('hardModeToggle');
    if (toggle) {
        toggle.classList.toggle('active', hardMode);
    }
}

function toggleHardMode() {
    if (gameOver) return;

    if (!hardMode && gameState && gameState.guesses.length > 0) {
        // Trying to enable — check if all existing guesses are hard-mode-valid
        const check = canEnableHardMode();
        if (!check.valid) {
            const warning = document.getElementById('hardModeWarning');
            if (warning) {
                warning.textContent = check.message;
                warning.style.display = 'block';
            }
            return;
        }
    }

    // Clear warning on successful toggle
    const warning = document.getElementById('hardModeWarning');
    if (warning) warning.style.display = 'none';

    hardMode = !hardMode;
    localStorage.setItem('hardMode', hardMode);
    const toggle = document.getElementById('hardModeToggle');
    if (toggle) {
        toggle.classList.toggle('active', hardMode);
    }
    saveProgressToServer();
    saveStatsToServer();
}

function canEnableHardMode() {
    if (!gameState || gameState.guesses.length <= 1) return { valid: true };

    const allGuesses = [...gameState.guesses];
    const savedGuesses = gameState.guesses;
    const savedHardMode = hardMode;

    hardMode = true;
    for (let i = 1; i < allGuesses.length; i++) {
        gameState.guesses = allGuesses.slice(0, i);
        const result = checkHardMode(allGuesses[i]);
        if (!result.valid) {
            gameState.guesses = savedGuesses;
            hardMode = savedHardMode;
            return { valid: false, message: `Can't enable — previous guesses don't meet hard mode rules` };
        }
    }
    gameState.guesses = savedGuesses;
    hardMode = savedHardMode;
    return { valid: true };
}

function updateHardModeUI() {
    const toggle = document.getElementById('hardModeToggle');
    const warning = document.getElementById('hardModeWarning');
    const disabled = gameOver;

    if (toggle) {
        toggle.disabled = disabled;
        toggle.style.opacity = disabled ? '0.4' : '';
        toggle.style.cursor = disabled ? 'not-allowed' : '';
    }
    if (warning) {
        // Hide warning when opening settings fresh; it gets shown dynamically on error
        warning.style.display = 'none';
    }
}

function checkHardMode(guess) {
    if (!hardMode || !gameState || gameState.guesses.length === 0) {
        return { valid: true };
    }

    // Build up known constraints from all previous guesses
    const requiredPositions = {}; // position -> letter (green)
    const requiredLetters = new Set(); // must be in guess (yellow)
    const absentLetters = new Set(); // confirmed not in word
    const excludedPositions = {}; // position -> Set of letters that can't be there (yellow)

    for (const prev of gameState.guesses) {
        const result = checkGuess(prev);
        const prevLetters = prev.split('');

        // First collect greens and yellows
        for (let i = 0; i < WORD_LENGTH; i++) {
            if (result[i] === 'correct') {
                requiredPositions[i] = prevLetters[i];
                requiredLetters.add(prevLetters[i]);
            } else if (result[i] === 'present') {
                requiredLetters.add(prevLetters[i]);
                // Letter is in the word but NOT at this position
                if (!excludedPositions[i]) excludedPositions[i] = new Set();
                excludedPositions[i].add(prevLetters[i]);
            }
        }

        // Then collect absent letters (only if not also green/yellow elsewhere)
        for (let i = 0; i < WORD_LENGTH; i++) {
            if (result[i] === 'absent' && !requiredLetters.has(prevLetters[i])) {
                absentLetters.add(prevLetters[i]);
            }
        }
    }

    // Check green positions
    for (const [pos, letter] of Object.entries(requiredPositions)) {
        if (guess[pos] !== letter) {
            return { valid: false, message: 'Correct letters must remain in place' };
        }
    }

    // Check yellow letters aren't in their excluded positions
    for (const [pos, letters] of Object.entries(excludedPositions)) {
        if (letters.has(guess[pos])) {
            return { valid: false, message: 'Try revealed letters in a new spot' };
        }
    }

    // Check yellow letters are present
    for (const letter of requiredLetters) {
        if (!guess.includes(letter)) {
            return { valid: false, message: 'Guess must use all revealed letters' };
        }
    }

    // Check absent letters are not used
    for (let i = 0; i < WORD_LENGTH; i++) {
        if (absentLetters.has(guess[i])) {
            return { valid: false, message: 'Cannot use eliminated letters' };
        }
    }

    return { valid: true };
}

function showSettings() {
    updateHardModeUI();
    if (typeof updateSettingsForAuth === 'function') updateSettingsForAuth();
    document.getElementById('settingsModal').classList.add('show');
}

// Modal functions
function showStats() {
    updateStatsDisplay();
    document.getElementById('statsModal').classList.add('show');
}

function showHelp() {
    document.getElementById('helpModal').classList.add('show');
}

function showWinModal(guessCount) {
    document.getElementById('winGuessCount').textContent = guessCount;
    document.getElementById('guessWord').textContent = guessCount === 1 ? 'guess' : 'guesses';
    document.getElementById('winModal').classList.add('show');
}

function showLossModal(word) {
    document.getElementById('answerReveal').textContent = word;
    document.getElementById('lossModal').classList.add('show');
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
