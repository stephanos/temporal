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

The ladder is strictly ordered. Each milestone assumes the previous one's acceptance criteria
hold. A milestone whose criteria fail blocks the next one; it never gets narrowed to pass.

## Baseline on 2026-09-08

The assessment that produced this document verified the following on the working tree.

- The root module declares `go 1.27` and `toolchain go1.27.0` since 2026-08-23. Gomad v3 pins
  `go1.26.4` in `tools/gomad3/toolchain/version/version.json` and builds targets with
  `GOTOOLCHAIN=local`. No Temporal package builds under the patched toolchain today.
- `tools/gomad3/.toolchain` and `tools/gomad3/.bin` do not exist in the checkout. The
  toolchain has to be rebuilt from source before any milestone below can be measured.
- The conformance fixture corpus under `tools/gomad3/internal/gomadtool/conformance/testdata`
  was never committed because the root `.gitignore` ignores every `testdata/` directory. The
  runtime tier, the interception tier, and every `io_*_toolchain_test.go` cannot run.
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
- Spec [fn-81](../.flow/specs/fn-81-delete-the-pre-testpilot-go-generations.md), reviewed
  as ready to ship, deletes every gomad tree, the functional probe, and its pack.

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
- **Platform.** milestones F0 through F6 are qualified on `darwin/arm64` only. Linux is
  milestone F7 and gates CI, never the determinism claim.
- **Server source changes are allowed but bounded.** A change under `common`, `service`,
  `temporal`, or `tests/testcore` is acceptable when it isolates an optional provider behind a
  build tag or an injection seam and the default build is unchanged. A change that alters
  runtime behavior for production builds needs its own review outside this plan.

## F0: decide retention

**Outcome.** The repository either keeps Gomad v3 with a carve-out from fn-81 or drops the goal
this document serves. Nothing below starts until this is settled, because fn-81 removes the
toolchain, the probe, and the pack in its first two commits.

**Details.**

- Amend fn-81 so its deletion set keeps `tools/gomad3`, `tools/gomad3sim`,
  `tools/gomad3integration`, `tests/gomadfunctional`, the root Makefile `gomad3*` targets, the
  `gomad3` workflow, and the root go.mod `require` and `replace` for `github.com/temporalio/gomad`
  only if `tools/gomad2` survives as the parity reference. If gomad2 goes, the parity manifest
  under `tools/gomad3/simulation/parity` loses its source paths and must be retired in the same
  change.
- `tools/gomad`, `tools/gomad1` and, if the parity manifest is retired, `tools/gomad2` are
  deleted as fn-81 specifies. gomad1 has no `go.mod`, so its 54,590 lines compile into every
  root-module build today for zero callers.

**Acceptance.**

- fn-81's deletion set and retention set name each gomad tree explicitly with a disposition.
- `go build -tags 'test_dep integration' ./...` passes after the amended fn-81 lands.
- `go run ./tools/planindex` passes with this document registered.

## F1: restore the checkout

**Outcome.** A developer on a clean `darwin/arm64` checkout can build the toolchain and run
every Gomad v3 gate that exists today, and the two integration contract tests pass.

**Details.**

- Commit the conformance fixture corpus. Add `!/tools/gomad3/internal/gomadtool/conformance/testdata/`
  and `!/tools/gomad3/internal/compatibilitypack/testdata/` to the root `.gitignore` next to the
  existing umpire allowances, then regenerate the fixtures from the driver if the originals are
  gone. The boundary manifest's `conformance_fixtures` field names `io_filesystem` and `io_net`
  as the semantic canary, so those two are mandatory.
- Point the root Makefile at the real `exec.sh` or move the script to the path the Makefile
  expects. Pick one; the compatibility-pack request records the path.
- Bump `HarnessSpecSchema` in `tools/gomad3/simulation/parity/manifest.go` and the JSON manifest
  to `gomad3.simulation-spec/v7`, or explain in the spec changelog why gomad3sim moved without
  the manifest.
