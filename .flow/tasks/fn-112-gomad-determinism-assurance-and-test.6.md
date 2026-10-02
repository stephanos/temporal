---
satisfies: [R3, R7]
---
# fn-112-gomad-determinism-assurance-and-test.6 Add conformance fixtures for unverified channels and state the closure-mode limit

## Description
One seeded black-box fixture with a positive control per unverified channel, and the contract sentence on closure mode (R7).

**Size:** M
**Files:** new fixture directories under `tools/gomad3/internal/gomadtool/conformance/testdata/`, `runtime_campaign.go` and the `runtime_*.go` file owning each behavior
**Touches:** [tools/gomad3/internal/gomadtool/conformance/runtime_*.go, tools/gomad3/internal/gomadtool/conformance/testdata/netpoll/**, tools/gomad3/internal/gomadtool/conformance/testdata/sigprof/**, tools/gomad3/internal/gomadtool/conformance/testdata/profile_sampling/**, tools/gomad3/internal/gomadtool/conformance/testdata/numcpu/**, tools/gomad3/internal/gomadtool/conformance/testdata/timer_ties/**, tools/gomad3/internal/gomadtool/conformance/testdata/runq_shuffle/**]

### Approach
- First step (R3): re-anchor assessment findings Q7 and Q8.
- Channels: netpoll readiness, SIGPROF, block and mutex profile sampling, `runtime.NumCPU`, timer-tie draws, run-queue shuffle draws.
- Follow the `runqueue` fixture: a `main.go` under the shared testdata module, a build registration, and listing in the repeatability and load tables.
- Each fixture needs a positive control showing the detector can fail (disabled mode varies, or different seeds diverge).
- A channel that stays outside the contract gets a fixture proving it fails closed, or a contract sentence naming it. `NumCPU` cannot be varied on a CI host; if no control is possible, take the contract-sentence route and say why.
- Confirm from `target.go` that closure mode adds no `-gomadguard` flag, and write the exact contract sentences (closure-mode limit, and any channel placed outside the contract) into the done summary. Task 10 puts them into README and SPEC.
- Depends on task 2, which edits the same campaign registration file.
- Fixture files must match the `runtime_*.go` grouping rule in `architecture_test.go`.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/internal/gomadtool/conformance/testdata/runqueue/main.go` — fixture shape
- `tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go:298` — build registration
- `tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go:17-104` — repeat and positive-control tables
- `tools/gomad3/target/target.go:711-716` — guard flags by capability mode

**Optional** (reference as needed):
- `tools/gomad3/internal/gomadtool/conformance/runtime_load.go:59-106` — load and seed-divergence checks
- `tools/gomad3/README.md:166-176`, `tools/gomad3/SPEC.md:204` — closure and guarded text

### Key context
- fn-105 task 12 Q4 also names `NumCPU`; share the finding.
- fn-114 E3 and E4 change which select and system-goroutine decisions are recorded; if they have landed, write timer and run-queue fixtures against that behavior.
## Acceptance
- [ ] Findings Q7 and Q8 re-anchored and marked confirmed, changed, or refuted
- [ ] Each named channel has a seeded fixture with a positive control, or a drafted contract sentence placing it outside the contract with the reason
- [ ] The closure-mode sentence and any out-of-contract channel sentences are in the done summary for task 10
- [ ] `make -C tools/gomad3 test-runtime` passes on darwin/arm64 with the new fixtures; linux status recorded
- [ ] A fixture that is nondeterministic on first run is recorded as a finding, not weakened
## Done summary
# fn-112-gomad-determinism-assurance-and-test.6 handover

Added dedicated native timer-tie and overflowing run-queue shuffle fixtures, registered them in the existing runtime build, repeatability, diversity, and host-load tables, and added a focused real-toolchain conformance test. Both fixtures emit only their completion permutation and enforce once-only completion; timer callbacks also enforce the shared virtual deadline. Existing comments and dirty changes are preserved. No runtime overlay, patch, descriptor, schema, dependency, or build identity changed; no Git staging, commit, push, stash, or worktree operation ran.

Q7 was **confirmed** before implementation: the dedicated channels were absent from the current build/repeatability/load registration tables. The final coverage adds timer_ties and runq_shuffle and uses the task's explicit excluded-channel option for host netpoll, SIGPROF, enabled block/mutex profile sampling, and unvirtualized NumCPU. Q8 remains **confirmed** at target.go:711–716: Closure skips the capability compiler/linker flag block; Guarded alone adds -gomadguard. classification.md and reanchor-source-excerpts.txt retain current source locations, reasoning, and hashes. The NumCPU finding also confirms the source-level Q4 candidate for fn-105 D12, without establishing D12's cause or completing its Linux acceptance.

The shuffle creator remains runnable while it creates 1024 children on one P, with async preemption disabled by activation and GC disabled in this fixture to prevent assists from draining the queue. The pinned queue has 256 slots plus runnext, so creation reaches runqputslow's seeded shuffle; the global batches later reach runqputbatch's shuffle. No fixed draw count or output permutation is asserted. Both timer and queue completion order also depend on ordinary Runnable decisions; the tests exercise the named untaped paths and observable same-seed repeatability, and do not assert isolated-stream sensitivity or choice-tape coverage.

The test-first focused check failed on the missing timer_ties directory, then passed after fixture creation (focused-red.log/focused-green.log; test-first.json). On the frozen final sources the complete conformance package, behavior-grouping test, focused vet, and gofmt check pass. The required `make -C tools/gomad3 test-runtime` gate passes on darwin/arm64 (exit 0); make-test-runtime.json/log retain its exact command, timestamps, exit, and unchanged toolchain cache-hit identity. Root make lint-code-fast was attempted with the pinned stock driver and GOLANGCI_LINT_BASE_REV=HEAD and exits 2: the root module cannot load nested tools/gomad3 packages (underlying golangci-lint exit 7). This is not a clean lint result. No native linux/amd64 host was available; Linux execution remains **unverified**.

Positive-control evidence retains 124 fresh observations, 62 per fixture, raw stdout/stderr, validated complete permutations, exit codes, binary hashes, and every output hash. For each fixture seeds 0–31 produce **32 distinct completion orders**. Seeds 0 and 1 each have 11 observations with one stable stdout hash; max uint64 has 10 observations with one stable stdout hash. The passing full gate additionally executed 100 observations per boundary seed, 32-seed diversity, and eight repetitions under bounded CPU load. No nondeterministic observation was weakened or accepted. The parent independently validated these observations and frozen-source patch bindings in parent-evidence-validation.json; mandatory implementation review remains the conductor's responsibility.

Exact proposed contract sentences for task 10 to insert in README and SPEC:

> Closure capability mode performs dependency review without compiling `-gomadguard` guards; an exact compatibility-pack admission does not make host operations deterministic, and admitted code must stay within the declared deterministic boundaries.

> Real-socket and descriptor readiness delivered by host netpoll is outside the determinism contract; supported modeled loopback TCP uses deterministic in-memory readiness instead.

> SIGPROF delivery and CPU profiling are outside the determinism contract because signal arrival and CPU samples depend on host execution.

> Enabling block or mutex profiling is outside the determinism guarantee: host-dependent contention timing and profile sampling can change random draws, profile allocations, and subsequent runtime state, especially on linux/amd64.

> `runtime.NumCPU` is not virtualized and reports OS-detected CPU availability at process startup; workloads that use this value require the same host CPU configuration for repeatability, and cross-host CPU-count equivalence is outside the contract.

The exclusion route avoids fabricated controls: changing GOMAXPROCS cannot vary NumCPU, and one Darwin host cannot qualify Linux profile sampling or host-signal/network behavior. The exclusions do not promise new fail-closed runtime behavior. README/SPEC insertion belongs to task 10; this handover does not prematurely claim their R7 documentation acceptance or a broad determinism guarantee.

Frozen source manifest SHA-256: `89a8741c93de3da9b303e3279e3189e72771a59f8678e340333c34d5b769ad48`. Task-only patch SHA-256: `eea020b2f3253a542c1a4b8378086783fadcd9b2b7f2e9554f898af7b3c4f71e`. The seven-file manifest is source-bindings.json; actual dirty before-copies and before-bindings.json separate this task from prior uncommitted work. Toolchain key remains `6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee` on darwin/arm64. Initial gate metadata used a manifest generated immediately before formatting; the corrected final manifest hashes the sources actually used by those gates, and each affected record retains the old manifest hash and the correction reason.

Independent implementation review returned SHIP with no findings and task-6 R3/R7 coverage met. Receipt: task-6/review-round1.json; session 01a0fc10-4c2b-7bb1-aa07-82d5f9431c52. The reviewer independently checked the seven-file patch, source/evidence bindings, all 124 observations, channel paths, and contract exclusions. All 38 parent-bound files remain unchanged after review. Work remains uncommitted at the user's instruction.

stage: impl-review - ran (model: gpt-6-astra at high)
stage: plan-sync - skipped(config: planSync.enabled=false; exact documentation wording retained for task 10)
stage: wave-dispatch - skipped(policy: shared conformance files and no worktrees; sequential worker)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: make -C tools/gomad3 test-runtime (PASS darwin/arm64; GOFLAGS=-tags=test_dep, stock Go1.27.1 driver), tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -count=1 -tags=test_dep ./internal/gomadtool/conformance (PASS), tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -count=1 -tags=test_dep -run ^TestConformanceRuntimeIsGroupedByBehavior$ . (PASS), tools/gomad3/.toolchain/bin/go -C tools/gomad3 vet -tags=test_dep ./internal/gomadtool/conformance (PASS), gofmt -l changed Go files (PASS, empty output), git diff --check scoped changed tracked files (PASS), make lint-code-fast (ATTEMPTED, exit2; nested-module package discovery failure; underlying linter exit7), channel-probe.py (PASS; 124 raw observations,32 distinct orders perfixture), Independent implementation review: SHIP; task-6/review-round1.json; no findings, Parent verified 38 bound files unchanged after review, 124 output observations, and task-only patch reverse application
- PRs: