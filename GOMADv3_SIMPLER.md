# GOMAD v3 Simplification Review

## Executive Summary

GOMAD v3 can be simplified, but primarily by making its existing deep-module
boundaries more honest. The runtime patch, World model, process containment,
canonical records, and crash-safe artifact validation are not accidental
complexity; removing them would spread security, determinism, or failure-mode
logic into callers. The highest-leverage simplifications are instead:

1. represent an I/O profile once as immutable data and one implementation seam,
   rather than repeating the only supported profile in resolution, validation,
   build-overlay, bootstrap, Runner, and replay code;
2. make `internal/process` own a typed launch-resource/descriptor plan, rather
   than extending parallel field lists, FD enums, `ExtraFiles`, and close lists
   for every capability;
3. move batch/partial publication mechanics out of the Runner scheduling loop
   into the artifact subsystem; and
4. make World transport an explicit execution capability, while keeping World
   itself intact.

This review used the working tree on 2026-08-11. The committed architecture
already includes the Runner, supervisor/bootstrap, World, Record/Artifact/Replay,
and one exact I/O profile. During the review, concurrent uncommitted work added a
second profile, the read-only mount path (`internal/romount` plus Runner/process
and standard-library hooks), and a consolidated `tools/gomadv3/ARCHITECTURE.md`.
Those conclusions are therefore directional and explicitly identified below;
they should be rechecked after the in-progress changes reach a coherent commit.

| Rank | Recommendation | Impact | Confidence | Effort | Class |
|---|---|---:|---:|---:|---|
| 1 | D1: immutable profile specification and shared implementation seam | High | High | Medium | Near-term |
| 2 | D2: process-owned launch-resource/descriptor plan | High | High | Medium-high | After current mount work stabilizes |
| 3 | D3: artifact-owned batch journal | Medium-high | High | Medium | Near-term |
| 4 | C1: centralize record-facing semantic conversions | Medium | High | Low-medium | Near-term |
| 5 | C3: split the shell suite into explicit tiers | Medium | High | Medium | Near-term |
| 6 | D4: explicitly enable World transport only for World-aware targets | Medium | Medium | Medium | Speculative/migration required |

The deletion test supports the existing top-level modules: deleting Runner,
World, Artifact, Record, Replay, target preparation, or process supervision
would scatter their policies. It does not support the current shallow I/O
`Profile`, duplicated launch plumbing, or Runner-owned persistence primitives:
deleting those abstractions would lose little encapsulation because their
details are already repeated in callers.

## Design-Level Simplifications

### D1. Turn I/O profiles into immutable specifications, not repeated switches

**Recommendation.** Add a small registry of immutable `ProfileSpec` values. A
spec should contain the exact public name, target contract, inventory,
implementation family/version, platform/toolchain contract, and build-overlay
policy. Keep one concrete shared implementation for the current deterministic
network/filesystem/entropy/SQLite/transcript behavior. Do not introduce a Go
interface until a second implementation family exists.

The in-progress second profile has reduced some one-profile checks to
`profileArgument` switches, but profile policy remains repeated at several seams:

- `internal/ioprofile/profile.go:13-19` defines profile/version/selectors,
  `Resolve` at `:45-73` constructs inventory, `ValidatePreparedTarget` at
  `:75-106` repeats its target, argv, tags,
  toolchain, platform, environment, and identity rules.
- `internal/ioprofile/bootstrap.go:30-63` checks the same profile before encoding,
  while `DecodeBootstrapFrame`/`profileForIdentity` at `:66-99` iterate the
  supported names again.
- `internal/ioprofile/sqlite_overlay.go:32-99` branches on the same name before
  applying the implementation's pinned SQLite overlay.
- `internal/runner/runner.go:254-293`, `:809-829`, and `:937-975` separately
  resolve, prepare, validate, bootstrap, and record the profile.
- `internal/replay/replay.go:128-160` and `:202-209` resolve and compare its
  identity again.

The working tree already adds batch-security as a second exact identity backed
by the same implementation (`internal/ioprofile/profile.go:14-18`), and a dated
design proposes batch-terminate on the same basis
(`tools/gomadv3/docs/2026-08-11-activity-batch-terminate-profile-design.md:1-15`).
Continuing the switch pattern would multiply stringly policy. With a registry,
Runner and Replay resolve once and then consume an opaque, validated spec; the
shared implementation receives that spec when preparing the build overlay and
bootstrap frame.

**Prerequisite.** Freeze which fields distinguish public profile identity from
shared implementation identity, including the still-evolving read-only-mount
identity. The uncommitted second-profile diff changes `implementationVersion`
and its hash input (`internal/ioprofile/profile.go:16`, `:68-72`), so it changes
the existing batch-cancel implementation digest despite the compatibility goal.
Resolve that explicitly and preserve existing bytes/digests unless a schema
migration is intended.

**Behavioral risk.** A seemingly harmless reordering of inventory fields or
change to implementation-version hashing breaks replay compatibility. Pin the
old canonical inventory, bootstrap frame, and manifest identities as golden
bytes before refactoring.

### D2. Let `internal/process` own a typed launch-resource plan

**Recommendation.** Keep the Runner -> supervisor -> bootstrap -> target stages,
but introduce an internal `launchResources` (or equivalent) which owns:

- capability presence (`World`, I/O transcript, expected transcript, read-only
  mount broker);
- stable target FD numbers and each stage's inherited FD order;
- pipe/backing creation;
- per-process `ExtraFiles` construction; and
- the close/dup ownership plan for success and every partial failure.

Expose nested capability values in `process.Request`, rather than the current
flat World/I/O/mount field run. This should be a concrete internal table, not a
plugin interface.

The caller trace is `runner.runSeed` (`internal/runner/runner.go:765-831`) and
`replay.Replay` (`internal/replay/replay.go:152-160`) -> `process.Run` ->
`SupervisorMain` -> `BootstrapMain` -> target. At the public process seam,
`internal/process/process.go:23-46` currently mixes command, deadlines, capture,
World, transcript, replay, mounts, and writers in one request. Inside the Unix
implementation:

- two independent FD enums live at `internal/process/process_unix.go:24-51` and
  a third target enum at `internal/process/bootstrap_unix.go:17-26`;
- the same capability is represented again in `supervisorRequest` and
  `targetBootstrapRequest` at `process_unix.go:59-87`;
- `process.Run` builds ordered `ExtraFiles` and matching close lists at
  `process_unix.go:199-215`;
- `SupervisorMain` reconstructs optional descriptors and builds the next
  `ExtraFiles`/close lists at `process_unix.go:564-603` and `:618-710`; and
- `BootstrapMain` manually duplicates and closes final descriptors at
  `bootstrap_unix.go:28-103`.

The in-progress mount feature demonstrates the maintenance cost: one capability
added two FDs to all three stages, booleans to two wire structs, two
`ExtraFiles` append sites, and several close/error paths. A single descriptor
plan improves locality without weakening the boundaries.

**Prerequisite.** Land or pause the mount work so there is one stable descriptor
layout to characterize. Record the exact existing FD numbers and inheritance
sets in tests.

**Behavioral risk.** FD reordering, a leaked write end, or closing the wrong end
can turn clean EOF into a hang or expose a host resource to the target. Preserve
stable target FDs, explicit error joins, close-on-exec behavior, and bounded
cleanup; test injected failure at each allocation/start/dup phase.

### D3. Give Artifact an explicit batch-journal abstraction

**Recommendation.** Keep scheduling, failure policy, classification, and
manifest construction in Runner. Move filesystem state transitions for a batch
into an artifact-owned `BatchJournal`: private directory creation, preparation
and per-seed partial state, `runs.jsonl` creation/append/sync, batch hash and
final `batch.json` publication, and partial cleanup. Runner should express
semantic transitions; the journal should enforce the durable layout.

`runner.runLocal` spans `internal/runner/runner.go:185-664`. Within it, storage
setup is at `:207-250`, `runs.jsonl` lifecycle at `:302-320` and `:621-630`,
per-run partial transitions at `:765-853`, and final batch publication at
`:634-663`. Runner also implements `atomicWriteContext`, directory fsync, and
private directory creation at `:1086-1189`. `artifact.Store.Publish` already
owns private staging, payload hashing, fsync, collision handling, and atomic
no-replace publication (`internal/artifact/store.go:40-188`), including its own
directory-sync helpers at `:314-330`. The committed design baseline even
describes `artifact/store.go` as owning the partial-run lifecycle
(`git show HEAD:GOMADv3_RUNNER.md`, lines 316-330), but the implementation
leaves that lifecycle in Runner.

This is a depth problem, not merely a long-function problem: extracting arbitrary
Runner helper functions would keep policy and mechanics interleaved. An
artifact-owned journal gives crash safety one implementation owner and leaves
`runLocal` visibly about orchestration.

**Prerequisite.** Characterize the exact on-disk states expected after preparation
failure, cancellation, overall timeout, target-supervision failure, and success.
The existing assertions in `internal/runner/runner_test.go:127-134`, `:355-403`,
and `internal/runner/runner_mode_unix_test.go:25-27` are the starting contract.

**Behavioral risk.** Changing fsync order or partial retention can reduce crash
diagnostics even if normal tests pass. Move mechanics without changing names,
modes, atomicity, or timing of durable transitions.

### D4. Make World transport an explicit execution capability

**Recommendation.** Keep the World module and the canonical `none` World record,
but make the child transport opt-in in the prepared execution plan. A target or
profile which declares World gets the current config/record descriptors and
replay input. A target without World gets `record.NoneWorld()` without allocating,
inheriting, or draining World pipes.

World is explicitly connected by target code through `world/child.Open`
(`tools/gomadv3/README.md:149-170`). A repository-wide Go-source search found
production code for the session but no production caller in this tree; the only
calls are process integration tests at `internal/process/process_test.go:424`
and `:448`. Nevertheless every `process.Request` must provide positive World
limits (`internal/process/process.go:98-100`), `process.Run` always allocates and
drains a World pipe (`internal/process/process_unix.go:120-123`, `:186-191`,
`:273-287`), `SupervisorMain` always creates and transmits World configuration
(`:605-647`), and bootstrap always installs its descriptors
(`internal/process/bootstrap_unix.go:82-90`). Runner later converts an empty
record to `NoneWorld` (`internal/runner/runner.go:462-489`, `:1004-1007`).

This change clarifies two valid architectures: transparent standard-library I/O
profiles, and explicitly modeled World sessions. It should not force current
transparent network/filesystem shims through World; doing that before a concrete
quiescence need would merge distinct modules and add machinery.

**Prerequisite.** Audit downstream targets outside this repository for implicit
reliance on the reserved descriptors, add an explicit Runner/profile declaration,
and migrate them before making absence the default.

**Behavioral risk.** An existing target may call `world/child.Open` without a new
declaration. Because that becomes a compatibility change, this recommendation is
speculative and should follow D1/D2, not block them.

## Code-Level Simplifications

### C1. Centralize record-facing semantic conversions

Runner and Replay independently encode facts which must agree exactly:

- `worldFailureReason` is duplicated at `internal/runner/runner.go:922-934` and
  `internal/replay/replay.go:402-414`;
- target stderr classification is implemented in `runner.classify` at
  `runner.go:871-919` and again in `replay.actualReason` at
  `replay.go:373-400`;
- `[32]byte` -> `record.SHA256` conversion is duplicated at `runner.go:1000-1002`
  and `replay.go:424-426`; and
- build-info projection is duplicated at `internal/target/target.go:511-522`
  and `internal/replay/replay.go:428-438`, even though Replay already imports
  `target`.

Export the pure build-info projection from `target`, add a digest conversion to
`record`, and give execution-outcome classification one internal owner used by
both recording and replay. Do not put process/World classification into
`record`; Record should remain schema-focused. This is a small reuse change with
medium impact because drift here creates false replay divergence or false match.

### C2. Consolidate secure regular-file opening at a meaningful depth

`internal/artifact/open_nofollow_{unix,other}.go` and
`internal/target/open_nofollow_{unix,other}.go` are byte-for-byte identical except
for package name; the same is true of `linkcount_{unix,other}.go`. Their callers
are artifact verification (`internal/artifact/open.go:280-348`) and target
hash/copy/bounded-read paths (`internal/target/target.go:524-611`).

Move these to a small `internal/safefile` module only if it owns the complete
invariant—open without following a final symlink, validate a regular file and
link count, and re-stat the opened handle as appropriate. Merely exporting two
one-line wrappers would create a shallower package than the duplication it
removes. Preserve the platform-specific fail-closed behavior.

### C3. Split `test.sh` into reproducible tiers while retaining one gate

`tools/gomadv3/test.sh` is 1,158 lines and has only one early mode,
`validate` (`:166-170`). It then combines toolchain/cache/lock failure tests,
stock compatibility, runtime/clock/map stress, root Make integration, and the
upstream runtime suite (`:1152-1156`). `tools/gomadv3/Makefile:19-26` exposes
`validate` and the full shell run, but not focused shell tiers. The repository's
committed testing-gap document already specifies the useful seam: `validate`,
`test-builder`, `test-runtime`, `test-upstream`, and an all-tier `test`
(`git show HEAD:GOMADv3_TESTS.md`, lines 278-292).

Implement those modes in the existing driver/shared helpers first; separate
files only when a tier has an independent setup/cleanup contract. The full test
must continue to execute all tiers in the current order. This improves failure
locality and developer iteration without weakening coverage or adding another
test framework.

### C4. Correct the altitude of the evolving mount client

This finding applies only to the uncommitted mount snapshot. The design plan
places the client/cache in `overlay/src/internal/gomadio` and leaves `os` as the
standard-library adapter
(`tools/gomadv3/docs/2026-08-12-lazy-read-only-mount-plan.md:98-108`). The current
snapshot instead puts the wire constants, status/kind types, ordinal state,
framing, descriptor I/O, entry model, open-handle model, and `os.File`
translation together in `overlay/src/os/gomad.go:32-87` and `:441-547`.

After the feature is behaviorally complete, move framing, ordinal validation,
bounded response decoding, immutable entries, and handle state behind a narrow
`internal/gomadio` client. Keep only path/error/`FileInfo`/`DirEntry` translation
in `os`. Host and overlay code cannot share a Go package, so the protocol will
still have two definitions (`internal/romount/wire.go:11-30` and the overlay);
pin them with golden frame tests rather than adding a generator prematurely.

Two safe local cleanups are visible in the same snapshot:

- `gomadFileReaddir` allocates names, `DirEntry` values, and `FileInfo` values for
  every call, then returns only one collection (`overlay/src/os/gomad.go:344-387`);
  allocate only the requested representation.
- `romount.StatusError` is declared but is not emitted or handled
  (`internal/romount/wire.go:23-30`); remove it until the wire contract defines
  its payload and target behavior.

Do not perform these cleanups while the mount protocol is still being rearranged;
avoid generating merge noise in security-sensitive work.

### Simplify-code dimension screen

| Dimension | Result |
|---|---|
| Unnecessary machinery | No broad machinery deletion justified. `Preparer`/`Executor` injection is used for isolated tests, and coordinator/supervisor stages are load-bearing. D4 can remove unused World transport per run. |
| Reuse | C1 and C2 are real two-consumer reuse opportunities; D1 prevents the second profile from copying policy. |
| Duplication | Outcome/build-info/digest conversion, safe-file helpers, and Runner/artifact durability helpers are the substantive clusters. |
| Parameter/state sprawl | D2 is the primary fix; adding more flat request/config fields would worsen it. |
| Leaky/stringly abstractions | The profile name/selector/version repetition is the clearest leak; FD ordering is a second implementation leak. |
| Efficiency | Avoid unconditional World pipes (D4) and three-way `Readdir` allocation (C4). No evidence supports optimizing World queues or canonical validation. |
| Clarity/standards | Tier the shell suite and keep Runner at orchestration altitude. Prefer concrete specs/tables over speculative interfaces. |
| Dead code/comments | Only the provisional `StatusError` is clear dead surface. Preserve historical rationale until current contracts are extracted. |
| Wrong-altitude fixes | Do not solve Runner length with arbitrary helper extraction, put wire mechanics in `os`, or put outcome semantics in Record. D2-D3/C4 identify the owning modules. |

One apparent cleanup is specifically rejected: `overlay/src/internal/gomadtrace/trace.go:171-230`
contains a hand-written SHA-256. Importing `crypto/sha256` there is not a safe
line-count reduction: the standard SHA implementation's dependency graph reaches
`os`, while overlaid `os` imports `internal/gomadtrace`, creating an import cycle.
Keep the implementation unless a patched-toolchain dependency check proves a
cycle-free standard primitive is available; retain vector tests either way.

## What Should Stay Complex

- **Runtime patch and toolchain build/validation.** `build.sh`, `test.sh`, and the
  patch/overlay validators pin source checksums, reject scope expansion and
  collisions, apply zero-fuzz snapshots, normalize build inputs, serialize same-key
  builders, and publish atomically (`tools/gomadv3/README.md:91-99`). Fewer checks
  would make the binary's claimed identity less trustworthy. Keep runtime changes
  small, but do not replace NUL-safe validation or immutable build caching with a
  shorter best-effort script.
- **Coordinator, supervisor, and bootstrap stages.** The coordinator owns the
  overall process-tree watchdog (`internal/runner/coordinator.go:47-125`); the
  supervisor owns target group termination/reaping and final reports
  (`internal/process/process_unix.go:564-760`); bootstrap installs fixed FDs before
  `exec` (`internal/process/bootstrap_unix.go:28-110`). Collapsing these processes
  would weaken containment and crash cleanup. Simplify their descriptor plan,
  not their responsibilities.
- **World core.** World is already a deep, concrete, independently tested module
  with stable identities, deterministic ordering, bounds, snapshots, and replay.
  Its deletion test is documented in the committed baseline
  (`git show HEAD:GOMADv3_WORLD.md`, lines 75-87); removing it would force
  adapters to invent queues, identities, capacity behavior, and replay. D4
  concerns optional transport only.
- **Record, Artifact, and Replay validation.** Canonical JSON, strict decoding,
  complete hashes, no-follow/link-count checks, private staging, fsync, atomic
  no-replace, and pre-execution replay checks protect canonical identity and
  crash safety (`internal/artifact/store.go:40-188`,
  `internal/replay/replay.go:188-230`). Consolidate ownership, but do not weaken
  fail-closed checks or accept partial records.
- **Read-only mount capture.** If the feature lands, pinned roots, symlink/special
  file rejection, before/after identity checks, sorted complete directories, and
  all allocation/request bounds are load-bearing
  (`internal/romount/capture.go:75-99`, `:101-128`, `:159-266`). The broker is a
  valid deep module. Simplification belongs in its caller plumbing and overlay
  adapter, not in containment.
- **Exact profile selectors and identities.** A registry should centralize exact
  selectors, toolchain/platform requirements, inventories, and bootstrap hashes;
  it must not generalize them into permissive user input.

## Recommended Sequence

1. **Freeze the current contract.** Land or isolate the mount work. Add golden
   bytes for the existing profile inventory/bootstrap/manifest and characterize
   batch partial states and descriptor layouts.
2. **Apply safe locality fixes.** Centralize build-info/digest/outcome semantics
   (C1), then consolidate the complete safe-file invariant if the resulting
   module is deep enough (C2). No record format changes.
3. **Create the profile registry (D1).** Consolidate the in-progress
   batch-cancel/batch-security specs, preserve the batch-cancel identity, then
   add further exact profiles against the same implementation without new
   switches.
4. **Create the batch journal (D3).** Move durability mechanics one state
   transition at a time, leaving Runner policy unchanged.
5. **Refactor launch resources (D2).** Use the stable post-mount FD layout as the
   characterization baseline. Convert one capability at a time without changing
   numeric target descriptors.
6. **Split test tiers (C3).** Keep the existing full gate and exact test order;
   expose narrower reproduction commands.
7. **Revisit the mount adapter (C4).** Only after record and replay support is
   complete, move client mechanics to `internal/gomadio` and make allocation
   cleanups.
8. **Decide explicit World enablement (D4).** Audit external callers and migrate
   deliberately. Do not bundle this compatibility change with descriptor
   refactoring.

## Verification Strategy

No long black-box tests or generated-toolchain mutations were run for this
analysis. Implementation should be verified in increasing cost/risk order:

1. **Golden compatibility:** compare canonical profile inventory, implementation
   digest, bootstrap frame, manifest, failure signature, `runs.jsonl`, and
   `batch.json` bytes before and after D1/C1/D3.
2. **Focused Go tests:** run affected package tests with `-tags test_dep` for
   `ioprofile`, `target`, `record`, `artifact`, `runner`, `process`, `replay`,
   `worldrecord`, and (once stable) `romount`.
3. **Capability matrix:** exercise process launch with baseline only, World only,
   I/O record, I/O replay, and I/O plus mounts. Assert inherited FDs are present
   only when declared; inject failure at pipe/backing creation, both process
   starts, request writes, dup/install, target exit, and cleanup.
4. **Crash/durability matrix:** interrupt preparation, run append, failure
   publication, batch write, rename, and fsync. Require either the previous
   complete state or the documented partial state, never a valid-looking partial
   artifact.
5. **Replay equivalence:** record and replay the existing exact batch-cancel
   profile; verify the same first-divergence field for mutated outcome, streams,
   I/O transcript, and World data.
6. **Mount-specific verification:** use golden host/overlay wire frames; enforce
   bounds before allocation; mutate/delete host inputs after record and prove
   replay never reopens them. Re-run this only after the mount implementation is
   coherent.
7. **Tier parity and full gate:** prove the union of new shell tiers executes the
   same cases in the same order as the former monolith, then run
   `make -C tools/gomadv3 test` and relevant repository formatting/lint checks.

At every step retain checks for canonical identity, bounded memory/output,
descriptor closure, process-group disappearance, complete transcripts, strict
replay preflight, file modes, no symlink/hard-link acceptance, fsync ordering,
and stable error classification.

## Documentation Observations

- `tools/gomadv3/README.md` remains the appropriate command/current-behavior
  contract. Describe mounts there only when artifact persistence and
  host-independent replay are implemented, not when the first broker path works.
- The committed baseline's `GOMADv3_NEXT.md:1-43` and
  `GOMADv3_TESTS.md:21-23` had drifted: they described now-implemented Runner,
  World, external adapters, and exploration as future or excluded. Concurrent
  uncommitted work deletes those files and adds
  `tools/gomadv3/ARCHITECTURE.md:1-40`, explicitly making README authoritative
  and recording the four ownership boundaries. That consolidation directly
  addresses the noise found by this review; keep it.
- Before deleting the baseline `GOMADv3_CLOCK.md`, `GOMADv3_RUNNER.md`, and
  `GOMADv3_WORLD.md`, ensure their non-obvious rationale, rejected alternatives,
  failure analysis, and deletion tests survive in the architecture document or
  versioned history. The new architecture already preserves the most important
  runtime/clock, containment, record, and World decisions
  (`tools/gomadv3/ARCHITECTURE.md:42-175`).
- Keep dated files under `tools/gomadv3/docs/` explicitly labeled as proposals or
  implementation plans. Do not let the in-progress mount design read as a shipped
  contract, and keep observed qualification results separate from proposals.

## Appendix: Evidence Map

| Area | Interface / caller trace | Principal evidence | Simplification verdict |
|---|---|---|---|
| CLI/Runner/coordinator | `cmd/gomad.runExplore` -> `runner.Run` -> `runIsolated`/`runLocal` | `cmd/gomad/main.go:114-165`; `internal/runner/runner.go:175-185`; `internal/runner/coordinator.go:47-166` | Keep outer isolation; simplify profile and persistence ownership. |
| Target preparation | `runner.runLocal` -> `target.Prepare` -> prepared target verification | `internal/runner/runner.go:241-299`; `internal/target/target.go:93-181` | Deep enough; centralize profile overlay policy and build-info conversion. |
| I/O profile | Resolve -> overlay -> target validate -> bootstrap -> record/replay | `internal/ioprofile/profile.go:43-101`; `bootstrap.go:30-91`; `sqlite_overlay.go:32-99`; `runner.go:254-293`; `replay.go:128-160` | Highest-priority shallow/stringly seam. |
| Process supervision | Runner/Replay -> `process.Run` -> supervisor -> bootstrap -> target | `internal/process/process.go:23-127`; `process_unix.go:102-380`, `:564-760`; `bootstrap_unix.go:28-160` | Keep stages and error handling; consolidate resource/FD plan. |
| World | target `child.Open` -> World -> bounded recording -> worldrecord | `world/child/child.go`; `world/world.go`; `world/recording.go`; `internal/worldrecord/worldrecord.go:17-205` | Keep deep module; make transport explicit/optional. |
| Record/Artifact/Replay | Runner manifest -> `Store.Publish`; `artifact.Open` -> replay preflight -> process | `internal/runner/runner.go:526-663`; `internal/artifact/store.go:40-188`; `internal/artifact/open.go:280-348`; `internal/replay/replay.go:66-230` | Keep strictness; Artifact should also own batch durability. |
| Runtime/overlay/toolchain | validated patch + overlay -> immutable custom GOROOT | `tools/gomadv3/build.sh`; `tools/gomadv3/test.sh:1-170`; `tools/gomadv3/README.md:91-99` | Complexity is load-bearing; split tests, do not weaken validation. |
| Read-only mount (uncommitted) | CLI -> Runner mapping -> process broker -> overlay `os` client | `internal/romount/config.go:16-58`; `capture.go:75-300`; `wire.go:38-183`; `overlay/src/os/gomad.go:32-87`, `:441-547` | Broker is deep; launch plumbing and overlay client locality need follow-up after stabilization. |
| Tests/docs | Make targets -> Go suites + monolithic shell gate | `tools/gomadv3/Makefile:1-26`; `test.sh:166-170`, `:1152-1158`; committed `GOMADv3_TESTS.md:278-292` | Expose focused tiers; reconcile historical plans with current contract. |
