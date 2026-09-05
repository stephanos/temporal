# Umpire semantic inventory

> Generated from typed Lean catalogs. Do not edit semantic meaning in this file.

## Outcome families

### `umpire.semantic-inventory.outcome-family.01-planning`

Owner: `Umpire.PlanningOutcome`

Bounded planning outcomes.

| Outcome | Meaning |
| --- | --- |
| `found` | Planning selected one Model Trace. |
| `verified-within-limits` | Planning verified the requested universal claim within complete Limits. |
| `no-such-trace-within-complete-limits` | Complete bounded search found no matching Model Trace. |
| `limit-reached` | Planning reached its search Limit before completing the Query. |
| `unsatisfiable` | The checked Behavior admits no Model Traces. |
| `invalid` | Planning rejected the Query. |

### `umpire.semantic-inventory.outcome-family.02-execution-phase`

Owner: `Umpire.Artifact.PhaseOutcomeStatus`

Execution phase outcomes.

| Outcome | Meaning |
| --- | --- |
| `not-started` | The phase did not start. |
| `succeeded` | The phase completed successfully. |
| `failed` | The phase failed. |
| `timed-out` | The phase reached its time Limit. |
| `canceled` | The phase was canceled. |

### `umpire.semantic-inventory.outcome-family.03-control-attempt`

Owner: `Umpire.Artifact.ControlAttemptStatus`

Requested control-attempt outcomes.

| Outcome | Meaning |
| --- | --- |
| `accepted` | The control was accepted. |
| `rejected` | The control was rejected. |
| `unsupported` | The control is unsupported. |
| `failed` | The control attempt failed. |
| `canceled` | The control attempt was canceled. |
| `not-attempted` | The control was explicitly not attempted. |

### `umpire.semantic-inventory.outcome-family.04-source-closure`

Owner: `Umpire.Artifact.SourceClosureStatus`

Raw Evidence source-closure outcomes.

| Outcome | Meaning |
| --- | --- |
| `closed` | The evidence source closed completely. |
| `partial` | The evidence source closed only partially. |
| `failed` | Closing the evidence source failed. |

### `umpire.semantic-inventory.outcome-family.05-cleanup`

Owner: `Umpire.Artifact.CleanupStatus`

Run cleanup outcomes.

| Outcome | Meaning |
| --- | --- |
| `complete` | Cleanup completed. |
| `incomplete` | Cleanup left open handles. |
| `failed` | Cleanup failed. |

### `umpire.semantic-inventory.outcome-family.06-operational`

Owner: `Umpire.Artifact.OperationalStatus`

Overall operational outcomes.

| Outcome | Meaning |
| --- | --- |
| `succeeded` | Execution succeeded. |
| `incomplete` | Execution is incomplete. |
| `failed` | Execution failed. |

### `umpire.semantic-inventory.outcome-family.07-observation`

Owner: `Umpire.ObservationStatus`

Observation Evaluation outcomes.

| Outcome | Meaning |
| --- | --- |
| `accepted` | Observation Evaluation produced one Evidence-backed Model Trace. |
| `unknown` | Observation Evaluation could not decide from the available Evidence. |
| `conflict` | Observation Evaluation found contradictory Evidence. |
| `unsupported` | Observation Evaluation does not support the supplied Evidence vocabulary. |

### `umpire.semantic-inventory.outcome-family.08-implementation-link`

Owner: `Umpire.ImplementationLinkStatus`

Implementation Link application outcomes.

| Outcome | Meaning |
| --- | --- |
| `applied` | The Implementation Link produced one complete destination Model Trace. |
| `invalid` | The Implementation Link or its source input was invalid. |
| `unknown` | The Implementation Link could not decide within the available input or Limits. |
| `conflict` | The Implementation Link found contradictory mappings or Evidence. |
| `unsupported` | The Implementation Link does not support the supplied vocabulary. |

### `umpire.semantic-inventory.outcome-family.09-semantic-property`

Owner: `Umpire.SemanticVerdictStatus`

Semantic Property evaluation outcomes.

