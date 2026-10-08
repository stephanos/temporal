---
satisfies: [R2]
---
# fn-138-retries-and-deadline-capabilities.2 Deadline capability: kit Properties, per-declaration names and lifter refusals

## Description
Implement R2's Deadline companion Properties, repeated declaration binding and located refusals on fixtures. Reuse task .1's action-filtered transition verification. Task .2 follows .1 because both modify the capability expander and fixture files.

**Size:** M
**Files:** `model/temporal/capabilities/Deadline.scala` (new) and focused tests; `model/irgen/Capabilities.scala`; `model/irgen/testdata/lifts/CapabilitySections.scala`, `CapabilitySectionRejects.scala`, scoped `expected/*`; `model/irgen/test/Fixtures.test.scala`; `tools/umpire/check/checking_test.go`; `tools/umpire/export/quint_test.go`; `tools/umpire/ir/framework_test.go`.
**Touches:** [model/temporal/capabilities/Deadline.scala, model/temporal/capabilities/Deadline.test.scala, model/irgen/Capabilities.scala, model/irgen/testdata/lifts/**, model/irgen/test/Fixtures.test.scala, model/check/test/CapabilityVocabulary.test.scala, tools/umpire/check/*test.go, tools/umpire/export/*test.go, tools/umpire/ir/framework_test.go]
**Source gate:** fn-128.5 DONE and integrated, persisted plan reviewed, and fn-138.1 complete. This scratch preparation grants no implementation start.
**Batch:** Run focused Scala/Go/reader/export proof and regenerate only scoped lifter fixture goldens. Production IR/Cases/mirrors, complete regeneration, full gates and live execution remain at the shared fn-128.6/fn-129.5 boundary. Record focused evidence; the conductor owns Flow and canonical plan writes.

### Approach

- Mirror task .1's typed case class/companion form in `Deadline.scala`. Bind timer class, covered role type and witness, armed predicate and typed terminal timeout fact. Bind explicit retryability, before-state eligibility and any pending control predicates for a retryable timer. Require TimedOut/covered role for every binding and Waiting plus the roles of declared retry-control branches. An omitted pause branch requires no Suspended case.
- Implement an action-filtered holdsAcross window Property. Every firing reads `armed(before)` and covered-role membership of the before-state. Do not key it on a terminal fact, because an eligible retry records no terminal timeout fact. Implement the landing Property against the same selected timer with before-state eligibility and the R2 settlement table. Pending cancel settles TimedOut; eligible ordinary/pause retries are Waiting/Suspended, while nonretryable and exhausted firings are TimedOut.
- Terminal settlement requires the exact typed timeout fact of the binding. Eligible retry forbids the terminal timeout fact family, including a differently typed timeout value. The binding supplies the existing typed fact family selection using current finite fact forms; no new schema, Step field or unconditional attempt-timeout fact is required.
- Retain each val's instance in expansion and own-field/type-parameter resolution. Check at least two Deadlines of different covered roles, armed predicates and typed timeout values. Generate `<machine>.<val>.<property>` names for new kinds only and retain companion origin, free verify form and independent computed totals/bounds. Reuse task .1's bound-override fanout across matching new instances and ambiguous claim/waiver refusal. Keep older kind IDs unchanged.
- Refuse duplicate Deadline terminal timeout types on one machine, with both val names and both declaration positions. Do not restore the old kind-only duplicate refusal for permitted new instances. Refuse a non-Phased machine, missing covered/landing role, foreign/unbound/non-timer class and invalid binding at the relevant declaration.
- Extend the existing refusal harness and framework name list with Deadline. The existing Scala kind-vocabulary gate at `model/check/test/CapabilityVocabulary.test.scala` must list Deadline alongside the Go framework gate; add only the new name and run its focused test. Keep the capability name distinct in documentation from the realization helper `deadlines(...)` and the feature's `deadline` signature object.

### Investigation targets

**Required:**

- Task .1's `model/temporal/capabilities/Retries.scala` and revised `model/irgen/Capabilities.scala` for instance bindings and free transition verify generation.
- Completed post-fn-128.5 Activity `system/System.scala` for start-to-close eligibility, cancellation precedence and statusTimedOut facts; re-anchor the final source before coding.
- `model/temporal/features/nexus/workflow/system/System.scala:212,234` for the existing typed timeout effect and role windows.
- `model/irgen/testdata/lifts/CapabilitySections.scala`, `CapabilitySectionRejects.scala` and `model/irgen/test/Fixtures.test.scala:329` for repeated binding/refusal fixtures.

**Optional:**

- `tools/umpire/check/checking_test.go`, `tools/umpire/export/quint_test.go` for the selector/transition tests task .1 extends.
- `model/temporal/realize/Modules.scala` for existing typed deadline realization expansion; it is a read target, not a requested modification.

### Key context

Activity schedule-to-start's armed predicate includes dispatch now. Start-to-close retries only while policy allows it and cancellation is absent. Its pause retry preserves dispatch backoff. Existing statusTimedOut(type) remains terminal and maps to the TimedOut API status. Nexus deadlines always terminate. Heartbeat integration belongs to fn-129 after it introduces the timer; this task neither adds that timer nor changes its schema.
## Acceptance
- [ ] A fixture with two Deadline vals of different timeout types, covered roles and armed predicates lifts independent own bindings, companion origin, `<machine>.<val>.<property>` names and bounded free verify Queries. Existing capability names/IDs remain unchanged.
- [ ] Every selected firing checks its before-state armed predicate and covered role, including an eligible retry that emits no terminal timeout fact. Mutants firing unarmed, outside the covered role or during a disallowed dispatch window fail with nonempty replayable counterexamples. Unrelated actions do not evaluate or exercise the window Property.
- [ ] Eligible ordinary retry is Waiting and eligible pause retry is Suspended with no terminal timeout fact of any type. Exhaustion, nonretryable deadline and cancellation-pending firing are TimedOut with the binding's exact typed timeout fact. Tests refute wrong retry/terminal targets, cancellation-as-retry, missing/wrong typed terminal facts and a terminal fact emitted on an eligible retry.
- [ ] Refusal fixtures cover missing Phased, covered role, TimedOut, a required retry/control role, foreign/unbound/non-timer classes and malformed bindings. Duplicate timeout-type refusal names both vals and both positions. Different types pass; absent control branches do not require absent control roles.
- [ ] Scoped Scala/irgen and Go reader/export fixture tests pass with no unchecked warning; selector/before-state export agreement is retained. Scoped fixture golden diffs are reviewed, and the framework name gate lists Deadline. Production regeneration, Cases/full gates and live proof remain deferred to the shared boundary.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
