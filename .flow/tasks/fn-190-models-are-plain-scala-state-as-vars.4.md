---
satisfies: [R3, R4]
---
# fn-190-models-are-plain-scala-state-as-vars.4 Effects as assignment blocks, transitions as if/match, reads by name, claims as check blocks: lifter rules and refusals

Touches: model/framework/reify/body/Expressions.scala, BodyEngine.scala, BodySyntax.scala, model/framework/dsl/Syntax.scala, model/framework/dsl/Machine.scala, model/framework/dsl/ExpectationBlock.scala

## Description
**Phase 1 (DSL only), owner phasing 2026-10-17.** Change model/framework/dsl, realize and samples only; the gate is the native checks (dsl/realize/samples compile, their native suites pass, block-form lint). The plugin, lifter, export and IR are untouched: the author surface and native semantics: assignment effects, transitions as `if`/`match`, reads by name, claims as `check` blocks under `invariant:`/`transition:`/`after(action):`, the new sections and Scenario builders, retirement of `property`/`expect`/`across`/`after.state`/`where`/`~>`/`when`/`always`/`enters`/`query`/`Limits` from the public DSL, and compile-time refusals with the spec's texts. The lifter rules and lift-time refusals move to fn-190.9. This supersedes fn-164's remaining public `expect` surface.

The lifter turns an effect body into one `copy` per path with sequential substitution; handles `+=`, `if`/`match` arms, `choose` arms as blocks, `reject`, `records`, `crash`; an input is a parameter. A var read lifts to `field(s, name)` in guards, properties, monitors, refinement and evidence; `before(x)` reads the transition's first parameter. Properties are blocks of `check` statements under `invariant:`, `transition:` and `after(action):`, taking the block by name; bare names read the state now and `before` is the State record of the state a transition started from; `check`, `oneOf`, `branch`, `records` and holes keep today's lifting and diagnostics; retire `property`, `expect`, `across` and `after.state` from the public DSL; a query names its `limits` as today. Claims are members of `invariants` and `postconditions` with no kind word (`after(action):` the only word in a postcondition); `progress` holds `leadsTo(from=, to=, within=)` named after its val; the Scenario builder gains `only(actions)` and `fair(actions)`; a progress claim is checked by `query verify` with the four answers and the cycle report shape in the spec; retire `assume`, `Assumption`, the assumption vararg, the name string and `restrict` as the progress idiom; a realization naming a query over an `only`/`fair` scenario is refused. Claims checked by default from every start, `in scenario` on a claim with inline `only`/`fair`/`starts`, `oneOf(domain)` as a var initialiser lifted to the start list, `queries` kept only for witnesses, pinned paths and realization checks (`verify`/`find`), negative controls as tests (`assertRefuted`); retire `query`, `limits` and `Limits` from the DSL; the Go budget argument, the IR bounds-field removal and the result shape are fn-190.7's. Actions are declared in their owner: `action` in an `Actor` object, `step` in an `Internal` object; retire `internal` and `action(this)`; refuse misplacement with the spec's text. The section `rules` is renamed `transitions` (marker trait `Transitions`; the structure and order lints and SEMANTICS follow); transitions are plain control flow: an `on(action)` block is an `if`/`match` chain whose leaves are a call to a named effect, `reject(r)` or `()`, never an assignment (refuse with the spec's text); `case a | b`, `case a if g`, `case _: Role`, `case _ => ()`; an input is the block's parameter; first match lifts as nested `if`; retire `where`, `~>`, `when`, `always`, `rejects` and `enters`; keep `disabled`; the overlap lint becomes a warning. `enter`, `after.state` and the `s =>` forms are removed. Refuse an assignment to a state var outside an effect, a read outside a lifted scope, an empty property block, `before` outside a transition and a rule leaf of the wrong shape with the exact texts in the spec. Helpers that assign are effect fragments lifted inline; helpers that only read are functions.

**Size:** L

### Verification
- Fixtures for each construct and the three refusals; the Kettle and Bank effects and properties lift to the retained copy/field/predicate shapes except positions.
- A failing `check` in a block is reported on its own, as today.
- Kettle `boils` in `heatingAlone` verifies with longest path 81; the same claim in `fromRoomTemperature` reports the press/depress cycle naming `heat`.
- The retry-facet phase table (as `transitions`) lifts to the same arms as its `when` form except positions.
- Focused dsl and reify suites pass.
## Execution status

The native implementation includes assignment effects, plain transitions, section claims, progress
environments and complete `oneOf(domain)` start products. Assertion and alternative evaluation now
skip after a failed requirement, including fact arguments; a reached fact argument evaluates once.
All five short-circuit regressions pass in the 538-test native run; the lazy fact-argument correction
is committed as `36f548713d`. This is a source checkpoint, not a task closure.

Pinned queries now share a distinct-state budget across starts and schedule layers, keep matches
on their own paths and require a complete schedule before finding a witness. Terminal witnesses
and counterexamples stand at a budget cut; an unproved cut or reached effect hole answers unknown.
The eight pinned regressions pass in the 488-test DSL/realize run, and all 29 source-lint tests pass.
Scoped formatting and a fresh read-only review also pass. The broader sample run was stopped during
Bank graph evaluation and is not counted as passing; its log is
`.flow/tmp/pinned-queries-native-gate.log`.

Native scenario selection now preserves exact input classes: `only(a(input))` filters before
effects execute, while `only(a)` keeps every class; `fair(a(input))` applies only to that class,
while `fair(a)` still applies separately to every class. The eight scenario regressions reproduce
four failures before the fix and pass after it. Final verification passes 510 DSL/realize tests,
42 tests in eight selected sample suites, 29 source-lint tests and scoped formatting. A fresh-context
independent review (same model family) found no material issue. Logs are
`.flow/tmp/scenario-classes-{red,green,core-gate,samples-gate,lint-gate,fmt-check}.log`.
The expanded Bank exhaustive proof remains unclaimed; user Bank and negative-control edits were
left untouched. The native task remains open for the requirements below.

Native safety claim holes now retain unknown evidence per member instead of escaping. Invariant,
postcondition and ad-hoc checks distinguish missing evidence from a known failure; `oneOf` can
still establish a known successful alternative, including after an unknown branch guard. Free and
pinned searches cannot turn unknown predicates into witnesses. A pinned counterexample survives a
later hole, while a prior match still requires a complete schedule. Native diagnostics resolve
declared hole names. An unscoped section abort remains conservative until the whole-member boundary
is implemented.

Twelve focused hole regressions pass in the final 522-test DSL/realize run. The initial baseline
reproduced eight failures with one positive control; the branch-guard regression reproduced three
failures before scoped recovery. All eight selected sample suites pass 42 tests, source lint passes
29 tests, and scoped formatting passes. Fresh-context same-family read-only review found no material
in-scope issue. Logs: `.flow/tmp/claim-holes-{red,guard-red,core-gate,samples-gate,lint-gate,fmt-check}.log`.
The whole-member naked-guard boundary remains open.
The expanded Bank exhaustive proof and all held IR/Go/Temporal gates remain unclaimed.

Native progress and monitor predicate holes now produce contextual unknown results. Progress
reads predicates while discovering states, so an initial missing predicate executes no effects;
consumed searches restart for later checks. Monitor holes stop the affected product path while
other branches can still establish a violation. Ordinary callback exceptions still propagate.
Unknown outgoing classes cannot establish a deadlock or prove a fair class disabled; a known
non-progress cycle still establishes a missed finite deadline when fairness is unknown. Progress
reports retain the missing row and hole, and the cycle diagnostic avoids claiming known fairness.

All 13 progress/monitor hole regressions pass in the final 535-test DSL/realize run. The baseline
reproduced ten failures with three positive controls; the initial-effect and diagnostic regression
reproduced three further failures before the final change. Eight selected sample suites pass 42
tests, source lint passes 29 tests, scoped formatting passes, and a fresh-context same-family
read-only review and re-review found no remaining material issue. Logs are
`.flow/tmp/progress-monitor-holes-{red,early-red,core-gate,samples-gate,lint-gate,fmt-check}.log`.
The expanded Bank exhaustive proof and all held IR/Go/Temporal gates remain unclaimed.

Safety claims now carry their own inline or named `in` environment. Default checking and explicit
`verify`/`find` inherit it, while an explicit query environment overrides it. Exact input-class
selection, starts, independent unscoped claims, derived bindings, predicate holes and caller budgets
remain intact. An all-scoped safety section runs no unrestricted search. Native section reloading
preserves checks preceding the scoped tail; unsupported instance sections refuse evaluation.

Pinned invariants now check initial states, including an empty schedule. Default safety reports
retain postcondition vacuity for free and pinned environments, including empty and disabled paths;
zero-step reports warn without evaluating a postcondition on a fabricated step. Warnings remain
suppressed for incomplete exploration.

All 23 safety-environment controls pass in the 558-test DSL/realize run. The initial surface test
reproduced accidental Boolean membership; the first runtime run reproduced eight failures. Review
then reproduced three pinned initial-state/vacuity failures and three zero-step warning failures.
Eight selected sample suites pass 42 tests, source lint passes 29 tests, and scoped formatting
passes. A fresh-context same-family read-only review and re-review found no remaining material
issue. Logs: `.flow/tmp/safety-environments-{surface-red,red,pinned-red,empty-red,core-gate,samples-gate,lint-gate,fmt-check}.log`.
The expanded Bank exhaustive proof and all held IR/Go/Temporal gates remain unclaimed.

Whole Unit-member execution now stops after a false or unknown requirement, including naked guards
and helper vals. Later members continue independently; branches retain their own stop boundaries,
and a hole inside an `after` block stops its containing member. Ordinary helper exceptions still
propagate. Declaration copies skip Unit bodies, and static named scoped query targets resolve
through those copies while a same-section target keeps direct access to its initialized value.

The isolated native class-file transformer uses getter source lines and full-method local scopes
to locate initializers. Mutable field writes do not split members; local name shadowing, forward
field writes and instance-section discovery have regression controls. Ambiguous same-line ranges
refuse with a declaration diagnostic, including helpers optimized to local slots. Scope comparison
uses actual bytecode length, so generated scenario locals retain their ordinary lifetimes.
`Hole.reached[A]` preserves subsequent bytecode while still stopping on a reached hole; its new
typed-call lifting obligation is explicitly retained in .9.

All 24 short-circuit/declaration controls pass in the 636-test native run. Fresh-context same-family
read-only review and subsequent re-review found no remaining material issue in this checkpoint.
The final native log is `.flow/tmp/whole-member-boundary-code-scope-native-gate.log`; failing
controls and intermediate verification remain under `.flow/tmp/whole-member-boundary-*`.
Discovery and source lint pass 37 checks (154 production files, no findings), and formatting passes
all seven changed Scala files. Logs are `.flow/tmp/whole-member-boundary-check-gate.log` and
`.flow/tmp/whole-member-boundary-fmt-check.log`.

The native task remains open: public legacy vocabulary is still reachable through record-state
compatibility, and naked helpers before a scoped claim's final `in` expression can still execute
during metadata discovery. Parenthesizing the whole scoped block already defers it, but that does
not establish the unparenthesized form. A macro probe can see the enclosing val's complete RHS
after typing, but early type completion can expose a val tree without its RHS. The follow-up probe
therefore captured no prefix (`.flow/tmp/claim-whole-scope-probe.log`); reliable full-member capture
remains unproven and unimplemented. No production change followed this probe.
Plugin/lifter semantics remain in .9 and the engine/IR proofs remain in .10. No held gate is counted
as passing.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