- Reconcile the qualification-set schema. Either the loader in `qualification/set/set.go`
  accepts `v3` with `suites` and `run_timeout`, or the two manifests return to `v1`. The CI
  workflow asserts the `v6` report schema, so the loader change is the smaller diff.
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

## F2: port the toolchain to go1.27.0

**Outcome.** The patched toolchain matches the root module's `toolchain` directive, so Temporal
packages build under it again.

**Details.**

- Run the upgrade flow in `tools/gomad3/upgrade`: `patch-materialize` the go1.26.4 patch onto
  the go1.27.0 source archive, resolve rejects by hand across the 16 patched files, then
  `patch-regenerate`, `make generate`, and `make -C tools/gomad3 upgrade-dossier`.
- The compiler fingerprint check fails the build for any intercepted `os` or `net` function
  whose body changed upstream with a stable signature. Each such function needs a re-reviewed
  entry in `deterministicio/boundary/manifest.json`. Expect the count of intercepts to move
  from 131.
- Update `version.json` (archive URL, SHA-256, patch and overlay allowlists), regenerate
  `version_generated.mk`, `expected-intercepts-go1.27.0.txt`, and the boundary report and
  upgrade guide for `go1.27.0-darwin-arm64`.
- The DTrace clock audit needs root. Record whether it ran; a dossier without it stays
  `qualified=false` and that is the honest state, not a failure of this milestone.

**Constraints.**

- The patch applies with zero fuzz. A hunk that needs fuzz is rewritten.
- The boundary diff between go1.26.4 and go1.27.0 is approved by digest through
  `GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256`, never waved through.
- Keep the go1.26.4 descriptor in git history only. Two live pins double every later milestone.

**Acceptance.**

- `tools/gomad3/.toolchain/bin/go version` reports `go1.27.0`.
- `make -C tools/gomad3 validate-toolchain` passes.
- The upgrade dossier reports every gate passed and the boundary diff approved or empty.
- Milestone F1's acceptance criteria still hold on the new toolchain.
- `make gomad3-qualification` reproduces the current Temporal corpus result of 5 supported and
  11 unsupported with the same blocker paths.

## F3: qualify the existing functional probe

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
- Remove `tools/umpire3` from the `./tests` closure. It imports `os/exec` and arrives through
  the monitor seam that fn-81 deletes anyway.
- Re-run the analysis with the cloud archiver, Elasticsearch, Cassandra, and SDK worker
  providers excluded by tag to measure how much of the closure is optional. Do not make that
  separation a prerequisite if linked mode already closes the set.

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

## F6: a package-level functional slice

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

## F7: any functional test, and CI

**Outcome.** The whole `./tests` package is enumerated in the qualification set, every test has
a disposition, and the unsupported count is zero on `darwin/arm64`. A Linux bundle lets the
gate run in CI.

**Details.**

- Generate the manifest from `go test -list` output instead of curating it by hand, so a new
  test lands with a default expectation of `qualified` and fails the set if it is not.
- Split the set into shards with `gomad plan` and `execute-shard` so a full run fits the
  90-minute CI budget. Merge with `gomad merge`.
- Build the `linux/amd64` platform bundle per COMPAT-7 with its own boundary manifest,
  adapters, and publication primitives. Artifacts replay only on the bundle that produced them,
  so the Linux run is a second qualification, never a replay of the Mac one.
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

## Out of scope

- Fault injection, partitions, crash-restart, and multi-node scenarios through `tools/gomad3sim`.
  Those are the simulation track and cannot host `testcore`; nothing here depends on them.
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
- **Upstream Go releases** invalidate the patch and the boundary manifest each time. Milestone 2
  is the first port; every later Go bump repeats it.
- **Compatibility packs pin exact module versions.** Every dependency bump in the root go.mod
  that touches a packed module invalidates the pack and reopens milestone F4.
- **Deletion pressure.** fn-81 is reviewed and ready. Until milestone F0 lands, every commit on
  the branch can remove the code this plan depends on.
