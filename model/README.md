# Temporal Lean model

Lean owns behavioral meaning in this directory. Generated declarations describe API and dynamic
configuration structure; handwritten modules decide what that structure means. Runtime clients,
credentials, worker resources, callback authority, and live identifiers remain outside the model.

## Generated structural catalogs

`umpire-gen-lean-api` consumes serialized protobuf descriptor sets and exclusively owns
`Temporal/API.lean` and `Temporal/API/`. The generated modules describe messages, enums, maps,
oneofs, presence, recursion, and unary gRPC methods behind the stable `Temporal.API` facade. They do
not provide clients or assign product behavior.

`umpire-gen-lean-dynamic-config-catalog` snapshots Temporal's initialized production registry and
exclusively owns `Temporal/DynamicConfig.lean` and `Temporal/DynamicConfig/`. Handwritten
interpretation and validation live under `Temporal/System/Configuration/`; Callback and Matching
semantics remain in their owning consumer packages.

From the repository root, regenerate the catalogs through their owners:

```sh
make umpire-gen-lean-api
make umpire-gen-lean-dynamic-config-catalog
```

Each owner validates its complete candidate output before replacing managed files.

## Case production

`Umpire.Case` is the reusable, Temporal-independent IR. A Case contains exactly one bounded Program
and one deterministic Contract. Programs contain typed acyclic instruction graphs; Contracts
contain safety and bounded-liveness monitor machines over Run Events and declared Observations.

`Umpire.Case.Compiler` lowers checked Producer inputs and rejects unsupported constructs.
`Temporal.Feature.Nexus3.Testpilot` lowers the checked success-only Nexus3 completion Query and
witness into the async Nexus example. `Temporal.Testpilot` supplies the unrelated `GetSystemInfo`
example and the six public-facade conformance fixtures. `Temporal.Tool.Testpilot` renders canonical
ProtoJSON. The broader Nexus3 Markdown sketches remain design material rather than executable
coverage. Lean is the first Producer, while the Case format and Go runtime remain independent of
Lean.

Testpilot terms have precise boundaries:

- Slots are immutable, single-assignment private execution state. They are not recorded
  automatically.
- Observations are declared typed Run Event fields available to Contracts. Declared response
  projections are ordinary data; Slot privacy does not imply general response secrecy.
- Contract expiry is checked before transitions on every event kind. Captures are bounded and
  isolated per rule and per Run.
- Run disposition, cleanup status, and Verdict remain independent. A proved violation survives
  later cleanup failure.

## Semantic authoring and planning

The retained semantic model uses separate `Target`, `Property`, `Behavior`, `Query`, `Space`,
`Exploration`, and `Promotion` APIs. A checked Target owns behavior; Properties state trace claims;
Behaviors constrain trace shape; Queries ask bounded questions; Spaces and Exploration select
finite candidates. These packages do not perform runtime I/O.

`Umpire.Promotion` remains scenario-neutral. It replans an unchanged checked Query, validates the
complete planning anchor and exact source bytes, and returns an opaque review-only source value.
It has no Case execution authority and imports no Temporal scenario.

The `temporal-model-inspect` executable exposes the retained checked catalog and emits deterministic
planning artifacts. Generated Views remain navigation and test wrappers around that planning data;
they do not execute a Case or determine a Verdict.

### Ordinary Nexus authoring

`Temporal.Feature.Nexus` is the compiled established walkthrough. Read
`Lifecycle.Semantics`, `Lifecycle.Target`, the three `Operations` modules, and `Observation` in that
order. The finite `Lifecycle.finiteMachine` is the ordinary Target seam: authors still provide the
ordered domains, encoders, enumerators, closure proofs, and Action-executability proof, while
`targetDefinition` and `authoredTarget` remove repeated record and planning transport. Authors who
need an independently specified authoritative relation can use the expert `TransitionKernel` path.

Property, Behavior, Query, and Observation inputs remain ordinary values. Call each language's
`check` operation to inspect its typed `Except` error, then supply explicit checker-success evidence
to its `checked` operation. Stable `DefinitionId` suffixes, source locations, providers/connectors,
Target-owned outcomes, and stage-specific `QueryLimitSpec` values are authored choices; declaration
order and instance search choose none of them. Planning returns `Except KnownGapError PlannerRun`.
An optional checked `authoredKnownGaps` set is composed with phase gaps before search or artifact
publication. Gaps describe limits and missing evidence; they cannot make a Property pass or imply
that an omitted limitation was detected.

Lean syntax used by the walkthrough:

- `:=` defines a value; `{ base with field := value }` makes a record update.
- `.case` selects an inferred enum or structure constructor.
- `Except Error Value` is either `.error error` or `.ok value`; `do` and `←` stop on the first error.
- `by` starts a proof, and the explicit proof argument to `checked` is the raw/check/checked seam.
- `#guard_msgs` compiles an expected elaboration failure; `#print axioms` reports transitive trust.

