---
satisfies: [R17]
---
# fn-133-lean-typed-realizations.8 Derive per-class carrier schemas from typed realizations

## Description
Replace the unpaired action-level .schema[T] lists with carrier mappings derived from typed realization bindings. The authoritative association is perform(actionClass -> instruction): for example control(pause) -> pauseActivity -> METHOD_PAUSE_ACTIVITY_EXECUTION -> PauseActivityExecutionRequest. Derive metadata per realization and concrete action class, preserving the binding's pattern semantics and supporting withFields variants, derived realizations, multiple carriers, protobuf command payloads, and typed worker/handler messages. Actions without a realization have no inferred carrier: keep model declarations independent and let R6 report missing realizations where required. Retire the duplicate .schema authoring form and teach the mapping in README. Model and Testpilot behavior remains unchanged; record deliberate Umpire metadata changes.

## Acceptance
- [ ] The four standalone activity control classes expose separate derived mappings to PauseActivityExecutionRequest, UnpauseActivityExecutionRequest, RequestCancelActivityExecutionRequest, and TerminateActivityExecutionRequest; order in an old schema list never determines association.
- [ ] RPC request carriers come from generated method descriptors. Non-RPC typed command payloads and worker/handler protobuf carriers are covered by the same mechanism; internal/timer/fault steps do not receive invented carrier schemas.
- [ ] Mappings are scoped to realization and concrete action class. Partial/exact patterns, combined inputs, withFields variants, multiple instructions, and derived binding additions/replacements retain their actual association. Auxiliary onPath evidence/read calls do not become the performed action's carrier.
- [ ] Unrealized classes remain explicitly unmapped, and R6 distinguishes absent realization coverage from an internal action that needs no transport. No additional schema declaration is required to author an abstract Model.
- [ ] Remove live .schema[T] declarations and retire their DSL/lifter form with a clear migration diagnostic. Update affected IR metadata, fixtures, consumers, and README; preserve any aggregate compatibility view only if derived from the per-class mapping.
- [ ] Tests pin the concrete class-to-method-to-message mapping, non-RPC carriers, partial/combined class coverage, derived overrides, auxiliary calls, and unmapped cases. Negative fixtures cover ambiguous or unsupported derivation without fabricating a semantic correctness guarantee from message names.
- [ ] Projection evidence accounts for intentional metadata and source-position deltas and proves unchanged machine tables, Query answers, existing Case execution, and verdicts. Run focused checks during implementation and the required full gates once at the fn-133 batch boundary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
