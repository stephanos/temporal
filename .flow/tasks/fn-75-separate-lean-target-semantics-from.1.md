---
satisfies: [R1, R3, R4, R5, R6, R7]
---
# fn-75-separate-lean-target-semantics-from.1 Extract checked Target semantics and pure admission ownership

## Description
Establish the Target ownership split and prove the narrow semantic import before consumer migration.

**Size:** M
**Files:** model/Umpire/Target/Language.lean, model/Umpire/Target.lean, new Target semantic/projection/frontend modules, Target import tests and finite adapters as required.
**Touches:** [model/Umpire/Target/**, model/Umpire/Target.lean]

### Approach
- Before editing, capture exact canonical compatibility fixture hashes and transitive axiom inventories for moved checked/admission/projection/replacement declarations and checkedTarget's default argument. Persist task-unique baseline artifacts and declaration mapping for task 3; record commands and inventories in evidence. Do not derive the baseline after extraction.
- Separate pure representation/projection from syntax capture and TermElabM integration. Keep private checked construction with its single admission authority; pure occurrence/diagnostic data may remain there when needed. Preserve names and signatures through the existing authoring facade.
- Introduce Umpire.Target.Semantics and a positive import test that exercises kernel and finite planning contracts. Verify actual import closure, including dependencies of default arguments. Preserve checkedTarget's default-call contract without using an unchecked constructor bridge. Resolve any conflict here before migrating consumers.
- Preserve canonical implementations and the withEquivalentKernel proof obligations. Retain finite adapters in their existing owners, with narrower imports only where required by the new ownership. Move all existing comments with declarations and document new module boundaries.

### Investigation targets
**Required:**
- model/Umpire/Target/Language.lean:73 — private assembly, pure checking and elaboration currently colocated
- model/Umpire/Target/ImportTests.lean:47 — constructor rejection guards
- model/Umpire/Target/Tests/Compatibility — frozen byte/fingerprint fixtures
- model/Umpire/Target/Tests/FiniteMachine.lean:474 — existing axiom audit pattern
- model/Umpire/Target/Tests/FiniteTable.lean:124 — finite trust checks
**Optional:**
- model/Umpire/Planning/Tests/Fixtures.lean:219 — checked kernel replacement consumer

### Verification
Run serially with LEAN_NUM_THREADS=1 using the established model toolchain: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Target.ImportTests, plus the new positive semantic-import root. Verify imported module closure and preserved baseline fixtures immediately. No generated outputs or toolchain changes are expected.

## Acceptance
- [ ] Pre-edit trust inventories, declaration mapping and fixture hashes are captured and handed to task 3.
- [ ] Semantic import and ordinary authoring imports compile; constructor-forgery guards remain effective and no elaborator dependency enters the semantic closure.
- [ ] Existing Target validation, authoring diagnostic, finite adapter and compatibility tests pass with unchanged expected bytes and comments preserved.
- [ ] R1/R3/R4 ownership and default-argument trust constraints have an executable early proof; any unresolved split is replanned before task 2.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: