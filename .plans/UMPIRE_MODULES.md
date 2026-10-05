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

## Module ownership

Paths below are destinations. Public entry points retain their current signatures and behavior,
except that caller-visible checker and producer types acquire names in their owning public facade.
Returned data types, their used methods and error/status values are part of the same contract.
Standard libraries, existing support libraries and generated schemas are permitted where already
used; the dependency column constrains domain ownership, not those ordinary dependencies.

Each module's README describes it for its users: `model/README.md` for the whole system,
`tools/umpire/README.md` for the Go tooling, and the READMEs beside Testpilot, the Temporal Driver,
the functional fixtures and the canary.

| Module and destination | One job | Public interface | Permitted domain dependencies |
| --- | --- | --- | --- |
| Umpire IR, `api/umpire/v1`, `proto/internal/temporal/server/api/umpire/v1` | Represent a lifted Model. | Existing protobuf messages under the new package names. | Protobuf support; no Testpilot schema dependency added. |
| DSL, `model/umpire` | Declare finite Models in Scala. | Authoring and realization declarations, with the realization script helpers and the open traits a system's kit extends (`Addressee`, `Activation`, `Instruction`, `Recorded`, `Setting`) in `umpire/realize`; the capability mechanism, naming no capability (`CapabilityOf`, `capabilities`, `except`, `overriding`, `cited` in `Capabilities.scala`; `CapabilityKind`, `Law`, `Catalog` in `Catalog.scala`); no native evaluator. | Scala standard library and generic ScalaPB typing/runtime metadata; no Temporal-specific, IR generator or gate import, and no Temporal name (`TestFrameworkNamesNoTemporal`). |
| Models, `model/temporal` | Declare Temporal behavior. Features live under `features/`, the entities they share under `shared/`. A feature reads top to bottom in one feature file per folder, an object per machine, held to that order by the IR generator's declaration-order lint, and splits its subjects into subpackage folders laid out alike, as `features/standaloneactivity` with `record/` and `withTaskQueue/` does. Every Model folder holds one feature file named after it (`features/nexuscaller/NexusCaller.scala`, `shared/taskqueue/TaskQueue.scala`, …) beside its `Realization.scala`; no folder splits its files by kind. The bounds more than one folder runs its Queries under are declared once, in `shared/Bounds.scala`. | Existing machine, claim, Query and realization roots, and one `irFile` declaration per checked-in IR file, in its feature file's `object Files`. | DSL and shared feature kit; generated Temporal API, Testpilot and well-known message classes with their ScalaPB/gRPC compile-time runtime. |
| Standalone Nexus operation, `model/temporal/features/nexusoperation` | Declare one operation started through StartNexusOperationExecution, the laws' second entity. | `NexusOperation.scala`: the machine object `Operation` with `nexusOperation` and `laws.operationCapabilities` (Closable, Terminable, Cancelable, Describable, overriding closedIsRejectedUniformly), `OperationRealization.standalone`, `Files.nexusOperationFile` (`irFile("nexus-operation")`). | DSL, the laws and the shared realization kit; no feature import. |
| Capabilities and laws, `model/temporal/capabilities` (shared Temporal kit) | State each law once, for every entity that declares the capabilities bringing it. | The capability kinds `Closable`, `Terminable`, `Cancelable`, `Pausable`, `Pollable`, `Describable` (`Capabilities.scala`); the law objects `terminalStatesAreFinal`, `closedIsRejectedUniformly`, `terminateSettles`, `cancelIsRequested`, `pausedIsNotDispatched`, each with its server citations; the one `given catalog`. A feature declares its own capabilities in the `laws` object of its machine object. | DSL and its realization declarations (`RunExpectation`, `StatusTable`); no feature import. |
| Task queue, `model/temporal/shared/taskqueue` (shared Temporal kit) | Declare the durable task queue a feature composes as an entity of its own. | `TaskQueue.scala`: the opaque contract `DispatchQueue.dispatchQueue`, the providers refining it in `MatchingQueue` (`matchingQueue`, `lossyMatchingQueue`, the violating controls), `MatchingQueue.properties` (`queueLaws`, `storageLossDrops`) and the provider Queries in `MatchingQueue.queries`. | DSL only; no feature import. |
| Realization kit, `model/temporal/realize` (shared Temporal kit) | Declare what every Temporal realization says alike, once. | Temporal's realization vocabulary in `Realize.scala`: `Role`, `RoleKind`, `RequiredSetting`, `WorkerActivation`, `WorkerInstruction`, `FaultKind`, `WorkflowHistory`. In `Kit.scala`: roles, environment bindings, the correlation window, the controller script, the kit's poll interval and deadlines, run-record evidence helpers, `temporalRealization`, the expected Runs `satisfied` and `inconclusive(reason)` a Query declares; sugar `field(_.name) :=` in its `Syntax.scala`. | DSL and its realization declarations, generated Temporal API and Testpilot messages; no feature import. |
| IR generator, `model/irgen` | Translate typed trees into Umpire IR, after holding every source it reads to its declaration order (`Order.scala`: no read before a declaration, no initialization cycle, a feature file's reading order and places). | `Lifter`, `LiftError`, existing CLI root/prefix arguments, and `--ir`, which lifts every declared IR file (or the ones named) in one run, writing the law sidecar `<file>.laws.json` beside each IR file whose Models declare capabilities. | Generated ScalaPB IR and linked API metadata, ScalaPB runtime/ProtoJSON support, TASTy/Quotes and compiler libraries; Models and the Temporal kit read as TASTy, never source imports; the kit's vocabulary matched by fully qualified name. |
| Gate, `model/check` | Verify the authored model pipeline. | One Scala program, `--update`; internal schema-generation modes `--generate-ir [--if-stale]` and `--generate-api [--if-stale]`. It names no Model declaration: one `lift --ir` run writes every IR file the Models declare. A second entry point, `umpire.check.metrics`, prints source metrics. | Processes/files, stamped ScalaPB generation, IR generator outputs, Go checks; no authoring dependency on the gate. |
| Reader, `tools/umpire/model` | Interpret the meaning of one admitted Model. | `Load`, `Validate`, `Check`, `Build`, `NewInterpreter` (with `Why` and `Reads`, the decision trace), `NewRealizer` (with `Refinement`, the refinement Check reads), `TypeOf`, `PayloadFields`, `GuardProblem`, `Unknown`, `ExpectationID` and `EnumID` (the one spelling of an expected Run's ids); `IRPaths`, which leaves out the law sidecars, `ReadLawSidecar` and `LawViolations`; existing value, receipt, scope, table and bound-claim data needed by consumers. | Umpire IR and its private checker; no Testpilot package or schema. |
| Checker, `tools/umpire/model/internal/checker` | Evaluate finite table claims. | Private to the reader; retained implementation declarations only. | No Testpilot or lower/conformance/export imports. |
| Lowering, `tools/umpire/lower` | Produce a Case from a Query's witness. | `NewProducer`, `Producer.Lower`, `Realizable`, `EvidenceElement`, `FieldAt`, `Identity`, `IdentityFor`, `GenerateCases`, `DecodeManifest`, `FindGeneratedCase` (a Case's manifest entry by its fingerprint), `ExpectedRun.Check`, `SyncCases`; lowering standings, inventories and manifest data. | Reader, Testpilot facade/schema and private producer. |
| Producer, `tools/umpire/lower/internal/producer` | Assemble the executable Program and Contract. | Private lowering implementation copied from live `caseproducer` code. | Reader's table/claim types, Testpilot facade/schema; never the private checker directly. |
| Conformance, `tools/umpire/conformance` | Assess whether Run evidence is explained by a Model. | `Prepare`, returned `Factory.Binding`/`New`, existing limits, `DefaultLimits` (the ceilings the live tests and the commands share) and located errors; `Admits` becomes private because its callers are in the package. | Reader and Testpilot facade/schema. |
| Export, `tools/umpire/export` | Compare another backend's reading with the reader. | Current `Open`/`OpenWithin`, export/check/agreement methods and tool runners used by its opt-in tests. | Reader is its only model-domain dependency; generated IR and existing process/protobuf libraries allowed. |
| Lint, `tools/umpire/lint` | Report what an admitted Model leaves unreached, unasked or unrealized, and its specification holes. | `Read`, `Of`, `Model.Lint`, `ReadAccepted`, `Accepted.Judge`, `Forward`, `WriteFindings`, `WriteCoverage`, `WriteTables` (with each machine's laws, read from the law sidecar); finding, tally, verdict, acceptance and law-table data. What it reads of lowering its command hands it as `Lowering`. | Reader only. |
| Exploration, `tools/umpire/explore` | Select model-declared executable candidates. | `New`, `Plan.Reduce`, `Plan.Proposal`, `ReadProposal`, `Serve`, `RenderTrace`, candidate/plan data. | Reader, lowering, Testpilot facade/schema, campaign, replay and recordedrun. |
| Testpilot, `common/testing/testpilot` | Execute an admitted Case. | Existing `Prepare` → `PreparedCase.Run`, assessment and Driver contracts unchanged; `ConcludeVerdict`, the one Verdict aggregation recorded-Run readers check against. | Testpilot schema and existing internal runtime; nothing under `tools/umpire`, `model` or archives, including tests. |
| Recorded Runs, `common/testing/testpilot/recordedrun` | Preserve a closed Run's exact identities and evidence. | Existing `Encode`, `Decode`, `CaseIdentity`, `Digest`, support/agreement checks and data; exclusive writer becomes `Write`; canonical `Signature`, `RunSignature`, `ReportSignature`, `Line`/`Equal` and their data/prefix. | Testpilot facade/schema and Case-file encoding. |
| Case-file encoding, `common/testing/testpilot/casefile` | Admit the exact stored and canonical artifact byte forms. | `Compact`, `Persisted`, `Canonical`, `ErrNoncanonical`; keep `Indent` only if externally used. | JSON standard library only. |
| Evaluation, `common/testing/testpilot/evaluation` | Assess one closed Run under an Evaluation Profile. | Existing `Admit`, `Assess` (one fixed precedence over the Verdict and, when the caller supplies one, the Model assessment; a Profile is policy only, no reason table), `LoadProfile`/`LoadProfileIn`, `ParseProfile`, receipt encode/decode/identity and caps APIs. | Recorded Runs, Case-file encoding, Testpilot facade/schema; no replay, campaign or Model dependency (a Model assessment arrives as a `testpilot.Assessment`); receipt tests use publication directly. |
| Publication, `common/testing/testpilot/publish` | Publish review artifacts without replacing existing bytes. | `Publish`, `Check`, `Resolve`, `Within`, publication/conflict data; gains `Proposal`, `WriteProposals` and the existing exclusive proposal writer. | Filesystem/context only. |
| Campaign, `common/testing/testpilot/campaign` | Coordinate bounded Case execution over the producer-neutral bridge protocol. | Existing bridge/session, protocol frames, `Drive`/`DriveRecording`, Binder and cap APIs. | Testpilot facade/schema, Temporal binding adapter and Case-file helpers if used; no Model dependency. |
| Replay, `common/testing/testpilot/replay` | Decide whether a recorded violation reproduces through the bridge protocol. | Existing `Admit`, `Execute`, bridge/reducer, violation-key/classification, report and proposal APIs. | Testpilot, campaign, recordedrun, Case-file encoding, publication and Temporal binding for existing bounded teardown; no Umpire CLI dependency. |
| Temporal binding, `common/testing/testpilot/temporal/binding` | Bind a Case to deployment-owned Temporal resources. | Existing `Deployment`, `Open`, `Prepare`/`PrepareWith`, `Campaign.Bind`, bound Run (`Run`, and `RunAssessed` with an assessment beside it)/release and flag adapters. | Testpilot and Temporal Driver/provisioning libraries; no campaign import or Model dependency. |
| CLI edges, `tools/umpire/internal/cli` | Apply common command-edge policy. | `Interruptible`, `WriteLine`, `Flatten`, `OutsideModel`. | Publication; no runtime importing this package. |
| Commands, `tools/umpire/cmd/*` | Run the chosen pipeline operation. | The seven retained executables listed below and their existing argument contracts, and `umpire-lint`; `umpire-run` and `umpire-assess` take `--model` to run the Model assessment beside the Contract Verdict. | Relevant live modules only; no archived executable. `umpire-run` and `umpire-assess` import conformance, lowering (the manifest) and the reader for `--model`, and `umpire-assess` the Temporal Driver (`common/testing/testpilot/temporal`) to derive the offline Profile; `umpire-run` still links no test cluster and no new server service. |
| Functional wiring, `tests/testcore/testpilot` | Bind Cases to functional test clusters. | Existing fixture and cluster helper APIs with reviewed fixture identities. | Live Testpilot and Umpire modules; nothing imports this package from runtime/tooling. |
| Canary, `tools/canary` | Run the policy's pinned Case against a deployment on manual dispatch. | Existing canary entry point and policy contract. | Testpilot runtime/helpers and Temporal binding; no consumer imports from tooling/runtime. |

The reader combines admission, interpretation and checking because those are one caller operation:
reading a Model's meaning. Exploration combines reader, lowerer and runtime protocol data because
its result is a model-selected executable candidate. These are deliberate exceptions to splitting
an interface merely because its explanation can contain “and.” Campaign and replay remain separate:
one coordinates candidate execution, the other owns reproduction and reduction decisions.

The Model author contract now permits generated Temporal API, Testpilot and well-known ScalaPB
message classes, generated unary gRPC method constants, and the ScalaPB runtime needed to compile
them in `model/temporal`. The DSL exposes typed schema, selector, operand and constant-message
constructors; it has no public string form for protobuf messages, methods, paths or enum names.
The gate checks Model source for remaining proto-name literals. The IR generator writes those typed
selections back into the unchanged text-bearing IR, which Go validates independently at lowering.

The framework `model/umpire` names no Temporal concept (fn-114.12). Its `umpire/realize` keeps what
a realization of any system needs and open traits a system's kit extends; Temporal's realization
vocabulary is in `model/temporal/realize/Realize.scala`. `TestFrameworkNamesNoTemporal`
(`tools/umpire/model/framework_test.go`) checks every file under `model/umpire`, prose and
identifiers, against a word list (temporal, workflow, activity, nexus, namespace, task queue,
worker, history, chasm, matching, frontend and the six capability kinds). A mention stays only under
an allowance that names its path and reason, and an allowance that keeps no mention fails, so the
list only shrinks. It is empty since fn-122.8 moved the capability vocabulary and its laws to
`model/temporal/capabilities`.

Temporal's driver tooling names Temporal concepts by design:

- The IR generator as a whole writes Temporal realizations into the IR. `Realizations.scala` accepts
  `umpire.realize.` and `temporal.realize.` as the realization vocabulary (`vocabularyPackages`),
  matches the kit's `WorkflowHistory.event` by fully qualified name, and writes each vocabulary
  class by its simple name to the IR oneof member of that name, which is Temporal's
  (`Workflow`, `Fault`, `NexusReply`, …). It writes a member of a vocabulary class or
  object by name without following its body, and follows the kit's top-level helpers like a
  Model's own defs. `Lift.scala`'s `lifted()` reads every jar's TASTy except the framework's, so the
  kit and the laws are lifted source. `Syntax.scala`'s `requestAssignment` hook lifts
  the kit's `field(_.name) :=`. `Capabilities.scala` expands a capability declaration through the
  given catalog; it recognizes a capability by the framework's `CapabilityOf` and `CapabilityKind`
  and names none of Temporal's (`CapabilityVocabulary.test.scala`).
- The IR schema `proto/internal/temporal/server/api/umpire/v1/ir.proto`: its realization messages
  are Temporal's (`Role` and its kinds, `RequiredSetting`, the script activations
  `WorkflowActivation`, `NexusHandlerActivation` and `ActivityActivation`, `Fault`,
  `WorkflowCommand`, `NexusReply`, `NexusCompletion`, `AttemptFailure`, and `Evidence.history`); its
  machine, claim and table messages are not.
- The reader's realization admission (`tools/umpire/model/validate_realization.go`), lowering
  (`tools/umpire/lower`) and Testpilot (`common/testing/testpilot`, with the Temporal Driver in
  `common/testing/testpilot/temporal`) are the Temporal driver.

The [parity claim inventory](umpire-migration-claims.json) maps retired oracle comparisons to
frozen evidence and surviving checks, and preserves their original commentary with source attribution.

## Public types and file boundaries

The reader owns the public names for every checker type its API exposes. Copy the checker into the
reader's `internal/checker`, then declare public aliases for the required immutable/data contracts
and narrowly expose the existing operations needed by the producer and export. This preserves Go
assignability and the existing algorithms without allowing sibling packages to import the checker.
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
data (`ActionDecl`, `Party`, `Entity`, `ClassExample`, `TableClass`, `ProgressAnswer`). Recorded exceptions to the
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
what is now `model/irgen`: `Context`, `Types`, `Constants` (values folded at lift time), `Expressions`,
`Declarations`, `Realizations`, `Compositions`, `Claims`, `Lifting` (the class the concern traits mix
into) and the entry point `Lift.scala`.

Task 9 renamed the IR namespace to `umpire/v1` (Go alias `umpirespb`, JVM package
`io.temporal.server.api.umpire.v1`); `modelir` is renamed, not kept. The migration manifest and
audit JSON keep the old names as the captured baseline. `tools/umpire/model/schema_test.go` with
`testdata/schema/before-rename` freezes the pre-rename descriptor and wire bytes and names the old
package on purpose, so retired-name checks exempt them; its 14 `.gz` files are separate from the
1,411 migration goldens. `make lint-api` passes; `proto/api-linter.yaml` excludes the IR proto's path from the AIP
rules it predates.

Task 12 made the dependency rules executable in `tools/umpire/model/ownership_test.go` and the
Testpilot boundary tests: production and test imports are checked separately, the lowerer reaches
`explore`, `conformance` and `recordedrun` only from `package lower_test`, the golden helper imports
only IR, protobuf and the standard library, and no live file imports an archive. The lowerer's and
conformance's own tests may import the Temporal Driver package for admission (catalog and derived
Profile) only. `TestEveryToolingPackageHasALiveCaller` requires a live importer or a Make/CI runner
for every package under `tools/umpire`. `TestModelNamesNoRetiredFrontEnd` is the vocabulary check
outside `model/`; the gate runs it as its first step, also under `--skip-go-checks`. The export
runner is opt-in Go tests in `tools/umpire/export/tools_test.go` (`UMPIRE_BACKENDS=require`, pinned
tool versions there); `run.sh`, `UMPIRE_BACKEND_FLAGS` and `--install` are gone, and
`make umpire-check-backends` is deferred by the owner because it needs P and .NET installed.
`common/testing/testpilot/campaign/integration_test.go` is removed (its subject is archived), CI no
longer builds the descriptor set, and `make lint-code` shares `lint-code-fast`'s Go-only patch.

## Immutable migration goldens

The test-only helper `model/scalav2/goir/internal/golden` moves to
`tools/umpire/internal/golden`. It admits the fixed twelve-input inventory and the eighteen
path/twelve source-label substitutions above plus the six fixture renames, with only
standard-library, protobuf and IR imports.
No production package imports it. Reader tests and their data move to `tools/umpire/model`;
lowerer tests and their data move to `tools/umpire/lower`. Task 3 removes the existing legacy
oracle edges that currently bring Testpilot into the reader's test graph. The new reader golden
support introduces no such edge or production test-construction API.

Only the lowerer's external test package may import exploration, conformance, Testpilot runtime
and recordedrun to verify the complete artifact identity chain. This integration-test exception
does not permit those edges in lowerer production code. A narrow bridge defined only in a lowerer
`*_test.go` file may expose the existing local Query/realization conversion to that external test
package. The original Nexus oracle transfer reuses frozen IR with the five existing comparative
history `Exhaustive` adjustments and the captured original Source/Identity. Admission also requires
removing the corresponding `asyncNexus` / `controller` / `history` closing-read declarations.
Both ordered vectors must equal `temporal.nexus.caller.evidence.` followed by `started`, `completed`,
`failed`, `canceled`, `timedOut`; no other declaration is removed. Full oracle equality must prove
that every other value is unchanged. This bridge is absent from production builds.
Task 12 checks production and test imports separately. When removing parity tests, retain `tableSide`/`sideOf` as dedicated reader test support;
replace the temporary refined-row oracle comparison while retaining its frozen golden values.

There are 48 reader snapshots (inputs, semantics, declarations and refined Properties) and 1,363
artifact snapshots (original, mapped and original producer-oracle evidence). Each complete Case
has independently inspectable Program and Contract files. Read-only verification strictly matches
the current IR inputs, then regenerates both complete original and mapped artifact variants plus
the producer-oracle inventory from frozen inputs. It compares every entry, including the inactive
variant, and rejects missing or additional entries. Capture is an explicit test entry point that
rejects any existing destination. Run captures sequentially,
using separate new absolute directories whose parents exist:

```sh
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/model -run '^TestCaptureMigrationGoldens$' -count=1 -args -capture-goldens=/absolute/new-reader-capture
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/lower -run '^TestCaptureMigrationGoldens$' -count=1 -args -capture-goldens=/absolute/new-artifact-capture
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/model/... ./tools/umpire/lower/...
```

The commands name the packages at their current locations. The captured original evidence stays immutable when the
legacy oracles retire; subsequent capture generators must preserve those original inputs and
relationships through admitted IR when each old constructor retires. Task 3 replaces the Nexus
handwritten Model oracle; generic job-only checker builders remain until task 5 copies the producer
and performs their admitted-IR transfer. Capture output is never automatically installed over a baseline.
Two independent captures must match byte-for-byte before any intentional baseline addition.
Read one Contract without rewriting or expanding the whole baseline:

```sh
python3 - <<'INSPECT'
import gzip, json
from pathlib import Path
root = Path('tools/umpire/lower/testdata/migration')
path = next(root.glob('original/ir/nexus-caller.json/queries/*/contract.json.gz'))
print(path)
print(json.dumps(json.loads(gzip.decompress(path.read_bytes())), indent=2))
INSPECT
```

After relocation the inspection root is `tools/umpire/lower/testdata/migration`. Reject missing or
additional baseline entries. Task 12 removed the branch's blanket `testdata/` ignore rule and its
exceptions, so fixture trees need no ignore exception.

## Scala layout and commands

`model/project.scala` is the shared directive file copied from `scala/project.scala`. Compile the
DSL with exactly that file and `model/umpire`; compile/test/package Models with that file plus
`model/umpire` and `model/temporal`. Authoring tests are co-located with their Models; compiler and IR generator
refusal fixtures live under `irgen/testdata`. The IR generator and gate have independent `project.scala`
files. Their generated jars, fixtures
and stamps live under ignored `model/build`; the gate names source roots explicitly, so a broad
recursive Scala compile never merges the authoring, IR generator and gate projects. Lint/format commands
visit these explicit roots. fn-113 removed the native Scala evaluator and transferred its independent
checks to Go tests over the IR; the audit is `umpire-scala-evaluator-audit.md`. The Nexus domains and
step functions live in `model/temporal/features/nexuscaller/NexusCaller.scala`, in the `effects` of each machine's object, with no kernel or prelude package.
Owner-approved early cleanup in task7, immediately after relocation, removes only the unused
`Canonical.scala`, `Lower.scala` and `Alterer` plumbing from fn113 Part A / R1 after checking callers;
this does not retire the evaluator. Task 7 delivered it: the two files, `Alterer` and the helpers left without
a caller are gone, and `model/umpire` outside tests was 2,415 lines at that point; after fn-113 Part D
it is 1,191 lines.

Checked IR, generated Cases, specimens, specs, README and SEMANTICS move directly under `model/`.
Reserve `model/examples` for fn-119 authoring examples and `tools/umpire/explore` for fn-120 explorer
work; no example or UI is implemented here. Root lists kept their meaning until fn-114.1 moved them
into Scala: each IR file is an `irFile` val beside its Models, and the gate's `Roots.scala` is gone.
Since fn-114 closed, the Models own every root of `model/ir`: each Model folder's feature file
declares its IR files and their roots in its `object Files` (for example `features/nexuscaller/NexusCaller.scala`), and the
gate and the IR generator name none of them, so adding a root or an IR file edits only the Models.

| Command after migration | Existing source / behavior |
| --- | --- |
| `make umpire-check-model` | `umpire-check-scala`; runs `mise exec -- scala-cli run model/check`. |
| `make umpire-gen-model` | `umpire-gen-scala`; same program with `-- --update`. |
| `make lint-model`, `fmt-model`, `fix-model` | Model formatting and language checks, explicit new roots; IR jar prerequisite uses the gate's schema-generation mode. |
| `make umpire-check-cases`, `umpire-gen-cases` | Run `umpire-gen-cases` without/with `--update`; default managed tree is `model/cases`. |
| `make umpire-check-lint` | Run `umpire-lint` over every IR file of `model/ir`, as the gate does; findings are fixed or accepted with a reason in `model/ir/<file>.lint.json`. |
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

Since task 10 the model Make entrypoints run the Scala gate, now at `model/check`, preceded by the gate's
own test suite. The IR generator's `project.scala` compiles `check/Tools.scala` so its tests share
the gate's process seam; scala-cli has no test-scoped file directive, so this one test-time
dependency of the IR generator on the gate is a recorded exception. The IR generator suite rewrites its expected
files only when the gate passes `UMPIRE_LIFTER_UPDATE`.

Since task 14 the IR generator's fixtures are plain `.scala` files in `model/irgen/testdata`, which
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
`*.scala`), which the golden helper applies as a separate exact `source_path_renames` list to the
current IR only; the frozen mapped goldens keep the captured `.scala.fixture` spelling in their
derived digests and Case IDs. Preserve every line and column; historical-attribution removal leaves
blank lines where necessary. New, unexpected source strings fail migration verification rather than
being accepted by a generic prefix normalizer. The six checked Models and six expected fixture IR
files form the initial IR inventory; all their Queries and unsupported standings enter task 2's
baseline. Confinement remains the existing source-position test, not a new runtime loader rule.

fn-113 extends this historical migration contract with the closed projection in
`tools/umpire/internal/golden/config.json`: positions by file, alpha-normalized parameters, exact
listed function and type renames, and the Nexus source-path rename. Only a mapped-original type
catalog with an actual listed type substitution is re-sorted by its resulting names; current type
order, enum cases and record fields remain strict. Exploration Case IDs may vary because they hash
the whole candidate IR. Tables, Definition IDs, refinement rows, fingerprints, Query answers,
Query Case bytes and every other exploration Case byte retain the independent frozen baseline.
fn-114.9's folder moves are two more append-only lists there, `source_path_moves` and
`source_package_moves`; a test fails if an old path reappears.

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

The canary uses the existing `nexus-caller.json` `nexusProtocol/syncCompletion` Query. Its pinned
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

All subsequent tasks run the task-2 golden and relevant consumer checks. Closing checks compare gate
outputs against the dirty authorized baseline, include existing production/test import directions,
verify future generated fixtures are not ignored, and enforce R25 from a file outside `model/`.
An independent reviewer evaluates this map before implementation moves; the conductor owns the
review receipt and Flow lifecycle.
