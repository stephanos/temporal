---
satisfies: [R1, R2, R3]
---
# fn-143-rename-modelumpire-to-modelframework.1 Move model/umpire to model/framework, package umpire to framework; regenerate; prove the diff is the umpire.→framework. mapping; docs

## Description
**Size:** M
**Touches:** [model/**, tools/umpire/**, Makefile, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_SPEC.md, MILESTONES.md]

`git mv model/umpire model/framework`; rename the package `umpire` to `framework` in every declaration and import; update the Makefile source and scalafix lists, `.scalafix.conf`, the syntax lint's framework paths, the lifter's fully qualified framework names, and the gate tests. Regenerate once and classify the IR/Case diff (R2). Search the repository for leftover `model/umpire` and `umpire.` framework references; keep product names (`tools/umpire`, `umpire-*` targets, `umpire.v1`).
## Acceptance
- [ ] `model/framework/` with package `framework`; no `model/umpire` path or `umpire.` framework reference remains outside dated history (R1)
- [ ] Regenerated IR/Case diff contains only paths, positions, the `umpire.` → `framework.` mapping and the fingerprints it implies; Definition IDs, Query answers and receipts unchanged (R2)
- [ ] Lifter, gate, Model tests, `make lint-model`, `make umpire-check-model` and the Go tooling suite pass (R3)
## Done summary
Renamed the reusable Scala framework from `model/umpire` / `umpire.*` to `model/framework` / `framework.*` while preserving Umpire product, checker, lifter, protocol, and service names. The scratch before/after seal proved the declared path, position, qualified-name, and implied-fingerprint mapping without changing production `model/ir` or `model/cases`.

Focused Scala source, lifter, checker, syntax, formatting, and Go consumer tests passed. The full production regeneration, full model gate, full Go suite, and integrated Batch 1 review remain assigned to fn-145.4; the attempted full lifter fixture run timed out and is recorded as inconclusive rather than green, while its focused checks and scratch equivalence passed. The pre-existing scalafix/JDK failures were not treated as task success.

baseline: none (the parent spec defines no Quick commands)

Tier: session (jev-unavailable(no_key)); actual_model: gpt-5.6-sol

stage: impl-review - ran [2026-10-09T03:31:32Z..2026-10-09T03:41:40Z]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a19fe918b3251a83de2bb65cc143aa3f3e4f181d, 0043edb969b32227fe63953ed56c8f8adbccf00b
- Tests: make fmt-model, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model, mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check, mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen --test-only umpire.irgen.RolesSuite, mise exec -- scala-cli test --suppress-outdated-dependency-warning model --test-only temporal.IrFilesTest, make lint-model-syntax, go test ./tools/umpire/ir, go test ./tools/umpire/interp, go test ./tools/umpire/conformance, go test ./tools/umpire/check -run TestFrameworkLayout, go test ./tools/umpire/lower/internal/producer, scratch before/after Scala package and lift plus .flow/tmp/fn1431/equivalence.go, scratch fixture equivalence via .flow/tmp/fn1431/fixture_equivalence.go
- PRs: