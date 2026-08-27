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

Model scenarios use three separate but composable forms:

- A `Property` describes portable meaning over a capability-limited semantic trace: what must hold,
  independently of how a trace is found.
- A `Behavior` constrains setup and controllable actions, including ordering and exactness: what the
  planner may drive. Outcomes and observations remain owned by the target model.
- A `Query` combines checked properties and behavior with explicit bounds and a deterministic
  planning policy: what bounded search should find or verify.

Their shared substrate is a checked Target. A family maintainer records explicit states, actions,
outcomes, transitions, capabilities, laws, and optional finite planning in a `TargetDefinition`,
adds each provider or connector through the sealed `TargetComposition` builder, and calls
`AuthoredTarget.make` once. `checkTarget` returns a checked value or a source-located diagnostic.
Ordinary Property, Behavior, and Query authors consume that value through their `ofTarget`
adapters without assembling provider or connector collections, completeness records, finite
ordering, or planner kernels.

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
   mapping, and the composition from a complete typed evidence bundle through qualification,
   independent Property evaluation, and strict Query aggregation.
5. [`Nexus.Experimental.AutoClose`](Temporal/Feature/Nexus/Experimental/AutoClose.lean) and
   [`Nexus.Experimental.CallerClosure`](Temporal/Feature/Nexus/Experimental/CallerClosure.lean)
   are explicit opt-in material for the detailed AutoClose proofs and inspectable Workflow–Nexus
   caller-closure regression. They are not part of the ordinary Feature learning surface.

`Temporal/Feature/` owns product-visible behavior, `Temporal/System/` owns configuration and other
mechanisms, and `Temporal/Tool/Inspect.lean` owns the inspector registry. The ordinary
`Temporal.Feature` facade exports `Nexus.Lifecycle`, `Nexus.Operations`, and `Nexus.Observation` but
no Experimental module. Those core walkthroughs compile directly and deliberately are not
registered with the inspector; the inspector explicitly opts into the experimental caller-closure
regression. The resulting `DrivePlan` and `ExperimentSpec` values are pure model artifacts: they
describe selected requests, model-owned outcomes, and semantic observations. They do not start a
Temporal server or execute Nexus operations.

## Offline Observation

Import `Umpire.Observation` for the reusable API or `Temporal.Feature.Nexus.Observation` for its one
current Temporal-owned synthetic profile. The public offline sequence is:

1. Declare an `EvidenceProfileDeclaration` and `ObservationMappingDeclaration` against a checked
   Target, then call `checkObservation` to obtain one canonical `CheckedObservationPlan`.
2. Supply one complete synthetic `EvidenceBundle` and call `qualifyEvidence`. Only the `qualified`
   result carries a `QualifiedTrace`; `unknown`, `conflict`, and `unsupported` carry typed
   diagnostics and no partial trace.
3. Call `evaluateQualifiedProperty` independently for each required checked Property. Its status is
   `satisfied`, `violated`, `unknown`, `conflict`, or `unsupported` and retains the applied evidence
   bound plus coordinate-based clause derivations when available.
4. Call `summarizeQueryVerdicts`. It succeeds only when every required Property has exactly one
   resolved result for the same Query, trace, and evidence bound; otherwise its status is
   `incomplete`.

Field dispositions make retention explicit: `retain` keeps an approved normalized value, `redact`
keeps only a contribution marker, `hash` keeps only a token under the named/versioned synthetic
digest policy, and `reject` refuses present input. Raw evidence is not a field of `QualifiedTrace`,
`SemanticPropertyVerdict`, or `StrictQuerySummary`.

The Nexus profile maps the ordinary BasicLifecycle state, start/cancel/succeed action,
transition-outcome, and lifecycle-observation vocabulary. Its state, action, outcome, and
observation fields are retained, its raw-detail field is rejected, and its bound is two evidence
records. The synthetic example supplies a scheduled record followed causally by a started record
and a closure at sequence two. `evaluateSyntheticEvidence` qualifies that bundle, evaluates the
existing asynchronous-start Property, and produces one satisfied strict summary. The nearby tests
also preserve the exact offline `unknown`, `conflict`, and `unsupported` outcomes for incomplete,
ambiguous, contradictory, mismatched, rejected, or otherwise unusable synthetic bundles.

