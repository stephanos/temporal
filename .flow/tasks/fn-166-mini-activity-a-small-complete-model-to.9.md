---
satisfies: [R6, R7]
---
# fn-166-mini-activity-a-small-complete-model-to.9 Mini-guided author documentation and migration convention

## Description
Completes R7 and supplies R6's rule source and isolated mini proof. Integration into the executable Make/Scalafix route remains in held task 10.

**Size:** M
**Files:** shared custom-rule source and focused fixtures under `model/migrate/`, `model/README.md`, `model/SEMANTICS.md`, convention sentence in open Scala DSL batch specs.
**Touches:** [model/migrate/**, model/README.md, model/SEMANTICS.md, .flow/specs/*.md]

### Approach
- Replace the hypothetical job in `model/SEMANTICS.md:13` with the complete committed mini feature from task 8. Cite actual files and source declarations for each example; do not invent a declaration for a guide snippet.
- Update README Writing a Model with the complete three-level mini feature and its named hard instances.
- Establish one shared custom-rule source location/naming under `model/migrate/` and author the worked rename as an actual Scalafix rule with isolated compile/transformation fixtures. Reuse the repository's existing Scalafix dependency; inspect its API before coding. A direct in-process source test is source proof, not proof of the production loader. Do not add a competing general-purpose rewrite engine or repeat fn-167.4's gap transformation.
- Select a concrete rename that preserves semantic declaration identities and source positions so the specified byte-identical lift is demonstrable. A rule that renames an exported definition cannot silently substitute an expected identity change for that proof.
- Record the mini-first migration policy and the actual source-test route. `Makefile:689` currently loads built-ins only; its custom-rule loading/configuration integration belongs to task 10.
- Carry the required convention sentence into each open DSL spec through Flow setters. Preserve unrelated owner edits and completed histories. No Make/config/production-loader change or real-feature edit belongs here.

### Verification
Check cited paths/declarations, run the mini-focused compile covering them, and test the actual rule source against isolated copied-mini input with exact lift comparison and rename/refusal controls. Report commands requiring held infrastructure rather than claim they passed. R6's executable-route integration remains incomplete until task 10.

## Acceptance
- [ ] SEMANTICS uses the committed mini feature as its running example and every snippet cites actual source; README names its complete hard-instance inventory.
- [ ] One shared migration source location/naming and mini-first convention is documented consistently in README and every open Scala DSL batch spec.
- [ ] The worked example is actual compiled rule source with isolated copied-mini transformation and the required byte-identical lift proof; no alternative migration engine is introduced.
- [ ] Docs state current versus held behavior accurately; no production-loader, real-feature migration, full-set or review credit is claimed.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
