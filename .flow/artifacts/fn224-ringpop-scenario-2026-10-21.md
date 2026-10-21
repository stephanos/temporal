# fn-224.3 — three-node ringpop partition/heal scenario

Branch `fn224-ringpop`. Raw evidence: `/home/agent/wt/evidence/fn224/scenario/` (runs, bisect variants, `DIAGNOSIS.md`).

## Fixture

- Guest: nested module `tools/gomad/internal/backends/wasm/testdata/ringpop` (`main.go`, `go.mod`, `go.sum`). It uses `ringpop-go` v0.1.0 and `tchannel-go` v1.22.1 directly; `go.mod` requires the Temporal module through `replace ../../../../../../..` so MVS selects exactly Temporal's versions. All 172 `go.sum` lines also appear in the root `go.sum`; `GOOS=wasip1 GOARCH=wasm go build -mod=readonly .` exits 0 offline.
- Scenario: `command/testdata/scenario/ringpop.json` — nodes a/b/c at 10.0.0.1-3:7946, six directed links, `c` partitioned from `a` and `b` at 10 s (symmetric) and healed at 30 s. Boot args: self, ring hosts, view 25 s, heal 30 s, deadline 90 s. Existing schema fields only. `observation_bytes` is 64 MiB, the `MaximumObservationBytes` cap.
- Installation config: `command/testdata/scenario/ringpop-config.json` (the scenario `config.json` with larger call, transcript, descriptor and NetworkV2 capacities; same clock, `ReadStepNanos=0`).
- `TestOperatorRingpopScenarioPreflight` normalizes the scenario (3 nodes, 6 links, 4 faults) and admits the config (exit 0, `evidence/fn224/scenario/preflight-test.log`).
- Operator guide: `CLI.md`, "Explore a typed cluster scenario".

Convergence witness: after the heal each node needs three consecutive one-second polls with all three members reachable, then prints `converged <node> after heal <d> checksum <n>` and runs to the shared 90 s deadline, so no peer stops before every node has observed the full ring. A missing convergence panics and is retained.

## Results

Identities: gomad `sha256:407263f9be00…`, helper `sha256:b25439140f0c…`, compiler `sha256:1675694ef690…`, model `sha256:73c1a4f3f64d…`, guest module `sha256:8249a1869116…`.

| Seed | Outcome | Wall | Partitioned view (a / b / c) | Converged | Replay |
| --- | --- | --- | --- | --- | --- |
| 7 | success | 456 s | {a,b} / {a,b} / {a,c} | all three 3 s after heal, checksum 990001539 | `reproduced=true result=success choice-replay=none`, 453 s |
| 19 | success | 448 s | {a,b} / {a,b} / {a,c} | all three 3 s after heal, checksum 990001539 | `reproduced=true result=success choice-replay=none`, 456 s |

Cluster replay checks the retained module, lifecycle, network evidence and per-incarnation choices; `choice-replay=none` is the documented cluster replay mode. The two seeds produced identical observable output; ringpop's member selection did not diverge between them. During the partition `c` still listed `a` as reachable (suspicion not yet escalated) while `a` and `b` had dropped `c`; this is ringpop SWIM behavior on the modeled network and is not asserted.

Seed 19's first attempt failed before execution with `gomad-wasmhost has unexpected link count 2`: a diagnostic `perf record` had hard-linked the shared helper into `~/.debug`. The link was removed and seed 19 rerun; the failure is infrastructure, not a scenario outcome.

## Findings filed

The first fixture used a 1 MiB `observation_bytes`. The cluster broker charges 64 B of World payload per brokered operation for the life of the run, and three gossiping nodes use about 940 operations per virtual second, so the budget ran out at 17.4 s virtual. The helper then misclassified the error as infrastructure and lost the terminal result: helper and Go both blocked until the wall watchdog fired, with empty streams. A 300 s timeline with 64 MiB was too slow for a 15 min watchdog (about 3-8 wall s per virtual s, growing). Both are gomad defects, filed as fn-235 (.1 fast classified failure, .2 superlinear World commitment cost). They are not ringpop or tchannel faults: the same three-member ring in one guest process ran 40 s virtual in 27 s wall. Whether the run-lifetime 64 B/operation charge against `observation_bytes` is intended is an owner question recorded in fn-235.

## Boundary (R4)

No diff under `tests/testcore`, `wasm/wasi`, `internal/model`, `wasm/toolchain` or `wasm/wasmhost` on this branch.

## Rerun

`/home/agent/wt/evidence/fn224/scenario/explore.sh <label> 7,19` (installation `evidence/fn224/scenario/gomad-wasm.json` from `ringpop-config.json`, cwd = guest module), then `gomad replay <artifact>`.

## Review and candidate rerun (2026-10-21, `ab7625c647`)

