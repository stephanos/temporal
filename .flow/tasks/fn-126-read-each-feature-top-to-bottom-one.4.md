---
satisfies: [R4, R5, R13, R15, R16, R17]
---
# fn-126-read-each-feature-top-to-bottom-one.4 Make the machine object the machine, with effects, rules and sections, on the standalone activity

## Description
Early proof point for the declaration shape. This task reshapes the framework and the lifter once and converts the standalone activity (all three subpackages):
- the machine object is the machine (R15);
- rules say when an action fires, and effects say what it does (R16);
- every kind of member sits in a section (R17, R4 e);
- single-use Scenarios are inlined (R13, activity).

Meaning stays frozen (R5).

**Cross-spec entry gate:**
- Task 3 is done.
- Never alongside fn-124.8.
- Before fn-124.7 (the restructured functions are recorded in its harness).

**Size:** L
**Files:**
- `model/umpire/{Machine,Compose,Claims,Capabilities,Syntax}.scala`: the `Machine`, `Derived` and `Composition` object forms, `Rules`, `Syncs`, the rule sugar with `Core form:` docs, the disjointness check, and the inherited `property`, `scenario` and `capabilities(limits)` members;
- `model/irgen/{Declarations,Compositions,Claims,Syntax,Lift}.scala` and fixtures;
- `model/check/{SyntaxRule,Gate}.scala` and the R4 lint;
- `model/temporal/features/standaloneactivity/**`;
- `tools/umpire/internal/golden/{config,original}.json`.

**Touches:** [model/umpire/**, model/irgen/**, model/check/**, model/temporal/features/standaloneactivity/**, tools/umpire/internal/golden/**, model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**]

### Approach
- **Object forms (R15).**
  - `abstract class Machine[S, O, F]` with `start`, `end(s)`, optional `entity`, `refines`, `visible`, `unobservable` and `evidence`, and the inherited claim builders.
  - `Derived(base-derivation)`, where the derivation is today's `rebind`/`extend`/`restrict`/`refining`/`assuming`/`unmonitored`/`withMember` expression.
  - `Composition[S](members*)` with `object syncs extends Syncs`.
  - Names come from the object, with its first letter lowered (R7).
  - Pick the runtime wiring for sections without reflection (an object cannot be overridden, and nested objects initialize lazily), and prove it in munit.
  - The builder forms stay until task 5 retires them.
- **Rules (R16).**
  - Implement `Rules` and `Rules(phase)` with `when(g) { … }`, `in(p*) { … }` and `disabled(action*)`.
  - Rule-form arguments for `rebind`/`extend`, and the bare-binding `rebind(action ~> e)` that keeps guards.
  - Lower each action's rules in order to one step function `if g1 then e1 else … else Nil`, named `<object>.rules.<action>`.
  - The disjointness check runs at construction, exhaustively over the `Finite` state type and input values, and names both rules and a witness state.
  - The model gate initializes every IR-file root, so an overlap fails the gate.
  - Paired fixtures (rules / hand-written core step) lift to identical tables.
- **Lifter.** Read the object forms and their sections once. Refusal fixtures:
  - `start` or `end` missing;
  - `rules` missing;
  - a section outside a machine, composition or file top level;
  - a nested section;
  - an effect outside `effects`;
  - `disabled`/`Nil` in an effect;
  - `in` without a phase projection;
  - an unbound action class in a rule;
  - a bare binding in `extend`;
  - colliding object names;
  - an overlap (a munit or gate fixture).
- **Lint (R17, R4 e).** Section membership and order; no hand-written `action ~> step` in a Model.
- **Convert the standalone activity**, following the spec's API sketch with today's names:
  - effects named for what they do;
  - positive rules;
  - `disabled(process.workerStop)` on the product;
  - `properties`, `laws` (with `protocolCapabilities` reading the realization), `queries` with single-use Scenarios inlined (names kept);
  - `ActivityWorker`, `StandaloneActivity`, and the record designs and compositions as `Derived`/`Composition` objects;
  - `object Files` for the roots.

  The seven server-rejected pause/unpause pairs keep their reasons in the accepted `silent-rejection` findings.
- **Regenerate.** Prove by the reader's projection that tables, refinement rows, answers, lint findings and Contracts are unchanged. Record the function names and bodies (re-capturing `original.json` if the projection cannot express them).

### Investigation targets
**Required:**
- `model/umpire/Machine.scala` (whole), `model/umpire/Compose.scala:1-120`, `model/umpire/Claims.scala:1-160`
- `model/irgen/Declarations.scala:140-200` (today's machine `Block` reading), `model/irgen/Compositions.scala`
- `model/check/SyntaxRule.scala`
- `.flow/tasks/fn-112-make-the-standalone-activity-scala.6.md` (explicit arms), `model/ir/activity.lint.json` (silent-rejection reasons)
**Optional:**
- `.plans/DSL_SIMPLIFICATION.md` section 2 (lifter costs)

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

### Execution constraints
- Stop before task 5 if the activity's tables, refinement rows or Query answers move, or if the disjointness check cannot be made exhaustive in the gate.

## Acceptance
- [ ] `Machine`, `Derived` and `Composition` object forms, `Section`-based sections, `Rules` (with `when`, `in`, `disabled`) and `Syncs` exist. Each rule heading has a `Core form:` doc, and the paired fixtures lift to identical tables.
- [ ] The lifter reads the object forms and refuses each R15/R16/R17 case at its line, with a fixture per case. The disjointness check refuses an overlap naming the machine, the action class, both rules and a witness state, and the gate runs it over every root.
- [ ] The standalone activity (all three subpackages) is written in the object forms. No effect returns `disabled`/`Nil`, no inverted guard remains, single-use Scenarios are inlined with their names, and capabilities sit in `laws`.
- [ ] Tables, refinement rows, IDs, names, answers, lint findings and Contracts are unchanged. The function names and bodies are recorded as R5 deltas.
- [ ] All gates of the spec's Verification pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
