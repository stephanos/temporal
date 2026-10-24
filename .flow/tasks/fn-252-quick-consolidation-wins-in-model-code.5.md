---
satisfies: [R5, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.5 Test consolidation: trybuild cases, guest harness, protocol::Method, catalog-syntax tests

## Description
Implement **trybuild cases (B5)**, **Guest test harness (B6)** and **Catalog-syntax tests (B7)** from the spec. Where cases or tests live in `temporal/annotate`, follow fn-251's facade removal if it has landed.

## Acceptance
- [ ] Every removed UI case maps to a table row or unit test with the same message and span.
- [ ] `GuestExt`/`ResponseExt`, `protocol::Method` and `assert_additions_refused` exist once.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
