# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

fn-88, fn-89 and fn-92 to fn-94 are queued. fn-83, fn-84, fn-87, fn-85, fn-86, fn-46, fn-33,
fn-22, fn-26, fn-29, fn-91 and fn-90 are delivered, each with SHIP implementation and completion
reviews; their task receipts in `.flow/` and the git history carry the details. fn-91 (2026-09-27)
renamed the Driver seam's Go to the opaque handle family and holds the eight retired names in the
vocabulary gate. fn-90 (2026-09-27) re-measured the three intermittent live Testpilot failures with
the `make umpire-repeat-run` harness; none reproduced on today's identities. It fixed two causes.
The umpire-run test now runs on a cluster with the system worker, so the `--create` namespace
deletion finishes and the test asserts it. The pair Case's controller read the scheduled events
before the second operation was scheduled; it now reads them after the workflow closes. The
evidence-ordering and async-Nexus failures closed as not reproduced on the successors. No
quarantine exists.

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire.

### Delivery queue

1. **fn-88 — Veil concrete checker as the Umpire search engine**
   ([spec](../.flow/specs/fn-88-veil-concrete-checker-as-the-umpire.md)). Makes Veil's concrete
   model-checker library the engine behind `Umpire.Search`, consuming the `FiniteTable` the
   `machine` command already enumerates, with Properties and Scenarios lowered to monitor automata
   through the lowering the Case Producer uses. Its first task is a developer-machine compatibility
   probe of the checker library under the repository toolchain; `adopt` continues, and
   `defer-incompatible` closes the rest as not applicable and hands the engine question to a
   separate `FiniteTable` to TLA+ exporter spec. Today's `Search.lean` stays as the frozen reference
   engine and differential oracle. The reasoning is sections 1 and 6 of
   [UMPIRE4_DIRECTION](UMPIRE4_DIRECTION.md). It adopts no Veil DSL and no SMT path, and it neither
   runs fn-23's sandboxed gate nor resumes fn-24 or fn-25.
2. **fn-89 — One Contract Rule per entity**
   ([spec](../.flow/specs/fn-89-one-contract-rule-per-entity.md)). It waited for fn-90, since both
   touch the Nexus pair Case; fn-90 is delivered, and its pair Case program change is in place. A Rule over N instances crosses the wire once with its per-instance values
   instead of N lowered copies; Verdicts stay identical, pinned by a differential test, and only the
   multi-instance pair fixture changes.
3. **fn-92 — Compose entity machines into one Model**
   ([spec](../.flow/specs/fn-92-compose-entity-machines-into-one-system.md)). Adds a `compose`
   command that builds one Model from entity machines with declared action synchronization over a
   reachable-state enumeration, `restrict:` and `extend:` keys that derive machines from a source
   table, and `Workflow` and `Worker` entity modules the Start and Outage use cases share; hosts the
   first cross-entity claims as `verify` Queries. Version one realizes no Case over a composition and
   leaves the caller module, its fixtures, and the canary's pinned Case identity untouched; moving
   the operation entity is the named follow-up. Depends on fn-88, fn-89 and fn-90.
4. **fn-93 — Simplify the Lean model**
   ([spec](../.flow/specs/fn-93-simplify-the-lean-model.md)), after everything above. A
   simplification campaign over the handwritten code in `model/`: about 88,100 of its 128,630 Lean
   lines, measured 2026-09-26. Purely generated code is out of scope. It starts by fixing the defects the investigation found. Production Models get the
   `schema:` and `evidence:` checks they skip today, one checker replaces the `#print axioms` pins
   that assert nothing, and test modules no root builds get wired in. It then derives what is
   written out by hand: enum wire names, keyword parsing, the command registry, one diagnostic
   type and one JSON helper set. Every golden, Case fixture, Definition ID and Behavior Fingerprint
   stays byte-identical. Behind owner decisions recorded per task, it deletes code that retired
   rules left behind (`Umpire.Variations`, the offline Evidence evaluation chain, and the Run,
   Evidence, Result and Set artifacts, the guarded Property forms, the Bool/Prop denotation copies,
   and the Lean-side correlated Monitors). It also reuses core Lean where hand-rolled helpers
   repeat, and cuts `model/` Markdown to one README and one ARCHITECTURE. Authoring gains shorthand without losing any
   construct. It absorbs fn-60's re-scoped aim, and fn-60 becomes superseded when fn-93 closes.
5. **fn-94 — Simplify the Testpilot Go runtime**
   ([spec](../.flow/specs/fn-94-simplify-the-testpilot-go-runtime.md)), after fn-89, fn-90 and
   fn-91. It is independent of fn-93 and may run beside it. It is the Go counterpart of fn-93, over
   the 42,067 handwritten lines of `common/testing/testpilot`, `tests/testcore/testpilot`, the live
   tests and the hand-written `.proto`. Generated code is out of scope.
   - It first settles whether the `initial_state_fields` and `prior_fields` wire fields should be
     read or removed: Lean emits them, and Go never reads or validates them.
   - It removes the residue of the retired untyped Nexus path and the other dead and test-only code.
   - It validates admitted data once rather than again in each Driver, which follows from SEM-16.
   - It shares the copied primitives, one opcode table and one workflow binding type.
   - It consolidates the test fixtures and fakes.

   Behind a per-arm owner decision, it removes protocol arms that no producer emits. The corpus,
   Driver identity bytes, route wire bytes and the live identity count stay as they are.

### Carried forward, not specs

- **A field path through a repeated field** (deferred from fn-86 R2): selecting the element the
  observation correlates to the row's entity instance. `Temporal.Case.FieldPath` rejects every
  repeated-field path today.
- **The canary CI job's cold Lean build** (from fn-29): the `canary` job in
  `.github/workflows/umpire.yml` is the first to build Lean targets (`canary-check-case`,
  `umpire-check-evaluation-profiles`) with `cache: false` inside a 30-minute timeout; confirm the
  first CI run fits.
- **The worker ignores the timeouts a Case declares on Finish and NexusHandlerReply** (from
  fn-90.6): the workflow interpreter runs Finish, and the handler interpreter returns its reply,
  with no timeout, so the 5000 ms bounds on `finish-workflow` and `respond-async` bind nothing.
  A Case-authority gap.
- **Run capture records no workflow or handler instruction events** (from fn-90.3 and .6): a
  recorded Run holds controller events only, so a bounded worker step's own latency cannot be read
  from it. Sizing such a bound needs those events or a Driver-side timing field.
- **Agent shells do not apply mise's `CC` fix** (from fn-90.5 to .7): with no mise shell hook, the
  `_.source` of `develop/mise-env.sh` does not run, and cgo builds fail with `stddef.h not found`
  until `CC=/usr/bin/clang` is set by hand.
- **The server dates a completion-before-start started event in local time labelled UTC** (from
  fn-90.7): the synthesized `NEXUS_OPERATION_STARTED` carried `2026-09-26T22:44:46Z` (PDT wall
  time, whole seconds) between events at `05:44:46.xZ`. Testpilot never reads `EventTime`, so no
  Verdict depends on it; an upstream issue is the next step.
- **A multi-instance Case with a retryable handler error** (from fn-90.8): its controller would run
  `pending-attempts` before the scheduled read that fn-90.8 moved after the workflow closes. No
  such Case exists today; the first one needs that order settled.

## Gate baselines

Re-measured 2026-09-20 on a four-core, 16 GB cloud session at the fn-86 closeout:

| Gate | This session |
| ---- | ------------ |
| `make umpire-check-regression` | exit 0 after `go clean -cache` -- 590 Lean jobs, the offline checks, **29 passing live identities** |
| `make lint-model` | exit 0, measured 2026-09-26 on a macOS host with `LEAN_NUM_THREADS=1`: the import graph passes with the authoring-path rule and both controlled violations asserted, the declaration linters report no findings, and the final `lake --wfail lint --builtin-only` step reports no warnings. The 163 generated findings were fixed in `umpire-gen-lean-api`, which writes `Method`'s `DecidableEq` and `Repr` instances without instances of its phantom payload types, and declares fieldless messages with no placeholder field and fieldless oneof arms as nullary constructors. The 24 builtin warnings in handwritten modules were fixed at their source: `enum` registers an unused-variables ignore function so its constructor fields are spared as `inductive`'s are, a `machine` attaches its doc comment to the declared Model, `Umpire/Command/Syntax.lean` uses `Level.zero` and `String.trimAscii`, and `Umpire/Value/Encoding.lean` unfolds `encodeNat` with `rw` rather than `simp` |
| `make lint-code` | 0 issues over the changed packages (`GOLANGCI_LINT_BASE_REV=9484405 make lint-code-fast`); the full `make lint-code` is not measurable in a shallow clone with no `main` merge base |

Measured 2026-09-26 on a macOS host at the fn-29 closeout, not a re-measured baseline:
`make umpire-check-regression` exit 0 with **45 passing live identities** (13 of them
`TestTestpilotCanary*`); `make lint-code-fast` red on findings outside the canary (the typecheck
error of a deliberately broken `tools/umpire` testdata fixture and staticcheck findings in untouched
`tests/*.go`); `LEAN_NUM_THREADS=1 make lint-model` also reported an unused `[BEq α]` in
`Umpire/Command/Refinement.lean` (present since 2026-09-19 and missed by the baseline above), since
removed, and the 163 generated findings and the 24 `--wfail` warnings of its builtin-lint step have since been fixed, so it exits 0; `make umpire-check-plan-index` passes
after `.plans/index.json` was resynced with Flow the same day.

**`make lint-code` under-reports when the disk is low.** golangci-lint aborts with
`no space left on device (typecheck)` and still exits with a count — `Issues before processing:
11800, after processing: 1` in the failing case against `14507 -> 161` in a healthy one. Run
`go clean -cache` before trusting this gate.

**fn-90 closeout, 2026-09-27, macOS host, commit 7d0997b990.** The per-identity loops of
`make umpire-repeat-run`, at the fn-90.3 baseline counts, all ran with zero failures. Before
(b9bb1a58ad) and after, 95% Clopper-Pearson upper bounds:

| Identity | Mode | Before | After |
| -------- | ---- | ------ | ----- |
| `TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint` | process | 0/50 (7.1%) | 0/50 (7.1%) |
| `TestTestpilotNexusPairCase` | process | 0/200 (1.8%) | 0/200 (1.8%) |
| `TestTestpilotNexusCallerAsyncCompletion` | process | 0/200 (1.8%) | 0/200 (1.8%) |
| `TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone` | process | 0/200 (1.8%) | 0/200 (1.8%) |
| `TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone` | process | 0/200 (1.8%) | 0/200 (1.8%) |
| the two caller tests, `-count=4` | in-process | 0/52 each (6.9%) | 0/52 each (6.9%) |

Between the two, a loop at be465e5966 found one new pair Case signature (1/200, Run INCOMPLETE,
`observe_failed`: unauthorized operation transition), which fn-90.8 fixed. Host load stayed at
about 2 to 9. Five consecutive `make umpire-check-live-tests` runs then passed with 45 passing
identities, an empty failure set and no retry, in about 6 minutes each. The umpire-run test now
takes about 4 s in the gate, against about 35 s before fn-90.4 removed its 30 s teardown wait. One
`make umpire-check-regression` followed, exit 0 in about 14 minutes with the same 45 live identities.

**`make umpire-check-retired-vocabulary` is the slowest offline gate** by an order of magnitude
(about twenty minutes in a cloud session).

**Environment notes.** On macOS, mise's lean4 `clang` shadows the system one and fails cgo builds
with `stddef.h not found`; `mise.toml` sources `develop/mise-env.sh`, which sets `CC=/usr/bin/clang`
unless `CC` is already set, and the Makefile exports the xcrun clang, so `make` targets need no
manual `CC`. A direct `go test` or `go vet` in a shell with no mise hook still does (see the
carried-forward item).
Live tests still need a physical `TMPDIR`
(`TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P)`; the default macOS path traverses the `/var` symlink).
`go vet -tags test_dep` is not the gate: the live suite builds with `-tags 'test_dep integration'`,
which compiles strictly more files. In a cloud session, `mise` is a passthrough shim, so
`/opt/temporal-toolchain/*/bin` must be on `PATH` before any `make` target that uses `lake` or
`protoc`; warm the Go module cache (`go mod download`) before the first gate and discard the
`go.sum` hashes that download adds; and `git fetch origin main` gives the branch a name but not a
merge base, because the clone is shallow. A fresh clone cannot change task status: runtime state
lives in the clone's `.git` common-dir, so every task reads `todo` and `start`, `done` and
`spec close` refuse until they are replayed there.

**Repeating a live test.** `make umpire-repeat-run SELECT='<regex>' COUNT=<n> MODE=process|in-process
[RECORD=<file.jsonl>] [UMPIRE_REPEAT_FLAGS=...]` runs a `^TestTestpilot...` selection `n` times;
`RECORD` defaults to a timestamped file under `./.build/umpire-repeat/`. `process` starts one test
process per iteration; `in-process` runs one process with `-count=n`. It builds the test binary
once and appends one JSON record per iteration, with the failure signature each failing test
prints (`TESTPILOT-SIGNATURE`) and, through `UMPIRE_REPEAT_RUN_DIR`, the captured Runs. It then
prints per-test and per-signature failure rates with 95% intervals. It stops when the tests' inputs
change during a loop, and `umpire-repeat summarize` merges record files from one commit. It is a
diagnostic tool, not a gate: nothing runs it in CI. A `--create` namespace deletion finishes only
on a cluster that runs the system worker.

## Deferred and superseded

**fn-79 — Nexus operation cancellation:** [spec](../.flow/specs/fn-79-deferred-nexus-operation-cancellation.md).
Includes former fn-78.5/.8/.9 cancellation scope and fn-77’s cancellation qualification. Resume only
on an explicit user request; autonomous delivery approval does not override this deferral. Generic
fn-78 syntax/monitoring/qualification and fn-70 remain deliverable without it. Existing shutdown
and bounded cleanup cancellation behavior stays in scope. On resume it re-plans on fn-85
(entities, actions, sets) and takes the cancel Query and the Testpilot cancel instructions that
fn-85 left out; its re-planning note in Flow lists them, and its `## Scope` records the
cancellation-race behavior fn-86 .5 deleted with the Race prototype (2026-09-20).


**fn-70 — Scheduled canary proof of concept:** deferred by user decision; it was previously
queued after fn-78 as the second model consumer. Resume on an explicit user request. Nothing in
the delivery queue depends on it. On resume it inherits fn-80's `temporal.DeriveProfile`, so a
second model consumer no longer hand-writes a `ProfileSpec`. The bind-and-run helpers landed as
test-local `bindCase`/`runCase` in `tests/testpilot_run_case_test.go` rather than exported from the
fixture package, because exporting them would compile the whole server into a Quick command; a
canary test under `tests/` reuses them where they are. fn-83 extracts the provisioning that forced that
placement into `common/testing/testpilot/temporal/provision` and adds `umpire-run` (both landed);
on resume fn-70 consumes both and re-anchors its catalog entry on an fn-85 canary set, since fn-85
removes the `case` block that replaced the fn-68 Producer.

[Spec](../.flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md).
Retained scope on resume: fn-78 first, with fn-68, fn-71, fn-72, and fn-73 as transitive
prerequisites.
Nine tasks cover all ten requirements, with a SHIP plan review. Implementation follows fn-77
to serialize shared Producer edits; fn-77 is not a semantic prerequisite.

Implement the second consumer under `tools/canary`: manual check selection, a fresh scheduled
Workflow each minute, Activity-owned Testpilot execution, bounded results, and isolated repeated
measurements. Consume the Driver and binding interfaces delivered above; do not repeat their
implementation work. Retain the cross-consumer proof; fn-73 already owns the live proof that one
Case byte sequence runs against two environment bindings.

This is a local/development prototype. It does not depend on fn-26 or fn-29 and does not authorize
production deployment or replace fn-29's separately scoped production-canary design.


These entries are outside the delivery queue and are not prerequisites for it.

| Deferred spec                                                             | Revisit when                                                                                                   |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| [fn-60](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md)      | Optional handwritten canonical JSON consolidation becomes worth prioritizing; it has no downstream dependency. |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the proven model family.                                                |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md)         | Optional checker adoption becomes valuable.                                                                    |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md)    | A verification receipt/profile platform is justified.                                                          |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md)  | A second verification backend is justified; caller closure remains historical.                                 |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md)        | Real Claim Assessment evidence supports release governance.                                                    |

fn-88 consumes Veil's concrete checker as a Search engine and leaves the symbolic verification
path, fn-23, fn-24 and fn-25, deferred as above.

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Testpilot proposal with an independent oracle.
