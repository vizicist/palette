// State
let currentPatch = '*';
let currentCategory = 'quad';
let advancedMode = false;
let lastSinglePatch = 'A'; // Track last single selection for toggle
let showingParams = false; // Toggle between presets and parameters view
let activeAdventure = null;

// Cached parameter definitions and enums for string param dropdowns
let cachedParamDefs = null;
let cachedParamEnums = null;

// Attract mode state
let attractModeActive = false;
let attractAllowGui = false;

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('touchscreen') === '1') {
        document.body.classList.add('touchscreen-embed');
    }

    // Check guidefaultlevel to determine initial mode
    try {
        const level = await API.call('global.get', { name: 'global.guidefaultlevel' });
        if (level === '1' || level === 1) {
            advancedMode = true;
        }
    } catch (e) { /* default to normal mode */ }

    // Check if attract GUI display is allowed
    try {
        const val = await API.call('global.get', { name: 'global.attractallowgui' });
        attractAllowGui = val === 'true' || val === true;
    } catch (e) { /* default to false */ }

    setupAdventureScreen();
    setupControls();
    setupHelpOverlay();
    setupAttractOverlay();
    setupCategoryTabs();
    setupPatchSelector();
    setupSigilSequencer();
    showAdventureScreen();

    // Hide the Record button until we confirm OBS is reachable.
    updateRecordButtonVisibility();
    setInterval(updateRecordButtonVisibility, 10000);

    // Start polling engine status every 2 seconds
    setInterval(pollStatus, 2000);
    setInterval(refreshStepperStatus, 250);
});

function setupAdventureScreen() {
    document.getElementById('btn-adventure-space').addEventListener('click', startSpacePalette);
    document.getElementById('btn-adventure-sigil').addEventListener('click', showSigilSequencer);
    document.getElementById('btn-sigil-start-over-bottom').addEventListener('click', showAdventureScreen);
}

async function startSpacePalette() {
    activeAdventure = 'space';
    document.getElementById('adventure-screen').classList.add('hidden');
    document.getElementById('sigil-screen').classList.add('hidden');
    document.getElementById('main-container').classList.remove('hidden');
    setAdvancedMode(advancedMode, false);
    await loadPresets();
    await pollStatus();
}

async function showAdventureScreen() {
    activeAdventure = null;
    hideAttract();
    hideHelp();
    document.getElementById('adventure-screen').classList.remove('hidden');
    document.getElementById('sigil-screen').classList.add('hidden');
    document.getElementById('main-container').classList.add('hidden');
    document.getElementById('title-bar').classList.add('hidden');
    document.getElementById('category-tabs').classList.add('hidden');
    document.getElementById('patch-selector').classList.add('hidden');
}

async function showSigilSequencer() {
    activeAdventure = 'sigil';
    hideAttract();
    hideHelp();
    document.getElementById('adventure-screen').classList.add('hidden');
    document.getElementById('main-container').classList.add('hidden');
    document.getElementById('title-bar').classList.add('hidden');
    document.getElementById('category-tabs').classList.add('hidden');
    document.getElementById('patch-selector').classList.add('hidden');
    document.getElementById('sigil-screen').classList.remove('hidden');
    await refreshStepperStatus();
}

async function updateRecordButtonVisibility() {
    const btn = document.getElementById('btn-record');
    if (!btn) return;
    let running = false;
    try {
        const result = await API.obsPing();
        running = !!(result && result.running);
    } catch (e) { /* treat errors as "not running" */ }
    btn.style.display = running ? '' : 'none';
}

async function loadPresets() {
    const grid = document.getElementById('preset-grid');
    grid.classList.add('grid-mode');
    grid.innerHTML = '<div class="loading">Loading...</div>';

    try {
        const list = await API.getSavedList(currentCategory);

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

        const html = presets.map(name => {
            const display = name.replace(/_/g, '<br>').replace(/^(<br>)+/, '').replace(/(<br>)+$/, '');
            return `<button class="preset-btn" data-name="${name}">${display}</button>`;
        }).join('');

        grid.innerHTML = html;

        grid.querySelectorAll('.preset-btn').forEach(btn => {
            btn.addEventListener('click', () => loadPreset(btn.dataset.name));
        });
    } catch (e) {
        grid.innerHTML = `<div class="error">${e.message}</div>`;
    }
}

function setupSigilSequencer() {
    const sequencer = document.getElementById('sigil-sequencer');
    let html = '<div class="stepper-position" id="stepper-position">';
    for (let step = 0; step < 16; step++) {
        html += `<div class="stepper-position-cell" data-step="${step}"></div>`;
    }
    html += '</div>';
    for (let row = 0; row < 4; row++) {
        const label = String.fromCharCode(65 + row);
        html += `<section class="sigil-row" data-row="${row}">`;
        html += '<div class="sigil-row-controls">';
        html += `<div class="sigil-row-label">${label}</div>`;
        html += '<button class="sigil-control sigil-record" data-action="record">Record</button>';
        html += '<button class="sigil-control sigil-clear" data-action="clear">Clear</button>';
        html += '<select class="sigil-route" aria-label="MIDI route">';
        html += '<option value="off">Off</option>';
        html += '<option value="bidule">Bidule</option>';
        html += '<option value="samplesplitter">Samples</option>';
        html += '<option value="both">Both</option>';
        html += '</select>';
        html += '</div>';
        html += '<div class="sigil-steps">';
        for (let step = 0; step < 16; step++) {
            html += `<button class="sigil-step" data-step="${step}" aria-label="Row ${label} step ${step + 1}">${step + 1}</button>`;
        }
        html += '</div>';
        html += '</section>';
    }
    sequencer.innerHTML = html;

    sequencer.addEventListener('click', async (e) => {
        const step = e.target.closest('.sigil-step');
        if (step) {
            const row = step.closest('.sigil-row');
            const patch = row.dataset.patch;
            try {
                await API.stepperToggle(patch, Number(step.dataset.step));
                await refreshStepperStatus();
            } catch (err) {
                console.error('Failed to toggle step:', err);
            }
            return;
        }

        const control = e.target.closest('.sigil-control');
        if (!control) return;

        const row = control.closest('.sigil-row');
        const patch = row.dataset.patch;
        if (control.dataset.action === 'record') {
            const recording = !control.classList.contains('active');
            try {
                await API.stepperSetRecord(patch, recording);
                await refreshStepperStatus();
            } catch (err) {
                console.error('Failed to set stepper record:', err);
            }
        } else if (control.dataset.action === 'clear') {
            try {
                await API.stepperClear(patch);
                await refreshStepperStatus();
            } catch (err) {
                console.error('Failed to clear stepper track:', err);
            }
        }
    });

    sequencer.querySelectorAll('.sigil-row').forEach(row => {
        const label = String.fromCharCode(65 + Number(row.dataset.row));
        row.dataset.patch = label;
    });

    sequencer.querySelectorAll('.sigil-route').forEach(select => {
        select.addEventListener('change', async () => {
            const row = select.closest('.sigil-row');
            try {
                await API.stepperSetRoute(row.dataset.patch, select.value);
                await refreshStepperStatus();
            } catch (err) {
                console.error('Failed to set stepper route:', err);
            }
        });
    });

    document.getElementById('btn-stepper-play').addEventListener('click', async () => {
        const btn = document.getElementById('btn-stepper-play');
        try {
            if (btn.classList.contains('active')) {
                await API.stepperStop();
            } else {
                await API.stepperPlay();
            }
            await refreshStepperStatus();
        } catch (err) {
            console.error('Failed to change stepper playback:', err);
        }
    });
}

async function refreshStepperStatus() {
    if (activeAdventure !== 'sigil') return;
    let status;
    try {
        status = await API.stepperStatus();
    } catch (err) {
        return;
    }
    if (!status || !status.tracks) return;

    const playBtn = document.getElementById('btn-stepper-play');
    if (playBtn) {
        playBtn.classList.toggle('active', !!status.playing);
        playBtn.textContent = status.playing ? 'STOP' : 'PLAY';
    }

    document.querySelectorAll('.stepper-position-cell').forEach(cell => {
        cell.classList.toggle('active', Number(cell.dataset.step) === Number(status.step));
    });

    for (const patch of ['A', 'B', 'C', 'D']) {
        const track = status.tracks[patch];
        const row = document.querySelector(`.sigil-row[data-patch="${patch}"]`);
        if (!track || !row) continue;
        const recordBtn = row.querySelector('.sigil-record');
        recordBtn.classList.toggle('active', !!track.recording);
        recordBtn.textContent = track.recording ? 'Recording' : 'Record';
        const route = row.querySelector('.sigil-route');
        if (route && route.value !== track.route) {
            route.value = track.route || 'bidule';
        }
        row.querySelectorAll('.sigil-step').forEach(btn => {
            const step = Number(btn.dataset.step);
            const events = track.steps && track.steps[step] ? track.steps[step] : [];
            btn.classList.toggle('active', events.length > 0);
            btn.dataset.count = String(events.length);
        });
    }
}

async function loadParams() {
    const grid = document.getElementById('preset-grid');
    grid.classList.remove('grid-mode');
    grid.innerHTML = '<div class="loading">Loading parameters...</div>';

    try {
        // Load paramdefs and paramenums if not cached (for string param dropdowns)
        if (!cachedParamDefs) {
            cachedParamDefs = await API.getParamDefsJson();
        }
        if (!cachedParamEnums) {
            cachedParamEnums = await API.getParamEnums();
        }

        // Get parameter values - use different API for global vs patch categories
        let paramsStr;
        if (currentCategory === 'global') {
            paramsStr = await API.getGlobalParams('global.');
        } else {
            const patchToQuery = currentPatch === '*' ? 'A' : currentPatch;
            paramsStr = await API.getPatchParams(patchToQuery, currentCategory);
        }

        // Parse "name=value\n" format
        const lines = paramsStr.split('\n').filter(l => l.trim());
        if (lines.length === 0) {
            grid.innerHTML = '<div class="loading">No parameters found</div>';
            return;
        }

        // Parse into array of {name, value}
        let params = lines.map(line => {
            const eqIdx = line.indexOf('=');
            if (eqIdx < 0) return null;
            return {
                name: line.substring(0, eqIdx),
                value: line.substring(eqIdx + 1)
            };
        }).filter(p => p);

        // Sort by name
        params.sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));

        // For effect category, filter out sub-params of disabled effects
        if (currentCategory === 'effect') {
            // Find which effects are enabled (boolean params without ":" that are "true")
            const enabledEffects = new Set();
            for (const p of params) {
                if (!p.name.includes(':') && p.value === 'true') {
                    enabledEffects.add(p.name);
                }
            }
            // Filter: show all main params (no ":"), only show sub-params if parent is enabled
            params = params.filter(p => {
                if (!p.name.includes(':')) return true; // Always show main effect toggle
                const parentName = p.name.split(':')[0];
                return enabledEffects.has(parentName);
            });
        }

        // Build parameter list HTML
        let html = '<div class="param-header">';
        html += '<button class="param-header-btn" id="btn-param-init">Init</button>';
        html += '<button class="param-header-btn" id="btn-param-rand">Rand</button>';
        html += '<button class="param-header-btn" id="btn-param-save">Save As</button>';
        html += '</div>';
        html += '<div class="param-list">';

        for (const p of params) {
            const isNumeric = !isNaN(parseFloat(p.value));
            const isBool = p.value === 'true' || p.value === 'false';
            const isMainEffectBool = currentCategory === 'effect' && isBool && !p.name.includes(':');
            const isEffectSubParam = currentCategory === 'effect' && p.name.includes(':');

            let rowClass = 'param-row';
            if (isEffectSubParam) rowClass += ' effect-sub-param';
            if (isMainEffectBool) rowClass += ' effect-main-row';
            html += `<div class="${rowClass}" data-param="${p.name}">`;

            if (isMainEffectBool) {
                // Main effect toggle: single wide button with +/- and name
                const isEnabled = p.value === 'true';
                const btnClass = isEnabled ? 'effect-toggle-enabled' : 'effect-toggle-disabled';
                const symbol = isEnabled ? '-' : '+';
                html += `<button class="param-ctrl effect-toggle ${btnClass}" data-action="toggle">`;
                html += `<span class="effect-symbol">${symbol}</span>`;
                html += `<span class="effect-label">${p.name}</span>`;
                html += `</button>`;
                html += `<span class="param-value" style="display:none">${p.value}</span>`;
                html += `<span class="param-controls"></span>`;
            } else {
                html += `<span class="param-name">${p.name}</span>`;
                // Use slider for numeric params in effect sub-params, visual, sound, misc
                const isFloat = isNumeric && p.value.includes('.');
                const isInt = isNumeric && !p.value.includes('.');
                const sliderCategory = isEffectSubParam || currentCategory === 'visual' || currentCategory === 'sound' || currentCategory === 'misc' || currentCategory === 'global';

                // For bool params in slider categories, use Enabled/Disabled button
                if (sliderCategory && isBool) {
                    const isEnabled = p.value === 'true';
                    const btnClass = isEnabled ? 'bool-toggle-enabled' : 'bool-toggle-disabled';
                    const btnLabel = isEnabled ? 'Enabled' : 'Disabled';
                    html += `<span class="param-value" style="display:none">${p.value}</span>`;
                    html += `<span class="param-controls">`;
                    html += `<button class="param-ctrl bool-toggle ${btnClass}" data-action="toggle">${btnLabel}</button>`;
                    html += `</span>`;
                } else {
                    // Check if this is a string param with enum values
                    const isString = !isNumeric && !isBool;
                    let enumValues = null;
                    let enumName = null;
                    if (isString && cachedParamDefs && cachedParamEnums) {
                        // Look up the param definition to get the enum name from "min" field
                        const paramDef = cachedParamDefs[p.name];
                        if (paramDef && paramDef.valuetype === 'string' && paramDef.min) {
                            enumName = paramDef.min;
                            if (cachedParamEnums[enumName]) {
                                enumValues = cachedParamEnums[enumName];
                            }
                        }
                    }

                    if (enumValues && enumValues.length > 0) {
                        // String param with enum - show dropdown
                        html += `<span class="param-value" style="display:none">${p.value}</span>`;
                        html += '<span class="param-controls">';
                        html += `<select class="param-select">`;
                        for (const opt of enumValues) {
                            const selected = opt === p.value ? ' selected' : '';
                            html += `<option value="${opt}"${selected}>${opt || '(empty)'}</option>`;
                        }
                        html += `</select>`;
                        html += '</span>';
                    } else if (isString) {
                        // String param without enum - show text input
                        const escaped = p.value.replace(/"/g, '&quot;');
                        html += `<span class="param-value" style="display:none">${p.value}</span>`;
                        html += '<span class="param-controls">';
                        html += `<input type="text" class="param-text" value="${escaped}" data-original="${escaped}">`;
                        html += '</span>';
                    } else {
                        html += `<span class="param-value">${p.value}</span>`;
                        html += '<span class="param-controls">';
                        const paramDef = cachedParamDefs ? cachedParamDefs[p.name] : null;
                        if (sliderCategory && isFloat) {
                            const val = parseFloat(p.value);
                            const sMin = paramDef && paramDef.min !== undefined ? parseFloat(paramDef.min) : 0;
                            const sMax = paramDef && paramDef.max !== undefined ? parseFloat(paramDef.max) : 1;
                            const range = sMax - sMin;
                            const step = range > 0 ? Math.max(range / 1000, 0.001) : 0.01;
                            html += `<input type="range" class="param-slider" min="${sMin}" max="${sMax}" step="${step}" value="${val}">`;
                        } else if (sliderCategory && isInt) {
                            const val = parseInt(p.value);
                            const sMin = paramDef && paramDef.min !== undefined ? parseInt(paramDef.min) : 0;
                            const sMax = paramDef && paramDef.max !== undefined ? parseInt(paramDef.max) : 127;
                            html += `<input type="range" class="param-slider param-slider-int" min="${sMin}" max="${sMax}" step="1" value="${val}">`;
                        } else if (isNumeric) {
                            html += `<button class="param-ctrl" data-action="dec2">--</button>`;
                            html += `<button class="param-ctrl" data-action="dec">-</button>`;
                            html += `<button class="param-ctrl" data-action="inc">+</button>`;
                            html += `<button class="param-ctrl" data-action="inc2">++</button>`;
                        } else if (isBool) {
                            html += `<button class="param-ctrl param-toggle" data-action="toggle">toggle</button>`;
                        }
                        html += '</span>';
                    }
                }
            }
            html += '</div>';
        }
        html += '</div>';

        grid.innerHTML = html;

        // Setup event handlers for controls
        setupParamControls();
        setupParamHeaderButtons();
    } catch (e) {
        grid.innerHTML = `<div class="error">${e.message}</div>`;
    }
}

function setupParamControls() {
    // Increment/decrement buttons
    document.querySelectorAll('.param-ctrl').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            e.stopPropagation();
            const row = btn.closest('.param-row');
            const paramName = row.dataset.param;
            const valueEl = row.querySelector('.param-value');
            const currentValue = valueEl.textContent;
            const action = btn.dataset.action;

            let newValue;
            if (action === 'toggle') {
                newValue = currentValue === 'true' ? 'false' : 'true';
            } else {
                const num = parseFloat(currentValue);
                const isInt = Number.isInteger(num) && !currentValue.includes('.');
                let delta;
                if (action === 'dec2') delta = isInt ? -10 : -0.1;
                else if (action === 'dec') delta = isInt ? -1 : -0.01;
                else if (action === 'inc') delta = isInt ? 1 : 0.01;
                else if (action === 'inc2') delta = isInt ? 10 : 0.1;
                newValue = (num + delta).toFixed(isInt ? 0 : 3);
            }

            // Ensure value is a string
            const valueStr = String(newValue);
            console.log('Setting param:', paramName, '=', valueStr, 'patch:', currentPatch);

            try {
                // Use different API for global vs patch parameters
                if (currentCategory === 'global') {
                    await API.setGlobalParam(paramName, valueStr);
                } else if (currentPatch === '*') {
                    for (const p of ['A', 'B', 'C', 'D']) {
                        await API.setPatchParam(p, paramName, valueStr);
                    }
                } else {
                    await API.setPatchParam(currentPatch, paramName, valueStr);
                }
                valueEl.textContent = valueStr;

                // Update bool toggle button label and class
                if (action === 'toggle' && btn.classList.contains('bool-toggle')) {
                    const isEnabled = valueStr === 'true';
                    btn.textContent = isEnabled ? 'Enabled' : 'Disabled';
                    btn.classList.remove('bool-toggle-enabled', 'bool-toggle-disabled');
                    btn.classList.add(isEnabled ? 'bool-toggle-enabled' : 'bool-toggle-disabled');
                }

                // Refresh list if toggling an effect boolean (affects sub-param visibility)
                if (currentCategory === 'effect' && action === 'toggle' && !paramName.includes(':')) {
                    // Save scroll position before refresh
                    const paramList = document.querySelector('.param-list');
                    const scrollTop = paramList ? paramList.scrollTop : 0;
                    await loadParams();
                    // Restore scroll position after refresh
                    const newParamList = document.querySelector('.param-list');
                    if (newParamList) newParamList.scrollTop = scrollTop;
                }
            } catch (err) {
                console.error('Failed to set param:', err);
            }
        });
    });

    // Slider inputs for numeric params
    document.querySelectorAll('.param-slider').forEach(slider => {
        slider.addEventListener('input', async (e) => {
            const row = slider.closest('.param-row');
            const paramName = row.dataset.param;
            const valueEl = row.querySelector('.param-value');
            const isInt = slider.classList.contains('param-slider-int');
            const valueStr = isInt ? String(parseInt(slider.value)) : parseFloat(slider.value).toFixed(3);

            // Update display immediately
            valueEl.textContent = valueStr;

            try {
                if (currentCategory === 'global') {
                    await API.setGlobalParam(paramName, valueStr);
                } else if (currentPatch === '*') {
                    for (const p of ['A', 'B', 'C', 'D']) {
                        await API.setPatchParam(p, paramName, valueStr);
                    }
                } else {
                    await API.setPatchParam(currentPatch, paramName, valueStr);
                }
            } catch (err) {
                console.error('Failed to set param:', err);
            }
        });
    });

    // Select dropdowns for string params with enums
    document.querySelectorAll('.param-select').forEach(select => {
        select.addEventListener('change', async (e) => {
            const row = select.closest('.param-row');
            const paramName = row.dataset.param;
            const valueEl = row.querySelector('.param-value');
            const valueStr = select.value;

            // Update hidden value element
            valueEl.textContent = valueStr;

            try {
                if (currentCategory === 'global') {
                    await API.setGlobalParam(paramName, valueStr);
                } else if (currentPatch === '*') {
                    for (const p of ['A', 'B', 'C', 'D']) {
                        await API.setPatchParam(p, paramName, valueStr);
                    }
                } else {
                    await API.setPatchParam(currentPatch, paramName, valueStr);
                }
            } catch (err) {
                console.error('Failed to set param:', err);
            }
        });
    });

    // Text inputs for string params without enums - submit on Enter, cancel on Escape
    document.querySelectorAll('.param-text').forEach(input => {
        input.addEventListener('keydown', async (e) => {
            const row = input.closest('.param-row');
            const valueEl = row.querySelector('.param-value');

            if (e.key === 'Escape') {
                // Restore original value from data attribute
                e.preventDefault();
                input.value = input.dataset.original || '';
                input.blur();
            } else if (e.key === 'Enter') {
                e.preventDefault();
                const paramName = row.dataset.param;
                const valueStr = input.value;

                // Update hidden value element
                valueEl.textContent = valueStr;

                try {
                    if (currentCategory === 'global') {
                        await API.setGlobalParam(paramName, valueStr);
                    } else if (currentPatch === '*') {
                        for (const p of ['A', 'B', 'C', 'D']) {
                            await API.setPatchParam(p, paramName, valueStr);
                        }
                    } else {
                        await API.setPatchParam(currentPatch, paramName, valueStr);
                    }
                    // Update data-original to reflect the new saved value
                    input.dataset.original = valueStr;
                    // Brief visual feedback - flash the input
                    input.style.backgroundColor = '#4a4';
                    setTimeout(() => { input.style.backgroundColor = ''; }, 200);
                } catch (err) {
                    console.error('Failed to set param:', err);
                    input.style.backgroundColor = '#a44';
                    setTimeout(() => { input.style.backgroundColor = ''; }, 200);
                }
            }
        });
    });
}

function setupParamHeaderButtons() {
    // Init button - set all params to default values
    const initBtn = document.getElementById('btn-param-init');
    if (initBtn) {
        initBtn.addEventListener('click', async () => {
            try {
                // Get init values for the current category
                const initValues = await API.getParamInits(currentCategory);

                // Apply all values in a single batch call per patch
                if (currentPatch === '*') {
                    await Promise.all(['A', 'B', 'C', 'D'].map(p =>
                        API.setPatchParams(p, initValues)
                    ));
                } else {
                    await API.setPatchParams(currentPatch, initValues);
                }

                // Refresh the params display
                await loadParams();
            } catch (err) {
                console.error('Failed to init params:', err);
                alert('Init failed: ' + err.message);
            }
        });
    }

    // Rand button - randomize all params
    const randBtn = document.getElementById('btn-param-rand');
    if (randBtn) {
        randBtn.addEventListener('click', async () => {
            try {
                // Get random values for the current category
                const randValues = await API.getParamRands(currentCategory);

                // Apply all values in a single batch call per patch
                if (currentPatch === '*') {
                    await Promise.all(['A', 'B', 'C', 'D'].map(p =>
                        API.setPatchParams(p, randValues)
                    ));
                } else {
                    await API.setPatchParams(currentPatch, randValues);
                }

                // Refresh the params display
                await loadParams();
            } catch (err) {
                console.error('Failed to randomize params:', err);
                alert('Rand failed: ' + err.message);
            }
        });
    }
}

