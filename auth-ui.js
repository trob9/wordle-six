// Auth and Leaderboard UI
let currentUser = null;

(async function initAuth() {
    try {
        const resp = await fetch('/auth/me');
        const data = await resp.json();
        currentUser = data.user;
    } catch (e) {
        currentUser = null;
    }
    renderAuthUI();
    loadTopPlayers();
    if (currentUser && currentUser.banned) {
        showBannedScreen();
    } else {
        if (typeof initGame === 'function') initGame();
        if (currentUser && currentUser.is_new) {
            showWelcomeModal();
        }
    }
})();

function renderAuthUI() {
    const area = document.getElementById('authArea');
    if (!area) return;

    // Clear existing content
    area.textContent = '';

    if (currentUser) {
        const dropdown = document.createElement('div');
        dropdown.className = 'auth-dropdown';

        const btn = document.createElement('button');
        btn.className = 'icon-btn';
        btn.setAttribute('aria-label', 'Account');
        btn.style.cssText = 'overflow: hidden; padding: 0;';
        btn.addEventListener('click', toggleAuthMenu);

        if (currentUser.avatar_url) {
            const img = document.createElement('img');
            img.src = currentUser.avatar_url;
            img.alt = '';
            img.style.cssText = 'width: 100%; height: 100%; object-fit: cover; border-radius: 0.375rem;';
            btn.appendChild(img);
        }

        const menu = document.createElement('div');
        menu.className = 'auth-dropdown-menu';
        menu.id = 'authMenu';

        const info = document.createElement('div');
        info.style.cssText = 'padding: 0.5rem 0.75rem; font-size: 0.75rem; color: var(--text-muted); border-bottom: 1px solid var(--border-color); margin-bottom: 0.25rem;';
        info.textContent = 'Signed in as ';
        const strong = document.createElement('strong');
        strong.textContent = currentUser.display_name;
        info.appendChild(strong);

        const logoutBtn = document.createElement('button');
        logoutBtn.textContent = 'Sign out';
        logoutBtn.addEventListener('click', signOut);

        menu.appendChild(info);
        menu.appendChild(logoutBtn);
        dropdown.appendChild(btn);
        dropdown.appendChild(menu);
        area.appendChild(dropdown);
    } else {
        const dropdown = document.createElement('div');
        dropdown.className = 'auth-dropdown';

        const btn = document.createElement('button');
        btn.className = 'icon-btn';
        btn.setAttribute('aria-label', 'Sign in');
        btn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>';
        btn.addEventListener('click', toggleAuthMenu);

        const menu = document.createElement('div');
        menu.className = 'auth-dropdown-menu';
        menu.id = 'authMenu';

        const providers = [
            { name: 'GitHub', path: '/auth/github', icon: 'github' },
            { name: 'Discord', path: '/auth/discord', icon: 'discord' },
            { name: 'Google', path: '/auth/google', icon: 'google' }
        ];

        providers.forEach(p => {
            const provBtn = document.createElement('button');
            const icon = document.createElement('span');
            icon.className = 'provider-icon';
            icon.textContent = p.name[0]; // Simple fallback letter icon
            provBtn.appendChild(icon);
            const text = document.createTextNode(' ' + p.name);
            provBtn.appendChild(text);
            provBtn.addEventListener('click', () => { window.location.href = p.path; });
            menu.appendChild(provBtn);
        });

        dropdown.appendChild(btn);
        dropdown.appendChild(menu);
        area.appendChild(dropdown);
    }
}

function toggleAuthMenu() {
    const menu = document.getElementById('authMenu');
    if (menu) menu.classList.toggle('show');
}

// Close auth menu on outside click
document.addEventListener('click', (e) => {
    const menu = document.getElementById('authMenu');
    if (menu && menu.classList.contains('show')) {
        const dropdown = menu.closest('.auth-dropdown');
        if (!dropdown.contains(e.target)) {
            menu.classList.remove('show');
        }
    }
});

async function signOut() {
    await fetch('/auth/logout', { method: 'POST' });
    currentUser = null;
    // Clear game state so next account doesn't see stale data
    localStorage.removeItem('gameState');
    localStorage.removeItem('stats');
    localStorage.removeItem('hardMode');
    window.location.reload();
}

// Leaderboard
async function loadTopPlayers() {
    try {
        const resp = await fetch('/api/leaderboard?limit=3');
        const data = await resp.json();
        renderTopPlayers(data.leaderboard);
    } catch (e) {
        // Silently fail - leaderboard is optional
    }
}

