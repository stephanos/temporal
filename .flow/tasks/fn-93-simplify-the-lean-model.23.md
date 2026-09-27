---
satisfies: [R16]
---
# fn-93-simplify-the-lean-model.23 Move the test-only production surface into test support (A10)

## Description
Lane A10. Move `Operation/Parameterized.lean` (99; its fixture feeds `Search/Tests/{BackendVeil,Product}` and `Workflow/Start/Tests`), the `Run` and `Verdict` namespaces and ~20 production-unused builders of `Testpilot/Authoring.lean` (~679-740), and `SearchView.ofFinite`, `ofCheckedQuery?`, `FiniteKernelOrder` (`Search.lean` ~76-177) into test support.

**Size:** M
**Files:** `model/Umpire/Operation/Parameterized.lean`, `model/Umpire/Model/Tests/Parameterized.lean`, `model/Umpire/Search/Tests/{BackendVeil,Product,Fixtures}.lean`, `model/Umpire/Search/VisibilityTests.lean`, `model/Temporal/Feature/Workflow/Start/Tests.lean`, `model/Testpilot/Authoring.lean`, `model/Umpire/Search.lean`, a test-support module per package, `model/ModelLint/ImportGraph.lean` (test-support policy if a new namespace)
**Touches:** [model/Umpire/Operation/**, model/Umpire/Model/Tests/**, model/Umpire/Search.lean, model/Umpire/Search/Tests/**, model/Umpire/Search/VisibilityTests.lean, model/Temporal/Feature/Workflow/Start/Tests.lean, model/Testpilot/**, model/ModelLint/**]
**Depends on other specs:** fn-88 closed (Search surfaces, R12 pins: lower only); fn-94 lane G deletes builders in `Testpilot/Authoring.lean` — re-read fn-94's state and move only what remains.

### Approach
- For each candidate, grep production users (non-test modules) first; any hit keeps it.
- Moves keep declarations' names where tests reference them via `open`; update imports.

### Investigation targets
**Required:**
- `model/Testpilot/Authoring.lean:660-745`
- `model/Umpire/Search.lean:70-180`
- `model/Umpire/Operation/Parameterized.lean`

### Quick commands
```sh
cd model && lake build
LEAN_NUM_THREADS=1 make lint-model
make umpire-check-goldens
```

## Acceptance
- [ ] Listed declarations live in test support; production modules contain none of them
- [ ] fn-88's Search pins lowered, not raised; goldens byte-identical
- [ ] `lint-model` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
