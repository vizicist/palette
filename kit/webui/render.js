import { initialPageDefaultRoute, Routes } from './routes.js';
import { patchNames, patchSigils, stepperNumSteps, UIState } from './state.js';

export function setupAppTitleFit() {
    fitAppTitle();
    window.addEventListener('resize', fitAppTitle);
    if (document.fonts && document.fonts.ready) {
        document.fonts.ready.then(fitAppTitle).catch(() => {});
    }
}

export function fitAppTitle() {
    const title = document.getElementById('app-title');
    const text = document.getElementById('app-title-text');
    if (!title || !text) return;

    title.style.setProperty('--app-title-scale', '1');
    const availableWidth = Math.max(1, title.clientWidth - 12);
    const naturalWidth = Math.max(1, text.scrollWidth);
    const scale = Math.min(1, availableWidth / naturalWidth);
    title.style.setProperty('--app-title-scale', scale.toFixed(3));
}

export function applyInitialPageMode() {
    document.body.classList.remove('initial-pro', 'initial-bss');
    document.body.classList.add(`initial-${UIState.initialPage}`);
    for (const patch of patchNames) {
        updatePalettePadRoute(patch, initialPageDefaultRoute(UIState.initialPage));
    }
}

export function updateRitualNav() {
    document.querySelectorAll('.ritual-nav-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.screen === UIState.activeAdventure);
    });
}

export function updatePalettePadRoute(patch, route) {
    const pad = document.querySelector(`.palette-pad[data-pad="${patch}"]`);
    if (!pad) return;
    const normalized = route === Routes.samples || route === Routes.both ? Routes.samples : Routes.bidule;
    pad.dataset.route = normalized;
    pad.classList.toggle('sample', normalized === Routes.samples);
    pad.classList.toggle('synth', normalized === Routes.bidule);
    const button = pad.querySelector('.palette-pad-route');
    if (button) {
        const mode = normalized === Routes.samples ? 'Prophesize' : 'Oscillate';
        button.setAttribute('aria-label', `Mode: ${mode}`);
    }
}

export function renderStepperIndicator() {
    if (UIState.activeAdventure !== 'sigil') return;
    let step = 0;
    if (UIState.stepperTiming.playing && UIState.stepperTiming.clicksPerSecond > 0) {
        const elapsedMs = performance.now() - UIState.stepperTiming.receivedAt;
        const estimatedClick = UIState.stepperTiming.click + (elapsedMs * UIState.stepperTiming.clicksPerSecond / 1000);
        step = Math.floor(estimatedClick / UIState.stepperTiming.stepLength) % stepperNumSteps;
    }
    document.querySelectorAll('.stepper-position-cell').forEach(cell => {
        cell.classList.toggle('active', UIState.stepperTiming.playing && Number(cell.dataset.step) === step);
    });
}

export function updateStepperIndicator() {
    renderStepperIndicator();
    requestAnimationFrame(updateStepperIndicator);
}

export function flashSigilForPatch(patch) {
    const sigil = patchSigils[patch];
    if (!sigil) return;
    const img = document.querySelector(`.sigil-band img[data-sigil="${sigil}"]`);
    if (img) {
        img.classList.remove('flash');
        void img.offsetWidth;
        img.classList.add('flash');
    }
}

export function setPalettePadActivity(patch, active) {
    const pad = document.querySelector(`.palette-pad[data-pad="${patch}"]`);
    if (pad) {
        pad.classList.toggle('morph-active', active);
    }
}

export function updatePresetButtons() {
    const selected = UIState.selectedPresets.get(UIState.presetKey());
    document.querySelectorAll('#preset-grid .preset-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.name === selected);
    });
}

export function updatePatchButtons() {
    const buttons = document.querySelectorAll('#patch-selector .patch-btn');
    buttons.forEach(b => b.classList.remove('active'));

    if (UIState.currentPatch === '*') {
        buttons.forEach(b => b.classList.add('active'));
    } else {
        const btn = document.querySelector(`#patch-selector .patch-btn[data-patch="${UIState.currentPatch}"]`);
        if (btn) btn.classList.add('active');
    }
}

export function showHelp() {
    UIState.helpVisible = true;
    const helpFrame = document.querySelector('#help-overlay iframe');
    if (helpFrame) {
        const helpPage = UIState.initialPage === 'bss' ? 'bss_helpscreen.html' : 'helpscreen.html';
        if (!helpFrame.src.endsWith(helpPage)) {
            helpFrame.src = helpPage;
        }
    }
    document.getElementById('help-overlay').classList.remove('hidden');
}

export function hideHelp() {
    UIState.helpVisible = false;
    document.getElementById('help-overlay').classList.add('hidden');
}

export function showAttract() {
    if (!UIState.attractModeActive) {
        UIState.attractModeActive = true;
        document.getElementById('attract-overlay').classList.remove('hidden');
    }
}

export function hideAttract() {
    if (UIState.attractModeActive) {
        UIState.attractModeActive = false;
        document.getElementById('attract-overlay').classList.add('hidden');
    }
}

export function showResetModal() {
    const overlay = document.getElementById('restart-overlay');
    const modal = document.getElementById('restart-modal');
    const message = document.getElementById('restart-message');
    modal.classList.remove('hidden');
    message.classList.add('hidden');
    overlay.classList.remove('hidden');
}

export function hideResetModal() {
    document.getElementById('restart-overlay').classList.add('hidden');
}

export function showResetMessage() {
    document.getElementById('restart-modal').classList.add('hidden');
    document.getElementById('restart-message').classList.remove('hidden');
}

export function updateRecordButton(remaining) {
    const btn = document.getElementById('btn-record');
    btn.innerHTML = `REC<br>${Math.round(remaining)}s`;
}

export function resetRecordButton() {
    const btn = document.getElementById('btn-record');
    btn.classList.remove('recording');
    btn.textContent = 'RECORD';
}
