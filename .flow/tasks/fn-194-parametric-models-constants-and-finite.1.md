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

**Touches:** `model/design/parametric-models.md`, `model/design/parametric-prototype/**`.

**Native checkpoint boundary (conductor, 2026-10-18).** Run against committed fn-190.16 (`d7b02a26b7`, wording `270d8cd03b`, closure `653fa13563`). This task probes the existing DSL before changing it; production framework, sample, checker and lint edits belong to .2/.3. Use the product `framework.checker` and current plain Machine form, not retired RecordMachine/Rules/expect/query/Limits or a replacement explorer. Keep prototype declarations/tests separate so the framework gate does not accidentally discover them. Use the native strict flags and a portable run script. Test two distinct instances in one process with genuinely different supply-dependent reachable graphs, parameter-dependent starts and no shared vars/schema state. Test current naming honestly: holder naming is a requirement to prove or refute, not something to simulate with overridden names and report as automatic support. If an Architecture bullet is unsupported, record observed refusals/behavior and choose a concrete implementation alternative for .2 rather than weakening the spec or changing production here. Realizations may be probed with current instance types. IR/export/Go/Temporal and historical fixtures remain held. The design verdict and prototype paths explicitly named by this task are in scope despite the general framework-only boundary. `make model-check-changed` routes to held Temporal work: record that limitation and use authorized native/prototype commands without claiming Make passed.
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
