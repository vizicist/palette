// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    await loadPresets();
    setupControls();
});

async function loadPresets() {
    const grid = document.getElementById('preset-grid');
    grid.innerHTML = '<div class="loading">Loading...</div>';

    try {
        const list = await API.getSavedList('quad');

        // Handle both array and newline-separated string formats
        let presets;
        if (Array.isArray(list)) {
            // Strip category prefix (e.g., "quad.All_Moire" -> "All_Moire")
            presets = list.map(p => {
                const dotIdx = p.indexOf('.');
                return dotIdx >= 0 ? p.substring(dotIdx + 1) : p;
            });
        } else {
            presets = list.split('\n');
        }

        // Trim, remove carriage returns, and filter out empty, underscore-prefixed, and curly-brace-prefixed presets
        presets = presets.map(p => p.trim().replace(/\r/g, '')).filter(p => p && !p.startsWith('_') && !p.startsWith('{'));

        // Sort alphabetically (case-insensitive)
        presets.sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()));

        if (presets.length === 0) {
            grid.innerHTML = '<div class="loading">No presets found</div>';
            return;
        }

        grid.innerHTML = presets.map(name => {
            const display = name.replace(/_/g, '<br>').replace(/^(<br>)+/, '').replace(/(<br>)+$/, '');
            return `<button class="preset-btn" data-name="${name}">${display}</button>`;
        }).join('');

        grid.querySelectorAll('.preset-btn').forEach(btn => {
            btn.addEventListener('click', () => loadPreset(btn.dataset.name));
        });
    } catch (e) {
        grid.innerHTML = `<div class="error">${e.message}</div>`;
    }
}

async function loadPreset(name) {
    try {
        await API.loadQuad(name);
    } catch (e) {
        alert('Load failed: ' + e.message);
    }
}

function setupControls() {
    document.getElementById('btn-complete-reset').addEventListener('click', async () => {
        try {
            await API.completeReset();
        } catch (e) {
            alert('Reset failed: ' + e.message);
        }
    });

    document.getElementById('btn-soft-reset').addEventListener('click', async () => {
        try {
            await API.softReset();
        } catch (e) {
            alert('Reset failed: ' + e.message);
        }
    });

    document.getElementById('btn-help').addEventListener('click', () => {
        alert('Space Palette Pro\n\nClick a preset to load it.');
    });
}
