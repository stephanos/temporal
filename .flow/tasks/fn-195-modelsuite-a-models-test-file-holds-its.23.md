---
satisfies: [R4, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.23 Native explicit conformance replay: selected-class coverage and complete verified eligibility

## Description
Integrate checked explicit conformance admission into the existing native authored replay path. This is partial R4/R7 coverage and consumes the preceding exact-binding/expectation/monitor admission task. Serialize after committed `.21` because its registration/runtime adapter seam is active work. Do not substitute an uncommitted `.21` result or a feasibility report for a prerequisite.

### Files and Touches

**Size:** M
**Touches:** [model/framework/checker/Replay.scala, model/framework/checker/SuiteExecution.scala, model/framework/dsl/NativeRuntime.scala, model/framework/checker/SuiteExecution.test.scala, model/framework/dsl/NativeConformance.scala]

Exactly five physical files:

- `model/framework/checker/Replay.scala` — optional internal selected-class/position admission callback at actual selection.
- `model/framework/checker/SuiteExecution.scala` — conformance request admission, existing replay integration and result publication.
- `model/framework/dsl/NativeRuntime.scala` — declaration-owned native execution operation/result seam.
- `model/framework/checker/SuiteExecution.test.scala` — real adapter, callbacks, completion, budget and evidence controls.
- `model/framework/dsl/NativeConformance.scala` — extend the preceding request carrier with opaque verified-result ownership.

`checker/Native.scala` needs no edit: its existing direct `SuiteExecution.forSuite` delegation already returns the exact NativeSuiteExecution interface. Keep existing replay callers compatible with the default/no-op callback route.

### Implementation contract

Add `NativeSuiteExecution.conformance(request)` returning the DSL-owned `NativeConformanceCheck(check: SuiteCheck, eligible: Option[EligibleConformance])`. `EligibleConformance` is opaque to authors with a framework-private construction boundary and no public copy/Boolean-forging route. Its authority binds the exact admitted request and complete verified native result. Ordinary native reporting continues to use `check.evidence`; a ticket is not an observed server verdict or a Case.

At actual execution admission invoke `ConformanceAdmission.admit(request)` before replay can fail early, retaining exact owner, checked coverage, full expectation and canonical monitor validation. Use the exact suite's one resolved immutable execution budget and the existing SuiteExecution-to-Replay adapter. Add an optional internal admission callback for the actual selected class and source position at Replay's selection boundary before `engine.steps`. Validate against the selected checked coverage there. Inputs/targets run exactly once per restored frontier as the existing replay semantics require. Do not prewalk, run a second replay, execute deferred inputs during capture, or evaluate callbacks merely to discover classes on paths that already failed.

Issue eligibility only after complete verified execution of the whole finite authored family, with every actual selected class admitted. A successful prefix is insufficient. Any disabled alternative, thrown effect, failed positioned expectation, unsupported selected class, hole, unknown or budget cut yields no eligible ticket. Preserve decisive failure evidence found before a cut. Every finite alternative/repetition/continuation reached during a completed successful family is checked; failure or incompleteness requires no speculative execution of unreached callbacks and earns no eligibility.

Publication belongs to actual execution attempts, not test enumeration. Before a new execution attempt invalidate that declaration's previously published eligibility; publish a fresh ticket only if the new attempt completes and verifies. Failed/cut/reentrant attempts leave the current selector publication empty, and repeat execution never accumulates duplicate tickets. Ordinary repeated suite/test enumeration preserves idempotent registration and does not invalidate a successful result or execute a body. Keep request/result graphs owner-local and collectible, with no global strong registry.

### Verification and handoff

Required investigation targets: the five declared files plus `model/framework/checker/Native.scala` and `model/framework/checker/Checker.test.scala`. Preserve the existing direct native adapter forwarding and state-restored replay selection seam.

Baseline the reached current Replay/SuiteExecution/Checker controls, then verify actual conformance execution through the real NativeSuiteExecution adapter. Include covered and uncovered finite alternatives, a later uncovered branch, repeated/composed classes, failure before an unreached continuation, budget cut before family completion, restored-state input counters, expectation positions, foreign owner/monitor prevalidation, reentry and repeat-attempt publication. Verify every old replay caller retains its existing behavior through the absent callback path. Exercise immutable exact-suite budget, state restoration, disabled/effect/hole evidence, legitimate live results and released request/result ownership.

Record exact current native commands, terminal exits, nonzero counts and immutable source/artifact bindings under task-owned scratch paths. Keep declaration compilation/loading checker-independent and preserve unavailable-checker refusal. The public selector/capture task consumes only the opaque result ticket; sample/capability paths and actual Registry integration remain blocked `.11` obligations. Preserve all historical evidence, old dependencies and `.3` retirement ordering. Actual repository Make, Go, Temporal reads, export/lifting/IR/schema/backend/generated/regeneration, server/recorded-Run/Case operations and publishing remain held.

## Acceptance
- [ ] Actual conformance execution admits the exact request before early replay failure and validates each actual selected class/position through `.17` coverage at Replay's selection boundary before engine steps. Complete finite alternatives, repetitions and composed classes are covered; unsupported/foreign/malformed selected classes refuse. Existing callers use an unchanged no-op path. No prewalk, second replay or extra target/input callback evaluation is introduced; restored-frontier counters prove the existing evaluation count/order.
- [ ] NativeSuiteExecution.conformance uses the real SuiteExecution adapter and the exact suite's immutable resolved budget. Complete verified execution of the whole finite authored family is necessary for an opaque eligibility ticket; covered prefixes, later uncovered alternatives, unreachable continuations after failure, disabled/effect/assertion failures, holes, unknowns and cuts produce none. Preserve positioned conditions/state, decisive pre-cut evidence and ordinary `check.evidence` reporting without executing unreached callbacks for coverage.
- [ ] DSL-owned NativeConformanceCheck carries SuiteCheck plus optional opaque eligibility; authors cannot construct/copy a ticket or forge eligibility from raw metadata/Booleans. A new actual execution attempt invalidates prior publication and publishes only on complete verified success; failure/cut/reentry leaves it empty and repeated attempts do not accumulate tickets. Repeated enumeration preserves idempotent registration and successful publication without running bodies. Result ownership remains local and releases after legitimate retained results are dropped.
- [ ] Reached current replay/checker baselines and new real-adapter callback/budget/evidence/eligibility/lifetime controls execute nonzero counts with terminal receipts and immutable bindings. Checker-independent declaration loading and missing-runtime refusals remain valid. The five-file scope includes the carrier and uses Native.scala's existing direct delegation. Sample/capability/Registry/full-R4 claims, `.11`'s old holds and early bridge retirement remain excluded from this partial handoff.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: