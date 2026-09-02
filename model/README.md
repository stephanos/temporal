# Temporal Lean model

Lean owns behavioral meaning in this directory. `umpire-gen-lean-api` consumes serialized protobuf
descriptor sets and projects their protobuf and gRPC structure behind the stable `Temporal.API`
module boundary. Generated declarations do not assign product semantics to fields or RPCs.

The generator exclusively owns `Temporal/API.lean` and the complete `Temporal/API/` directory:

- `API/Proto.lean` contains the runtime-independent `Bytes`, `MessageRef`, and typed `Method`
  support structures.
- `API/Types.lean` contains structural message, enum, map, oneof, presence, and recursion
  projections. Namespaces continue to derive from protobuf packages; for example,
  `temporal.server.api.adminservice.v1.DescribeMutableStateRequest` becomes
  `Temporal.Server.Api.Adminservice.V1.DescribeMutableStateRequest`.
- `API.lean` imports both child modules and declares every typed RPC in its protobuf-derived
  service namespace. Same-package message references are short and cross-package references are
  qualified. These declarations do not provide an RPC client or server runtime.

Bytes and recursive links remain deliberately bounded abstractions. The generator does not
interpret arbitrary protobuf options as product semantics; authored model families explicitly
interpret the structural metadata they use.

The repository root `Makefile` is the only Make interface for this model. After changing public,
internal, or CHASM APIs, regenerate the descriptor-backed modules and verify them locally:

```sh
make umpire-gen-lean-api
go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api
make umpire-build-model
```

Generation is deterministic and silent on success. Each run validates all three artifacts and
their output paths before mutation, then replaces the owned API outputs while preserving adjacent
authored modules.

## Dynamic configuration catalog

`umpire-gen-lean-dynamic-config-catalog` constructs a generation-time snapshot of Temporal's initialized
production registry and projects its structural metadata behind the `Temporal.DynamicConfig`
module boundary. The generator exclusively owns `Temporal/DynamicConfig.lean` and the complete
`Temporal/DynamicConfig/` directory:

- `DynamicConfig/Types.lean` defines the structural schemas for keys, value codecs, precedence,
  defaults, constraints, and generation fixtures.
- `DynamicConfig/Settings.lean` contains the complete ordered setting catalog and its canonical
  generation identity.
- `DynamicConfig.lean` is the public facade that imports both generated child modules.

These declarations record generation-time registry structure. They do not parse deployment YAML,
read live server configuration, or execute Go converters in Lean. Handwritten Lean outside the
owned boundary is responsible for classifications, typed interpretations, consumer-specific
meaning, and any explicit replacement for an opaque generated default. Shared interpretation and
validation live under `Temporal/System/Configuration/`; Callback- and Matching-specific semantics
live under `Temporal/System/Callback/` and `Temporal/System/Matching/`, with focused tests assembled
by `TemporalModelTests`.

From the repository root, regenerate and verify the catalog with:

```sh
make umpire-gen-lean-dynamic-config-catalog
go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog
make umpire-build-model
```

For an unchanged initialized registry, repeated generation produces byte-identical modules. Each
run elaborates all three candidate modules before replacing the retained generated output.

## Semantic authoring and planning

Model scenarios use four separate but composable forms:

- A `Property` describes portable meaning over a capability-limited semantic trace: what must hold,
  independently of how a trace is found.
- A `Behavior` constrains setup and controllable actions, including ordering and exactness: what the
  planner may drive. Outcomes and observations remain owned by the target model.
- A `Query` combines checked Properties and Behavior with explicit Limits and a deterministic
  planning policy: what bounded search should find or verify.
- A `Space` composes one checked Query with finite choices, request-only faults, and seek-only
  coverage goals: which canonical points may be lowered or compiled as one atomic batch.

Their shared substrate is a checked Target. For the ordinary complete finite case, a family
maintainer records ordered setups, states, actions, outcomes, observations, encoders, initial
states, transitions, and the residual coverage/executability evidence once in a `FiniteMachine`.
The adapter derives membership authority, the complete behavior domain, and the exact dependent
finite-planning input. The maintainer places those values in a `TargetDefinition`, adds each
provider or connector through the sealed `TargetComposition` builder, and calls
`AuthoredTarget.make` once. `checkTarget` returns a checked value or a source-located diagnostic.
Targets whose authoritative propositions are intentionally independent of their enumerators use
direct `TransitionKernel` construction as the expert route. Both routes converge before checking;
neither changes the checked Target consumed downstream.
Ordinary Property, Behavior, and Query authors consume that value through their `ofTarget`
adapters without assembling provider or connector collections, completeness records, finite
ordering, or planner kernels. Space authors start from the resulting checked Query rather than
redeclaring any target, Property, Behavior, or planner semantics.

