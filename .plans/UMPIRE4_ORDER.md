# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

### 1. fn-68 — Minimal Nexus3 success demonstration

[Plan](../.flow/specs/fn-68-minimal-nexus3-success-demonstration.md) reviewed **SHIP**.
Deliver after fn-69; its fn-62 and fn-64 dependencies are complete.

Execute the three tasks in order:

1. Compile the success-only model, keeping `Nexus3/Nexus.lean` within 250 lines.
2. Lower its checked Query and witness into a Case, with integration in separate Nexus3 modules.
3. Generate the fixture and pass the existing live Temporal test for scheduled → started → succeeded.

Keep the demonstration minimal: one Property, one Behavior, one Query, and no cancellation or new
DSL. Model changes must affect the generated Case or reject explicitly.

## Downstream delivery

Each spec needs a fresh Testpilot plan review before implementation. Prior reviews of the
retired execution architecture do not approve the rewritten plans. Completed dependencies do not
block replanning or execution.

| Spec | Dependencies | Next action |
| --- | --- | --- |
| [fn-33 — Bounded exploration](../.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md) | fn-40, fn-64, and fn-69 | Review whole-Case candidates, serial coordination through Testpilot, lost iterations, semantic coverage, and bounded 10x behavior. |
| [fn-22 — Replay and reduction](../.flow/specs/fn-22-deterministic-replay-semantic.md) | fn-5, fn-64, and fn-69 | Resolve **MAJOR_RETHINK** before implementation: separate exact candidate identity from Contract-relative violation equivalence, prove the negative Case before reduction, and retain explicit offline semantic replay and checked promotion. |
| [fn-26 — Qualification receipts](../.flow/specs/fn-26-local-qualification-receipts-and-staged.md) | fn-48, fn-64, and fn-69 | Review offline Testpilot Case/Profile/Run/Verdict admission, receipt multiplicity, and idempotent publication. Assessment must never create or replay a Run. |
| [fn-29 — Production canary](../.flow/specs/fn-29-bounded-production-canary-execution-and.md) | **fn-26**; fn-48, fn-64, and fn-69 | After fn-26 ships, review external policy and credentials, serial Testpilot Runs, leases, lost Runs, reconciliation without redispatch, and publication. |

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire.

## Completed cutovers

- [fn-69](../.flow/specs/fn-69-extract-testpilot-from-umpire.md): moved the Case protocol and
  reusable runtime to `common/testing/testpilot`, moved the functional Driver to
  `tests/testcore/testpilot`, refined the Testpilot protobuf model, migrated all consumers, and
  removed the former Umpire protocol/runtime owners.
- [fn-66](../.flow/specs/fn-66-remove-unused-umpire-tooling-after.md): removed the unused public
  Artifact package and CLI plus orphaned internal codecs; retained the narrow Experiment reader,
  Testpilot, generators, regression gates, and downstream contracts.
- [fn-64](../.flow/specs/fn-64-umpire-case-runtime.md): Testpilot, Lean-produced Nexus proof,
  independent six-class facade corpus, and full regression gate.
- [fn-65](../.flow/specs/fn-65-design-and-prototype-approachable.md): separate Nexus2 authoring
  prototype. Its [requirement comparison](../model/Temporal/Feature/Nexus2/EVIDENCE.md) covered fn-62 R3.
- [fn-62](../.flow/specs/fn-62-make-ordinary-temporal-model-authoring.md): remaining established
  authoring requirements, including Observation construction and model-owned Known Gaps;
  [compatibility evidence](../model/Temporal/Feature/Nexus/EVIDENCE.md).

All five passed whole-spec completion review. Nexus2 remains a prototype with explicit adoption
boundaries.

## Deferred and superseded

These entries are outside the delivery queue and are not prerequisites for it.

| Deferred spec | Revisit when |
| --- | --- |
| [fn-60](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md) | Optional handwritten canonical JSON consolidation becomes worth prioritizing; it has no downstream dependency. |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the proven model family. |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md) | Optional checker adoption becomes valuable. |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md) | A verification receipt/profile platform is justified. |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md) | A second verification backend is justified; caller closure remains historical. |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md) | Real Claim Assessment evidence supports release governance. |

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Testpilot proposal with an independent oracle.