async function loadPreset(name) {
    try {
        if (currentCategory === 'global') {
            await API.loadGlobal(name);
        } else if (currentCategory === 'quad') {
            if (currentPatch === '*') {
                // Load quad to all patches
                await API.loadQuad(name);
            } else {
                // Load only this patch's portion of the quad
                await API.loadPatch(currentPatch, 'quad', name);
            }
        } else if (currentPatch === '*') {
            // Load to all patches
            for (const p of ['A', 'B', 'C', 'D']) {
                await API.loadPatch(p, currentCategory, name);
            }
        } else {
            await API.loadPatch(currentPatch, currentCategory, name);
        }
    } catch (e) {
        alert('Load failed: ' + e.message);
    }
}

function setupControls() {
    document.getElementById('btn-start-over').addEventListener('click', showAdventureScreen);

    document.getElementById('btn-complete-reset').addEventListener('click', async () => {
        // In advanced mode, COMPLETE RESET returns to normal mode
        if (advancedMode) {
            setAdvancedMode(false);
            return;
        }
        showResetModal();
    });

    document.getElementById('btn-help').addEventListener('click', () => {
        showHelp();
    });

    document.getElementById('btn-record').addEventListener('click', async () => {
        const btn = document.getElementById('btn-record');
        if (btn.classList.contains('recording')) {
            // Stop recording early
            try {
                await API.obsRecordStop();
            } catch (e) {
                console.error('Failed to stop recording:', e);
            }
            stopRecordUI();
        } else {
            // Start recording
            try {
                const result = await API.obsRecord();
                if (result && result.recording) {
                    startRecordUI(result.remaining);
                }
            } catch (e) {
                console.error('Failed to start recording:', e);
                btn.textContent = 'RECORD';
            }
        }
    });
}

