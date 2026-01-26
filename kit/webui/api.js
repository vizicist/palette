const API = {
    async call(api, params = {}) {
        const body = JSON.stringify({ api, ...params });
        console.log('API call:', api, body);
        try {
            const resp = await fetch('/api', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body
            });
            console.log('API response status:', resp.status);
            const data = await resp.json();
            console.log('API response data:', data);
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
        } catch (e) {
            console.error('API fetch error:', e);
            throw e;
        }
    },

    getStatus() { return this.call('global.status'); },
    getSavedList(category) { return this.call('saved.list', { category }); },
    getParamDefs(category) { return this.call('saved.paramdefs', { category }); },
    getParamDefsJson() { return this.call('saved.paramdefsjson'); },
    getParamInits(category) { return this.call('saved.paraminits', { category }); },
    getParamRands(category) { return this.call('saved.paramrands', { category }); },
    getParamEnums() { return this.call('saved.paramenums'); },
    loadGlobal(filename) { return this.call('global.load', { category: 'global', filename }); },
    loadQuad(filename) { return this.call('quad.load', { category: 'quad', filename }); },
    loadPatch(patch, category, filename) {
        return this.call('patch.load', { patch, category, filename });
    },
    audioReset() { return this.call('global.audio_reset'); },
    completeReset() { return this.audioReset(); },
    softReset() { return this.audioReset(); },

    // Parameter APIs
    getPatchParams(patch, category) {
        return this.call('patch.getparams', { patch, category });
    },
    setPatchParam(patch, name, value) {
        return this.call('patch.set', { patch, name, value });
    },
    // Set multiple params at once (params is an object of name: value pairs)
    setPatchParams(patch, params) {
        return this.call('patch.setparams', { patch, ...params });
    },
    getGlobalParams(prefix) {
        return this.call('global.getwithprefix', { name: prefix });
    },
    setGlobalParam(name, value) {
        return this.call('global.set', { name, value });
    }
};
