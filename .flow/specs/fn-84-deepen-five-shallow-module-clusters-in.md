# Deepen five shallow module clusters in Umpire and Testpilot

> HTML render lens: `.flow/artifacts/fn-84-deepen-five-shallow-module-clusters-in/spec.html` (local file, gitignored; open from disk) — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview

Five independent architecture scans of the Umpire Lean library and the Testpilot Go runtime, run
on 2026-09-09 at HEAD `ebb94a44e`, each returned one top deepening candidate. This spec carries
those five as one task each. Every candidate turns a shallow module cluster, where one decision is
spelled out in two to six places kept in agreement by hand, into one deep module with a small
interface that its callers and its tests both cross. No behavior changes. Every task names the pin
that proves it.

The five, in the order the scans recommend:

| Task | Area | Deepening |
| --- | --- | --- |
| .1 | Temporal worker Driver (Go) | one `outage` module owned by the worker registry |
| .2 | Testpilot facade (Go) | one Driver-contract leaf package replacing the hand-kept public/internal mirror |
| .3 | Umpire authoring core (Lean) | one `admit` operation owning the Model to Property, Scenario, Query, search-view, search chain |
| .4 | Umpire offline Evidence (Lean) | the structure module returns a verdict instead of a bag of findings two callers re-judge |
| .5 | Model-to-Case seam (Lean) | the Contract rule is derived from the checked field Property the way the correlated path already is |

## Goal & Context
<!-- scope: business -->

A Temporal engineer who adds one feature to any of these five areas today edits the same decision
in several files and then writes or updates a test whose only job is to assert the copies still
agree. The scans measured that cost on the current tree:

- The last seven worker-fault commits each touched the same four worker files, and one test exists
  only to check that `Validate` and `Open` reach the same admission verdict for a fault.
- The Testpilot facade carries 14 type declarations that are field-for-field copies of internal
  ones, plus five adapter structs whose roughly 30 methods convert a struct into an identical struct
  and call through. The copy has reached the comments.
- Six Temporal callers each define their own admission-error union over the same Umpire checkers,
  write the same seven-stage check-then-search chain by hand, and transport the search view across
  a proved Model equality with 18 hand-written `Eq.mpr (congrArg …)` terms.
- The offline Evidence structure module computes an evidence graph and returns 17 finding kinds; its
  two callers each hold a private copy of the finding-to-diagnostic mapping and of the precedence
  order, and causal reachability is written four times across the tree.
- The typed field Producers derive only the read path of a Contract rule from the Property. The
  comparison, the presence structure, the expected literal and the rule's state machine are
  re-authored per Producer, so a Property edit can leave the Contract comparing the old thing
  without any diagnostic.

The goal is locality and leverage. After this spec, each of those decisions lives in one module,
callers cross one interface, and the tests that reached into private maps, fields, or finding lists
assert through that interface instead.

Sequencing: fn-82 (vocabulary unification) renames or moves files in all five areas, including the
Go `Opcode` rename in the facade and the worker Driver's admission half. This spec depends on fn-82
at the spec level and is written in fn-82's vocabulary. Task .5 additionally depends on fn-83,
which introduces the generic `Umpire.Case.Producer` at the same seam; .5 deepens the module both
the generic Producer and the expert Producers call.

## Architecture & Data Models
<!-- scope: technical -->

Vocabulary in this section follows the codebase-design skill: a module has one interface and one
implementation; a module is deep when much behavior sits behind a small interface; a seam is where
the interface lives; an adapter satisfies an interface at a seam.

### .1 Worker outage module

Today a deliberate outage (a Fault) is four modules joined by two struct fields and a bool. The
worker Driver's definition-preparation resolves the fault's queue and writes `faultQueues` and
`hasFault` into the program definition; the Session reads them back to dispatch `InjectFault`; the
registry flips `group.stopped` under its lock in four places under three lock disciplines; a
test-only stop/resume facade exists beside the production transition path, and 14 of the 16 fault
tests drive the facade.

After: one `outage` module inside the worker package, owned by the registry, holds the
dedicated-group state machine and its admission. Definition preparation asks it once for an outage
plan; `Validate` and `Open` share that answer. The Session keeps only the effect-contract adapter
that turns a settle operation into an effect handle. The registry's `dedicated` grouping input
comes from the plan.

### .2 Driver-contract leaf package

Today MOD-14 (no production package outside Testpilot imports the internal execution package) is
satisfied by a hand-maintained copy of the whole Driver-facing contract in the facade plus a
translation family. After: one leaf package under the Testpilot module holds the Driver-facing
vocabulary and imports neither the execution package nor the IR package, so the facade and the
execution package both import it. The facade keeps the types that genuinely hide execution and IR
(`Driver`, `PreparedProgram`, `EntrypointPlan`, `InstructionPlan`, `Expression`) and re-exports the
leaf's types as Go type aliases so every existing Driver compiles unchanged. One thin adapter
remains for the two Driver methods that take a `PreparedProgram`.

The probe on 2026-09-09 confirmed the mirrored declaration ranges reference no IR symbol, so the
leaf can hold all of them.

### .3 Admitted Query

Today the authoring chain is seven staged public calls (`Property.check`, `Scenario.check`, Known
Gap canonicalization, `Query.check`, search-view construction, `search`, and the result match)
with the ordering constraints between them, the check contexts, and the search-view transport all
living in the callers. After: `Umpire.Search.admit` owns the chain. It lives in `Umpire.Search`
rather than `Umpire.Query` because Search already imports Query and the search view is Search's
type. It returns either one `AdmissionDiagnostic` union (the stage that rejected plus that stage's
typed error, byte-identical to what the stage returns today) or an `AdmittedQuery` that already
holds the checked Property, Scenario, Query and its `SearchView`. The search view is indexed by
the checked Model, and so is `AdmittedQuery`.

Two transports exist today and both are kept, moved inside `Umpire.Search`: `SearchView.retarget`
is the one blessed transport of a view across a proved Model equality, and
`AdmittedQuery.withQuery` re-pairs an admitted base Query's view with another checked Query over
the same Model. The second is what the Variations compiler, the Exploration engine and its
candidate universe, the Exploration session and Promotion do today, each with one view and many
Queries; they switch their `(query, view)` inputs to `AdmittedQuery`. The four `.checked`
constructors gain the same `:= by native_decide` auto-param `Umpire.model` already has, so callers
stop naming an `_isSome` theorem.

Promotion switches to an `AdmittedQuery` in this spec like the other one-view-many-Queries sites.
`search` and `SearchView` themselves stay public, because fn-22's replay and promotion work rests
on being able to search a checked Query against a view it holds; hiding them is outside this spec.

### .4 Evidence structure verdict

Today the structure module returns findings, an origin mode, closure expectations, closures and
links; the raw-bundle caller and the accepted-trace caller each map findings to their own failure
kind and each re-establish a precedence order. After: the structure module returns a verdict. The
one thing that genuinely differs between the callers, the audience (raw bundle or accepted trace),
becomes a parameter. Findings, closure expectations and precedence move behind the seam. The
callers name their failure kind for a fault the module identified.

### .5 Derived Contract rule

Today `Umpire.Case.Correlated.lower` takes a checked correlated Property and returns a Contract
lowering plus a correspondence certificate, and the Compiler admits it. The monitor-rule path has
no such module. After: `Umpire.Case.Projection` (the module fn-82 R5 already establishes for
reading declared Run values into model fields) gains `lower`, which takes the checked field
Property, the declared Observation and a realization record, and returns the Contract lowering,
the request-side coverage mapping the same Property implies, and a certificate. The realization
record carries exactly what the Property does not state and the two shipped rules need: the
request-side literal assignments the Program makes (today the workflow-type constant and the
per-operation name), the rule identity suffix, and the capture policy for a cross-event
comparison. The certificate states that every field the rule reads is a coordinate the Property
compares and every literal it compares against is a value the Program assigns. Both shipped rule
shapes, the two-transition safety rule of the typed unary Producer and the three-state capture
rule of the typed Nexus Producer, are adapters of this one lowering, the way the correlated path's
one lowering already serves its two Cases. `Compiler.compile` admits it beside the correlated
lowering. The three step-walkers over field coordinates (request coverage, observed read path,
projection coverage) become one private walker; their rejection lists differ per side today (the
request side accepts a keyed step the read side rejects, the projection side accepts a first
index), so the walker keeps per-side exceptions and lists them. No fifth Case submodule is added,
which keeps fn-82 R4's submodule set intact.

If the capture rule cannot be produced byte-identically from `lower` after the safety rule is,
the task narrows R5 to the safety-rule shape, records the capture rule as a `CONSIDER(umpire)`
beside the typed Nexus Producer, and says so in its summary; that narrowing is the only admitted
deviation and it is the task's to report, not to decide silently.

## API Contracts
<!-- scope: technical -->

Interfaces only. Bodies are the tasks' job.

```go
// .1  package worker (Go), owned by the registry
func PlanOutages(plans []testpilot.InstructionPlan, roles []testpilot.PreparedRole,
                 registrations map[string]bool) (OutagePlan, error)
func (OutagePlan) Requires() bool          // the registry's dedicated-group input
func (o *Outage) Begin(ctx context.Context, roleID string, kind FaultKind) (Settle, error)
type Settle func(ctx context.Context) error
func (o *Outage) Restore(ctx context.Context) error
func (o *Outage) Stopped(ctx context.Context) ([]string, error)   // takes the registry lock

// .2  package contract (Go), leaf: imports neither internal/execution nor internal/ir
type Coordinate, DriverIdentity, ReservationIdentity, ReservationRequest, EffectResult, OpaqueCapability
type EffectHandle interface { Wait; Cancel; Drain }
type ReservationHandle interface { ... }
type CapabilityBridge interface { Publish; Await; Consume }
type Session interface { ... }   // the operation set the facade's Session exposes today
// package testpilot keeps: Driver, PreparedProgram, EntrypointPlan, InstructionPlan, Expression
// and declares `type Coordinate = contract.Coordinate` etc. for every moved type
```

```lean
-- .3  Umpire.Search   (checked : CheckedModel is the index of both the view and the admitted Query)
def admit (checked : CheckedModel) (property : Property) (scenario : Option Scenario)
    (query : Query) (gaps : KnownGapSet := {}) :
    Except AdmissionDiagnostic (AdmittedQuery checked)
-- `query` is fn-82's authored record: form, limits, and the identity and policy inputs
-- (family, key, source, policy, ending, exercise) that CheckedQuery.id and the fingerprints read.
def AdmittedQuery.search : AdmittedQuery checked → PlanResult
def AdmittedQuery.searchWithIntent : AdmittedQuery checked → PlanIntent → PlanResult
def AdmittedQuery.analyzeBranches : AdmittedQuery checked → BranchReport
def AdmittedQuery.withQuery : AdmittedQuery checked → (q : CheckedQuery) → q.model = checked →
    AdmittedQuery checked          -- one view, another Query over the same Model
def SearchView.retarget : SearchView checked → (h : checked = checked') → SearchView checked'
def AdmissionDiagnostic.located : AdmissionDiagnostic → LocatedError
-- Property.checked / Scenario.checked / Query.checked gain (ok : … := by native_decide)

-- .4  Umpire.Evidence (structure module)
def analyze : Facts → Closures → RequiredKinds → Option LinkSupport → EvidenceStructure
def orderingFault? : EvidenceStructure → Audience → Option OrderingFault
def closureFault?  : EvidenceStructure → Audience → Option ClosureFault
def factsInOrder   : EvidenceStructure → List Fact
def linkSupport    : EvidenceStructure → LinkSupport
inductive Audience | raw | accepted

-- .5  Umpire.Case.Projection
structure Realization where
  literals : List (PropertyFieldPath × Value)   -- request-side values the Program assigns
  ruleSuffix : String
  capture : CapturePolicy                       -- .none for a safety rule; .crossEvent for the capture rule
def lower (property : CheckedFieldProperty) (observation : ObservationDeclaration) (root : String)
    (realization : Realization) : Except Compiler.Error Lowered
structure Lowered where
  contract : ContractLowering
  coverage : CoverageRequest
  certificate : Correspondence   -- every field the rule reads is a coordinate `property` compares;
                                 -- every literal it compares against is in `realization.literals`
-- Compiler.compile : Input → Except Compiler.Error Case   (signature unchanged; admits Lowered inside)
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **Behavior-neutral by construction.** Memory records a prior module extraction that added a
  stricter check while moving a validator and rejected previously accepted carriers. Each task
  compares the moved decision against its base on the same fixtures before adding any guard; a
  hardening opportunity found on the way is recorded as a `CONSIDER(umpire)` and not landed here.
- **.1 refusal identity.** A fault on a queue whose lease is not dedicated returns the same
  `ErrUnsupportedOperation` it does today; the server Driver's `InjectFault` refusal and its
  comment are untouched. `Begin` after `Close` has started returns an error and flips nothing.
  `Restore` after a failed resume still releases the hold and the dedicated group is gone
  afterwards, which is what the registry does today. `Begin` twice in the same direction conflicts. The one-`FAULT_INJECTED`-event-per-realized-instruction rule of
  draft EVD-20 is enforced by the scheduler on a succeeded outcome, so a `Settle` error must surface
  as a non-succeeded instruction outcome and produce zero fault events.
- **.2 alias purity.** Every moved type is re-exported by alias, so `conformance_test.go` and
  `facade_external_test.go` compile without edits. The `Quarantine` type switch that unwrapped the
  facade's own adapters is deleted with the adapters, together with the facade test that
  instantiates them. Refusing to quarantine a handle a Session did not issue is each Driver
  Session's decision and already lives in the delivery ledger and the server Driver; the task pins
  it through the existing conformance suite rather than moving it. The mirrored `MaxOpcode`
  justification comment survives once, in the leaf.
- **.3 optional Scenario.** `admit` accepts a Property-only Query (no Scenario), because two of
  the six callers admit that way today. Each stage's typed error is carried unchanged inside
  `AdmissionDiagnostic`. `SearchView.retarget` and `AdmittedQuery.withQuery` replace the five
  production `Eq.mpr` transports (the Variations compiler, the Exploration engine, the Nexus
  operations index and the two experimental Nexus modules) and the thirteen fixture copies; the
  only `Eq.mpr (congrArg …)` over a view that remains is the body of `SearchView.retarget`. If a
  site needs a transport neither operation expresses, the task records it rather than keeping an
  `Eq.mpr` there. The `native_decide` auto-param must not raise elaboration time of the Switch
  example or any Nexus Model beyond the current `lake build` of the same targets by more than the
  noise floor the task measures first.
- **.4 precedence.** The raw and accepted paths establish precedence in different orders today. The
  task first proves, on every fixture in the mutation and evaluation suites, that the two orders
  never disagree on a reachable input; where they can, precedence is an audience-specific table
  inside the module so that no diagnostic byte changes. Both faults firing at once is answered by
  the same precedence table.
- **.5 rejection lists.** The three field-step walkers genuinely differ per side today: the
  request-side walker accepts a keyed map step the read-side walker rejects, and the projection
  walker accepts a first-index step. The one private walker therefore carries per-side exceptions
  from the start and lists each with the Case that relies on it; a disagreement is resolved to the
  stricter answer only where no checked-in Case relies on the looser one. A Property with no field
  atom lowers to no rule and is not an error. A rule the Property does not imply is rejected by
  name.
- **Retired vocabulary.** New module and type names avoid every token the retired-vocabulary gate
  lists; file relocations are mirrored in the gate's scanned-path registry in the same change so the
  gate keeps failing closed rather than silently. Public names this spec deletes are added to the
  retired list.
- **Spec-rule text.** Task .2 drafts a MOD-14 restatement naming the leaf package and marks it
  pending GOV-02, following the fn-80 pattern for EVD-20 and EVD-21. No rule is approved here.

## Quick commands

```bash
# Go (.1, .2)
go test -count=1 -tags test_dep ./common/testing/testpilot/...
go test -count=1 -tags test_dep ./common/testing/testpilot -run '^TestCaseRuntimePublicFacadeConformance$'
go test -count=1 -tags test_dep ./common/testing/testpilot/temporal/worker/ -run '^TestFault|^TestSession|^TestPreparedDefinition|^TestWorkerProfile'
make lint-code
make umpire-check-live-tests

# Lean (.3, .4, .5)
cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Search.Tests Umpire.Search.VisibilityTests
cd model && mise exec -- lake build Umpire.Evidence.Tests Umpire.Evidence.Tests.Mutations
cd model && mise exec -- lake build Umpire.Case.CompilerTests Umpire.Case.Tests.FieldLowering umpire-scoped-fixtures temporal-testpilot
cd model && mise exec -- lake build UmpireTests
make lint-model
make umpire-check-case-runtime-conformance

# Final gate, once per task at close
make umpire-check-regression
```

Test target names above follow fn-82's module layout; a task uses whatever names fn-82 landed.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** One `outage` module inside the worker package owns outage admission and the
  dedicated-group state machine. Definition preparation calls `PlanOutages` once and `Validate` and
  `Open` share the returned `OutagePlan`, so the test that asserted the two agree no longer
  exists. The test-only stop/resume facade is deleted and every fault test drives `Begin`,
  `Settle`, `Restore` and `Stopped` through the production path; no fault test reads a registry
  group field (the runtime tests' registry-emptiness reads are outside this criterion). The worker
  README's stated outage contract stays true. Errors: an `OutagePlan` naming a queue no entrypoint
  registers is refused at plan time with the same message as today; a fault on a non-dedicated
  lease returns `ErrUnsupportedOperation`; `Begin` after `Close` began returns an error and flips no
  group; `Restore` after a failed resume still releases the hold and the group is gone; a `Settle`
  error yields a non-succeeded outcome and zero `FAULT_INJECTED` events; the live worker-outage
  tests and the `^TestTestpilot` gate pass with unchanged Verdicts.
- **R2:** A leaf package under the Testpilot module holds the Driver-facing contract and imports
  neither the execution package nor the IR package; the execution package and the facade both
  import it; the facade re-exports every moved type by alias and keeps only `Driver`,
  `PreparedProgram`, `EntrypointPlan`, `InstructionPlan` and `Expression` as its own. The four
  pass-through adapters, both coordinate converters and the profile policy copy loop are deleted.
  `go list -deps` shows no production package outside Testpilot importing the execution package.
  The facade test that instantiated the deleted adapters is deleted with them. Errors:
  `conformance_test.go` and `facade_external_test.go` compile unedited; each Driver Session still
  refuses to quarantine a handle it did not issue, pinned through the existing conformance suite;
  the facade conformance test passes; the MOD-14 restatement is present in `UMPIRE4_SPEC.md` and
  marked pending GOV-02.
- **R3:** `Umpire.Search.admit` takes the checked Model, the Property, an optional Scenario, the
  authored Query record (form, limits, identity and policy) and Known Gaps, and returns
  `Except AdmissionDiagnostic AdmittedQuery`; the six Temporal callers plus the Switch example
  obtain their search result through it and their bespoke admission-error unions are deleted; the
  Variations compiler, the Exploration engine and candidate universe, the Exploration session and
  Promotion take an `AdmittedQuery` and re-pair Queries with `AdmittedQuery.withQuery`; the only
  `Eq.mpr (congrArg …)` over a search view in the tree is the body of `SearchView.retarget` inside
  `Umpire.Search`; the four `.checked` constructors take the `native_decide` auto-param and no
  `_isSome` theorem remains whose only use was that argument; the search view's proof fields are
  not reachable from outside `Umpire.Search` except through `search`, which stays public for
  Promotion. Errors: each stage's rejection surfaces as its own `AdmissionDiagnostic` constructor
  carrying the stage's typed error unchanged; a Property-only Query admits; `CheckedQuery.id`,
  the search-run scope and every fingerprint are byte-identical to today; admit then search yields
  the same `PlanResult` bytes as today's staged calls on the Switch example and every Nexus Model;
  the Variations goldens do not move.
- **R4:** The Evidence structure module exposes `analyze`, `orderingFault?`, `closureFault?`,
  `factsInOrder` and `linkSupport` with an `Audience` parameter; findings, closure expectations and
  precedence are not exported; the raw and accepted callers hold no finding-to-diagnostic matcher
  family and no reachability copy; one generic reachability walker, owned on the `Shared` side
  because MOD-09 forbids `Shared` importing Umpire, serves the correlated projection and the
  Evidence structure module over their different node types. Errors: every diagnostic in the mutation suite is byte-identical before and after;
  the origin-mode matrix passes through the new interface; a fixture on which the two old precedence
  orders would have disagreed is listed in the task summary with the audience-specific table entry
  that preserves it.
- **R5:** `Umpire.Case.Projection.lower` takes the checked field Property, the declared
  Observation and a realization record (request-side literals, rule suffix, capture policy) and
  returns the Contract lowering, the coverage request and a correspondence certificate, and
  `Compiler.compile` admits it beside the correlated lowering; the typed unary and typed Nexus
  Producers call it and hold no hand-written read path, rule, coverage request or lowering-error
  helper; the three field-step walkers are one private walker whose per-side exceptions are listed
  with the Case each serves. If the typed Nexus capture rule cannot be reproduced byte-identically,
  the task narrows to the safety-rule shape, leaves that Producer's rule in place under a
  `CONSIDER(umpire)`, and reports the narrowing. Errors: a Property whose compared coordinate moves
  moves the rule; a coordinate the Property stops naming is no longer read; a literal the Program
  does not assign rejects by name; a rule the Property does not imply rejects by name; each
  unsupported step kind rejects by name once per side; every checked-in Case fixture is
  byte-identical and the conformance and live gates pass with unchanged Verdicts.
- **R6:** Each task closes with `make umpire-check-regression` green and a recorded equivalence
  pin: .1 the fault suite and the live outage tests, .2 the unedited conformance and facade tests,
  .3 the Switch and Nexus `PlanResult` byte comparison plus goldens, .4 the mutation suite, .5 the
  byte-identical Case fixtures. Errors: a fixture, golden, fingerprint or Verdict that moves fails
  the task; no fixture is hand-edited (fn-82's regenerate-never-edit rule applies).
- **R7:** Documentation that describes the old shape is updated in the same task: the Umpire
  architecture document's model lifecycle, Evidence and Case-production sections, the model README's
  check-then-checked walkthrough and its stale live-gate selector, the Testpilot execution, composite
  and worker READMEs, the components plan, and the glossary in `tools/umpire/CONTEXT.md` for each
  new term a module introduces (outage plan, admitted Query, evidence structure, derived rule), each
  entry saying which of the existing senses it belongs to. Errors: the documentation gate test in the
  regression package keeps every pinned fragment byte-identical after whitespace normalization, or
  the task updates the gate deliberately and says so; a claim the scans found already false today
  (the Contract read path is fully derived from the Property) is made true by .5, not reworded.

## Boundaries
<!-- scope: business -->

- No renames for their own sake; fn-82 owns vocabulary. Names introduced here are the minimum the
  new modules need and follow fn-82's spellings.
- No semantic change to the evaluation budget (`CONSIDER(umpire)` on the cubic reservation stays
  a separate spec), to delivery routing, to the generators' publication tooling, to Property
  evaluation combinators, or to `Umpire.Json` sealing. Those were surfaced and are recorded below.
- No deletion of `Umpire.Evidence`, `Umpire.Artifact.RunRecord`, `Umpire.Variations`,
  `Umpire.Exploration` or `Umpire.Promotion`; fn-22, fn-33, fn-79 and fn-80 reserve those decisions.
- No proto field-number changes, no new instruction kinds, no new command syntax.
- No approval of MOD-14's restated text or of draft EVD-20; both stay pending GOV-02.
- No generated Lean API drift verification or CI expansion (declined ledger entry
  `generated-api-drift-verification`); existing fixture and golden regeneration checks stay.
- Drivers outside this repository are out of scope; the alias plan keeps in-repo Drivers compiling.

## Decision Context
<!-- scope: both -->

Five scans, each scoped to one area, ran independently and each nominated one candidate; the spec
carries exactly those five so that the selection is the scans' and not a second editorial pass.
The order .1 to .5 follows the scans' own recommendation: the worker outage is where the repository
is already paying visibly, the leaf package is the purest deletion-test win, and the three Lean
candidates wait for fn-82 to finish moving their files.

Rejected as scope here, each recorded so the next architecture review does not re-surface it
without a reason: a sealed-artifact module finishing `Umpire.Json` (fn-60 already owns canonical
JSON deepening); deepening the Temporal Case-support constructors into a workflow-backed
realization (fn-83's templates cover the same ground); `DeriveProfile` as the only Go Profile source
(small, fold into a later cleanup); one owner for the evaluation budget (changes semantics near a
ceiling, needs its own spec); one owner for the environment binding snapshot (must byte-preserve
the fingerprint, separate); a delivery directory owning routes and redelivery (largest and hottest
correctness path); a Producer publication module for the generators (L, tooling rather than
runtime); decidable-obligation combinators for Property evaluation (proof-heavy, in the file fn-82
renames most); a search-claim module (Worth exploring); Implementation Link coverage decided from
the Model's vocabulary (proof-strength question unresolved).

`admit` lives in `Umpire.Search` and not in `Umpire.Query` because the search view is Search's type
and Search imports Query; placing it in Query would invert the import direction `lint-model`
enforces. Task .5 extends `Umpire.Case.Projection` rather than adding a `Rule` submodule because
fn-82 R4 fixes the Case submodule set and fn-83's generic Producer already calls into Projection.

Dispatch is serial in the order .1 to .5 even though only .5 declares a task dependency: every
pair of tasks shares a documentation or test-root file (`tools/umpire/CONTEXT.md`, the model test
root, the Umpire architecture document, the model README), and wave dispatch fails closed on an
overlapping `Touches` line. That matches the intended order and no artificial split is made to
manufacture a parallel wave. The spec-level dependency on fn-83 holds .1 and .2 as well as .5;
fn-83 has no tasks yet and the two Go tasks are cheap to hold, so one dependency edge is kept over
a task-level one that fn-83 cannot yet anchor.

Prior art: fn-31 deepened the checked Model so examples stopped assembling completeness evidence
and search views by hand; the six callers still assemble the chain above that, which is what .3
finishes. fn-74 deepened worker activation with the same ownership argument .1 applies to outages.
fn-75's equivalent-Machine seam is the expert counterpart of `SearchView.retarget`.

## Early proof point

Task .1 (the outage module) validates the approach: the code moves largely intact inside one
package, and if the fault suite cannot be rewritten onto `Begin`, `Settle`, `Restore` and
`Stopped` without reaching into registry state, the interface is wrong and the remaining four
tasks re-check their own interfaces before starting.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | worker outage module | .1 | — |
| R2 | Driver-contract leaf package | .2 | — |
| R3 | admitted Query | .3 | — |
| R4 | Evidence structure verdict | .4 | — |
| R5 | derived Contract rule | .5 | — |
| R6 | equivalence pins and regression gate | .1, .2, .3, .4, .5 | — |
| R7 | documentation and glossary | .1, .2, .3, .4, .5 | — |

## References

- Architecture review report (temp file, regenerable): `architecture-review-20260909-225448.html`
  in the OS temp directory; five scans, one per area.
- `.plans/UMPIRE4_SPEC.md` MOD-12, MOD-13, MOD-14, SEM-16, SEM-17, ART-09 to ART-14, EVD-12,
  EVD-19, EVD-20 (draft), GOV-02.
- fn-82 (vocabulary), fn-83 (generic Producer), fn-31, fn-74, fn-75 (prior deepenings), fn-22 and
  fn-33 (consumers of `search` and of the search-view transport sites).
- Memory: behavior-neutral refactors must not strengthen validation; full integration gates must
  select the complete migrated suite; moved conformance tests must not import functional adapters.