A future adapter has one typed handoff: construct a complete `EvidenceBundle` containing the
profile identity/version, records, closure facts, and—when applicable—compatible alternatives and
their missing discriminator. Each record preserves identity, profile identity/version, kind,
sequence, causal parents, typed fields with optional digest metadata, optional binding facts, and
an optional fault target. Umpire, not the adapter, compiles mappings, enforces the bound and
dispositions, qualifies evidence, evaluates Properties, and aggregates verdicts. No adapter is
implemented in this release, and these modules do not execute Temporal, collect or persist live
evidence, prove runtime conformance, promote a result, or support another profile.

Build each stage through the final module and target names:

```sh
cd model
mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests
mise exec -- lake build Temporal.Feature.Nexus.OperationsTests
mise exec -- lake build Umpire.Observation.Tests.Compilation
mise exec -- lake build Umpire.Observation.Tests
mise exec -- lake build Temporal.Feature.Nexus.ObservationTests
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
and verification modules.

The focused check builds `Temporal`, `UmpireTests`, `TemporalModelTests`,
`TemporalExperimentalTests`, and `temporal-model-inspect`. The ordinary and experimental test
aggregates stay separate while both remain covered by full regression. The check rejects obsolete
interfaces, reusable Umpire domain leaks, and invalid `Temporal.System.Configuration` imports of
its `Temporal.System.Callback.Configuration` or `Temporal.System.Matching.Configuration`
consumers; compares repeated inspection with both checked-in target-state fixtures byte-for-byte;
and verifies that unknown or invalid inspector requests emit one structured diagnostic with no
artifact JSON on standard output. It also clean-regenerates the checked-in caller-closure Go and
Markdown projections and runs their focused Go tests. It does not require or contact a running
Temporal server.

Generate or check the stable regression projections from the repository root:

```sh
make umpire-gen-regression-projections
make umpire-check-regression-projections
```

Generation transactionally replaces the complete managed pair:

- `tools/umpire/regression/catalog_generated_test.go`, an ordinary Go test that calls only the
  fixture-backed `RequireProjection` helper;
- `model/Temporal/Tool/Generated/Regressions.md`, the readable projection index.

The current catalog is deliberately closed to the stable caller-closure regression. Lean and its
canonical `ExperimentSpec` fixture remain the semantic source of truth. Inspector provenance is
canonical relative to `model/`; generated navigation adds the repository-facing `model/` prefix.
The semantic fingerprint is lowercase SHA-256 over the exact UTF-8 bytes of the decoded
`ExperimentSpec.semanticIdentity`, rendered as `sha256:<64 lowercase hexadecimal characters>`.
Run the generation target after an intentional semantic change, review both outputs together, and
use either projection check or the encompassing `make umpire-check-regression` gate to prove the
checked-in pair is clean. The generated Go test can verify its fixture without Lean or a running
Temporal service, but projection success is not runtime execution, execution evidence, or a
conformance result.

Inspect either checked scenario directly with:

```sh
make umpire-inspect SCENARIO=workflow-nexus.query.exact-action-caller-closure
make umpire-inspect SCENARIO=switch.query.exact-action
```

On success the inspector writes one canonical JSON `ExperimentSpec` to standard output. The
compiler and inspector do not write an artifact file, start a live server, execute a workflow, or
collect evidence. Runtime driving, construction of a future adapter's `EvidenceBundle`, and
promotion remain separate work; offline qualification is the current `Umpire.Observation` API.

Generated API declarations remain generated structures only. Behavioral meaning, including whether
a selected action is applicable and which transition outcomes are possible, remains owned by the
authored Lean model.
