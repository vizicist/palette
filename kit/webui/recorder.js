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

// Whether YouTube upload credentials are configured in the engine; arrives
// with every obsrecord snapshot and decides whether upload buttons render.
let youtubeConfigured = false;

export function handleOBSRecordStatus(status) {
    updateRecordButtonVisibility(!!(status && status.obsrunning));
    youtubeConfigured = !!(status && status.youtubeconfigured);
    if (status && status.recording) {
        startRecordUI(status.remaining);
    } else {
        stopRecordUI();
    }
    handleUploadStatus(status && status.upload);
}

// Toast when a YouTube upload finishes or fails. Upload state arrives with
// every obsrecord snapshot, so remember the last state to only announce
// transitions.
let lastUploadAnnounced = '';

function handleUploadStatus(upload) {
    if (!upload || upload.state === 'uploading') return;
    const key = `${upload.state}:${upload.file}:${upload.url || upload.error || ''}`;
    if (key === lastUploadAnnounced) return;
    lastUploadAnnounced = key;
    if (upload.state === 'done') {
        showToast(`Uploaded ${recordingDisplayName(upload.file)} to YouTube`);
    } else if (upload.state === 'error') {
        showToast(`YouTube upload of ${recordingDisplayName(upload.file)} failed: ${upload.error}`);
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
    setupPlayerSeek();
}

// The video's built-in controls auto-hide during playback, so a separate
// always-visible seek bar sits under the video. While the user is dragging
// it, timeupdate must not fight the thumb position.
function setupPlayerSeek() {
    const video = document.getElementById('player-video');
    const seek = document.getElementById('player-seek');
    let dragging = false;

    video.addEventListener('timeupdate', () => {
        if (!dragging && video.duration > 0) {
            seek.value = String(Math.round((video.currentTime / video.duration) * 1000));
        }
    });
    seek.addEventListener('pointerdown', () => { dragging = true; });
    seek.addEventListener('pointerup', () => { dragging = false; });
    seek.addEventListener('pointercancel', () => { dragging = false; });
    seek.addEventListener('input', () => {
        if (video.duration > 0) {
            video.currentTime = (Number(seek.value) / 1000) * video.duration;
        }
    });
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

// recordingDisplayName shows the generated filename without its .mp4 suffix.
function recordingDisplayName(filename) {
    return filename.replace(/\.mp4$/i, '');
}

function renderRecordings(recordings) {
    const list = document.getElementById('recordings-list');
    list.innerHTML = '';
    if (!Array.isArray(recordings) || recordings.length === 0) {
        list.innerHTML = '<div class="recordings-empty">No recordings yet.</div>';
        return;
    }
    for (const rec of recordings) {
        // The name gets its own full-width line so it is never truncated;
        // the meta text and action buttons share the line below it.
        const row = document.createElement('div');
        row.className = 'recording-row';

        const name = document.createElement('span');
        name.className = 'recording-name';
        name.textContent = recordingDisplayName(rec.name);

        const bottom = document.createElement('div');
        bottom.className = 'recording-bottom';

        const meta = document.createElement('span');
        meta.className = 'recording-meta';
        meta.textContent = [
            formatRecordingDuration(rec.duration),
            formatRecordingSize(rec.size),
            formatRecordingTime(rec.modtime)
        ].filter(Boolean).join(' · ');

        const actions = document.createElement('div');
        actions.className = 'recording-actions';

        const play = document.createElement('button');
        play.className = 'recording-play modal-button';
        play.type = 'button';
        play.textContent = '▶ Play';
        play.addEventListener('click', () => playRecording(rec.name));

        const del = document.createElement('button');
        del.className = 'recording-delete modal-button';
        del.type = 'button';
        del.textContent = '🗑 Delete';
        del.addEventListener('click', () => deleteRecording(rec.name));

        actions.appendChild(play);
        if (youtubeConfigured) {
            const upload = document.createElement('button');
            upload.className = 'recording-upload modal-button';
            upload.type = 'button';
            upload.textContent = '⇧ YouTube';
            upload.addEventListener('click', () => uploadRecording(rec.name));
            actions.appendChild(upload);
        }
        actions.appendChild(del);

        bottom.appendChild(meta);
        bottom.appendChild(actions);

        row.appendChild(name);
        row.appendChild(bottom);
        list.appendChild(row);
    }
}

async function uploadRecording(name) {
    if (!window.confirm(`Upload ${recordingDisplayName(name)} to YouTube?`)) return;
    try {
        await API.youtubeUpload(name);
        showToast(`Uploading ${recordingDisplayName(name)} to YouTube…`);
    } catch (e) {
        console.error('Failed to start YouTube upload:', e);
        showToast(`Upload failed: ${e.message}`);
    }
}

async function deleteRecording(name) {
    if (!window.confirm(`Delete ${recordingDisplayName(name)}? This cannot be undone.`)) return;
    try {
        await API.obsRecordDelete(name);
        showToast(`Deleted ${recordingDisplayName(name)}`);
    } catch (e) {
        console.error('Failed to delete recording:', e);
        showToast(`Delete failed: ${e.message}`);
        return;
    }
    // Refresh the list so the row disappears.
    try {
        renderRecordings(await API.obsRecordList());
    } catch (e) {
        console.error('Failed to refresh recordings:', e);
    }
}

function playRecording(name) {
    const video = document.getElementById('player-video');
    document.getElementById('player-title').textContent = recordingDisplayName(name);
    document.getElementById('player-seek').value = '0';
    video.src = `/recordings/${encodeURIComponent(name)}`;
    document.getElementById('player-overlay').classList.remove('hidden');
    video.play().catch(() => { /* autoplay may be blocked; controls remain */ });
}

function closePlayer() {
    const video = document.getElementById('player-video');
    video.pause();
    video.removeAttribute('src');
    video.load();
    document.getElementById('player-seek').value = '0';
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
    if (isNaN(d.getTime())) return iso;
    // Compact form (no seconds) so the whole meta line fits on one line.
    return d.toLocaleString([], { dateStyle: 'short', timeStyle: 'short' });
}
