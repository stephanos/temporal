---
status: done
---

# Plan: Implement the Gomad v3 deterministic Go runtime toolchain

## Context

Implement `GOMAD_ALT.md` as a side-by-side, Unix-only Go 1.26.4 toolchain under `tools/gomadv3`. The toolchain must preserve upstream behavior when `GOMADSEED` is absent and make supported runtime-controlled scheduling, `select`, map, channel, and synchronization choices repeatable when the seed is present.

The runtime source customization remains intentionally small, but has no numeric file or line limit. Automated checks prohibit compiler, `cmd/go`, public-package, map-implementation, channel, GC, platform, assembly, and generated-output changes. Any expansion beyond the expected `runtime/gomad.go`, `runtime/rand.go`, and `runtime/proc.go` surface requires a minimized failing black-box test.

## Pattern Survey

### Analogous Features
- `Makefile:232` — Versioned tool binaries use file dependencies plus `.stamp` files; the stamp is touched only after the install command succeeds.
- `Makefile:301` — `go-install-tool` builds in a temporary directory and publishes the completed binary with `mv`, keeping partial output away from the stable tool path.
- `tools/gomadv2/test.sh:1` — A tool-local test driver self-locates, fails fast, excludes generated state from formatting checks, builds its executable, prepares fixtures, and runs focused tests with `test_dep`.
- `tools/gomadv2/internal/tests/script/script_test.go:16` — CLI black-box tests use `testscript` fixtures, isolated work directories, an explicitly prepared `PATH`, and a tool-local generated cache.
- `tools/gomadv2/internal/tests/script/testdata/gomadseeds.txtar:1` — A compact executable fixture verifies default and explicit seed selection through command output.
- `tools/gomadv2/internal/tests/script/testdata/testbuilding.txtar:1` — A black-box fixture checks cached/uncached behavior and invalidation after an input file changes.
- `tools/gomadv2/metatesting/metatest.go:259` — Determinism checking runs every test twice with the same seed and compares both complete checksums and complete log output.
- `tools/gomadv2/metatesting/metatest.go:295` — Seed coverage is expressed as repeated subprocess-style runs across a seed range.
- `tools/gomadv2/internal/tests/behavior/chan_test.go:124` — Existing behavior tests expose channel and scheduler ordering as complete output strings; the same file covers ready `select` cases at line 150.
- `tools/gomadv2/internal/tests/behavior/sync_test.go:10` — Mutex/WaitGroup contention exposes user-visible acquisition order for deterministic comparison.
- `tools/gomadv2/internal/tests/race/race_test.go:192` — A black-box subprocess harness removes ambient scheduler/debug variables, supplies `GOMAXPROCS=1`, captures combined output, and treats runtime fatal errors specially.
- `develop/buf-breaking.sh:51` — Repository validation creates a disposable tree with `mktemp`, registers cleanup with `trap`, and invokes Make against that isolated tree.
- None found for downloading and checksum-verifying an upstream source archive, applying a source patch with zero fuzz, validating patch path allowlists/denylists, or delegating from the root Makefile to a checked-in tool-local Makefile.

### Reusable Utilities
- `Makefile:301` — `go-install-tool` — Existing Make macro for version-pinned, temporary-directory tool construction and final-path publication.
- `tools/gomadv2/internal/gomadtool/gomadtool.go:34` — `BuildConfig.AsDirname` — Encodes OS, architecture, and race mode into a generated-artifact directory name; it is internal to the separate Gomad v2 module.
- `tools/gomadv2/gomadmain/main.go:98` — `hashFile` — Computes SHA-256 file content hashes used by Gomad v2 artifact invalidation; it is unexported within Gomad v2.
- `tools/gomadv2/gomadmain/main.go:128` — `copyFileIfChanged` — Avoids replacing identical cached artifacts so their modification times remain stable; it is unexported within Gomad v2.
- `tools/gomadv2/metatesting/metatest.go:259` — `CheckDeterministic` — Existing same-seed exact-result comparison helper within the Gomad v2 module.
- `tools/gomadv2/metatesting/metatest.go:295` — `CheckSeeds` — Existing seed-range execution helper within the Gomad v2 module.
- `Makefile:151` — `silent_exec` — Preserves command exit status while suppressing successful command output and surfacing failures.

### Convention Anchors
- Tool isolation: substantial tools live under `tools/<tool>`; Gomad v2 is a self-contained module rooted at `tools/gomadv2/go.mod:1`, with its own driver and tests.
- Root Make responsibilities: repository-wide variables and user-facing build/test targets live in `Makefile`; complex validation is delegated to scripts, as at `Makefile:480` and `develop/buf-breaking.sh:1`.
- Generated-state placement: repository-wide derived state uses hidden directories such as `.bin`, `.stamp`, and `.gomad`, ignored centrally at `.gitignore:1`; a single narrowly scoped artifact may instead use a directory-local ignore, as at `.github/actions/build-docker-images/scripts/.gitignore:1`.
- Cache ownership: Gomad v2 keeps translated source, binaries, and Go build cache beneath one disposable `.gomad` root, defined at `tools/gomadv2/internal/gomadtool/gomadtool.go:22` and configured at `tools/gomadv2/gomadmain/main.go:57`.
- Shell behavior: tool-local scripts self-locate before operating (`tools/gomadv2/test.sh:2`); validation scripts use Bash fail-fast settings and cleanup traps (`develop/buf-breaking.sh:19`, `develop/buf-breaking.sh:52`).
- Shell standards coverage: `Makefile:103` discovers all repository shell scripts, and `Makefile:485` runs ShellCheck over that complete set.
- Test tags: repository tests derive a tag set containing `test_dep` at `Makefile:48`, while Gomad v2’s standalone driver supplies `-tags=test_dep` explicitly at `tools/gomadv2/test.sh:12`.
- Fixture placement: command-level inputs and expected behavior are colocated under package-specific `testdata` directories, exemplified by `tools/gomadv2/internal/tests/script/testdata/gomadseeds.txtar:1`; generated working state remains outside those fixtures.
- Child-process errors: Gomad v2 forwards subprocess exit codes and reports non-exit failures at the command boundary (`tools/gomadv2/gomadmain/main.go:273`).
- Generated-output verification: generated Go files use the standard `Code generated by ... DO NOT EDIT.` marker, formatting skips them at `Makefile:397`, and CI’s final cleanliness gate rejects regenerated drift at `Makefile:780`.

### Proposed Alignment
Blend the repository’s hidden disposable-state, versioned file-target, temporary-build publication, fail-fast shell, and black-box `testdata` patterns. The custom Go archive/checksum pipeline, exact patch application, prohibited-area validation, and root-to-nested Make delegation have no existing repository analogue and therefore remain purpose-specific Gomad v3 concerns.

## Implementation Steps

1. **Write the black-box contract first**
   - Add a standalone standard-library-only fixture module at `tools/gomadv3/testdata/go.mod` so tests do not load Temporal dependencies.
   - Add small executable fixtures for activation and runtime randomness, scheduler spawn/yield/block/ready/close behavior, ready `select` cases, map create/clear/clone/iteration/NaN behavior, buffered and unbuffered channels, and mutex/semaphore-visible ordering.
   - Add a `go test` fixture package proving that the seed reaches a generated test binary.
   - Add `tools/gomadv3/test.sh` with helpers for exact output-and-status comparison across 100 fresh same-seed processes, diversity checks across different seeds, parallel seed isolation, watchdog timeouts, and cached/uncached `go run` and `go test` invocations.
   - Cover absent, `0`, `1`, maximum `uint64`, empty, malformed, and overflowing seeds; verify enabled mode forces one P and disables async preemption while disabled mode retains upstream settings.
   - Run the focused contract and record the expected RED failure because the v3 toolchain and runtime behavior do not exist yet.

2. **Implement reproducible toolchain construction**
   - Add `tools/gomadv3/build.sh` with fail-fast Bash settings and cleanup traps. Pin `go1.26.4.src.tar.gz` and SHA-256 `4f668a32fbfc1132e6a881fb968c2f1dada631492a339211735fbb255a42602d`.
   - Reject unsupported non-Unix hosts with an actionable error. Resolve and validate an installed bootstrap Go without assuming the repository's ordinary `go` binary is suitable.
   - Compute the build key from Go version, source checksum, patch and overlay checksums, host OS/architecture, bootstrap Go version, and a canonical build-environment revision; clear ambient Go build tuning before `make.bash` and reuse a completed immutable keyed build.
   - Snapshot and validate the patch and overlay before computing their checksums, download to a temporary file, verify the archive before publishing it to the cache, extract into a disposable directory, reject overlay collisions, copy the exact overlay snapshot, and apply the exact patch snapshot with zero fuzz to the checksum-verified source tree.
   - Serialize same-key builds with an atomic pre-owned lock, revalidate the immutable cache entry under the lock, run `src/make.bash` with an explicit `GOROOT_BOOTSTRAP` and canonical Go/C/C++ build inputs, validate the resulting version, then atomically publish the keyed build through the stable `.toolchain/bin/go` path and write the success stamp last.
   - Add `tools/gomadv3/Makefile` targets for `toolchain` and `test`, retaining file/stamp dependencies where inputs are statically knowable and delegating dynamic build-key validation to `build.sh`.
   - Verify a second build reuses the same keyed artifact, a patch change selects a new key, and failed patch/build attempts cannot leave a valid stamp or replace the stable binary.

3. **Add the minimal deterministic runtime customization**
   - Create `go1.26.4.patch` against the exact source archive for upstream modifications and add net-new runtime source under `overlay`, preserving all upstream comments and keeping disabled branches structurally unchanged.
   - Add `src/runtime/gomad.go` through the overlay with `gomadEnabled`, `gomadSeed`, and `gomadInit`. Scan the raw Unix environment before ordinary runtime environment initialization, distinguish absence from an empty value, parse unsigned 64-bit seeds without allocation-sensitive dependencies, and reject invalid values before user initialization.
   - In deterministic mode, have `gomadInit` force `debug.asyncpreemptoff`, enable the existing `randomizeScheduler` path, and leave unsupported platforms outside the activation contract.
   - Change `src/runtime/rand.go:randinit` so deterministic mode initializes the existing global ChaCha8 state solely from `gomadSeed`; after minimized scheduler failures, seed each M directly from the same seed while retaining `rand`, `randn`, `cheaprand`, `cheaprandn`, and `maps_rand` unchanged.
   - Change `src/runtime/proc.go:schedinit` to call `gomadInit` immediately before `randinit`, select one initial P when enabled, and keep the existing GOMAXPROCS environment/default branches unchanged when disabled. Convert `randomizeScheduler` from the race-only constant to the smallest state that Gomad initialization can enable, disable `sysmon`, and reset M0 random state and its scheduler tick before user initialization to remove observed startup-timing drift. After staged channel and mutex waiters prove FIFO behavior has no cross-seed diversity, add the permitted seeded choice inside the existing local `runqget` representation only for one-P Gomad mode.
   - Run the activation/random golden-sequence tests through RED then GREEN before enabling the wider scheduler, `select`, map, channel, and synchronization suite.

4. **Enforce source scope without numeric budgets**
   - In `tools/gomadv3/test.sh`, inspect patch headers and overlay paths before building and reject changes outside the runtime package or inside prohibited compiler, `cmd/go`, public-package, map-implementation, channel, GC, platform, assembly, generated, or test-output areas.
   - Verify the patch applies exactly to a pristine checksum-verified Go 1.26.4 tree and contains no binary/generated hunks.
   - Keep behavioral tests external under `tools/gomadv3/testdata`. If any deterministic behavior still fails, preserve and minimize that failure before adding another runtime hunk; do not expand the patch speculatively.

5. **Expose the opt-in root workflow and document the contract**
   - Add `GOMADV3_GO` and `gomadv3-go`, `gomadv3-run`, and `gomadv3-test` targets to the root `Makefile`, delegating construction to `tools/gomadv3/Makefile`.
   - Validate required `GOMADSEED`, `GOMADV3_RUN`, and `GOMADV3_PACKAGES` values before invoking Go. Set `GODEBUG=asyncpreemptoff=1`, `GOMAXPROCS=1`, and the seed consistently; include `-tags test_dep` for repository tests and preserve argument forwarding through `GOMADV3_ARGS`.
   - Add `tools/gomadv3/.gitignore` for `.toolchain/` and `README.md` covering build/run/test commands, Unix-only support, deterministic-input requirements, unsupported cgo/race/timer/I/O/finalizer behavior, CPU-bound starvation, shared-random-stream coupling, one-P throughput, cache disk use, and the security risk of deterministic map seeds.
   - Keep the repository's normal Go, unit-test, and build targets unchanged.

