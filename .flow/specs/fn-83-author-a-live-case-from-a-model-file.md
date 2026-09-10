## Goal & Context
<!-- scope: business -->

Status: implementation specification derived from the 2026-09-09 gap analysis of
[UMPIRE4_VISION](../../.plans/UMPIRE4_VISION.md) against `model/`, the Testpilot Go runtime, and
the live tests under `tests/`. [UMPIRE4_SPEC](../../.plans/UMPIRE4_SPEC.md) remains normative.
This spec is written in the vocabulary and module layout fn-82 lands (Model, Machine, Step, Fact,
Scenario, Property, Query forms `find` and `verify`, Limit, Projection, Provenance,
`Umpire.Variations`, `Temporal.Feature.Nexus.Success`, the `umpire-case` executable) and starts
after fn-82 closes. Line-level citations of the tree before fn-82's moves live in the task files.

### What the vision asks for and what exists

| Vision item | State on the current tree |
| --- | --- |
| Single model of behavior | The checked Model exists and is small: the Nexus success Model is 43 lines of command syntax. The Model alone does not yield a Case. Every Case has a hand-written Producer that spells out its Program in raw Lean constructor calls: 380 lines for the Nexus success Case, 227 for the worker-outage Case, which carries no Model at all. |
| Deterministic, checked-in regression tests | Done. Lean renders canonical ProtoJSON, six fixtures are committed under `tests/testcore/testpilot/testdata`, and `umpire-check-case-runtime-conformance` fails on drift. |
| Same model for local, CI, canary | Half done. Unchanged Case bytes bind to two physical environments through `DeriveProfile`. No consumer other than the functional tests exists; the canary is deferred (fn-70, fn-29). |
| White-box and black-box modes | Black-box structurally: the Driver reaches the server only over gRPC and HTTP and holds no in-process handle. Nothing runs a Case against an endpoint outside the functional test cluster, and no example shows an assertion that needs white-box access. |
| Developer-friendly API | The five-command Model syntax is friendly. Everything after it is not. The Producer is raw Lean; the typed examples use no syntax at all (550 and 940 lines with proof terms); adding the worker-outage Case touched 14 files and 585 lines; two hand-maintained registries name every Case; the only tutorial documents keywords that do not exist. |
| Exploration mode | Lean-only. `Umpire.Exploration` is a complete pure engine whose output is a Plan, and no path turns a Plan into a Case. `umpire-fuzz` does not exist. |
| Faults as first-class citizens | One fault pair (worker stop, worker resume) works end to end on the real Driver. It is declared in raw Lean beside a hand-written Program, not in the Model syntax. |
| Non-linear steps (IDs unknown until runtime) | Done. Slots, response projections, `RunRef`, Contract captures, and correlated captures exist and are live. |
| Pre-programmed SDK workers | Done for the shapes in use. Workflow and Nexus-handler entrypoints are interpreted inside a real Go SDK worker. Activity entrypoints are validated but have no interpreter; signals, timers, child workflows and continue-as-new are absent from the instruction set. |
| Guided exploration, guided fuzzing | Absent beyond the offline guided selector. |
| Clock skew across processes | Partial. Deadlines are logical (`rule_events`, operation transitions) or one host's milliseconds. |
| Translate existing functional tests | Absent. No test in the tree is a translation, and no author outside the Umpire maintainers has written one. |

Of the vision's seven acceptance criteria, three are met (Go SDK driven, one fault injected,
deterministic file artifacts), one is half met (two consumers), and three are unmet (explorative
test, translated functional test, syntax friendly enough for Lean newbies past the Model block).

### What this spec does

The single largest distance from the vision is that **a developer who can write the Model cannot
get a running test without also writing a Producer**. Every other unmet item either depends on
closing that gap or is already owned by an open spec. Exploration (fn-33) cannot emit Cases
because there is no generic Producer; the canary (fn-70) needs nothing new from authoring; clock
skew and guided fuzzing have no consumer yet.

So this spec makes one thing true: **a Model file plus one `case` block is a running, checked-in,
deterministic live test, and a fault is one line in a Scenario.** It measures that claim by
translating one upstream functional test using nothing but the command syntax, and by
re-authoring the worker-outage Case as a Model with two fault lines. The translation also
supplies the vision's white-box example: one upstream assertion reads mutable state through the
admin service, and the translation records it as a Known Gap that no black-box Run can close.

It deliberately adds no exploration, no canary consumer, no Program-level surface syntax, no new
fault or instruction kinds, and no clock model. Boundaries names the owning spec for each.

## Architecture & Data Models
<!-- scope: technical -->

### Ownership

```text
Model file (command syntax)                     model/Temporal/Feature/<Feature>/<Name>.lean
  model / property / scenario [fault] / limits / query / case
        |
        v  elaborates into
Checked Model + selected witness                 Umpire.Model, Umpire.Search (existing)
        |
        v  Producer, written once                Umpire.Case.Producer   (new, SCP-02 clean)
Realization template + evidence + faults         Temporal.Case.Template.* (new, Temporal-owned)
        |
        v  compile                               Umpire.Case.Compiler   (existing)
Case bytes                                       umpire-case --list / --render
        |
        v  one generator, one functional table   umpire-gen-case-runtime-conformance
tests/testcore/testpilot/testdata/<name>-case.json
        |
        v  one live test per Case                tests/testpilot_<name>_case_test.go
```

The Producer is written once. It reads the checked witness trace, the Scenario's Action order,
the Property's clauses, and the `case` block's evidence mapping, and builds the Program from a
named realization template and the Contract by the derivation fn-80 R1 proved for the Nexus
success Case. `Temporal.Feature.Nexus.Success.Producer` is deleted; its generic half becomes
`Umpire.Case.Producer` and its Program becomes one template.

### R1 the `case` block and the generic Producer

A sixth command joins the five fn-82 respells. It names a `find` Query, a realization template
with its parameters, and the history event that confirms each Action:

```lean
case asyncNexusSuccess fixture "async-nexus"
  realizes completion
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    awaitStart   ← history nexusOperationStarted
    awaitSuccess ← history nexusOperationCompleted
```

`fixture` names the checked-in file and derives every identity the Case carries: the Case ID is
`temporal.case.<fixture>`, the Program ID `<caseId>.program`, the Contract ID `<caseId>.contract`,
and the run-scope literal `<fixture>`. The grammar has no other identity slot. `realizes` takes a
`find` Query; a `verify` Query rejects with a located error, because a Case realizes one selected
trace. `as` names a template and its parameters. `evidence` maps each Action the Scenario selects
to one recorded history event kind. The Step each event confirms (state, outcome, facts) is read
from the checked Machine along the witness trace, so the author writes it once, in the `steps`
block. Hook names for fault lines are resolved here too, because only the `case` block knows the
template. The `case` command is an elaborator, not a macro: it resolves names and registers the
Case (R4).

Definition IDs and sources are per file. The command syntax derives the definition family from
the enclosing namespace and the source location from the elaborating file, so two Models that
both name a `lifecycle` in different files carry distinct Definition IDs and distinct Provenance
sources. The Nexus success family keeps its current value under that derivation.

`Umpire.Case.Producer` is the reusable half. Its input is an Umpire-owned record: the checked
Model, its vocabulary as `ModelValue` lists, the checked Property, the checked Scenario, the
witness trace, and the Query's ID, source, and Known Gaps; the Temporal authoring bundle converts
to it at the call site, because MOD-01 forbids the Producer from importing a Temporal type. With
it come an abstract `Realization`, an evidence mapping, and fault lines. Its output is the
`Umpire.Case.Compiler.Input` the Nexus success Producer builds by hand today. Nothing in it names
Temporal (SCP-02); the Temporal-named constants move to the templates. The Correlated-clause
derivation, the same-step vacuity rejection, the coverage check that every Property clause
appears in the Contract, and the Provenance bindings move unchanged. Witness selection is
deterministic under PLN-02, which is what makes the fixture stable.

Proof of R1 in two steps. The extraction alone (Realization built by hand with today's IDs)
regenerates the async-Nexus fixture byte-identical. The `case` block then regenerates it with the
derived identities: the diff is limited to the Case ID, Program ID, Contract ID, run-scope literal,
and Provenance; with those fields masked the Program and Contract are byte-identical, the receipt
lists the diff, and the live async-Nexus tests pass with no change beyond the fixture name.

### R2 realization templates

`Temporal.Case.Template` owns the Temporal-specific Programs. Two templates cover every shape the
Model-bearing and fault-bearing checked-in Cases use:

- `nexusOperation service operation responds sync|async`: a controller-started workflow
  schedules one Nexus operation on the Case's endpoint role; the handler responds synchronously,
  or asynchronously with the controller completing it; the controller reads history. The `async`
  form is today's async-Nexus Program verbatim. In the `sync` form nothing sequences the history
  read after completion, so the template first reads with the close-event filter, as the
  worker-outage Program does, and only then performs the full read the Projection consumes.
  Hooks: `start`, `completion`.
- `workflow type`: a controller-started workflow that finishes; the controller reads history. This
  is today's worker-outage Program without its two fault instructions. Hooks: `start`,
  `completion`. Its fault rule ID stays `worker-outage-order` so the existing artifact assertion
  keeps its meaning.

A template is a Lean value, not syntax; adding a third is one Lean file. The template owns the
evidence sources the Correlated capability reads, because they are Temporal-specific and depend
on which event kinds the `case` block maps: for each admitted event kind the template states the
attributes field, the operation-key path, the evidence kind ID, the source ID, and the scope
literal, and its Program is a function of the resolved evidence rules, so the history node's
projection targets are built from the mapping rather than patched afterwards. The Nexus template
keys operations by the scheduled event ID as today; the `workflow` template runs one workflow per
Run, so it keys by a stable path on the single close event such as `event_id`, because a
completed-workflow event names no operation and an operation key is always a path, never a
literal. The
`evidence` block's `history <eventKind>` resolves against the generated history event attribute
oneof in `Temporal.API`, so a misspelled kind rejects with the admitted kinds listed, the same way
an unknown Action does today. This resolution reads generated names the author never declared,
which AUT-09 as drafted does not cover; see Parked unknowns.

The typed examples (`TypedUnary`, `TypedNexus`) stay on the expert path unchanged.

### R3 faults in the Scenario

A Scenario may carry fault lines:

```lean
scenario completionSurvivesOutage on lifecycle
  operation starts pending
  actions exactly [completion: awaitCompletion]
  fault workerStop before start
  fault workerResume after start
```

`fault <kind> before|after <hook>` names one of the two existing fault kinds and one hook of the
template the `case` block chooses. The Producer owns the hook-to-instruction map the fault
lowering has always left to its caller: it lowers each line through the existing intent and
realization types in `Umpire.Variations`, with the template's task-queue role as the outage role;
the lowering itself keeps ignoring the occurrence. Ordering in a controller entrypoint comes only
from dependency edges, and a realization can only make the fault depend on something, so the
Producer also rewrites edges: for `before`, the hook's instruction gains a dependency and guard
on the fault; for `after`, the hook's successors gain a dependency on the fault. That is what
makes a stop precede the start it names, as today's hand-written Program does. A hook the
template does not declare rejects by name at the `fault` line, listing the template's hooks. Two
fault lines with the same kind, hook and placement reject as duplicates. Stop and resume pairing
is not checked at elaboration; the outage-order rule the Producer adds to the Contract whenever a
Scenario carries faults (fn-80 R4, a `rule_events` Deadline over the recorded `FAULT_INJECTED`
events in declared order) answers an unpaired stop at Run time.

Proof of R3: the worker-outage Case file is deleted and re-authored as a Model (`pending →
completed` on `awaitCompletion`, one Fact `completed`), a Property requiring the completed state
and fact, the Scenario above, and a `case` block on the `workflow` template. The fixture changes
bytes: its Provenance carries Model bindings, and its completion requirement is now Correlated
clauses rather than the hand-written safety rule, so the Verdict carries the outage-order rule
(same ID, terminal state `resumed`) plus one satisfied rule per clause. Both live outage tests
pass against that shape, and the offline artifact test pins the new bytes and the `rule_events`
Deadline.

### R4 one registry, two authored files per Case

The `case` command registers its Case in a Lean environment extension that stores the
declaration name of the Case value and the fixture name; a persistent extension lives in the
`.olean`, so it holds names, never closures. A term elaborator in the renderer's root module
materializes the registered names into the sorted list the executable enumerates, and the
duplicate-fixture diagnostic fires there. The two typed examples register their existing Case
values with an explicit call so `--list` covers every functional fixture; their Programs,
Profiles, and Contracts do not change. The synthetic and conformance Cases do not register: the
Go conformance builder keeps naming them by renderer argument, because its entries carry expected
Verdicts the registry does not model. `umpire-case --list` prints every registered Case as
`<case-id> <fixture-name>` sorted by Case ID; `--render <id>` prints its canonical bytes. A Model
file the aggregator module does not import registers nothing, so its fixture is absent and its
live test fails at load; that failure is loud enough that no lint is added. The Go generator's
hand-written functional table and its fixed-count guard are deleted; it calls `--list` and
renders each entry. The Lean renderer's functional dispatch table is deleted.

The offline artifact tests split by kind. One table-driven test over the testdata directory
decodes every fixture, prepares it against its derived Profile, and pins its identity; the
per-Case semantic assertions (Deadline units, tenfold load, run isolation, typed Profiles) stay
as small per-Case functions. `runCase` derives a default binding (namespace, task queue, Nexus
endpoint) from the fixture name so a live test passes only the name; tests that vary bindings
still pass one. The evidence helpers the async-Nexus test defines move to the shared live-test
helper file.

Adding a Case after this spec touches exactly two authored files: the Model `.lean` file (its
`inductive` declarations included) and one live Go test whose body is the `runCase` call and
Verdict assertions. The fixture JSON is generated. R6's commit is the measurement.

### R5 the authoring walkthrough

`model/AUTHORING.md` replaces the Nexus success tutorial. It walks from an empty file to a green
live test: the `inductive` declarations, the five blocks, the `case` block,
`make umpire-gen-case-runtime-conformance`, the Go test, and `make umpire-check-regression`. It
lists every located diagnostic an author can hit, each with the fix. Every Lean code block in it
is the literal content of a region of the R6 Model file delimited by `-- authoring: <name>`
marker comments, and a Go test under `tools/umpire` asserts that equality, so the tutorial cannot
drift again. The old tutorial's "proposed, not implemented" forms move verbatim to a `DESIGN.md`
beside the Nexus success Model. The integration note beside it is reduced to the ID convention
and the Case boundary.

### R6 translate one upstream functional test

The upstream Nexus workflow suite's synchronous completion test (`TestNexusOperationSyncCompletion`)
is re-authored as a Model file using only the six commands and its `inductive` declarations, with
no new template and no Producer code. It exercises the `nexusOperation` template's `sync` form.
A `COVERAGE.md` beside the Model maps each upstream assertion to a Property clause, a Contract
rule, or a Known Gap in the Case's Provenance: the result value and the completed history event
are carried; the handler-supplied link on the completed event is a Known Gap unless the template
can express it; the admin-service mutable-state assertion is a Known Gap of kind white-box, which
is the vision's example of a test a black-box consumer cannot run. It also records that the
upstream handler is an external Nexus server while the translation's handler runs inside the
Testpilot worker. The upstream test is not deleted.

### R7 run a Case against any endpoint

Provisioning leaves the functional test file. `common/testing/testpilot/temporal/provision`
registers a namespace and polls its description until the namespace cache serves it, creates a
Nexus endpoint, and deletes both through the operator service, over the operator and
workflow-service gRPC clients only; it takes no test environment, no `testing.T`, and no
server-internal namespace package, which is the seam the fn-70 deferral note recorded as the
reason the helpers stayed test-local. The live tests call it. Transport credentials are the
insecure ones the live tests use; TLS and authentication flags are out of scope.

`tools/umpire/cmd/umpire-run` takes a fixture path, a gRPC address, an HTTP address, a namespace,
a task queue, an optional Nexus endpoint name, `--create`, and `--timeout`. It derives the Profile
through `DeriveProfile`, builds the composite Driver with its own SDK worker, runs once, and prints
the Run status, cleanup status, Verdict status, and one line per rule Verdict. Exit codes: 0
satisfied, 1 violated, 2 inconclusive, 3 preparation, infrastructure, or Run error. Without
`--create` the resources must exist and are never deleted; with it an existing namespace or
endpoint is an error, and both are deleted on exit. SIGINT and the timeout cancel the Run and
tear down best-effort, printing one stderr line per resource that could not be removed. Only
Cases whose Profile `DeriveProfile` derives are runnable (three of the six checked-in fixtures
today); a typed fixture rejects with the `PreparationError` category on stderr. The CLI takes no
lock; concurrent invocations need caller-unique names. It imports the Driver and the SDK, never
`tests/testcore` or `service/`.

Proof of R7: a tagged test starts the functional cluster, runs the binary as a subprocess against
its addresses with the async-Nexus fixture, and asserts exit 0. This is the vision's black-box
mode on real bytes, and the seam fn-70 and fn-29 consume.

## API Contracts
<!-- scope: technical -->

```lean
namespace Umpire.Case

/-- The checked authoring bundle, Umpire-owned; the Temporal authoring bundle converts to it. -/
structure Producer.Input where
  model : CheckedModel
  vocabulary : Vocabulary            -- states, actions, outcomes, facts as ModelValue lists
  property : CheckedProperty
  scenario : CheckedScenario
  witness : Trace
  queryId : DefinitionId
  querySource : SourceLocation
  knownGaps : KnownGaps

structure Identity where
  caseId : String                    -- temporal.case.<fixture>; program and contract IDs derive
  fixture : String

inductive HookPlacement | before | after
structure Hook where
  name : String
  instruction : InstructionRef

/-- What the template knows about one admitted event kind. -/
structure EvidenceSource where
  eventKind : String
  attributesField : String
  operationKeyPath : Path
  kindId : DefinitionId
  sourceId : DefinitionId

structure EvidenceRule where           -- one resolved (action, source) pair
  action : ModelValue
  source : EvidenceSource

structure Realization where
  roles : Array RoleDefinition
  environment : Array EnvironmentDefinition
  program : Identity → List EvidenceRule → Program
  historyObservation : String
  operationKey : DefinitionId
  scopeField : DefinitionId
  taskQueueRole : String
  faultRuleId : String
  hooks : List Hook
  sources : List EvidenceSource

structure EvidenceMapping where
  action : ModelValue
  eventKind : String

structure FaultLine where
  kind : FaultKind            -- .workerStop | .workerResume
  hook : String
  placement : HookPlacement

def Producer.produce
    (input : Producer.Input) (identity : Identity) (realization : Realization)
    (evidence : List EvidenceMapping) (faults : List FaultLine)
    (required : List DefinitionId := []) :
    Except LoweringError testpilot.v1.Case

end Umpire.Case
```

```lean
namespace Temporal.Case.Template
inductive Response | sync | async
def nexusOperation (service operation : String) (responds : Response) : Umpire.Case.Realization
def workflow (workflowType : String) : Umpire.Case.Realization
end Temporal.Case.Template
```

Command syntax additions (exact grammar is a task decision; these forms are normative):

```text
case <name> fixture "<fixture-name>"
  realizes <query>
  as <template> <param>*
  evidence
    (<action> ← history <eventKind>)+

scenario ... (fault <workerStop|workerResume> <before|after> <hook>)*
```

The registry stores `(declaration name, case ID, fixture name)` per registered Case; the
renderer's root module materializes the sorted list through a term elaborator.

```text
umpire-case --list                   # <case-id> <fixture-name>, sorted by case-id
umpire-case --render <case-id>       # canonical ProtoJSON on stdout
```

```text
umpire-run --case <path> --grpc <addr> --http <addr> --namespace <ns> --task-queue <q>
           [--nexus-endpoint <name>] [--create] [--timeout <duration>]
exit 0 satisfied | 1 violated | 2 inconclusive | 3 preparation, infrastructure, or Run error
```

Proto: no field-number changes, no new messages, no new fault kinds, no new instruction kinds.

## Edge Cases & Constraints
<!-- scope: technical -->

- A `case` whose Query is `verify` rejects at elaboration naming the Query. A `find` Query whose
  Search returns no witness rejects at production.
- An `evidence` line for an Action the Scenario never selects, or a selected Action with no
  evidence line, rejects by Action name listing the selected Actions. An event kind the generated
  API does not know rejects listing the admitted kinds.
- A `fault` line on a Scenario no `case` block realizes is never elaborated and needs no
  diagnostic; one whose hook the chosen template lacks rejects at the `fault` line.
- Two `case` blocks with the same fixture name reject at registration.
- The Producer keeps fn-80's vacuity rule: a `require` whose response holds earlier in the witness
  trace than its Action rejects as `property.clause-early-response`. R6 must not weaken it; an
  assertion that needs to is a Known Gap.
- Hand-editing a fixture remains forbidden (ART-11, ART-12); every byte change goes through the
  owning target and is explained in the task receipt.
- `umpire-run` never embeds addresses or credentials in a Case (QLF-01) and never broadens Limits
  (CLI-02).
- The generic Producer stays under SCP-02: no Temporal name, history attribute name, or role ID
  string. Those live in `Temporal.Case.Template`.

## Quick commands

```sh
cd model && lake build                          # elaborate every Model and case block
make umpire-gen-case-runtime-conformance        # regenerate every registered fixture
make umpire-check-case-runtime-conformance      # fail on fixture drift
make lint-model                                 # import boundaries incl. SCP-02 for Umpire.Case.Producer
go test -tags test_dep ./tools/umpire/...       # tutorial-drift and umpire-run unit tests
make umpire-check-live-tests                    # tagged live Cases against the functional cluster
make umpire-check-regression                    # the full gate
```

## Acceptance Criteria

- **R1:** `Temporal.Feature.Nexus.Success.Producer` no longer exists; the success Model file
  ends in a `case` block; `Umpire.Case.Producer` takes the Umpire-owned input record and passes
  `lint-model` under MOD-01 and SCP-02; the extraction step regenerates the async-Nexus fixture
  byte-identical, and the `case` step regenerates it with a diff limited to the Case ID, Program
  ID, Contract ID, run-scope literal, and Provenance, listed in the receipt; two Models in
  different files get distinct Definition IDs and sources, pinned by `#guard`; both live
  async-Nexus tests pass. Errors: `verify` Query, unmapped selected Action, evidence for an
  unselected Action, and no witness each reject with a located message pinned by `#guard_msgs`.
- **R2:** `Temporal.Case.Template.nexusOperation` (sync and async) and `.workflow` exist as Lean
  values with hooks `start` and `completion` and their evidence sources; the sync form's history
  read waits for the close event before the full read. Errors: an unknown history event kind in
  `evidence` rejects listing the admitted kinds, pinned by `#guard_msgs`; no other error surface.
- **R3:** the worker-outage Case file no longer exists; the outage Case is a Model file with two
  `fault` lines; a Producer unit test asserts the rewritten dependency edges for `before` and
  `after`; both live outage tests pass with the outage-order rule (same ID, terminal state
  `resumed`) and every clause rule satisfied; the artifact test pins the `rule_events` Deadline.
  Errors: unknown hook and duplicate fault line reject with located messages pinned by
  `#guard_msgs`; unpaired stop is unchecked at elaboration and answered by the outage-order rule
  at Run time.
- **R4:** the Go generator has no functional Case table and no fixed-count guard, and the Lean
  renderer has no functional dispatch table; `umpire-case --list` prints every checked-in
  functional fixture name sorted, and nothing else; the shared artifact test is table-driven over
  the testdata directory; `runCase` derives a default binding from the fixture name. Errors:
  duplicate fixture name rejects at the materializing elaborator, pinned by `#guard_msgs`;
  `--render` of an unknown ID exits non-zero naming the known IDs.
- **R5:** `model/AUTHORING.md` exists and the old tutorial does not; a Go test asserts each Lean
  block in it equals the marked region of the R6 Model file. Errors: a missing or duplicate
  marker fails that test naming the marker; no other error surface.
- **R6:** one Model file translates `TestNexusOperationSyncCompletion` using only the six
  commands and its `inductive` declarations; `COVERAGE.md` beside it maps every upstream assertion
  to a Property clause, a Contract rule, or a Known Gap, with the mutable-state assertion recorded
  as a white-box Known Gap; the commit that adds it touches exactly two authored files; its live
  test passes. Errors: no error surface beyond R1 to R4.
- **R7:** `go build ./tools/umpire/cmd/umpire-run` succeeds without linking `tests/testcore` or
  `service/`; the provisioning package takes no test environment or `testing.T`; a tagged test
  runs the binary against the functional cluster with the async-Nexus fixture and asserts exit 0.
  Errors: exit 1, 2, and 3 covered by unit tests through a fake Driver; unreachable address,
  missing fixture, typed fixture, existing resource with `--create`, timeout, and SIGINT each exit
  3 with one stderr line naming the cause and any leaked resource.
- **R8:** `make umpire-check-regression` passes; `make lint-code` reports no finding in a file
  this spec touched; `lint-model` stays at or below its inherited count; the roadmap and the
  documents the docs-gap scan named are updated. Errors: no error surface.

## Early proof point

Task .1 extracts the generic Producer and proves it by regenerating the async-Nexus fixture
byte-identical through the extracted code. If that needs a Nexus-specific branch inside
`Umpire.Case.Producer`, stop: the template boundary is wrong or fn-80's derivation was never
generic, and R2 through R7 build on it.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | `case` block, generic Producer, per-file families, Nexus success Producer deleted | .1, .2, .3 | — |
| R2 | realization templates, evidence sources, event kind resolution | .2, .6 | — |
| R3 | fault lines with edge rewriting, worker-outage re-authored | .5 | — |
| R4 | Case registry, generator table deleted, artifact and live-test helpers | .3, .4 | — |
| R5 | authoring walkthrough with drift test | .8 | — |
| R6 | translated upstream test with coverage record | .6 | — |
| R7 | provisioning package and `umpire-run` | .7 | — |
| R8 | full gate, documents, roadmap | .8 | — |

## Boundaries

- **Exploration and fuzzing** stay in fn-33, which gains this spec as a dependency for the
  generic Producer. Nothing here selects candidates, scores coverage, or loops.
- **Canary** stays in fn-70 and fn-29. `umpire-run` is a one-shot CLI with no policy, lease, or
  publication; fn-29 consumes the provisioning package, not the binary.
- **No Program surface syntax.** Templates are Lean values; the instruction set is not respelled.
- **No new fault kinds, instruction kinds, proto messages, or field numbers.** Activity
  interpreters, signals, timers, child workflows, and RPC-level faults are separate proposals.
- **No clock model.** Deadlines stay logical or single-host.
- **No change to the typed examples' Programs, Profiles, or Contracts** or to fn-77's
  generated-operation path; they only register their existing Case values, and typed fixtures are
  not runnable through `umpire-run`.
- **No TLS or authentication flags** on `umpire-run`.
- **No new CI workflow** and no generated-API drift gate (declined ledger).
- **No edits to historical `.plans` documents** other than `UMPIRE4_ORDER.md`, and
  `UMPIRE4_SPEC.md` only for a `case` concept entry and one drafted rule under GOV-02.
- **Depends on fn-82 closing first.** The command syntax, module paths, and executable names used
  here are fn-82's. fn-82's task that respells the old tutorial can keep that respell minimal,
  because R5 deletes the file.

## Decision Context

Templates were chosen over a Program surface syntax because the four shipped Cases use two Program
shapes, and a syntax that could express arbitrary instruction DAGs would be a second language for
authors to learn on top of the Model. A template is a name plus parameters; an author who needs a
third shape asks for it or writes one Lean file, and the Model syntax does not change either way.

Evidence is mapped per Action rather than per Step because the Machine already says what each
Action produces along the witness trace. Repeating the state, outcome, and facts beside the event
kind is what the Nexus success Producer does today, and it is the part of that file an author is
least able to write.

Faults name hooks, not instruction IDs, because instruction IDs are the template's private
realization and the vision asks for faults to be first class in what an author writes. Hooks are
resolved in the `case` block because only it knows the template; the diagnostic still lands on
the `fault` line.

`umpire-run` is included even though the canary is deferred because it is the cheapest item that
turns "black-box structurally" into "black-box on real bytes", it is what a developer reaches for
after the tutorial's green test, and it is the seam both canary specs assume. Exit code 3 is
separate from 2 so CI can tell an unreachable server from a real inconclusive Verdict.

The synchronous completion test was chosen over the asynchronous one for R6 because it is 65
lines, it exercises the one template parameter R2 adds, and its final assertion is the white-box
example the vision asks for. Translating an upstream test is the acceptance measurement rather
than a nice-to-have: it is the one activity the maintainers' own examples cannot satisfy.

Rejected as overkill: a lint that every Model file is imported by the aggregator (a missing
fixture already fails loudly); a `--external-worker` mode for the CLI; folding the typed Profile
builders into `DeriveProfile` so all six fixtures run from the CLI.

## Parked unknowns

- EVD-20, EVD-21, and AUT-09 were approved on 2026-09-10 under GOV-02. R2's evidence resolution
  reads generated names AUT-09 does not cover, so task .8 drafts one amendment for approval; a
  human approves or rejects it before this spec's completion review, and the code does not wait
  for it.

