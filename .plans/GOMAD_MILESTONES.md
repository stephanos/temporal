# Gomad v3: Milestones to a Deterministic Temporal Functional Test

**Plan date:** 2026-09-08

## Purpose

This document is the delivery ladder for one goal: run any test in the Temporal functional
package (`./tests`, built on the `testcore` one-box cluster with in-memory SQLite and loopback
gRPC) under Gomad v3 so that the same seed produces the same run, and a retained artifact
replays byte-exactly. [GOMAD3_NEXT.md](GOMAD3_NEXT.md) remains the capability roadmap across
all four tracks. This document orders the subset of that work that the functional-test goal
depends on, adds the repairs the roadmap does not know about, and states an acceptance
criterion for each step that a reviewer can check with a command.

F0–F7 deliver the functional-test goal. F8 follows with architecture maintenance that
consolidates protocol and execution-policy ownership while preserving the qualified behavior.
F9 extends the supported target shape from the server's own module to a downstream module that
embeds the server; it depends on F7, not on F8.

The ladder is strictly ordered. Each milestone assumes the previous one's acceptance criteria
hold. A milestone whose criteria fail blocks the next one; it never gets narrowed to pass.

## Work tracking

The remaining work of every open milestone is tracked as a flow-next spec under `.flow/specs/`,
one spec per milestone, each depending on the previous one so the ladder order is enforced by
`flowctl ready`. The specs, their tasks, and their acceptance criteria (R-IDs) are the
authoritative list and order of work; this document keeps the milestone rationale, constraints,
and status history. When a spec's scope changes, change the spec and summarize the change in the
milestone's status here.

| Milestone | Spec | State |
| --- | --- | --- |
| F0 | none | done |
| F1 | `fn-95-gomad-f1-restore-the-checkout-on` | done |
| F2 | `fn-96-gomad-f2-close-the-go127-port-on` | done (the DTrace clock audit needs a root run) |
| F3 | `fn-97-gomad-f3-qualify-the-frontend` | done on darwin/arm64 (linux/amd64 not re-measured) |
| F4 | `fn-98-gomad-f4-close-the-tests-capability` | done on darwin/arm64 (activity batch cancel is tier-3 `intermittent`) |
| F5 | `fn-99-gomad-f5-one-workflow-executing` | done on darwin/arm64 (linux/amd64 not re-measured) |
| F6 | `fn-100-gomad-f6-a-package-level-functional` | done on darwin/arm64 (linux/amd64 not re-measured) |
| F7 | `fn-101-gomad-f7-any-functional-test-and-ci` | open |
| F8 | `fn-102-gomad-architecture-consolidate` | open; depends on F7; plan reviewed |
| F7+ | `fn-103-gomad-seeded-virtual-clock-ticks` | open; depends on F7; seeded virtual-clock ticks (default decided by measurement) |
| F9 | `fn-104-gomad-run-a-downstream-cell-under-the` | open; depends on F7; downstream-module targets ([GOMAD_CLOUD.md](GOMAD_CLOUD.md)) |

Work a spec with `/flow-next:work <spec>`; list what is ready with `flowctl ready`.

## Baseline on 2026-09-08

The assessment that produced this document verified the following on the working tree.

- The root module declares `go 1.27` and `toolchain go1.27.0` since 2026-08-23. Gomad v3 pins
  `go1.26.4` in `tools/gomad3/toolchain/version/version.json` and builds targets with
  `GOTOOLCHAIN=local`. No Temporal package builds under the patched toolchain today.
- `tools/gomad3/.toolchain` and `tools/gomad3/.bin` do not exist in a fresh checkout. The F1
  spike on 2026-09-08 rebuilt them: the go1.26.4 build takes 75 seconds of wall time and 770 MB
  on disk, and `gomad doctor` reports the host, toolchain, runner, all four adapters, and the
  artifact store available.
- Building the probe with the patched toolchain fails with
  `go: go.mod requires go >= 1.27 (running go 1.26.4; GOTOOLCHAIN=local)`. Without
  `GOTOOLCHAIN=local` the patched `go` binary silently switches to the stock go1.27.0 toolchain
  from the module cache and reports `go1.27.0`, so any wrapper that drops that variable builds an
  unpatched binary. The runner sets it; the root Makefile's `gomad3-run` and `gomad3-test`
  targets rely on the nested Makefile doing the same.
- The conformance fixture corpus under `tools/gomad3/internal/gomadtool/conformance/testdata`
  was never committed because the root `.gitignore` ignores every `testdata/` directory. The
  runtime tier, the interception tier, and every `io_*_toolchain_test.go` cannot run. The F1
  spike confirmed `make -C tools/gomad3 test-runtime` fails on the first fixture. The missing set
  is 25 runtime fixtures (`activation`, `activation_io`, `automatic_gc`, `channels`,
  `choice_replay`, `clock`, `clock_bench`, `clock_cgo`, `clock_deadlock`, `clock_gotest`,
  `clock_io`, `clock_race`, `clock_spin`, `clock_synctest`, `gotest`, `intercept`, `maps`,
  `preemption`, `random`, `runqueue`, `scheduler`, `scheduler_min`, `select`, `sync`,
  `toolchain`), the `interceptfail` set, the boundary canaries `io_filesystem`,
  `io_filesystem.host-escape`, and `io_net`, and the compatibility-pack fixture `v041`. No
  generator or embedded copy exists in the tree, so they have to be re-authored from the
  expectations in the `runtime_*.go` conformance files.
- The root `Makefile` targets `gomad3-run` and `gomad3-test` point at `tools/gomad3/exec.sh`,
  which does not exist. The script lives under `internal/gomadtool/conformance/scripts`.
- `tools/gomad3/simulation/parity/manifest.go` pins `gomad3.simulation-spec/v6` while
  `tools/gomad3sim/types.go` declares `v7`. `tools/gomad3integration/qualification/temporal.json`
  is schema `gomad3.qualification-set/v3` while `tools/gomad3/qualification/set/set.go` accepts
  only `v1`. Both tests in `tools/gomad3integration` fail.
- The Temporal corpus qualifies 5 of 16 leaf unit tests. Ten of the eleven unsupported cases
  are `import:os/exec` reached through cloud credential packages; one is arm64 assembly in
  `github.com/cespare/xxhash/v2`.
- The only functional probe, [`tests/gomadfunctional/frontend_test.go`](../tests/gomadfunctional/frontend_test.go),
  starts the one-box cluster and completes one `GetSystemInfo` call. It executed once under the
  experimental linked capability mode with the approved pack
  `temporal-functional-compute-darwin-arm64`. It has never been through `gomad qualify` and is
  absent from the corpus. No functional test has a determinism claim.
- The dependency closure of `./tests` with tags `test_dep` holds 27 non-standard packages that
  import a forbidden capability and are covered by no pack. The ones live in every cluster are
  `go.uber.org/fx` and `go.temporal.io/server/temporal` (`os/signal`),
  `go.temporal.io/server/common/config` (`os/exec` for the password command), and
  `github.com/prometheus/client_golang/prometheus` (`golang.org/x/sys/unix`). Every assembly and
  linkname finding in that closure is already covered by an approved pack or adapter.
- The I/O transcript is a fixed 64 MiB mapping of 128-byte records
  (`toolchain/runtime/overlay/src/internal/gomadtrace/trace.go`), about 524k modeled
  operations per execution. Overflow is a failed run with no exact replay.
- `make -C tools/gomad3 toolchain` fails before building anything because `validate-toolchain`
  finds `choice/internal/wire/wire_generated.go` stale. Running `make -C tools/gomad3 generate`
  rewrites six generated files (532 lines): the choice wire codec's implementation digest, every
  boundary probe ID in `deterministicio/boundary_generated.go`, the livecap and gomadcap
  protocol tables, and `spec_go126.go`. The diff is identical under a go1.26.4 and a go1.27.0
  host, so the committed outputs are stale against their committed inputs, not host-dependent.
  Any retained artifact from before the regeneration carries the old identities. The chain also
  needs two passes to converge: `protocol-generate` embeds `ProducerImplementationSHA256` and
  `GuardImplementationSHA256` in `target/internal/livecap/protocol_generated.go`, digests of the
  gomadcap overlay file that the same pass rewrites, so `validate-toolchain` fails again after
  one `make generate` and passes after a second.
- Spec fn-81 in its first
  revision deleted every gomad tree, the functional probe, and its pack. Its 2026-09-08
  amendment retains Gomad v3 and the probe, and deletes gomad, gomad1, and gomad2 together with
  the Gomad v3 parity manifest that read gomad2 source paths.

## Constraints that apply to every milestone

- **No policy widening.** A milestone never grants `syscall`, `os/exec`, `os/signal`, or
  `golang.org/x/sys` generically. Every exception is an exact compatibility pack bound to a
  module version, go.sum hash, per-file SHA-256, owner, and workload, reviewed under the
  existing `discover`, `review`, `generate --approve-review`, `check`, `qualify` flow.
- **No source translation and no test rewriting.** Determinism comes from the patched
  toolchain and the reviewed boundary. A Temporal test that needs a Gomad-specific overlay of
  its own source, as gomad1 required, is a blocker to record, never a fix to ship.
- **Fail-closed stays.** An unmodeled boundary operation terminates the process. A milestone
  that needs a new modeled operation adds it with a semantic contract, a resource bound,
  transcript coverage, exact replay, and a negative test, per COMPAT-5 in
  [GOMAD3_NEXT_COMPATIBILITY.md](GOMAD3_NEXT_COMPATIBILITY.md).
- **Evidence over narration.** A milestone is done when its command produces the stated
  report on a clean checkout. A passing local run that depends on untracked state does not
  count.
- **Platform.** The boundary manifest qualifies `darwin/arm64` and `linux/amd64` (added
  2026-09-26 as one manifest with a per-platform declaration override, not a second bundle).
  Each platform is its own qualification and artifacts replay only where they were produced.
  The macOS sandbox test and the DTrace clock audit remain `darwin/arm64` only. The modernc
  libc adapter has a Linux target since 2026-09-27 (COMPAT-5): it hooks the musl syscall
  trampolines, and `modernc-libc-xsys-v047-linux-amd64` admits the facts the rewritten module
  still carries, so the libc and SQLite core workloads qualify on both platforms. Each
  platform's compatibility packs are its own; `compatibility-pack-qualification` qualifies the
  requests that name the host.
- **Server source changes are allowed but bounded.** A change under `common`, `service`,
  `temporal`, or `tests/testcore` is acceptable when it isolates an optional provider behind a
  build tag or an injection seam and the default build is unchanged. A change that alters
  runtime behavior for production builds needs its own review outside this plan.

## F0: decide retention

**Outcome.** The repository either keeps Gomad v3 with a carve-out from fn-81 or drops the goal
this document serves. Nothing below starts until this is settled, because fn-81 removes the
toolchain, the probe, and the pack in its first two commits.

**Status.** Applied on 2026-09-08. fn-81 and its five tasks now retain Gomad v3, the probe, and
the pack, and delete gomad, gomad1, gomad2, and the parity manifest.

**Details.**

- Amend fn-81 so its deletion set keeps `tools/gomad3`, `tools/gomad3sim`,
  `tools/gomad3integration`, `tests/gomadfunctional`, the root Makefile `gomad3*` targets, the
  `gomad3` workflow, and the root go.mod `require` and `replace` for `github.com/temporalio/gomad`
  only if `tools/gomad2` survives as the parity reference. If gomad2 goes, the parity manifest
  under `tools/gomad3/simulation/parity` loses its source paths and must be retired in the same
  change. **Resolved: fn-81 took the second branch** — gomad2 was deleted and the parity manifest
  was retired in the same commit, along with its `script_policy.go` check, the
  `tools/gomad3integration` parity assertions, and the README and glossary entries that described
  it.
- `tools/gomad`, `tools/gomad1` and, if the parity manifest is retired, `tools/gomad2` are
  deleted as fn-81 specifies. gomad1 has no `go.mod`, so its 54,590 lines compile into every
  root-module build today for zero callers.

**Acceptance.**

- fn-81's deletion set and retention set name each gomad tree explicitly with a disposition.
- `go build -tags 'test_dep integration' ./...` passes after the amended fn-81 lands.
- `go run ./tools/planindex` passes with this document registered.

## F1: restore the checkout

**Spec.** `fn-95-gomad-f1-restore-the-checkout-on`

**Outcome.** A developer on a clean `darwin/arm64` checkout can build the toolchain and run
every Gomad v3 gate that exists today, and the two integration contract tests pass.

**Status.** Partially applied on 2026-09-26, on the `gomad` branch rebuilt as upstream
`951c5516e` plus Gomad v3 (the Umpire, Testpilot, canary, and Lean trees are gone, so the root
`.gitignore` no longer ignores `testdata/` and needs no allowances). Done on Linux: the
generated outputs validate and the Linux CI job now runs `make validate`; the manifest loader
reads `gomad3.qualification-set/v3` and the core corpus moved to it; the root wrappers point at
the conformance `exec.sh`; the `tagged` wrapper fixture exists; and the runtime-tier fixture
corpus is authored under `internal/gomadtool/conformance/testdata` as module `gomad3.test`:
`activation`, `activation_io`, `automatic_gc`, `channels`, `choice_exploration`, `choice_replay`,
`clock`, `clock_bench`, `clock_cgo`, `clock_deadlock`, `clock_gotest`, `clock_io`,
`clock_race`, `clock_spin`, `clock_synctest`, `gotest`, `io_fd5`, `maps`, `preemption`,
`random`, `runqueue`, `scheduler`, `scheduler_min`, `select`, `sync`, the `intercept` package,
and the nine `interceptfail` packages named by `compiler-tests.json`. Once `linux/amd64`
became a qualified platform (below) the patched toolchain was built here and the builder,
live-capability, interception, upstream, overlay, and world tiers passed; the runtime tier
passed every clock, linking, scheduling, perturbation, host-load, map-family, oracle, and
activation check and then found a same-seed divergence in the repeatability sweep: about 0.3
to 1 percent of runs of the allocation-heavy fixtures (`channels`, `sync`, `automatic_gc`,
`scheduler`) print a different interleaving for one seed, with `NumGC == 0`, a different
`HeapAlloc`, and no divergence under `GOGC=off`. F2 traced that divergence to the seeded
scheduler drawing from per-M random streams whenever no choice trace was attached, so which M
picked up the P after a hand-off changed the interleaving; the scheduler now draws from
process-wide seeded states and the runtime tier's repeatability sweep passes on linux/amd64
(the GC-dimension risk this document names remains a risk, not an observed defect; on
2026-09-27 one `automatic_gc` seed diverged on its fourth same-seed run while the runtime tier
shared the machine with two other test suites, and the tier passed when rerun alone, so keep
the sweep on an unloaded runner until that risk is closed). The I/O fixture corpus was
completed on 2026-09-27: `io_filesystem` (every modeled and every refused
`os` operation, the `isolated` host-escape mode), `io_net` (every modeled and refused `net`
operation over the loopback model), `io_net_races` (the ten close, deadline, backlog, and
port-exhaustion cases), `io_signal` and `io_user` (guarded mode), `io_entropy`, `io_ro_mount`,
`io_ro_mount_failure`, and the `io_failure` go-test fixture; `libc_adapter` and `sqlite_adapter`
landed the same day. On linux/amd64 `TestBoundaryManifestSemanticCanaries` now observes a
positive probe for all 131 manifest entries and every `io_*_toolchain_test.go` and
`replay_io_integration_test.go` passes. Authoring `io_signal` exposed a Linux-only defect in
guarded mode: the compiler guards every exported entry point of the `syscall` package, and on
Linux `syscall.Syscall` has a Go body (darwin's is a body-less libc trampoline), so a guarded
target tripped `GOMAD_CAPABILITY_DENIED` on its own `fmt.Println`. `syscall.Write` now hands the
descriptors it still allows (stdout, stderr, and the trace transport) to
`runtime.gomadSyscallWrite`, which reaches the kernel without the guarded trampoline; the
guard on `syscall.Syscall` itself stays. CONSIDER(gomad): gosim's approach, a syscall-number
dispatcher behind `syscall.Syscall*` that models the few numbers the stdlib reaches and denies
the rest, would subsume this routing on Linux and is the natural next step if guarded mode
needs more than output; it does not make the boundary platform-agnostic, because the stock
runtime's netpoll, clock, and output paths stay host-specific unless the scheduler is replaced
by source translation, which this plan rules out. Still open in the host tier on Linux, all
pre-existing and reproduced on the commit before the fixtures landed: the `runner` package's
fake preparer pins `darwin/arm64` and `go1.26.4` provenance, so its campaign, inspect, portable
plan, and umask tests fail with "deterministic I/O requires Go go1.27.1 on linux/amd64" and
`TestRunReportsPeriodicProgressWhileTargetIsRunning` waits forever for an executor that never
starts; the two world-transport replay tests report "invalid I/O terminal frame"; and the
execution package's choice-trace tests report "choice trace unterminated" and its process-group
cleanup test finds the group still present. The non-Linux branch of
`COMPATIBILITY_PACK_QUALIFICATIONS` in `tools/gomad3/Makefile` still qualifies
`modernc-libc-xsys-v041` against `internal/compatibilitypack/testdata/v041`, which was never
committed until the darwin run below authored it. Three more defects surfaced while qualifying on Linux: the run-queue choice
decision kept two 8 KiB candidate buffers on the system stack, which Linux sizes at 16 KiB for
non-main threads, so every seeded run with choice tracing died with `morestack on g0` (the
buffers now live in static scheduler scratch); the adapters' source-inventory pins were
recorded with the pre-rename `gomadv3.` digest header and had been stale on every platform
since 2026-08-23 (re-pinned); and the root module rebuilt on upstream
`951c5516e` carries `golang.org/x/net v0.58.0` and `google.golang.org/grpc v1.83.2` while the
adapters pin `v0.57.0` and `v1.80.0`, so every Temporal-corpus analysis fails with
"unsupported golang.org/x/net version" until the adapters are re-pinned (COMPAT-5 upgrade
flow; the two rewritten x/net files are byte-identical between the versions). Re-pinned on
2026-09-27 to `golang.org/x/net v0.58.0` and `google.golang.org/grpc v1.83.2`: the three
rewritten files and both prepared packages are byte-identical to the previous pins, so only the
module inventories moved; the gRPC `internal` prepared source set is now recorded per platform
like the x/net one.

Done on darwin/arm64 on 2026-09-27, on the `gomad` branch rebased onto upstream main the same day
(the rebased tree equals a clean merge). With `GOROOT` on a stock go1.27.1, `make gomad3` built
the go1.27.1 toolchain from source (key `85c444f9…`) with no darwin build failure, and
`gomad doctor` reports the host, toolchain, runner, all seven adapters, and the artifact store
available. `make -C tools/gomad3 test` passes all ten tiers; the harness, toolchain, intercept,
overlay, world, builder, live-capability, and upstream tiers passed unchanged, and the host tier
needed five repairs (commit `45e6788d97`): the darwin source-set pins of the libc, memory, x/net,
and gRPC adapters were stale and are re-pinned, with the darwin `v047` and `isatty-v021` libc
packs regenerated (only adapter identities moved); the `libc_adapter` fixture's `fstat` call is
split per platform because darwin's modernc libc has no `Tstat`; the exec-provenance check now
compares the stamped `go1.27.1-X:nogreenteagc` version that `GOEXPERIMENT=nogreenteagc` builds
carry, which failed on every platform; the `runner` fake preparer takes its target from the
deterministic profile instead of pinning `darwin/arm64` and `go1.26.4`, which removes the pin
behind the Linux failures and the hang recorded above (not yet rerun on linux/amd64), and a stale
seed-environment order assertion it exposed is fixed; and the boundary test manifest's `os`
package fingerprint is re-pinned. `make gomad3-integration-test` passes with both tests running,
and the core qualification set reports selected 5, supported 5, unsupported 0, failed 0, and
infrastructure errors 0, with `choice_replay_exact` for all five workloads
(`concurrency-state-invariant`, `filesystem-transaction`, `loopback-tcp-roundtrip`,
`modernc-libc-boundary`, `sqlite-transaction`). `compatibility-pack-qualification` failed at
first because the `v041` fixture was missing; commit `7ea97c052e` authors it as module
`gomad3.compatibility.v041` (modernc libc v1.72.3, x/sys v0.41.0, test
`TestLibcCompatibilityClosure`) and regenerates `modernc-libc-xsys-v041`, whose libc and memory
adapter identities had drifted like `v047`'s (capabilities, linkname directives, and platform
scope unchanged; review
`sha256:e93c386eae937f3b91cf28da8549527db8cbc724ec96061e99ab2dd7f7e34eed`), so all four darwin
packs qualify. One host prerequisite, not a Gomad defect: the runtime tier's `clock_cgo` build
needs a working host `clang`, and a `clang` from mise's lean4 install that shadows Xcode's cannot
find `stddef.h`; the runs put `/usr/bin` ahead of it on `PATH`.

**Details.**

- Commit the regenerated files from `make -C tools/gomad3 generate` and verify
  `make -C tools/gomad3 validate-toolchain` passes on a clean checkout. Make the generator chain
  converge in one pass by ordering the gomadcap overlay generation before the livecap protocol
  digest, or by having `generate` loop until `-check` passes. Add that gate to the
  gomad3 workflow's `core` job so generated outputs cannot drift again without a red check.
- Re-author and commit the conformance fixture corpus. Each fixture is a small Go program whose expected output, exit
  status, and timing are pinned by the `runtime_*.go` conformance files, so those files are the
  specification. Author `io_filesystem`, `io_net`, and `io_filesystem.host-escape` first, since
  the boundary manifest names them as the semantic canary, then `activation`, `clock`, `maps`,
  `select`, `runqueue`, and `choice_replay`, which the core qualification set and the CI
  assertion depend on. The rest follow in the order `make -C tools/gomad3 test-runtime` fails.
- Point the root Makefile at the real `exec.sh` or move the script to the path the Makefile
  expects. Pick one; the compatibility-pack request records the path.
- ~~Bump `HarnessSpecSchema` in `tools/gomad3/simulation/parity/manifest.go` and the JSON manifest
  to `gomad3.simulation-spec/v7`.~~ Moot: fn-81 retired the parity manifest.
- Reconcile the qualification-set schema. Either the loader in `qualification/set/set.go`
  accepts `v3` with `suites` and `run_timeout`, or the two manifests return to `v1`. The CI
  workflow asserted a `v6` report schema that nothing produces; the loader now reads manifest
  `v3` and CI asserts the `v1` report schema the tool emits.
- Add the missing `tools/gomad3integration/testdata/tagged` fixture that `TestPublicWrappers`
  runs, or delete that test.

**Acceptance.**

- `make gomad3` builds the toolchain from source and `tools/gomad3/.bin/gomad doctor` reports
  the runner available.
- `make -C tools/gomad3 test` passes, including `intercept-test`, `test-runtime`, and every
  `*_toolchain_test.go`.
- `make gomad3-integration-test` passes.
- The core qualification set reports `selected == 5`, `supported == 5`, `unsupported == 0`, and
  every workload `choice_replay_exact`.

## F2: port the toolchain to go1.27

**Spec.** `fn-96-gomad-f2-close-the-go127-port-on`

**Outcome.** The patched toolchain satisfies the root module's `go 1.27` directive, so Temporal
packages build under it again.

**Details.**

- Pin the newest go1.27 patch release whose official source checksum can be verified from
  several independent package sources (go1.27.1 on 2026-09-27). Materialize the previous patch
  onto the new source, resolve rejects by hand across the 16 patched files, then
  `patch-regenerate`, `make generate` (twice), and `make -C tools/gomad3 upgrade-dossier`.
- The compiler fingerprint check fails the build for any intercepted `os` or `net` function
  whose body changed upstream with a stable signature. Each such function needs a re-reviewed
  entry in `deterministicio/boundary/manifest.json`.
- Update `version.json` (archive URL, SHA-256, patch and overlay allowlists), regenerate
  `version_generated.mk`, the expected-intercepts file, and the boundary report and upgrade
  guide for the new version.
- The DTrace clock audit needs root. Record whether it ran; a dossier without it stays
  `qualified=false` and that is the honest state, not a failure of this milestone.

**Status.** Applied on 2026-09-27 on `gomad-linux`, pinned to go1.27.1 (go.dev and dl.google.com
are unreachable from cloud sessions, so the official `go1.27.1.src.tar.gz` checksum
`4e408aba…238b1` was cross-checked against the Homebrew, Void, and nixpkgs package sources; the
source itself came from the `golang.org/toolchain` module, whose 16 patched files are identical
to go1.27.0's). The go1.26.4 patch applied with one reject (`schedinit`'s comment text around
`gomadInit()` changed) and needed two runtime follow-ups: the runtime split `m.cheaprand` into a
32-bit and a 64-bit field, so the seeded `mrandinit` now sets both, and `internal/runtime/math`
lost `Mul64`, so the select choice uses `math/bits`. Go 1.27's linker rejects a pull linkname
to the assembly `runtime.nanotime1`, so `internal/gomadfs` reads the host clock through the
push-linknamed `runtime.gomadWallNanotime`. The boundary manifest is `go1.27.1-v1`: only
`(*File).Chdir` (gained a `testlog` record) and `(*Resolver).LookupSRV` (doc comment) changed
bodies, the os.Pipe override for linux/amd64 is unchanged, and the intercept count stayed at
131. The regenerated patch (`toolchain/runtime/go1.27.1.patch`) applies with zero fuzz and the
generated compiler spec is now found by the `spec_go` prefix instead of a hard-coded name. On
linux/amd64 every tier passes (`test-harness`, `test-toolchain`, `intercept-test`,
`overlay-test`, `world-test`, `test-builder`, `test-live-capability`, `test-upstream`,
`test-runtime`) and `core-qualification-set` reports 3 supported with exact replay and the 2
expected `unsupported_target` results; the Linux CI job now gates on the runtime tier too. Two
Linux-only findings were fixed along the way: the upstream tier pins `GOROOT` to its workspace
link because go1.27's `go list` resolved GOROOT directories against this module when run from
a GOROOT nested inside it, and the seeded scheduler drew from per-M random streams without a
choice trace (see F1's status). Not done here: the macOS `upgrade-dossier` run, the DTrace
audit, and the `GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256` variable for the go1.26.4-v2 to
go1.27.1-v1 boundary diff, all of which need the darwin/arm64 runner; and the Temporal corpus
acceptance, which the adapter re-pin (F1 status) has since unblocked but which still needs the
darwin/arm64 run.

Done on darwin/arm64 on 2026-09-27. `tools/gomad3/.toolchain/bin/go version` reports go1.27.1
darwin/arm64 and `validate-toolchain` passes. `make -C tools/gomad3 upgrade-dossier` against the
go1.26.4 baseline `75f4d101c5` passes every gate except `host-clock-escape`, which needs root:
manifest-validation, toolchain-and-compiler, host-world-and-probes, builder, runtime, and
disabled-upstream pass, the retained gomad3-core corpus is 5/5 supported with exact replay, and
the dossier stays `qualified=false` only for the clock audit. The first run's clock-audit gate
failed for a different reason: the `testdata/clock_audit` test program was lost in the gomadv3
rename (the root `.gitignore` ignores `testdata/`) and the script looked for `clock_audit.d`
under `scripts/` instead of the module root; commit `68d36aadfe` restores both. The boundary
diff from go1.26.4-v2 to go1.27.1-v1 (0 added, 0 removed, 131 changed) was reviewed entry by
entry and approved by rerunning with the digest
`sha256:86f18fc8cda31fe234d345f70384e8d5ae94e9cbb883beb5f8399e73f6d4798f`, which is the value
the `GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256` repository variable needs; setting it needs a repo
admin and is not done. The reviewed entries:

- **Manifest.** `go_version` moved from go1.26.4 to go1.27.1 and `manifest_version` from
  go1.26.4-v2 to go1.27.1-v1; `hook_policies`, `reviewed_candidates`, and `platforms` are
  unchanged, and the linux/amd64 `os.Pipe` override only moved within its entry, which
  canonical JSON ignores.
- **`os.(*File).Chdir`.** Its declaration changed because upstream logs `Getwd()` through
  `testlog.Logger()` after a successful `Fchdir`. The entry is modeled, so the hook replaces the
  body and the testlog record never runs under the model.
- **`net.(*Resolver).LookupSRV`.** Only a doc comment on the returned cname changed; the body is
  identical.
- **128 package-hash-only entries.** 59 in `os`, 68 in `net`, and `os/signal.Stop`, whose own
  body hashes identically. The upstream changes behind the new package hashes (`dirFS.ReadLink`'s
  error path, the `ReadFile` doc comment, darwin `readdir`'s EBADF skip, `root_*.go`, the lazy
  `signal.Notify` handler, `signalError.Is`) all lie outside the intercepted declarations.
  `os/user` is byte-identical upstream.

The DTrace clock audit did not run: the session had no non-interactive root (`sudo -n true`
asks for a password). `make -C tools/gomad3 clock-audit` passes its generator and validation
checks, finds the toolchain ready, and stops with exit 2 at `gomad3 clock audit requires root
DTrace privileges`, so root is the only thing missing; `sudo make -C tools/gomad3 clock-audit`
completes it.

`make gomad3-qualification` reproduces the Temporal corpus with expectations met: 5 supported
and 11 `unsupported_target` tier 2 workloads with the same first-blocker paths as the manifest
(temporal-cache-concurrent's darwin blocker is `xxhash_arm64.s`), both tier 3 workloads
`nondeterministic` inside their `intermittent` expectations, and 0 infrastructure errors over
18/18 completed. Three defects stood in the way, all fixed in commit `a8777f5d73`.
`temporal-functional-compute-darwin-arm64` pinned `golang.org/x/crypto` v0.54.0 while the
module is at v0.55.0, so the pack never activated; it is refreshed through discover, review, and
generate (review `sha256:f17760eeb6fa02c315586cf99923c0f55aa3bf92f75f8508728e61fd7b32614f`,
only the module pin moved). The new `temporal-functional-tests-darwin-arm64` pack (review
`sha256:b009df6c9380eeb9766b3bf7144df7d6477018ee879ae8586ed2a2742b841ce6`) admits the
`syscall` and `golang.org/x/sys/unix` imports of the Prometheus client's darwin process
collector, the counterpart of the linux procfs admission, and closes the `./tests` closure on
darwin (see F4's status); both darwin packs pass `compatibility-pack-qualification`. And a replay
whose choice tape replays exactly while a stream digest diverges was rejected as invalid, so the
child `gomad qualify` exited 3 and the set counted an infrastructure error; it is now classified
`replay_divergence` and no longer projected as exact seed replay. The frontend probe's darwin/arm64
expectation changed from `qualified` to `intermittent` on the observed darwin runs: seed 17
qualified with exact choice replay in one run, and both seeds were `nondeterministic` (a
`choice_profile` or stderr divergence) in others. The darwin CI assertion now also pins the
platform, `unsupported == 11`, and `supported + failed == 7`.

**Constraints.**

- The patch applies with zero fuzz. A hunk that needs fuzz is rewritten.
- The boundary diff between go1.26.4 and go1.27.1 is approved by digest through
  `GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256`, never waved through.
- Keep the go1.26.4 descriptor in git history only. Two live pins double every later milestone.

**Acceptance.**

- `tools/gomad3/.toolchain/bin/go version` reports `go1.27.1`.
- `make -C tools/gomad3 validate-toolchain` passes.
- The upgrade dossier reports every gate passed and the boundary diff approved or empty.
- Milestone F1's acceptance criteria still hold on the new toolchain.
- `make gomad3-qualification` reproduces the current Temporal corpus result of 5 supported and
  11 unsupported with the same blocker paths.

## F3: qualify the existing functional probe

**Spec.** `fn-97-gomad-f3-qualify-the-frontend`

**Outcome.** `TestFrontendSystemInfo` carries a checked determinism claim. This is the first
functional test in the corpus and the first proof that the one-box cluster boots, serves an RPC,
and shuts down under virtual time repeatably.

**Details.**

- Run `gomad qualify --seed 11 --repeat 2 --choices --replay-successes` and the same for seed
  17 against `go-test ./tests/gomadfunctional -- -test.run '^TestFrontendSystemInfo$'` with tags
  `disable_grpc_modules,test_dep`.
- Add the probe to `temporal.json` as a tier 3 suite with `capability_mode` set to whatever mode
  actually qualifies it. If that is `linked` or `guarded`, the manifest says so and the report's
  supported count is labelled experimental. COMPAT-6 is not complete until a real workload
  qualifies without policy widening; this probe is that workload.
- Record transcript and choice-tape utilization for the run. These two numbers size milestone F5.

**Status.** Started on 2026-09-27 on linux/amd64. With the adapters re-pinned, `gomad analyze`
of the probe (`go-test ./tests/gomadfunctional -- -test.run '^TestFrontendSystemInfo$'`, tags
`disable_grpc_modules,test_dep`) runs to completion over 1666 packages and classifies the probe
`unsupported` in all three capability modes; linked and guarded mode first needed the
live-capability extractor to read ELF executables (it parsed Mach-O only). The blockers are the
linux/amd64 counterparts of what `temporal-functional-compute-darwin-arm64` admits on darwin,
and every compatibility pack is scoped to `darwin/arm64`: `add_exact_pack` for the amd64
assembly in xxhash (both modules), snappy, klauspost/compress (zstd, huff0, cpuinfo, xxhash),
murmur3, go-farm, edwards25519, chacha20poly1305, poly1305, x/sys/cpu, and reflect2, plus the
linknames in x/sys/unix (`auxv.go`, `syscall_linux.go`, `vgetrandom_linux.go`), reflect2,
bigfft, and modernc/memory (`mmap_unix.go`); `add_adapter` for modernc.org/libc, whose adapter
rewrites the darwin sources only (`abi0_linux_amd64.s`, `tls_linux_amd64.s`); and, in linked
mode, `model_operation` for nine denied boundaries (DNS lookups, interface addresses, raw
connections, UDP resolution, `process.kill`, symlinks, raw descriptors). Closure mode also
reports `os/exec`, `os/signal`, and `os/user` as `remain_unsupported`. Qualifying the probe on
Linux therefore needs a `temporal-functional-compute-linux-amd64` pack, a Linux modernc libc
adapter, and either the denied-boundary models or guarded mode; on darwin/arm64 the milestone
proceeds as written. The Linux host tier also carries a pre-existing failure unrelated to this
work: the simulated-target choice-trace tests in `runner/internal/execution` report "choice
trace unterminated" on Linux with the go1.26.4 toolchain too, while the real-target choice
paths (runtime tier, core qualification) replay exactly.

Continued on 2026-09-27 on linux/amd64, still on `gomad-linux`. Guarded mode is the mode that
reaches the probe: closure mode keeps `os/exec`, `os/signal`, and `os/user` as
`remain_unsupported` through the cloud credential chains, fx, and the SDK, and linked mode
adds nine denied boundaries. `temporal-functional-compute-linux-amd64` (request, review, and
generated pack under `internal/compatibilitypack`) admits exactly the 25 guarded-mode blockers:
the amd64 assembly of edwards25519, cespare/xxhash, go-farm, snappy, klauspost huff0/cpuinfo/zstd
and its xxhash, reflect2, murmur3, chacha20poly1305, poly1305, and x/sys/cpu, plus the reflect2
and x/sys/cpu linknames; it is the Linux amendment of the darwin pack, whose four entries are the
arm64 subset. Its Makefile qualification passes and `compatibility-pack check` is current. Making
the probe run exposed four defects, all fixed here: the x/net adapter flipped `empty.s` to
`//go:build !darwin`, which compiled that assembly on Linux and made every Linux closure that
imports x/net sockets unsupported (now `//go:build ignore`); gRPC's Linux-only channelz socket
introspection, TCP user-timeout and CPU-time helpers, and non-blocking ready reader call
`(*TCPConn).SyscallConn` and `x/sys/unix`, so a guarded target died with
`GOMAD_CAPABILITY_DENIED` in `grpc.Server.Serve` (the gRPC adapter now compiles gRPC's own
non-Linux implementations of those three files under a Linux constraint, exactly the path
darwin takes); the capability review rejected any closure that compiled some packages of an
adapter's module but not the adapter's prepared package ("adapter prepared package is absent"),
which is every Temporal package that reaches x/net/http2 through gRPC without tchannel's
`x/net/ipv4`, so ten of the sixteen representative suites could not even be analyzed on Linux
(an adapter copy is byte-identical outside its prepared package, so that case is an inert
adapter and is now accepted and still recorded); the qualification-set schema refused tier 3;
and the set collapsed every non-qualified seed of a replaying workload into `runner_failure`
because it demanded successful-replay evidence before looking at the classification, so a
manifest could not state the `nondeterministic` expectation its own schema defines (successful
replay and exact choice replay are now what `qualified` requires, and other seeds keep their
classification and retained evidence). The `target` and `compatibilitypack` unit tests that
assumed a darwin/arm64 host now select for each pack's governed platform and derive their
adapter digests, so both packages pass on Linux.
With those in place `make gomad3-qualification` on Linux analyzes all 17 suites, qualifies the
five package suites with exact replay, reports the eleven expected `unsupported_target` results
(`temporal-cache-concurrent` carries a linux/amd64 expectation naming `xxhash_amd64.s`), and
the probe itself boots the one-box cluster, serves `GetSystemInfo`, and exits 0 under guarded
mode with a 183 MB retained success artifact (the suite's `success_bytes_limit` is 256 MiB).

What does not hold yet is the acceptance's repeatability: two same-seed repetitions of the probe
diverge in the choice trace, and replaying a recorded success diverges too, so the set reports
the probe `nondeterministic` and the manifest states that as its linux/amd64 expectation until
it is fixed. Evidence so far, all on seed 11: the first divergent decision is a run-queue pick
between ordinal 4 and ordinal 121 depending on the pair, always with the same alternative set
but a different physical order or a different selected rank, and in one pair a different
alternative count at ordinal 12; the failing repetition then blocks with no timer nearer than
the test's 90 s deadline and fails on `context deadline exceeded` while the other serves the
RPC. `GOGC=off` does not change it, so it is not the GC dimension; `setarch -R` does not change
it, and the heap-derived membership address is identical across runs, so it is not address
layout; the kernel honors Go's arena hints here. Re-seeding the runtime's per-M `rand` and
`cheaprand` streams did not remove it either, but that change stays because it closes a real
hand-off race: every M started from the same seeded position and advanced independently, so
which M held the P decided map hash seeds, timer tie-breaks (`t.rand`), and semaphore tickets;
the M holding the P now draws from process-wide seeded states (`runtime.gomadRuntimeRand`,
`gomadRuntimeCheapRand`) and Ms without a P keep their own streams for lock backoff. The
divergence at ordinal 4, during runtime initialization with three runnable goroutines in a
different run-queue order, pointed at the scheduler paths the choice trace does not govern.

Instrumenting those paths (run-queue picks, goroutine creation, readies, syscall exits,
M creation, GC starts, per-process event logs for record and replay) found eight host-timing
channels in the patched runtime, each confirmed by an event diff between two same-seed runs and
each closed in `go1.27.1.patch` and `overlay/src/runtime/gomad.go`:

- A goroutine whose runner syscall returned while another goroutine held the P was put on the
  global run queue, which `findRunnable` drained every 61 schedticks ahead of the local queue
  with no recorded choice. Under Gomad the global queue is now admitted only when the local
  queue is empty and through the recorded run-queue choice (`gomadAdmit`), and such syscall
  returns go to a dedicated arrival queue that is admitted one goroutine per idle window, after
  timers, so neither the moment nor the grouping of returns changes the decision stream.
- A syscall returning to an idle P resumed its goroutine directly, skipping `schedule` and the
  timers already due, while the same return to a busy P ran those timers first. Both paths
  now go through `findRunnable`; a return that finds a quiescence round-trip in flight waits
  for its answer instead of taking the P, and a non-advance answer starts the P for it.
- `sysmon` retook Ps and requested preemption after wall-clock intervals and injected the
  forced-GC goroutine on a wall-clock cadence; both are disabled under Gomad.
- Background mark workers ran whenever the P was free, so the amount of marking done while a
  goroutine waited on the runner, and therefore where the collection completed in the program,
  followed host time. Workers now take a slot only when runnable goroutines or parked assists
  exist and idle marking is off.
- Ms were created on demand at hand-offs, allocating `m` structures and g0 stacks at host-timed
  moments and moving every later heap address and collection trigger; eight Ms are reserved
  before user code and user code starts only once all have parked.
- The runtime copied the Gomad control variables into `envs`, which the runner varies between
  recording and replay; the runtime now reads them from argv only.
- Timer ties at one instant were broken with `cheaprand`, whose process-wide stream is also
  drawn on contended runtime lock hand-offs (host-timed, and frequent once Ms are reserved), so
  the runtime tier's `clock_race` fixture diverged 4 ways in 20 same-seed runs; timers now
  draw from their own seeded stream, and heap-profile sampling, which draws from the same
  stream and allocates a bucket per sample, is off under Gomad. The fixture is 20/20 identical
  again and `make test-runtime` passes.

With all of them in place the probe's two fresh same-seed repetitions produce identical choice
traces in most runs (runs 26, 27, 29, 30 of the day; runs 25, 28, 31 still differed at
ordinals 56/119/772 with one alternative more or fewer), and replaying a retained execution
still diverges at the same ordinals. Per-package `inittrace` allocation counts are identical
between recording and replay, yet the first collection triggers one span earlier under replay,
so the remaining channel is in how the two runs' allocations fall on span boundaries rather
than in what they allocate. One such channel was the generated wire digest, which copied its
whole input into a fresh buffer: a replay hashes the recorded I/O transcript where a recording
hashes an empty one, so the two heaps differed by the transcript's size before user code ran.
The generated codecs now hash in place (`sum256` streams 64-byte blocks and pads in a fixed
buffer), which moved the replay's first divergence from ordinals 56/772 to a select in
`goro.(*AdaptivePool).Do` at ordinal 117 whose case readiness differs, so the replay's
program state still departs from the recording's before that point; the runner-side model
responses (their order across the two transport readers, or their simulation time relative to
the quiescence protocol) are the open suspect. The manifest
therefore states `unrepeatable` (a new set expectation accepting `nondeterministic` or
`replay_divergence`) for linux/amd64 so the corpus report records whichever the run produced,
`make test-runtime`, `make validate`, and the core set (5/5 qualified, exact replay) pass on the
new toolchain, and F3's acceptance remains open on Linux; darwin/arm64 has not been run.

Qualified on 2026-09-27 on darwin/arm64. The probe first diverged there at choice ordinal 4,
and the channel was darwin PIE ASLR rather than an output path: the darwin/arm64 linker
produces only position-independent executables and the kernel slides every image, so type
descriptors, globals, and functions had different addresses in every process. reflect2's type
cache and reflect's `sync.Map` lookup caches hash those addresses and allocated a different
number of hash-trie nodes during package initialization, so the heap layout, the first
collection, and the run-queue order followed the slide; the divergences first observed there
at ordinals 1778 and 2247 were downstream of it. The fix (cf1bc982c2, with the environment
sizing fix in 85c71462b8) re-executes an activated darwin target once through `posix_spawn`
with `POSIX_SPAWN_SETEXEC` and the ASLR-disabling attribute before runtime initialization
continues, keeping the pid, descriptors, and process group; a kernel that ignores the attribute
fails the target closed, and linux/amd64 targets are untouched. The `static_addresses` fixture
in the runtime repeatability tier fails on a sliding image. With it the probe qualifies in
guarded mode (tags `disable_grpc_modules,test_dep`) on seeds 11 and 17: both repetitions of
each seed produce identical evidence, and every retained success replays with `replay_match`
and `choice_replay_exact`. Seed 11 used 249 transcript records (31872 bytes) and recorded 3350
choice decisions in 4627 records (444192 tape bytes of the 8 MiB tape); seed 17 used 253
records (32384 bytes) and recorded 3301 decisions in 4572 records (438912 tape bytes).
`temporal.json` now expects the probe `qualified` on darwin/arm64 with capability mode
`guarded`, and `make gomad3-qualification` on darwin reports 6 supported, 11 unsupported, 1
failed (`user-timers-workflow`, still `intermittent` on darwin), and 0 infrastructure errors;
the darwin CI assertion accepts 7/0 or 6/1 supported/failed and requires the probe `qualified`
(403b1d36ec). Linux was not re-measured on this host. The ASLR channel is darwin-specific:
the README states that linux/amd64 targets are not position independent, and `setarch -R`
already left the Linux divergence unchanged, so the linux/amd64 expectation stays
`unrepeatable` until the probe is measured there.

**Constraints.**

- No new pack unless the analyzer names a blocker not already in
  `temporal-functional-compute-darwin-arm64`. If it does, the pack request is amended and
  re-reviewed, not appended silently.
- The test itself stays 16 lines. A probe that needs a Gomad-specific harness is not a probe.

**Acceptance.**

- Both seeds produce identical canonical evidence across two repetitions and replay with
  `replay_match` and `choice_replay_exact`.
- `temporal.json` lists the probe and `make gomad3-qualification` reports it supported.
- The CI workflow's Temporal assertion is updated to the new supported count and passes on
  dispatch.
- The report records transcript bytes used and choice decisions recorded for the probe.

## F4: close the capability closure for `./tests`

**Spec.** `fn-98-gomad-f4-close-the-tests-capability`

**Outcome.** `gomad analyze --capability-mode=closure go-test ./tests` returns zero
uncovered findings, so any test in the package can at least be prepared. This is the milestone
that turns "one probe" into "any test".

**Details.**

- Start from the 27-package inventory in the baseline. Classify each as one of three cases.
  - **Eliminated by the linker.** Cloud credential chains (`cloud.google.com/go/auth`,
    `aws-sdk-go-v2/credentials/processcreds`, `k8s.io/client-go` auth exec, `oauth2/google`),
    `gnostic-models`, `tchannel-go`, `thrift`, `mysql`, `pgx`, `pq`, and `auto-scaled-workers`
    are reachable only through providers no functional test instantiates. Verify each with
    linked mode. Where the linker keeps one because of an `init` function or a registered
    plugin, isolate the provider behind a build tag or a lazy registration in server code.
  - **Live and needs a pack.** `prometheus/client_golang` reads process metrics through
    `x/sys/unix`; `logrus`, `x/term`, and `go-isatty` probe terminals. Each gets an exact pack
    that admits the specific facts, matching the shape of `modernc-libc-xsys-v047`.
  - **Live and needs a seam.** `go.uber.org/fx` and `temporal/interrupt.go` install signal
    handlers, and `common/config` runs a password command. The cluster in `testcore` needs an
    option that skips signal installation and the password command, and `fx` needs its
    shutdowner wired without `os/signal`. These are small server-side changes with a stock-Go
    default.
- ~~Remove `tools/umpire3` from the `./tests` closure.~~ Done by fn-81, which deleted both the
  tree and the monitor seam it arrived through.
- Re-run the analysis with the cloud archiver, Elasticsearch, Cassandra, and SDK worker
  providers excluded by tag to measure how much of the closure is optional. Do not make that
  separation a prerequisite if linked mode already closes the set.

**Status.** Applied on 2026-09-27 on `gomad-linux` for linux/amd64. `gomad analyze
--capability-mode=closure --format=json --build-tag disable_grpc_modules --build-tag gomad
--build-tag test_dep go-test ./tests` reports `supported` with zero blockers over 1022 packages,
selecting `modernc-libc-xsys-v047-linux-amd64` and the new
`temporal-functional-tests-linux-amd64` pack. The 54 blockers of the untagged closure fell in
four steps, each measured with the same command:

- **Server seams behind the `gomad` build tag (54 → 18).** `temporal/interrupt.go` (no signal
  handler), `common/config` (the persistence password command is refused), the SQL errno checks
  in `sqlplugin`, the Google Cloud and S3 archivers behind `common/archiver/provider`, AWS
  request signing for Elasticsearch, ringpop membership (`temporal` serves only clusters that
  supply `StaticServiceHosts`), the auto-scaled-workers component in `service/worker`, and the
  MySQL and PostgreSQL drivers in `tests/testcore`. SQL plugin names moved to a driver-free home
  in `sqlplugin` so `visibility` and `persistence-tests` compare names without linking drivers.
  Three functional tests that import a compute provider or the MySQL driver directly are
  excluded under the tag; shared worker-deployment helpers moved to an untagged file. The
  default build is unchanged and `make lint-code-fast` is clean.
- **Adapters for what a seam cannot reach (18 → 22, then 3).** `os/signal` in fx and the Go
  SDK and `os/user` in `otel/sdk/resource` are `remain_unsupported`, so no pack may admit them,
  and they live in third-party modules a build tag cannot touch. The milestone text assumed a
  server-side fix; it needed three new exact adapters (`go.uber.org/fx@v1.24.0`,
  `go.temporal.io/sdk@v1.48.0`, `go.opentelemetry.io/otel/sdk@v1.44.0`) and an extension of
  the gRPC adapter for the three portable files that import `syscall`. Each rewrite is anchored
  to exact file digests. Removing the MySQL driver also removed `filippo.io/edwards25519` from
  the closure, which deactivated `temporal-functional-compute-linux-amd64` (its activation
  requires every listed module) and surfaced the 19 assembly and linkname facts it had been
  admitting; those and the three `procfs` facts are the `add_exact_pack` residue.
- **One pack for the `./tests` closure (3 + 19 → 0).** `temporal-functional-tests-linux-amd64`
  names the `functional-tests` workload, targets `./tests` under the `gomad` tag, and admits
  the amd64 assembly of xxhash, go-farm, snappy, klauspost/compress, and murmur3, the reflect2
  assembly and linknames, and `import:syscall` and `import:golang.org/x/sys/unix` in
  `prometheus/procfs`. Discover, review, generate, check, and qualify all pass;
  `compatibility-pack-qualification` qualifies all three linux requests.
- **Profile identity.** New adapters change the deterministic I/O inventory, so the profile
  implementation digest moved and `modernc-libc-xsys-v047-linux-amd64` was regenerated
  through the same review flow. The darwin/arm64 packs could not be regenerated from a linux
  host, because discovery reviews the host platform; on 2026-09-27 F1's darwin run regenerated
  the libc packs and F2's refreshed the compute pack. The darwin prepared source-set pins of
  the three new adapters were computed with `GOOS=darwin GOARCH=arm64 go list` over the
  rewritten modules; F2's darwin run analyzed the closure with them unchanged. On 2026-09-27
  F4's darwin/arm64 run observed them: `TestRewrittenModulePreparedPackageSourceSetIdentity`
  reviews each prepared package through the server module graph, and with each darwin pin
  zeroed in turn the review reported `d8b65806…` (fx), `45cd8411…` (SDK `internal`), and
  `796855ab…` (otel `resource`), equal to the committed pins, so none changed. On the same
  host `compatibility-pack check` reports every pack current and
  `compatibility-pack-qualification` qualifies all six requests of the darwin set.

The remaining acceptance items, measured on linux/amd64:

- **Server build and stock suite.** `go build ./...` passes with and without the tag, the unit
  tests of every touched package pass, `make lint-code-fast` is clean, and
  `go test -tags test_dep -run '^TestActivityAPIBatchCancelClientTestSuite$' ./tests` passes
  natively with the seams at their defaults.
- **The eleven unsupported leaf cases.** With `--build-tag gomad`, none of them carries a
  forbidden import any more. `./common/persistence/tests`, `./common/cache`,
  `./common/persistence`, `./common/persistence/sql/sqlplugin/sqlite`, `./service/matching`,
  `./service/history/workflow`, and `./service/history/workflow/update` report only
  `add_exact_pack` facts: the amd64 assembly of xxhash (first blocker for all but
  `persistence/tests`, whose first is `filippo.io/edwards25519/field`), go-farm, snappy,
  klauspost/compress, murmur3, and `x/sys/unix`, plus the procfs imports. They do not flip to
  `qualified` because `temporal-functional-tests-linux-amd64` activates only when every module
  it names is in the closure, and these smaller closures lack reflect2 or procfs; each closure
  shape needs its own exact pack, which F5 and F6 add for the workloads they qualify. The
  manifest keeps the eleven untagged so their expectations stay observed on both platforms;
  tagging them would require darwin/arm64 first-blocker evidence this host cannot produce.
- **The `./tests` suite under the tag.** `gomad qualify --capability-mode closure` of
  `TestActivityAPIBatchCancelClientTestSuite` with the tag prepares and runs: the closure is
  supported, one repetition passes the whole suite in 4.5 s of virtual time (1.5 s wall,
  87k choice records, 8.4 MiB choice tape), the other times out a test at the 90 s virtual
  deadline, so the run classifies `nondeterministic`. The suite's `choice_bytes` of 8 MiB
  overflows (`choice_trace_overflow`); 64 MiB is enough. That is the F5 starting point, and the
  F3 repeatability gap is the same one. The manifest entry is unchanged; the linux CI job
  asserts the closed closure directly instead.

With the new adapters in place the core set still qualifies 5/5 with expectations met, and the
frontend probe under guarded mode still lands inside its `unrepeatable` expectation
(`replay_divergence` on this run).

On darwin/arm64 the `./tests` closure is closed at commit `a8777f5d73` (F2's darwin run): with
the `gomad` tag, `gomad analyze --capability-mode=closure go-test ./tests` reports `supported`
with zero blockers. The new `temporal-functional-tests-darwin-arm64` pack admits the
Prometheus client's darwin process-collector `syscall` and `x/sys/unix` imports, and both darwin
packs pass `compatibility-pack-qualification`. F4's darwin/arm64 run re-observed it at commit
`a861e0dbb3`: the same command with `--format=json` reports `supported` with zero blockers and
no `unsupported_target` findings over 1043 packages, selecting `modernc-libc-xsys-v047`,
`modernc-libc-xsys-v047-isatty-v021`, `reflect2-go126`,
`temporal-functional-compute-darwin-arm64`, and `temporal-functional-tests-darwin-arm64`.
`temporal.json` names no packs: a pack binds to the workload its request names (both
functional-tests packs name `functional-tests`), and the qualification set for packs is the
per-platform `COMPATIBILITY_PACK_QUALIFICATIONS` list in `tools/gomad3/Makefile`, where the
darwin tests pack sits exactly as the linux one does. The darwin `core` CI job now asserts the
closed closure with the same step as the linux job.

F4 was applied on darwin/arm64 on 2026-09-27. Every acceptance item was measured on this host:

- **Packs and pins.** Every pack is current against the profile digest. The fx, SDK, and otel
  darwin prepared source-set pins were observed and are unchanged (see above).
- **No `os/exec` or `os/signal` admission.** The spec completion review found that the
  `modernc-libc-xsys-v041`, `-v047`, and `-v047-linux-amd64` packs admitted `import:os/exec`
  and `import:os/signal` for `modernc.org/libc`. The libc adapter now replaces the bodies of
  `system` and `pause` (darwin), `signal` (darwin `libc_unix.go`), and `signal` and `system`
  (musl) with an unconditional refusal and drops those imports, so the prepared module no
  longer carries either fact on either platform. Pack and request validation now reject any
  pack or request that admits `os/exec`, `os/signal`, or `os/user`. The three darwin libc
  packs (`v041`, `v047`, and `v047-isatty-v021`) were rediscovered, reviewed, and regenerated
  on darwin/arm64, and the new darwin prepared source-set pin was observed there. Discovery
  reviews only the host platform, so the linux pin and the `-v047-linux-amd64` request were
  derived, not observed. The pin is the source-set digest of the linux/amd64 `go list` file
  set of the rewritten module; the same derivation reproduces the observed darwin pin exactly.
  The request evidence carries that pin, the new `libc_musl.go` digest, and the
  host-independent replacement inventory, without the two facts. A linux/amd64
  `compatibility-pack-qualification` must still confirm them, and a wrong value fails closed
  as a source-set mismatch. After the change, the `./tests` closure still reports
  `supported` with zero blockers, the core set qualifies 5/5, and `make gomad3-qualification`
  meets its expectations as recorded below.
- **The `./tests` closure.** It reports `supported` with zero blockers over 1043 packages,
  selecting the five packs listed above.
- **The eleven leaf cases.** None of them carries a forbidden import on darwin. Ten are
  `qualified` with exact replay: the sqlite persistence boundary, the cache, dither, poller,
  queue-key, sqlite schema rewrite, transition-history, update-abort, version-set merge, and
  workflow-backoff cases. All but the cache case run with the `gomad` tag. The eleventh,
  activity batch cancel, is now a tier-3 `./tests` suite. It runs to success, but its
  same-seed evidence is `intermittent` (`replay_divergence` on the latest run). Two new exact
  darwin packs make this possible. `temporal-leaf-xsys-darwin-arm64` admits the
  `golang.org/x/sys/unix` facts that the Prometheus client reaches.
  `temporal-leaf-xxhash-darwin-arm64` admits the arm64 assembly of xxhash and
  klauspost/compress. `!gomad` build-tag seams keep the Cassandra, MySQL, and PostgreSQL
  files out of the gomad build of `common/persistence/tests`. On linux/amd64,
  `platform_expectations` now expects the ten leaf cases to stop at
  `foreign:assembly:xxhash_amd64.s`. That is an `add_exact_pack` fact the darwin-scoped packs
  do not admit, not a forbidden import.
- **The corpus.** `make gomad3-qualification` meets its expectations with 16 supported,
  0 unsupported, 2 failed (both intermittent tier 3), and 0 infrastructure errors, 18/18
  completed. Commit `5285983bb9` fixed set-report validation for a seed whose repetitions
  recorded different choice tapes. Such a seed now withholds choice-replay exactness; before
  the fix, it failed the run as unusable.
- **Server build and stock suite.** `go build ./...` passes with and without
  `-tags gomad,test_dep,disable_grpc_modules`. The unit tests of the seam packages
  (`temporal`, `common/config`, `sqlplugin`, `common/archiver/provider`, the Elasticsearch
  client, `service/worker`, `tests/testcore`, and `service/history/workflow/update`) pass
  under `test_dep`. The 82 SQLite tests of `common/persistence/tests` pass with and without
  the tag. Under the `gomad` tag, five tests fail as the seams intend:
  `TestNewServer`, `TestNewServerWithOTEL`, and `TestNewServerWithJSONEncoding` configure
  ringpop membership, and the two `TestSQLResolvePassword_Command*` tests run the password
  command that the tag refuses. `make lint-code-fast` cannot run on this branch as written.
  It lints every package changed since the merge base, and that set includes the nested
  `tools/gomad3` and `tests/mixedbrain` modules and the `integration`-only
  `tools/gomad3integration`, all of which fail to typecheck in the main module.
  `make lint-code` over the seam packages reports 0 issues, and so does its `go vet
  -vettool=errortype` step once the stale go1.26 `.bin/errortype` is rebuilt with go1.27.
  `go test -tags test_dep -run '^TestActivityAPIBatchCancelClientTestSuite$' ./tests` passes
  natively with the seams at their defaults.

**Constraints.**

- Closure mode is the support claim. Linked mode is evidence for what to isolate, never the
  final answer, because reflection and interface dispatch keep eliminated code alive across Go
  releases.
- A pack request names the workload it unlocks. A pack with no workload is refused.
- No pack admits `os/exec`. The password command and any `exec` reachable from `temporal` are
  removed from the test build, never allowed.

**Acceptance.**

- `gomad analyze --capability-mode=closure --format=json go-test ./tests` reports zero
  `unsupported_target` findings on `darwin/arm64`.
- Every pack added is listed in `temporal.json` and passes
  `make -C tools/gomad3 compatibility-pack-qualification`.
- The server builds and the stock functional suite passes with the new seams at their defaults.
- The eleven currently unsupported leaf cases in `temporal.json` flip to `qualified` or carry a
  blocker that is not a forbidden import.

## F5: one workflow-executing functional test, deterministic

**Spec.** `fn-99-gomad-f5-one-workflow-executing`

**Outcome.** A test that starts a workflow, completes a workflow task through the task poller,
fires at least one timer, and reads history back qualifies with exact replay. This exercises
frontend, history, matching, SQLite writes, inter-service gRPC, and virtual time together.

**Details.**

- Pick the smallest existing suite that does all of the above without activities, signals, or
  Nexus. Do not write a new test; the point is an unchanged Temporal test.
- Raise or parameterize the transcript bound. A cluster doing SQLite plus gRPC will plausibly
  exceed 524k modeled operations. The bound moves into the profile with a runner flag and an
  artifact-identity field, and a run that overflows reports the count it reached.
- Audit the run for fail-closed denials and busy loops. `signal.Notify`, `os.Pipe`, symlinks,
  `File.Fd`, UDP, IPv6, DNS beyond `localhost`, and `runtime.GOMAXPROCS(n>1)` terminate the
  process. A goroutine that polls without blocking freezes virtual time until the wall watchdog
  kills the run, because preemption is off. Each denial found becomes either a modeled
  operation with its own contract or a recorded blocker.
- Measure `runtime.AddCleanup` and finalizer activity. Dynamic config uses cleanups on cached
  constrained values. If cleanup timing changes evidence between repetitions, the milestone
  records that as a GC-dimension divergence and opens the deterministic-GC research item in
  [GOMAD3_NEXT.md](GOMAD3_NEXT.md) rather than masking it.

**Constraints.**

- Exact choice-tape replay is not required. The tape caps at 64 MiB and a full cluster run
  exceeds it. Seed-level repeatability plus exact I/O transcript replay is the bar.
- The wall watchdog is a safety net. A test that only passes with a watchdog longer than its
  stock `go test` time is a finding, not a pass.
- The test passes under stock Go with the same source before and after.

**Acceptance.**

- `gomad qualify --repeat 4` on two seeds produces identical canonical evidence per seed and
  replays with `replay_match`.
- The report records transcript bytes, transcript records, choice decisions, virtual time
  elapsed, wall time elapsed, and peak goroutine count.
- Zero watchdog terminations and zero `GOMAD_CAPABILITY_DENIED` throws across all repetitions.
- The suite is added to `temporal.json` as tier 3 and the CI assertion is updated.

**Status.** Worked on 2026-09-27 on linux/amd64 with `TestUserTimersTestSuite` (`./tests`, tags
`disable_grpc_modules,gomad,test_dep`, stock `go test` 4.2 s), which starts a workflow, completes
workflow tasks through the poller, fires user timers, and reads history back. Getting the
one-box cluster to run at all under the deterministic profile took three modeled operations,
each with its contract, bound, transcript coverage, replay, and negative test: the in-memory
filesystem now starts with the process temp directory (`/tmp`), so SQLite's temp files and the
test's temp paths resolve; shared writable file mappings model SQLite's WAL index (`-shm`),
aliasing identical mappings, rejecting overlapping writable regions, read-only and volume
files, and more than 64 MiB mapped; and read-only mount lookups (the schema directory, mounted
with `--io-ro-mount schema=/go.temporal.io/server/schema`) are plain host syscalls rather than
simulation transport, so quiescence keeps virtual time still while a lookup is pending instead
of advancing 90 s and failing the boot. The evidence report gained the fields this milestone
asks for: `peak_goroutines` (from the choice terminal frame, sampled at every goroutine
creation), `virtual_time_elapsed_nanos` (from the simulation-time arbiter), and per-execution
`wall_elapsed_nanos` in the qualification report.

Measured with `gomad qualify --repeat 8` on the rebuilt toolchain
(`d6d1f5c59112560769f54430006180ab6ccba596ba8b99b109285cd55394f933`, the measured build differing from it only by a runtime comment), 64 MiB choice tape,
success artifacts retained and replayed: every repetition exits 0 with no watchdog termination
and no `GOMAD_CAPABILITY_DENIED`; 15 I/O transcript records (the mount lookups; SQLite lives in
the in-memory filesystem); 5761 choice decisions in 8143 records (782 KB tape) on seed 11, 5686
in 8022 on seed 17; virtual time elapsed 4.008 s; wall time 0.27 s to 0.39 s per repetition
(cluster boot included, versus 4.2 s under stock Go); peak 664 goroutines; stderr 52 KB and
identical across every repetition of both seeds. Seed 11 is `qualified`: eight identical
evidence digests, each replaying `exact`. Seed 17 is `nondeterministic`: three digests in eight
repetitions, differing only in the choice trace (`first_divergence: choices`), with the divergent
repetitions also failing their own replay at ordinal 2102, the run-queue decision where
`runtime.runCleanups` becomes runnable after the collector's `sweepdone`.

Finding that cause closed four host-timing channels in the runtime, each confirmed by an event
diff between same-seed runs (allocation traces, seeded draw counters, span refills, and mark
statistics printed from a debug build): the type-assertion and interface-switch caches decide
whether to allocate a new cache from `cheaprand`, and that process-wide seeded stream was also
consumed by lock hand-off anti-starvation draws (`unlock2Wake`), work-steal order (`stealWork`),
and pcvalue-cache eviction during stack walks (`pcvalue`), all of which happen at host-timed
moments; those three now draw from the M's own stream (`gomadHostCheapRand`, patched in
`lock_spinbit.go`, `proc.go`, `symtab.go`). A GC stack scan (any `suspendG`) now waits for a
goroutine inside a plain host syscall to return and queue itself as an arrival
(`gomadAwaitHostSyscallExit`, patched in `preempt.go`), so the collector no longer sees either
the frames inside a pipe write or mount lookup or the frames after it depending on the host.
With those in place, same-seed runs have identical `mallocgc` sequences (166,286 allocations
through decision 20), identical seeded draw counts, identical stack-scan bytes, and identical
heap base addresses, yet the fourth collection scans 24 more heap bytes in one run than in the
other, after which span refills, `heapLive`, and the cleanup wake drift. That is the GC-dimension
divergence this milestone anticipated: the channel is in the collector's view of live memory,
not in allocation or scheduling. It is recorded here rather than masked, and
[GOMAD3_NEXT.md](GOMAD3_NEXT.md#milestone-7-evaluate-research-extensions) opens the
deterministic-GC research item with the evidence and candidate designs.

The suite is in `temporal.json` as tier 3 (`user-timers-workflow`) with the new qualification
expectation `intermittent`, which accepts `qualified`, `nondeterministic`, or
`replay_divergence` per seed so the corpus report records whichever the run produced; CI asserts
18 selected and completed workloads, 11 expected boundaries, and either 6 supported and 1 failed
or 5 supported and 2 failed. Acceptance therefore stands as: metrics, denials, watchdogs, and
manifest met; seed-level repeatability met on one of the two seeds with exact replay, open on
the other through the GC dimension. darwin/arm64 has not been run.

Measured on darwin/arm64 on 2026-09-27, where two further channels surfaced. The suite first did
not run past schema load: SQLite's WAL index mapping calls `sysconf(_SC_PAGESIZE)`, and the
darwin modernc libc adapter refused `Xsysconf` as a host capability, so every WAL database
panicked. The adapter now answers `_SC_PAGESIZE` (4096, the model's block and mapping
granularity), `_SC_NPROCESSORS_ONLN` (1), and `_SC_GETPW_R_SIZE_MAX` (128) from the model
(`gomadSysconf`), with the adapter template, the darwin source-set pin, and the four libc packs
re-pinned; the core corpus gained `sqlite-write-ahead-log`, refused before and qualified after.
With the suite running, same-seed runs diverged at the first run-queue decision after the first
collection (ordinal 33 on seed 11, 28 on seed 17), with identical runnable sets in a different
order. A debug toolchain showed the first assist of a cycle draining different mark work, for
two reasons: the write barrier shaded a not-yet-marked `m` struct through `execute`'s
`gp.m = mp`, and which M runs a goroutine after a syscall hand-off is host timing; and an M
parked through `findRunnable` kept its `allpSnapshot`, so the `allp` array was greyed from the
M in one run and from the globals in another. The assist that ends the mark phase stops on a
work boundary, so the boundary moved and the run-queue order followed. The runtime now greys
every `m`, `g0`, `gsignal`, and `g` at mark start in `startTheWorldWithSema`
(`gomadGreyRuntimeStructures`), and `snapshotAllp` no longer parks `allp` in the M under Gomad;
the choice-controller identity and derived wire codecs changed with the overlay, and the core
corpus gained `mount-reads-under-collection`, nondeterministic before and qualified after.

`gomad qualify --repeat 4` then produced identical canonical evidence per seed, every retained
success replaying `exact`, with no watchdog termination and no `GOMAD_CAPABILITY_DENIED`. Seed 11:
17 transcript records (2176 bytes), 5774 choice decisions in 8140 records (781 KB tape), virtual
time 4.008 s, wall time 1.57 s to 3.20 s, peak 663 goroutines. Seed 17: 17 transcript records
(2176 bytes), 5701 decisions in 8008 records (769 KB tape), virtual time 4.008 s, wall time
1.52 s to 1.85 s, peak 664 goroutines. stderr is 52 KB and identical across repetitions.
`TestActivityAPIBatchCancelClientTestSuite` also qualifies on both seeds (58.8k decisions,
4.5 s virtual, peak 1565 goroutines). `temporal.json` now states `qualified` for
`user-timers-workflow` and `activity-batch-cancel-boundary` on darwin/arm64, and
`make gomad3-qualification` on darwin/arm64 meets every expectation with 18 of 18 workloads
supported, 0 failed, and 0 infrastructure errors; the darwin CI assertion requires exactly that.
The linux/amd64 expectations stay `intermittent` as last measured there; the mark-start greying
and the regenerated libc packs change the runtime linux/amd64 targets use too, so linux/amd64
needs a requalification run before its expectation can move. Acceptance is met on darwin/arm64.

## F6: a package-level functional slice

**Spec.** `fn-100-gomad-f6-a-package-level-functional`

**Outcome.** A named slice of at least ten `./tests` suites runs through `qualify-set` and every
suite is either qualified or classified with an exact blocker. This is where "any test"
becomes measurable instead of anecdotal.

**Details.**

- Choose suites that cover activities, signals, queries, updates, child workflows, continue-as-
  new, and cron. Each family pulls a different service path.
- Run the slice with `--repeat 2` on two seeds and collect per-suite transcript and decision
  counts, watchdog terminations, and denials.
- Triage every non-qualified suite into one of four classes and record the class in the
  manifest expectation: capability blocker, unmodeled boundary operation, busy loop or
  watchdog, or evidence divergence with the same inputs. The last class is a Gomad defect and
  takes priority over all others because it falsifies the determinism claim.
- Add per-suite `required_probes` where a suite depends on a modeled operation, so a later
  boundary change that silently drops the operation fails the set.

**Constraints.**

- The suites stay unchanged. `testcore` options may gain flags; test bodies may not.
- A suite that needs a longer `run_timeout` than two minutes of wall time records why. Virtual
  time is free, wall time is a spin or a host escape.

**Acceptance.**

- `make gomad3-qualification` completes with `infrastructure_errors == 0` and
  `failed == 0`; every non-qualified suite has an expectation whose blocker string names a
  package, an operation, or a finding identity.
- No suite is classified as evidence divergence. If one is, the milestone stays open until the
  divergence is explained and fixed in Gomad, or the suite's nondeterminism is shown to be a
  test bug and fixed upstream.
- At least eight of the ten suites qualify.

**Status.** Started on 2026-09-27 on linux/amd64 with ten suites: `TestActivityTestSuite`,
`TestSignalWorkflowTestSuiteChasm`, `TestQueryWorkflowSuite`, `TestWorkflowUpdateSuite`,
`TestChildWorkflowSuite`, `TestContinueAsNewTestSuite`, `TestCronTestSuite`,
`TestWorkflowTestSuite`, `TestCancelWorkflowSuite`, and `TestWorkflowTimerTestSuite`, each run
through `gomad qualify --repeat 2` on seeds 11 and 17 (64 MiB tape, one retained and replayed
success, closure mode under the `gomad` build tag, schema mounted read-only). Every suite runs
the cluster to a successful exit within 0.4 s to 3.4 s of wall time, with 15 transcript
records, no watchdog termination and no `GOMAD_CAPABILITY_DENIED`; decisions range from 8.7k
(timers) to 134k (updates), peak goroutines from 705 to 4.6k.

The first pass, under the Green Tea collector, reproduced almost nothing: 19 of 20 seed runs
were `nondeterministic` and `TestWorkflowUpdateSuite` failed outright. Listing every scanned
object per cycle from a debug toolchain traced the divergence to Green Tea's scan-work
accounting (span batches credited by `objects * elemsize`, sparsely reached objects by pointer
extent), which made the pacer's trigger for the next cycle depend on the order the marker
reached the `m` structs, and that order on which M held the P. Targets now build with
`GOEXPERIMENT=nogreenteagc`; with the classic collector `TestUserTimersTestSuite` reproduces
eight of eight on both seeds, and the slice reproduces fresh repetitions and replays in 6 of 20
seed runs (`qualified`: activity 17, query 17, continue-as-new 11, workflow 11, timer 11) and
fresh repetitions in 6 more whose retained replay diverged. That replay-only channel was then
found in the runtime's environment filtering: it copied every environment entry into a Go
string before dropping the Gomad control variables, whose values differ between a recording and
its replay (choice mode, tape descriptor and size), so the replay's heap differed from the
recording's before user code ran. The filter now runs on the C string
(`gostringnocopy`) and copies only kept entries; the fix is in the overlay and its
requalification is the next step.

The update suite's failure was a server finding, not a Gomad one: the history update registry
sorts pending updates by `admittedTime` over a map, and under virtual time two updates admitted
within one clock tick carry the same time, so `TestUpdatesAreSentToWorkerInOrderOfAdmission`
saw them reordered. Admissions now carry a process-wide sequence number that breaks the tie
(`compareAdmission`), with a unit test, and the suite passes under Gomad on both seeds.
`TestCancelWorkflowSuite` is the one suite whose retained replay reproduces the choice tape
exactly while its evidence still differs; `gomad replay --observed DIR` now retains the
replayed streams so that difference can be diffed. Triage for the six seed runs that still
diverge between fresh repetitions (signal, update, child, cron 17, timer 17) is open; none of
the slice is in `temporal.json` yet, and the acceptance stands as: outcome and evidence bounds
met, evidence divergence still present and under investigation, so the milestone is open.

F5's darwin/arm64 run (2026-09-27) found a further same-seed divergence in the classic collector:
which M structs the first assist of a cycle greyed depended on host timing, fixed by greying the
scheduler structures at mark start. That fix may change the slice's divergence picture above,
which predates it, and the deterministic-GC research item in
[GOMAD3_NEXT.md](GOMAD3_NEXT.md#milestone-7-evaluate-research-extensions) may need its
conclusion updated once the slice is re-measured.

Re-measured on darwin/arm64 on 2026-09-28 (toolchain
`bb4304eb5330c18e658bfc0806f69b0ce1399944229a06f41134cb391b05e01d`). All ten suites qualify on
seeds 11 and 17 with `gomad qualify --repeat 2`: both fresh repetitions of every seed produce
the same evidence, every retained success replays with `replay_match` and exact choice replay,
and every repetition exits 0 with no watchdog termination and no `GOMAD_CAPABILITY_DENIED`.
The five suites that had diverged between fresh repetitions on linux, and the cancel suite, then
ran with `--repeat 4` on both seeds; all 24 fresh repetitions reproduced the `--repeat 2`
evidence and replayed exactly. No suite is classified as evidence divergence on darwin, so
there was nothing left to triage. Per-suite measurements, identical in those runs and in the
full set run below (17 transcript records, 2176 bytes, in every suite; wall time is the
longest single execution in the set run):

| Suite | Decisions (seed 11 / 17) | Choice records (11 / 17) | Virtual time | Peak goroutines | Wall time | Watchdogs | Denials |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `TestActivityTestSuite` | 27907 / 27595 | 42309 / 41842 | 13.6 s / 13.1 s | 1455 | 1.7 s | 0 | 0 |
| `TestCancelWorkflowSuite` | 21777 / 21774 | 30927 / 30973 | 0 s | 1355 | 1.8 s | 0 | 0 |
| `TestChildWorkflowSuite` | 26045 / 25863 | 39014 / 38763 | 13.0 s | 1248 | 1.7 s | 0 | 0 |
| `TestContinueAsNewTestSuite` | 51612 / 52239 | 80173 / 80963 | 36.4 s | 1170 | 2.0 s | 0 | 0 |
| `TestCronTestSuite` | 9901 / 10008 | 14473 / 14651 | 15.0 s | 720 | 1.6 s | 0 | 0 |
| `TestQueryWorkflowSuite` | 25633 / 25581 | 39046 / 38998 | 11.3 s | 1409 | 1.8 s | 0 | 0 |
| `TestSignalWorkflowTestSuiteChasm` | 57986 / 59741 | 86512 / 89312 | 57.0 s | 1944 | 2.2 s | 0 | 0 |
| `TestWorkflowTimerTestSuite` | 8720 / 8596 | 12796 / 12646 | 8.0 s | 707 | 1.6 s | 0 | 0 |
| `TestWorkflowUpdateSuite` | 132626 / 132540 | 206774 / 206561 | 22.5 s / 22.4 s | 4588 | 3.0 s | 0 | 0 |
| `TestWorkflowTestSuite` | 56053 / 55687 | 86253 / 85681 | 13.1 s / 13.4 s | 2135 | 2.1 s | 0 | 0 |

The cancel suite's 0 s of virtual time is real: every log timestamp is the simulation epoch and
all four cancel tests appear in its stderr. The query suite's stdout is only `PASS` because it
logs to stderr, which names each query test. No suite needs more than the two-minute
`run_timeout`; the 20-minute overall timeout covers the cold `./tests` build.

The linux divergences are explained, as an inference, by channels fixed after the linux
measurement, all platform-neutral: the Green Tea scan-work accounting (targets build with
`GOEXPERIMENT=nogreenteagc`), the environment filter that copied Gomad control variables into Go
strings (the replay-only differences, including cancel's), mark-start greying and the M's
`allp` snapshot (the fresh-repetition divergences in signal, update, child, cron 17, and timer
17), and the update registry's admission order (`compareAdmission`, the update suite's test
failure). The darwin-only ASLR re-execution cannot account for them. linux/amd64 has not been
re-measured, so that inference is unconfirmed there.

The ten suites are in `temporal.json` as tier 3 `functional-*` workloads, `qualified` on
darwin/arm64. Each requires the modeled probes every measured run observed
(`stdlib.net.interfaces`, `stdlib.os.newfile`, `stdlib.os.openfile`), so a boundary change
that drops one fails the set. `stdlib.os.getwd` was required at first and dropped on 2026-09-28:
only darwin's `os` package init calls `Getwd`, so every linux run failed the suites with
`semantic_coverage_failure`. Their linux/amd64 expectation is `intermittent`
with `finding: GOMAD_MILESTONES.md#f6-a-package-level-functional-slice`: expectations gained a
`finding` field, which `qualify-set` requires on every non-qualified, non-`unsupported_target`
expectation and rejects on the others, and the existing F3 and F5 expectations name their
sections the same way.

A full `make gomad3-qualification` retains about 11 GiB of Campaigns, more than the free disk
of the measuring host and of a CI macOS runner, and an earlier full run was stopped by its disk
guard at 22 of 28 workloads. `qualify-set --prune-qualified-artifacts`
(`GOMAD3_QUALIFICATION_PRUNE=1`) now deletes a qualified seed's Campaigns once its successful
repetitions were replayed exactly and the set report holds its evidence, keeping the seed's
qualification report; the set report records `qualified_artifacts_pruned` and a per-seed
`artifacts_pruned`, and a report that marks any other seed pruned is rejected. With it, one
uninterrupted `make gomad3-qualification GOMAD3_QUALIFICATION_PRUNE=1` at manifest
`sha256:e10842d4` took 31 minutes and peaked at 340 MiB of retained artifacts: expectations met,
28 of 28 selected, completed, and supported, 0 failed, 0 infrastructure errors, 56 of 56 seeds
replayed with none diverged, and the darwin CI assertion true. Every one of the 56 seed reports
exits 0 with identical evidence across both repetitions. Acceptance is met on darwin/arm64: ten
of ten suites qualify, none is classified as evidence divergence, and every non-qualified
expectation names a finding.

## F7: any functional test, and CI

**Spec.** `fn-101-gomad-f7-any-functional-test-and-ci`

**Outcome.** The whole `./tests` package is enumerated in the qualification set, every test has
a disposition, and the unsupported count is zero on `darwin/arm64`. A Linux bundle lets the
gate run in CI.

**Details.**

- Generate the manifest from `go test -list` output instead of curating it by hand, so a new
  test lands with a default expectation of `qualified` and fails the set if it is not.
- Split the set into shards with `gomad plan` and `execute-shard` so a full run fits the
  90-minute CI budget. Merge with `gomad merge`. **Changed 2026-09-28/29:** only a smoke test on a
  named selection of functional tests is run, in CI and locally; the full `./tests` set is not run
  as a gate because it does not scale. The generated manifest still gives every test a
  disposition, and a change is validated on the smoke selection plus the suites it affects.
- ~~Build the `linux/amd64` platform bundle per COMPAT-7 with its own boundary manifest,
  adapters, and publication primitives.~~ Done ahead of order on 2026-09-26: the toolchain,
  compiler, linker, and deterministic I/O profile accept `linux/amd64`, and the `core-linux` CI
  job builds the toolchain and runs the conformance tiers and the core corpus on Linux.
  Artifacts replay only on the platform that produced them, so the Linux run is a second
  qualification, never a replay of the Mac one. The modernc libc adapter and its pack landed
  for Linux on 2026-09-27. Still open for Linux then: a host-clock escape audit to replace DTrace,
  and the `./tests` closure. The audit was replaced on 2026-09-29 by a static inventory (R5
  amended): on linux/amd64 the clock is read through the vDSO, invisible to seccomp and ptrace,
  and the interception is platform-neutral Go, so `toolchain/clock_inventory_test.go` pins every
  standard-library host-clock reference per platform. Its first run found three unguarded reads
  recorded as escapes: `gcMarkTermination` writes host wall time to `MemStats.LastGC` (and
  `debug.GCStats`, and the Prometheus `go_memstats_last_gc_time_seconds` gauge), the FIPS
  entropy source's `monoTime` (FIPS mode only), and the execution tracer's clock snapshot
  (tracing is outside the contract); on Linux also `syscall.Gettimeofday` behind the `syscall`
  pack gate. `LastGC` is the one reachable in ordinary targets.
- Linux re-measurement, 2026-09-28 (fork run 36493869196, linux/amd64, both seeds, exact
  replay): all 18 supported workloads qualify, 10 stay at the amd64 xxhash boundary, 0 failed,
  0 infrastructure errors, 36 replays with none diverged, 40 s of reported execution time
  (the requalification step took 19 minutes of job wall time), 9.0 GB of
  retained Campaigns. `temporal.json` now expects `qualified` on linux/amd64 for the F5
  user-timers and activity batch cancel suites and the ten F6 suites. The F3 probe stays
  `intermittent` there: seed 17 was `nondeterministic` in run 36466396209 and qualified in the
  three runs since. Per-seed linux execution time and retained bytes (seed 11 / seed 17):

  | Workload | Seconds | Campaign MB | Trace MB | Choice decisions |
  |---|---|---|---|---|
  | frontend-system-info (F3) | 0.2 / 0.5 | 366 | 0 | 3353 / 3289 |
  | user-timers-workflow (F5) | 0.4 / 0.4 | 322 | 1 | 5701 / 5729 |
  | activity-batch-cancel-boundary (F4/F5) | 2.4 / 2.6 | 339 | 16 | 58580 / 58654 |
  | functional-activity (F6) | 1.1 / 1.0 | 329 | 8 | 27728 / 28424 |
  | functional-cancel (F6) | 0.9 / 1.0 | 327 | 5 | 21699 / 21837 |
  | functional-child-workflow (F6) | 1.0 / 1.0 | 328 | 7 | 26059 / 25880 |
  | functional-continue-as-new (F6) | 1.9 / 2.0 | 336 | 15 | 52018 / 51969 |
  | functional-cron (F6) | 0.5 / 0.5 | 324 | 2 | 9960 / 9897 |
  | functional-query (F6) | 0.9 / 1.0 | 329 | 7 | 25696 / 25495 |
  | functional-signal-chasm (F6) | 2.3 / 2.4 | 338 | 16 | 57913 / 59670 |
  | functional-timer (F6) | 0.4 / 0.4 | 323 | 2 | 8669 / 8614 |
  | functional-update (F6) | 4.8 / 4.6 | 361 | 39 | 132936 / 133145 |
  | functional-workflow (F6) | 2.0 / 1.8 | 337 | 16 | 55444 / 55698 |

- Move the Temporal qualification from weekly cron to a required check on changes under
  `tools/gomad3`, `tests`, `tests/testcore`, `go.mod`, and any server package the closure
  review names.

**Constraints.**

- "Any test" means unsupported count zero, not expectations met. The set-level report must
  print `supported`, `unsupported`, `failed`, and `infrastructure_errors` separately, per
  COMPAT-2, and the gate reads `unsupported == 0`.
- A test that stays unsupported after milestone F6's triage is either fixed upstream or
  excluded by name in the manifest with an owner and a date. Silent skips are refused.

**Acceptance.**

- The Temporal qualification-set report for `./tests` shows `unsupported == 0`,
  `failed == 0`, `infrastructure_errors == 0` on `darwin/arm64`.
- The same set on `linux/amd64` shows the same counts.
- The required CI check runs on pull requests touching the listed paths and finishes inside the
  budget.
- Each newly added test in `./tests` on a subsequent pull request appears in the report without
  a manifest edit.

**Status (darwin/arm64, 2026-09-29).** The full `./tests` set was run eight times on darwin
while the manifest generator and sharding landed (run 8: 138 supported, 0 unsupported, 0
infrastructure errors, 3 failed), and then retired as a gate (user decision, 2026-09-29: it is not
scalable); the sharding, merge and caches stay as tooling. The three failures and the two
whole-suite divergence exclusions of run 7 shared one channel, found with a buffered
event log of every span allocation, greying and drain slice from a debug toolchain: Go 1.27
records in each P the M that took it (`p.oldm`, a copy of the M's 8-byte `self` weak handle,
written by `acquirep`); which M takes the P after a runner-syscall hand-off is host timing, so
the scan of the P greyed a different handle per run, 8 bytes of scan work moved between
drain slices, the assist or worker ending a slice allocated a work buffer or not, and every
later page of the heap followed: `%p` in the server's membership resolver logs differed
between a recording and its replay (`nexus-otel` seed 17, 6 of 8 replays), and the callbacks
and chasm signal suites diverged between fresh repetitions. `gomadGreyRuntimeStructures`
now shades every M's `self` handle and every P's `oldm` handle at mark start. With the fix,
on the production toolchain: `TestNexusOTELSuite`, `TestSignalWorkflowTestSuiteLegacy`,
`TestSignalWorkflowTestSuiteChasm` (seeds 11 and 17, `--repeat 4`, every retained success
replayed 6 more times) and `TestCompletionCallbacksSuite` (`--repeat 2`, 4 replays each)
qualify with exact replays, `make -C tools/gomad3 test-runtime` and the core set (7 of 7,
the mount fixture now also hands the P off on every round and prints its page layout) pass.
One residual channel remains and is recorded, not fixed: about one chasm seed-11 replay in
thirty (1 of 28) still differs in stderr, and the instrumented runs place it before any
collection, at decision 46 of cluster start, where two heap-span refills (size classes 11 and
50) swap order between same-seed runs; its allocating goroutine was not identified.
`TestVersioningFunctionalSuite` never diverged: its WAL-mode SQLite database passes 16 MiB
and the in-memory filesystem refused the write (`SQLITE_IOERR_WRITE`, every later
transaction failed); the per-file bound is now 256 MiB and the total 1 GiB, and the suite
runs to the end in about 20 minutes of wall time per repetition with one deterministic
failure left, `TestDescribeTaskQueueEnhanced_ReportFlags` ("poller info should not be
reported", versioning_test.go:4203), which is not yet demonstrated as a test bug and stays
an open finding rather than an exclusion. The `TestWorkerOperation` skip of the OTEL suite
is now a demonstrated test bug (it sorts the client and server spans by StartTime, equal
under virtual time). The exclusions that remain in `tests.generator.json` are the four
I/O-transcript overflows and the named subtests; the callbacks and chasm exclusions are
removed.

## F8: consolidate architecture after functional qualification

**Spec.** [fn-102-gomad-architecture-consolidate](../.flow/specs/fn-102-gomad-architecture-consolidate.md)

**Outcome.** The runtime bootstrap protocol and shared Runner execution policy each have one
owner, public Runner interfaces expose usable contracts, and architecture checks enforce the
intended package structure. The existing runtime, Runner, World, Record, and Artifact seams
remain intact. The [architecture assessment](../.flow/artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
records the findings and evidence behind this work.

**Details.**

- Generate the runtime's allocation-free bootstrap header and seed decoder from the existing
  I/O schema, register its overlay file in the version descriptor, and preserve the current
  early-read and later full-validation phases.
- Extract private completed-execution assessment and retention policy shared by seed, choice,
  and simulation exploration. Keep scheduling, cancellation, counters, seed-ordinal publication,
  and atomic exploration-round commits with their existing owners.
- Move `Executor` and `ReplayExecutor` injection behind private dependencies across campaign,
  replay, resume, shard execution, and minimization. Migrate repository callers and preserve
  the usable `Preparer` and `ArtifactReplayer` contracts.
- Discover all host packages in architecture checks, inspect both qualified platform source
  sets, and test ownership, targeted host-effect restrictions, and public signature visibility
  with negative fixtures. Exclude runtime overlays and independent fixture modules explicitly.
- Reconcile platform, determinism, choice-exploration, and simulation documentation with the
  implementation and current qualification evidence, preserving stable requirement IDs.

**Constraints.**

- F8 depends on F7. Capability closure, observed repeatability/replay defects, and Linux clock
  auditing stay with F4–F7; this maintenance work cannot narrow their acceptance criteria.
- Preserve CLI behavior, schemas, failure classifications and precedence, resource bounds,
  and replay compatibility checks. Canonical bytes remain unchanged for identical supplied
  inputs and identities; rebuilding the implementation legitimately changes its identity.
- Removing public executor injection is an explicit Go source API change. Do not expose
  process or wire internals to make those interfaces implementable.
- Preserve fresh-process isolation, native timer ownership, independent corpus retention,
  and existing recovery rules. No generic strategy framework, scheduler redesign,
  deterministic-GC implementation, or new simulation capability belongs to this milestone.

**Acceptance.** The spec's R1–R6 are authoritative; the milestone summary is:

- The actual runtime bootstrap consumer passes valid, empty, truncated, malformed-header,
  and seed-boundary cases on both qualified platforms, retaining later checksum/identity
  rejection and disabled/direct-seed behavior.
- All three exploration strategies use the shared policy while preserving fixed-identity
  evidence, error precedence, retention limits, cancellation, and interrupted recovery.
- Deterministic 10-job and 100-job cases at parallelism 2 preserve execution and retention
  bounds; data-flow review finds no new total-seed-sized policy state or full-payload copies.
  This is a resource-bound check, not a throughput or memory benchmark claim.
- Intended public Runner use compiles outside its subtree, private fake-executor tests remain
  effective, and architecture fixtures catch ownerless packages, forbidden effects/imports,
  and inaccessible public signature types.
- Documentation matches current evidence. Generated-source checks, focused tests, full Gomad
  gates on `darwin/arm64` and `linux/amd64`, and project lint pass. An unavailable environment
  is recorded as a blocker, never as passing qualification.

**Status.** Planned on 2026-09-27 with six tasks and an explicit dependency on F7. Flow-Next
validation passes and plan review returned `SHIP`; implementation has not started. The
existing architecture test package passed during assessment, which is not full runtime
qualification.

## F9: run a downstream cell under the deterministic contract

**Spec.** [fn-104-gomad-run-a-downstream-cell-under-the](../.flow/specs/fn-104-gomad-run-a-downstream-cell-under-the.md);
rationale and capability inventory in [GOMAD_CLOUD.md](GOMAD_CLOUD.md).

**Outcome.** A Go module outside this repository that embeds the Temporal server as a library
and adds its own in-process services (a replicated storage layer on an embedded key-value
engine, a gossip-membership control plane, a replication sidecar over a blob store) is a
supported `gomad` target: its in-process cluster smoke test prepares, analyzes as `supported`,
and either qualifies with exact replay on darwin/arm64 or has every non-qualified outcome
classified with a finding.

**Details.**

- Runner: `explore`, `qualify`, and `analyze` gain `--working-dir`; adapter selection and the
  schema read-only mount resolve from the module root and the server's module directory (a
  local `replace` or the module cache) instead of the process working directory. The forced
  build environment (`GOWORK=off`, cleared `GOFLAGS`, `GOENV=off`, `-mod=readonly`) is
  documented with its consequences: no workspace files, no vendoring, private-module settings
  as exported variables.
- Compatibility packs: one reviewed downstream pack per platform for the `syscall` and
  `golang.org/x/sys` facts the downstream closure reaches through libraries no Temporal pack
  covers, plus re-pins of the two modules already at newer versions downstream. Packs stay
  embedded; an external pack directory is a recorded consideration, not a requirement.
- Adapters: two exact, digest-anchored adapters for `remain_unsupported` imports under the
  membership layer (a metrics library that registers a signal handler, an address library that
  shells out). One version per module; drift fails closed.
- Boundary operations that closure analysis cannot see get a recorded disposition — modeled
  with COMPAT-5 evidence, target-injectable with the injection point named, or denied with the
  exact finding: datagram sockets, advisory file locks, `statfs`, concrete listener-type
  assertions, all-interface binds, port probing, process metrics, and long readiness waits
  under the virtual clock (policy owned by `fn-103`).
- A downstream-seam guide states the `gomad` tag convention the server already uses
  (`_gomad.go` / `!gomad` pairing, default build unchanged, linked mode as the measurement of
  what the linker removes) for the seam classes the assessment found: volume-discovery and
  repository-root subprocesses, signal handlers in lifecycle and test helpers, a CLI package
  leaking into the in-process closure, and cloud credential chains behind blob-store and
  metrics providers.

**Constraints.**

- Every constraint under "Constraints that apply to every milestone" holds. No downstream
  source change is made from this repository; the guide and the analysis name the sites, the
  downstream repository owns the seams.
- Out-of-process services the downstream integration harness normally requires (a wide-column
  base store, write-ahead-log proxies, a coordination store, an object store) are a boundary.
  Gomad does not model them. Their pure-Go, loopback-only, in-process substitutes — the same
  shape as `testcore`'s SQLite and `StaticServiceHosts` — are downstream work and a
  precondition for running the server-embedding harness, not for running the storage services
  alone.
- Every deterministic-I/O adapter version matched downstream on 2026-09-29. That is
  coincidence, not contract; the milestone does not add multi-version adapters.

**Acceptance.** The spec's R1–R6 are authoritative; the milestone summary is:

- `gomad analyze --capability-mode=closure` over the downstream in-process cluster test package
  reports `supported` on darwin/arm64 with the new packs and adapters, run from the downstream
  checkout against this branch's toolchain.
- `gomad qualify --repeat 2` on two seeds passes with exact replay, or every other outcome is
  classified (capability blocker, unmodeled operation, watchdog, evidence divergence) with its
  finding recorded here.
- README and ARCHITECTURE record downstream-module support as a supported target shape with
  its limits.

**Status.** Assessed on 2026-09-29 on the `gomad` branch (`8bf340bb9`) against one downstream
module by reading; measured the same day with the toolchain built from `3bc1fe643` on
darwin/arm64, from the downstream module root with the server replaced by this checkout (no
other `go.mod` change was needed; every shared dependency version already matched). Closure
mode: 1,818 packages, `unsupported`, 82 blockers (29 `remain_unsupported`, 45 `add_exact_pack`,
8 `model_operation`). Linked mode: 78 live, 37 eliminated — 11 `remain_unsupported` (five cloud
credential chains through the CLI package and blob-store provider, the membership layer's
address library, five downstream subprocess and signal sites), 34 `model_operation` (nine in the
storage engine's filesystem layer: `chown`, `link`, deadlines, raw descriptors,
`ReadFrom`/`WriteTo`; DNS, interface, and address resolution; one UDP listen; `statfs`;
`process.signal`), and 33 `add_exact_pack` (seven arm64 assembly files, nine `syscall`/`x/sys`
imports, seventeen linknames). Two Gomad findings: the `x/sys` packs bind their activation to
the libc adapter, which does not activate in a closure without SQLite, so `x/sys` facts the
server never sees are blockers here; and the linker removes the metrics library's `os/signal`,
so linked mode needs one new adapter rather than two. The full inventory is in
[GOMAD_CLOUD.md](GOMAD_CLOUD.md) under "Baseline measurement"; the raw reports name downstream
packages and are not retained. The server-embedding harness remains out of reach because of its
external service topology. Spec created; no tasks yet.

## Out of scope

- New fault injection, partition, crash-restart, and multi-node capabilities through
  `tools/gomad3sim`. Those are the simulation track and cannot host `testcore`; F8 only
  consolidates shared Runner policy used by the existing exploration strategies.
- Multi-P scheduling, deterministic GC, DPOR, and preemption bounding. These are BUG-7 research
  items and no milestone above requires them.
- Revival of gomad1 or gomad2. [GOMAD_CMP.md](GOMAD_CMP.md) records why.

## Open risks

- **GC timing** is not controlled and shares the seeded runtime stream. Allocation-heavy suites
  may diverge between repetitions for reasons no milestone above fixes. Milestone F5 measures
  it; a positive finding reopens the research item.
- **Spin loops** anywhere in the cluster stall virtual time. The matching and history services
  contain pollers with backoff, which are fine, but a single `for {}` with a non-blocking
  select is fatal under this runtime.
- **Upstream Go releases** invalidate the patch and the boundary manifest each time. Milestone F2
  is the first port; every later Go bump repeats it.
- **Compatibility packs pin exact module versions.** Every dependency bump in the root go.mod
  that touches a packed module invalidates the pack and reopens milestone F4.
- **Deletion pressure.** fn-81 is reviewed and ready. Until milestone F0 lands, every commit on
  the branch can remove the code this plan depends on.
