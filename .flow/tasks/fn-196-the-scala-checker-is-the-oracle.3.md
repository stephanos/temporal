---
satisfies: [R4]
---
# fn-196-the-scala-checker-is-the-oracle.3 DSL completion: measure the Scala checker and decide the Go deletion and the lowering location

## Description
**DSL completion phase (owner, 2026-10-17).** Measure `verifyAll` on the six current samples at their intended budgets and on the largest practical native graphs available under this phase's holds. Write `model/design/scala-checker-measurement.md` and a reproducible measurement script beside it. Compare with recorded Go measurements where they exist. End with an explicit go/no-go on deleting the Go check, a reasoned choice of Case lowering location (Go over IR or Scala emitting the Case proto), and kept/moved/dropped dispositions for IR-only refinement, rule-disjointness and totals obligations.

**Size:** M

**Touches:** `model/design/scala-checker-measurement*`; conditional `model/framework/checker/Visited.scala` only if measurements justify a collision-safe visited-set change. Other production edits require the conductor's exact scope authorization before editing. Worker lifecycle writes under .flow and task-unique handovers are exempt.

### Measurement contract

- Read `.flow/tmp/fn1963-prep/measurement-sources.md` and `instance-cache.md` as source-backed preparation, not acceptance evidence. Re-anchor their function names and current APIs. The current integrated checker source is `8f84a96e49`, closure `a0cc2feb84`, root milestone `76046961d8`; native baseline663, class-lint7, RuleSuite31, boundary66, discovered suites173.
- Produce an actual per-graph table of reached states, executed steps, verifyAll wall time and live heap, with exact owner, constructor world, starts, budget and verdict coverage. Define every counter and its measurement scope. Distinguish ordinary state exploration from monitor-product/history costs and scoped-start searches. Label graph-statistics runs separately if they are not the same verifyAll invocation. Retain expected refuted and unknown answers; a budget cut is not full exploration.
- State machine/JVM/toolchain versions, JVM heap limits, warm/cold policy, repetitions and GC/retention method. Time checker execution separately from compilation/packaging/startup. Do not call post-return heap after the working graph became collectible the graph's live working set. Do not label allocation bytes, current uncollected heap or RSS as live heap. Separate timing runs from intrusive heap probes if needed. Keep every raw observation and failed attempt under task scratch, with exact terminal commands and a runnable script/note recipe. If a minimal production probe seam is necessary for honest measurement, request its exact checker-only change before editing.
- Compare recorded Go work honestly. The retained fn172 heap snapshots include uncollected garbage, and old Go/default-scope checks and current native verifyAll do not cover identical workloads. Historical catalogs, serialized table lines, enabled steps and reached states are different quantities. No invented speedup ratios, peak/live equivalence or current Go execution is authorized.
- Inspect parked equivalence sources for current-DSL compatibility. Their retained old framework/API is not the current checker. Respect held Temporal bodies; use the task's authorized scaled-sample fallback when current native replay is unavailable, explaining the source-backed mismatch. Start with genuinely larger finite worlds (for example Voting majority quorums on five then seven acceptors), scale safely and record cuts/resource failures. Do not substitute the smaller unanimous world as the largest sample. The exact measured graph, rather than historical catalog size, establishes size.
- Inspect single-live-graph working set separately from repeated fresh-instance retention in one JVM. Global naming/claim/channel caches may retain discarded parametric Models. Do not silently clear product caches or alter semantics to improve the heap number. If retention is material, report measured reachability and the owning lifetime seam to the conductor before a narrow fix; hash-only deduplication is not a substitute for explaining retained owners.
- Build a hash-keyed visited set only if measurement requires it, preserving structural identity under collisions, all concrete states/paths/outcomes and existing budget/cut behavior. Re-measure and test deliberate collisions. No performance optimization is required merely because the task mentions one.

### Decisions and phase boundaries

Give an explicit go/no-go with the measured limits and semantic prerequisites. On no-go, record the reason and the reopened boundary; do not silently declare the Scala replacement complete. The decision does not execute Go deletion or release Phase3.4. Choose the lowering location from actual dependencies: current Go lowering still uses check.Check, Realizer tables and Query.Answer; Scala realization conversion currently emits IR rather than Testpilot Cases. Do not equate removing one check call with removing the interpreter dependency. List the retained/moved/dropped owner for refinement (including replacement compositions), rule disjointness and totals; a stale comment is not proof that a current check exists. Make required future migration work explicit without implementing it here.

Oracle-direction docs belong to .5; ModelSuite authoring/discovery/wiring belongs to fn195. No ModelSuite prototype, new author API, Go/IR/export/lifter/schema/backend/generated change, Temporal migration or actual Go-toolchain run. Root owns MILESTONES. All222 disabled Temporal files, broad GuideSourcesTest and both real-Go sample bodies remain held before execution; no held Make route or Make-green claim.

### Verification and handover

Run an own focused pre-edit checker baseline; no baseline handoff follows the root milestone/scope commits. Exercise the measurement script from its documented instructions and confirm the committed note's numbers against raw output. Format before final exact-byte checks. Use actual gate classification/check/receipts; FULL must not become a docs-only skip. Preserve native663, class7, Rule31, boundary66, source173 and fresh declaration/core/plugin/six-sample realization/public-question source and separate-library proofs when full classification or checker edits require them, using fn1962 final recipes without weakening strict flags or held routes. Every command must reach terminal exit; observation timeout never licenses restart of a live handle.

Use `.flow/tmp/fn1963/{summary.md,evidence.json,final-commands.md}` for handover. The worker owns the only compiler/measurement lane and all scoped staging/commits; helpers may perform disjoint source/report work, with no concurrent heavy JVMs and all writers reconciled/frozen before verification. Remain on umpire, review none by explicit phase-batch deferral, plan-sync false, tracker inactive, parent spec OPEN. No publishing or history rewrite. Return only when scope is complete or genuinely requires external authority, with terminal handles and evidence, not a timeboxed partial result.
## Acceptance
- [ ] Measurement note with per-graph numbers and the Go-deletion go/no-go
- [ ] Lowering location decided with reasons
- [ ] IR-only checks listed as kept, moved or dropped

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
