// Recording UI: the RECORD button (with countdown), the record-options modal,
// the recordings list page, and the in-app video player overlay.
import { API } from './api.js';
import { resetRecordButton, showToast, updateRecordButton } from './render.js';

let recordCountdownInterval = null;

export function updateRecordButtonVisibility(running = false) {
    const btn = document.getElementById('btn-record');
    if (!btn) return;
    btn.style.display = running ? '' : 'none';
}

export function handleOBSRecordStatus(status) {
    updateRecordButtonVisibility(!!(status && status.obsrunning));
    if (status && status.recording) {
        startRecordUI(status.remaining);
    } else {
        stopRecordUI();
    }
}

// setupRecording wires the RECORD button, the record-options modal, and the
// recordings/player overlays. Call once at startup.
export function setupRecording() {
    document.getElementById('btn-record').addEventListener('click', async () => {
        const btn = document.getElementById('btn-record');
        if (btn.classList.contains('recording')) {
            // Already recording: pressing REC stops it early.
            try {
                await API.obsRecordStop();
            } catch (e) {
                console.error('Failed to stop recording:', e);
                showToast('Failed to stop recording');
            }
            stopRecordUI();
        } else {
            // Not recording: offer to start a recording or list existing ones.
            showRecordModal();
        }
    });

    document.getElementById('record-overlay').addEventListener('click', (e) => {
        const btn = e.target.closest('.record-opt-btn');
        if (!btn) return;
        const action = btn.dataset.action;
        if (action === 'start') {
            startRecordingFromModal();
        } else if (action === 'list') {
            showRecordingsPage();
        } else {
            hideRecordModal();
        }
    });

    document.getElementById('recordings-back').addEventListener('click', hideRecordingsPage);
    document.getElementById('player-back').addEventListener('click', closePlayer);
}

function startRecordUI(remaining) {
    const btn = document.getElementById('btn-record');
    btn.classList.add('recording');

    // Re-sync to the authoritative remaining value, then tick down locally
    // once per second. Status notifications only fire on start/stop, so the
    // local interval is what makes the REC button count down in between.
    let secondsLeft = Math.max(0, Math.round(remaining));
    updateRecordButton(secondsLeft);

    if (recordCountdownInterval) {
        clearInterval(recordCountdownInterval);
    }
    recordCountdownInterval = setInterval(() => {
        secondsLeft = Math.max(0, secondsLeft - 1);
        updateRecordButton(secondsLeft);
        if (secondsLeft <= 0) {
            clearInterval(recordCountdownInterval);
            recordCountdownInterval = null;
        }
    }, 1000);
}

function stopRecordUI() {
    if (recordCountdownInterval) {
        clearInterval(recordCountdownInterval);
        recordCountdownInterval = null;
    }
    resetRecordButton();
}

// Record options modal + recordings list page
function showRecordModal() {
    document.getElementById('record-overlay').classList.remove('hidden');
}

function hideRecordModal() {
    document.getElementById('record-overlay').classList.add('hidden');
}

async function startRecordingFromModal() {
    hideRecordModal();
    try {
        const result = await API.obsRecord();
        if (result && result.recording) {
            startRecordUI(result.remaining);
        } else {
            showRecordError();
        }
    } catch (e) {
        console.error('Failed to start recording:', e);
        showRecordError();
    }
}

// Flag on the RECORD button (and via a toast) that a recording failed to
// start (e.g. OBS rejected the connection), so the failure isn't silent.
function showRecordError() {
    const btn = document.getElementById('btn-record');
    if (recordCountdownInterval) {
        clearInterval(recordCountdownInterval);
        recordCountdownInterval = null;
    }
    btn.classList.remove('recording');
    btn.classList.add('record-error');
    btn.textContent = 'REC FAILED';
    showToast('Recording failed to start — check OBS connection');
    setTimeout(() => {
        btn.classList.remove('record-error');
        resetRecordButton();
    }, 2500);
}

async function showRecordingsPage() {
    hideRecordModal();
    const list = document.getElementById('recordings-list');
    list.innerHTML = '<div class="recordings-empty">Loading…</div>';
    document.getElementById('recordings-overlay').classList.remove('hidden');
    try {
        const recordings = await API.obsRecordList();
        renderRecordings(recordings);
    } catch (e) {
        console.error('Failed to list recordings:', e);
        list.innerHTML = '<div class="recordings-empty">Failed to load recordings.</div>';
    }
}

function hideRecordingsPage() {
    document.getElementById('recordings-overlay').classList.add('hidden');
}

function renderRecordings(recordings) {
    const list = document.getElementById('recordings-list');
    list.innerHTML = '';
    if (!Array.isArray(recordings) || recordings.length === 0) {
        list.innerHTML = '<div class="recordings-empty">No recordings yet.</div>';
        return;
    }
    for (const rec of recordings) {
        const row = document.createElement('div');
        row.className = 'recording-row';

        const info = document.createElement('div');
        info.className = 'recording-info';

        const name = document.createElement('span');
        name.className = 'recording-name';
        name.textContent = rec.name;

        const meta = document.createElement('span');
        meta.className = 'recording-meta';
        meta.textContent = [
            formatRecordingDuration(rec.duration),
            formatRecordingSize(rec.size),
            formatRecordingTime(rec.modtime)
        ].filter(Boolean).join(' · ');

        info.appendChild(name);
        info.appendChild(meta);

        const play = document.createElement('button');
        play.className = 'recording-play modal-button';
        play.type = 'button';
        play.textContent = '▶ Play';
        play.addEventListener('click', () => playRecording(rec.name));

        row.appendChild(info);
        row.appendChild(play);
        list.appendChild(row);
    }
}

function playRecording(name) {
    const video = document.getElementById('player-video');
    document.getElementById('player-title').textContent = name;
    video.src = `/recordings/${encodeURIComponent(name)}`;
    document.getElementById('player-overlay').classList.remove('hidden');
    video.play().catch(() => { /* autoplay may be blocked; controls remain */ });
}

function closePlayer() {
    const video = document.getElementById('player-video');
    video.pause();
    video.removeAttribute('src');
    video.load();
    document.getElementById('player-overlay').classList.add('hidden');
}

function formatRecordingDuration(seconds) {
    if (typeof seconds !== 'number' || seconds <= 0) return '';
    const total = Math.round(seconds);
    const m = Math.floor(total / 60);
    const s = total % 60;
    return `${m}:${String(s).padStart(2, '0')}`;
}

function formatRecordingSize(bytes) {
    if (typeof bytes !== 'number' || bytes < 0) return '';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let n = bytes;
    let i = 0;
    while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
    return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function formatRecordingTime(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    return isNaN(d.getTime()) ? iso : d.toLocaleString();
}
