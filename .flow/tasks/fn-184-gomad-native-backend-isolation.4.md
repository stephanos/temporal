---
satisfies: [R5]
---
# fn-184-gomad-native-backend-isolation.4 Measure, document and hand over the isolated candidate

## Description
Repeat the task .1 measurement on the frozen integrated candidate on the same quiet host: default-set and selected-set package count, production/test lines and wall time for build, vet, lint, test-harness, world-test, test-wasm; report medians with ranges and the host identity. Update tools/gomad/README.md, ARCHITECTURE.md (module layout, maintenance gates, host tooling boundary) and MILESTONES.md (native section header, constraints) to describe the selection, the default-build refusal and the preserved native gates, stating that native execution remains deferred under fn-128/fn-149. Produce one batch handover mapping tasks .1 to .4 and R1 to R5 to identities, commands, exits and the independent review verdict.

**Size:** S
**Files:** tools/gomad/README.md, tools/gomad/ARCHITECTURE.md, MILESTONES.md, .flow/artifacts/fn184-*.md

## Acceptance
- [x] Before/after table retained with host identity, medians and ranges — package/line counts retained; wall-time medians and ranges **transferred to fn-185 R6** (owner decision 2026-10-17)
- [x] README, ARCHITECTURE and MILESTONES describe selection, refusal and preserved gates; no native qualification claimed
- [x] Batch handover maps every task and R-ID to evidence and the independent review — `.flow/artifacts/fn184-closeout-2026-10-17.md`
## Done summary
Candidate `f22f8ef227` (merge of gomad `3dae95b99f`) fast-forwarded into gomad; Phase-B gates exit 0 (archcheck, validate, lint on both sets; test-wasm on run 2 after installing the in-tree helper). Structural counts recorded: before 80 packages / 118,144 prod / 131,789 test lines; after default 52 / 80,909 / 85,481; after selected 82 / 118,414 / 132,400. README, ARCHITECTURE and MILESTONES describe the selection, refusal and preserved gates without claiming native qualification. Wall-time medians and ranges are transferred to fn-185 R6 (owner decision 2026-10-17). Detail: `.flow/artifacts/fn184-closeout-2026-10-17.md`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f22f8ef227, 6898eceb5a
- Tests: GOMAD_WASM_TEST_HELPER=<helper 1440ad5d> GOMAD_WASM_CONFORMANCE_HELPER=<helper 1a54997b> make -C tools/gomad test-wasm GOMAD_STOCK_GO=<pinned go1.27.1>, bash /home/agent/wt/evidence/fn184/measure.sh structural
- PRs:
- Detailed evidence: `.flow/artifacts/fn184-closeout-2026-10-17.md` (R-ID map, commits, commands, exits, reviews, fn-185 R6 transfer)
