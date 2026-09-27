---
satisfies: [R18]
---
# fn-93-simplify-the-lean-model.33 Property stack and Producer internals (A9)

## Description
Lane A9. One `PropertyPattern.toAtom?`/`ofAtom?` pair in `Property.lean` replaces the five pattern/atom conversions (`Case/Producer.lean` ~411-427, `Property/Correlated.lean` ~44-57, `Property/Check.lean` ~431, 451, 1127-1149); the Producer's `patternHolds` calls the shared step evaluator; `checkClause` (~`Check.lean:709-922`) extracts `checkException` and `checkTemporalShape`; one `PropertyPredicate.atoms` enumeration (every atom, through disjunction and negation) replaces only the traversals with that meaning: `fieldOperands` (`Evaluate.lean` ~424), `predicateAtoms` (`Check.lean` ~577) and `predicateReferences` (`Evaluate.lean` ~2034, if B6 kept it). `establishedFields` (`Check.lean` ~502), which deliberately ignores disjunction and negation, and `conjuncts` (`Case/Projection/Lowering.lean` ~138), which rejects them, keep their structure; they may share a structure-preserving fold with explicit per-connective handling, never the flat list; `resolveEvidence` (~551) and `alternativeRules` (~607) become one witness walk.

**Size:** M
**Files:** `model/Umpire/Property.lean`, `model/Umpire/Property/Check.lean`, `model/Umpire/Property/Evaluate.lean`, `model/Umpire/Property/Correlated.lean`, `model/Umpire/Case/Producer.lean`, `model/Umpire/Case/Projection/Lowering.lean`
**Touches:** [model/Umpire/Property.lean, model/Umpire/Property/**, model/Umpire/Case/Producer.lean, model/Umpire/Case/Projection/Lowering.lean]
**Depends on other specs:** fn-88.4 and fn-89.4 (Producer, Lowering, Check, Evaluate).

### Approach
- Before refactoring, add a `#guard_msgs` test where none exists that triggers both `evidence.action-unmapped` and `evidence.kind-ambiguous` conditions and pins the first; the order of every diagnostic stays.
- One sub-refactor per commit (conversions, `checkClause`, atoms fold, witness walk), each byte-checked.

### Investigation targets
**Required:**
- `model/Umpire/Case/Producer.lean:400-440,540-620`
- `model/Umpire/Property/Check.lean:420-460,700-930,1120-1150`
- `model/Umpire/Property/Correlated.lean:40-60`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression umpire-check-case-runtime-conformance
```
## Acceptance
- [ ] One conversion pair, one atom enumeration for the three compatible traversals, one witness walk; `checkClause` shares its two checks
- [ ] The negative pins for field presence established only under disjunction or negation (`model/Umpire/Property/Tests/Fields.lean` ~148) and for lowering a disjunction (`model/Umpire/Case/Tests/FieldLowering.lean` ~745) pass unchanged
- [ ] Precedence test added and green; every diagnostic order and every golden/fixture byte-identical
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
