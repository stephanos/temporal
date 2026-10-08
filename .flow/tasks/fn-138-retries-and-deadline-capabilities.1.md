---
satisfies: [R1]
---
# fn-138-retries-and-deadline-capabilities.1 Retries capability and action-filtered transition verification

## Description
Implement R1's companion-defined Retries binding and the narrow reader support for action-filtered transition verify Properties. Prove both on focused fixtures before any production Model adoption. The owner-question proposals are resolved in the parent spec's Decision Context.

**Size:** M
**Files:** `model/temporal/capabilities/Retries.scala` (new) and its focused tests; `model/irgen/Capabilities.scala`; `model/irgen/testdata/lifts/CapabilitySections.scala`, `CapabilitySectionRejects.scala` and scoped `expected/*`; `model/irgen/test/Fixtures.test.scala`; `tools/umpire/check/claims.go`, `checking_test.go` and the existing reader type facade if its key-transition signature changes; `tools/umpire/internal/engine/keyclaims.go`, `keyclaims_test.go`; `tools/umpire/export/quint_test.go` and selector agreement tests; `tools/umpire/ir/framework_test.go`; `model/SEMANTICS.md`.
**Touches:** [model/temporal/capabilities/Retries.scala, model/temporal/capabilities/Retries.test.scala, model/irgen/Capabilities.scala, model/irgen/testdata/lifts/**, model/irgen/test/Fixtures.test.scala, tools/umpire/check/**, tools/umpire/internal/engine/keyclaims.go, tools/umpire/internal/engine/keyclaims_test.go, tools/umpire/export/*test.go, tools/umpire/ir/framework_test.go, model/SEMANTICS.md]
**Source gate:** Start implementation only after fn-128.5 is DONE and integrated, the conductor has persisted the refreshed plan and its independent plan review has passed. This draft does not bypass the gate.
**Batch:** Focused Scala/Go/reader/export fixture proof and scoped lifter fixture golden updates run in this task. Production IR/Cases/mirrors, complete regeneration, full gates and live runs wait for the shared fn-128.6/fn-129.5 boundary. The worker records focused commands and commits its own task; the conductor owns Flow state and boundary evidence.

### Approach

- Follow `model/temporal/capabilities/Closable.scala:8` for a typed capability case class and companion Properties. Bind one failure class with a constant retryable classification; repeated declarations handle another failure class without a decoder framework. Supply named defs for count, finite policy maximum or explicit unlimited, retriesRemaining, and declared optional pending controls. Require Waiting/Failed and only the roles for declared control branches. A missing pause branch must not demand Suspended on Nexus. The Property fixes its target role from the classification/control table, never from an arbitrary expected-landing predicate.
- Use the existing `property when <class> holdsAcross` form (`model/umpire/Claims.scala:43`, `model/irgen/Claims.scala:277`). Preserve class/action selectors through `tools/umpire/check/claims.go:633` into key-level transition checking at `tools/umpire/internal/engine/keyclaims.go:31`. Apply the selector before reading the predicate or marking exercise (`keyclaims.go:124`). Preserve transition-find refusal (`internal/engine/claims.go:133`), unknown/error precedence, limits and witness replay. Keep nonselected predicates unread even when their body would error. Existing unfiltered key-transition callers retain behavior and source compatibility through the smallest private constructor adjustment; no new public DSL or schema.
- Update SEMANTICS Claims and What the reader implements to describe supported action-filtered transition verify and the retained transition-find refusal. Update the exact disk.durableStays fixture expectation in `tools/umpire/check/checking_test.go:302`, which currently pins this new supported form as Unsupported. Export already reads selectors before evaluating `(before, after)` (`tools/umpire/export/slice.go:400`, `quint.go:955`); exercise that path and reader/export agreement, changing export production code only if the focused proof exposes a necessary gap.
- Carry the declaring val through expansion for new Retries/Deadline kinds only. Generate `<machine>.<val>.<property>` and keep companion origin. In `model/irgen/Capabilities.scala:339,737`, resolve a new kind's own parameter against its current declaration before cross-capability lookup. Keep existing kind names/IDs, duplicate and ambiguity behavior. The existing companion-Property bound override applies to all matching instances of the new kind, allowing task .3 to bound only new Queries while preserving the older default. Refuse an ambiguous claim/waiver lookup on repeated new instances rather than selecting the first; this task adds no new instance-selection API. Role ownership is fallback for a Property with no own binding field; own-field Retries must lift without Pollable. Preserve the existing Pausable+Pollable interaction requirement.
- Generate transition claims as free verify Queries even when a when_class selector is present (`Capabilities.scala:775`). Never create a pinned find or Run expectation for them. Tests must prove names, origins, free form, computed totals and bounded Queries, as well as separate fields in repeated declarations.
- Establish a max-one/count-two reachable mutant with representation ceiling two. Check before-state exhaustion on a failure that increments the count, ordinary/pause/cancel priority, fatal failure, and retryable cancellation even when exhausted. Counterexamples must be nonempty and replay through a fresh interpretation.
- Add Retries and subsequently Deadline to the existing framework vocabulary restriction. Keep Temporal kind names out of `model/umpire`. Do not use the framework mechanism to add production feature behavior.

### Investigation targets

**Required:**

- `model/temporal/capabilities/Closable.scala`, `Pausable.scala`, `Pollable.scala` and `model/umpire/Claims.scala` for current witnesses, own roles and selector API.
- `model/irgen/Capabilities.scala:173,339,737` for duplicates, role/field ownership, expansion and generated Query form.
- `tools/umpire/check/claims.go:633` and `tools/umpire/internal/engine/keyclaims.go:31,124` for the refused selector and skipped trigger.
- `tools/umpire/export/slice.go:352,400`, `quint.go:955` and `quint_test.go` for selector and before-state agreement.
- `model/irgen/test/Fixtures.test.scala:329` and `model/irgen/testdata/lifts/CapabilitySectionRejects.scala` for current refusal wiring.

**Optional:**

- `tools/umpire/check/checking_test.go`, `tools/umpire/internal/engine/keyclaims_test.go`, `keylower_test.go` for receipt/exercise/replay and retained lowering refusal patterns.
- Completed post-fn-128.5 `model/temporal/features/activity/standalone/system/System.scala` and Nexus workflow `system/System.scala` for exact supplied defs/classes; re-anchor their final names before implementation.

### Key context

Nexus increments attempts on failure while Activity increments on poll. Eligibility must read before-state. Nexus handlerError classes and network.fault originate in Waiting and declare neither Pollable nor pending pause. Activity fatal failure is Failed even with cancellation pending; retryable cancel settlement is Canceled, including exhaustion. fn-128.3 owns these decisions. An actual conformance failure requires a human under AGENTS.md, not a change to the independent Model.
## Acceptance
- [ ] R1 fixtures lift Retries with own fields, companion origin, `<machine>.<val>.<property>` names, computed totals and free verify Queries for class-filtered transitions. A Waiting-source fixture with no Pollable and no Suspended role passes. Two instances bind different action classes and own field values without ambiguity or leakage. A companion-Property bound override reaches both new instances while the old default remains pinned; ambiguous new-instance claim/waiver lookup refuses instead of choosing the first. Existing capability names/IDs and Pausable+Pollable interaction stay pinned.
- [ ] Reader/engine tests exercise when_class and when_action transitions, preserve the correct before-state, skip unrelated actions without evaluating a failing/erroring predicate or marking exercise, and mark a selected step exercised. A selected predicate error remains a declaration error, and holes/limits retain their existing statuses. Unfiltered transitions retain ordered receipt behavior. Transition find and transition lowering remain refused.
- [ ] The ordinary eligible retry lands in Waiting, pending pause in Suspended, retryable pending cancel in Canceled, and fatal or ordinary/pause exhaustion in Failed. Tests refute wrong targets, wrong control precedence, premature failure and extra retry. Retryable cancellation at exhaustion remains Canceled. The before-state exhaustion fixture distinguishes the count before and after an incrementing failure. Every intended red has a nonempty replayable witness.
- [ ] A reachable finite policy max-one/count-two mutant fails the policy-bound Property despite a counter representation ceiling of two. The valid finite-one/two fixtures pass; explicit unlimited retries at the representation ceiling without claiming a finite policy maximum.
- [ ] Missing Phased, Waiting/Failed or a required declared-control role, foreign/unbound failure classes, malformed/unnamed function bindings and ambiguous foreign fields fail at the declaration with the machine and relevant role/field named. An absent pause/cancel binding requires no absent control role. Old refusal diagnostics stay fixed except the named disk.durableStays selector/transition specimen, whose expected answer must become the independently checked supported result.
- [ ] Go reader/engine/export tests agree on selector-skipped/exercised and before-state results; the existing Quint fixture contains selector-filtered transition claims. Focused backend evaluation is reported when tools are available, with no success-by-skip claim. Scoped Scala/irgen tests pass without unchecked warnings, and the framework kind-name gate lists Retries.
- [ ] SEMANTICS documents the narrow selector-supported transition verify contract and retained transition-find refusal. Focused tests and scoped fixture golden diffs are recorded. Production regeneration, full suites, Cases and live proof remain explicitly deferred to the shared boundary.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
