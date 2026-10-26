---
satisfies: [R5, R7]
---
# fn-255-thorough-framework-testing-generated.6 Coverage report per framework crate; aim generators and corpora

Touches: [model/umpire/mise.toml, model/design/coverage.md, model/umpire/dev/**, model/umpire/fuzz/corpus/**]

## Description
Show which refusal and lowering branches no test reaches, and use that to aim .2's generator and .3's corpora (spec R5).

**Size:** S
**Files:** `model/umpire/mise.toml` (cargo-llvm-cov), `model/design/coverage.md`, generator weights, corpus additions.
**Touches:** [model/umpire/mise.toml, model/design/coverage.md, model/umpire/dev/**, model/umpire/fuzz/corpus/**]

### Approach
- Pin cargo-llvm-cov via the framework mise file. Collect line coverage and explicitly enabled `--branch` coverage on the dated nightly from .3, with its matching llvm-tools component, in an isolated target directory. Keep the stable production toolchain unchanged. Record the toolchain and exact coverage flags; the [cargo-llvm-cov documentation](https://github.com/taiki-e/cargo-llvm-cov#usage) requires nightly for branch coverage.
- Provide scheduled per-crate coverage commands with test/case manifests and total work/time caps from R7; report completed/remaining crates/tests when capped. Full coverage campaigns are not closure prerequisites.
- At closure, run a bounded representative test/case manifest, then list its unreached refusal and lowering branches in `umpire-front`, `umpire-ir`, `umpire-tla` and the interpreter, with file:line.
- Adjust generator weights and add corpus seeds aimed at representative uncovered branches; repeat only the bounded manifest and record comparable before/after branch evidence. Label this smoke coverage separately from broader scheduled per-crate reports.

## Acceptance
- [ ] Per-crate scheduled commands and capped partial-report controls work; bounded smoke coverage records toolchain, flags, manifest, work counts and time in `model/design/coverage.md`.
- [ ] Representative unreached refusal/lowering branches drive a generator/corpus adjustment with comparable bounded before/after evidence; full campaigns remain scheduled-only.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