Independent review: SHIP-WITH-FIXES (`/home/agent/wt/evidence/fn224/REVIEW.md`). Fixed in `ab7625c647`: H1 (ringpop fixture module classified in archcheck `expectedModules` and lintcode; `TestWASMArchitecture` and `make lint-code-fast GOLANGCI_LINT_BASE_REV=66391696db` exit 0), M3 (guest asserts a/b dropped c at the view, compares full member sets after the heal, re-checks at the deadline; exit status replaces an output stop marker, recorded in CLI.md), L1, L2 (do-not-tidy note), L4. The earlier evidence above used the fn-221 sweep build of `1d7b37f00a` (M1).

Rerun on binaries built from `ab7625c647` (gomad `sha256:7ec6be501b8c…`, helper `sha256:558e2989a2cd…`, runtime from the branch; `/home/agent/wt/evidence/fn224/candidate/run-all.log`):

| Check | Result |
| --- | --- |
| Scenario seed 7 / 19 (tightened guest) | success both; same views and checksum 990001539; replay `reproduced=true result=success choice-replay=none` both |
| `TestRpoSuite/TestMonitor` seed 0 | success, leaf PASS |
| Its standalone replay | `execute replay target: choice trace malformed` (rc 3) |
| `TestRpoSuite/TestCompareMembers` seed 0 replay (empty choice tape) | same error |
| One-box `TestFrontendSystemInfo` seed 0 (M2) | not run: preparation failed with `no space left on device` (host disk 100%) |

The standalone replay failure is not caused by fn-224: the ringpop test module does not import `temporal/`, so its guest is identical with and without this branch, and it replayed exactly (27/27) on the sweep build. It is either a regression in standalone `--choices` replay at `66391696db`+ (the fn-228/fn-232 replay path) or an installation mismatch from pairing HEAD gomad with the older frozen stock compiler `29cb18ad…`; not attributed. Hand to the replay owner.

**Merge precondition (M2):** before merging `fn224-ringpop` into `gomad`, run the one-box `TestFrontendSystemInfo` at one seed with the larger slim module on a host with free disk (`candidate/run-all.sh` step 3). If it exceeds a recorded capacity, gate the unstub behind a per-workload tag per the spec.

## Merge with `gomad` (2026-10-21, `87fcc5e24e`, `502e7ce811`)

`gomad` merged into the branch twice without conflicts: first `3ad88e1cde` (219 commits ahead of the branch base), then `4466cc160e` (Flow specs and tasks only). Auto-merged files: `cmd/tools/lintcode/main.go` and `main_test.go`, `archcheck/inventory.go`, WASM `CLI.md` and `README.md`, `command/scenario_test.go`. The fn-224 delta against `gomad` is still the same 23 files. Raw evidence lives in `/home/agent/wt/evidence/fn224/`: `candidate2/` holds the merge commit, the controls and the one-box runs; `candidate3/` holds the evidence candidate described below.

Source gates on `87fcc5e24e` (stock Go 1.27.1, `candidate2/gates.sh`, `gates.log`, `gate-*.log`) all exit 0:

- `make -C tools/gomad test-wasm-slim-refusals` (3 named PASS)
- `go test -tags gomad,test_dep -run TestDefaultMembershipRefusesUnbuiltRingpop ./temporal`
- `go test -tags test_dep -run TestDefaultMembershipLinksRingpop ./temporal`
- `go test ./cmd/tools/lintcode`
- `TestWASMArchitecture`
- `TestSlimGuestClosureCutsOnlyOptionalProviderSeams` (default 1471 packages, slim 1384)
- `TestOperatorRingpopScenarioPreflight`
- `make lint-code-fast GOLANGCI_LINT_BASE_REV=gomad`
- `gofmt -l` (empty)
- R4 boundary diff against `gomad` under `tests/testcore`, `wasm/wasi`, `internal/model`, `wasm/toolchain`, `wasm/wasmhost` and `tools/gomad/rust` (empty)

`temporal/membership.go` selects `!gomad`, so the slim guest selects exactly one real membership implementation, and the gomad-tag stub stays a source-selection check (tightened R1).

### One-box cells with ringpop linked (M2)

These ran on the fn-224 tip `ab7625c647` with its own launcher and helper (`candidate/bin`: gomad `sha256:7ec6be501b8c…`, helper `sha256:558e2989a2cd…`), seed 0, 4 GiB memory, 2^50 fuel, 10 min execution timeout, `--choices`. Script: `candidate2/onebox-premerge2.sh`, log `onebox-premerge2.log`.

| Cell | Package / selection | Outcome | Leaf PASS line | Wall | Module |
| --- | --- | --- | --- | --- | --- |
| frontend-system-info | `./tests/gomadfunctional` `^TestFrontendSystemInfo$` | success | `--- PASS: TestFrontendSystemInfo` | 375 s | 215,011,592 B `sha256:1d258c87cd74…` |
| mandatory smoke | `./tests` `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$` | success | `--- PASS: TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds` | 223 s | 314,172,065 B `sha256:89ad10e3db1d…` |

Neither cell hit a capacity, fuel or memory refusal, so the unstub needs no per-workload tag. The first attempt (`candidate2/onebox-controls/premerge`) selected `TestFrontendSystemInfo` in `./tests`. That package does not contain the test, so the attempt ran no tests and does not count. The independent re-review found this.

