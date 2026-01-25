const API = {
    async call(api, params = {}) {
        const body = JSON.stringify({ api, ...params });
        const resp = await fetch('/api', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body
        });
        const data = await resp.json();
        if (data.error) throw new Error(data.error);
        // Result may be a JSON string that needs parsing
        let result = data.result;
        if (typeof result === 'string' && (result.startsWith('{') || result.startsWith('['))) {
            try {
                result = JSON.parse(result);
            } catch (e) {
                // Not JSON, return as-is
            }
        }
        return result;
    },

    getStatus() { return this.call('global.status'); },
    getSavedList(category) { return this.call('saved.list', { category }); },
    loadQuad(filename) { return this.call('quad.load', { category: 'quad', filename }); },
    audioReset() { return this.call('global.audio_reset'); },
    completeReset() { return this.audioReset(); },
    softReset() { return this.audioReset(); }
};
