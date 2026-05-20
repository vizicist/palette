import { LocalNATS } from './local_nats.js';
import { Subjects } from './subjects.js';

let localNATS = null;

export async function setupUIStateFeed(handlers) {
    localNATS = new LocalNATS();
    await localNATS.connect();
    await Promise.all([
        localNATS.subscribe(Subjects.uiStatus, handlers.status),
        localNATS.subscribe(Subjects.uiStepper, handlers.stepper),
        localNATS.subscribe(Subjects.uiCursor, handlers.cursor),
        localNATS.subscribe(Subjects.uiOBSRecord, handlers.obsRecord)
    ]);
}

export function requestUISnapshot() {
    if (!localNATS) {
        throw new Error('UI NATS feed has not been initialized');
    }
    return localNATS.request(Subjects.uiSnapshotRequest);
}

export function applyUISnapshot(snapshot, handlers) {
    if (snapshot && snapshot.status) handlers.status(snapshot.status);
    if (snapshot && snapshot.stepper) handlers.stepper(snapshot.stepper);
    if (snapshot && snapshot.cursor) handlers.cursor(snapshot.cursor);
    if (snapshot && snapshot.obsrecord) handlers.obsRecord(snapshot.obsrecord);
}
