# Umpire module and migration map

This map fixes the fn-115 destinations and compatibility boundaries before source moves. The audit
covers the actual working tree at `8fff42d9a8f77a07e8f351f3871cc8030dfe1db6`, including the authorized,
uncommitted fn-107 implementation. It does not use that commit as the content baseline. The owner
has delegated the choices; the conductor obtains independent review of these artifacts before any
restructuring. This document records a map, not a review verdict or completed migration.

The [package audit](umpire-migration-audit.json) records each source package/module, its importers,
Make/CI evidence and disposition. The [migration manifest](umpire-migration-manifest.json) records
current files, the index, protected originals, source substitutions and every retained Case fixture.
The JSON files are evidence inventories; this map supplies their architectural decisions.
fn-124.7 retired the migration goldens and their harness (see "Migration goldens, retired"); the
passages below that describe them are history.

**Reachable-exploration amendment (GOV-02, owner, 2026-10-13).** The owner accepted D1–D8 in
[fn-173's Conversation Evidence](../.flow/specs/fn-173-reachable-exploration-drop-full-tables.md#conversation-evidence)
and [design §10.1](../model/design/reachable-exploration.md#101-decisions). The revised ownership
below is approved architecture, not an implementation or acceptance claim. fn-173.2/.3 replace
eager Tables, Scala catalog/equivalence walks and Query totals; .4/.5 add native state values;
.7/.8/.9 implement backend coverage, answering and the TLC route. No other pending governance
amendment is approved here. Earlier migration compatibility passages retain their historical scope;
they do not require full Tables or their old fingerprint encoding for the new architecture.

## Module ownership

Paths below are destinations, except the explicitly planned Answering module. Public entry points
list current signatures unless a fn-173 replacement is marked planned. Current eager table,
catalog, total and equivalence surfaces remain implementation descriptions until fn-173.2/.3;
they do not override the approved reached-graph contract in the SPEC. Public entry points retain
their current signatures and behavior for the recorded migration,
except that caller-visible checker and producer types acquire names in their owning public facade.
Returned data types, their used methods and error/status values are part of the same contract.
Standard libraries, existing support libraries and generated schemas are permitted where already
used; the dependency column constrains domain ownership, not those ordinary dependencies.

Each module's README describes it for its users: `model/README.md` for the whole system,
`tools/umpire/README.md` for the Go tooling, and the READMEs beside Testpilot, the Temporal Driver,
the functional fixtures and the canary.

`model/OutputPaths.scala` (`umpire.paths.OutputPaths`) is a neutral filesystem leaf shared by
authoring, the IR generator and the gate. It resolves logical Model names to generated artifacts
beside their source owners under features, foundations and actors, including nested and flat
owner folders. Supplemental retained IR and lint files keep their original names under the
owner's `generated/legacy`; generated Cases and their manifest sit directly in `generated`,
and retained Cases and their manifest sit directly in `generated/legacy`. The leaf imports only
the standard library and imports no DSL, Model, IR generator or gate. The DSL's `IrFile`
delegates its artifact readers to this exact leaf;
this permitted dependency does not authorize other imports between the three Scala projects.

| Module and destination | One job | Public interface | Permitted domain dependencies |
| --- | --- | --- | --- |
| Umpire IR, `api/umpire/v1`, `proto/internal/temporal/server/api/umpire/v1` | Represent a lifted Model. | Existing protobuf messages under the new package names. | Protobuf support; no Testpilot schema dependency added. |
| DSL, `model/framework/dsl` (package `framework.dsl`) and the realization core, `model/framework/realize` | Declare Models in Scala, with finite action input domains and native state values. | Authoring and realization declarations, with scoped protobuf builders, realization script helpers and the open traits a system's kit extends in `framework/realize`; `ConstructedRealization`, its immutable dependency identities and conversion context form the system-neutral export protocol; the capability mechanism, naming no capability (`CapabilityOf`, `CapabilityKind`, `Capabilities`, `claim`, `except`, `overriding`, `bound` in `Capabilities.scala`); no native evaluator. Native `Map`/`Set`/`List`, checked `Int` and explicit `oneOf` input domains are planned in fn-173.4/.5; current catalog construction/checking retires in .3. | Scala standard library, generic ScalaPB typing/runtime metadata and the generated Umpire IR messages used by finite signature/schema construction and by `ConstructedRealization.Context` and its result; no Temporal-specific, IR generator or gate import, and no Temporal name (`TestFrameworkNamesNoTemporal`). It builds on its own (`framework/dsl/project.scala`), with no Temporal API or Models on its classpath. |
| Compiler reification, `model/framework/reify` | Retain compiler-proven Model identities and typed bodies before TASTy is packaged. | `reify/plugin` supplies the pinned Scala compiler plugin; `mode=realization` activates family provenance, declaration naming and the minimal nonsignature schema transforms needed by constructed realizations while later endpoint, rule and capture phases remain held. Framework-private annotation carriers persist proofs between compiler runs and are not author-facing outside the framework. | The pinned Scala compiler, shared inspect leaf, generated Umpire IR and exact build-owned generated-API artifacts. Export may consume reify output; reify imports neither export nor any system or Model, and the DSL imports no reify implementation. Stored carriers assume non-hostile compiled dependencies; they are provenance metadata, not signatures for adversarial TASTy. The realization mode currently recognizes the shared Temporal kit's generated-API and self-naming carrier shapes; removing that residual policy coupling remains separate from real-feature migration. |
| Models, `model/temporal` | Declare Temporal behavior. Features live under `features/`, grouped by kind and form; the task queue lives under `foundations/taskqueue/` and the shared worker and client under `actors/`. A feature or form reads top to bottom in one feature file per folder, one object per machine (`object M extends Machine[S, O, F]`, `Derived(...)` or `Composition(...)`, with its `states`, `refinement`, `effects`, `monitors`, `rules`, `properties`, `capabilities` and `queries` sections; a failure model or a negative control says so with the marker `FailureModel` or `NegativeControl`), held to that order by the IR generator's declaration-order lint. A feature with two levels keeps only shared types, its signature and exports in its root feature file and its levels in `product/` and `system/`; each level's own file owns that level's `Phase`, `State` and `Fact`, and one file per subject sits beside it. In `features/activity/standalone`, `system/System.scala` retains the lifecycle types and machine; `system/RetryTimeouts.scala`, `system/Heartbeat.scala`, `system/ResponseByID.scala` and `system/Reset.scala` hold derived subject models; `system/Dispatch.scala` and `system/DispatchRaces.scala` hold the dispatch protocol and races; and `system/DispatchWithTaskQueue.scala` and `system/DispatchWithWorker.scala` hold the queue and worker compositions. The IR generator's structure lint holds that layout. Every Model folder holds one feature file named after it (`features/nexus/workflow/Workflow.scala`, `foundations/taskqueue/TaskQueue.scala`, …). An executable realization is a typed object beside its owning machine level; derived realizations name their base and machine. No folder splits its files by declaration category. The bounds more than one folder runs its Queries under are declared once, in `Bounds.scala`. | Existing machine, claim, Query and realization roots, and one `irFile` declaration per checked-in IR file, in its feature file's `object exports`; kind general files need no exports. | DSL and shared feature kit; generated Temporal API, Testpilot and well-known message classes with their ScalaPB/gRPC compile-time runtime. |
| Nexus kind, `model/temporal/features/nexus` | State what either form of a Nexus operation does. | `Nexus.scala` declares shared Reply, Resolution, Outcome, input tokens and canonical handler/network/timeout/terminate actions; `product/Product.scala` owns NexusProduct and its terminalIsFinal Property. Both forms refine it and bind canonical actions to their own Entity through immutable aliases. The kind owns neither exports nor a System. | DSL and shared Temporal kit; neither form. A form's System refines only its local Product or its enclosing kind's Product. |
| Activity kind, `model/temporal/features/activity` | Group the forms of an activity. | One general `Activity.scala` declaring Timeout, TimeoutType, Failure, the failure input, the worker's unbound poll/respondCompleted/respondFailed/respondCanceled actions, the timers and the deadlines, and `standalone/Standalone.scala`, which binds the worker actions to its own `activity` Entity through immutable aliases. The standalone form owns its Entity, per-RPC client actions, `product/` and `system/` levels and exports. A workflow form arrives with the deferred workflow-activity work. | DSL and shared Temporal kit; current form dependencies stay form-owned. |
| Standalone Nexus operation, `model/temporal/features/nexus/standalone` | Declare one operation started through StartNexusOperationExecution, another owner of shared capability Properties. | `Standalone.scala` owns its local signature and exports; `system/System.scala` owns Phase, State, Fact and NexusSystem, its shared-Product refinement, terminalHolds and own cancelIsRequested Queries and capabilities (Closable, Terminable, Describable, overriding closedIsRejectedUniformly). `Realization.scala` owns the typed `Standalone` realization object; `exports.nexusStandalone` declares `irFile("nexus-standalone")`. | DSL, shared capability and realization kits and its enclosing Nexus kind/Product; no other form. |
| Capabilities and Properties, `model/temporal/capabilities` (shared Temporal kit) | State each shared Property once, on the companion of every capability that brings it. | Seven capability kinds `Closable`, `Terminable`, `Pausable`, `Pollable`, `Describable`, `Retries`, `Deadline`. Retries checks before-state failure eligibility, control priority and finite policy maxima; Deadline checks armed role windows and retry/terminal settlement with typed timeout facts. Repeated instances generate `<machine>.<val>.<property>` with their companion origins. A feature explicitly roots its `capabilities` section and bounds generated Queries in `queries`; new transition checks are free verify Queries with nothing to realize. `Properties.test.scala` requires each shared Property on two distinct state types and inventories actual conditional expansion. | DSL and its realization declarations (`RunExpectation`, `StatusTable`); no feature import. |
| Task queue, `model/temporal/foundations/taskqueue` (shared Temporal kit) | Declare the durable task queue a feature composes as an entity of its own. | `TaskQueue.scala`: the queue's types and signature (its actions, faults and the storage-loss assumption); `product/Product.scala`: the opaque contract `TaskQueueProduct` and its storage-loss variant `TaskQueueProductUnderStorageLoss` (a failure model); `system/System.scala`: the provider `TaskQueueSystem` that refines it (a failure model under ordinary crashes), `LossyMatchingQueue` (refining the storage-loss contract) and the negative controls `ForgetfulQueue` and `VolatileQueue`; the claims `TaskQueueSystem.properties.queueClaims` and the provider Queries each in its provider's own `queries`. | DSL only; no feature import. |
| Actors, `model/temporal/actors` | Declare the actors multiple features share. | `worker/Worker.scala`: worker actions and the `Polling` machine; `client/Client.scala`: `Client extends Actor`, the caller of Temporal's public API. | DSL only; no feature import. |
| Realization kit, `model/temporal/realize` (shared Temporal kit) | Declare what every Temporal realization says alike, once. | `Realize.scala` owns Temporal vocabulary; `Kit.scala` owns roles, bindings, default controller behavior and expectations; `Modules.scala` owns `RequestBase`, `DescribedStatus`, `HistoryEvidence` and deadline expansion; `Objects.scala` owns typed `Realizes(machine)`/`DerivesFrom(base, machine)` objects and their ordered sections; `Conversion.scala` converts those constructed values into IR under the generic realization context; `Syntax.scala` owns typed field sugar and protobuf scopes. | DSL and its realization declarations, generated Umpire IR, Temporal API and Testpilot messages; no feature import. |
| Typed-source reader, `model/framework/inspect` | Share typed source information and the liftable-subset policy between checks and export. | `Index` with source-position lookup, positioned `LiftError`, and typed tree/library admission in `LiftableSubset`; no declaration lowering. | TASTy/Quotes and the standard library; no lint, exporter, Model or system dependency. |
| Typed-source and marker checks, `model/framework/lint` | Enforce declaration order, layout, conventions, function-body subset and failure/negative-control markers before publishing IR. | Admission, order/structure/convention passes and semantic capture/helper subset checks over the shared index; marker checks over lifted Models, declaration identities and enabled-rule actions; `ModulePolicy` for a system's import rules. The gate reports source lint, IR lift and marker checks in one inspector invocation. Checked lift enables author subset lint while plain lift retains its positioned refusal precedence. | Shared inspect leaf, generated Umpire IR and the framework check runner; no exporter or Temporal source dependency. Test suites may use the exporter fixture harness. |
| IR generator, `model/framework/export` (package ``framework.`export` ``), with Temporal's half in `model/temporal/export` | Translate typed trees and constructed realization roots into Umpire IR, orchestrating source checks before lifting and marker checks before writing. | `Lifter`, compatibility `LiftError`, existing `lift` CLI root/prefix arguments and `--ir`; `checkAndLift` reports the gate's three stages over one shared index. The driver collects realization values through `ConstructedRealization`, pre-lifts their declared identities and merges conversion results atomically with the tree-produced declarations. The Java-service `Plugin` extends lint's `ModulePolicy`; Temporal's `TemporalPlugin` supplies its module map. | Shared inspect and lint modules, generated Umpire IR, ScalaPB runtime/ProtoJSON support, TASTy/Quotes and compiler libraries; the framework's half compiles without the Temporal API or any `temporal.*` source. Temporal realization messages are assembled inside the Temporal kit rather than read by framework tree matching. |
| Checks, `model/framework/check` (`lint/`, `equivalence/`) | Run tools, scope a change, hold sources to the model's rules and compare fresh lifts. | Current: `Tools`, `ChangeCheck` over a `Registry`, `ModelSources`, the source rules (comment, block form, syntax, `Boundary`), `Equivalence`, `RenameMap`, `StepLedger`, `Lifts`. Planned in fn-173.3: retire Scala `StepTables`, `Equivalence`, `StepLedger`, `InterpreterTables`, the `Coverage.scala` baseline, change-check equivalence and sealing while IR is stale. Model semantic checks use Go, not a Scala catalog walk or explorer. | Processes/files and the standard library; no Model, no Temporal half. |
| Gate, `model/temporal/check` | Verify the authored model pipeline. | One Scala program, `--update`; internal schema-generation modes `--generate-ir [--if-stale]` and `--generate-api [--if-stale]`; `--check-boundary`. It names no Model declaration: one `lift --ir` run writes every IR file the Models declare. Further entry points: `temporal.check.changed`, the change-scoped check over Temporal's coverage registry, and `temporal.check.metrics`, the source metrics. | The framework's checks, processes/files, stamped ScalaPB generation, IR generator outputs, Go checks; no authoring dependency on the gate. |
| Migration, `model/framework/migrate` (the harness) and `model/temporal/migrate` (the rules) | Prove a source rule on a copy of a feature before the sweep applies it. | `prove.sh` (`FEATURES`, a runner, a target); each rule with its proof runner and suite; a rule is deleted once the sweep applied it. | Scalafix core; the framework's checks for the lift comparison; no Model import. |
| IR admission, `tools/umpire/ir` | Load and validate an IR Model, finite input domains and exploration declarations; perform static IR checks and structural identity. | `Load`, `Validate`, `IRPaths`, `ExpectationID`, `EnumID` and admission data. Current `WithTotals`/`RequireTotals` retire in fn-173.3. Planned static checks cover rule overlap, exhaustive matches, refinement maps and role closedness, growth warnings and roots; canonical structural hashing uses the machine's behavior dependency closure. | Umpire IR, interp and realization; no answer, export or Testpilot. |
| Interpretation, `tools/umpire/interp` | Interpret Values, Machines, Channels and decision traces into reached graphs with rows computed on demand. | Current: `Build`, `NewInterpreter`, `NewInterpreterWithin`, `Why`, `Reads`; value, table, limits and interpretation data. State stays private behind `Ceilings()`, `Model()` and `Type(name)`; counts use `CountOf` and `Int64()`. Planned in fn-173.2: row provider/exploration bounded by each machine's declared reach bound, replacing whole-state enumeration and `Ceilings`. | Umpire IR and private engine; no other reader package, answer, export or Testpilot. |
| Checking, `tools/umpire/check` | Bind and answer Properties, Queries, compositions, refinements and progress claims through Go. | `Check`, `NewRealizer`, `Realizer`, `Bound`, `Declared`, `Composed`, `Unknown`; receipt, scope, trace and bound-claim data. Planned in fn-173.2: per-region closure and receipt `Reached`, `Closed`, `Checker`, with accepted open Query scopes; existing composition/progress ceilings retire. Backend orchestration belongs to answer, not this module. | Umpire IR, ir, interp and private engine; no answer, export, realization or Testpilot. |
| Answering, `tools/umpire/answer` (**planned until fn-173.8**) | Orchestrate a Query's selected checker and required confirmations over one admitted Model. | Planned narrow answer/confirm entry point taking an admitted Model and declared Query selection, returning one checker receipt per Query with identity, checker/version, bounds/scope, status and Go-replayed trace. Admit export coverage, run Go and selected backends, replay every returned trace through Go and compare required answers; do not expose backend internals or author new behavior. Concrete API names are settled in fn-173.8. | Umpire IR, ir, interp, check and export only; no internal engine, lowering, realization, lint, exploration or Testpilot. Lower layers must not import answer. |
| Realization admission, `tools/umpire/realization` | Admit executable realizations and type guards, operands and payloads. | `Admit` through the validator's `Admitter` adapter; `TypeOf`, `PayloadFields`, `GuardProblem`, `Paths`, `ACause` and typed operand data. | Umpire IR and interp; no ir, check or Testpilot. |
| Engine, `tools/umpire/internal/engine` | Evaluate claims over bounded reached graphs. | Private to interp and check; current eager finite-table algorithms remain until fn-173.2 replaces them. No whole-state catalog is required by the approved architecture. | No higher tooling layer, answer, export or Testpilot imports. |
| Export, `tools/umpire/export` | Export a Model to a backend and run it to decide or confirm a Query. | Current `Open`/`OpenWithin`, Quint export/check/agreement methods and opt-in tool runners remain second-opinion implementation. Planned in fn-173.7/.8/.9: explicit export coverage and backend Query execution/trace data; answer owns checker selection, replay admission and combined receipts. | ir, interp and check are its only model-domain dependencies; generated IR and existing process/protobuf libraries allowed; no answer, internal engine or Testpilot. |
| Retired in fn-228.6 | Case lowering (`tools/umpire/lower` and its producer), conformance, IR-bridge exploration, Model lint (`tools/umpire/lint`), `internal/runtimecel`, `internal/cli`, the `umpire-gen-cases`, `umpire-lint` and `umpire-ir-bridge` commands, `tests/testcore/testpilot`, and the Testpilot Case runtime: the root `testpilot` facade, `internal/{execution,ir,verification,testsupport}`, `cel`, `predicate`, `contract`, `casefile`, `recordedrun`, `evaluation`, `publish`, `campaign`, `replay` and the Case Temporal Driver (`temporal`, `temporal/{server,worker,binding,internal/*}`). | None; the Runner (`umpire.local/model/runner`) asks the Model to Propose, Judge and Answer and records a Run. | — |
| Durations, `common/testing/testpilot/duration` | Convert protobuf durations to whole milliseconds with checked bounds. | `Milliseconds`, `FromMilliseconds`. | Protobuf well-known types only; used by `tools/umpire/realization`. |
| Go host, `model/umpire/hosts/go/host` | Carry opaque model protocol requests over the wasm ABI. | `umpire.local/model/host`: `Host`, `New`, `Annotate`, probe transport. | Generated protocol packages and wazero; no system schemas or server module. |
| Go Runner, `model/umpire/hosts/go/runner` | Drive one Model and Driver per Run and publish the recorded Run. | `umpire.local/model/runner`: `Runner`, `Run`, `Driver`, `Main`, `Publish`. | The host and generated protocol packages; no system adapter. |
| Generated Go protocol, `model/umpire/hosts/go/gen` | Represent shared protocol messages. | `umpire.local/model/gen/umpire/{model,realization,annotations,run}/v1`. | Protobuf support; this tree contains only generated code and is checked as a whole. |
| Kettle reference, `model/umpire/reference/kettle` | Independent Go server and Kettle host/parity controls beside the Rust reference. | Module `umpire.local/kettle`, service API under `gen/kettle/v1`, `Serve`, `Options`, `Mutants`. | Shared host and generated envelopes through a local `replace` onto `../../hosts/go`; gRPC/protobuf support. |
| Runner's Temporal Driver, `common/testing/testpilot/temporal/act` | Act and observe for the Runner (fn-228.4): send a proposed request over the generated WorkflowService API, report Observations, Callbacks, Done and Faults, and let one in-process SDK worker take only the tasks a read instruction admits. | `Options` (with `Transport` for a credentialed connection), `Configure` (the `UMPIRE_TEMPORAL_*` environment), `Factory` (`New` is a `runner.NewDriver`, `Close` removes provisioning) and `Driver` (`runner.Driver`, `runner.Redactor`). | Exactly `umpire.local/model/runner` and `umpire.local/model/gen/umpire/{run,model,realization}/v1` and the Temporal read instruction `umpire.local/temporal/umpire/temporal/v1`, never the Host, guest or Realization (its tests may load the guest `umpire.local/temporal`); the Temporal API and SDK, and `temporal/provision`. It is the only Testpilot package that imports `umpire.local`. |
| Provisioning, `common/testing/testpilot/temporal/provision` | Register and remove a namespace for a Driver that creates one. | `Create`, `Clients`, `Resources`, `Cleanup`. | The Temporal API only. |
| Delivery gate, `common/testing/testpilot/temporal/control` | Hold one activity dispatch at the server's test hook until released. | `DeliveryGate`, `NewDeliveryGate`. | `common/testing/testhooks`; no Testpilot schema. |
| Commands, `tools/umpire/cmd/umpire` | The one host loop over a Model guest. | `umpire <question\|explore\|reduce\|replay\|repeat>` over `runner.Main`, registering the Temporal Driver. | The runner and `temporal/act`. |
| Canary, `tools/canary` | Run the committed activity Model through the Runner against a deployment on manual dispatch, under a lease that fences each Run's activity, and publish each Run. | `umpire-canary run\|reconcile --output --recovery`; policy v2 (`modelHash`, three coordinates) and recovery record v2. | The runner, `temporal/act`, the committed guest `umpire.local/temporal`, and the Temporal API and SDK; no consumer imports from tooling/runtime. |

### Testpilot format 4.0 ownership (historical)

**Retired in fn-228.6.** This section describes the Case runtime before its retirement: cel-go,
`internal/ir`, `cel`, `predicate`, lowering, `internal/runtimecel` and conformance no longer exist,
and nothing may reintroduce them. Of the packages it names, only `duration` remains, imported by
realization admission. The Testpilot protos stay as schema, with no runtime consumer.

Testpilot's `expression.proto` carries canonical `cel.expr.ParsedExpr` and typed domain bindings;
the pinned `cel.dev/cel-go v0.32.0` owns ordinary predicate checking and evaluation. The checked
canonical-to-engine protobuf bridge in `internal/ir` is the sole use of the engine's legacy AST
package. `common/testing/testpilot/cel` builds native operators, `predicate` exposes the same bounded
projected evaluation to conformance, and `duration` checks exact whole-millisecond conversion.
These packages depend only on Testpilot schema, its admitted catalog and standard runtime support.
The Umpire Model evaluator and neutral symbolic operand IR remain independent of Testpilot.

The closed Go import inventory admits `cel` only from lowering, its neutral
`internal/runtimecel` operand translator and conformance; `predicate` only from conformance;
and the schema-independent `duration` leaf only from realization admission, lowering and
conformance. The operand translator reads `interp` source diagnostics and `realization` path
authority, and only lowering/conformance call it. Finite `ir`, `interp`, `check` and the engine
still reject Testpilot schemas and runtime helpers. Their Duration test fixtures construct
native protobuf values locally without Testpilot imports or time.Duration narrowing.

The leaf `evidence.proto` owns generalized extraction declarations, ordered declaration-ID response
lifts and runtime evidence identities/values. `correlated.proto` owns Contract policy and complete
state/result tables. Program extraction never derives or narrows the independent Contract policy.
A complete local state is its finite atom plus ordered fields; a result is action, destination,
outcome and facts. Transitions authorize prior-state/result pairs and projections preserve result
order. Admission charges expanded structures before enumeration.

Testpilot's seven elapsed quantities and Umpire realization timeout/interval/wait/deadline hints
use protobuf Duration with exact checked millisecond conversion; counts, ordinals and logical
bounds retain their integer types. Response paths determine cardinality, local instruction
references omit entrypoint coordinates, and marker arms use protobuf Empty. Format 4.0 activates
once after source consumers integrate; managed Cases and exact Run companions regenerate at the
shared checkpoint. The old fixture compatibility passages below record earlier migrations.

### Umpire schema declaration owners

The schema rooted at `proto/internal/temporal/server/api/umpire/v1/ir.proto` has nine owners in one
protobuf package. `ir.proto` owns Model; `common.proto` positions; `value.proto` types and
values (currently finite types; native state collections and checked integers are planned in
fn-173.4); `expression.proto` functions, expressions and patterns; `machine.proto` channels,
actions, machines, compositions, refinements, monitors, assumptions and holes; `claim.proto` Properties, Scenarios,
Queries, progress and expected Runs; `operand.proto` typed operands and protobuf data;
`script.proto` activations and commands; and `realization.proto` realizations, roles, evidence,
controls and API behavior. Imports are acyclic and point toward their supporting declarations.

Go and JVM message packages remain `api/umpire/v1` and `io.temporal.server.api.umpire.v1`.
The gate generates and stamps the complete imported Umpire closure, and excludes that closure from
the linked API jar. Make prerequisites, linked Go descriptor freshness and API-linter exemptions
cover all nine files. Standard protobuf Empty supplies the marker payloads; containing oneofs
retain the discriminator. The descriptor ledger retires the old local Empty name across messages
and enums. No Umpire-to-Testpilot schema import is added.

### Shared leaves decision (fn-145 R6)

Retain separate domain owners. Umpire's `RequiredSetting` in `realization.proto` describes what a
realization requires; Testpilot's in `program.proto` describes what a Program needs at preparation.
Both currently carry `key` and `value` at fields 1 and 2. Lowering copies those two fields in
`tools/umpire/lower/realization.go`, and its inventory checks their values and declaration order.
`TestTheRequiredSettingsAreTheProgramsAsDeclared` exercises changed values, reordered, missing and
extra requirements; `TestPrepareChecksRequiredSettingsAgainstTheConfiguration` rejects a missing
or differently valued Profile setting. These are concrete loss-of-requirement failures at the
translation boundary, with existing regression coverage. Inspection found no current field or
enum mismatch that requires a shared schema dependency.

Expected Run disposition and cleanup belong to `claim.proto`; observed Run disposition and cleanup
belong to Testpilot's `run.proto`. Lowering translates their named values through Testpilot's enum
descriptors rather than assuming equal wire numbers. `TestManifestRejectsInvalidMetadata` rejects
unknown disposition and cleanup names, and `TestExpectedRunChecksEachDeclaredValueByEquality`
detects crossed and unknown observed values. Sharing the enums would couple Model expectations to
the runtime protocol without removing that semantic check.

fn-144 R17 will extend Testpilot requirements with EQUAL, AT_MOST and AT_LEAST relations, all origins,
and a symbolic role binding for namespace- or task-queue-scoped settings. Typed key declarations
and scopes belong to Temporal foundations; lowering owns the union and conflict checks; Testpilot
preparation resolves bindings and checks the Profile's value. A future adapter that copies only
`key` and `value` would lose a relation, origin or scope. fn-144 must extend its translation and
inventory checks together, including conflicting origins and unavailable scoped values. That
planned risk is not evidence that today's two-field adapter has already failed. Revisit a shared
leaf only after a demonstrated maintenance failure and an explicit decision naming its schema
owner and import direction. This evaluation introduces no shared type or dependency.

The standalone activity's executable realizations live in
`model/temporal/features/activity/standalone/system/Realization.scala`, in the System subjects' level.
Its root feature file names the typed realization objects directly in the IR exports. A Product
realization is added only for an executable Product subject; Nexus retains its root placement.

The kind level adds only its general file, an optional `product/`, and named form folders,
`workflow/` and `standalone/`. A general file holds types and signature, with no machine. Each form
follows the ordinary feature rules for a single level or Product/System levels. The structure lint
rejects a form directly under `features/`, a machine in the kind's general file and a second general
file; `model/framework/export/testdata/layout/r4/` pins each refusal at its source line. The `relay` fixture in
`layout/kinds/` proves both forms refining one kind Product. Shared entities keep their flat layout
under `foundations/` and `actors/`.

The reader's packages follow one-way layers: realization uses interp; ir uses interp and realization;
check uses ir and interp. Realization admission receives the validator's small `Admitter` adapter,
so it does not import ir. `Realizer`, `Bound`, `Declared` and `Composed` stay with check's binding.
The engine is shared privately by interp and check. Its current eager algorithms and data contracts
remain until fn-173.2; the approved Table is only the bounded reached graph.
Exploration combines reader, lowerer and runtime protocol data because its result is a model-selected
executable candidate. Campaign coordinates execution; replay owns reproduction and reduction.

The planned answer layer sits above both check and export. It admits the Query's `checkedBy` and
`confirm` selections only when every dependent ingredient is exported, replays all backend traces
through Go, and returns one receipt with checker/version and bounds/scope. Answer disagreement,
rejected witnesses and incomparable results fail closed; a Go hole or checked-integer overflow
cannot establish agreement. Check, export, ir, interp, realization and engine import no answer;
answer imports no Case runtime. Lowering, lint and Query-answer commands consume its public receipt
contract once fn-173.8 implements it, without moving backend orchestration into those consumers.
Their domain dependencies may include answer at that checkpoint; the private producer and
conformance keep their current dependency directions.

Reached Tables start at machine starts, Scenario starts and Exploration candidates, exploring
breadth-first in class/result order within a per-machine reach bound with the feature-wide default
in `model/temporal/Bounds.scala`. Closure belongs to root regions: closed means a fixpoint, open
means a bound stopped exploration. An open Query names its completed depth and reached count,
requires a reasoned `open-graph` acceptance and, for Temporal `verify`, `.open(because = ...)`.
These scopes preserve independent inconclusive statuses and make no claim about unreachable or
unexplored state values. Finite input domains remain required, by catalog or `oneOf(...)`; native
partial Maps, Sets, Lists and checked signed 64-bit state integers require no whole-state catalog.
`Inbox` remains the channel representation; a channels-as-collections library is deferred.

Behavior Fingerprints follow the SPEC's canonical IR structural hash, not explored Tables:
the machine's behavior dependency closure and own starts are included, Scenario starts belong to
Scenario and Query identity, and reach bounds/closure are excluded from machine identity.
Helper/parameter/local names and positions are erased, declarations canonicalized against source
order, and callees Merkle-hashed; semantic type/case names and meaningful sequence order remain.
An equivalent expression rewrite can change the hash. Artifact versioning, source authority and
stable rule IDs continue to apply; historical table parity is not extensional equivalence evidence
for this hash.

The Model author contract now permits generated Temporal API, Testpilot and well-known ScalaPB
message classes, generated unary gRPC method constants, and the ScalaPB runtime needed to compile
them in `model/temporal`. The DSL exposes typed schema, selector, operand and constant-message
constructors; it has no public string form for protobuf messages, methods, paths or enum names.
The gate checks Model source for remaining proto-name literals. The IR generator writes those typed
selections back into the unchanged text-bearing IR, which Go validates independently at lowering.

The framework `model/framework` names no Temporal concept (fn-114.12), and since fn-169 it holds the
lifter, the checks and the migration harness too, self-contained and Temporal-free: it compiles with
no Temporal API jar and no `temporal.*` source on its classpath, which `make check-model-boundary`
enforces (`framework/check/lint/Boundary.scala` and an alone-compile of each framework module). Its `framework/realize` keeps what
a realization of any system needs and open traits a system's kit extends; Temporal's realization
vocabulary is in `model/temporal/realize/Realize.scala`. `TestFrameworkNamesNoTemporal`
(`tools/umpire/ir/framework_test.go`) checks every file under `model/framework`, prose and
identifiers, against a word list (temporal, workflow, activity, nexus, namespace, task queue,
worker, history, chasm, matching, frontend and the eight capability kinds). A mention stays only under
an allowance that names its path and reason, and an allowance that keeps no mention fails, so the
list only shrinks. It is empty since fn-122.8 moved the capability vocabulary and shared Properties to
`model/temporal/capabilities`.

Superseded (fn-169, owner 2026-10-12): fn-114.12 and fn-122.8 recorded the IR generator and the
gate as Temporal's driver tooling, naming Temporal concepts by design. The tooling now joins the
framework; Temporal's parts sit in `model/temporal/{export,check,migrate}`, and what a mechanical move
left of Temporal in the framework's tools, as names they match, is the residual coupling fn-169.2
removes (`model/README.md`, "Residual Temporal coupling"). The record below describes historical
residue; fn-174.2 retired its realization-reader portion while retaining the generic tree helpers
used by other declaration kinds:

- `framework/export/Realizations.scala` and its `temporal.realize` vocabulary matching are gone.
  `framework/export/TreeValues.scala` retains the generic descriptor-driven tree helpers only for
  declaration kinds that still lift from trees; realization roots never enter it. Temporal
  realization assembly now belongs to `temporal/realize/Conversion.scala` through
  `ConstructedRealization`.
- `Capabilities.scala` expands a capability declaration through the companion definitions; it
  recognizes a capability by the framework's `CapabilityOf` and `CapabilityKind`
  and names none of Temporal's (`CapabilityVocabulary.test.scala`).
- The IR schema `proto/internal/temporal/server/api/umpire/v1/ir.proto`: its realization messages
  are Temporal's (`Role` and its kinds, `RequiredSetting`, the script activations
  `WorkflowActivation`, `NexusHandlerActivation` and `ActivityActivation`, `Fault`,
  `WorkflowCommand`, `NexusReply`, `NexusCompletion`, `AttemptFailure`, and `Evidence.history`); its
  machine, claim and table messages are not.
- The reader's realization admission (`tools/umpire/realization/validate_realization.go`), lowering
  (`tools/umpire/lower`) and Testpilot (`common/testing/testpilot`, with the Temporal Driver in
  `common/testing/testpilot/temporal`) are the Temporal driver.

The [parity claim inventory](umpire-migration-claims.json) maps retired oracle comparisons to
frozen evidence and surviving checks, and preserves their original commentary with source attribution. It is
history: its paths point at snapshots fn-124.7 retired.

## Public types and file boundaries

The signatures and migration transfers in this section record the existing implementation and
fn-115 evidence. In particular, `Scope.Progress`/`ComposeCeiling`, full Table fingerprints and
Scala step-table compatibility remain current or historical surfaces, superseded by the planned
fn-173 changes above; this section does not require their retention after those tasks land.

Interp and check own the public names for every engine type their APIs expose. Their aliases for
the required immutable/data contracts preserve Go assignability and existing algorithms. Only these
two packages import `internal/engine`; the producer and export use the public reader contracts.
`Scope.Compose`, `Scope.Progress`, receipt witnesses/limits/refinement/monitor results, `Machine.Table`
and the bound Property/Query surfaces must all use reader names in their declarations. `Table`,
`Query`, `Atom`, `Result`, `Trace`, `TraceStep`, `Limits`, `ComposeCeiling`, `ProgressKind`,
`RefinementFailure`, `MonitorVerdict` and the full dependency closure of those public signatures are
included. The audit's `reader_checker_public_aliases` names the additional direct external uses,
including canonical `Fingerprint`/`Quote`/`Group` operations used by the producer. Methods retain
behavior and ordering; this task does not replace the algorithms or add admission rules.
Existing reader `Found`, `NotFound` and `LimitReached` names remain their current `ReceiptKind`
constants. The required checker `Outcome` receives a public alias; outcome comparisons explicitly
convert the existing same-valued constants (for example `Outcome(Found)`). Verify equality with the
private checker constants; do not merge the receipt/outcome types or add a synonym constant family.

`Row` is public data in the `Table.Rows` signature closure. `NewTable`, `TableSpec`, `KeyProperty`,
`KeyScenario` and `KeyFind` remain private-checker operations, not public test-construction APIs.
fn-113 Part C deleted the checker's typed declaration layer (machine, step, Property and Scenario
builders, reflection domains and keys), its test-support files and the production branches only
those fixtures reached. The checker reads keys only, and the reader no longer aliases typed class
data (`ActionDecl`, `Actor`, `Entity`, `ClassExample`, `TableClass`, `ProgressAnswer`). Recorded exceptions to the
outside-caller rule: `export` keeps the surface listed in its row although no package imports it,
and exported `Table` fields are unaudited (`Table.Stuck` is read by reader tests only).

Lowering exposes `Identity` and `IdentityFor` from its private producer so exploration and fixture
callers have no direct `caseproducer` import. Move producer-specific white-box tests with that
private implementation; higher-level tests use the lowerer's public API. Public declaration
inventory and actual selector callers are recorded in `surface_inventory`. R18's cleanup keeps
only the entry-point families above, the types required by their signatures and verified external
uses; test-only implementation helpers become package-private. Export's test-only entry points
become private when the same-package tests can use them. No compatibility forwarding package remains.

`recordedrun.Write` receives the actual exclusive creation logic from `replay.WriteRecordedRun`;
replay's aliases and pure forwarding functions disappear from live copies. CLI publication/path
forwarders likewise disappear. `Proposal`/`WriteProposals` belong to publication; replay keeps a
small local reporting function for its two `WriteLine` callers. This avoids making runtime replay
depend on a package that knows the Model output boundary. Preserve all existing write conflict,
symlink, partial-write, error and cancellation behavior during these moves.

The existing failure-signature encoder and tests move from `tests/testcore/testpilot/signature*.go`
to `common/testing/testpilot/recordedrun/signature*.go`: a canonical diagnostic projection of a
closed Run belongs with its recorded evidence. Preserve `SignatureLinePrefix`, `Signature`, its
rule/diagnostic data, `RunSignature`, `ReportSignature`, `Line` and `Equal` unchanged. Functional
assertions and the command's `TestSignatureOfReadsTheLineALiveTestLogs` use this same encoder;
the test still checks the actual encoder against the command parser. Task 4 copies the code/tests
and switches functional callers. Keep the old functional definitions only while the frozen tools
command still needs them; task 6 switches the copied command test and removes those definitions.
No tooling-to-functional-wiring exception or forwarding aliases remain after cutover.

Producer test claims have explicit transfers in the audit's `test_claim_transfers`. Task 2 captures
all seven original Nexus typed/keyed pairs and their full Cases, table/claim/witness facts and
identical realization/source inputs. Task 3 replaces the legacy parity oracle with equivalent
admitted IR fixture comparisons against that frozen evidence in the live lowerer; the original
`model/go/caseproducer` tests remain frozen. Task 5 omits their superseded oracle imports only in
the new producer copy. This is separate from R22 execution pin replacement. Task 5 carries all ten occurrence/projection/fingerprint tests into the private
producer copy, constructing the equivalent generic job IR with reader admission/Build and
`NewRealizer.Find`. Preserve domains, result/fact ordering within each row, nondeterministic alternatives, evidence,
IDs, paths, predicates and limits. The two remaining keyed-test claims move into same-package
lowerer tests using reader Queries and local realization conversion: retry/backoff-only node
placement, and exact Produce/Preflight evidence-source rejection agreement. Keep their assertions
and comments. No raw checker constructor is added to the reader for these tests.

The synthetic job fixture alone needs a closed representation projection. The initial row-only
proposal assumed declaration-order actions; task-5 execution disproved that assumption because
reader `classes` sorts action keys. Preserve the original baseline and interpreter. For exactly
`oracles/job/{once,retried,closed}` use these original-to-reader sequences:

| Field | Original | Admitted reader |
| --- | --- | --- |
| Actions (also the corresponding Definition IDs) | submit, take, fail, settle, finish, drop, check, close | check, close, drop, fail, finish, settle, submit, take |
| Row keys | idle-submit, queued-take, queued-drop, running-fail, waiting-settle, running-finish, running-check, running-close | idle-submit, queued-drop, queued-take, running-check, running-close, running-fail, running-finish, waiting-settle |
| Reachable | idle, queued, running, dropped, waiting, done | idle, queued, dropped, running, done, waiting |

Require exact vectors, lossless original decode/re-encode, and complete per-key row equality. The
only result projection is `waiting-settle.results[0].facts` and `running-check.results[0].facts`:
require exactly one result in each named row, original nil and actual non-nil empty, and project
those two expected fields to empty arrays in memory. All other result/fact/evidence ordering and
contents remain exact. Project only these fields of expected `table.json` and the corresponding
action-ID order in `ids.json`; never mutate reader outputs to imitate handwritten constructors.

Query canonicalization hashes the ordered action domain, so this exact ordering also changes the
Query's Case provenance fingerprint and the resulting Case identities. Permit only that derived
chain in `case.json` and `identity.json`. Establish the original Query fingerprint against frozen
Case provenance after replacing only the admitted canonical Query's action-domain fingerprint
with the fingerprint of the exact original action-ID vector. Equivalently, reconstruct the canonical descriptor from frozen scenario/property/table inputs,
derive its property fingerprint with `Table.PropertySemantic`, and require the original Query
fingerprint before changing only the action domain. This cryptographically checks every other
canonical field. Then the exact admitted action-domain fingerprint derives the expected new
Query fingerprint; replace only the uniquely identified Query provenance definition in an
in-memory expected Case, and recompute its two identities through the existing runtime encoders.
Require lossless original Case serialization and verify its original identities before projection.
Never copy actual Case bytes or identities into expected data. Program, Contract, Query answers,
witnesses, property/scenario/target fingerprints, and all other artifacts stay byte-strict.

Negative checks reject other orders, missing/duplicate/unknown keys, changed result cardinality,
nonempty or reordered facts, unlisted nil/empty conversions, changed canonical Query fields,
unrelated provenance changes and incorrect hashes. Apply the projection before the existing
whole-inventory comparison so inactive variants and unknown/missing/extra entries remain checked.
All 1,411 immutable snapshots and production Model comparisons remain unchanged and order-sensitive.
Any additional difference stops this synthetic fixture transfer.

The original checker `model/go/umpire/replay_test.go` also imports handwritten Models. Leave it
frozen and omit it from the private checker copy. Transfer its comment and exact witness replay /
rebound-Action-ID rejection assertions into the reader-owned replay tests over admitted activity
and Nexus IR, with a nonvacuous expected Query inventory. This keeps checker tests independently
scoped; `Query.Replay` already belongs to the public Query method closure.

The reader's `load.go` retains file decoding; `validate.go` owns whole-IR admission, with key and
identity admission in `validate_keys.go` and realization admission in `validate_realization.go`; operand typing
and protobuf paths live in separate concern files, with focused tests. The IR generator first gains one
typed context carrying `Quotes`, definition/symbol indexes, source-prefix mapping and declaration
accumulators. Extract types, expressions, declarations, realizations, compositions and claims into
concern files around that context; the entry point only arranges inspection and output. This is
structure work, not a second interpretation of the Model. Task 8 delivered it as ten files in
what is now `model/framework/export`: `Context`, `Types`, `Constants` (values folded at lift time), `Expressions`,
`Declarations`, `Realizations`, `Compositions`, `Claims`, `Lifting` (the class the concern traits mix
into) and the entry point `Lift.scala`.

Task 9 renamed the IR namespace to `umpire/v1` (Go alias `umpirespb`, JVM package
`io.temporal.server.api.umpire.v1`); `modelir` is renamed, not kept. The migration manifest and
audit JSON keep the old names as the captured baseline. `tools/umpire/ir/schema_test.go` with
`testdata/schema/before-rename` freezes the pre-rename descriptor and wire bytes and names the old
package on purpose, so retired-name checks exempt them; its 14 `.gz` files outlived the 1,411
migration goldens. `make lint-api` passes; `proto/api-linter.yaml` excludes the IR proto's path from the AIP
rules it predates.

Task 12 made the dependency rules executable in `tools/umpire/ir/ownership_test.go` and the
Testpilot boundary tests: production and test imports are checked separately, the lowerer reaches
`explore`, `conformance` and `recordedrun` only from `package lower_test`, and no live file imports
an archive. The lowerer's and
conformance's own tests may import the Temporal Driver package for admission (catalog and derived
Profile) only. `TestEveryToolingPackageHasALiveCaller` requires a live importer or a Make/CI runner
for every package under `tools/umpire`. `TestModelNamesNoRetiredFrontEnd` is the vocabulary check
outside `model/`; the gate runs it as its first step, also under `--skip-go-checks`. The export
runner is opt-in Go tests in `tools/umpire/export/tools_test.go` (`UMPIRE_BACKENDS=require`, pinned
tool versions there); `run.sh`, `UMPIRE_BACKEND_FLAGS` and `--install` are gone, and
`make umpire-install-backends` warms Quint's pinned npm package; `make umpire-check-backends`
runs its evaluator agreement and Apalache bounded checks. `TestQuintOwnsBackendExport` currently
enforces the single export backend and its installation boundary. fn-173.7/.8/.9 must update
coverage and ownership checks with the selected backend routes; this current test is not a
permanent prohibition on the approved backend-answer architecture.
`common/testing/testpilot/campaign/integration_test.go` is removed (its subject is archived), CI no
longer builds the descriptor set, and `make lint-code` shares `lint-code-fast`'s Go-only patch.

## Migration goldens, retired

fn-124.7 retired the immutable migration goldens with the test-only helper that read them: its
closed projection and original-baseline delta, the original archive, the 48 reader snapshots and the
1,363 artifact snapshots, and the migration, original-baseline and comparative-Model tests. Since
fn-126.4, fn-126 proves equality with the reader projection (tables and every Check receipt), not
with these snapshots. The behaviour tests only the goldens exercised were restated against
`model/ir` first: they pin counts, rows, witnesses and Case members, and each fails on a seeded
change to the live IR. What a Model means is pinned by the reader's tests over `model/ir`; what a
Case contains, by the managed Case trees `umpire-check-cases` compares byte for byte. The
schema-rename capture `tools/umpire/ir/testdata/schema/before-rename` stays.

Only the lowerer's external test package may import exploration, conformance, Testpilot runtime
and recordedrun. This integration-test exception does not permit those edges in lowerer production
code. `tableSide`/`sideOf` stay as dedicated reader test support.

## Scala layout and commands

`model/framework/dsl/project.scala` is the framework core's directive file: compile the DSL and the
realization core with `model/framework/dsl` and `model/framework/realize` alone. `model/project.scala`
adds the Temporal Models' settings: compile/test/package Models with that file plus
`model/framework/dsl`, `model/framework/realize` and `model/temporal`'s Model folders. Tests are
co-located with what they test (no `test/` folders); compiler and IR generator refusal fixtures live
under `framework/export/testdata`. The IR generator, the checks and the migration rules have
independent `project.scala` files, and a Temporal tool builds from its framework half first and its
Temporal half second (`scala-cli <command> model/framework/export model/temporal/export`). Their generated jars, fixtures
and stamps live under ignored `model/build`; the gate names source roots explicitly, so a broad
recursive Scala compile never merges the authoring, IR generator and gate projects. Lint/format commands
visit these explicit roots. fn-113 removed the native Scala evaluator and transferred its independent
checks to Go tests over the IR; the audit is `umpire-scala-evaluator-audit.md`. The Nexus domains and
effects live in `model/temporal/features/nexus/product/Product.scala` and each form's `system/*.scala`, in the `effects` of each machine's object, with no kernel or prelude package; each machine's `rules` say when its effects fire.
Owner-approved early cleanup in task7, immediately after relocation, removes only the unused
`Canonical.scala`, `Lower.scala` and `Alterer` plumbing from fn113 Part A / R1 after checking callers;
this does not retire the evaluator. Task 7 delivered it: the two files, `Alterer` and the helpers left without
a caller are gone, and `model/framework` outside tests was 2,415 lines at that point; after fn-113 Part D
it is 1,191 lines.

Checked IR, generated Cases, specimens, specs and README move directly under `model/`; SEMANTICS
moves under `model/framework/`.
Reserve `model/examples` for fn-119 authoring examples and `tools/umpire/explore` for fn-120 explorer
work; no example or UI is implemented here. Root lists kept their meaning until fn-114.1 moved them
into Scala: each IR file is an `irFile` val beside its Models, and the gate's `Roots.scala` is gone.
Since fn-114 closed, the Models own every root of `model/ir`: each Model folder's feature file
declares its IR files and their roots in its `object exports` (for example `features/nexus/workflow/Workflow.scala`), and the
gate and the IR generator name none of them, so adding a root or an IR file edits only the Models.

| Command after migration | Existing source / behavior |
| --- | --- |
| `make umpire-check-model` | `umpire-check-scala`; runs `mise exec -- scala-cli run model/temporal/check`. |
| `make umpire-gen-model` | `umpire-gen-scala`; same program with `-- --update`. |
| `make lint-model`, `fmt-model`, `fix-model` | Model formatting and language checks, explicit new roots; IR jar prerequisite uses the gate's schema-generation mode. |
| `make umpire-check-cases`, `umpire-gen-cases` | Run `umpire-gen-cases` without/with `--update`; default managed tree is `model/cases`. |
| `make umpire-check-lint` | Run `umpire-lint` over selected features' existing `generated/ir.json` files; default to all lifted primary models, skipping unlifted features and legacy IR. Findings are fixed or accepted with a reason in the feature's `generated/ir.lint.json`. |
| `make umpire-check-fixtures`, `umpire-gen-fixtures` | Same generator with `--kind functional` without/with `--update`; only the explicitly supported R22 replacement subset is managed. |
| `make canary-check-case`, `canary-gen-case` | Same generator with `--kind canary` without/with `--update`; one pinned Case and reviewed policy identity. |
| `make umpire-ir-bridge` | Build the existing `cmd/umpire-ir-bridge` to `.build/umpire-ir-bridge`; default IR root becomes `model/ir`. |
| `make umpire-run`, `umpire-fuzz`, `umpire-repeat`, `umpire-replay`, `umpire-assess` | Existing command build interfaces. |
| `make umpire-fuzz-run`, `umpire-replay-run` | Existing deployment flags; use the live IR bridge and model-declared set/target names. |
| `make umpire-repeat-run`, `umpire-assess-run` | Existing caller-supplied Run/Case interfaces. |
| `make umpire-check-exploration-bridge`, `umpire-check-replay-bridge` | Existing live protocol claims exercised against the IR bridge; no Lake invocation or success-by-skip. |
| `make umpire-check-live-tests` | Existing live functional gate, preserving a nonzero executed-test floor. |
| `make umpire-check-backends` | `UMPIRE_BACKENDS=require` opt-in Go tests in `tools/umpire/export`, using the copied pinned `quint.sh`; preserve existing tool versions/output environment controls. |
| `make umpire-check-testpilot-protocol` | Preserve the live protobuf comment/schema check; remove its retired frontend generation/build tail. |
| `make canary-build` and other live canary build/run targets | Preserve production canary build and existing operator interfaces. |
| `make umpire-rerecord-pinned-runs` | Remains an explicit operator action targeting current live fixture identities; never runs automatically as part of this migration or ordinary tests. |

Since task 10 the model Make entrypoints run the Scala gate, now at `model/temporal/check`, preceded by the gate's
own test suite. The IR generator's `project.scala` compiles `check/Tools.scala` so its tests share
the gate's process seam; scala-cli has no test-scoped file directive, so this one test-time
dependency of the IR generator on the gate is a recorded exception. The IR generator suite rewrites its expected
files only when the gate passes `UMPIRE_LIFTER_UPDATE`.

Since fn-156 the authoring, IR generator and positive fixture builds compile with `-Werror`,
`-Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns` and
`-language:strictEquality`; the gate has strict equality without `-Werror`. `lint-model` also runs
`lint-model-conventions`, the IR generator's `framework.export.lintConventions` over the compiled TASTy
of the authoring, IR generator and gate roots with their tests. It holds Scala imports and qualified
references to the "Permitted domain dependencies" column above (`R4.module-import`,
`R4.module-reference`), bans unqualified cross-level `Phase`/`Fact`/`State` imports (`R5`), whole-case
enum/state wildcards and infix `in`. These checks live in the IR generator, beside its declaration
order lint, because they read types; fn-141 inherits them with the lint extraction.

Since task 14 the IR generator's fixtures are plain `.scala` files in `model/framework/export/testdata`, which
`//> using exclude` keeps out of the IR generator's build; the tests copy a fixture to scratch before
building it. Scalafmt checks all fixtures; fn-113 formatted the five formerly excluded `lifts`
sources after its golden comparison began projecting positions by file. Scalafix runs on `testdata/lifts` as its own
root with all rules (`-Werror:false`, because the unused parameters it keeps are fixture content);
the refusal fixtures `unsupported`, `werror`, `crossed`, `nonfinite`, `samestate` and
`realizationRefusals` stay outside it. `lint-model` and
`fix-model` depend on `model/build/model-scala.jar`, packaged by the gate's command.
A documented `--skip-go-checks` option may omit only its embedded Go test invocation when combined
verification runs the complete live Go suite separately with `test_dep`; default invocation still
runs Go checks. Record the check/update commands and that covering suite together. Scala checks,
lifting and Case generation/verification remain enabled. Explicit temporary cache directories are
retired by guarded renames into ignored history instead of recursive deletion. Existing CI keeps
runtime/canary unit checks, harness checks and the production canary build, and checks live managed
Cases; obsolete renderer steps are removed. Canary-kind generation/check wiring is added with its
implementation in task11, not represented by a stub during relocation.

The retained executable set is `umpire-run`, `umpire-fuzz`, `umpire-repeat`, `umpire-replay`,
`umpire-assess`, `umpire-gen-cases`, `umpire-ir-bridge`. Their old source files are copied out of the
frozen tools tree. Delete targets/variables whose only purpose is archived generation, inventory,
views, model-module indexes or retired frontend checks. `commands` in the audit records the exact
old bodies and disposition; no old target becomes a stub. Preserve current profile JSON as live
configuration data while archiving its old renderer. The existing Umpire CI jobs use the root mise
configuration and live Go/fixture checks, retain the canary harness check and production build, and
receive no new JVM job. Default unit tests neither run Scala nor regenerate fixture bytes.

## Archive and intermediate-state rules

These rules record how fn-115 created the `model0/` and `tools/umpire0/` archives. fn-124 deleted
both on 2026-10-04; git history keeps them.

1. Check the manifest's original hashes and index before any extraction. Task 2 captures the golden
   while the current paths and original oracles still exist. Task 3 transfers each oracle claim
   before removing its import; original model/go source remains untouched for the archive.
2. Promote neutral helpers by copying into their final Testpilot destinations. Update live consumers
   outside old `tools/umpire`; the original tools tree stays byte-identical. Its old commands can
   continue using their old helpers until the namespace swap. Copied helper tests read copied
   fixtures, never fixtures through an archive path. Leave Model-specific `campaign/bridge_live_test.go` and `replay/bridge_live_test.go`
   in the frozen original tooling during task 4 rather than copying its Lean executable dependency
   into Testpilot; task 6 transfers those claims to live IR bridge integration tests before archiving.
3. Task 5 copies the checker to `model/scalav2/goir/internal/checker` and the producer to
   `model/scalav2/goir/testpilot/internal/producer` **before** switching reader Query ownership.
   In that same tested task, the lowerer uses its local producer copy, and that producer uses public
   `model/scalav2/goir` types/operations. Never pass the copied checker's Query to the original
   `model/go/caseproducer`. Retarget external identity callers through public lowering. Task 6 moves
   both private subtrees with their owners. Original checker/producer and old tools stay unchanged.
4. In one tested relocation task, guard that archive destinations do not exist, rename old tools
   to `tools/umpire0`, place the selected live command/CLI copies into the newly free tools namespace,
   move the reader/lowerer/conformance/export/explore there, and move authoring to `model/`. Add
   nested archive `go.mod` files and required README metadata. Update every live import and essential
   Make/gate/test path within this task. Never finish a task with a live archive import.
5. Move each retired model tree into its matching `model0/<tree>` directory. The original Quint
   launcher also has a live copy under export. Move the four `.plans/lean` documents individually
   into `.plans/archive/lean`: all four destination names are absent; the existing sixteen documents
   remain byte-identical. Only the archive index is edited.
6. Keep path/import-only relocation evidence separate from subsequent cleanup. Use guarded
   filesystem operations and content manifests; do not stage, commit, create worktrees or recursively
   delete. Preserve every original byte recorded in `protected_originals`, except the explicit
   archive metadata additions. Cache exclusions are listed, not guessed from a broad `testdata` rule.

Original legacy source-dependent tests outside the archive trees also require explicit disposition:
preserve full original bytes of `tests/testcore/testpilot/exploration_bridge_test.go` and
`protobuf_lean_authoring_test.go` under `tools/umpire0/functional-fixtures/` with collision guards.
These are explicit additions to the isolated archive, not edits to its 843 protected originals.
Their live replacements retain useful protobuf schema assertions and transfer bridge protocol claims
from the old binary-dependent functional helper, `campaign/bridge_live_test.go` and
`replay/bridge_live_test.go` to the existing IR bridge tests before removing old executable paths. Preserve all runtime/fixture/negative-test claims. The audit does not authorize
archiving a functional test merely because its Case still needs an R22 exception.

The owner explicitly deleted `.github/workflows/umpire-production-canary.yml`. Preserve its obsolete
workflow-only test, `tools/canary/workflow_test.go`, under
`tools/umpire0/functional-fixtures/workflow_test.go`; keep the shared repository-root helper and
checks of the remaining offline CI in the live package. Runtime authority and policy checks remain.

During task 4's copy-before-archive state, frozen
`tools/umpire/regression/TestTestpilotOwnsCaseProtocolAndRuntime` rejects the six newly permitted
Temporal Driver edges: production `campaign/run.go` and `replay/report.go` to `temporal/binding`,
and fixture tests `campaign/integration_test.go`, `evaluation/admission_test.go`,
`replay/driver_test.go`, `replay/key_test.go`. Keep its original bytes. Transfer its useful generic
runtime dependency restrictions and negative checks into live Testpilot ownership tests with only
these map-approved allowances. The transitional broad Go command excludes exactly that obsolete
test with `-skip '^TestTestpilotOwnsCaseProtocolAndRuntime$'`; every other old regression test still
runs. Record the original failure and revised green command. Task 6's archive exclusion removes
this temporary command exception; final main-module checks have no such skip.

## Exact artifacts and fixture compatibility

Only the eighteen observed source paths and twelve exact Model source labels in the manifest may
change in their typed fields, plus the six lifter fixture renames of task 14 (`*.scala.fixture` to
`*.scala`), which the golden helper applied as a separate exact `source_path_renames` list to the
current IR only; the frozen mapped goldens kept the captured `.scala.fixture` spelling in their
derived digests and Case IDs. Preserve every line and column; historical-attribution removal leaves
blank lines where necessary. New, unexpected source strings fail migration verification rather than
being accepted by a generic prefix normalizer. The six checked Models and six expected fixture IR
files form the initial IR inventory; all their Queries and unsupported standings enter task 2's
baseline. Confinement remains the existing source-position test, not a new runtime loader rule.

fn-113 extended this historical migration contract with the golden helper's closed projection
(retired by fn-124.7): positions by file, alpha-normalized parameters, exact
listed function and type renames, and the Nexus source-path rename. Only a mapped-original type
catalog with an actual listed type substitution is re-sorted by its resulting names; current type
order, enum cases and record fields remain strict. Exploration Case IDs may vary because they hash
the whole candidate IR. Tables, Definition IDs, refinement rows, fingerprints, Query answers,
Query Case bytes and every other exploration Case byte retain the independent frozen baseline.
fn-114.9's folder moves were two more append-only lists there, `source_path_moves` and
`source_package_moves`; `tools/umpire/ir/layout_test.go` still fails if an old path reappears.

Recompute hashes in the manifest's dependency order. Ordinary generated Cases retain the same
`IdentityFor` inputs and IDs while mapped provenance changes their canonical checksum. Exploration
hashes the entire deterministic candidate IR, so the candidate digest, digest-derived Case/Program/
Contract IDs and local references, Case checksum, proposal payload/hash/name and rendered source
links change together. Their semantic data and candidate ordering stay fixed. Model assessment's
source-stripped identity remains identical, including its persisted `goir.model/v1` domain. Stable
`scala.`, `scala.explore.`, `scala-model` and `goir.model/v1` strings are compatibility data, not
historical package names to rewrite. The proto rename freezes descriptor structure and wire meaning.

R22 replacement is a separate artifact decision. The inventory contains sixteen already-generated
Scala Cases, eight legacy functional Nexus/control Cases with actual replacement Queries, one
pinned canary replacement, five functional/synthetic exceptions and seventeen runtime-conformance
exceptions. Each row records original bytes, Program/Contract inspection hashes, sources, consumer
evidence, the exact replacement Query where one exists, or the absent declaration/missing primitive.
The missing functional declarations concern two-operation Nexus correlation, System Info, workflow
start, workflow-worker outage/recovery and the synthetic payload admission fixture. No fictional
Scala Query is used as a replacement.

The canary uses the existing `nexus-workflow.json` `nexusSystem/syncCompletion` Query. Its pinned
bytes, explicit policy Case identity, fixture constants and Profile-dependent expectations change
as one reviewed replacement. Require admission under the existing literal role/opcode/command and
ceiling authority; never expand production credentials to make migration pass. Historical Runs, evaluation receipts and saved proposals are never rewritten to impersonate new
executions. Recorded Runs store a Case checksum rather than embedded Case bytes. Before replacing
the execution pins, copy their exact old Cases into
`common/testing/testpilot/replay/testdata/nexusCallerControl-forgedCompletion-case.json` and
`tools/canary/assessment/testdata/nexusCallerCanary-syncCompletion-case.json`. Historical record tests
use these companions with the recorded original Profile/binding identities. These are historical
compatibility fixtures, not execution pins or absent-primitive exceptions. Existing crossed-identity
rejection remains the compatibility decision.

The runtime-conformance fixtures keep their exact bytes and expected outcomes under R22's exception.
There is no Scala synthetic Model/Query/Realization for those scripted Driver cases; admission-
negative Cases additionally cannot be emitted by legitimate lowering because their invalidity is
the test. Their current hashes, adjacent expectations and archived renderer arguments preserve
inspectable mutation provenance. Task 11 does not invent a negative-fixture authoring language or
replace them with unrelated successful Queries. Ordinary tests keep using these retained files.

Task 11 executed this inventory: eight functional Cases and the canary pin are lowered from the Scala
model, and the 22 exceptions keep their bytes. `umpire-gen-cases --kind model|functional|canary`
writes three managed trees (`model/cases`, `tests/testcore/testpilot/testdata/generated`,
`tools/canary/casebinding/testdata`); a pinned Case is its `model/cases` file byte for byte, selected
by `lower.SelectCases`. `make umpire-gen-fixtures`/`umpire-check-fixtures` and
`canary-gen-case`/`canary-check-case` are the entry points. After a Model change the order is
`umpire-gen-model`, `umpire-gen-fixtures`, then `canary-gen-case` with the policy `caseIdentity` and a
new canary record. The control's pinned-run entry still names the historical record; re-recording it
requires changing the companion, `controlProfile`, `controlKey` and the receipt goldens together.
Recorded R22 gap: a Property phase clause lowers to no `STATE` rule (only whole-state equality
does), so the legacy control rule `state-succeeded` has no lowered counterpart. Per-fixture results
are in the manifest's `fixtures[].result` and `fixture_migration`.

All subsequent tasks ran the task-2 golden, until fn-124.7 retired it, and run the relevant consumer
checks. Closing checks compare gate
outputs against the dirty authorized baseline, include existing production/test import directions,
verify future generated fixtures are not ignored, and enforce R25 from a file outside `model/`.
An independent reviewer evaluates this map before implementation moves; the conductor owns the
review receipt and Flow lifecycle.
