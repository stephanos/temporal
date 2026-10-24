---
satisfies: [R4]
---
# fn-191-gomad-sometimes-assertions-and-a-site.4 Campaign flags, never-hit summary, must-hit outcome and corpus features

## Description
gomad-campaign: --coverage=semantic, --require-probe=NAME, --must-hit=NAME (repeatable, globs) and --keep-successes=novel accept catalogued names on cooperative WASM; unknown names refused before execution (status 2). Campaign summary lists never-hit sometimes entries with execution counts; a never-hit --must-hit entry ends a completely observed campaign with a dedicated outcome and status 1; incomplete observation yields inconclusive coverage unless an authenticated positive hit already satisfies the entry. Hit sets enter the existing corpus features. The log-scanner Always oracle in soft_assertions.rs stays as fallback for pre-change artifacts with its inconclusive semantics. Depends on .3.

## Acceptance
Flag parsing and refusal tests; summary golden with never-hit entries; dedicated must-hit outcome and exit status; inconclusive path tested with an incomplete incarnation; corpus feature golden includes hit IDs; pre-change artifact still assessed by the scanner.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
