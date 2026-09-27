---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.19 Delete Search/Branches and the Property result layer (B6, decision D6)

## Description
Lane B6, part one; requires fn-88.7 (fn-88's final task) closed. Delete `Search/Branches.lean` (655) if nothing but tests reaches it, the test-only wrapper `AdmittedQuery.analyzeBranches` (`Search/Admission.lean:147-149`), the Branches imports in `Command/Authoring`, and its `semanticRoots` lint entry. Then delete `Property/Evaluate`'s full-result and joint-obligation layer (`PropertyClauseResult`, `PropertyEvaluation`, case-applicability and overlap analyses, `Joint*`; ~`Evaluate.lean:1729-2175`) and `JointConflicts` tests (247).

### Owner decision
- **D6 — delete `Search/Branches`, the result layer and the guarded forms after fn-88. Recommended default: taken.** Record first; if declined, close this task and task 20 with no change. **Decision interaction:** if D1 was declined, `Variations/Compiler` still imports `Search.Branches`; keep `Branches` (delete only the result layer and the wrapper nothing kept uses) and list it.

**Size:** M
**Files:** `model/Umpire/Search/Branches.lean`, `model/Umpire/Search/Admission.lean`, `model/Umpire/Command/Authoring.lean` (import), `model/ModelLint/ImportGraph.lean` (`semanticRoots`), `model/Umpire/Property/Evaluate.lean`, `model/Umpire/Property/Tests/JointConflicts.lean`, `model/Umpire/Property/Tests/Evaluation.lean` (parts), Search tests reaching Branches, `model/UmpireTests.lean`, `model/Umpire/Search/VisibilityTests.lean`, `model/ModelLint/ModuleIndex.lean`, vocabulary gate, `model/README.md:92`, `model/Umpire/ARCHITECTURE.md:124`
**Touches:** [model/Umpire/Search/**, model/Umpire/Command/Authoring.lean, model/ModelLint/**, model/Umpire/Property/Evaluate.lean, model/Umpire/Property/Tests/**, model/UmpireTests.lean, tools/umpire/internal/retiredvocabulary/**, model/README.md, model/Umpire/ARCHITECTURE.md]
**Depends on other specs:** fn-88.7 closed (fn-88 R12 kept `analyzeBranches` while the frozen reference engine existed).

### Approach
- Confirm nothing in production reaches `Branches` after fn-88 (`grep -rn 'Search.Branches\|analyzeBranches'`); confirm `Command/Authoring` uses nothing from it.
- `.branches` in `Case/Relation.lean:131-135` keeps its one-group, one-case, same-step shape; relation fingerprints stay byte-identical.
- With B2 gone, confirm nothing in production reaches `evaluateProperty`'s full-result layer before deleting it; E2 entries for deleted theorems go in the same commit.

### Investigation targets
**Required:**
- `model/Umpire/Search/Branches.lean` importers
- `model/Umpire/Property/Evaluate.lean:1720-2180`
- `model/Umpire/Case/Relation.lean:125-140`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression umpire-check-retired-vocabulary
LEAN_NUM_THREADS=1 make lint-model
```
## Acceptance
- [ ] D6 recorded; Branches (unless a kept Variations package imports it, then listed), its wrapper, lint root entry and the result layer gone
- [ ] Relation fingerprints and every golden byte-identical
- [ ] `lint-model` (incl. semantic roots) and every gate green
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
