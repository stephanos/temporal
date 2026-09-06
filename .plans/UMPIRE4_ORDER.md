# Umpire 4 delivery order

Build on the completed Case Runtime and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

### 1. fn-66 — Remove unused Umpire tooling

[Plan](../.flow/specs/fn-66-remove-unused-umpire-tooling-after.md) reviewed **SHIP**.
Its fn-62 dependency and fn-64 foundation are complete.

Execute the three tasks in order:

1. Inventory every remaining `tools/umpire` package and command, its consumers, and candidate tests
   and fixtures. Resolve ambiguous ownership before deletion.
2. Retire the unused public artifact package and CLI, their exclusive tests and fixtures, and
   obsolete build and documentation references.
3. Trim orphaned internal codecs while preserving the complete Experiment reader and retained
   runtime, Producer, authoring, generation, regression, and downstream contracts.

Extend the [fn-64 deletion ledger](../.flow/artifacts/fn-64-umpire-case-runtime/task8-migration-ledger.md)
without repeating that cutover. Fn-60 is not a dependency.

### 2. fn-60 — Deepen handwritten Lean canonical JSON construction

[Plan](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md). **Optional maintenance**, with
no open dependency; it does not gate fn-66 or downstream delivery.

Consolidate handwritten Core Limit, Target, Behavior, Query, Space, Exploration, Observation, and
Implementation Link formatters behind `Umpire.Json`. Preserve exact bytes, metadata, fingerprints,
interfaces, imports, trust, performance, and comments. Exclude `Umpire.Property`, generated protobuf
encoding, parsing, protocol changes, drift gates, and CI work.

Refresh the plan against the completed cutovers before execution, including existing boolean
support and focused per-task verification.

## Downstream delivery

Each spec needs a fresh Case Runtime plan review before implementation. Prior reviews of the
retired execution architecture do not approve the rewritten plans. Completed dependencies do not
block replanning or execution.

| Spec | Dependencies | Next action |
| --- | --- | --- |
| [fn-33 — Bounded exploration](../.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md) | fn-40 and fn-64 complete | Review whole-Case candidates, serial coordination, lost iterations, semantic coverage, and bounded 10x behavior. |
| [fn-22 — Replay and reduction](../.flow/specs/fn-22-deterministic-replay-semantic.md) | fn-5 and fn-64 complete | Resolve **MAJOR_RETHINK** before implementation: separate exact candidate identity from Contract-relative violation equivalence, prove the negative Case before reduction, and retain explicit offline semantic replay and checked promotion. |
| [fn-26 — Qualification receipts](../.flow/specs/fn-26-local-qualification-receipts-and-staged.md) | fn-48 and fn-64 complete | Review offline Case/Profile/Host/Run/Verdict admission, receipt multiplicity, and idempotent publication. Assessment must never create or replay a Run. |
| [fn-29 — Production canary](../.flow/specs/fn-29-bounded-production-canary-execution-and.md) | **fn-26**; fn-48 and fn-64 complete | After fn-26 ships, review external policy and credentials, serial Runs, leases, lost Runs, reconciliation without redispatch, and publication. |

All runtime work retains `PrepareCase(case, profile)` → `PreparedCase.Run(ctx, host)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Umpire.

## Completed cutovers

- [fn-64](../.flow/specs/fn-64-umpire-case-runtime.md): Case Runtime, Lean-produced Nexus proof,
  independent six-class facade corpus, and full regression gate.
- [fn-65](../.flow/specs/fn-65-design-and-prototype-approachable.md): separate Nexus2 authoring
  prototype. Its [requirement comparison](../model/Temporal/Feature/Nexus2/EVIDENCE.md) covered fn-62 R3.
- [fn-62](../.flow/specs/fn-62-make-ordinary-temporal-model-authoring.md): remaining established
  authoring requirements, including Observation construction and model-owned Known Gaps;
  [compatibility evidence](../model/Temporal/Feature/Nexus/EVIDENCE.md).

All three passed whole-spec completion review. Nexus2 remains a prototype with explicit adoption
boundaries.

## Deferred and superseded

These entries are outside the delivery queue and are not prerequisites for it.

| Deferred spec | Revisit when |
| --- | --- |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the proven model family. |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md) | Optional checker adoption becomes valuable. |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md) | A verification receipt/profile platform is justified. |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md) | A second verification backend is justified; caller closure remains historical. |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md) | Real Claim Assessment evidence supports release governance. |

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Case Runtime proposal with an independent oracle.
