# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

Status as of 2026-09-28. Delivered, each with SHIP implementation and completion reviews: fn-83,
fn-84, fn-87, fn-85, fn-86, fn-46, fn-33, fn-22, fn-26, fn-29, fn-91, fn-90, fn-89, fn-94 and fn-88. Their task receipts in `.flow/` and the git history carry
the details.

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire. Since fn-88, `Umpire.Search`
runs on Veil's concrete checker (a pure `bfsStep` loop with exact product-state deduplication and
a kernel replay gate on every witness) on Lean 4.32.0, with the old traversal kept as the frozen
`reference` backend and differential oracle; planning receipts record each monitored clause
kind's trust basis. After a protocol change,
`make umpire-rerecord-pinned-runs` refreshes every catalog-pinned recorded Run (fn-89.7).

### Delivery queue

1. **fn-92 — Compose entity machines into one Model**
   ([spec](../.flow/specs/fn-92-compose-entity-machines-into-one-system.md)); planned, 6 tasks,
   now that fn-88 is delivered. A `compose` command builds one Model from entity machines with declared action
   synchronization over a reachable-state enumeration; `restrict:` and `extend:` derive machines
   from a source table; `Workflow` and `Worker` entity modules are shared by the Start and Outage
   use cases; the first cross-entity claims are `verify` Queries. Version one realizes no Case over
   a composition and leaves the caller module, its fixtures and the canary's pinned Case identity
   untouched.
2. **fn-93 — Simplify the Lean model**
   ([spec](../.flow/specs/fn-93-simplify-the-lean-model.md)); planned, 43 tasks (Codex plan
   review SHIP), after fn-88 and fn-92. A simplification campaign over the handwritten Lean in
   `model/`: fix the defects the investigation found, derive what is written out by hand, delete
   what retired rules left behind (per-task owner decisions), reuse core Lean, and cut `model/`
   Markdown to one README and one ARCHITECTURE. Every golden, Case fixture, Definition ID and
   Behavior Fingerprint stays byte-identical. It absorbs fn-60's re-scoped aim.

### Carried forward, not specs

- **A field path through a repeated field** (deferred from fn-86 R2): selecting the element the
  observation correlates to the row's entity instance. `Temporal.Case.FieldPath` rejects every
  repeated-field path today.
- **The Umpire workflow's cold Lean build** (from fn-29 and fn-88 R20): both jobs in
  `.github/workflows/umpire.yml` build Lean targets from a cold `.lake` inside 30- and 40-minute
  timeouts. The first run with Veil (36377425097, 2026-09-28) failed in both jobs because a fresh
  checkout has no `proto/api.binpb`, the git-ignored descriptor set the model build reads; that
  input dates from 2026-09-19, and the earlier 2026-09-23 runs failed on it too. Before it stopped,
  the `portability` job built Veil's closure, the npm widget included, in 84 s with the runner's
  default Node. fn-88.7 added a `make proto/api.binpb` step to both jobs. The next push measures
  whether the whole cold build fits; if it does not, add a `.lake` cache step.
- **The worker ignores the timeouts a Case declares on Finish and NexusHandlerReply** (fn-90.6):
  the 5000 ms bounds on `finish-workflow` and `respond-async` bind nothing. A Case-authority gap.
- **Run capture records no workflow or handler instruction events** (fn-90.3, .6): a bounded
  worker step's own latency cannot be read from a recorded Run.
- **The server dates a completion-before-start started event in local time labelled UTC**
  (fn-90.7): the synthesized `NEXUS_OPERATION_STARTED` carried PDT wall time marked `Z`. No Verdict
  reads `EventTime`; an upstream issue is the next step.
- **`TestTestpilotNexusCallerScheduleToStartTimeout/chasm` failed once** (fn-94.17, 2026-09-28):
  the handler ran and completed without a reply when its worker should already have been stopped,
  suggesting a race around the stopped worker; it passed five times since. Measure it with
  `make umpire-repeat-run` before fixing.
- **A multi-instance Case with a retryable handler error** (fn-90.8): its controller would run
  `pending-attempts` before the moved scheduled read. No such Case exists; the first one needs the
  order settled.

## Gate baselines

Measured 2026-09-27 and 2026-09-28 on a macOS host, Lean 4.32.0 (fn-88.12, fn-90, fn-91, fn-89 and fn-94 closeouts):

| Gate | Result |
| ---- | ------ |
| `make umpire-check-regression` | exit 0, **45 passing live identities** (13 `TestTestpilotCanary*`), about 14 minutes |
| `make umpire-check-live-tests` | five consecutive passes at the fn-90 closeout, 45 identities, no retry, about 6 minutes each |
| `LEAN_NUM_THREADS=1 make lint-model` | exit 0: import graph with the authoring-path and search-backend-isolation rules and all three controlled violations, no declaration-linter findings, no `--wfail` builtin warnings (fn-88.7, Veil required, quiet host; 82 minutes single-threaded because the builtin step rebuilt the model). It needs about 6 GB and fails spuriously when another `lake build` shares `model/.lake` |
| `make umpire-check-lean-api` | exit 0; regenerates the Lean API into a temp directory and fails on any drift from the committed output |
| `make lint-code-fast` | 0 issues (it skips `testdata` and nested modules, as `./...` does) |
| `make umpire-check-plan-index` | valid |
| CI `Umpire` workflow (cold `.lake`, Veil required) | run 36393502944: portability 24m39s of 40, canary 17m03s of 30, both green |
| `TestTestpilotOwnsCaseProtocolAndRuntime` | passes: the Temporal Drivers reach Program ceilings through `testpilot.WithinProgramCeiling`, never `internal/ir` or `internal/execution` |