### Execution on the `gomad` tip is blocked by `gomad`, not by fn-224

The Go launcher at `3ad88e1cde` cannot complete a cooperative execution with the current runtime source. Each run below fails the same way with and without fn-224.

| Workload | Merge commit (launcher `459bcedeeea8…`, helper `211401449a1d…`) | `gomad` tip without fn-224 (control) |
| --- | --- | --- |
| One-box `./tests` selection seed 0 `--choices` | `runner_failure: success_artifact_publication: finalize artifact manifest: invalid choice trace identity` (0 choice records) | identical error (`onebox-controls/gomadtip`) |
| `TestRpoSuite/TestMonitor` seed 0 `--choices` | `runner_failure: … WASI invalid: runtime decision header` | identical error (`gomad-tip-control/`) |
| Ringpop scenario seeds 7, 19 | `runner_failure: finish simulation: unsupported or incomplete WASM inspection evidence: decision/trace mapping` | identical error with the tip launcher and helper on a 20 s deadline variant (`fn235/cohort/tip-refused/`) |

Cause: `38572d7e49` (goroutine creation sites) moved the guest runtime to decision header version 2. It updated the Rust helper and the Rust host, but not the Go host:

- `wasi/runtime.go:156` accepts only version 1.
- The Go cluster inspection mapping (`cluster/inspection.go:46`) rejects the new records.
- Record validation (`evidence/record/validation.go:433`) still requires the earlier trace version.

Running without `--choices` is not an alternative: cooperative preparation requires choice recording by design. The Rust `gomad` binary reads only `gomad.rust-wasm-installation/v1`, whose installer is fn-232.11. These failures belong to fn-232.

One-box `./tests` module on the tip: 314,171,975 B with fn-224 vs 311,040,589 B without it, a difference of +3,131,386 B (+1.01%). This is consistent with the matched fn-224.1 delta.

### Evidence candidate `39694cd12b` (gomad `92c69594f2` + fn-224 `ab7625c647`)

`92c69594f2` is the last `gomad` commit before `38572d7e49`. It has the same cluster inspection and World commitment source as the tip, including fn-235. Binaries were built from the candidate: launcher `sha256:c3f35c0bfaf3…`, helper `sha256:02169018a92e…`. Script: `candidate3/run.sh`, log `run.log`.

Scenario, checked-in `ringpop.json` (90 s), `--keep-successes=all --success-limit=100 --success-bytes=4GiB`:

| Seed | Outcome | Fault-phase proof (`candidate3/faultphase.py`) | Replay of the success |
| --- | --- | --- | --- |
| 7 | success | pass | `reproduced=true result=success choice-replay=none` |
| 19 | success | pass | `reproduced=true result=success choice-replay=none` |

`faultphase.py` checks the retained record. Network evidence must hold all eight directed `c`↔`a`/`b` `partition` transitions with outcome `ok`, ordered before the eight matching `heal` transitions. Each node's stdout must show `bootstrapped`, then `partitioned view`, then `converged … after heal`. All three nodes must exit 0 at the scenario `wait` with outcome `completed`.

The guest itself asserts the phases:

- At the 25 s view, `a` and `b` must have dropped `c`.
- After the 30 s heal, three consecutive fresh full-member-set polls must pass.
- At the deadline, the full view is checked again.

The bootstrap marker cannot satisfy completion, and every node lives to the shared deadline. Replay reproduces the retained record, so the same fault-phase evidence reappears.

The tightened .3 command form (`--working-dir <abs>/tools/gomad … go-run ./internal/backends/wasm/testdata/ringpop`) is refused by the Go toolchain: `main module (go.temporal.io/server/tools/gomad) does not contain package …/testdata/ringpop`. The guest is its own nested module. Recommendation, recorded and applied: use an absolute `--working-dir` naming the guest module and `go-run .`, as `CLI.md` documents. Both forms and their outcomes are retained.

Ringpop leaves, `--choices --choice-bytes=64MiB --keep-successes=all --success-limit=100`, seeds 0,1,2: all nine leaves (`TestRpoSuite` ×4, `TestRingpopSuite` ×5) succeed on all seeds. That is 27/27 retained successes with the leaf's own `--- PASS` line and no `--- SKIP`. All 27 standalone replays fail with `execute replay target: choice trace malformed`. The cause is the known Go-host replay attestation regression from `dd00c57f70` (`backends/observed_result.go`: replay requires `InputTapeSHA256`/`InputDecisions`, which the Go `runtime.collect` path does not fill), present since before the fn-224 base. The ringpop test package does not import `go.temporal.io/server/temporal`, so fn-224 cannot affect it. Exact replay of all 27 passes remains proven only on the fn-221 sweep build `1d7b37f00a` (fn-224.2 artifact). R2's "every pass replays exactly" is therefore not re-proven on an integrated candidate; fn-224.2 stays open until the fn-232 replay owner restores standalone replay.