Import [`Temporal.Feature.Nexus`](Temporal/Feature/Nexus.lean) as the single ordinary Nexus entry
point, then follow this simple-first reading order:

1. [`Nexus.Lifecycle.Semantics`](Temporal/Feature/Nexus/Lifecycle/Semantics.lean) owns the scheduled,
   started, canceled, and succeeded states; the start, cancel, and succeed events; and the three
   corresponding valid transitions.
2. [`Nexus.Operations.AsyncStart`](Temporal/Feature/Nexus/Operations/AsyncStart.lean) follows one
   complete Property → Behavior → Query → deterministic Planning walkthrough for starting an
   operation.
3. [`Nexus.Operations.Cancellation`](Temporal/Feature/Nexus/Operations/Cancellation.lean) follows
   the same complete walkthrough for canceling a started operation.
4. [`Nexus.Operations.SuccessfulCompletion`](Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean)
   follows the same complete walkthrough for successfully completing a started operation.
5. [`Nexus.Observation`](Temporal/Feature/Nexus/Observation.lean) is the offline evidence boundary
   for the ordinary lifecycle. It owns the sole synthetic BasicLifecycle profile, its checked
   mapping, and the composition from a complete typed evidence bundle through Observation Evaluation,
   independent Property evaluation, and strict Query aggregation.
6. [`Temporal.System.Nexus.Core`](Temporal/System/Nexus/Core.lean) independently describes the
   minimum pure mechanism states and transitions for dispatch, cancellation recording, and
   completion recording.
7. [`Temporal.System.Nexus.ImplementationLink`](Temporal/System/Nexus/ImplementationLink.lean)
   is the sole production leaf that imports both independently checked Nexus Targets and proves the
   bounded forward correspondence from System mechanism meaning to the unchanged Feature lifecycle.
8. [`Temporal.ImplementationLinkTests.Nexus`](Temporal/ImplementationLinkTests/Nexus.lean) composes
   accepted synthetic System traces through that checked link and keeps Observation, Implementation
   Link, and Property mutations at their responsible boundaries.
9. [`Nexus.Experimental.VariationSpace`](Temporal/Feature/Nexus/Experimental/VariationSpace.lean)
   is the opt-in two-by-two proof for finite Space authoring and atomic batch compilation over the
   ordinary Lifecycle and Operations model.
10. [`Nexus.Experimental.AutoClose`](Temporal/Feature/Nexus/Experimental/AutoClose.lean) and
   [`Nexus.Experimental.CallerClosure`](Temporal/Feature/Nexus/Experimental/CallerClosure.lean)
   are explicit opt-in material for the detailed AutoClose proofs and inspectable Workflow–Nexus
   caller-closure regression. They are not part of the ordinary Feature learning surface.

The stable [`Nexus.Lifecycle`](Temporal/Feature/Nexus/Lifecycle.lean) and
[`Nexus.Operations`](Temporal/Feature/Nexus/Operations.lean) facades expose those ordinary modules.
Their Target and Planning children are implementation reading for contributors who need the checked
Umpire machinery, not extra steps in the newcomer path. For a separate example of direct expert
`TransitionKernel` authoring, see [`Umpire.Examples.Switch`](Umpire/Examples/Switch.lean).

`Temporal/Feature/` owns product-visible behavior, `Temporal/System/` owns configuration and other
mechanisms, and `Temporal/Tool/Inspect.lean` owns the inspector registry. The ordinary
`Temporal.Feature` facade consumes `Temporal.Feature.Nexus`, which exports Lifecycle, Operations,
and Observation but no Experimental module. Those core walkthroughs compile directly and deliberately are not
registered with the inspector; the inspector explicitly opts into the experimental caller-closure
regression. VariationSpace is likewise explicit opt-in proof material and is not exported from the
ordinary Feature facade. The resulting `DrivePlan` and `ExperimentSpec` values are pure model
artifacts: they describe selected requests, model-owned outcomes, and semantic observations. They
do not start a Temporal server or execute Nexus operations.

`FiniteMachine` is the reusable Umpire Target authoring API; it does not prescribe how a Temporal
family is split across files. The Lifecycle and Operations facades keep the stable namespaces and
source provenance while focused children separate semantics, Target construction, planning, and
the three complete operation walkthroughs.

## Authored variation Spaces

Import `Umpire.Space` for the reusable package. This copyable check follows the checked-in Temporal
proof without moving it onto the ordinary Lifecycle/Operations learning surface:

```lean
import Umpire.Space
import Temporal.Feature.Nexus.Experimental.VariationSpace

open Umpire
open Temporal.Feature.Nexus.Experimental.VariationSpace

#check declaration     -- ExperimentSpaceDeclaration
#check queryResult     -- Except VariationSpacePreparationError (CheckedQuery LawStatement)
#check preparedResult  -- Except VariationSpacePreparationError PreparedVariationSpace
#check metadataResult  -- Except VariationSpacePreparationError CheckedSpaceMetadata
#check batchResult     -- Except VariationSpacePreparationError (List ExperimentSpec)
```

`declaration` adds two independent two-choice fault axes to the checked two-action Lifecycle Query.
`preparedResult` checks the Behavior, Query, and Space before it projects metadata and compiles the
batch. It returns the complete `PreparedVariationSpace` or one typed stage error, without assuming
that any check succeeded. `metadataResult` is the canonical in-memory input fn-5 will later
aggregate. `batchResult` contains exactly four canonically ordered `ExperimentSpec`s or no batch on
the first preparation or canonical point error. The proof-carrying compiler kernel is transported
from the existing Lifecycle kernel; Space does not create a second planner or target.

The base Properties remain pure. The start-delay and completion-handler-failure declarations ask a
future runtime to attempt faults at named required occurrences; they do not author an outcome or
prove realization. The unchanged target kernel supplies the started and succeeded outcomes in all
four specs. Coverage goals state what later exploration should seek, but this batch compiler neither
scores coverage nor selects a campaign. `lowerSpacePoint` and those checked goals are later C8
inputs; execution, persisted decoding, evidence, and conformance remain separate work.

`umpire-gen-tests` is the single public generation handoff. It accepts a named regression, test set,
or model-selected batch, then emits the same canonical v2 planning Artifacts without adding
participant, setup, ordering, termination, cleanup, or other runtime bindings. Fn-18's separate
RuntimeConfiguration boundary owns those bindings:

```bash
make umpire-gen-tests ARGS='temporal.nexus.basic-lifecycle.test-set.core --output /tmp/umpire-tests'
make umpire-gen-tests ARGS='temporal.nexus.basic-lifecycle.space.fault-matrix --output /tmp/umpire-tests'
```

The output directory contains one canonical manifest and one canonical artifact per selected point.
Fn-5 owns discovery and explanation. Operational endpoints, credentials, namespaces, and runtime
authority remain downstream bindings.

The retained boundary is exactly the embedded `umpire-drive-plan/v2` plus the persisted
`umpire-experiment/v2`, `umpire-runtime-configuration/v2`, `umpire-experiment-run/v2`,
`umpire-raw-evidence/v2`, `umpire-evidence/v2`, and `umpire-result/v2` families. Every document has
one representation: fixed-order JSON with two-space indentation, stable escaping and base-10
natural numbers, no trailing spaces, and exactly one terminal LF. Artifact Checksum input is the
UTF-8 domain, one LF, and that document's exact deterministic-pretty bytes with only its own
`artifactChecksum` field omitted; the ExperimentSpec preimage retains its already-sealed DrivePlan.
Behavior Fingerprints identify checked meaning, provenance checksums bind exact provenance, and
Artifact Checksums identify the complete persisted bytes; none substitutes for another.

Runtime, Evidence, and Result documents preserve their phase-specific Limits and Known Gaps rather
than turning exhaustion, unsupported interpretation, incomplete cleanup, or absent Evidence into a
successful Run Evaluation. Complete set admission accepts only the fixed two-member executable,
four-member execution, or six-member evaluation closure and verifies every Definition ID, Behavior
Fingerprint, checksum, binding, and reference before returning a value.

From the repository root, the read-only checks are:

```bash
make umpire-check-artifact FAMILY=umpire-experiment/v2 ARTIFACT=tools/umpire/artifact/testdata/switch-experiment-v2.json
make umpire-check-artifact-set SET=tools/umpire/artifact/testdata/valid-run-evaluation-set
```

Both checks are silent on success, write failures only to stderr, return 2 for invalid arguments and
1 for filesystem or admission failure, and never rewrite or publish their inputs. Immutable
publication remains the separate `PublishSet` API: it validates and privately stages the complete
set, installs one manifest-digest directory with a single rename, and lets readers observe absence
or one fully revalidated set. Publication removes abandoned private staging directories under its
lock; general artifact management, schema migration, generic envelopes, and runtime/platform/CI
orchestration remain outside this boundary. This pre-release correction supersedes fn-37's compact
spelling in place. Compact or alternate-whitespace input has no normalization, alias, fallback, or
migration path.

## Offline Observation

Import `Umpire.Observation` for the reusable API or `Temporal.Feature.Nexus.Observation` for its one
current Temporal-owned synthetic profile. The public offline sequence is:

1. Declare an `EvidenceProfileDeclaration` and `ObservationMappingDeclaration` against a checked
   Target, then call `checkObservation` to obtain one canonical `CheckedObservationPlan`.
2. Supply one complete synthetic `EvidenceBundle` and call `evaluateEvidence`, the single
   Observation admission handoff. Only the `accepted` result carries an opaque
   `EvidenceBackedTrace`; `unknown`, `conflict`, and `unsupported` carry typed diagnostics and no
   partial trace. The accepted type has no public constructor or record-update path and exposes only
   read-only projections; it is a checked handoff value, not another authoring or scenario language.
3. Call `evaluateObservationProperty` independently for each required checked Property with that
   accepted trace. It validates the query/Property relationship, capability access, and logical-time
   prerequisites without repeating Observation admission. Its status is `satisfied`, `violated`,
   `unknown`, `conflict`, or `unsupported` and retains the applied Evidence Limit plus
   coordinate-based clause Evidence Links when available.
4. Call `summarizeQueryVerdicts`. It succeeds only when every required Property has exactly one
   resolved result for the same Query, Model Trace, and Evidence Limit; otherwise its status is
   `incomplete`.

Field dispositions make retention explicit: `retain` keeps an approved normalized value, `redact`
keeps only a contribution marker, `hash` keeps only a token under the named/versioned synthetic
digest policy, and `reject` refuses present input. Raw evidence is not a field of `EvidenceBackedTrace`,
`SemanticPropertyVerdict`, or `StrictQuerySummary`.

All trace positions in this path use the single `Umpire.Core` coordinate API. Coordinates enumerate
initial state followed by each step's selected Action, Model Outcome, resulting state, and
observations in source order. Numeric step and observation positions are strictly one-based;
`ModelTrace.valueAt?` rejects zero and out-of-range positions, and
`ModelCoordinate.definitionKind` supplies the shared kind mapping.

The Nexus profile maps the ordinary BasicLifecycle state, start/cancel/succeed action,
transition-outcome, and lifecycle-observation vocabulary. Its state, action, outcome, and
observation fields are retained, its raw-detail field is rejected, and its Limit is two Evidence
records. The synthetic example supplies a scheduled record followed causally by a started record
and a closure at sequence two. `evaluateSyntheticEvidence` qualifies that bundle, evaluates the
existing asynchronous-start Property, and produces one satisfied strict summary. The nearby tests
also preserve the exact offline `unknown`, `conflict`, and `unsupported` outcomes for incomplete,
ambiguous, contradictory, mismatched, rejected, or otherwise unusable synthetic bundles.

`Temporal.Tool.RunEvaluation` now provides one closed adapter for the experimental caller-closure
scenario and its exact duplicate-delivery negative control. It converts either fn-19 four-source
Generated View into a complete `EvidenceBundle` while preserving raw fact identity, causality,
typed fields, digest metadata, and source closure. The fault adapter projects the complete raw
history onto the requested/completed lifecycle and selects only the checked, labeled synthetic
contribution; ordinary authority, worker, participant, and cleanup lifecycle facts remain in
RawEvidence but do not become semantic support. The checked `Temporal.System.Nexus.Observation`
plans—not Go—own the mappings, Limits, dispositions, and admitted Evidence-backed System traces.
Run Evaluation invokes Implementation Link application and Property evaluation only after that
admission succeeds and preserves each stage's status and no-partial-result guarantee. This is one
fixed local pair, not a generic profile loader: the model still does not execute Temporal, select a
checker or profile at runtime, perform replay or promotion, or qualify a non-local result.

## Checked Implementation Links

