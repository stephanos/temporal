---
satisfies: [R3, R5]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.3 Capture derivation provenance without changing behavior

## Description
Collect exact derivation provenance and validate inherited/local targets (Scala groundwork for R5). This task does not change Go findings or suppress them.

**Size:** M
**Files:** derivation extraction, gap target validation and fixtures.
**Touches:** [model/irgen/Declarations.scala, model/irgen/Gaps.scala, model/irgen/Context.scala, model/irgen/Lifting.scala, model/irgen/test/Gaps.test.scala, model/irgen/testdata/gaps/**, model/SEMANTICS.md]

### Approach
- Extend the existing derivation extractor before flattening destroys operation history. Preserve chained source origins and every existing drop/replacement/assumption behavior.
- Distinguish a local binding/claim change from inherited origin; refuse only inherited gaps whose relevant subject and context can be proven equivalent. Ambiguity is reported, never silently treated as equivalence.
- Supply task 6 with explicit provenance and conservative eligibility, not an assertion that all untouched bindings have the same reachability or claim judgment.

### Investigation targets
Required:
- `model/irgen/Declarations.scala:1204` — derivation extractor and flattened copies.
- `model/framework/Machine.scala:465` — restrict/drop semantics.
- `model/framework/Facet.scala:11` — facet ownership representation.
- `model/irgen/Capabilities.scala` — capability inheritance and waivers.
- `model/check/Equivalence.scala:369` — existing normalization.

### Verification
Add fixtures for restrict/rebind/extend chains, guard rebinding, monitor/refinement drops, replacement refinements, added assumptions and shared facets. Pin pre/post step tables and existing semantic projection for each operation; negative controls change a guard, outcome or claim context and must fail the pin. Record that consumer suppression is still unconfirmed until task 6's per-kind evidence.

## Acceptance
- [ ] Every existing derivation operation and chained origin is represented and tested.
- [ ] Pre/post step-table and semantic projection pins pass, with behavior mutations rejected.
- [ ] Inherited/local target controls pass conservatively; no Go finding suppression or full R5 claim occurs.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
