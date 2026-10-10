---
satisfies: [R1, R4, R5, R6]
---
# fn-153-checked-property-examples-pilot.5 Author the three contrast pairs and prove classification drift controls

## Description
Attach the three selected Properties' concrete contrast pairs. Connect their generated explanations to production classifier mutation controls and preserve their behavioral definitions.

**Size:** M
**Files:** `model/temporal/features/nexus/workflow/system/System.scala`, `model/temporal/features/nexus/standalone/system/System.scala`, `tools/umpire/check/illustrations_test.go`, `model/docs/property-illustrations/README.md` (new), generated IR metadata at the pilot regeneration
**Touches:** [model/temporal/features/nexus/workflow/system/System.scala, model/temporal/features/nexus/standalone/system/System.scala, tools/umpire/check/illustrations_test.go, model/docs/property-illustrations/README.md, model/ir/**]

### Approach
- Re-anchor `syncSucceeds`, `completionSucceeds` and the expanded `closedRejectsOrRepeats` Property against the integrated delivery-chain baseline. Preserve original predicate/function bodies and resolved ownership. If claim grouping/export moved their declarations, update task Touches through the conductor before edits.
- For synchronous success (`workflow/system/System.scala:274`), supply success with its completion fact, independently remove the fact, and separately change the resulting phase while retaining the fact. Explain each distinguishing detail rather than duplicating the predicate in prose.
- For the existing closed transition (`standalone/system/System.scala:150`), contrast identical closed state and permitted rejection/repeated accepted control with a typed changed closed after-state. Attach to the actual individually expanded Property without altering shared capability meaning or extending illustration support to compositions.
- For `completionSucceeds` (`workflow/system/System.scala:287`), contrast a selected completion with/without the recorded completion fact and explain why phase is not promised. Derive the unrelated-action probe for presentation using .3's classifier; do not author it as a passing positive or negative illustration.
- Run local always-true/false controls over each selected Property's valid positive/negative inputs. Preserve source/IR input pins, confirm classification-mismatch failure, then restore predicates. Keep any real model failures separate from successful negative illustrations.
- Perform the pilot's one production regeneration under the shared lock. Review all IR metadata and managed document differences against .1's immutable baseline. Do not let unmanaged edits or metadata alter behavior or generate live Cases; .6 joins the complete compatibility proof and gates.

### Investigation targets
**Required:**
- `model/temporal/features/nexus/workflow/system/System.scala:274` - synchronous success promise
- `model/temporal/features/nexus/workflow/system/System.scala:287` - completion applicability and phase boundary
- `model/temporal/features/nexus/standalone/system/System.scala:150` - existing closed transition definition
- `model/temporal/capabilities/Closable.scala` - expanded ownership of the selected transition
- `tools/umpire/check/illustrations_test.go` - classifier controls created by .3
**Optional:**
- `model/docs/property-illustrations/README.md` - deterministic output produced by .4

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/check -run 'Illustration'
make lint-model
```
Run the required production regeneration once and preserve its command, exit status, elapsed wall time and managed diff.
## Acceptance
- [ ] Exactly the three selected existing Properties carry at least one valid satisfied and one valid violated illustration, with all three synchronous-success contrasts present.
- [ ] The transition pair checks closed before/after state preservation, and the applicability document distinguishes selected completion from unrelated action without a vacuous success.
- [ ] Each selected Property's always-true/always-false controls fail for classification mismatch using valid inputs and the production classifier; original predicates remain intact.
- [ ] Regeneration produces the intended metadata/document only; baseline Case contents, transition relation, behavioral answers and identities remain protected for .6's complete proof.
- [ ] Hypothetical violations are visibly separated from reachable model failures and live execution evidence.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