function renderTopPlayers(entries) {
    const el = document.getElementById('topPlayers');
    if (!el || !entries || entries.length === 0) return;

    el.textContent = '';
    const rankClasses = ['gold', 'silver', 'bronze'];

    entries.forEach((entry, i) => {
        const div = document.createElement('div');
        div.className = 'top-player';

        const rank = document.createElement('span');
        rank.className = 'top-player-rank ' + rankClasses[i];
        rank.textContent = i + 1;
        div.appendChild(rank);

        if (entry.avatar_url) {
            const img = document.createElement('img');
            img.className = 'top-player-avatar';
            img.src = entry.avatar_url;
            img.alt = '';
            div.appendChild(img);
        }

        const name = document.createElement('span');
        name.className = 'top-player-name';
        name.textContent = entry.display_name;
        div.appendChild(name);

        const avg = document.createElement('span');
        avg.className = 'top-player-avg';
        avg.textContent = entry.true_avg.toFixed(2);
        div.appendChild(avg);

        el.appendChild(div);
    });

    el.style.display = 'flex';
}

async function showLeaderboard() {
    document.getElementById('leaderboardModal').classList.add('show');
    const list = document.getElementById('leaderboardList');
    list.textContent = 'Loading...';
    list.style.textAlign = 'center';
    list.style.color = 'var(--text-muted)';
    list.style.padding = '2rem 0';

    try {
        const resp = await fetch('/api/leaderboard');
        const data = await resp.json();
        renderLeaderboard(data.leaderboard);
    } catch (e) {
        list.textContent = 'Failed to load leaderboard';
    }
}

function renderLeaderboard(entries) {
    const list = document.getElementById('leaderboardList');
    list.textContent = '';
    list.style.textAlign = '';
    list.style.color = '';
    list.style.padding = '';

    if (!entries || entries.length === 0) {
        list.textContent = 'No players yet';
        list.style.textAlign = 'center';
        list.style.color = 'var(--text-muted)';
        list.style.padding = '2rem 0';
        return;
    }

    const rankClasses = { 1: 'gold', 2: 'silver', 3: 'bronze' };

    entries.forEach(e => {
        const row = document.createElement('div');
        row.className = 'lb-row';

        const rankDiv = document.createElement('div');
        rankDiv.className = 'lb-rank ' + (rankClasses[e.rank] || '');
        rankDiv.textContent = e.rank;
        row.appendChild(rankDiv);

        if (e.avatar_url) {
            const img = document.createElement('img');
            img.className = 'lb-avatar';
            img.src = e.avatar_url;
            img.alt = '';
            row.appendChild(img);
        } else {
            const placeholder = document.createElement('div');
            placeholder.className = 'lb-avatar';
            placeholder.style.cssText = 'background: var(--bg-secondary); display: flex; align-items: center; justify-content: center; font-size: 0.7rem; font-weight: 600;';
            placeholder.textContent = e.display_name[0];
            row.appendChild(placeholder);
        }

        const nameDiv = document.createElement('div');
        nameDiv.className = 'lb-name';
        nameDiv.textContent = e.display_name;
        if (e.rank <= 3) {
            const trophy = document.createElement('span');
            trophy.className = 'lb-trophy lb-trophy-' + (rankClasses[e.rank] || '');
            const ns = 'http://www.w3.org/2000/svg';
            const svg = document.createElementNS(ns, 'svg');
            svg.setAttribute('width', '14');
            svg.setAttribute('height', '14');
            svg.setAttribute('viewBox', '0 0 24 24');
            svg.setAttribute('fill', 'currentColor');
            ['M6 9V2h12v7a6 6 0 0 1-12 0Z',
             'M6 4H4.5a2.5 2.5 0 0 0 0 5H6',
             'M18 4h1.5a2.5 2.5 0 0 1 0 5H18',
             'M10 15v2c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22h10c0-1.76-.85-3.25-2.03-3.79-.5-.23-.97-.66-.97-1.21v-2'
            ].forEach(d => {
                const path = document.createElementNS(ns, 'path');
                path.setAttribute('d', d);
                svg.appendChild(path);
            });
            trophy.appendChild(svg);
            nameDiv.appendChild(trophy);
        }
        row.appendChild(nameDiv);

        const statsDiv = document.createElement('div');
        statsDiv.className = 'lb-stats';

        [
            { value: e.true_avg.toFixed(2), label: 'avg' },
            { value: Math.round(e.win_rate * 100) + '%', label: 'win' },
            { value: e.games_played, label: 'games' }
        ].forEach(s => {
            const stat = document.createElement('div');
            stat.className = 'lb-stat';
            const val = document.createElement('span');
            val.className = 'lb-stat-value';
            val.textContent = s.value;
            stat.appendChild(val);
            stat.appendChild(document.createTextNode(' ' + s.label));
            statsDiv.appendChild(stat);
        });

        row.appendChild(statsDiv);
        list.appendChild(row);
    });
}

