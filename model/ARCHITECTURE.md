# Temporal Lean model architecture

This directory contains neutral formal primitives, the reusable Umpire library, generated Temporal
structure, handwritten Temporal behavior, and the first Case Producer. The normative rules live in
[the Umpire 4 specification](../.plans/UMPIRE4_SPEC.md); the reusable API is described in
[Umpire/ARCHITECTURE.md](Umpire/ARCHITECTURE.md).

## Libraries and imports

| Import | Responsibility |
| --- | --- |
| `Shared` | Neutral transition and trace-replay primitives. |
| `Umpire` | Temporal-independent modeling, planning, promotion, and Case IR/compiler APIs. |
| `Temporal` | Generated Temporal structure, handwritten semantics, and Temporal Case production. |

Most consumers start with `import Umpire` or `import Temporal`. Focused imports should follow the
owner boundaries below:

```text
Shared
  └── transition and trace replay

Umpire.Core ──▶ Target ──▶ Property / Behavior / Query ──▶ Planning
                    ├────▶ Observation / ImplementationLink
                    └────▶ Space / Exploration / Promotion

Umpire.Case ──▶ Umpire.Case.Compiler
                         │
                         ▼
               Temporal.CaseRuntime

Temporal.API ───────────────────────────┐
Temporal.DynamicConfig ────────────────┤
Temporal.Feature ──────────────────────┼──▶ Temporal
Temporal.System ───────────────────────┤
Temporal.CaseRuntime ──────────────────┘
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

Planning artifacts and Generated Views remain useful model outputs. They are not inputs to the Case
Runtime and do not establish that a runtime action occurred.

## Case Runtime IR and Producer

`Umpire.Case` owns the closed version-one data model:

```text
Case
├── version, identity, provenance, definitions, Known Gaps
├── Program
│   ├── symbolic roles
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

`Umpire.Case.Compiler` checks producer inputs and lowers them into this IR. It rejects unsupported
instructions, contexts, types, paths, references, and limits instead of emitting an approximation.

`Temporal.CaseRuntime` is the first Producer. Its `GetSystemInfo` Case proves that the IR is not tied
to Nexus. Its async Nexus Case uses controller RPCs plus SDK workflow and Nexus-handler entrypoints
without adding a scenario opcode. The six conformance Cases cover the root Go facade's satisfied,
violated, inconclusive, static-rejection, cleanup-failure, and cross-Run classes.

`Temporal.Tool.CaseRuntime` is a build-time renderer only. Coordinates, credentials, clients,
workers, capabilities, and live IDs remain Host inputs.

## Go runtime boundary

The corresponding Go architecture is deliberately small:

```text
Case + immutable Profile
        │
        ▼
PrepareCase ──▶ PreparedCase
                    │
                    ▼
             Run(ctx, Host)
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
 internal execution       Contract verification
        │                       │
        └───────────┬───────────┘
                    ▼
             closed Run + Verdict
```

The root `tools/umpire` package owns the Profile/Host contract and the two calls. Scheduling,
recording, effect ownership, private Slot storage, and Monitor factories are internal. Static
preparation performs no Host I/O; a Prepared Case snapshots admitted inputs and supports independent
sequential and concurrent Runs.

Temporal server and worker authority do not overlap. The server Host supplies the authorized
descriptor catalog, transports prepared unary method/request pairs, and returns raw typed responses
and protocol status. Internal execution constructs requests and applies declared response
projections to private Slots and Run Observations. The worker Host uses Temporal SDK APIs for
workflow, activity, and Nexus-handler execution, owns reservation delivery, and cancels at
activation scope. The composite Host joins these capabilities without interpreting the Program or
Contract.

The Executor appends monotonic immutable Run Events. Each event has a unique source identity and
causal references to prior sources. The Evaluator observes the appended copy synchronously and uses
the same prepared Contract for offline evaluation. It checks horizon expiry before every
transition, keeps captures rule-local and Run-local, and records exact supporting event sequences.
Private Slots never become evidence automatically.

Stop prevents new controller dispatch and activation reservation, then cancellation, bounded drain,
and cleanup proceed through owned handles and a fresh cleanup context. Disposition, cleanup status,
and Verdict remain independent; a proved violation is not erased by cleanup failure. After closure,
late completion and Host diagnostics cannot mutate returned data.

## Artifact ownership and tests

`model/SEMANTIC_INVENTORY.md` is generated by `temporal-model-semantic-inventory`. The retained
planning Generated Views are generated by `umpire-gen-regression-views`. Case conformance fixtures
are rendered by `temporal-case-runtime` and published transactionally by
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