| Outcome | Meaning |
| --- | --- |
| `satisfied` | The semantic Property is satisfied. |
| `violated` | The semantic Property is violated. |
| `unknown` | The semantic Property cannot be decided from the available Evidence. |
| `conflict` | The semantic Property evaluation found conflicting Evidence. |
| `unsupported` | The semantic Property evaluation does not support the supplied input. |

### `umpire.semantic-inventory.outcome-family.10-strict-query`

Owner: `Umpire.StrictQueryStatus`

Strict Query projection outcomes.

| Outcome | Meaning |
| --- | --- |
| `satisfied` | Every required semantic Property is satisfied. |
| `violated` | At least one required semantic Property is violated. |
| `incomplete` | The strict Query does not have one complete consistent verdict set. |

## Projection sentinels

These rendered values represent an unevaluated projection; they are not outcome constructors.

| ID | Owner | Value | Meaning |
| --- | --- | --- | --- |
| `implementation-link.not-evaluated` | `Implementation Link` | `not-evaluated` | The optional Implementation Link stage was not evaluated. |

## Known Gap flows

Each row identifies one authored source, synthesized family, projection, exact carry, or test-only reference.

| Catalog ID | Owner | Lineage | Scope | Shape | Source/reference | Field mapping | Description |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `umpire.semantic-inventory.known-gap-source.01-execution-evidence` | `Umpire.Artifact` | authored | production | exact-known-gap | `umpire.known-gap.execution-evidence` | — | Execution Evidence is unavailable during pure planning. |
| `umpire.semantic-inventory.known-gap-source.02-artifact-migrations` | `Umpire.Artifact` | authored | production | exact-known-gap | `umpire.known-gap.artifact-migrations` | — | Artifact migration interpretation is outside pure planning. |
| `umpire.semantic-inventory.known-gap-source.03-artifact-reading` | `Umpire.Artifact` | authored | production | exact-known-gap | `umpire.known-gap.artifact-reading` | — | Persisted Artifact reading is outside pure planning. |
| `umpire.semantic-inventory.known-gap-source.04-evidence-evaluation` | `Umpire.Artifact` | authored | production | exact-known-gap | `umpire.known-gap.evidence-evaluation` | — | Runtime Evidence evaluation is outside pure planning. |
| `umpire.semantic-inventory.known-gap-source.05-runtime-scheduler-order` | `Umpire.Artifact` | authored | production | exact-known-gap | `umpire.known-gap.runtime-scheduler-order` | — | Runtime scheduler ordering is unavailable during pure planning. |
| `umpire.semantic-inventory.known-gap-source.06-runtime-storage-order` | `Umpire.Artifact` | authored | production | exact-known-gap | `umpire.known-gap.runtime-storage-order` | — | Runtime storage ordering is unavailable during pure planning. |
| `umpire.semantic-inventory.known-gap-source.07-runtime-transport-order` | `Umpire.Artifact` | authored | production | exact-known-gap | `umpire.known-gap.runtime-transport-order` | — | Runtime transport ordering is unavailable during pure planning. |
| `umpire.semantic-inventory.known-gap-source.08-promotion` | `Umpire.Artifact` | authored | production | exact-known-gap | `umpire.known-gap.promotion` | — | Promotion is not established by pure planning. |
| `umpire.semantic-inventory.known-gap-source.09-observation-diagnostic` | `Umpire.Observation` | synthesized | production | generated-known-gap-family | `umpire.observation.*` | — | A closed Observation diagnostic synthesized during Run Evaluation. |
| `umpire.semantic-inventory.known-gap-source.10-implementation-link-setup` | `Umpire.ImplementationLink` | authored | production | authored-implementation-link-known-gap-family | `setup` | — | Polymorphic authored setup Known Gaps retained by an Implementation Link declaration. |
| `umpire.semantic-inventory.known-gap-source.11-implementation-link-state` | `Umpire.ImplementationLink` | authored | production | authored-implementation-link-known-gap-family | `state` | — | Polymorphic authored state Known Gaps retained by an Implementation Link declaration. |
| `umpire.semantic-inventory.known-gap-source.12-implementation-link-action` | `Umpire.ImplementationLink` | authored | production | authored-implementation-link-known-gap-family | `action` | — | Polymorphic authored action Known Gaps retained by an Implementation Link declaration. |
| `umpire.semantic-inventory.known-gap-source.13-implementation-link-outcome` | `Umpire.ImplementationLink` | authored | production | authored-implementation-link-known-gap-family | `outcome` | — | Polymorphic authored outcome Known Gaps retained by an Implementation Link declaration. |
| `umpire.semantic-inventory.known-gap-source.14-implementation-link-observation` | `Umpire.ImplementationLink` | authored | production | authored-implementation-link-known-gap-family | `observation` | — | Polymorphic authored observation Known Gaps retained by an Implementation Link declaration. |
| `umpire.semantic-inventory.known-gap-source.15-implementation-link-relation` | `Umpire.ImplementationLink` | authored | production | authored-implementation-link-known-gap-family | `relation` | — | Polymorphic authored relation Known Gaps retained by an Implementation Link declaration. |
| `umpire.semantic-inventory.known-gap-source.16-implementation-link-capability` | `Umpire.ImplementationLink` | authored | production | authored-implementation-link-known-gap-family | `capability` | — | Polymorphic authored capability Known Gaps retained by an Implementation Link declaration. |
| `umpire.semantic-inventory.known-gap-source.17-request-raw-known-gap-input` | `Umpire.Case` | carried | production | admitted-known-gap-input | `umpire.case.known-gap-input` | — | Validated Case Known Gaps before stage-specific projection. |
| `umpire.semantic-inventory.known-gap-source.18-observation-known-gap-admission` | `Umpire.Observation` | carried | production | evidence-gap-admission-projection | `umpire.semantic-inventory.known-gap-source.17-request-raw-known-gap-input` | code -> code; subject.toList -> relatedDefinitionIds; kind -> absent; detail -> absent | Request and Raw Evidence Known Gaps admitted as lossy Evidence Gaps. |
| `umpire.semantic-inventory.known-gap-source.19-result-request-raw-known-gap-carry` | `Umpire.Artifact.Result` | carried | production | carried-catalog-entry | `umpire.semantic-inventory.known-gap-source.17-request-raw-known-gap-input` | kind -> kind; code -> code; subject -> subject; detail -> detail | Request and Raw Evidence Known Gaps carried exactly into Result. |
| `umpire.semantic-inventory.known-gap-source.20-result-observation-known-gap-carry` | `Umpire.Artifact.Result` | carried | production | carried-catalog-entry | `umpire.semantic-inventory.known-gap-source.09-observation-diagnostic` | kind -> kind; code -> code; subject -> subject; detail -> detail | Synthesized Observation Known Gaps carried exactly into Result. |
| `umpire.semantic-inventory.known-gap-source.21-test-capability` | `Umpire.PlanningTests.KnownGaps` | authored | test-only | exact-known-gap | `umpire.known-gap.capability-contract` | — | Test-only capability-contract Known Gap fixture. |
| `umpire.semantic-inventory.known-gap-source.22-test-input` | `Umpire.PlanningTests.KnownGaps` | authored | test-only | exact-known-gap | `umpire.known-gap.runtime-evidence` | — | Test-only input Known Gap fixture. |
| `umpire.semantic-inventory.known-gap-source.23-test-interpretation` | `Umpire.PlanningTests.KnownGaps` | authored | test-only | exact-known-gap | `umpire.known-gap.runtime-order` | — | Test-only interpretation Known Gap fixture. |
| `umpire.semantic-inventory.known-gap-source.24-test-claim-reference` | `Umpire.PlanningTests.KnownGaps` | carried | test-only | carried-catalog-entry | `umpire.semantic-inventory.known-gap-source.08-promotion` | kind -> kind; code -> code; subject -> subject; detail -> detail | Test-only use of the production planner promotion Known Gap. |
