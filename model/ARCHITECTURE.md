# Temporal Lean model architecture

This directory contains neutral formal primitives, the reusable Umpire library, generated Temporal
structure, handwritten Temporal behavior, and the first Case Producer. The normative rules live in
[the Umpire 4 specification](../.plans/UMPIRE4_SPEC.md); the reusable API is described in
[Umpire/ARCHITECTURE.md](Umpire/ARCHITECTURE.md).

## Libraries and imports

| Import | Responsibility |
| --- | --- |
| `Shared` | Neutral transition and trace-replay primitives. |
| `Testpilot` | Generated Testpilot protocol, producer-neutral authoring, and ProtoJSON policy. |
| `Umpire` | Temporal-independent modeling, planning, promotion, and Testpilot provenance. |
| `Temporal` | Generated Temporal structure, handwritten semantics, and Temporal Case production. |

Most consumers start with `import Umpire` or `import Temporal`. Focused imports should follow the
owner boundaries below:

```text
Shared
  └── transition and trace replay

Umpire.Core ──▶ Target ──▶ Property / Behavior / Query ──▶ Planning
                    ├────▶ Observation / ImplementationLink
                    └────▶ Space / Exploration / Promotion

Testpilot.Protocol ──▶ Testpilot.Authoring ──▶ Temporal.Testpilot
         │                       ▲                       ▲
         └────▶ Testpilot.ProtoJSON          Umpire provenance

Temporal.API ───────────────────────────┐
Temporal.DynamicConfig ────────────────┤
Temporal.Feature ──────────────────────┼──▶ Temporal
Temporal.System ───────────────────────┤
Temporal.Testpilot ────────────────────┘
```

`Umpire.*` never imports `Temporal.*`. `Temporal.Feature.*` and `Temporal.System.*` remain separate
except for the exact checked Implementation Link leaf. `Temporal.Tool.*` owns developer commands and
is not imported by the production aggregate. `make lint-model` checks these edges against the full
source inventory and compiled module metadata.

## Generated structure

`Temporal.API` contains descriptor-derived message, enum, field, presence, map, oneof, recursion,
and unary-method facts. `Temporal.DynamicConfig` contains the complete generated registry snapshot.
Neither generated family defines behavior. Handwritten Feature and System modules interpret only
the facts they explicitly import.

The owner commands validate a complete candidate before replacing managed output:

```sh
make umpire-gen-lean-api
make umpire-gen-lean-dynamic-config-catalog
```

## Semantic model

The retained semantic APIs keep these responsibilities separate:

- Target owns valid setup, state, Action, outcome, observation, transition, and capability domains.
- Property states a claim over model traces.
- Behavior constrains allowed trace shape without choosing target-owned outcomes.
- Query asks one bounded planning question.
- Space and Exploration select from a finite checked universe without performing runtime I/O.
- Observation and Implementation Link retain the offline semantic mapping path for model analysis.
- Promotion validates one exact scenario-neutral planned source for human review.

Planning artifacts and Generated Views remain useful model outputs. They are not inputs to Testpilot
and do not establish that a runtime action occurred.

## Testpilot protocol and Producers

The checked-in `.proto` closure rooted at
`proto/internal/temporal/server/api/testpilot/v1/case.proto` is the sole Case wire schema.
`Testpilot.Protocol` exposes the generated Lean declarations, and `Testpilot.Authoring` constructs
those generated values through context-safe helpers:

```text
Case
├── version, identity, opaque producer provenance
├── Program
│   ├── symbolic roles
│   ├── Case 1.1 symbolic environment declarations and direct resource references
│   ├── typed private Slots and declared Observations
│   ├── controller / workflow / activity / Nexus-handler DAGs
│   ├── cleanup graph
│   └── independent structural and runtime limits
└── Contract
    ├── deterministic safety and bounded-liveness rules
    ├── bounded captures
    ├── expiry-before-transition horizons
    └── independent work and storage limits
```

`Umpire.Case` retains only Umpire's producer-specific definitions, fingerprints, sources, and Known
Gaps and encodes them into opaque provenance bytes. It does not own a parallel Program, Contract,
Run, or field serializer. Producers validate their semantic inputs and use `Testpilot.Authoring`.
Umpire-backed Producers use `Umpire.Case.Compiler` for source-bound rule validation, exact opaque
provenance, and final assembly from generated values. `Testpilot.ProtoJSON` delegates canonical
encoding to `Protobuf.Json`.

`Temporal.Testpilot` is the first Producer. Its `GetSystemInfo` Case proves that the IR is not tied
to Nexus. Its async Nexus Case uses controller RPCs plus SDK workflow and Nexus-handler entrypoints
without adding a scenario opcode. The six conformance Cases cover the root Go facade's satisfied,
violated, inconclusive, static-rejection, cleanup-failure, and cross-Run classes.

