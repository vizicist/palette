export const stepperNumSteps = 8;

export const samplePlaybackQuantValues = [0, 0.25, 0.5, 1];

export const patchSigils = {
    A: 'chaos',
    B: 'oracle',
    C: 'sacred',
    D: 'directive'
};

export function normalizeInitialPage(page) {
    const value = String(page || '').trim().toLowerCase();
    return ['pro', 'bss'].includes(value) ? value : 'pro';
}

export const UIState = {
    currentPatch: '*',
    currentCategory: 'quad',
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
        return `${this.currentCategory}:${patch}`;
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
