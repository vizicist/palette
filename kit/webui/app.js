import { API } from './api.js';
import { initialPageDefaultRoute, Routes, routeLabel } from './routes.js';
import { patchNames, stepperNumSteps, samplePlaybackQuantValues, UIState } from './state.js';
import { applyUISnapshot, requestUISnapshot, setupUIStateFeed } from './ui_nats.js';
import {
    applyInitialPageMode,
    fitAppTitle,
    flashSigilForPatch,
    hideAttract,
    hideHelp,
    hideResetModal,
    renderStepperIndicator,
    resetRecordButton,
    setPalettePadActivity,
    setupAppTitleFit,
    showAttract,
    showHelp,
    showResetMessage,
    showResetModal,
    updatePalettePadRoute,
    updatePatchButtons,
    updatePresetButtons,
    updateRecordButton,
    updateRitualNav,
    updateStepperIndicator
} from './render.js';

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('touchscreen') === '1') {
        document.body.classList.add('touchscreen-embed');
    }

    try {
        await setupUIStateFeed({
            status: handleUIStatus,
            stepper: handleStepperStatus,
            cursor: handleCursorActivity,
            obsRecord: handleOBSRecordStatus
        });
        await seedUIState();
    } catch (e) {
        console.warn('NATS UI feed unavailable:', e);
        updateRecordButtonVisibility(false);
    }

    applyInitialPageMode();
    setupAppTitleFit();
    setupRitualNav();
    setupControls();
    setupHelpOverlay();
    setupAttractOverlay();
    setupCategoryTabs();
    setupPatchSelector();
    setupSigilSequencer();
    setupPalettePads();
    setupSamplePlaybackControls();
    setupTempoControl();
    await startInitialPage();

    requestAnimationFrame(updateStepperIndicator);
});

async function seedUIState() {
    try {
        applyUISnapshot(await requestUISnapshot(), {
            status: status => {
                syncStartupMode(status);
                handleUIStatus(status);
            },
            stepper: handleStepperStatus,
            cursor: handleCursorActivity,
            obsRecord: handleOBSRecordStatus
        });
    } catch (e) {
        console.warn('NATS UI snapshot seed failed:', e);
        updateRecordButtonVisibility(false);
    }
}

async function startInitialPage() {
    await startSpacePalette();
}

async function syncInitialPageFromEngine() {
    try {
        const snapshot = await requestUISnapshot();
        if (snapshot && snapshot.status) {
            syncInitialPageValue(statusMode(snapshot.status));
        }
    } catch (e) {
        // Ignore transient API errors.
    }
}

function syncInitialPageValue(page) {
    const previous = UIState.initialPage;
    UIState.setInitialPage(page);
    if (UIState.initialPage === previous) return;
    applyInitialPageMode();
    fitAppTitle();
    if (UIState.activeAdventure === 'space') {
        setAdvancedMode(UIState.advancedMode, false);
        updateRitualNav();
        updatePresetButtons();
    }
}

function syncStartupMode(status) {
    UIState.advancedMode = status && (status.guidefaultlevel === '1' || status.guidefaultlevel === 1);
    UIState.attractAllowGui = !!(status && status.attractallowgui);
    const mode = statusMode(status);
    if (mode) {
        UIState.setInitialPage(mode);
    }
}

function statusMode(status) {
    if (!status) return '';
    return status.mode || '';
}

function setupRitualNav() {
    document.getElementById('btn-nav-space').addEventListener('click', startSpacePalette);
    document.getElementById('btn-nav-sigil').addEventListener('click', showSigilSequencer);
}

async function startSpacePalette() {
    await stopStepperQuietly();
    UIState.setActiveAdventure('space');
    updateRitualNav();
    document.getElementById('sigil-screen').classList.add('hidden');
    document.getElementById('main-container').classList.remove('hidden');
    setAdvancedMode(UIState.advancedMode, false);
    await refreshStepperStatus();
    await loadPresets();
    await syncAttractStateFromEngine();
}

async function showSigilSequencer() {
    UIState.setActiveAdventure('sigil');
    updateRitualNav();
    hideAttract();
    hideHelp();
    document.getElementById('main-container').classList.add('hidden');
    document.getElementById('title-bar').classList.add('hidden');
    document.getElementById('category-tabs').classList.add('hidden');
    document.getElementById('patch-selector').classList.add('hidden');
    document.getElementById('sigil-screen').classList.remove('hidden');
    await API.stepperPlay().catch(err => console.error('Failed to start stepper playback:', err));
    await setStepperDefaults();
}

function updateRecordButtonVisibility(running = false) {
    const btn = document.getElementById('btn-record');
    if (!btn) return;
    btn.style.display = running ? '' : 'none';
}

async function loadPresets() {
    const grid = document.getElementById('preset-grid');
    grid.classList.add('grid-mode');
    grid.innerHTML = '<div class="loading">Loading...</div>';

    try {
        const list = await API.getSavedList(UIState.currentCategory);

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
        updatePresetButtons();
    } catch (e) {
        grid.innerHTML = `<div class="error">${e.message}</div>`;
    }
}

async function stopStepperQuietly() {
    try {
        await API.stepperStop();
    } catch (e) { /* Stepper may be unavailable during startup */ }
    await Promise.all(patchNames.map(patch =>
        API.stepperSetRecord(patch, false).catch(err => console.error(`Failed to disable stepper recording for ${patch}:`, err))
    ));
}

async function setStepperDefaults() {
    await Promise.all(patchNames.flatMap(patch => [
        API.stepperSetRecord(patch, true).catch(err => console.error(`Failed to enable stepper recording for ${patch}:`, err)),
        API.stepperSetRoute(patch, Routes.samples).catch(err => console.error(`Failed to set stepper route for ${patch}:`, err))
    ]));
}

function setupSigilSequencer() {
    const sequencer = document.getElementById('sigil-sequencer');
    let html = '<div class="stepper-position" id="stepper-position">';
    for (let step = 0; step < stepperNumSteps; step++) {
        html += `<div class="stepper-position-cell" data-step="${step}"></div>`;
    }
    html += '</div>';
    for (let row = 0; row < 4; row++) {
        const label = String.fromCharCode(65 + row);
        html += `<section class="sigil-row" data-row="${row}">`;
        html += '<div class="sigil-row-controls">';
        html += `<div class="sigil-row-label">${label}</div>`;
        html += '<button class="sigil-control sigil-clear" data-action="clear">Clear</button>';
        html += '<select class="sigil-route" aria-label="MIDI route">';
        for (const route of [Routes.samples, Routes.off, Routes.bidule, Routes.both]) {
            html += `<option value="${route}">${routeLabel(route)}</option>`;
        }
        html += '</select>';
        html += '</div>';
        html += '<div class="sigil-steps">';
        for (let step = 0; step < stepperNumSteps; step++) {
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
        if (control.dataset.action === 'clear') {
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

}

function setupPalettePads() {
    const stage = document.getElementById('palette-pad-stage');
    if (!stage) return;
    stage.addEventListener('click', (e) => {
        const pad = e.target.closest('.palette-pad');
        if (!pad) return;
        const patch = pad.dataset.pad;
        const route = pad.dataset.route === Routes.samples ? Routes.bidule : Routes.samples;
        updatePalettePadRoute(patch, route);
        API.stepperSetRoute(patch, route)
            .then(() => refreshStepperStatus())
            .catch(err => {
                console.error('Failed to set palette pad route:', err);
                refreshStepperStatus().catch(() => updatePalettePadRoute(patch, pad.dataset.route || Routes.samples));
            });
    });
}

function setupSamplePlaybackControls() {
    const quant = document.getElementById('sample-playback-quant');
    const words = document.getElementById('sample-playback-words');
    const newSet = document.getElementById('sample-playback-newset');
    if (!quant) return;

    const indexForQuant = (value) => {
        const numeric = Number(value);
        if (!Number.isFinite(numeric)) return 2;
        let bestIndex = 0;
        let bestDistance = Infinity;
        samplePlaybackQuantValues.forEach((candidate, index) => {
            const distance = Math.abs(candidate - numeric);
            if (distance < bestDistance) {
                bestIndex = index;
                bestDistance = distance;
            }
        });
        return bestIndex;
    };
    const setFromValue = (value) => {
        quant.value = String(samplePlaybackQuantValues[indexForQuant(value)]);
    };

    API.call('global.get', { name: 'global.sampleplaybackquant' })
        .then(setFromValue)
        .catch(() => setFromValue(0.5));

    const sendQuant = () => {
        const index = indexForQuant(quant.value);
        quant.value = String(samplePlaybackQuantValues[index]);
        API.setGlobalParam('global.sampleplaybackquant', String(samplePlaybackQuantValues[index]))
            .catch(err => console.error('Failed to set sample playback quantize:', err));
    };

    quant.addEventListener('change', sendQuant);

    if (words) {
        const clampWords = (value) => {
            const numeric = Math.round(Number(value));
            if (!Number.isFinite(numeric)) return 2;
            return Math.max(1, Math.min(16, numeric));
        };
        API.call('global.get', { name: 'global.sampleplaybackwords' })
            .then(value => {
                words.value = String(clampWords(value));
            })
            .catch(() => {
                words.value = '2';
            });
        words.addEventListener('change', async () => {
            const selected = clampWords(words.value);
            words.value = String(selected);
            words.disabled = true;
            if (newSet) {
                newSet.disabled = true;
                newSet.textContent = 'Busy';
            }
            try {
                await API.setGlobalParam('global.sampleplaybackwords', String(selected));
                await refreshStepperStatus();
                if (newSet) {
                    newSet.textContent = 'Ready';
                    setTimeout(() => {
                        if (newSet.textContent === 'Ready') newSet.textContent = 'Receive New Prophecies';
                    }, 1200);
                }
            } catch (err) {
                console.error('Failed to set sample playback word count:', err);
                if (newSet) newSet.textContent = 'Error';
            } finally {
                words.disabled = false;
                if (newSet) newSet.disabled = false;
            }
        });
    }

    if (newSet) {
        const newSetLabel = 'Receive New Prophecies';
        newSet.addEventListener('click', async () => {
            newSet.disabled = true;
            newSet.textContent = 'Busy';
            const busyStartedAt = performance.now();
            try {
                await API.reloadSamplePlaybackSet();
                await refreshStepperStatus();
                const remainingBusyMs = Math.max(0, 1000 - (performance.now() - busyStartedAt));
                if (remainingBusyMs > 0) {
                    await new Promise(resolve => setTimeout(resolve, remainingBusyMs));
                }
            } catch (err) {
                console.error('Failed to load new sample playback set:', err);
                newSet.textContent = 'Error';
                setTimeout(() => {
                    if (newSet.textContent === 'Error') newSet.textContent = newSetLabel;
                }, 2200);
                return;
            } finally {
                newSet.disabled = false;
            }
            newSet.textContent = 'Ready';
            setTimeout(() => {
                if (newSet.textContent === 'Ready') newSet.textContent = newSetLabel;
            }, 1200);
        });
    }
}

function setupTempoControl() {
    const slider = document.getElementById('tempo-slider');
    const value = document.getElementById('tempo-value');
    if (!slider || !value) return;

    let tempoTimer = null;
    const updateDisplay = () => {
        const bpm = Number(slider.value) || 120;
        value.textContent = `${bpm} BPM`;
    };
    const sendTempo = () => {
        updateDisplay();
        clearTimeout(tempoTimer);
        tempoTimer = setTimeout(async () => {
            const bpm = Number(slider.value) || 120;
            const factor = bpm / 120;
            try {
                await API.setTempoFactor(factor.toFixed(4));
                await refreshStepperStatus();
            } catch (err) {
                console.error('Failed to set tempo:', err);
            }
        }, 80);
    };

    updateDisplay();
    slider.addEventListener('input', sendTempo);
    slider.addEventListener('change', sendTempo);
}

function handleStepperStatus(status) {
    if (!UIState.wantsStepperStatus()) return;
    if (!status || !status.tracks) return;

    syncStepperTiming(status);
    renderStepperIndicator();

    for (const patch of patchNames) {
        const track = status.tracks[patch];
        if (!track) continue;
        updatePalettePadRoute(patch, track.route || initialPageDefaultRoute(UIState.initialPage));
        const row = document.querySelector(`.sigil-row[data-patch="${patch}"]`);
        if (!row) continue;
        const route = row.querySelector('.sigil-route');
        if (route && route.value !== track.route) {
            route.value = track.route || initialPageDefaultRoute(UIState.initialPage);
        }
        row.querySelectorAll('.sigil-step').forEach(btn => {
            const step = Number(btn.dataset.step);
            const events = track.steps && track.steps[step] ? track.steps[step] : [];
            btn.classList.toggle('active', events.length > 0);
            btn.dataset.count = String(events.length);
        });
    }
}

async function refreshStepperStatus() {
    if (!UIState.wantsStepperStatus()) return;
    try {
        const snapshot = await requestUISnapshot();
        if (snapshot && snapshot.stepper) handleStepperStatus(snapshot.stepper);
    } catch (err) {
        // User-triggered refreshes may race engine startup; UI push events will catch up.
    }
}

function syncStepperTiming(status) {
    UIState.syncStepperTiming(status);
}

async function loadParams() {
    const grid = document.getElementById('preset-grid');
    grid.classList.remove('grid-mode');
    grid.innerHTML = '<div class="loading">Loading parameters...</div>';

    try {
        // Load paramdefs and paramenums if not cached (for string param dropdowns)
        if (!UIState.paramDefs) {
            UIState.paramDefs = await API.getParamDefsJson();
        }
        if (!UIState.paramEnums) {
            UIState.paramEnums = await API.getParamEnums();
        }

        // Get parameter values - use different API for global vs patch categories
        let paramsStr;
        if (UIState.currentCategory === 'global') {
            paramsStr = await API.getGlobalParams('global.');
        } else {
            const patchToQuery = UIState.currentPatch === '*' ? 'A' : UIState.currentPatch;
            paramsStr = await API.getPatchParams(patchToQuery, UIState.currentCategory);
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
        if (UIState.currentCategory === 'effect') {
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
            const isMainEffectBool = UIState.currentCategory === 'effect' && isBool && !p.name.includes(':');
            const isEffectSubParam = UIState.currentCategory === 'effect' && p.name.includes(':');

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
                const sliderCategory = isEffectSubParam || UIState.currentCategory === 'visual' || UIState.currentCategory === 'sound' || UIState.currentCategory === 'misc' || UIState.currentCategory === 'global';

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
                    if (isString && UIState.paramDefs && UIState.paramEnums) {
                        // Look up the param definition to get the enum name from "min" field
                        const paramDef = UIState.paramDefs[p.name];
                        if (paramDef && paramDef.valuetype === 'string' && paramDef.min) {
                            enumName = paramDef.min;
                            if (UIState.paramEnums[enumName]) {
                                enumValues = UIState.paramEnums[enumName];
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
                        const paramDef = UIState.paramDefs ? UIState.paramDefs[p.name] : null;
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
            console.log('Setting param:', paramName, '=', valueStr, 'patch:', UIState.currentPatch);

            try {
                // Use different API for global vs patch parameters
                if (UIState.currentCategory === 'global') {
                    await API.setGlobalParam(paramName, valueStr);
                } else if (UIState.currentPatch === '*') {
                    for (const p of patchNames) {
                        await API.setPatchParam(p, paramName, valueStr);
                    }
                } else {
                    await API.setPatchParam(UIState.currentPatch, paramName, valueStr);
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
                if (UIState.currentCategory === 'effect' && action === 'toggle' && !paramName.includes(':')) {
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
                if (UIState.currentCategory === 'global') {
                    await API.setGlobalParam(paramName, valueStr);
                } else if (UIState.currentPatch === '*') {
                    for (const p of patchNames) {
                        await API.setPatchParam(p, paramName, valueStr);
                    }
                } else {
                    await API.setPatchParam(UIState.currentPatch, paramName, valueStr);
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
                if (UIState.currentCategory === 'global') {
                    await API.setGlobalParam(paramName, valueStr);
                } else if (UIState.currentPatch === '*') {
                    for (const p of patchNames) {
                        await API.setPatchParam(p, paramName, valueStr);
                    }
                } else {
                    await API.setPatchParam(UIState.currentPatch, paramName, valueStr);
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
                    if (UIState.currentCategory === 'global') {
                        await API.setGlobalParam(paramName, valueStr);
                    } else if (UIState.currentPatch === '*') {
                        for (const p of patchNames) {
                            await API.setPatchParam(p, paramName, valueStr);
                        }
                    } else {
                        await API.setPatchParam(UIState.currentPatch, paramName, valueStr);
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
                const initValues = await API.getParamInits(UIState.currentCategory);

                // Apply all values in a single batch call per patch
                if (UIState.currentPatch === '*') {
                    await Promise.all(patchNames.map(p =>
                        API.setPatchParams(p, initValues)
                    ));
                } else {
                    await API.setPatchParams(UIState.currentPatch, initValues);
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
                const randValues = await API.getParamRands(UIState.currentCategory);

                // Apply all values in a single batch call per patch
                if (UIState.currentPatch === '*') {
                    await Promise.all(patchNames.map(p =>
                        API.setPatchParams(p, randValues)
                    ));
                } else {
                    await API.setPatchParams(UIState.currentPatch, randValues);
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
        if (UIState.currentCategory === 'global') {
            await API.loadGlobal(name);
        } else if (UIState.currentCategory === 'quad') {
            if (UIState.currentPatch === '*') {
                // Load quad to all patches
                await API.loadQuad(name);
            } else {
                // Load only this patch's portion of the quad
                await API.loadPatch(UIState.currentPatch, 'quad', name);
            }
        } else if (UIState.currentPatch === '*') {
            // Load to all patches
            for (const p of patchNames) {
                await API.loadPatch(p, UIState.currentCategory, name);
            }
        } else {
            await API.loadPatch(UIState.currentPatch, UIState.currentCategory, name);
        }
        UIState.selectedPresets.set(UIState.presetKey(), name);
        updatePresetButtons();
    } catch (e) {
        alert('Load failed: ' + e.message);
    }
}

function handleCursorActivity(activity) {
    if (!UIState.wantsCursorActivity()) return;
    for (const patch of patchNames) {
        const count = Number(activity && activity[patch]) || 0;
        if (UIState.activeAdventure === 'sigil' && count > UIState.cursorActivityCounts[patch]) {
            flashSigilForPatch(patch);
        }
        setPalettePadActivity(patch, count > 0);
        UIState.cursorActivityCounts[patch] = count;
    }
}

function setupControls() {
    document.getElementById('btn-complete-reset').addEventListener('click', async () => {
        // In advanced mode, COMPLETE RESET returns to normal mode
        if (UIState.advancedMode) {
            await syncInitialPageFromEngine();
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

function setAdvancedMode(enabled, shouldLoadPresets = true) {
    UIState.setAdvancedMode(enabled);
    document.body.classList.toggle('advanced-mode', enabled);
    const categoryTabs = document.getElementById('category-tabs');
    const patchSelector = document.getElementById('patch-selector');
    const titleBar = document.getElementById('title-bar');
    const proInitialPage = UIState.initialPage === 'pro';

    if (enabled) {
        categoryTabs.classList.remove('hidden');
        patchSelector.classList.remove('hidden');
        titleBar.classList.toggle('hidden', !proInitialPage);
        updatePatchButtons();
    } else {
        categoryTabs.classList.add('hidden');
        patchSelector.classList.add('hidden');
        titleBar.classList.toggle('hidden', !proInitialPage);
        // Reset to quad category in normal mode
        UIState.resetNormalPresetView();
        if (shouldLoadPresets) {
            loadPresets();
        }
    }
}

function setupCategoryTabs() {
    document.querySelectorAll('#category-tabs .tab').forEach(tab => {
        tab.addEventListener('click', async () => {
            const clickedCategory = tab.dataset.category;

            if (clickedCategory !== UIState.currentCategory) {
                // Different category - switch to it, show presets
                document.querySelectorAll('#category-tabs .tab').forEach(t => t.classList.remove('active'));
                tab.classList.add('active');
            }
            UIState.toggleCategory(clickedCategory);

            if (UIState.showingParams) {
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

            UIState.selectPatch(patch);
            updatePatchButtons();
            updatePresetButtons();
        });
    });
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
        for (const p of patchNames) {
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

async function syncAttractStateFromEngine() {
    try {
        const snapshot = await requestUISnapshot();
        if (snapshot && snapshot.status) {
            handleUIStatus(snapshot.status);
        }
    } catch (e) {
        // Ignore transient API errors.
    }
}

function handleUIStatus(status) {
    if (UIState.activeAdventure !== 'space') {
        hideAttract();
    } else if (status && isTrueStatusValue(status.attractmode) && UIState.attractAllowGui && !UIState.helpVisible) {
        showAttract();
    } else {
        hideAttract();
    }

    updateRecordButtonVisibility(!!(status && status.obsrunning));

    if (status && Object.prototype.hasOwnProperty.call(status, 'attractallowgui')) {
        UIState.attractAllowGui = !!status.attractallowgui;
    }
    const mode = statusMode(status);
    if (mode) {
        syncInitialPageValue(mode);
    }
    if (status && status.presets && typeof status.presets === 'object') {
        Object.entries(status.presets).forEach(([key, value]) => {
            UIState.selectedPresets.set(key, value);
        });
        updatePresetButtons();
    }
}

function isTrueStatusValue(value) {
    return value === true || value === 'true';
}

// Recording UI
function startRecordUI(remaining) {
    const btn = document.getElementById('btn-record');
    btn.classList.add('recording');
    updateRecordButton(remaining);
}

function stopRecordUI() {
    resetRecordButton();
}

function handleOBSRecordStatus(status) {
    updateRecordButtonVisibility(!!(status && status.obsrunning));
    if (status && status.recording) {
        startRecordUI(status.remaining);
    } else {
        stopRecordUI();
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
