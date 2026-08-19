export const stepperNumSteps = 8;

export const samplePlaybackQuantValues = [0, 0.25, 0.5, 1];

export const patchNames = ['A', 'B', 'C', 'D'];

export const patchSigils = {
    A: 'chaos',
    B: 'oracle',
    C: 'sacred',
    D: 'directive'
};

// normalizeAttractVideoDestination mirrors the engine's own normalizing (see
// attractvideo.go): anything that isn't "gui" means the videos play on the
// Resolume output and this GUI has nothing to do with them.
export function normalizeAttractVideoDestination(value) {
    return String(value || '').trim().toLowerCase() === 'gui' ? 'gui' : 'main';
}

export function normalizeInitialPage(page) {
    const value = String(page || '').trim().toLowerCase();
    return ['pro', 'bss', 'pro2', 'goat'].includes(value) ? value : 'pro';
}

// themes lists the pro2 quad themes. Each curated theme is a `quad_*` directory
// of link files pointing at the real presets in the master `quad` directory; a
// theme is a curated subset of those presets. Add a theme here to expose it in
// the Theme Selector.
//
// The "All" theme is special: it is backed by the master `quad` directory
// itself, so it always shows every preset (including ones not linked into any
// curated theme). It is only shown in advanced mode (`advancedOnly`) and cannot
// be a copy/move destination (`masterView`) since its contents are automatic.
export const themes = [
    { name: 'Default', dir: 'quad_default' },
    { name: 'Chill', dir: 'quad_chill' },
    { name: 'Goat', dir: 'quad_goat' },
    { name: 'Melodic', dir: 'quad_melodic' },
    { name: 'Rhythmic', dir: 'quad_rhythmic' },
    { name: 'All', dir: 'quad', advancedOnly: true, masterView: true }
];

export const defaultThemeDir = themes[0].dir;

export function themeForDir(dir) {
    return themes.find(theme => theme.dir === dir) || null;
}

export function isThemeDir(dir) {
    return themes.some(theme => theme.dir === dir);
}

export const UIState = {
    currentPatch: '*',
    currentCategory: 'quad',
    currentTheme: defaultThemeDir,
    advancedMode: false,
    lastSinglePatch: 'A',
    showingParams: false,
    activeAdventure: null,
    initialPage: 'pro',
    selectedPresets: new Map(),
    cursorActivityCounts: { A: 0, B: 0, C: 0, D: 0 },
    stepperTiming: {
        playing: false,
        click: 0,
        clicksPerSecond: 0,
        stepLength: 1,
        receivedAt: 0
    },
    paramDefs: null,
    paramEnums: null,
    attractModeActive: false,
    attractAllowGui: false,
    // Whether the Theme Selector is offered at all, from global.showthemes. When
    // it is off the themes are not shown and the Quad presets come from the
    // default theme's directory.
    showThemes: true,
    // Whether the Show Goats button is offered, from global.showgoatsbutton.
    showGoatsButton: true,
    // Whether attract mode is on, as the engine last reported it. Kept because
    // what the GUI shows for it depends on several other things that change
    // independently of the status that carried it.
    attractModeOn: false,
    // global.attractvideos and global.attractvideodestination. Together they
    // say whether the attract videos should play on this screen; the file names
    // to play come from global.attractvideolist.
    attractVideos: true,
    attractVideoDestination: 'main',
    // global.attractvideoresize: whether a video that isn't the shape of the
    // screen fills it and loses its edges, rather than fitting inside it with
    // black bars.
    attractVideoResize: false,
    attractVideoFiles: [],
    attractVideoPlaying: false,
    helpVisible: false,

    wantsStepperStatus() {
        return this.activeAdventure === 'sigil' || (this.activeAdventure === 'space' && this.initialPage === 'bss');
    },

    wantsCursorActivity() {
        return this.wantsStepperStatus();
    },

    presetKey() {
        const patch = this.currentCategory === 'global' ? '*' : this.currentPatch;
        // Use the backend's saved-category key exactly. Status snapshots and
        // loads from other clients identify themed quads as "quad_goat:*",
        // for example, so a UI-only "quad@quad_goat:*" key never matched.
        const category = this.currentCategory === 'quad'
            ? this.currentTheme
            : this.currentCategory;
        return `${category}:${patch}`;
    },

    setInitialPage(page) {
        this.initialPage = normalizeInitialPage(page);
    },

    setActiveAdventure(adventure) {
        this.activeAdventure = adventure;
    },

    setAdvancedMode(enabled) {
        this.advancedMode = !!enabled;
    },

    setTheme(dir) {
        this.currentTheme = isThemeDir(dir) ? dir : defaultThemeDir;
    },

    // savedCategory maps a UI category to the saved directory to read/write
    // preset files from. Quad presets live in the current theme's directory;
    // every other category is theme-independent. Parameter definitions/inits
    // are NOT preset files, so they must keep using the bare category name.
    savedCategory(category) {
        return category === 'quad' ? this.currentTheme : category;
    },

    resetNormalPresetView() {
        this.currentCategory = 'quad';
        this.currentPatch = '*';
        this.showingParams = false;
    },

    toggleCategory(category) {
        if (category === this.currentCategory) {
            this.showingParams = !this.showingParams;
        } else {
            this.currentCategory = category;
            this.showingParams = false;
        }
    },

    selectPatch(patch) {
        if (patch === '*') {
            this.currentPatch = this.currentPatch === '*' ? this.lastSinglePatch : '*';
            return;
        }
        this.lastSinglePatch = patch;
        this.currentPatch = patch;
    },

    syncStepperTiming(status) {
        this.stepperTiming.playing = !!status.playing;
        this.stepperTiming.click = Number(status.click) || 0;
        this.stepperTiming.clicksPerSecond = Number(status.clicks_per_second) || 0;
        this.stepperTiming.stepLength = Math.max(1, Number(status.step_length) || 1);
        this.stepperTiming.receivedAt = performance.now();
    }
};
