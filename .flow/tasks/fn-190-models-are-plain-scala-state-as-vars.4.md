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

The original native requirements remain open: public legacy vocabulary is still reachable through
record-state compatibility; safety claim blocks do not yet carry their own `in` environment; and
naked Scala guards/helper vals after a failed check still require a whole-member execution boundary.
Actually evaluated claim holes currently propagate instead of producing an unknown result. The
native scenario class-selection and pinned-path budget/completion behavior also need reconciliation.
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
