# Umpire components

The Testpilot runtime component map: how a Case moves from its Producer through preparation and
execution to a Verdict. The rules it realizes are in the [Umpire 4 specification](UMPIRE4_SPEC.md).
The earlier delivery inventory, which recorded the removed pre-cutover architecture, is archived as
[`archive/UMPIRE4_COMPONENTS_HISTORY.md`](archive/UMPIRE4_COMPONENTS_HISTORY.md).

## Testpilot runtime component map

```text
Producer ──▶ Case { Program, Contract }
                 │
                 ▼
       testpilot.Prepare(case, Profile)
                 │
          immutable PreparedCase
                 │
                 ▼
       PreparedCase.Run(ctx, Driver)
                 │
          ┌──────┴──────┐
          ▼             ▼
 internal Executor   Contract Evaluator
          │             │
          └──────┬──────┘
                 ▼
      immutable Run + Verdict
```

The `.proto` closure rooted at `proto/internal/temporal/server/api/testpilot/v1/case.proto` owns the
Case wire schema; every Producer builds Cases from bindings generated from it.
`common/testing/testpilot` owns `Profile`, `Driver`, `PreparedCase`, `Prepare`, and
`PreparedCase.Run`. The vocabulary a `Driver` and its `Session` speak (coordinates, effect and
reservation handles, the handle bridge, Profile role policy, and Opcodes) is declared once in the
`common/testing/testpilot/contract` leaf, which the facade re-exports by alias and private execution
imports directly. Testpilot's private execution package owns scheduling, recording, effects, private
Slots, and cleanup. Its private verification package owns static Contract preparation, fresh Run-local
Monitors, bounded captures, expiry-before-transition semantics, and offline evaluation.

Exact Case 1.0 is the sole format. Resource-bearing Programs declare symbolic namespace, task-queue,
and named Nexus endpoint IDs; resource-free Programs may have an empty environment. Those IDs are not physical
names or transport addresses. `Prepare` snapshots the Profile-owned physical values, resolves private
prepared inputs, and adds the complete binding fingerprint to Prepared Case identity without changing
the symbolic source Case, Contract, Behavior Fingerprints, or producer provenance. `Run` compares the
full Driver identity, calls the no-I/O `Validate` hook, creates the Monitor, and only then calls
`Open`. The Temporal Driver derives worker and request resources from the same prepared roles.

Temporal Driver authority is split by execution context. `common/testing/testpilot/temporal/server` supplies the
authorized descriptor catalog and transports prepared unary method/request pairs, returning raw
typed responses and protocol status. Internal execution constructs requests and applies response
projections to Slots and Observations. `common/testing/testpilot/temporal/worker` owns SDK workflow, activity,
Nexus-handler execution, reserved activation delivery, and activation-level cancellation.
`common/testing/testpilot/temporal` composes those Driver capabilities without interpreting Case semantics.
Transport targets, credentials, SDK clients, callback authority, and lifecycle configuration remain
environment-owned Driver inputs in both resource modes.

A Producer owns only its opaque provenance and builds Cases from the generated protocol bindings.
Deterministic Case fixtures are owned by `umpire-gen-case-runtime-conformance`; its check generates
and validates a complete temporary tree before diffing, while promotion is a separate reviewed
target. The regression boundary includes the six facade classes, the full package-local suite, the
exact live selector and exact inherited failure identities, model builds, generated views, and the
semantic inventory.
