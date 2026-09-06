---
satisfies: [R1, R3]
---
# fn-66-remove-unused-umpire-tooling-after.1 Classify surviving tooling and freeze deletion evidence

## Description
Create the complete consumer and per-case deletion ledger for R1/R3 before either removal slice.

**Size:** M
**Files:** tools/umpire/CLEANUP_INVENTORY.md
**Touches:** [tools/umpire/CLEANUP_INVENTORY.md]

## Approach
- Verify fn-64/fn-62 closure and freeze actual HEAD plus the cumulative staged/source tree. Preserve user changes and fn-64's immutable ledger.
- Enumerate every current tools/umpire package and command. Name actual imports, command callers, Make/script/workflow/generator references, fixtures, or concrete retained-spec contracts. Record negative searches for removal rows; resolve every ambiguity before finishing.
- Validate the initial artifact package/CLI retirement and subsequent internal artifactv2 trim. Enumerate all 97 candidate Test functions, one Fuzz function, and 27 fixtures by name/path and owner; classify fixture origin and surviving references. Record preserved/replaced/intentionally-retired dispositions and reasons, while keeping baseline rows distinct from later realized deletions.
- Record the complete retained Experiment reader symbol closure used by both generated-view consumers, including seal/checksum helpers. Preserve runtime/Hosts, verification, generators, vocabulary, regression and generic Lean promotion. Keep tools/planindex outside scope.
- Capture retained generator-output identities and existing process-level command checks for later comparison. Record the exact lint baseline and per-deleted-file diagnostic subtraction; do not hide additional unused items or broaden deletion without revising the reviewed task scope.

## Investigation targets
**Required:**
- `tools/umpire/CONTEXT.md` — retained public ownership and vocabulary.
- `.flow/artifacts/fn-64-umpire-case-runtime/task8-migration-ledger.md` — immutable accounting pattern and generic promotion retention.
- `tools/umpire/cmd` — all command entrypoints, not only imported packages.
- `Makefile:996` — plan-index exclusion and artifact/generator/regression wiring.
- `tools/umpire/cmd/umpire-gen-regression-views/generated_view.go:12` — retained Experiment consumer.
- `tools/umpire/regression/generated_view.go:17` — second retained Experiment consumer.
- `.github/workflows/umpire.yml:34` — unchanged full gates.

## Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/artifact ./tools/umpire/cmd/umpire-artifact ./tools/umpire/internal/artifactv2
```
Use `go list -tags test_dep ./tools/umpire/...` and test-name inventory alongside repository reference searches; no deletion or new inventory program.

## Acceptance
- [ ] One ledger covers every current package/command with concrete evidence; zero unclassified or ambiguous rows remain and both prerequisites are verified closed.
- [ ] Every candidate Test/Fuzz name and fixture path has ownership/origin/reference evidence and a disposition; the 97/1/27 totals reconcile or a concrete reviewed correction explains the difference.
- [ ] Retained Experiment symbols, generated-output identities, process checks and exact lint baseline/subtractions are recorded separately from proposed deletion rows; fn-64's ledger and all source code remain unchanged.
- [ ] Focused baseline passes, and any additional unused item or scope conflict is resolved through a concrete task adjustment before .2 starts.

## Done summary
Created the frozen Umpire tooling cleanup ledger with concrete ownership for all 21 packages and seven commands, individual accounting for 97 Tests, one Fuzz target, and 27 fixtures, the retained Experiment reader closure, generated identities, process checks, and exact lint subtraction. The focused baseline passed with a physical canonical TMPDIR; no source, generated output, or fn-64 ledger changed.

Verification limitation: `flowctl gate classify --base ff9ea9827157255068a87086630651a43cc01060` reported `FULL: unmatched: .plans/UMPIRE4_ORDER.md` for root-owned Markdown. The sole task-owned overlay is `tools/umpire/CLEANUP_INVENTORY.md`, so full gates were not repeated; task .3 owns them.

stage: impl-review - ran [2026-09-06T15:21Z..2026-09-06T15:26:35Z] (NEEDS_WORK fixture-owner correction -> SHIP)
## Evidence
- Commits:
- Tests: TMPDIR=<physical /private/tmp/fn66-task1.*> go test -count=1 -tags test_dep ./tools/umpire/artifact ./tools/umpire/cmd/umpire-artifact ./tools/umpire/internal/artifactv2, go list -tags test_dep ./tools/umpire/..., inventory reconciliation: 21 packages, 7 commands, 97 Test, 1 Fuzz, 27 fixtures, GATE_SKIPPED:unittest:task-owned-doc-only - sole task overlay is tools/umpire/CLEANUP_INVENTORY.md; flowctl classify saw unrelated root-owned .plans/UMPIRE4_ORDER.md and full gates belong to .3
- PRs: