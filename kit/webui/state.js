export const stepperNumSteps = 8;

export const samplePlaybackQuantValues = [0, 0.25, 0.5, 1];

export const patchNames = ['A', 'B', 'C', 'D'];

export const patchSigils = {
    A: 'chaos',
    B: 'oracle',
    C: 'sacred',
    D: 'directive'
};

export function normalizeInitialPage(page) {
    const value = String(page || '').trim().toLowerCase();
    return ['pro', 'bss', 'pro2'].includes(value) ? value : 'pro';
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
    helpVisible: false,

    wantsStepperStatus() {
        return this.activeAdventure === 'sigil' || (this.activeAdventure === 'space' && this.initialPage === 'bss');
    },

    wantsCursorActivity() {
        return this.wantsStepperStatus();
    },

    presetKey() {
        const patch = this.currentCategory === 'global' ? '*' : this.currentPatch;
        // Quad selections are tracked per theme so switching themes shows the
        // right highlighted preset (and none if that theme has no selection).
        const category = this.currentCategory === 'quad'
            ? `quad@${this.currentTheme}`
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