`Temporal.Tool.Testpilot` is a build-time renderer only. Coordinates, credentials, clients,
workers, capabilities, and live IDs remain Driver inputs.

Literal-only Programs use Case 1.0. A binding-bearing Program uses Case 1.1 and declares a closed
set of symbolic text IDs for namespaces, task queues, and named Nexus endpoints. These declarations
express resource relationships, not physical values; an endpoint role or binding ID is not a network
address. The immutable Profile supplies physical values, while transport targets, credentials,
callback authority, SDK clients, and lifecycle configuration remain environment-owned Driver inputs.
Changing a Profile binding therefore changes prepared/Driver binding identity, not the source Case,
its Contract, Behavior Fingerprints, or opaque producer provenance.

## Go runtime boundary

The corresponding Go architecture is deliberately small:

```text
Case + immutable Profile
        │
        ▼
testpilot.Prepare ──▶ PreparedCase
                    │
                    ▼
             Run(ctx, Driver)
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
 internal execution       Contract verification
        │                       │
        └───────────┬───────────┘
                    ▼
             closed Run + Verdict
```

The Testpilot `.proto` files own the Case protocol. `common/testing/testpilot` owns the
Profile/Driver contract and the two Go calls.
Scheduling, recording, effect ownership, private Slot storage, and Monitor factories are internal.
Static preparation performs no Driver I/O; a Prepared Case snapshots admitted inputs and supports independent
sequential and concurrent Runs.

Temporal server and worker authority do not overlap. The server Driver supplies the authorized
descriptor catalog, transports prepared unary method/request pairs, and returns raw typed responses
and protocol status. Internal execution constructs requests and applies declared response
projections to private Slots and Run Observations. The worker Driver uses Temporal SDK APIs for
workflow, activity, and Nexus-handler execution, owns reservation delivery, and cancels at
activation scope. The composite Driver joins these capabilities without interpreting the Program or
Contract. The composite and its `server` and `worker` packages live under
`common/testing/temporaltestpilot`.

`Prepare` snapshots and resolves Case 1.1 bindings without target I/O. A Prepared Case retains the
unchanged symbolic source and private resolved instruction and role data, and its identity includes
the complete Profile binding fingerprint. `Run` compares that identity, invokes the Driver's no-I/O
`Validate` hook, creates the Contract Monitor, and then calls `Open`. The shared Temporal Driver has
an explicit symbolic mode that derives request carriers, worker namespaces, task queues, and Nexus
routes from those prepared bindings, and an explicit legacy mode for literal-only Case 1.0 Programs.
The modes cannot be mixed and legacy resource maps never fill missing symbolic bindings.

The Executor appends monotonic immutable Run Events. Each event has a unique source identity and
causal references to prior sources. The Evaluator observes the appended copy synchronously and uses
the same prepared Contract for offline evaluation. It checks horizon expiry before every
transition, keeps captures rule-local and Run-local, and records exact supporting event sequences.
Private Slots never become evidence automatically.

Stop prevents new controller dispatch and activation reservation, then cancellation, bounded drain,
and cleanup proceed through owned handles and a fresh cleanup context. Disposition, cleanup status,
and Verdict remain independent; a proved violation is not erased by cleanup failure. After closure,
late completion and Driver diagnostics cannot mutate returned data.

## Artifact ownership and tests

`model/SEMANTIC_INVENTORY.md` is generated by `temporal-model-semantic-inventory`. The retained
planning Generated Views are generated by `umpire-gen-regression-views`. Case conformance fixtures
are rendered by `temporal-testpilot` and published transactionally by
`umpire-gen-case-runtime-conformance`.

The Case fixture check creates the complete candidate tree in a physical temporary directory,
validates all Cases and expected projections, then recursively diffs it with the checkout.
Promotion is a separate target. Go tests consume checked-in fixtures without invoking Lean or
rewriting data.

`TemporalModelTests` imports the ordinary Temporal model tests. `UmpireTests` imports reusable
Umpire tests, including the scenario-neutral promotion source checks. `TemporalExperimentalTests`
retains experimental model tests that still exist, without restoring deleted runtime adapters.

The full regression boundary is:

```sh
make umpire-build-model
make umpire-check-regression
```

It checks owner-managed outputs, active vocabulary, the complete package-local Go suite, the exact
tagged `^TestUmpire` live selector and inherited failure identities, the Case facade corpus, generic
promotion, and the model roots. It intentionally defines no broad generated-Lean drift policy and
adds no GitHub Actions coverage.

## Superseded runtime history

Before fn-64, model docs described a portable plan, caller-specific execution adapter, resident
service, and separate Run Evaluation pipeline. Those runtime interfaces were removed. Historical
design documents mark them explicitly as superseded; they are not current package or command
boundaries.
