---
satisfies: [R1, R2, R3, R4, R6, R7, R8, R9]
---
# fn-161-facets-action-owning-behaviour-areas-of.1 Facet construct: framework, lifter, ownership check, layout rules

## Description
Build the facet construct in the framework and the lifter: a facet contributes `effects`, `rules` and `capabilities` sections, a machine lists its facets, and the lifter lifts the assembled machine as if every section were written in it. Also enforce one owning facet per action, explicit facet dependencies, and the facet layout rules. Include fixtures. Models aren't migrated here; fn-161.2 and fn-161.3 do that.

**Size:** L
**Files:** `model/framework/Syntax.scala` / `Machine.scala` (the facet base, rule merging, ownership check), `model/irgen/Declarations.scala` (reading facet sections), `model/irgen/Order.scala` and `Structure.scala` (layout and order rules), new `model/irgen/testdata` fixtures (lifts and refusals), `model/framework/*.test.scala`, `model/README.md`, `model/SEMANTICS.md`
**Touches:** [model/framework/*.scala, model/irgen/*.scala, model/irgen/testdata/**, model/README.md, model/SEMANTICS.md]

### Approach
- **Encoding:** the facet design sketched in conversation is `trait Pausing extends Facet[State, Outcome, Fact]: this: Dispatch => object effects …; object rules extends FacetRules …; object capabilities …`, mixed into the machine object's parents (`object ActivitySystem extends Machine[…], Phased[…], Dispatch, Pausing, …`). Validate it, or pick a better Scala encoding, and record why. Constraints:
  - DSL helpers (`effect { }`, `choose`, `enters`, the phase accessors, rule cases like `when(...)`) need the machine's types and the given `Owner` / `Phasing`; `Facet` must provide them.
  - Section objects nested in a trait must not clash when several facets each declare `object effects`. Expose them merged (e.g. the machine's `effects` view is the union), or name them per facet. Choose and document.
- **Merging rules:** each action's cases from all facets form one step function. The semantics' existing disjointness rule makes the order irrelevant to meaning. The lowering order is facet order, then written order.
- **Ownership check (R3):** every bound action, including timers, faults and `disabled`, belongs to exactly one facet. Refuse an action with rules in two facets, and one bound nowhere. Do this at machine construction (runtime) and in the lifter.
- **Dependencies (R4):** cross-facet step use is declared in the facet's type (a self-type), so a missing facet is a compile error. Add a fixture showing the error message.
- **Lifter:** today it reads a section's own body statements from the source tree (`Declarations.scala` around lines 198 and 270-274) and ignores parents. Teach it to follow a machine's facet parents to their sources and lift their sections as if written in the machine, in facet order.
- **Layout (R7):** one facet per file, named after it. No effect, rule or capability declaration outside a facet or the machine-wide places (state and phase types, `init`/`end`, projection, refinement, monitors, whole-feature queries). Add refusal fixtures for each.
- **Facets only (R8):** once a machine opts into facets, and globally after fn-161.4, refuse a rule, effect or capability declaration written in a declared `Machine` object outside a facet. Derived machines (`rebind`, `restrict`, …) and compositions are exempt. Single-facet machines are valid. Actions stay in the signature; a facet never declares an action. Provide the global switch so fn-161.4 can turn it on: e.g. the check runs for every declared machine and is skipped for machines in a temporary allow-list that fn-161.4 empties.
- **Facet vocabulary (R9):** each feature file declares its facet names once (e.g. a small enum or a declared list beside the signature; pick the encoding). A level's facet must take a name from its feature's vocabulary, and an action bound at both product and system must be owned by the same-named facet at both. Add refusal fixtures for an unknown facet name and for mismatched owners across levels.
- **Docs (R1, R6):** define facet in the README next to capability (noun vs adjective naming; "a facet declares capabilities"), and give the core form in SEMANTICS.
- **Equivalence (R2):** a fixture machine written both ways (single object vs facets) must lift to the same rows, outcomes and facts. Assert it in the irgen tests.

### Prerequisite
- fn-159.7 (`enters`) landed or designed, since facets assume trivial effects are gone. A working tree that builds the lifter.

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
```
## Acceptance
- [ ] A facet construct exists with `effects`, `rules` and `capabilities` sections; README and SEMANTICS document it with its core form and the facet/capability naming convention.
- [ ] A fixture machine written as one object and as facets lifts to the same rows, outcomes and facts.
- [ ] An action with rules in two facets, or owned by none, is refused at construction and by the lifter, naming the action and facets; a missing facet dependency fails to compile; refusal fixtures cover misplaced declarations.
- [ ] Rules, effects and capabilities outside facets in a declared machine are refused (with the allow-list fn-161.4 empties); each feature declares a facet vocabulary, and unknown facet names or cross-level owner mismatches are refused, with fixtures.
- [ ] `make umpire-check-model` and `make umpire-check-lint` pass with no checked-in model changed.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
