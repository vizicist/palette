# Palette Review Backlog

This backlog captures the findings from the repository-wide review performed on
2026-08-15. It is ordered by severity first and suggested implementation order
second.

## Ranking

- **P0 — Critical:** Immediate production emergency, active data loss, or broad
  compromise. No P0 findings were identified.
- **P1 — High:** Can freeze or terminate a major runtime component, corrupt
  important state, expose credentials/control, or produce a broken release.
- **P2 — Medium:** Significant correctness, reliability, race, or operational
  problem with a narrower trigger or recoverable impact.
- **P3 — Low:** Lower-impact correctness, maintainability, or defense-in-depth
  improvement.

## Status

Every P1 item has been worked on. 24 of the 27 P1 checkboxes are closed; the
three still open are noted in place:

- **14** - payload publishing is transactional, whole-version swap and
  configuration rollback are not.
- **19** (second box) - the macOS path handling in `PaletteHost.cpp` is unchanged.
- **21** (first box) - the build stops on a failed compile, but there is still no
  required-file manifest or hash check before packaging.

Two caveats that a checked box does not convey:

- **7** is restriction, not authentication. The HTTP server defaults to loopback;
  a wider `PALETTE_HTTP_BIND` still serves an unauthenticated API.
- The FFGL C++ fixes (**5**, **6**, **13**, **22A**) are compile-verified with a
  full MSBuild rebuild, and the macOS and Linux packaging fixes (**19**, **20**)
  are syntax-checked only. None of them has been run on its target.

Nothing in P2 or P3 below has been touched.

## Recommended next batch

1. Finish 14, 19 and 21 (see above).
2. Authenticated HTTP control, for installations that need a LAN-reachable API.
3. Run the macOS and Linux release builds to exercise 19 and 20 for real.
4. Go 1.25.13 and a cross-platform build/test CI workflow.

## P1 — High

### 1. Parameter reads can recursively deadlock

