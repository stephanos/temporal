---
satisfies: [R2, R3, R4, R5, R6]
---
# fn-123-declare-faults-as-the-environments.10 Native durability, crash derivation and storage-loss assumptions

## Description
Complete native durability/crashes behavior from .2 and the native portions of .4/.5. Original tasks retain IR, Go crash parity and provider migration proofs.

**Size:** M
**Files:** framework DSL Machine/RuleBindings/Assume/Syntax, native tests, ReliableBroadcast classification and local guidance.
**Touches:** [model/framework/dsl/**, model/framework/README.md, model/framework/migrate/README.md, model/framework/samples/ReliableBroadcast.scala]

### Approach
- Classify state fields with durable/ephemeral/inMemory and bind crashes through the existing native rule mechanism. Derive a native crash from the same retained classification, not a second hand-authored semantics.
- Validate full field coverage, whole/inner path overlap, durable fallbacks, duplicate mappings, out-of-domain reset, classification without crash and crash without classification. Budget fields on crashing machines must remain durable. Validate on construction/registration before first evaluation; no dormant declaration bypass.
- Preserve authored-over-derived rebind meaning and storageLoss's ordered one-per-fault derived assumption, refusing hand-authored duplicates.
- Native assuming currently discards its arguments: add ordered assumption storage and propagation through built/derived/restrict/rebind/extend before relying on storageLoss deduplication. Preserve authored-before-derived order and one stable assumption identity for the same fault across machines; no serialized RuleBindings packet may become a native evaluator.
- Follow the parent's Native declaration seams for completed-table/metadata admission timing, typed reset-validity evidence and exact authored-assumption multiplicity. Pin successive authored A,B then C,A as A,B,C,A; only generated storage-loss entries coalesce. Preserve internal fairness duplicates/empty segments and refuse authored/generated collisions. Invalid never-invoked crashes must refuse before executable native acquisition, without base-constructor forcing or state callback execution.
- Implement the parent spec's explicit 2026-10-16 budgeted derived-crash amendment: durability reset preserves every budget field, then the same atomic transition consumes only the firing derived crash's own budget once (`true` to `false`, positive `n` to `n-1`). Exhaustion disables before reset evaluation; other budgets are unchanged. All budget fields remain classified durable. Authored fault effects and authored crash overrides manage their own consumption and must not be automatically decremented again. Unbudgeted crashes stay unchanged; no hidden counter or added state field. Use fn-173 plain-Int/reached-state contracts, not whole-state catalogs; global Go budget enforcement remains held.
- Apply the parent's negative-Int boundary policy only when a generated crash consumes its own budget. Refuse a negative value with fault/binding location before reset/decrement; independently pin negative refusal, zero exhaustion and positive consumption with a poison reset. Do not evaluate model state at registration or add native enforcement wrappers around authored effects/overrides.
- Apply the parent's own-budget alias boundary: reject an active generated crash whose own resolved state field path is also another active fault binding's budget in that machine. Refuse at completed admission before callbacks, distinguish separate bindings from repeated internal rule aggregation, and retain distinct-field positive controls. Authored effects/overrides alone gain no blanket shared-path ban, automatic consumption or global budget enforcement.
- Pin native derived results against independently authored ordinary crash functions on small generic fixtures, including every fallback and rebind case. Do not read historical Temporal providers.
- Add the required durability classification to ReliableBroadcast's existing authored crash without changing its guard, effect, facts, outcome or enabledness. Do not replace that callback with an unconditional generated crash; the broader native sample representation rewrite remains held under fn-173.14.

### Investigation targets
**Required:** model/framework/dsl/Machine.scala; model/framework/dsl/RuleBindings.scala; model/framework/dsl/Assume.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Rules.test.scala; model/framework/dsl/Domain.scala.

### Verification
Run native fault/rule/assumption/refinement tests with nonzero counts. No Go, schema/exporter, regeneration or model/temporal inputs.

## Acceptance
- [ ] Native crash derivation matches independent per-field truth/result pins and supports authored override through rebind.
- [ ] All invalid classification/fallback/crash/budget combinations reject at the owning native declaration boundary.
- [ ] Independent Boolean/numeric pins prove exactly-once own-budget consumption, exhausted disabling before reset evaluation, other-budget/durable-field preservation, unchanged unbudgeted results and no double consumption of authored overrides.
- [ ] A negative plain-Int own budget refuses before reset with a located fault/binding diagnostic; zero and positive paired controls preserve the adopted semantics without state enumeration or registration-time callback execution.
- [ ] Generated own-budget aliasing with another active fault binding refuses before callbacks; distinct fields pass, and an authored override does not retain the generated-only alias restriction or consumption wrapper.
- [ ] Storage-loss assumptions retain deterministic identity/order, coalesce generated entries per fault and reject authored contributions duplicating a generated fault assumption.
- [ ] Authored assumptions retain exact multiplicity/order through copies and fault restriction; native acquisition rejects invalid dormant crash declarations without forcing uninitialized subclass state or evaluating model callbacks.
- [ ] Focused native checks pass; complete R3/R4 Go/provider proofs and all IR obligations remain open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
