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

Their shared substrate is a checked Target. A family maintainer records explicit states, actions,
outcomes, transitions, capabilities, laws, and optional finite planning in a `TargetDefinition`,
adds each provider or connector through the sealed `TargetComposition` builder, and calls
`AuthoredTarget.make` once. `checkTarget` returns a checked value or a source-located diagnostic.
Ordinary Property, Behavior, and Query authors consume that value through their `ofTarget`
adapters without assembling provider or connector collections, completeness records, finite
ordering, or planner kernels. Space authors start from the resulting checked Query rather than
redeclaring any target, Property, Behavior, or planner semantics.

Learn these forms in increasing order of domain and composition complexity:

1. [`Umpire.Examples.Switch`](Umpire/Examples/Switch.lean) is the smallest domain-neutral reference
   for the authored Target → checked Target → Query → Planning → Artifact path.
2. [`Nexus.Lifecycle`](Temporal/Feature/Nexus/Lifecycle.lean) is the ordinary Temporal starting
   point. It owns the scheduled, started, canceled, and succeeded states; the start, cancel, and
   succeed events; and the three corresponding valid transitions in one small checked target.
3. [`Nexus.Operations`](Temporal/Feature/Nexus/Operations.lean) adds one-action walkthroughs for
   starting, canceling, and successfully completing an operation. Each walkthrough exposes its
   authored and checked Property, exact-action Behavior, checked Query, and deterministic planner
   result over the shared lifecycle target.
4. [`Nexus.Observation`](Temporal/Feature/Nexus/Observation.lean) is the offline evidence boundary
   for the ordinary lifecycle. It owns the sole synthetic BasicLifecycle profile, its checked
   mapping, and the composition from a complete typed evidence bundle through Observation Evaluation,
   independent Property evaluation, and strict Query aggregation.
5. [`Temporal.System.Nexus.Core`](Temporal/System/Nexus/Core.lean) independently describes the
   minimum pure mechanism states and transitions for dispatch, cancellation recording, and
   completion recording.
6. [`Temporal.System.Nexus.ImplementationLink`](Temporal/System/Nexus/ImplementationLink.lean)
   is the sole production leaf that imports both independently checked Nexus Targets and proves the
   bounded forward correspondence from System mechanism meaning to the unchanged Feature lifecycle.
7. [`Temporal.ImplementationLinkTests.Nexus`](Temporal/ImplementationLinkTests/Nexus.lean) composes
   accepted synthetic System traces through that checked link and keeps Observation, Implementation
   Link, and Property mutations at their responsible boundaries.
8. [`Nexus.Experimental.VariationSpace`](Temporal/Feature/Nexus/Experimental/VariationSpace.lean)
   is the opt-in two-by-two proof for finite Space authoring and atomic batch compilation over the
   ordinary Lifecycle and Operations model.
9. [`Nexus.Experimental.AutoClose`](Temporal/Feature/Nexus/Experimental/AutoClose.lean) and
   [`Nexus.Experimental.CallerClosure`](Temporal/Feature/Nexus/Experimental/CallerClosure.lean)
   are explicit opt-in material for the detailed AutoClose proofs and inspectable Workflow–Nexus
   caller-closure regression. They are not part of the ordinary Feature learning surface.

`Temporal/Feature/` owns product-visible behavior, `Temporal/System/` owns configuration and other
mechanisms, and `Temporal/Tool/Inspect.lean` owns the inspector registry. The ordinary
`Temporal.Feature` facade exports `Nexus.Lifecycle`, `Nexus.Operations`, and `Nexus.Observation` but
no Experimental module. Those core walkthroughs compile directly and deliberately are not
registered with the inspector; the inspector explicitly opts into the experimental caller-closure
regression. VariationSpace is likewise explicit opt-in proof material and is not exported from the
ordinary Feature facade. The resulting `DrivePlan` and `ExperimentSpec` values are pure model
artifacts: they describe selected requests, model-owned outcomes, and semantic observations. They
do not start a Temporal server or execute Nexus operations.

## Authored variation Spaces

Import `Umpire.Space` for the reusable package. This copyable check follows the checked-in Temporal
proof without moving it onto the ordinary Lifecycle/Operations learning surface:

```lean
import Umpire.Space
import Temporal.Feature.Nexus.Experimental.VariationSpace

open Umpire
open Temporal.Feature.Nexus.Experimental.VariationSpace

#check declaration     -- ExperimentSpaceDeclaration
#check checkedResult   -- Except SpaceError (CheckedExperimentSpace LawStatement)
#check metadataResult  -- Except SpaceMetadataError CheckedSpaceMetadata
#check batchResult     -- Except SpaceCompilationError (List ExperimentSpec)
```

`declaration` adds two independent two-choice fault axes to the checked two-action Lifecycle Query.
`checkExperimentSpace context declaration` produces the complete `checked` Space or one typed
error. `projectCheckedSpaceMetadata checked` produces `metadataResult`, the canonical in-memory
input fn-5 will later aggregate. `compileBatch checked checkedKernel` produces `batchResult`: exactly
four canonically ordered `ExperimentSpec`s or no batch on the first canonical point error. The
proof-carrying `checkedKernel` is derived from the existing Lifecycle kernel; Space does not create
a second planner or target.

The base Properties remain pure. The start-delay and completion-handler-failure declarations ask a
future runtime to attempt faults at named required occurrences; they do not author an outcome or
prove realization. The unchanged target kernel supplies the started and succeeded outcomes in all
four specs. Coverage goals state what later exploration should seek, but this batch compiler neither
scores coverage nor selects a campaign. `lowerSpacePoint` and those checked goals are later C8
inputs; execution, persisted decoding, evidence, and conformance remain separate work.

## Offline Observation

Import `Umpire.Observation` for the reusable API or `Temporal.Feature.Nexus.Observation` for its one
current Temporal-owned synthetic profile. The public offline sequence is:

1. Declare an `EvidenceProfileDeclaration` and `ObservationMappingDeclaration` against a checked
   Target, then call `checkObservation` to obtain one canonical `CheckedObservationPlan`.
2. Supply one complete synthetic `EvidenceBundle` and call `evaluateEvidence`. Only the `accepted`
   result carries an `EvidenceBackedTrace`; `unknown`, `conflict`, and `unsupported` carry typed
   diagnostics and no partial trace.
3. Call `evaluateObservationProperty` independently for each required checked Property. Its status is
   `satisfied`, `violated`, `unknown`, `conflict`, or `unsupported` and retains the applied evidence
   Evidence Limit plus coordinate-based clause Evidence Links when available.
4. Call `summarizeQueryVerdicts`. It succeeds only when every required Property has exactly one
   resolved result for the same Query, Model Trace, and Evidence Limit; otherwise its status is
   `incomplete`.

Field dispositions make retention explicit: `retain` keeps an approved normalized value, `redact`
keeps only a contribution marker, `hash` keeps only a token under the named/versioned synthetic
digest policy, and `reject` refuses present input. Raw evidence is not a field of `EvidenceBackedTrace`,
`SemanticPropertyVerdict`, or `StrictQuerySummary`.

The Nexus profile maps the ordinary BasicLifecycle state, start/cancel/succeed action,
transition-outcome, and lifecycle-observation vocabulary. Its state, action, outcome, and
observation fields are retained, its raw-detail field is rejected, and its Limit is two Evidence
records. The synthetic example supplies a scheduled record followed causally by a started record
and a closure at sequence two. `evaluateSyntheticEvidence` qualifies that bundle, evaluates the
existing asynchronous-start Property, and produces one satisfied strict summary. The nearby tests
also preserve the exact offline `unknown`, `conflict`, and `unsupported` outcomes for incomplete,
ambiguous, contradictory, mismatched, rejected, or otherwise unusable synthetic bundles.

A future adapter has one typed handoff: construct a complete `EvidenceBundle` containing the
profile identity/version, records, closure facts, and—when applicable—compatible alternatives and
their missing discriminator. Each record preserves identity, profile identity/version, kind,
sequence, causal parents, typed fields with optional digest metadata, optional binding facts, and
an optional fault target. Umpire, not the adapter, compiles mappings, enforces the Limit and
dispositions, qualifies evidence, evaluates Properties, and aggregates verdicts. No adapter is
implemented in this release, and these modules do not execute Temporal, collect or persist live
Evidence, perform Run Evaluation, promote a result, or support another profile.

## Checked Implementation Links

Import `Umpire.ImplementationLink` for the reusable API. Authors declare finite correspondences
between independently checked source and destination Targets, provide a forward-simulation witness
indexed by those exact inputs, and call `checkImplementationLink` once. The checked value canonically
binds the two Target identities and Behavior Fingerprints, mapping version, support/Known Gap
partition, obligations, and positive application Limit. Proof terms remain nonserialized.

Application starts only after Observation Evaluation has accepted one complete System
`EvidenceBackedTrace`. `applyImplementationLink` replays that trace through the checked System
kernel before translating it positionally. Success contains one complete authoritative Feature
Model Trace and coordinate-complete Implementation Link Evidence Links; failure contains no partial
Feature trace. The three semantic results remain separate:

```text
EvidenceBundle ─ Observation Evaluation ─▶ accepted System trace | Observation diagnostic
accepted System trace ─ checked Implementation Link ─▶ Feature trace | Implementation Link diagnostic
Feature trace ─ checked Feature Property ─▶ satisfied | violated
```

The first Nexus correspondence covers ordinary start, cancellation, and successful completion.
AutoClose and CallerClosure remain Experimental and outside this seam. A future Run Evaluation may
orchestrate these checked stages, but it must retain the responsible layer, canonical identity, and
Evidence Links for every non-success rather than turning an Observation or Implementation Link
failure into a Property violation.

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
collect evidence. Runtime driving, construction of a future adapter's `EvidenceBundle`, and
promotion remain separate work; offline Observation Evaluation is the current `Umpire.Observation` API.

Generated API declarations remain generated structures only. Behavioral meaning, including whether
a selected action is applicable and which transition outcomes are possible, remains owned by the
authored Lean model.