Intermittent-failure rates at the fn-90 closeout (7d0997b990), all zero: the umpire-run test 0/50,
the pair, caller async, caller fixture-name and worker-outage tests 0/200 each, and the two caller
tests in one process 0/52 each. The fn-90.3 and fn-90.7 task receipts carry the per-loop records.

**`make lint-code` under-reports when the disk is low.** golangci-lint aborts with
`no space left on device (typecheck)` and still exits with a count. Run `go clean -cache` before
trusting this gate. **`make umpire-check-retired-vocabulary`** is the slowest offline gate, about
twenty minutes in a cloud session.

**Environment notes.** The model builds on Lean 4.32.0, pinned by `model/lean-toolchain` and
`mise.toml`. Run Lean through `make` targets or `mise exec -- lake`: a shell whose `PATH` still
carries another Lean install builds `model/.lake` under the wrong toolchain. On macOS, mise's lean4
`clang` shadows the system one and fails cgo builds with `stddef.h not found`; `mise.toml` sources
`develop/mise-env.sh`, which sets `CC=/usr/bin/clang` unless `CC` is set, and the Makefile exports
the xcrun clang, so `make` targets need no manual `CC`. A direct `go test` or `go vet` in a shell
without the mise hook (agent shells) needs `CC=/usr/bin/clang`. Live tests need a physical `TMPDIR`
(`TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P)`; the default macOS path traverses the `/var` symlink).
`go vet -tags test_dep` is not the gate: the live suite builds with `-tags 'test_dep integration'`.
A fresh clone lacks the git-ignored `proto/api.binpb` the model build reads; `make
proto/api.binpb` makes it. In a cloud session, `mise` is a passthrough shim, so
`/opt/temporal-toolchain/*/bin` must be on `PATH` before any `make` target that uses `lake` or
`protoc`; warm the Go module cache (`go mod download`) before the first gate and discard the
`go.sum` hashes that download adds; and `git fetch origin main` gives the branch a name but not a
merge base, because the clone is shallow. A fresh clone cannot change task status: runtime state
lives in the clone's `.git` common-dir, so every task reads `todo` and `start`, `done` and
`spec close` refuse until they are replayed there.

**Repeating a live test.** `make umpire-repeat-run SELECT='<regex>' COUNT=<n> MODE=process|in-process
[RECORD=<file.jsonl>]` runs a `^TestTestpilot...` selection `n` times, one process per iteration
or one process with `-count=n`, and prints per-test and per-signature failure rates with 95%
intervals from the `TESTPILOT-SIGNATURE` lines failing tests print; `UMPIRE_REPEAT_RUN_DIR`
captures each Run. It stops when the tests' inputs change mid-loop. It is a diagnostic, not a gate.

## Deferred and superseded

**fn-79 — Nexus operation cancellation:** [spec](../.flow/specs/fn-79-deferred-nexus-operation-cancellation.md).
Includes former fn-78.5/.8/.9 cancellation scope and fn-77’s cancellation qualification. Resume only
on an explicit user request; autonomous delivery approval does not override this deferral. Generic
fn-78 syntax/monitoring/qualification and fn-70 remain deliverable without it. Existing shutdown
and bounded cleanup cancellation behavior stays in scope. On resume it re-plans on fn-85
(entities, actions, sets) and takes the cancel Query and the Testpilot cancel instructions that
fn-85 left out; its re-planning note in Flow lists them, and its `## Scope` records the
cancellation-race behavior fn-86 .5 deleted with the Race prototype (2026-09-20).

**fn-70 — Scheduled canary proof of concept:** [spec](../.flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md).
Deferred by user decision; resume on an explicit user request. Nothing in the delivery queue
depends on it. A local/development prototype of a second model consumer under `tools/canary`
(manual check selection, a scheduled Workflow each minute, Activity-owned Testpilot execution,
bounded results); it does not depend on fn-26 or fn-29 and authorizes no production deployment.
On resume it re-plans on fn-85's canary set, inherits fn-80's `temporal.DeriveProfile`, consumes
fn-83's `provision` package and `umpire-run`, and reuses the test-local `bindCase` in
`tests/testpilot_run_case_test.go` with `runCapturedCase` in `tests/testpilot_signature_test.go`
(fn-94 removed the uncalled `runCase`). Its nine tasks had a SHIP plan review before those changes.

These entries are outside the delivery queue and are not prerequisites for it.

| Deferred spec                                                             | Revisit when                                                                                                   |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| [fn-60](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md)      | Optional handwritten canonical JSON consolidation becomes worth prioritizing; it has no downstream dependency. |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the proven model family.                                                |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md)         | Re-scope after fn-88: its toolchain gate is answered by fn-88's probes, and only its symbolic-checker scope remains. |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md)    | A verification receipt/profile platform is justified.                                                          |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md)  | A second verification backend is justified; caller closure remains historical.                                 |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md)        | Real Claim Assessment evidence supports release governance.                                                    |

fn-88 consumes Veil's concrete checker as a Search engine and leaves the symbolic verification
path, fn-23, fn-24 and fn-25, deferred as above; fn-24 and fn-25 now depend on fn-88.

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Testpilot proposal with an independent oracle.
