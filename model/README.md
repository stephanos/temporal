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

Learn these forms in increasing order of domain and composition complexity:

1. [`Umpire.Examples.Switch`](Umpire/Examples/Switch.lean) is the smallest domain-neutral example
   of the complete authored → checked → planned → artifact lifecycle.
2. [`BasicLifecycle`](Temporal/Feature/Nexus/Examples/BasicLifecycle.lean) adapts the authoritative
   Nexus lifecycle into one small Temporal-owned target with one capability and one provider.
3. [`BasicOperations`](Temporal/Feature/Nexus/Examples/BasicOperations.lean) adds two one-action
   walkthroughs over that shared target: asynchronously starting a scheduled operation, then
   reporting successful completion for a started operation. Each walkthrough exposes its authored
   and checked Property, exact-action Behavior, checked Query, and deterministic planner result.
4. [`CallerClosure`](Temporal/Feature/Nexus/CallerClosure.lean) is the advanced integration
   reference for Workflow–Nexus ownership, connector composition, cancellation, and multiple query
   modes. It is not the starting point for learning the DSLs.

`Temporal/Feature/` owns product-visible behavior, `Temporal/System/` owns configuration and other
mechanisms, and `Temporal/Tool/Inspect.lean` owns the inspector registry. The basic Nexus examples
compile directly and deliberately are not registered with the inspector. The resulting `DrivePlan`
and `ExperimentSpec` values are pure model artifacts: they describe selected requests,
model-owned outcomes, and semantic observations. The examples do not start a Temporal server or
execute Nexus operations.

Build each stage through the final module and target names:

```sh
cd model
mise exec -- lake build Umpire.Examples.Switch
mise exec -- lake build Temporal.Feature.Nexus.Examples.BasicLifecycleTests
mise exec -- lake build Temporal.Feature.Nexus.Examples.BasicOperationsTests
mise exec -- lake build Temporal.Feature.Nexus.CallerClosureTests
mise exec -- lake build Temporal TemporalModelTests temporal-model-inspect
```

From the Temporal repository root, run the focused regression check:

```sh
make umpire-check-regression
```

The focused check builds `Temporal`, `UmpireTests`, `TemporalModelTests`, and
`temporal-model-inspect`. It rejects obsolete interfaces, reusable Umpire domain leaks, and invalid
Feature/System import directions; compares repeated inspection with both checked-in target-state
fixtures byte-for-byte; and verifies that unknown or invalid inspector requests emit one structured
diagnostic with no artifact JSON on standard output. It also clean-regenerates the checked-in
caller-closure Go and Markdown projections and runs their focused Go tests. It does not require or
contact a running Temporal server.

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
collect evidence. Runtime driving, evidence qualification, and promotion are separate work.

Generated API declarations remain structural inputs only. Behavioral meaning, including whether
a selected action is applicable and which transition outcomes are possible, remains owned by the
authored Lean model.