- [x] Fix recursive read locking in
  [`kit/params.go:173-190`](kit/params.go#L173-L190).
- `GetWithPrefix` holds `vals.mutex.RLock()` and calls
  `ParamValueAsString`, which reacquires the same `RWMutex` through
  `paramValue`.
- If a writer queues between those acquisitions, the nested read blocks while
  retaining the outer read lock. The writer and subsequent parameter reads can
  then remain blocked forever.
- The same pattern exists in `GetGlobalParams`, `DoForAllParams`, and
  `JSONValues` at [`kit/params.go:78-100`](kit/params.go#L78-L100).
- Snapshot names and values under one read lock, release it, and then format
  values or invoke callbacks without relocking.
- **Done** in `ccd64b66`. Reads that walk the map use
  `paramValueAsStringLocked`; `GetGlobalParams`, `DoForAllParams` and
  `JSONValues` were fixed too.

### 2. Parameter saving can crash on concurrent map access

- [x] Make `ParamValues.Save` take a synchronized snapshot in
  [`kit/params.go:245-294`](kit/params.go#L245-L294).
- `Save` aliases and iterates `vals.values` without its mutex while concurrent
  `global.set` or `patch.set` calls can mutate that map.
- Ordinary overlapping web UI requests can trigger Go's fatal
  `concurrent map iteration and map write` error.
- Copy a complete string snapshot under `RLock`, then marshal after unlocking.
  Serialize writes targeting the same preset file.
- **Done** in `ccd64b66`. The scan is `paramsForCategory` under one read lock;
  marshalling and the disk write happen outside it.

### 3. Scheduler deletion callbacks can deadlock playback

- [x] Run deletion callbacks after releasing `sched.mutex` in
  [`kit/scheduler.go:277-295`](kit/scheduler.go#L277-L295).
- `deleteScheduledEvents` invokes `onDelete` while holding the scheduler mutex.
- Deleting a sample-backed active cursor reaches `DeleteSamplePlaybackStarts`
  during cleanup, which tries to acquire the same scheduler mutex again.
- Remove and collect matching elements under the lock, unlock, and only then
  run cleanup callbacks.
- **Done** in `65dba98a`. Matches are removed and collected under the lock,
  callbacks run after releasing it.

### 4. Audio teardown and device switching can deadlock

- [x] Separate device lifecycle synchronization from the audio mix mutex in
  [`pkg/samplesplitter/audio.go:535-559`](pkg/samplesplitter/audio.go#L535-L559).
- `Close` and `openDevice` call `malgo.Device.Uninit()` while holding `a.mu`,
  while the realtime callback starts by acquiring that mutex.
- Miniaudio waits for its worker thread during `Uninit`. If the callback is
  waiting on `a.mu`, teardown or an output change waits forever.
- Detach the device under a lifecycle lock, release the mix lock, and then
  uninitialize the detached device.
- **Done** in `3761ef1f`. The device is detached under the lock and
  uninitialised with it released.

### 5. FFGL daemon teardown has undefined behavior and hang paths

- [x] Replace the daemon's busy-wait lifecycle with synchronized shutdown and
  joining in
  [`ffgl/source/lib/palette/PaletteHost.cpp:119-225`](ffgl/source/lib/palette/PaletteHost.cpp#L119-L225).
- `daemon_stopped` is uninitialized, and all lifecycle flags are plain `bool`s
  shared between threads.
- The destructor waits on `daemon_stopped` even if `pthread_create` failed and
  no thread can ever set it.
- Initialize all state, use atomics or a mutex, request shutdown, and
  `pthread_join` only when thread creation succeeded.
- **Done** in `ff886b89`. Flags are atomic, `daemon_stopped` is initialised,
  and the destructor `pthread_join`s only when a thread was created.
  Compile-verified, not run inside Resolume.

### 6. Pending FFGL JSON requests can block teardown forever

- [x] Add cancellation and a bounded wait to
  [`PaletteHost::RespondToJson`](ffgl/source/lib/palette/PaletteHost.cpp#L909-L943).
- The network thread waits indefinitely for `ProcessOpenGL` to service
  `json_pending`.
- If rendering stops or teardown begins first, the network thread cannot
  observe shutdown while the destructor waits for that thread.
- Reject pending requests once GL shutdown starts, broadcast the condition
  during teardown, and use a bounded wait.
- **Done** in `ff886b89`. The wait is bounded (2s) and a timeout clears
  `json_pending`. Compile-verified, not run inside Resolume.

### 7. LAN HTTP control has no authorization

- [x] Restrict or authenticate the API exposed by
  [`kit/engine.go:351-409`](kit/engine.go#L351-L409).
- The HTTP server listens on `0.0.0.0` and exposes mutating APIs without
  authentication or request-origin validation.
- Reachable peers can change persistent boot state, reset processes, remove
  recordings, upload content, or stop the engine.
- Default to loopback, or require an authenticated session/token with
  authorization and Origin/CSRF checks before retaining LAN access.
- Apply small request-body limits to both POST endpoints.
- **Done.** Restricted, NOT authenticated, in `efd39350` and `65dba98a`. The
  server defaults to `LocalAddress` and both POST endpoints cap the body at
  1MB. Anyone who opts back in to a wider bind with `PALETTE_HTTP_BIND` is
  still serving an unauthenticated API - the token/Origin/CSRF work is still
  open.

### 8. An invalid MIDI-thru synth value terminates the scheduler

- [x] Validate and safely resolve `global.midithrusynth` in
  [`kit/scheduler.go:445-474`](kit/scheduler.go#L445-L474).
- String parameter assignment accepts arbitrary values, so
  `Synths[synthName]` can be nil.
- The next MIDI-thru event dereferences the nil synth. `Scheduler.Start`
  recovers the panic and returns, permanently stopping future scheduled events.
- Validate the parameter as an enum, guard the lookup, and isolate malformed
  events so one event cannot terminate the scheduler loop.
- **Done** in `3761ef1f`. `GetSynth` falls back to the dummy synth, with a nil
  guard behind it.

### 9. Overlapping sampleplayer one-shots restart older samples

- [x] Limit channel queue arbitration to looping voices in
  [`pkg/samplesplitter/audio.go:360-405`](pkg/samplesplitter/audio.go#L360-L405).
- `Play` marks the prior channel voice inactive and resets its position for
  every request, including `Loop == false`.
- The mixer suppresses inactive voices only when they are looping, so an older
  one-shot restarts and mixes with the new voice. Removing the new voice can
  reset and restart the older voice again.
- Let non-looping keyed voices mix independently and add an overlapping
  one-shot regression test.
- **Done** in `d3892e49`. Only looping voices are arbitrated, which is what the
  mixer honours.

### 10. A stop can be overtaken by an in-flight play

- [x] Add cancellation generations or reserve a cancellable voice before
  decoding in
  [`pkg/samplesplitter/audio.go:360-405`](pkg/samplesplitter/audio.go#L360-L405).
- Decode and full-segment rendering happen before the voice lock.
- A concurrent `StopVoice`, `StopNote`, `StopAll`, reload, or all-notes-off can
  clear a channel while `Play` is working. `Play` then acquires the lock and
  inserts the voice after the stop.
- **Done** in `d3892e49`. Stops bump per-voice, per-channel and global
  counters; `Play` notes them before decoding and abandons its voice if any
  moved.

### 11. Sample service callbacks race concurrent shutdown

- [x] Pin in-flight service calls against shutdown in
  [`kit/samplesplitter.go:187-199`](kit/samplesplitter.go#L187-L199) and
  [`kit/samplesplitter.go:307-318`](kit/samplesplitter.go#L307-L318).
- `withSamplePlaybackService` releases the global mutex after copying the
  service pointer. Shutdown can detach and close that service while scheduled
  note, reload, or setter callbacks continue using it.
- Service methods read `s.audio` without the lifecycle mutex while `Close`
  writes and clears it.
- Introduce an in-flight lease or lifecycle `RWMutex`, and make shutdown wait
  for active leases before closing native resources.
- **Done** in `4e7b3eb0`. Callers hold a lease for the callback; shutdown
  detaches first and waits for outstanding leases before closing anything
  native.

### 12. Web UI initialization waits forever for browser-local NATS

- [x] Make state-feed startup bounded and independent of control initialization
  in [`kit/webui/local_nats.js:4-32`](kit/webui/local_nats.js#L4-L32) and
  [`kit/webui/app.js:31-68`](kit/webui/app.js#L31-L68).
- The browser hardcodes `ws://127.0.0.1:9222` and retries forever before the UI
  registers most controls.
- A local NATS startup failure leaves the local UI inert. A browser opening
  `http://ENGINE-IP:3330` contacts its own loopback and can never connect.
- Use a bounded initial attempt, initialize controls independently, and provide
  an HTTP polling fallback or a same-origin authenticated WebSocket/SSE bridge.
- **Done** in `8e5189ec`. The feed gets 5s before the GUI comes up without it,
  and a late arrival seeds the UI rather than needing a reload.

### 13. Malformed UDP and port conflicts can terminate the FFGL host

- [x] Contain all OSC input exceptions inside the plugin boundary in
  [`ffgl/source/lib/nosuch/NosuchOscUdpInput.cpp:44-136`](ffgl/source/lib/nosuch/NosuchOscUdpInput.cpp#L44-L136).
- Oscpack packet construction can throw for malformed UDP, but construction and
  dispatch are outside any catch. The exception can escape the pthread entry
  function and terminate Resolume.
- A port conflict throws from `Listen`, bypassing the caller's error-return
  branch. Failure paths after `socket()` also leak the socket.
- Use RAII socket ownership, catch exceptions per datagram and at every FFGL ABI
  boundary, and add malformed-packet/fuzz coverage.
- **Done** in `ff886b89`. Per-datagram try/catch, the port conflict returns an
  error instead of throwing, and every failure path closes the socket.
  Compile-verified, not run inside Resolume.

### 14. Windows updates are not transactional

- [ ] Publish complete versions atomically with rollback in
  [`cmd/palette_installer/main_windows.go:136-159`](cmd/palette_installer/main_windows.go#L136-L159)
  and
  [`cmd/palette_installer/main_windows.go:311-428`](cmd/palette_installer/main_windows.go#L311-L428).
- Each live destination is deleted before its staged replacement is renamed.
  Later environment, shortcut, registry, and uninstaller steps have no rollback.
- An antivirus lock, disk error, or configuration failure can leave a mixture of
  old, new, and missing files.
- Stage and validate the complete version, retain the previous version, then
  perform an atomic swap and transactional configuration update.
- **Not closed.** Partly done in `1630fa9b`: publishing the payload is now
  all-or-nothing - the previous file is moved aside rather than deleted, and
  any failure restores every file already replaced. STILL OPEN: staging and
  swapping a complete version tree, and rolling back the environment, shortcut
  and registry steps in `configureInstall`.

### 15. Windows uninstall can report success after removal failures

- [x] Preserve recovery information and report removal errors in
  [`cmd/palette_installer/main_windows.go:462-500`](cmd/palette_installer/main_windows.go#L462-L500).
- Environment entries, shortcuts, and uninstall registration are removed before
  payload cleanup. Every payload `os.Remove` result is ignored.
- Locked or permission-denied files can remain after a success message while the
  uninstaller and repair entry remove themselves.
- Remove payload first, aggregate failures, and preserve the record and
  uninstaller whenever cleanup is incomplete.
- **Done** in `4e7b3eb0`. The payload goes first, failures are aggregated and
  reported, and the registration and install record survive a partial removal
  so it can be retried.

### 16. App and data uninstallers conflict over shared environment state

- [x] Define ownership for `PALETTE_DATAROOT` in
  [`cmd/palette_installer/main_windows.go:399-478`](cmd/palette_installer/main_windows.go#L399-L478).
- Both application and data installers set the same variable. Either uninstaller
  deletes it when its own recorded value matches, without checking whether the
  other installed component still needs it.
- Assign ownership to one component, or inspect/reference-count all Palette
  uninstall registrations and delete the shared value only with the last user.
- **Done** in `4e7b3eb0`. `PALETTE_DATAROOT` is deleted only when no other
  Palette uninstall registration remains; an unreadable registry counts as
  "still in use".

### 17. `addpalette` permits NATS config and subject injection

- [x] Validate identifiers and update NATS configuration transactionally in
  [`cmd/palette_hub/palette_hub.go:786-844`](cmd/palette_hub/palette_hub.go#L786-L844).
- The raw name is interpolated into quoted configuration and permission subjects
  without rejecting quotes, newlines, or NATS wildcards.
- The live configuration is overwritten before validation. Restore errors are
  ignored, and reload failure leaves the new configuration installed.
- Enforce a strict identifier grammar, read passwords from a protected input
  rather than argv, validate a temporary configuration, and atomically replace
  and roll back around reload.
- **Done** in `65dba98a`. Names must match `^[A-Za-z0-9_-]{1,64}$`, a candidate
  config is validated before it becomes live, and a failed reload restores the
  original. The password may also be read from stdin instead of argv.

### 18. Twitch credentials are included in connection-error logs

- [x] Remove the authentication token from errors in
  [`cmd/palette_chat/palette_chat.go:161-166`](cmd/palette_chat/palette_chat.go#L161-L166).
- An ordinary network or authentication failure formats the full token into the
  returned error, and `main` logs it.
- Log the username and upstream error only. Rotate any token that has already
  appeared in logs.
- **Done** in `65dba98a`. The error carries the username and the underlying
  error only. Any token that has already been logged should still be rotated.

### 19. macOS FFGL releases omit required code and use incompatible paths

- [x] Add `ShapeSprite.cpp` to
  [`build/macos/build_ffgl.sh:44-73`](build/macos/build_ffgl.sh#L44-L73).
- `Layer.cpp` calls `SpriteParametric::create`, whose definition is omitted from
  the macOS source list. The bundle can fail to link or load.
- **Done** in `1a9ac59a`. Verified by inspection: `Layer.cpp:246` calls
  `SpriteParametric::create`, defined in `ShapeSprite.cpp:162`. Not built on
  macOS.
- [ ] Replace backslash concatenation and shell-only `PALETTE` assumptions with
  `std::filesystem::path` and the macOS application-support directory in
  [`ffgl/source/lib/palette/PaletteHost.cpp:243-250`](ffgl/source/lib/palette/PaletteHost.cpp#L243-L250)
  and related asset loading.
- Add a post-build symbol/load smoke test launched without shell environment
  variables.
- **Not closed.** The backslash concatenation and shell-only `PALETTE`
  assumptions in `PaletteHost.cpp` are unchanged, and there is no post-build
  load smoke test.

### 20. Linux packaging omits runtime data and makes code runtime-writable

- [x] Package the required sanitized `data_default` tree in
  [`build/linux/build.sh:15-49`](build/linux/build.sh#L15-L49).
- The engine resolves its default data path under `/usr/local/palette`, but the
  release currently packages only binaries.
- **Done** in `1a9ac59a`. `build.sh` ships a sanitised `data_default`. Not run
  on Linux.
- [x] Keep program files root-owned in
  [`build/linux/install.sh:62-88`](build/linux/install.sh#L62-L88).
- The installer recursively gives the runtime account ownership of executable
  code linked from `/usr/local/bin`. Give that account ownership only of
  dedicated state, data, and log directories.
- **Done** in `1a9ac59a`. Program files stay root-owned; only the
  runtime-writable directories change hands. Not run on Linux.

### 21. Windows release scripts can package missing or stale artifacts

- [ ] Fail immediately on every build, move, and required copy in
  [`build/windows/build_bin.bat:61-193`](build/windows/build_bin.bat#L61-L193).
- Later successful commands can mask earlier failures, allowing packaging to
  proceed with a missing command or stale manually built executable.
- Build directly to explicit ship paths and verify a required-file manifest and
  hashes before packaging.
- **Not closed.** Partly done in `a18c2f52`: the five Go builds and their moves
  are checked through a `:build_go` subroutine, so a compile error stops the
  build instead of packaging a missing or stale binary with exit code 0. STILL
  OPEN: building directly to explicit ship paths, and verifying a required-file
  manifest with hashes before packaging.
- [x] Quote and validate every destructive target derived from
  `PALETTE_SOURCE` in `build_bin.bat`, `build_data.bat`, and `clean.bat`.
  Prefer one PowerShell implementation using `-LiteralPath`.
- **Done** in `1a9ac59a`. Every destructive target is quoted, and `clean.bat`
  and `build_data.bat` refuse to run unless `PALETTE_SOURCE` holds a `VERSION`
  file.

### 22. Hub day files can be accepted after partial writes

- [x] Make day-file creation and import atomic in
  [`cmd/palette_hub/palette_hub.go:463-512`](cmd/palette_hub/palette_hub.go#L463-L512)
  and
  [`cmd/palette_hub/palette_hub.go:743-752`](cmd/palette_hub/palette_hub.go#L743-L752).
- Dumping creates the final path before querying NATS, ignores write/close
  failures, and permanently skips any file that already exists.
- Import truncates an existing day and also ignores write errors.
- Write a same-directory temporary file, check marshal/write/sync/close, and
  rename only after full success.
- **Done** in `65dba98a`. Both dump and import go through `writeLinesAtomic`,
  which renames into place only after every write, the sync and the close
  succeed.

### 22A. FFGL logging recursively initializes when `PALETTE` is unset

- [x] Make logging initialization self-contained and safe before use from
  `DllMain` in
  [`ffgl/source/lib/nosuch/NosuchDebug.cpp:185-212`](ffgl/source/lib/nosuch/NosuchDebug.cpp#L185-L212).
- When `PALETTE` is missing, `RealNosuchDebugInit` calls `NosuchDebug` before
  setting `DebugInitialized` or creating `dMutex`. That immediately re-enters
  initialization and recurses until the plugin host overflows its stack.
- Initialize synchronization and a fallback sink before environment lookup, and
  never call the public logger from its own initializer. Minimize work performed
  under the Windows loader lock.
- **Done** in `ff886b89`. The mutex and fallback sink come up before the
  `PALETTE` lookup, so the diagnostic can no longer re-enter its own
  initialiser. Compile-verified, not run inside Resolume.

### 22B. Multiple Morph devices emit colliding cursor identities

- [x] Include a device serial or stable device index in cursor IDs emitted by
  [`cmd/gomorph/morph/morph.go:200-225`](cmd/gomorph/morph/morph.go#L200-L225)
  and match the source for cursor-up handling in FFGL.
- Every device uses its own contact IDs starting at the same values, while all
  devices share the same UDP sender. Down and drag partly distinguish source,
  but cursor-up ignores it.
- Simultaneous contacts with the same device-local ID can update or delete one
  another, causing jumps and stuck cursors.
- **Done** in `d365d3f5`. The device serial is part of the CID. The engine was
  unaffected - `kit/morph.go` already uses globally unique GIDs.

## P2 — Medium

### 23. Patch loop operations erase unrelated pending events

- [ ] Filter pending scheduler events by tag in
  [`kit/patch.go:591-656`](kit/patch.go#L591-L656).
- Fade, filter, and clear set the entire `pendingScheduled` slice to nil after
  finding one matching event.
- Acting on patch A can discard note-offs, sample stops, or cursor events for
  patches B, C, and D. Fade should mutate matching cursor events instead of
  deleting them.

### 24. Pro reload ignores changed analysis settings and directory contents

- [ ] Add a forced-reload path or a complete input fingerprint in
  [`pkg/samplesplitter/service.go:238-316`](pkg/samplesplitter/service.go#L238-L316).
- `LoadChannelSample` returns based only on directory, mode, loop, and rotate
  equality.
- Changes to word count, minimum duration, word threshold, or files inside the
  same directory therefore do not reload Pro channels.

### 25. Decoded PCM cache serves stale audio and grows without a bound

- [ ] Key or invalidate PCM by file identity and introduce a bounded LRU in
  [`pkg/samplesplitter/audio.go:808-835`](pkg/samplesplitter/audio.go#L808-L835).
- The cache is keyed only by pathname, unlike analysis caching, which includes
  size and modification time.
- Replacing a file can pair new cue boundaries with old PCM. Directory changes
  can accumulate large decoded and compressed buffers indefinitely.

### 26. BSS reload is destructive and can report success with no samples

- [ ] Build, validate, and preload candidate state before replacing live state
  in [`pkg/samplesplitter/service.go:202-220`](pkg/samplesplitter/service.go#L202-L220).
- Reload stops audio and clears the working cache before analysis.
- Empty, invalid, or entirely below-threshold directories can commit error-only
  state, preload zero paths, and still report audio healthy.
- Consolidate the duplicate HTTP reload implementation into the service method.

### 27. Audio-output changes destroy the working output before replacement

- [ ] Make output switching transactional in
  [`pkg/samplesplitter/audio.go:333-357`](pkg/samplesplitter/audio.go#L333-L357).
- The current device is uninitialized before the requested replacement is
  initialized and started.
- Failure leaves no active output while status can still identify the old
  device. Retain or reopen the previous device and update readiness/error state
  consistently.

### 28. `LastPlayback` is published and then mutated without locking

- [ ] Publish an immutable value snapshot in
  [`pkg/samplesplitter/service.go:401-428`](pkg/samplesplitter/service.go#L401-L428).
- `PlanNoteOn` stores the request pointer in `State.LastPlayback`, after which
  `NoteOnVoicePlanned` mutates `VoiceKey` outside the state lock.
- State snapshots return the same pointer while the web UI polls it.
- Set the voice key before publication or store and deep-copy values.

### 29. Samplesplitter configuration has unlocked runtime readers

- [ ] Replace direct exported `State.Config` access with a locked immutable
  snapshot in
  [`pkg/samplesplitter/server.go:67-112`](pkg/samplesplitter/server.go#L67-L112)
  and reload paths in `state.go`.
- Engine setters mutate configuration under `State.mu`, while HTTP handlers and
  reload routines read the fields without that lock.

### 30. One bad patch prevents later valid sample patches from syncing

- [ ] Aggregate per-channel failures in
  [`kit/samplesplitter.go:543-560`](kit/samplesplitter.go#L543-L560).
- Synchronization returns on the first invalid directory or load failure,
  leaving later enabled patches untouched.
- Attempt every channel, preserve previous state for failed channels, and
  bootstrap the service from any valid directory.

### 31. Runtime MIDI input changes never start the new listener

- [ ] Give `MidiIO` a synchronized listener lifecycle in
  [`kit/midi_windows.go:117-173`](kit/midi_windows.go#L117-L173) and the Unix
  equivalent.
- The live setter closes and replaces the port, but `ListenTo` is invoked only
  once before `Start` blocks forever.
- A device change disables MIDI input, and an engine started without a device
  cannot enable one later.

### 32. Failed OBS stops are recorded as successful

- [ ] Commit recording state only after OBS confirms the stop in
  [`kit/obs.go:540-597`](kit/obs.go#L540-L597).
- Manual and automatic stop paths clear `obsRecording` and the stop channel
  before `ObsCommand("recordstop")` succeeds.
- On failure, Palette reports not recording and future stop calls do nothing
  while OBS may continue recording.
- Add a stopping/error state and preserve a retry path.

### 33. Stepper snapshots retain mutable event pointers

- [ ] Deep-copy events into immutable status DTOs in
  [`kit/stepper.go:284-320`](kit/stepper.go#L284-L320).
- The snapshot copies slices but retains `*StepperEvent` pointers. JSON
  marshaling occurs after unlocking while note-off handling mutates duration.

### 34. `silenceAll` does not await loop-disable or clear requests

- [ ] Sequence all silence operations in
  [`kit/webui/app.js:1656-1663`](kit/webui/app.js#L1656-L1663).
- Eight promises are launched and ignored before all-notes-off and audio reset.
  Clear can run before looping is disabled or after reset, and rejected promises
  bypass the surrounding catch.
- Reuse the correctly ordered pattern in `stopNotesAndLoops`.

### 35. API panic recovery returns false success

- [ ] Return the recovered panic as an error in
  [`kit/engineapi.go:62-83`](kit/engineapi.go#L62-L83).
- The deferred recovery creates a local `err`, but the function has unnamed
  return values and therefore returns `"", nil` after recovery.
- HTTP callers receive a 200 success response, including for `global.debugnil`.

### 36. API request bodies have no size limit

- [ ] Apply `http.MaxBytesReader` and return HTTP 413 in
  [`kit/engine.go:359-379`](kit/engine.go#L359-L379).
- Read timeouts constrain duration, not allocation. A reachable peer can force
  large memory allocations through either POST endpoint.

### 37. `quad.loadrand` panics for an empty category

- [ ] Check for an empty preset list before modulo selection in
  [`kit/quad.go:362-373`](kit/quad.go#L362-L373).
- An empty or custom category performs modulo zero. Panic recovery then masks
  the failure as success unless the API recovery issue is fixed first.

### 38. Persistent current state still uses truncating writes

- [ ] Serialize per-path saves and use the atomic helper in
  [`kit/params.go:287-294`](kit/params.go#L287-L294),
  [`kit/quad.go:516-523`](kit/quad.go#L516-L523), and other preset/state writers.
- A crash or power loss can leave `_Current`, `_Boot`, quad state, or presets
  truncated. Concurrent requests can also write the same path.
- Improve the Windows branch of `WriteFileAtomic` to use an actual replacement
  primitive rather than remove-then-rename where practical.

### 39. Log archives can delete originals after failed ZIP finalization

- [ ] Propagate ZIP writer, destination close, and sync errors in
  [`kit/misc.go:580-625`](kit/misc.go#L580-L625).
- ZIP central-directory creation occurs during `Close`, but that error is
  deferred and ignored. `ArchiveLogs` can then clear the original logs after a
  disk-full or finalization failure.

### 40. Attract-mode fields have unsynchronized readers and writers

- [ ] Encapsulate all mutable attract state under one mutex or publish an
  immutable configuration snapshot in
  [`kit/attract.go:183-337`](kit/attract.go#L183-L337).
- API handlers update enabled state, timings, counts, and gesture settings while
  scheduler and cursor paths read or write them without matching synchronization.

### 41. Tempo accepts invalid values and races scheduler readers

- [ ] Validate and synchronize tempo state in
  [`kit/engineapi.go:448-455`](kit/engineapi.go#L448-L455) and
  [`kit/click.go:72-90`](kit/click.go#L72-L90).
- Zero, negative, `NaN`, and infinite factors are accepted. The raw factor is
  stored before derived timing is clamped, and quantization divides by it.
- Keep the factor and all derived values in one validated timing snapshot.

### 42. Historical hub requests use today's UTC offset

- [ ] Compute each day's boundaries in the intended IANA timezone in
  [`cmd/palette_hub/palettes_requestdays.sh:34-104`](cmd/palette_hub/palettes_requestdays.sh#L34-L104).
- Using the current offset for historical dates misattributes events across DST.
- Request half-open intervals from local midnight to the following midnight so
  subsecond events at the end of the day are not omitted.

### 43. Python API requests can hang or retry in a hot loop

- [ ] Use the configured retry session, bounded timeouts, backoff, cancellation,
  and a final error in
  [`python/palette.py:181-258`](python/palette.py#L181-L258).
- The configured session is never assigned or used. Requests use a 6000-second
  timeout and retry exceptions forever without sleeping.

### 44. Python helpers ignore `PALETTE_DATAROOT`

- [ ] Mirror the Go path-precedence rules in
  [`python/palette.py:93-120`](python/palette.py#L93-L120).
- Custom Windows `--data-root` installations currently make Python helpers read
  and write a different tree from the engine.

### 45. Parameter generation truncates headers before validating input

- [ ] Validate environment variables and the entire JSON model before opening
  outputs in
  [`python/generateparams.py:52-67`](python/generateparams.py#L52-L67) and
  [`python/generateparams.py:167-184`](python/generateparams.py#L167-L184).
- Malformed definitions can leave generated headers empty. An unset
  `PALETTE_DATA` is also treated as `None` rather than defaulting.
- Generate into temporary files and atomically replace only after success.

### 46. CLI lifecycle and argument handling have crash/hang cases

- [ ] Require the missing OSC argument before indexing it in
  [`cmd/palette/palette.go:229-240`](cmd/palette/palette.go#L229-L240).
- [ ] Handle `SIGINT` and `SIGTERM` through orderly cancellation in foreground
  CLI and engine commands.
- [ ] Exit nonzero after recovered fatal engine panics instead of reporting
  success.

### 47. Daily report locking can remain stale forever

- [ ] Replace the plain directory lock with `flock` or a validated PID lock in
  [`cmd/palette_hub/daily_update.sh:3-18`](cmd/palette_hub/daily_update.sh#L3-L18).
- A kill or reboot can leave the lock behind and silently disable every future
  report run. Add stale recovery and fail-fast shell behavior.

### 48. Go 1.25.12 has reachable standard-library vulnerabilities

- [ ] Upgrade the project and builders from Go 1.25.12 to at least Go 1.25.13 in
  [`go.mod:3`](go.mod#L3).
- `govulncheck ./...` found six reachable standard-library advisories, all fixed
  in 1.25.13:
  - `GO-2026-6218`
  - `GO-2026-6090`
  - `GO-2026-6089`
  - `GO-2026-6088`
  - `GO-2026-5972`
  - `GO-2026-5026`

## P3 — Low

### 49. Failed MIDI-port transitions retain stale status

- [ ] Make replacement transactional or explicitly clear current-port state in
  [`pkg/samplesplitter/midi.go:55-87`](pkg/samplesplitter/midi.go#L55-L87).
- If opening or listening on a replacement fails, the old port has already been
  stopped but remains reported as active.

### 50. Live `attractallowgui` changes render with the previous value

- [ ] Update state-derived fields before rendering in
  [`kit/webui/app.js:1686-1699`](kit/webui/app.js#L1686-L1699).
- A false-to-true or true-to-false change remains visually wrong until another
  status event arrives.

### 51. Stale failed web requests can replace a newer grid

- [ ] Check `gridLoadToken` in error and finalization paths as well as success
  paths in `kit/webui/app.js`.
- An older request that fails after a newer screen renders can replace the
  current grid with its stale error state.

### 52. CI does not exercise the main build or runtime surfaces

- [ ] Add a cross-platform CI workflow covering:
  - `go build ./cmd/...`
  - `go test -race ./...`
  - `go vet ./...`
  - `staticcheck ./...`
  - `govulncheck ./...`
  - JavaScript syntax checks
  - Python parsing/tests
  - FFGL tests and native build/load smoke tests where toolchains are available
  - Installed-layout smoke tests for Windows, macOS, and Linux packages
- The current workflow runs pre-commit and secret scanning only.
- Current measured coverage is approximately 23.9% for `kit`, 41.7% for
  `pkg/samplesplitter`, 19.9% for the installer, and effectively 0% for command
  packages. Prioritize concurrency and lifecycle scenarios not exercised by the
  current passing race tests.

## Review validation baseline

The following checks passed during the review:

- `go build ./cmd/...`
- `go vet ./...`
- `staticcheck ./...`
- Race-enabled and shuffled Go tests
- JavaScript syntax checks for 11 web UI files
- Python parsing for 20 project scripts/modules
- All 32 FFGL test checks

`govulncheck ./...` completed with the six reachable Go 1.25.12 advisories
listed above. The worktree was clean before this backlog file was added, and the
engine executable was not run during review.
