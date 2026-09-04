---
satisfies: [R3, R4, R5, R6]
---
# fn-63-consolidate-umpire-go-tests-into-golden.5 Prune redundant tests and lock regression gates

## Description
Finish R3-R6 by removing cross-package remnants, proving the reduction target, and aligning the existing fixture/package/live gates and conditional documentation with the final suite.

**Size:** M
**Files:** affected `tools/umpire/**/*_test.go`, `tools/umpire/**/testdata/**`, `Makefile`, `.github/workflows/umpire.yml`, and affected `tools/umpire/**/README.md`
**Touches:** [tools/umpire/**/*_test.go, tools/umpire/**/testdata/**, tools/umpire/README.md, tools/umpire/portableevaluation/README.md, tools/umpire/runevaluation/README.md, tools/umpire/runtime/README.md, Makefile, .github/workflows/umpire.yml]

### Approach
- Audit the completed migrations against the post-`fn-61` baseline and removal maps; delete remaining duplicate local loaders/builders/assertion matrices only when a named scenario or retained focused category covers them.
- Measure handwritten test lines, top-level tests/fuzzers, and added human-authored harness/manifest lines with the same commands as Task `.1`; report generated test files and oracle payload bytes separately.
- Replace direct in-checkout fixture generation with a temporary-root-only workflow: generate the complete tree, admit/validate every expected output, and diff it before any separately reviewed promotion. Failure or interruption before promotion must leave the checked-in tree byte-identical.
- Keep the normal fixture check non-mutating and extend its package/fixture coverage only as required by the new layout; do not add broad generated API drift verification or a new CI workflow.
- Preserve the complete `^TestUmpire` live selector, integration/test dependency tags, inherited-failure identity policy, and all packages needed by the final test topology.
- Update only READMEs whose fixture layout, generation command, package gate, or proof topology changed; document when a future test belongs in a scenario versus a focused invariant test.

### Investigation targets
**Required** (read before coding):
- `Makefile:1071-1083` — current direct generation and temporary diff workflows
- `Makefile:1143-1172` — complete live selector and failure-baseline policy
- `Makefile:1174-1180` — aggregate regression package list
- `.github/workflows/umpire.yml:34-40` — current package-local and live gates
- `tools/umpire/portableevaluation/README.md:193-242` — fixture and runtime proof documentation

**Optional** (reference as needed):
- `tools/umpire/runevaluation/README.md:125-184` — Run Evaluation proof topology
- `tools/umpire/runtime/README.md:3-43` — generated-test and CI contract
## Acceptance
- [ ] Handwritten `tools/umpire` test lines and top-level tests/fuzzers are each at least 15% below the Task `.1` baseline, and added human-authored harness/manifest code is smaller than the test code removed.
- [ ] The final removal map accounts for every deleted/merged test and shows the retained scenario or focused invariant category.
- [ ] Deliberate fixture generation writes only to a temporary root, validates a complete tree, and requires a separate reviewed promotion; a forced generation failure/interruption leaves the checked-in fixture tree byte-identical.
- [ ] Ordinary package tests, complete temporary-root fixture checks, aggregate regression, and full tagged `^TestUmpire` live selection pass with no narrowed package or inherited-failure baseline.
- [ ] Documentation matches the final fixture layout and commands, and no unrelated architecture documentation changes.
- [ ] `make fmt-imports` and `make lint-code` pass; public APIs, production imports/dependencies, generated protocols, runtime behavior, and existing comments remain unchanged.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