Import `Umpire.ImplementationLink` for the reusable API. Authors declare finite correspondences
between independently checked source and destination Targets, provide a forward-simulation witness
indexed by those exact inputs, and call `checkImplementationLink` once. The checked value canonically
binds the two Target identities and Behavior Fingerprints, mapping version, support/Known Gap
partition, obligations, and positive application Limit. Proof terms remain nonserialized.

Application starts only after Observation Evaluation has accepted one complete System
`EvidenceBackedTrace`. `applyImplementationLink` does not repeat Observation envelope admission. It
checks the Link's source Target, application Limit, mapping, support/Known Gap partition, and
translation while replaying the accepted trace through the checked System kernel. Success contains
one complete authoritative Feature Model Trace and coordinate-complete Implementation Link Evidence
Links; failure contains no partial Feature trace. The three semantic results remain separate:

```text
EvidenceBundle ─ Observation Evaluation ─▶ accepted System trace | Observation diagnostic
accepted System trace ─ checked Implementation Link ─▶ Feature trace | Implementation Link diagnostic
Feature trace ─ checked Feature Property ─▶ satisfied | violated
```

The first Nexus correspondence covers ordinary start, cancellation, and successful completion.
AutoClose and CallerClosure remain Experimental and outside the ordinary seam. The fixed local
caller-closure Run Evaluation now proves both a satisfied normal control and an accepted
duplicate-delivery control whose unchanged Feature Property reports only uniqueness violated. The
faulted run still has one real requested/completed cancellation lifecycle and callback; its second
semantic contribution is explicitly synthetic and test-owned, not a Temporal defect claim. Both
paths retain the responsible layer, canonical identity, and Evidence Links rather than turning an
Observation or Implementation Link failure into a Property violation. Other scenarios remain
unintegrated.

For one admitted four-member caller-closure execution set, the repository-root
`umpire-check-local-run-evaluation` target builds the fixed Go/Lean sibling pair, checks the set,
and immutably publishes the six-member extension containing Evidence and Result. See
[`tools/umpire/runevaluation/README.md`](../tools/umpire/runevaluation/README.md) for the exact
offline and paired-live commands, outputs, statuses, Limits, dispositions, and fail-closed
boundary. Operational success, accepted Observation Evaluation, applied Implementation Link, and
Property satisfaction or violation remain independently inspectable.

## Semantic inventory

[`SEMANTIC_INVENTORY.md`](SEMANTIC_INVENTORY.md) is the checked, generated navigation view of the
owner-published outcome, projection-sentinel, and Known Gap catalogs. Stage-owned types, the Result
schema, canonical Artifact bytes, and runtime behavior remain authoritative and unchanged; the
inventory is neither another semantic source nor a persistence schema.

From the repository root, regenerate or read-only check the single managed document:

```sh
make umpire-gen-semantic-inventory
make umpire-check-semantic-inventory
```

The narrow local drift check builds on the completed fn-20 dependency and the completed-prerequisite
fn-44/fn-45 baselines. It does not reopen broad generated API drift verification or add CI workflow
coverage.

Build each stage through the final module and target names:

```sh
cd model
mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests
mise exec -- lake build Temporal.Feature.Nexus.OperationsTests
mise exec -- lake build Umpire.Observation.Tests.Compilation
mise exec -- lake build Umpire.Observation.Tests
mise exec -- lake build Temporal.Feature.Nexus.ObservationTests
mise exec -- lake build Umpire.ImplementationLink.Tests
mise exec -- lake build Temporal.System.Nexus.ImplementationLinkTests
mise exec -- lake build Temporal.ImplementationLinkTests.Nexus
mise exec -- lake build Umpire.Examples.Switch
mise exec -- lake build Temporal.Feature.Nexus.Experimental.CallerClosureTests
mise exec -- lake build Temporal TemporalModelTests TemporalExperimentalTests temporal-model-inspect
```

From the Temporal repository root, run the focused regression check:

```sh
make umpire-check-regression
```

Run `make lint-model` for Lean declaration linting and the complete first-party import graph. Its
executed graph regressions, controlled failure fixture, and live metadata pass own the transitive
`Shared.*`/`Umpire.*`, `Umpire.*`/`Temporal.*`, `Temporal.Feature.*`/`Temporal.System.*`, and
opt-in `Temporal.Verify.*`/`Umpire.Verify.Veil` boundaries. They also keep every
`Umpire.Target.*` module, including its tests, below Query, Planning, Artifact, Temporal, runtime,
and verification modules. The only production System-to-Feature exception is the exact
`Temporal.System.Nexus.ImplementationLink` leaf. The only composed-test class is the exact
`Temporal.ImplementationLinkTests.Nexus` root; sibling System modules and test-root near misses
remain rejected.

The focused check builds `Temporal`, `UmpireTests`, `TemporalModelTests`,
`TemporalExperimentalTests`, and `temporal-model-inspect`. The ordinary and experimental test
aggregates stay separate while both remain covered by full regression. The check rejects obsolete
interfaces, reusable Umpire domain leaks, and invalid `Temporal.System.Configuration` imports of
its `Temporal.System.Callback.Configuration` or `Temporal.System.Matching.Configuration`
consumers; compares repeated inspection with both checked-in target-state fixtures byte-for-byte;
and verifies that unknown or invalid inspector requests emit one structured diagnostic with no
artifact JSON on standard output. It also clean-regenerates the checked-in Switch and caller-closure
Go and Markdown Generated Views and runs their focused Go tests. It does not require or contact a running
Temporal server.

Generate or check the stable regression Generated Views from the repository root:

```sh
make umpire-gen-regression-views
make umpire-check-regression-views
```

Generation transactionally replaces the complete managed four-output set:

- `tools/umpire/regression/catalog_generated_test.go`, an ordinary Go test that calls only the
  fixture-backed `RequireGeneratedView` helper;
- `model/Temporal/Tool/Generated/Regressions.md`, the readable Nexus Generated View;
- `tools/umpire/regression/switch_generated_view_test.go`, the corresponding Switch Go test; and
- `model/Umpire/Examples/Generated/Switch.md`, the readable Switch Generated View.

The current catalog is deliberately closed to the Switch and caller-closure regressions. Lean and
their canonical `ExperimentSpec` fixtures remain the model source of truth. Inspector provenance is
canonical relative to `model/`; generated navigation adds the repository-facing `model/` prefix.
Each Artifact Checksum is independently verified from the exact canonical v2 Artifact content and
rendered as `sha256:<64 lowercase hexadecimal characters>`. Run the generation target after an
intentional model change, review all four outputs together, and use either Generated View check or
the encompassing `make umpire-check-regression` gate to prove the checked-in set is clean. The
generated Go tests can verify their fixtures without Lean or a running Temporal service, but
Generated View success is not runtime execution, execution evidence, or a
Run Evaluation result.

Inspect either checked scenario directly with:

```sh
make umpire-inspect SCENARIO=workflow-nexus.query.exact-action-caller-closure
make umpire-inspect SCENARIO=switch.query.exact-action
```

On success the inspector writes one canonical JSON `ExperimentSpec` to standard output. The
compiler and inspector do not write an artifact file, start a live server, execute a workflow, or
collect evidence. The separate fixed caller-closure adapter can now construct one live
`EvidenceBundle` and run the checked local semantic path after execution. Broader runtime driving,
other profiles, replay, and promotion remain separate work; reusable offline Observation
Evaluation remains the `Umpire.Observation` API.

The same inspector exposes a closed Nexus discovery view containing exactly these Query identities:

1. `temporal.nexus.basic-lifecycle.query.async-start`
2. `temporal.nexus.basic-lifecycle.query.cancellation`
3. `temporal.nexus.basic-lifecycle.query.successful-completion`
4. `workflow-nexus.query.exact-action-caller-closure`

List the four entries or explain one exact identity from the repository root:

```sh
make umpire-list-nexus
make umpire-explain-nexus QUERY=workflow-nexus.query.exact-action-caller-closure
```

`make umpire-check-promotion` runs the sole fixed candidate,
`temporal.nexus.caller-closure.promotion.cancel-unique-regression`, and verifies its canonical
proposal, separate base and fault-bearing lineages, and embedded source elaboration without writing
the proposal or source. The proposal is inert: its source preserves the unchanged base Query's
target-owned expected cancellation count of one. It does not establish that the duplicate-delivery
failure occurred, replay it, minimize it, or install a Regression. Fn-22 alone may establish runtime
eligibility by reproducing and completely reducing the failure, validating Exact Replay, and
cross-binding that runtime Evidence to this proposal.

Generated API declarations remain generated structures only. Behavioral meaning, including whether
a selected action is applicable and which transition outcomes are possible, remains owned by the
authored Lean model.
