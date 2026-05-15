import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, join, normalize, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, '..');
const webuiDir = join(repoRoot, 'kit', 'webui');
const requireLive = process.argv.includes('--require-live');

let failures = 0;

function pass(message) {
    console.log(`PASS ${message}`);
}

function fail(message) {
    failures++;
    console.error(`FAIL ${message}`);
}

function warn(message) {
    console.warn(`WARN ${message}`);
}

function assert(condition, message) {
    if (condition) pass(message);
    else fail(message);
}

function liveAssert(condition, message) {
    if (condition) {
        pass(message);
    } else if (requireLive) {
        fail(message);
    } else {
        warn(`${message} (live server may be running an older installed build)`);
    }
}

function readWebUIFile(name) {
    const path = join(webuiDir, name);
    if (!existsSync(path)) {
        fail(`${name} exists`);
        return '';
    }
    pass(`${name} exists`);
    return readFileSync(path, 'utf8');
}

function checkSyntax(name) {
    const path = join(webuiDir, name);
    try {
        execFileSync('node', ['--check', path], { stdio: 'pipe' });
        pass(`${name} parses as JavaScript`);
    } catch (err) {
        fail(`${name} failed node --check\n${String(err.stderr || err.message)}`);
    }
}

function importedModules(name, source) {
    const imports = [];
    const importPattern = /(?:import\s+(?:[^'"]+?\s+from\s+)?|import\s*\()\s*['"](\.\/[^'"]+)['"]/g;
    let match;
    while ((match = importPattern.exec(source)) !== null) {
        imports.push(match[1]);
    }
    return imports.map(specifier => normalize(join(dirname(join(webuiDir, name)), specifier)));
}

async function fetchText(url, options) {
    const response = await fetch(url, options);
    const text = await response.text();
    return { response, text };
}

async function checkLiveServer() {
    const baseURL = 'http://127.0.0.1:3330';
    let index;
    try {
        index = await fetchText(`${baseURL}/?touchscreen=1`);
    } catch (err) {
        if (requireLive) fail(`live web UI reachable at ${baseURL}: ${err.message}`);
        else warn(`live web UI not reachable at ${baseURL}; static checks still ran`);
        return;
    }

    liveAssert(index.response.ok, 'live index returns HTTP 200');
    liveAssert(index.text.includes('Transmission and Oscillation Recontextualizer'), 'live index has app title');
    const liveIndexIsCurrent = index.text.includes('type="module" src="app.js"');
    liveAssert(liveIndexIsCurrent, 'live index serves module entry point');

    for (const name of ['app.js', 'api.js', 'state.js', 'render.js', 'style.css']) {
        try {
            const result = await fetchText(`${baseURL}/${name}`);
            liveAssert(result.response.ok, `live ${name} returns HTTP 200`);
            liveAssert(result.text.length > 0, `live ${name} is non-empty`);
        } catch (err) {
            if (requireLive) fail(`live ${name} fetch failed: ${err.message}`);
            else warn(`live ${name} fetch failed: ${err.message}`);
        }
    }

    try {
        const status = await fetchText(`${baseURL}/api`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ api: 'global.status' })
        });
        liveAssert(status.response.ok, 'live global.status returns HTTP 200');
        const data = JSON.parse(status.text);
        liveAssert(data && typeof data === 'object' && !data.error, 'live global.status returns JSON without error');
    } catch (err) {
        if (requireLive) fail(`live global.status failed: ${err.message}`);
        else warn(`live global.status failed: ${err.message}`);
    }

    try {
        const stepper = await fetchText(`${baseURL}/api`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ api: 'stepper.status' })
        });
        liveAssert(stepper.response.ok, 'live stepper.status returns HTTP 200');
        const outer = JSON.parse(stepper.text);
        const result = typeof outer.result === 'string' ? JSON.parse(outer.result) : outer.result;
        liveAssert(result && result.tracks && result.tracks.A && result.tracks.B && result.tracks.C && result.tracks.D, 'live stepper.status has A/B/C/D tracks');
    } catch (err) {
        if (requireLive) fail(`live stepper.status failed: ${err.message}`);
        else warn(`live stepper.status failed: ${err.message}`);
    }
}

console.log(`Palette web UI smoke test`);
console.log(`repoRoot=${repoRoot}`);

const index = readWebUIFile('index.html');
const app = readWebUIFile('app.js');
const api = readWebUIFile('api.js');
const state = readWebUIFile('state.js');
const render = readWebUIFile('render.js');

assert(index.includes('<script type="module" src="app.js"></script>'), 'index uses app.js as module entry point');
assert(!index.includes('<script src="api.js"></script>'), 'index does not load api.js as a legacy global script');

for (const id of [
    'app-title',
    'btn-nav-space',
    'btn-nav-sigil',
    'sigil-screen',
    'palette-pad-stage',
    'transmission-controls-panel',
    'transmission-quant',
    'transmission-words',
    'transmission-newset',
    'preset-grid'
]) {
    assert(index.includes(`id="${id}"`), `index contains #${id}`);
}

assert(app.includes("import { API } from './api.js';"), 'app imports API module');
assert(app.includes("from './state.js';"), 'app imports UI state module');
assert(app.includes("from './render.js';"), 'app imports render module');
assert(api.includes('window.API = API'), 'api preserves window.API for browser-console use');
assert(state.includes('export const UIState'), 'state exports UIState');
assert(render.includes('export function updatePalettePadRoute'), 'render exports pad route rendering');

for (const name of ['app.js', 'api.js', 'state.js', 'render.js']) {
    checkSyntax(name);
}

const moduleSources = new Map([
    ['app.js', app],
    ['api.js', api],
    ['state.js', state],
    ['render.js', render]
]);
for (const [name, source] of moduleSources) {
    for (const imported of importedModules(name, source)) {
        const relative = imported.startsWith(webuiDir) ? imported.slice(webuiDir.length + 1) : imported;
        assert(existsSync(imported), `${name} import ${relative} resolves`);
    }
}

await checkLiveServer();

if (failures > 0) {
    console.error(`web UI smoke test failed with ${failures} failure(s)`);
    process.exit(1);
}
console.log('web UI smoke test passed');
