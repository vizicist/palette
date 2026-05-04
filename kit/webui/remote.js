(function () {
    const params = new URLSearchParams(window.location.search);
    const targetHostEl = document.getElementById('target-host');
    const transportStateEl = document.getElementById('transport-state');
    const controller = document.getElementById('palette-controller');
    const activePointers = new Map();
    const defaultPressure = 0.25;
    const maxPressure = 1.0;
    const lingerDelayMs = 180;
    const pressureRisePerSecond = 0.55;
    const lingerMoveTolerance = 0.018;
    let targetHost = params.get('host') || '';
    let nextGid = Math.floor(Date.now() % 1000000);
    let lastSentAt = 0;
    let natsConnected = false;

    document.addEventListener('DOMContentLoaded', init);

    async function init() {
        if (params.get('labels') === '1' || params.get('padlabels') === '1') {
            document.body.classList.add('show-pad-labels');
        }
        await loadTargetHost();
        setTransportState(natsConnected ? 'ready' : 'error', natsConnected ? 'NATS proxy ready' : 'NATS disconnected');
        document.querySelectorAll('.pad').forEach(bindSurface);
    }

    async function loadTargetHost() {
        if (targetHost) {
            targetHostEl.textContent = targetHost;
            return;
        }
        try {
            const resp = await fetch('/api', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ api: 'global.status' })
            });
            const data = await resp.json();
            let status = data.result;
            if (typeof status === 'string') status = JSON.parse(status);
            targetHost = status.hostname || '';
            natsConnected = status.natsconnected === true || status.natsconnected === 'true';
        } catch (err) {
            targetHost = '';
            natsConnected = false;
        }
        targetHostEl.textContent = targetHost || 'local Palette';
    }

    function bindSurface(el) {
        el.addEventListener('pointerdown', onPointerDown);
        el.addEventListener('pointermove', onPointerMove);
        el.addEventListener('pointerup', onPointerUp);
        el.addEventListener('pointercancel', onPointerUp);
        el.addEventListener('lostpointercapture', onPointerUp);
    }

    function onPointerDown(event) {
        event.preventDefault();
        const source = event.currentTarget.dataset.source;
        if (!source) return;
        const gid = nextGid++;
        const coordinateEl = coordinateElementForEvent(event, source);
        const pos = normalizedElementPosition(event, coordinateEl);
        activePointers.set(event.pointerId, {
            gid,
            source,
            el: event.currentTarget,
            coordinateEl,
            pressure: defaultPressure,
            lastPos: pos,
            lingerStartedAt: performance.now(),
            pressureTimer: null
        });
        const active = activePointers.get(event.pointerId);
        event.currentTarget.setPointerCapture(event.pointerId);
        event.currentTarget.classList.add('active');
        startPressureTimer(event.pointerId);
        sendPointerEvent('down', event, active, true);
    }

    function onPointerMove(event) {
        const active = activePointers.get(event.pointerId);
        if (!active) return;
        event.preventDefault();
        const now = performance.now();
        updatePressureFromMove(event, active, now);
        if (now - lastSentAt < 24) return;
        lastSentAt = now;
        sendPointerEvent('drag', event, active, false);
    }

    function onPointerUp(event) {
        const active = activePointers.get(event.pointerId);
        if (!active) return;
        event.preventDefault();
        activePointers.delete(event.pointerId);
        stopPressureTimer(active);
        active.el.classList.remove('active');
        sendPointerEvent('up', event, active, true);
    }

    function coordinateElementForEvent(event, source) {
        return event.currentTarget.dataset.source ? event.currentTarget : controller;
    }

    function normalizedElementPosition(event, el) {
        const rect = el.getBoundingClientRect();
        return {
            x: clamp((event.clientX - rect.left) / rect.width),
            y: 1 - clamp((event.clientY - rect.top) / rect.height)
        };
    }

    function updatePressureFromMove(event, active, now) {
        const pos = normalizedElementPosition(event, active.coordinateEl);
        if (distance(pos, active.lastPos) > lingerMoveTolerance) {
            active.lastPos = pos;
            active.lingerStartedAt = now;
        } else if (now - active.lingerStartedAt > lingerDelayMs) {
            const elapsed = (now - active.lingerStartedAt - lingerDelayMs) / 1000;
            active.pressure = clampToRange(defaultPressure + elapsed * pressureRisePerSecond, defaultPressure, maxPressure);
        }
    }

    function startPressureTimer(pointerId) {
        const active = activePointers.get(pointerId);
        if (!active) return;
        active.pressureTimer = setInterval(() => {
            const latest = activePointers.get(pointerId);
            if (!latest) return;
            const now = performance.now();
            if (now - latest.lingerStartedAt <= lingerDelayMs) return;
            const elapsed = (now - latest.lingerStartedAt - lingerDelayMs) / 1000;
            latest.pressure = clampToRange(defaultPressure + elapsed * pressureRisePerSecond, defaultPressure, maxPressure);
            sendPressureDrag(pointerId, latest);
        }, 80);
    }

    function stopPressureTimer(active) {
        if (active.pressureTimer) {
            clearInterval(active.pressureTimer);
            active.pressureTimer = null;
        }
    }

    function sendPressureDrag(pointerId, active) {
        const pos = active.lastPos;
        const payload = {
            host: targetHost,
            api: 'cursor.event',
            ddu: 'drag',
            source: active.source,
            gid: String(active.gid),
            x: pos.x.toFixed(5),
            y: pos.y.toFixed(5),
            z: active.pressure.toFixed(5),
            area: '0.00100'
        };
        postNats(payload);
    }

    async function sendPointerEvent(ddu, event, active, immediate) {
        const pos = normalizedElementPosition(event, active.coordinateEl);
        if (ddu !== 'up') active.lastPos = pos;
        const payload = {
            host: targetHost,
            api: 'cursor.event',
            ddu,
            source: active.source,
            gid: String(active.gid),
            x: pos.x.toFixed(5),
            y: pos.y.toFixed(5),
            z: (ddu === 'up' ? 0 : active.pressure).toFixed(5),
            area: pointerArea(event).toFixed(5)
        };

        if (immediate) {
            await postNats(payload);
        } else {
            postNats(payload);
        }
    }

    async function postNats(payload) {
        try {
            const resp = await fetch('/nats/api', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            const data = await resp.json();
            if (!resp.ok || data.error) throw new Error(data.error || resp.statusText);
            setTransportState('ready', 'NATS proxy ready');
        } catch (err) {
            setTransportState('error', err.message || 'NATS proxy error');
        }
    }

    function pointerArea(event) {
        const w = event.width || 1;
        const h = event.height || 1;
        return Math.max(0.001, Math.min(1, (w * h) / 10000));
    }

    function clamp(value) {
        return Math.max(0, Math.min(1, value));
    }

    function clampToRange(value, min, max) {
        return Math.max(min, Math.min(max, value));
    }

    function distance(a, b) {
        const dx = a.x - b.x;
        const dy = a.y - b.y;
        return Math.sqrt(dx * dx + dy * dy);
    }

    function setTransportState(kind, message) {
        transportStateEl.className = kind;
        transportStateEl.textContent = message;
    }
})();
