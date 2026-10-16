---
satisfies: [R1]
---
# fn-190-models-are-plain-scala-state-as-vars.1 Prototype the plain-Scala form on Kettle and record the verdict

Touches: model/design/plain-scala-models-prototype.md, .flow/tmp/fn190-prototype (README.md, run.sh)

## Description
Throwaway prototype, no production source changed. On a copy of Kettle: state as vars on the Machine, effects as assignment blocks (including `+=`, an `if` arm and a `choose` with blocks), guards and expectations reading vars by name, `before(x)` in a transition property, `UpTo[140]` as a transparent alias of Int whose bound the plugin reads from the var's type tree, `scenario.starts(temperature = -40)`, the four properties as check blocks under `invariant:`/`transition:`/`after(action):` with `before` as the State record and bare names as the state now, the four claims as members of `invariants`/`postconditions` with no kind word, a `boils` progress claim (`leadsTo(from=, to=, within=)`) checked by `verify` in a `scenario.only(heater.heat, heater.failover)`, a `oneOf(-40 to 40)` start lifted to the IR start list, no `queries` section (every claim checked by default, `in only(...)` on the progress claim), the negative control as `assertRefuted` in the sample test, `object heater extends Internal` with `step` and `action` without `this`. Also try, and report on without committing to: `object transitions:` without `extends Transitions` with facet order from the Machine's `extends` clause (on the Facets fixture), the sample as top-level definitions under `package samples.kettle` with the Machine named `Kettle`, and outcomes/facts as member types with defaults so the plain case is `extends Machine`, the four transitions (`object transitions`, renamed from `rules`) as `if g then effects.e()` with the effects named in `effects` and one rule as a `match` on a phase var exercising `case a | b if g`, `case _: Role`, `reject(r)` and `case _ => ()`. A stand-in plugin phase (prototype-local, like the fn-174 plugin prototype) lifts these to the existing `field`/`copy` IR messages and prints them; compare with the retained Kettle IR for the same effects and predicates. Measure and record: sequential-read substitution (`a = 1; b = a`), the bound surviving the alias, the synthesised record's name and field order, whether a var read outside a lifted scope can be refused at the read, that an `if`/`match` rule chain lifts to the same arms as the `when`/`where` form (first match as nested `if`), and that `before.x` lifts to the transition's first parameter and a bare name to the second with per-`check` diagnostics unchanged. Write the verdict and the chosen alternative for anything that failed in model/design, keep the probe with a README and run.sh.

**Size:** M

### Verification
- run.sh reruns the probe and prints the lifted IR; the design note names the logs.
- The note's verdict table covers every item above, including the property form.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
