---
satisfies: [R1, R2, R3]
---
# fn-194-parametric-models-constants-and-finite.1 DSL completion: prototype a parametric Bank and record the verdict

## Description
**DSL completion phase (owner, 2026-10-17).** Before changing the DSL, try the class form on Bank in a prototype: `class Model[Addr: Finite](holder: Addr, supply: Int) extends Machine[Outcome, Nothing]` with `Addr` declared as an enum in the test file and the instance built there. Prove or refute each Architecture bullet of the spec on it: var initialisers and `oneOf` starts reading the parameters, the `VarSchema` given and the var runtime being per instance (two instances in one process, no shared state), naming an instance by its holder, and what the structure lint needs to admit the class form. Record the verdict and the chosen spelling for each bullet in `model/design/parametric-models.md`, keeping the prototype files beside a README and run script as the design-verdicts convention asks.

**Size:** M

**Files:** `model/design/parametric-models.md`, a prototype under `model/design/parametric-prototype/` (Bank copy, its test, README, run.sh).

### Verification
- The prototype's test suite passes on the native runtime with two Bank instances of different `supply` giving different reachable graphs.
- The note states a verdict per Architecture bullet and names the alternative where one does not hold.
- Quick: `cd model/framework && mise exec -- scala-cli test .` scoped to the prototype suite (the README Native checks recipe is the full gate).

## Acceptance
- [ ] Verdict note in `model/design` with one verdict per Architecture bullet
- [ ] Prototype kept with README and run script; two instances checked in one process without shared state
- [ ] Instance naming by holder demonstrated or an alternative named

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
