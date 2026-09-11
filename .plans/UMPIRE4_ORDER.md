# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

### 1. Finish the Model command surface — fn-83

[fn-83 — Author a live Case from a Model file](../.flow/specs/fn-83-author-a-live-case-from-a-model-file.md),
from the 2026-09-09 gap analysis of [UMPIRE4_VISION](UMPIRE4_VISION.md). Its goal was that a
Model file plus one `case` block is a running, checked-in, deterministic live test. On 2026-09-10
the user replaced the `case` direction (see fn-85 below), so fn-83 now finishes the command surface
and hands the rest to fn-85.

**Landed.** A generic `Umpire.Case.Producer` (.1); the `nexusOperation` and `workflow` realization
templates (.2); the `case` command, the Lean Case registry and `umpire-case --list/--render` (.3);
the generator reading that registry with a live test named by its fixture alone, which landed
before .4 was blocked; a provisioning package and the `umpire-run` CLI (.7); every generated
Testpilot JSON fixture stored indented for review (.9); the five Model commands moved into
`Umpire.Command`, with AUT-07a drafted (.10); Query-declared Known Gaps replacing the hard-coded
ones (.11); the `enum` command and a generated Setup (.12).

**Also landed (2026-09-10).** .13 made Facts optional so a Model stops restating its states; .14
reports authoring mistakes in `property`, `scenario` and `query` at their line while the Model file
compiles; .15 respelled the commands under one reading rule (column-0 declaration, indented
`word:` key, derived labels). .14 and .15 skip the `case` command. **Eleven of seventeen tasks are
done and nothing is open** — the six below are blocked, so fn-83 is paused on fn-85, not stalled.

**Blocked 2026-09-10, superseded by fn-85.** .4's remainder, .5 (fault lines against template
hooks), .6 (sync Nexus translation), .8 (tutorial), .16 (fixture name from the `case` name) and .17
(template arguments). fn-85's final task closes them and names where each concern went, so fn-83
itself closes only after fn-85. Its former spec-level dependents were re-anchored on 2026-09-10:
fn-33, fn-29, fn-70 and fn-79 now depend on fn-85, whose sets and Realization replace the `case`
block they consumed; fn-84 and fn-46 dropped the dependency. Their tasks that touched fn-83's open
work carried a dated note to start after fn-83 .15, because Flow cannot record a cross-spec task
dependency; .15 is done, so those notes are satisfied.

AUT-07a (.10) and the AUT-09 amendment .8 would have drafted are the GOV-02 items fn-83 leaves; the
amendment moves to fn-85's rule drafts.

### 2. Deepen five shallow module clusters in Umpire and Testpilot — fn-84

[fn-84 — Deepen five shallow module clusters in Umpire and Testpilot](../.flow/specs/fn-84-deepen-five-shallow-module-clusters-in.md),
from the 2026-09-09 architecture review of `model/Umpire`, `model/Testpilot`, `common/testing/testpilot`,
`tests/testcore/testpilot` and `tools/umpire`. Five independent scans, one per area, each nominated
one deepening: a shallow module cluster where one decision is spelled out in two to six places and
kept in agreement by hand becomes one deep module with a small interface that callers and tests both
cross. The spec carries exactly those five, one task each, in the order the scans recommend:

| Task | Deepening | Pin |
| ---- | --------- | --- |
| .1 | one `outage` module owned by the worker registry; `Validate` and `Open` share one `OutagePlan` | fault suite and live worker-outage tests, unchanged Verdicts |
| .2 | one Driver-contract leaf package replacing the 14-declaration facade mirror and its four adapters | conformance and facade tests compile unedited |
| .3 | `Umpire.Search.admit` owning the Property, Scenario, Query, view and search chain | `PlanResult` bytes, goldens and fingerprints unchanged |
| .4 | the offline Evidence structure module returns a verdict per audience instead of findings two callers re-judge | mutation suite diagnostics byte-identical |
| .5 | `Umpire.Case.Projection.lower` derives the Contract rule from the checked field Property | Case fixtures byte-identical under ART-11 |

No behavior changes; every task records an equivalence pin before it moves code and closes with
`make umpire-check-regression`. The plan review is SHIP after one fix round.

**Next to run, before fn-87 and fn-85; nothing blocks it.** fn-82 closed on 2026-09-10 and its renames in
all five areas have landed. Task .5 shares its seam with `Umpire.Case.Producer`, and .3's admission
chain now sits in `Umpire.Command` after fn-83 .10 moved it; both tasks re-read those owners before
starting. fn-84 goes before fn-85 because every task is behavior-preserving with byte-identical
fixture and golden pins, which are cheapest to hold before fn-85 replaces the async-Nexus fixture
and the Producer's Program assembly, and because fn-85 builds on `Umpire.Search.admit` (.3). The
spec-level dependency on fn-83 was removed on 2026-09-10, and the note on .3 and .5 to start after
fn-83 .15 is satisfied now that .15 is done. fn-22 and fn-33 depend on this
spec in turn: fn-33's exploration bridge sits on the search-view transport sites .3 replaces, and
fn-22's promotion path consumes `search`, which .3 keeps public. Dispatch is serial in task order
because every pair of tasks shares a documentation or test-root file.

Boundaries: no renames beyond what the new modules need, no change to the evaluation budget
(`CONSIDER(umpire)` on the cubic reservation stays separate), to delivery routing, to the
generators' publication tooling, to Property evaluation combinators, or to `Umpire.Json` sealing.
EVD-20 is approved (2026-09-10, GOV-02); the MOD-14 restatement .2 drafts stays pending.
Candidates the scans surfaced and the spec declined are listed in its Decision Context.

### 3. Tighten the Testpilot protocol — fn-87

[fn-87 — Tighten the Testpilot protocol: glossary names, one expression language, structure by concept](../.flow/specs/fn-87-tighten-the-testpilot-protocol-glossary.md),
from the 2026-09-11 review of the Testpilot protocol (855 lines in seven files). The review found
thirty issues: three expression languages for the same operators (fourteen duplicated Program and
Contract messages plus a correlated predicate language), names that break SEM-19 (`RunStatus` for
Run disposition, correlated `clauses` of type `CorrelatedRule`, "projection" and "capability" each
naming two concepts), per-kind Run Event fields for faults, duplicate encodings (opaque handles,
`natural`, capture types, three binding shapes, a nested `version`), an enum no wire message
references, a layout that spreads the correlated capability over three files, and no written
checklist for extending the protocol.

The spec renames to glossary words, restructures into one file per concept with every message
documented, replaces the three expression languages with one `Expression` checked per context at
preparation, gives Run Events a payload oneof, removes the duplicates, and adds an extension
checklist.

On 2026-09-11, with breaking changes allowed, it also took on the defaults and readability the
review of `typed-nexus-case.json` called for. That fixture is 2,025 lines, and 77% of its bytes are
21 parameterized model values of about 11,000 characters each; every instruction guard in the six
checked-in Cases is "every dependency succeeded"; each Case writes 24 to 83 limit fields. So
instructions run in entrypoint order unless `after:` or a guard says otherwise; the environment
bindings, activation reservations and outcome fields are derived; resource ceilings move to the
Profile, keeping only meaningful bounds in the Case (an SEM-16 amendment); provenance becomes
structured rows, with Case-local names and short model values mapped to Definition IDs and
fingerprints there; fixtures print fields in declaration order with string field paths and named
enums; a comparison with an absent operand is false, so presence checks disappear; and Run-only
messages leave the Case's import closure.

Verdicts do not change, except where the absent-operand rule is checked against every conformance
class and live test: R1 and R2 land first as mechanical changes, proven by an equivalence test that
maps every pre-migration fixture to the new protocol under a declared mapping. The wire has no
compatibility promise (`buf` breaking ignores the package), so there is no `v2` and no shim. New
capabilities stay with their owners: typed worker instructions and per-Case observation declarations
with fn-85 R10, cancel with fn-79, correlated transitions over structured machine state with fn-85.

**Depends on fn-84**; **fn-85 depends on it** (both recorded in Flow), so fn-85's instructions and
fn-86's migrated Cases are authored on the final shapes. Needs a plan review and a task breakdown.

### 4. Model side effects as typed actions and run query sets — fn-85

[fn-85 — Model side effects as typed actions and run query sets](../.flow/specs/fn-85-model-side-effects-as-typed-actions-and.md),
from the 2026-09-10 design session recorded in
[the Nexus design specimen](../model/Temporal/Feature/Nexus/DESIGN.md). The `case` block kept every
side effect (the RPC, its request fields, the handler's reply, timeouts, cancel) in a hand-written
Program template, so a Model could not tell a retryable handler error from a non-retryable one and a
Property could not read the fields that make the difference. The design surveyed the server's Nexus
operation behavior and all 16 Nexus functional test files, which the Model must eventually express.

The spec makes three things true. Side effects are part of the Model: entities with identity,
actions a party performs with typed inputs grouped into classes and an example per class, machines
that keep each entity's state and rows, and observations that confirm each row. A Temporal-owned
realization in `Temporal.Case` binds them to RPCs, Testpilot instructions, history events and
dynamic config. Queries are grouped into sets by purpose: a set binds each party to `driven` or
`observed`, and a functional set compiles to one Case per Query. And the Nexus caller-side operation
runs this way end to end: a product machine and a protocol machine that refines it, with a
functional set of seven Queries translated from the Nexus functional tests, each run under both the
HSM and CHASM implementations.

The early proof point rebuilds today's async-Nexus Case from the new abstractions and must match
its fixture with identities masked before any Testpilot protocol change. Worker instructions then
carry the Temporal API messages the actions' schemas name (the Nexus schedule command's attributes
with its three timeouts, a `StartOperationResponse` or `HandlerError` reply, a completion payload or
failure) instead of a Testpilot field per server option, and a Case declares each observation once
for both its Program and its Contract, which gives the retry Query's attempt count its read source.
Canary and exploratory sets are admitted with their
coverage targets enumerated; running them stays in fn-70, fn-29 and fn-33. Whole-Program templates
and the `case` command are removed.

**Needs a plan review and a task breakdown**; the spec has no tasks yet. Flow records its
dependencies on fn-84 and fn-87; the fn-83 tasks it builds on (.13, .14, .15) are done. It closes fn-83's six
blocked tasks as superseded.

**Cancellation stays deferred.** The design's cancel Query and its Testpilot instructions overlap
fn-79, which resumes only on an explicit user request, so on 2026-09-10 they moved out of fn-85 into
fn-79's re-planning note.

Boundaries: no composition (update-, query- or activity-backed handlers, several callers), no
reset, no visibility or standalone operations, no metrics or spans as observations, no endpoint
registry, matching or cross-cluster topology, no HTTP transport fault kind, no schema interface, no
change to the hand-written Models (fn-86 retires them).

### 5. Retire hand-written Models: one authoring path through the commands — fn-86

[fn-86 — Retire hand-written Models: one authoring path through the commands](../.flow/specs/fn-86-retire-hand-written-models-one.md),
from the 2026-09-10 decision to align completely on the developer-facing commands. After fn-85 the
tree still carries about 7,000 lines that build Umpire records or Testpilot Cases directly in Lean:
the two typed Nexus examples, the Race, Lifecycle, Operations and Experimental Nexus models, the
worker-outage and get-system-info Cases with no Model, and Umpire's Switch example. AUT-08 still
names that path as an expert alternative.

The spec makes the commands the only authoring path for feature Models and gives every piece of
covered behavior a destination before anything is deleted. An inventory lists each hand-written
module with the Properties, goldens, fixtures, live tests and tools that read it. The commands gain
field relations (a Property comparing action input, action result and observation fields through
their schemas), and the typed examples migrate onto them with their crossed-pairing mutations
intact. The worker-outage and get-system-info Cases become command Models, the Switch example is
re-authored, and the Race and Experimental models are deleted with their behavior written into fn-79
and fn-33. A `lint-model` import rule then keeps feature Models from importing Umpire's authoring
owners except through `Umpire.Command`, and GOV-02 drafts remove AUT-08's expert alternative.

Two decisions frame it: the typed examples migrate rather than being deleted first, because they
carry the only field-level Property coverage; and `Temporal.System.Nexus` stays as the single
exception, because it is the spec's only Feature-to-System Implementation Link (SEM-08, MOD-04).
Testpilot conformance and synthetic Cases stay, since they test the runtime.

**Depends on fn-85** (recorded in Flow). The early proof point expresses the typed unary Property as
a field relation that lowers to the same Contract field reads before any deletion. Needs a plan
review and a task breakdown.

**Follow-up after fn-86, not yet a spec:** one Contract monitor declared per entity and instantiated
per instance, replacing the per-instance rule copies Producers emit today (the typed Nexus Case
carries its operation rules twice). It changes how the runtime evaluates rules, so it gets its own
spec once every Case comes from the commands.

### Additional open specs

These remain open in Flow and are outside the first-canary critical path.

| Spec                                                                                              | Dependencies | Next action                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-46 — Lean module impact index](../.flow/specs/fn-46-export-lean-model-module-impact-index.md) | fn-45 | Refreshed three-task plan is SHIP against current model owners. fn-83 .15 is done, so task .1 is free to start; refresh the module rows for `Umpire.Command` (fn-83 .10) and, once planned, fn-85's new modules. Deliver the shared loader, pure dependency/facade/test index, and opt-in export/check commands. |

## Gate baselines

Measured 2026-09-10 after the upstream merge, on a host with disk headroom:

| Gate | Baseline |
| ---- | -------- |
| `make umpire-check-regression` | exit 0 — 571 Lean jobs, eight conformance checks, nine passing live identities |
| `make lint-model` | 163, all in generated `Temporal/API/Proto.lean`; `Umpire.Lint` and `Shared` clean |
| `make lint-code GOLANGCI_LINT_FIX=false` | **161** (exit 2, inherited red) |
| `go vet -tags test_dep ./...` | 15 pre-existing diagnostics |
| `go run ./tools/planindex` | ~49, pre-existing `.plans` registration drift |

**`make lint-code` under-reports when the disk is low.** Earlier runs recorded 128 and were believed
for several specs. golangci-lint aborts with `no space left on device (typecheck)` and still exits
with a count — `Issues before processing: 11800, after processing: 1` in the failing case against
`14507 -> 161` in a healthy one. Run `go clean -cache` before trusting this gate, and treat any
number below 161 as a truncated run rather than an improvement.

Live tests need `CC=/usr/bin/cc` (mise's lean4 clang shadows the toolchain; cgo fails with
`stddef.h not found`) and a physical `TMPDIR` (`TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P)`; the
default macOS path traverses the `/var` symlink). `go vet -tags test_dep` is NOT the gate — the live
suite builds with `-tags 'test_dep integration'`, which compiles strictly more files.

## Downstream delivery

Each spec needs a fresh Testpilot plan review before implementation. Prior reviews of the
retired execution architecture do not approve the rewritten plans. Completed dependencies do not
block replanning or execution.

| Spec                                                                                              | Dependencies                       | Next action                                                                                                                                                                                                                                   |
| ------------------------------------------------------------------------------------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-33 — Bounded exploration](../.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md)    | fn-40, fn-64, fn-69, fn-84, and **fn-85** | Re-plan on fn-85: an exploratory set's coverage goal (rows, result classes, members of claimed input classes) is the candidate space, and a divergent class member is the counterexample Promotion keeps. Then review whole-Case candidates, serial coordination through Testpilot, lost iterations, semantic coverage, and bounded 10x behavior. |
| [fn-22 — Replay and reduction](../.flow/specs/fn-22-deterministic-replay-semantic.md)             | fn-5, fn-64, and fn-69             | Resolve **MAJOR_RETHINK** before implementation: separate exact candidate identity from Contract-relative violation equivalence, prove the negative Case before reduction, and retain explicit offline semantic replay and checked promotion. |
| [fn-26 — Qualification receipts](../.flow/specs/fn-26-local-qualification-receipts-and-staged.md) | fn-48, fn-64, and fn-69            | Review offline Testpilot Case/Profile/Run/Verdict admission, receipt multiplicity, and idempotent publication. Assessment must never create or replay a Run.                                                                                  |
| [fn-29 — Production canary](../.flow/specs/fn-29-bounded-production-canary-execution-and.md)      | **fn-26**; fn-48, fn-64, fn-69, fn-83 .7, and fn-85 | After fn-26 ships, review external policy and credentials, serial Testpilot Runs, leases, lost Runs, reconciliation without redispatch, and publication. It runs an fn-85 canary set and consumes fn-83's provisioning package. |

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire.

## Deferred and superseded

**fn-79 — Nexus operation cancellation:** [spec](../.flow/specs/fn-79-deferred-nexus-operation-cancellation.md).
Includes former fn-78.5/.8/.9 cancellation scope and fn-77’s cancellation qualification. Resume only
on an explicit user request; autonomous delivery approval does not override this deferral. Generic
fn-78 syntax/monitoring/qualification and fn-70 remain deliverable without it. Existing shutdown
and bounded cleanup cancellation behavior stays in scope. On resume it re-plans on fn-85
(entities, actions, sets) and takes the cancel Query and the Testpilot cancel instructions that
fn-85 left out; its re-planning note in Flow lists them.


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

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Testpilot proposal with an independent oracle.