// Help overlay functionality
function setupHelpOverlay() {
    // Listen for messages from the help iframe
    window.addEventListener('message', async (e) => {
        if (e.data === 'closeHelp') {
            hideHelp();
            await silenceAll();
        } else if (e.data === 'advancedMode') {
            hideHelp();
            await silenceAll();
            setAdvancedMode(true);
        }
    });
}

let helpVisible = false;

function showHelp() {
    helpVisible = true;
    document.getElementById('help-overlay').classList.remove('hidden');
}

function hideHelp() {
    helpVisible = false;
    document.getElementById('help-overlay').classList.add('hidden');
}

function setAdvancedMode(enabled, shouldLoadPresets = true) {
    advancedMode = enabled;
    const categoryTabs = document.getElementById('category-tabs');
    const patchSelector = document.getElementById('patch-selector');
    const titleBar = document.getElementById('title-bar');

    if (enabled) {
        categoryTabs.classList.remove('hidden');
        patchSelector.classList.remove('hidden');
        titleBar.classList.add('hidden');
        updatePatchButtons();
    } else {
        categoryTabs.classList.add('hidden');
        patchSelector.classList.add('hidden');
        titleBar.classList.remove('hidden');
        // Reset to quad category in normal mode
        currentCategory = 'quad';
        currentPatch = '*';
        if (shouldLoadPresets) {
            loadPresets();
        }
    }
}

function setupCategoryTabs() {
    document.querySelectorAll('#category-tabs .tab').forEach(tab => {
        tab.addEventListener('click', async () => {
            const clickedCategory = tab.dataset.category;

            if (clickedCategory === currentCategory) {
                // Same category clicked - toggle between presets and params
                showingParams = !showingParams;
            } else {
                // Different category - switch to it, show presets
                document.querySelectorAll('#category-tabs .tab').forEach(t => t.classList.remove('active'));
                tab.classList.add('active');
                currentCategory = clickedCategory;
                showingParams = false;
            }

            if (showingParams) {
                await loadParams();
            } else {
                await loadPresets();
            }
        });
    });
}

function setupPatchSelector() {
    document.querySelectorAll('#patch-selector .patch-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const patch = btn.dataset.patch;

            if (patch === '*') {
                // Toggle between all selected and last single selection
                if (currentPatch === '*') {
                    // Currently all selected, switch to single
                    currentPatch = lastSinglePatch;
                    updatePatchButtons();
                } else {
                    // Currently single, switch to all
                    currentPatch = '*';
                    updatePatchButtons();
                }
            } else {
                // Single patch selected
                lastSinglePatch = patch;
                currentPatch = patch;
                updatePatchButtons();
            }
        });
    });
}

function updatePatchButtons() {
    const buttons = document.querySelectorAll('#patch-selector .patch-btn');
    buttons.forEach(b => b.classList.remove('active'));

    if (currentPatch === '*') {
        // Highlight all buttons (*, A, B, C, D)
        buttons.forEach(b => b.classList.add('active'));
    } else {
        // Highlight only the selected single patch
        const btn = document.querySelector(`#patch-selector .patch-btn[data-patch="${currentPatch}"]`);
        if (btn) btn.classList.add('active');
    }
}

