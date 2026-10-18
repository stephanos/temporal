---
satisfies: [R3, R4, R5, R6, R7]
---
# fn-75-separate-lean-target-semantics-from.3 Qualify Target compatibility and trust and document the semantic boundary

## Description
Complete compatibility/trust evidence and document the final ownership after consumer migration.

**Size:** M
**Files:** Target compatibility/import/kernel tests, Planning replacement tests, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md.
**Touches:** [model/Umpire/Target/ImportTests.lean, model/Umpire/Target/SemanticsImportTests.lean, model/Umpire/Target/Tests/**, model/Umpire/TargetTests.lean, model/Umpire/Planning/Tests/**, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md]

### Approach
- Replace empty expected-error guards in `model/Umpire/Target/SemanticsImportTests.lean` with explicit expected diagnostics, as established by task 2's Planning import test. Demonstrate that each constructor/frontend guard fails when its protected identifier is made available in a controlled test. Preserve the positive semantic checks and independent actual metadata-closure proof.
- Compare moved trust-bearing declarations and default-argument dependencies with task 1's pre-edit inventories. Audit new proof declarations against the explicit existing assurance boundary; do not accept placeholders or extra compiler trust. Keep exact commands and full inventories in evidence.
- Reuse frozen compatibility/mutation/authoring tests without rewriting expected bytes. Verify canonical metadata, fingerprints, selected source spans and fallback errors against the baseline.
- Close only uncovered R6 tests: available and absent replacement planning, unchanged enumeration/order, and insufficient equivalence-proof rejection. Preserve independent completeness obligations and existing finite-machine/table tests.
- Update public import guidance and ownership diagrams; distinguish semantic consumers from ordinary authoring without adding another authoring language. Preserve module/declaration comments.
- Run combined model build and lint gates, including the existing experiment consumer through its established script. Run existing staleness checks when any owned generated surface is affected; do not expand broad generated API drift or CI scope.

### Investigation targets
**Required:**
- model/Umpire/Target/Tests/Compatibility — frozen fixtures
- model/Umpire/Planning/Tests/Fixtures.lean:219 — real replacement-kernel consumer
- model/Umpire/ARCHITECTURE.md:16 — public import table
- model/ARCHITECTURE.md:17 — dependency diagram
- model/README.md:66 — author import guidance
- experiments/umpire-dsl/run-baseline.sh:4 — additional consumer
- Makefile:1404 — complete model lint gate

### Verification
Run serial Lean commands with LEAN_NUM_THREADS=1: make umpire-build-model, make lint-model, and the existing experiment baseline script. Run make lint-code GOLANGCI_LINT_FIX=false with the established macOS compiler/TMPDIR environment. Compare any inherited diagnostics against the verified baseline; do not claim the lint gate clean or claim unreached phases ran. Task 3 owns final combined verification for all three tasks.

## Acceptance
- [ ] Before/after transitive axiom evidence proves no trust expansion, including checkedTarget's default argument.
- [ ] Frozen canonical/fingerprint and authoring diagnostic fixtures remain unchanged; available/unavailable planning and insufficient replacement proofs retain their contracts.
- [ ] Public import documentation describes the actual final boundary and preserves approachable authoring entrypoints.
- [ ] Full model build, model lint, affected experiment consumer and required non-fixing Go lint have recorded outcomes; inherited failures are precisely classified against baseline.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: