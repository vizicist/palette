// Keep bypass changes in click order, including when Resolume/API replies lag.
export function createPatchSelectionController(state, setPreview) {
    let pending = Promise.resolve();

    function enqueue(action) {
        const result = pending.then(action);
        pending = result.catch(() => {});
        return result;
    }

    async function clearPreview() {
        if (!state.previewPatch) return;
        await setPreview(state.previewPatch, false);
        state.previewPatch = null;
    }

    return {
        select(patch) {
            return enqueue(async () => {
                const preview = state.guiLevel === 2 && patch !== '*'
                    && state.currentPatch === patch && state.previewPatch !== patch;
                await clearPreview();
                if (preview) {
                    await setPreview(patch, true);
                    state.previewPatch = patch;
                }
                state.selectPatch(patch);
            });
        },
        clearPreview() {
            return enqueue(clearPreview);
        }
    };
}