// Attract mode overlay
function setupAttractOverlay() {
    const overlay = document.getElementById('attract-overlay');
    overlay.addEventListener('click', exitAttractMode);
    overlay.addEventListener('touchstart', (e) => {
        e.preventDefault();
        exitAttractMode();
    }, { passive: false });
}

async function silenceAll() {
    try {
        for (const p of ['A', 'B', 'C', 'D']) {
            API.setPatchParam(p, 'misc.looping_on', 'false');
            API.call('patch.clear', { patch: p });
        }
        await API.call('quad.ANO');
        await API.audioReset();
    } catch (e) {
        console.error('Failed to silence all:', e);
    }
}

async function exitAttractMode() {
    await API.call('global.attract', { onoff: 'false' }).catch(() => {});
    await silenceAll();
    hideAttract();
}

function showAttract() {
    if (!attractModeActive) {
        attractModeActive = true;
        document.getElementById('attract-overlay').classList.remove('hidden');
    }
}

function hideAttract() {
    if (attractModeActive) {
        attractModeActive = false;
        document.getElementById('attract-overlay').classList.add('hidden');
    }
}

async function pollStatus() {
    if (activeAdventure !== 'space') {
        hideAttract();
        return;
    }

    try {
        const status = await API.getStatus();
        if (status && status.attractmode === 'true' && attractAllowGui && !helpVisible) {
            showAttract();
        } else {
            hideAttract();
        }
    } catch (e) {
        // Ignore polling errors
    }
}

// Reset modal
function showResetModal() {
    const overlay = document.getElementById('restart-overlay');
    const modal = document.getElementById('restart-modal');
    const message = document.getElementById('restart-message');
    modal.classList.remove('hidden');
    message.classList.add('hidden');
    overlay.classList.remove('hidden');
}

function hideResetModal() {
    document.getElementById('restart-overlay').classList.add('hidden');
}

function showResetMessage() {
    document.getElementById('restart-modal').classList.add('hidden');
    document.getElementById('restart-message').classList.remove('hidden');
}

// Recording UI
let recordInterval = null;

function startRecordUI(remaining) {
    const btn = document.getElementById('btn-record');
    btn.classList.add('recording');
    updateRecordButton(remaining);

    recordInterval = setInterval(async () => {
        try {
            const status = await API.obsRecordStatus();
            if (status && status.recording) {
                updateRecordButton(status.remaining);
            } else {
                stopRecordUI();
            }
        } catch (e) {
            stopRecordUI();
        }
    }, 1000);
}

function updateRecordButton(remaining) {
    const btn = document.getElementById('btn-record');
    btn.innerHTML = `REC<br>${Math.round(remaining)}s`;
}

function stopRecordUI() {
    const btn = document.getElementById('btn-record');
    btn.classList.remove('recording');
    btn.textContent = 'RECORD';
    if (recordInterval) {
        clearInterval(recordInterval);
        recordInterval = null;
    }
}

document.addEventListener('click', async (e) => {
    const btn = e.target.closest('.restart-btn');
    if (!btn) return;
    const action = btn.dataset.action;

    if (action === 'cancel') {
        hideResetModal();
        return;
    }

    if (action === 'complete') {
        await silenceAll();
        showResetMessage();
        try {
            await API.call('global.done');
        } catch (e) {
            // Expected - engine exits so connection drops
        }
        return;
    }

    if (action === 'audio') {
        await silenceAll();
        // Stop and restart Bidule
        try {
            await API.setGlobalParam('global.process.bidule', 'false');
            await new Promise(r => setTimeout(r, 1000));
            await API.setGlobalParam('global.process.bidule', 'true');
        } catch (e) {
            console.error('Failed to restart audio:', e);
        }
        hideResetModal();
        return;
    }

    if (action === 'visuals') {
        // Stop and restart Resolume
        try {
            await API.setGlobalParam('global.process.resolume', 'false');
            await new Promise(r => setTimeout(r, 1000));
            await API.setGlobalParam('global.process.resolume', 'true');
        } catch (e) {
            console.error('Failed to restart visuals:', e);
        }
        hideResetModal();
        return;
    }
});

