# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

Status as of 2026-09-27. Delivered, each with SHIP implementation and completion reviews: fn-83,
fn-84, fn-87, fn-85, fn-86, fn-46, fn-33, fn-22, fn-26, fn-29, fn-91, fn-90, fn-89 and fn-94. Their
task receipts in `.flow/` and the git history carry the details. Delivered on 2026-09-27 and
2026-09-28:

- **fn-90** re-measured the three intermittent live Testpilot failures with the
  `make umpire-repeat-run` harness; none reproduced on today's identities. It fixed two causes:
  the umpire-run test now runs on a cluster with the system worker, so the `--create` namespace
  deletion finishes and is asserted, and the pair Case's controller reads the scheduled events
  after the workflow closes instead of before the second operation is scheduled. No quarantine
  exists.
- **fn-91** renamed the Driver seam's Go to the opaque handle family and holds the eight retired
  names in the vocabulary gate.
- **fn-89** sends a Rule over N instances once with its per-instance values instead of N lowered
  copies; Verdicts and admission equal the expansion (except that the authored surface is bounded
  as written), pinned by a differential test. `make umpire-rerecord-pinned-runs` refreshes every
  catalog-pinned recorded Run after a protocol change, machine-free.
- **fn-94** simplified the handwritten Testpilot Go runtime with no Run or Verdict change. The
  `BindingFingerprint` and route wire bytes are unchanged; removing the model-value expression
  reference moved the Driver catalog identity, and that commit re-recorded the pinned Runs. Go's
  correlated monitor now reads `initial_state_fields` and `prior_fields` as Lean's does. Dead and
  test-only code is gone, admitted data is validated once, one opcode table drives instruction
  binding, and copied primitives and test fakes are shared. `evidence_field_id` and
  `correlated_capture` stay, because Lean's correlated Producer emits both. The live identity count
  stayed at 45. The line-count floors were missed; the fn-94.17 receipt reports the measurement and
  the reasons.

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire.

### Delivery queue

1. **fn-88 — Veil concrete checker as the Umpire search engine**
   ([spec](../.flow/specs/fn-88-veil-concrete-checker-as-the-umpire.md)); 7 of 12 tasks done.
   Veil's concrete model-checker library becomes the engine behind `Umpire.Search`, over the
   `FiniteTable` the `machine` command enumerates, with Scenarios and Properties lowered to a
   product of a progress automaton and bounded monitors; today's traversal stays as the frozen
   `reference` backend and differential oracle. The first probe was `defer-incompatible` (Veil
   declares Lean 4.32.0); the spec was amended (R22) to allow moving the toolchain and an `IO`
   checker run during command elaboration, with every witness kernel-replayed and absence answers
   trusted from the checker under the differential test and a stated 64-bit state-hash assumption.
   The second probe decided `adopt`, and the model moved to Lean 4.32.0 (task .12). Done: the
   probes, the backend seam, the product state space, the monitor lowering and the GOV-02
   drafts. In progress: .5, the Veil Lake dependency, the adapter and the isolation lint rule.
   Remaining: .6 backend selection and the kernel replay gate, .9 the differential test and the
   Caller, Pair and three-instance pins, .10 Exploration and Replay keys with the golden re-pin,
   and .7 docs and the rollback drill. The reasoning is sections 1 and 6 of
   [UMPIRE4_DIRECTION](UMPIRE4_DIRECTION.md). It adopts no Veil DSL and no SMT path.
2. **fn-92 — Compose entity machines into one Model**
   ([spec](../.flow/specs/fn-92-compose-entity-machines-into-one-system.md)); planned, 6 tasks,
   after fn-88 (fn-89 is delivered). A `compose` command builds one Model from entity machines with declared
   action synchronization over a reachable-state enumeration; `restrict:` and `extend:` derive
   machines from a source table; `Workflow` and `Worker` entity modules are shared by the Start
   and Outage use cases; the first cross-entity claims are `verify` Queries. Version one realizes
   no Case over a composition and leaves the caller module, its fixtures and the canary's pinned
   Case identity untouched.
3. **fn-93 — Simplify the Lean model**
   ([spec](../.flow/specs/fn-93-simplify-the-lean-model.md)); planned, 43 tasks
   (Codex plan review SHIP), after fn-88, fn-89 and fn-92. A simplification campaign over the roughly 88,100 handwritten of `model/`'s 128,630
   Lean lines: fix the defects the investigation found, derive what is written out by hand, delete
   what retired rules left behind (per-task owner decisions), reuse core Lean (re-checked
   against Lean 4.32.0), and cut `model/` Markdown to one README and one ARCHITECTURE. Every
   golden, Case fixture, Definition ID and Behavior Fingerprint stays byte-identical. It absorbs
   fn-60's re-scoped aim.
### Carried forward, not specs

- **A field path through a repeated field** (deferred from fn-86 R2): selecting the element the
  observation correlates to the row's entity instance. `Temporal.Case.FieldPath` rejects every
  repeated-field path today.
- **The canary CI job's cold Lean build** (from fn-29): the `canary` job in
  `.github/workflows/umpire.yml` builds Lean targets with `cache: false` inside a 30-minute
  timeout; confirm the first CI run fits, and again once fn-88 adds Veil to the build.
- **The worker ignores the timeouts a Case declares on Finish and NexusHandlerReply** (fn-90.6):
  the 5000 ms bounds on `finish-workflow` and `respond-async` bind nothing. A Case-authority gap.
- **Run capture records no workflow or handler instruction events** (fn-90.3, .6): a bounded
  worker step's own latency cannot be read from a recorded Run.
- **The server dates a completion-before-start started event in local time labelled UTC**
  (fn-90.7): the synthesized `NEXUS_OPERATION_STARTED` carried PDT wall time marked `Z`. No Verdict
  reads `EventTime`; an upstream issue is the next step.
- **A multi-instance Case with a retryable handler error** (fn-90.8): its controller would run
  `pending-attempts` before the moved scheduled read. No such Case exists; the first one needs the
  order settled.

## Gate baselines

Measured 2026-09-27 on a macOS host, Lean 4.32.0 (fn-88.12, fn-90 and fn-91 closeouts):

| Gate | Result |
| ---- | ------ |
| `make umpire-check-regression` | exit 0, **45 passing live identities** (13 `TestTestpilotCanary*`), about 14 minutes |
| `make umpire-check-live-tests` | five consecutive passes at the fn-90 closeout, 45 identities, no retry, about 6 minutes each |
| `LEAN_NUM_THREADS=1 make lint-model` | exit 0: import graph with the authoring-path rule and both controlled violations, no declaration-linter findings, no `--wfail` builtin warnings. It needs about 6 GB and fails spuriously when another `lake build` shares `model/.lake` |
| `make umpire-check-lean-api` | exit 0; regenerates the Lean API into a temp directory and fails on any drift from the committed output |
| `make lint-code-fast` | 0 issues (it skips `testdata` and nested modules, as `./...` does) |
| `make umpire-check-plan-index` | valid |

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