6. **Verify compatibility, repeatability, diversity, and standards**
   - Run patch validation before every toolchain test, then run the upstream `runtime` tests with `GOMADSEED` absent.
   - Run all same-seed fixtures for at least 100 fresh processes and compare complete output and exit status. Run distinct seeds only against fixtures with real alternatives and require more than one observed result.
   - Repeat direct-binary, `go run`, and `go test` checks with warm/cold caches and parallel child processes; run the watchdog case to pin the cooperative-scheduling limitation without using wall time as scheduler input.
   - Run shell formatting/static checks applicable to the new scripts, `make fmt-imports` if Go fixture formatting requires it, focused Go tests with `-tags test_dep`, and finally `make lint-code`.
   - Re-read `GOMAD_ALT.md` against the patch and verification output. Report any deliberately narrowed claim rather than weakening or skipping a failing test.

## Verification

- `make gomadv3-go` — produces `tools/gomadv3/.toolchain/bin/go`; a repeat invocation reports/reuses the same complete build key.
- `make -C tools/gomadv3 test` — validates patch scope and exact application, runs disabled upstream runtime tests, and passes all activation and deterministic black-box cases.
- `GOMADSEED=1 make gomadv3-run GOMADV3_RUN=./tools/gomadv3/testdata/scheduler/main.go` — repeated invocations have identical complete output and exit status.
- `GOMADSEED=1 make gomadv3-test GOMADV3_PACKAGES=./tools/gomadv3/testdata/gotest/determinism_test.go` — proves the custom test binary receives deterministic mode with `test_dep` enabled.
- Invalid or missing required Make variables fail before Go starts and name the variable and expected usage.
- Empty, malformed, and overflowing `GOMADSEED` values fail before user `init`; seeds `0` and maximum `uint64` succeed.
- Without `GOMADSEED`, explicit upstream GOMAXPROCS/debug behavior remains effective and representative map/scheduler runs retain upstream randomness.
- `make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=HEAD lint-code` — exits successfully after the implementation and fixture changes without mutating unrelated files.

## Failure Modes and Trade-offs

- Interrupted downloads, extraction, patching, or compilation remain confined to temporary/keyed paths; no success stamp is published and the prior stable toolchain stays usable.
- A 10x seed workload scales through independent processes and cores, while each deterministic process intentionally remains at one P. The immutable cache avoids rebuilding per seed at the cost of substantial local disk usage.
- The shared runtime random stream makes schedules sensitive to program changes and added random consumers. This is accepted for the fixed-program reproducibility contract and avoids extra PRNG/domain machinery.
- External readiness, timers, signals, cgo, race mode, and finalizers can vary the runnable set and remain outside the contract. CPU-bound goroutines can starve because async preemption is disabled.
- Deterministic map seeding weakens hash-randomization security; the mode is restricted to trusted tests and must not be enabled in production.
- No numeric patch budget is enforced. Patch growth is limited by automated prohibited-area checks, exact-source applicability, preserved upstream disabled paths, and the requirement for a minimized failing test before each additional runtime hunk.

## Context Files

- `GOMAD_ALT.md` — approved behavior, scope, patch-minimization, workflow, and success criteria.
- `Makefile` — root target, stamp, tool installation, test-tag, formatting, and lint conventions.
- `tools/gomadv2/test.sh` — closest tool-local fail-fast test-driver convention.
- `tools/gomadv2/metatesting/metatest.go` — existing exact same-seed and seed-range comparison semantics.
- `tools/gomadv2/internal/tests/behavior/chan_test.go` — established observable scheduler/channel/select fixture shapes.
- `$(go env GOROOT)/src/runtime/rand.go` — exact Go 1.26.4 random-state initialization and map-randomness bridge.
- `$(go env GOROOT)/src/runtime/proc.go` — exact Go 1.26.4 bootstrap, GOMAXPROCS, and randomized scheduler paths.
