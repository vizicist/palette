import assert from 'node:assert/strict';
import { UIState } from '../kit/webui/state.js';
import { createPatchSelectionController } from '../kit/webui/patch-selection.js';

function setup(level = 2, send) {
    const state = { ...UIState, currentPatch: '*', lastSinglePatch: 'A', previewPatch: null };
    state.setGUILevel(level);
    const calls = [];
    const controller = createPatchSelectionController(state, async (patch, bypass) => {
        calls.push([patch, bypass]);
        if (send) await send(patch, bypass);
    });
    return { state, calls, ...controller };
}

// Select, preview, then select again: preset targeting stays on A throughout.
{
    const t = setup();
    await t.select('A');
    assert.equal(t.state.currentPatch, 'A');
    assert.equal(t.state.previewPatch, null);
    assert.deepEqual(t.calls, []);
    await t.select('A');
    assert.equal(t.state.currentPatch, 'A');
    assert.equal(t.state.previewPatch, 'A');
    await t.select('A');
    assert.equal(t.state.previewPatch, null);
    assert.equal(t.state.currentPatch, 'A');
    assert.deepEqual(t.calls, [['A', true], ['A', false]]);
}

// Rapid taps must restore the old layer before bypassing a different layer.
{
    const t = setup(2, () => new Promise(resolve => setTimeout(resolve, 1)));
    await Promise.all(['A', 'A', 'B', 'B', '*'].map(pad => t.select(pad)));
    assert.equal(t.state.currentPatch, '*');
    assert.equal(t.state.previewPatch, null);
    assert.deepEqual(t.calls, [['A', true], ['A', false], ['B', true], ['B', false]]);
    await t.select('*');
    assert.equal(t.state.currentPatch, 'B'); // Preserve the original * toggle.
    assert.equal(t.state.previewPatch, null);
}

// Advanced/normal mode retain repeated-click selection without bypassing.
for (const level of [0, 1]) {
    const t = setup(level);
    await t.select('D');
    await t.select('D');
    assert.equal(t.state.currentPatch, 'D');
    assert.equal(t.state.previewPatch, null);
    assert.deepEqual(t.calls, []);
}

// A failed restore must keep the old preview identified so it can be retried.
{
    let failRestore = true;
    const t = setup(2, (_pad, bypass) => {
        if (!bypass && failRestore) throw new Error('restore failed');
    });
    await t.select('C');
    await t.select('C');
    await assert.rejects(t.select('D'), /restore failed/);
    assert.equal(t.state.previewPatch, 'C');
    assert.equal(t.state.currentPatch, 'C');
    failRestore = false;
    await t.select('D');
    assert.equal(t.state.previewPatch, null);
    assert.equal(t.state.currentPatch, 'D');
    await t.select('D');
    t.state.setGUILevel(0);
    await t.clearPreview();
    assert.equal(t.state.previewPatch, null);
    assert.deepEqual(t.calls.at(-1), ['D', false]);
}

// Do not show Preview when enabling the bypass fails.
{
    const t = setup(2, () => { throw new Error('send failed'); });
    await t.select('A');
    await assert.rejects(t.select('A'), /send failed/);
    assert.equal(t.state.currentPatch, 'A');
    assert.equal(t.state.previewPatch, null);
}

console.log('Patch selection and preview checks passed.');