`Temporal.Feature.NexusTests` compiles this facade-only path, including an authored gap reaching a
real selected artifact, Observation evaluation, malformed identity/reference, missing proof,
incomplete Target, invalid transition, and invalid Observation specimens. The exact compatibility,
trust, and cost inventory is in [the established evidence record](Temporal/Feature/Nexus/EVIDENCE.md).

The experimental [Nexus2 authoring prototype](Temporal/Feature/Nexus2/README.md) demonstrates the
ordinary finite route, guarded Properties, bounded case analysis, and constructor/frontend
measurements under its narrow prototype exceptions. It is a separate `temporal.nexus2.*` model,
not the established migration or a production authoring rule. Editor responsiveness, cold/repeated
elaboration, human readability, product-owner usability, and broader syntax approval remain
unmeasured. Its [evidence inventory](Temporal/Feature/Nexus2/EVIDENCE.md) records those boundaries.

## Runtime ownership

The Go runtime is the consumer of canonical Case data:

```text
Lean or another Producer
        │
        ▼
      Case { Program, Contract }
        │
        ▼
testpilot.Prepare(case, Profile) ──▶ immutable PreparedCase
        │
        ▼
PreparedCase.Run(ctx, Driver) ──▶ immutable Run + Verdict
```

`common/testing/testpilot` owns the Case protocol and public Profile, Driver, and two-call facade.
Its private execution package owns scheduling, recording, effect lifecycle, private Slot state,
and bounded cleanup. Its private verification package owns Contract preparation, fresh Run-local
Monitors, and offline evaluation. Umpire remains the Lean authoring and Producer owner.

Temporal authority remains split:

- `tests/testcore/testpilot/server` supplies the authorized descriptor catalog and transports prepared
  unary method/request pairs, returning raw typed responses and protocol status.
- `tests/testcore/testpilot/worker` owns SDK workflow, activity, and Nexus-handler interpretation,
  reserved activation delivery, and activation-level cancellation.
- `tests/testcore/testpilot` composes server and worker Drivers without interpreting scenario or Contract
  semantics.

Internal execution constructs typed requests and applies declared response projections to private
Slots and Run Observations.

## Generated artifacts

The checked semantic inventory is the generated navigation view
[`SEMANTIC_INVENTORY.md`](SEMANTIC_INVENTORY.md). Its owner commands are:

```sh
make umpire-gen-semantic-inventory
make umpire-check-semantic-inventory
```

The retained planning Generated Views are owned transactionally:

```sh
make umpire-gen-regression-views
make umpire-check-regression-views
```

The Testpilot conformance and example trees are independently owner-managed by the same tool:

```sh
make umpire-check-case-runtime-conformance
make umpire-gen-case-runtime-conformance  # separate reviewed promotion
model/.lake/build/bin/temporal-testpilot async-nexus
mise exec -- go test -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotAsyncNexusCase$'
```

The check builds the Lean renderer, creates and validates both the complete twelve-file conformance
tree and the two-file `tests/testcore/testpilot/testdata` example tree under one physical temporary
root, and recursively diffs each independently owned root against the checkout. The promotion target
is a separate action. Ordinary Go tests only read checked-in fixtures; they invoke neither Lean nor
a rewrite mode.

The corpus contains exactly these facade proof classes:

1. satisfied;
2. violated;
3. inconclusive;
4. static preparation rejection;
5. cleanup failure after a proved violation; and
6. cross-Run isolation.

Lean-produced Cases compare byte-for-byte. Runtime results compare through one named closed stable
projection, while Run IDs, elapsed values, event/source identities, causal references, activation
identities, support references, and diagnostics are checked structurally. There is no generic
normalization or ignore mechanism.

## Build and regression

From the repository root:

```sh
make umpire-build-model
make umpire-check-regression
make fmt-imports
make lint-code
```

The aggregate regression check regenerates owner-managed artifacts into temporary roots, checks the
active vocabulary and semantic inventory, runs every package under `tools/umpire` with
`-tags test_dep`, builds the complete Lean roots including generic promotion and the Case renderer,
and runs the complete live selector with `-tags 'test_dep integration' -run '^TestUmpire'`. The live
gate compares the entire inherited failure-identity set, so both additions and deletions fail.

`make lint-model` runs Lean declaration linting and validates the complete first-party import graph.
The regression boundary intentionally adds no broad generated-Lean API drift check and no new
GitHub Actions surface.

## Superseded runtime history

The pre-fn-64 portable-plan, resident-executor, caller-specific adapter, and separate Run Evaluation
interfaces were removed. Historical planning documents label those names explicitly as superseded;
they are not supported runtime entry points or compatibility targets.
