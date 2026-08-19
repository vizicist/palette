# Palette Review Backlog

This backlog captures what is left of the repository-wide review performed on
2026-08-15. Items are removed as they are dealt with, so everything below is
still outstanding; an item that was only partly addressed says what was done and
what remains.

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

Of the original 54 items (1 to 52, plus 22A and 22B), 32 have been dealt with
and removed. What is left is 22: four P1 remnants, 15 in P2 - one of which, 38,
is also partly addressed - and three in P3.

Worth knowing about the work already done:

- The FFGL C++ fixes were compile-verified with a full MSBuild rebuild, and the
  macOS and Linux packaging fixes are syntax-checked only. Nothing in either has
  been run on its target platform.
- CI now builds and runs `go test -race` on Linux, Windows and macOS, with vet,
  staticcheck, govulncheck, a web UI syntax pass and a Python byte-compile, so
  regressions in what has been fixed should surface on their own from here.

## Recommended next batch

1. Finish the four P1 remnants: 7 (authentication), 14 (version swap), 19
   (macOS paths), 21 (release manifest).
2. The sample-service reload and state cluster - 24, 26, 28, 29, 30 - which is
   the largest untouched group and sits entirely on the audio path.
3. Run the macOS and Linux release builds, the only way to actually exercise
   the packaging fixes.

## P1 — High

### 7. LAN HTTP control is restricted but not authenticated

- The server now defaults to loopback and both POST endpoints cap the body at
  1MB (`efd39350`, `65dba98a`), so it is no longer reachable from the LAN out of
  the box.
- [ ] Add an authenticated session or token, with authorization and
  Origin/CSRF checks, for installations that set `PALETTE_HTTP_BIND` to reach
  the GUI from elsewhere on the venue network. Such an installation is still
  serving an unauthenticated API that can change boot state, restart processes,
  delete recordings, upload content, or stop the engine.

### 14. Windows updates are not transactional

- [ ] Publish complete versions atomically with rollback in
  [`cmd/palette_installer/main_windows.go:136-159`](cmd/palette_installer/main_windows.go#L136-L159)
  and
  [`cmd/palette_installer/main_windows.go:311-428`](cmd/palette_installer/main_windows.go#L311-L428).
- Each live destination is deleted before its staged replacement is renamed.
  Later environment, shortcut, registry, and uninstaller steps have no rollback.
- An antivirus lock, disk error, or configuration failure can leave a mixture of
  old, new, and missing files.
- Publishing the payload is already all-or-nothing (`1630fa9b`): the previous
  file is moved aside rather than deleted, and any failure restores every file
  already replaced.
- Still to do: stage and validate a complete version tree and swap it in one
  step, and make the environment, shortcut and registry steps in
  `configureInstall` roll back with it.

### 19. macOS FFGL uses Windows-shaped paths

- `ShapeSprite.cpp` is now in the macOS source list (`1a9ac59a`), so the bundle
  has `SpriteParametric::create`. That was verified by inspection only - nothing
  here has been built on macOS.
- [ ] Replace backslash concatenation and shell-only `PALETTE` assumptions with
  `std::filesystem::path` and the macOS application-support directory in
  [`ffgl/source/lib/palette/PaletteHost.cpp:243-250`](ffgl/source/lib/palette/PaletteHost.cpp#L243-L250)
  and related asset loading.
- Add a post-build symbol/load smoke test launched without shell environment
  variables.


### 21. Windows packaging does not verify what it shipped

- The five Go builds and their moves now fail the build through a `:build_go`
  subroutine (`a18c2f52`), and every destructive target is quoted, with
  `clean.bat` and `build_data.bat` refusing to run unless `PALETTE_SOURCE`
  holds a `VERSION` file (`1a9ac59a`).
- [ ] Build directly to explicit ship paths, and verify a required-file
  manifest with hashes before packaging, so a release cannot be assembled from
  whatever happens to be lying in the ship tree.

## P2 — Medium

### 24. Pro reload ignores changed analysis settings and directory contents

- [ ] Add a forced-reload path or a complete input fingerprint in
  [`pkg/samplesplitter/service.go:238-316`](pkg/samplesplitter/service.go#L238-L316).
- `LoadChannelSample` returns based only on directory, mode, loop, and rotate
  equality.
- Changes to word count, minimum duration, word threshold, or files inside the
  same directory therefore do not reload Pro channels.

### 26. BSS reload is destructive and can report success with no samples

- [ ] Build, validate, and preload candidate state before replacing live state
  in [`pkg/samplesplitter/service.go:202-220`](pkg/samplesplitter/service.go#L202-L220).
- Reload stops audio and clears the working cache before analysis.
- Empty, invalid, or entirely below-threshold directories can commit error-only
  state, preload zero paths, and still report audio healthy.
- Consolidate the duplicate HTTP reload implementation into the service method.

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

### 33. Stepper snapshots retain mutable event pointers

- [ ] Deep-copy events into immutable status DTOs in
  [`kit/stepper.go:284-320`](kit/stepper.go#L284-L320).
- The snapshot copies slices but retains `*StepperEvent` pointers. JSON
  marshaling occurs after unlocking while note-off handling mutates duration.

### 38. Preset writes are atomic but not serialized

- `_Current`, `_Boot`, quad state and theme links all go through
  `WriteFileAtomic` now (`3761ef1f`), so a crash cannot leave one truncated.
- [ ] Serialize saves that target the same path: two concurrent requests can
  still race each other's writes, and the last one wins arbitrarily.
- [ ] Use a real replacement primitive on Windows instead of
  remove-then-rename, which leaves a brief window where the file is absent.

### 39. Log archives can delete originals after failed ZIP finalization

- [ ] Propagate ZIP writer, destination close, and sync errors in
  [`kit/misc.go:580-625`](kit/misc.go#L580-L625).
- ZIP central-directory creation occurs during `Close`, but that error is
  deferred and ignored. `ArchiveLogs` can then clear the original logs after a
  disk-full or finalization failure.

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
