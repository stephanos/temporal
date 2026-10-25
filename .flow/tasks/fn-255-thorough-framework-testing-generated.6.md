---
satisfies: [R5]
---
# fn-255-thorough-framework-testing-generated.6 Coverage report per framework crate; aim generators and corpora

Touches: [mise.toml, model/design/coverage.md, model/umpire/dev/**, model/umpire/fuzz/corpus/**]

## Description
Show which refusal and lowering branches no test reaches, and use that to aim .2's generator and .3's corpora (spec R5).

**Size:** S
**Files:** `mise.toml` (cargo-llvm-cov), `model/design/coverage.md`, generator weights, corpus additions.
**Touches:** [mise.toml, model/design/coverage.md, model/umpire/dev/**, model/umpire/fuzz/corpus/**]

### Approach
- Pin cargo-llvm-cov via mise. Collect line and branch coverage per framework crate from the workspace tests plus the generator's fixed-seed cases.
- List unreached refusal and lowering branches in `umpire-front`, `umpire-ir`, `umpire-tla` and the interpreter, with file:line.
- Adjust generator weights and add corpus seeds aimed at those branches; record branch coverage before and after.

### Acceptance
- [ ] Coverage per framework crate recorded in `model/design/coverage.md`.
- [ ] Unreached refusal and lowering branches listed; generator and corpora adjusted; before/after recorded.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
