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

// Attract screens per mode. Modes without an entry use the default image
// already in the markup.
const attractImages = {
    goat: { src: 'goat_attractscreen.png', alt: 'Dirty Goat Roadhouse' }
};
const defaultAttractImage = { src: 'sppro_attractscreen.png', alt: 'Space Palette Pro' };

function applyAttractImage(mode) {
    const img = document.querySelector('#attract-overlay .attract-image');
    if (!img) return;
    const { src, alt } = attractImages[mode] || defaultAttractImage;
    // Only touch src when it changes, so switching modes doesn't make the
    // browser re-fetch and flash the image.
    if (!img.getAttribute('src').endsWith(src)) {
        img.setAttribute('src', src);
    }
    img.setAttribute('alt', alt);
}

export function applyInitialPageMode() {
    document.body.classList.remove('initial-pro', 'initial-bss', 'initial-pro2', 'initial-goat');
    document.body.classList.add(`initial-${UIState.initialPage}`);
    applyAttractImage(UIState.initialPage);
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
        const mode = normalized === Routes.samples ? 'Words' : 'Music';
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
    // Before the overlay goes away, not after: a <video> that is merely hidden
    // goes on playing, and its soundtrack would carry on over whatever the
    // person who just touched the pads is playing.
    hideAttractVideo();
    if (UIState.attractModeActive) {
        UIState.attractModeActive = false;
        document.getElementById('attract-overlay').classList.add('hidden');
    }
}

// The attract overlay shows one of two things: the attract screen image, or -
// when global.attractvideodestination is "gui" - the attract videos playing
// here rather than on the Resolume output. app.js owns the playlist and decides
// when to move on; these two are the DOM half of it.

function attractImageElement() {
    return document.querySelector('#attract-overlay .attract-image');
}

// applyAttractVideoResize puts the current global.attractvideoresize onto the
// elements: the video fills its frame, and the frame gives up the bottom
// third of the screen to the venue logo. Called both when the parameter
// changes and on every file, so a video that starts after the change is framed
// the same way as the one playing when it happened.
export function applyAttractVideoResize() {
    const video = document.getElementById('attract-video');
    if (!video) return;
    const fill = !!UIState.attractVideoResize;
    video.classList.toggle('attract-video-fill', fill);

    const logo = document.getElementById('attract-video-logo');
    if (logo) {
        // The logo goes with the video and only with the video: the space it
        // sits in is space taken from one. It must not appear over the attract
        // screen image, which is what shows when there is nothing to play.
        logo.classList.toggle('hidden', !fill || video.classList.contains('hidden'));
    }
}

// showAttractVideoSource points the overlay's video element at one file and
// starts it. The attract image is hidden while it plays: the video keeps its
// aspect ratio, so unless it happens to be the shape of its frame there is
// space around it that the image would otherwise show through.
export function showAttractVideoSource(src) {
    const video = document.getElementById('attract-video');
    if (!video) return;
    const img = attractImageElement();
    if (img) img.classList.add('hidden');
    video.classList.remove('hidden');
    applyAttractVideoResize();
    video.src = src;
    // Always silent. These videos are the idle screen, not a performance: the
    // GUI screen sits next to whoever walks up, and audio out of it would talk
    // over the room and over Bidule. The Resolume destination is the one that
    // carries any soundtrack.
    //
    // Set here as well as on the element, rather than trusting the markup: this
    // is the only thing standing between a fresh set of files and unexpected
    // noise, and it also means muted autoplay - which browsers always allow -
    // so the videos start without waiting for anyone to touch the screen.
    video.muted = true;
    video.play().catch(() => {
        // Nothing to retry: muted autoplay is what browsers permit, so a
        // refusal here is the file, not the policy. The error handler in app.js
        // moves on to the next one.
    });
}

// hideAttractVideo stops playback and gives the attract image back. The flag is
// cleared here rather than by the callers so that it cannot disagree with the
// element: hideAttract calls this too, and anything that took the overlay down
// without clearing the flag would leave app.js believing the videos were still
// running and refusing to start them again.
//
// Clearing it first also matters because tearing the source down can itself
// raise an error event, which app.js must not read as a file that wouldn't
// play.
export function hideAttractVideo() {
    UIState.attractVideoPlaying = false;
    const video = document.getElementById('attract-video');
    if (video && !video.classList.contains('hidden')) {
        video.pause();
        video.removeAttribute('src');
        // Drop what has been buffered, rather than holding a whole video in
        // memory between attract runs.
        video.load();
        video.classList.add('hidden');
    }
    const logo = document.getElementById('attract-video-logo');
    if (logo) logo.classList.add('hidden');
    const img = attractImageElement();
    if (img) img.classList.remove('hidden');
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

// showToast briefly displays a message at the bottom of the screen. Used to
// surface failures that would otherwise only reach the console.
let toastTimer = null;
export function showToast(message) {
    let toast = document.getElementById('ui-toast');
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'ui-toast';
        document.body.appendChild(toast);
    }
    toast.textContent = message;
    toast.classList.add('visible');
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toast.classList.remove('visible'), 3000);
}

export function resetRecordButton() {
    const btn = document.getElementById('btn-record');
    btn.classList.remove('recording');
    btn.textContent = 'RECORD';
}