// Ban screen
function showBannedScreen() {
    const gameArea = document.querySelector('.game-area');
    if (gameArea) {
        gameArea.textContent = '';
        const msg = document.createElement('div');
        msg.style.cssText = 'text-align: center; padding: 3rem 1rem;';
        const title = document.createElement('div');
        title.style.cssText = 'font-family: "Cormorant Garamond", Georgia, serif; font-size: 1.5rem; font-weight: 600; margin-bottom: 0.75rem; color: var(--text-primary);';
        title.textContent = 'Account Suspended';
        const desc = document.createElement('p');
        desc.style.cssText = 'font-size: 0.9rem; color: var(--text-secondary); line-height: 1.6;';
        desc.textContent = 'Your account has been banned from Wordle Six. If you believe this is an error, please contact the administrator.';
        msg.appendChild(title);
        msg.appendChild(desc);
        gameArea.appendChild(msg);
    }
}

// Display name management
function showWelcomeModal() {
    const input = document.getElementById('welcomeNameInput');
    if (input && currentUser) {
        input.value = currentUser.display_name || '';
    }
    document.getElementById('welcomeModal').classList.add('show');
}

async function saveWelcomeName() {
    const input = document.getElementById('welcomeNameInput');
    const error = document.getElementById('welcomeNameError');
    const name = input.value.trim();

    if (!name || name.length > 20) {
        error.textContent = 'Name must be 1-20 characters';
        error.style.display = 'block';
        return;
    }

    try {
        const resp = await fetch('/api/display-name', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name })
        });
        if (!resp.ok) {
            const errText = await resp.text();
            error.textContent = errText.trim() || 'Failed to save name';
            error.style.display = 'block';
            return;
        }
        currentUser.display_name = name;
        currentUser.is_new = false;
        renderAuthUI();
        closeModal('welcomeModal');
    } catch (e) {
        error.textContent = 'Failed to save name';
        error.style.display = 'block';
    }
}

async function saveDisplayName() {
    const input = document.getElementById('displayNameInput');
    const btn = document.getElementById('displayNameSave');
    const name = input.value.trim();

    if (!name || name.length > 20) return;

    btn.textContent = '...';
    try {
        const resp = await fetch('/api/display-name', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name })
        });
        if (!resp.ok) {
            const errText = await resp.text();
            const errEl = document.getElementById('displayNameError');
            errEl.textContent = errText.trim() || 'Error';
            errEl.style.display = '';
            btn.textContent = 'Save';
            setTimeout(() => { errEl.style.display = 'none'; }, 3000);
            return;
        }
        currentUser.display_name = name;
        renderAuthUI();
        loadTopPlayers();
        btn.textContent = 'Saved';
        setTimeout(() => { btn.textContent = 'Save'; }, 1500);
    } catch (e) {
        btn.textContent = 'Error';
        setTimeout(() => { btn.textContent = 'Save'; }, 1500);
    }
}

function updateSettingsForAuth() {
    const setting = document.getElementById('displayNameSetting');
    if (!setting) return;
    if (currentUser) {
        setting.style.display = '';
        const input = document.getElementById('displayNameInput');
        if (input) input.value = currentUser.display_name || '';
    } else {
        setting.style.display = 'none';
    }
}

// Submit result to API after game ends
async function submitResultToAPI(won, guesses, hardModeActive) {
    if (!currentUser) return;
    try {
        const resp = await fetch('/api/result', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                date: getDateString(),
                won: won,
                guesses: won ? guesses : null,
                hard_mode: hardModeActive,
                client_time: new Date().toISOString(),
                tz_offset: new Date().getTimezoneOffset()
            })
        });
        const data = await resp.json();
        if (data.tz_warning && typeof showTzWarning === 'function') showTzWarning();
        loadTopPlayers();
    } catch (e) {
        // Silent fail
    }
}
