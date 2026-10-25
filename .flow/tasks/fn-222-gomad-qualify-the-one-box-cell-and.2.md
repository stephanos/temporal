---
satisfies: [R2]
---
# fn-222-gomad-qualify-the-one-box-cell-and.2 Define the per-test support list and produce the first one

## Description
Proceed now on the Go candidate for the producer contract and guest-knowledge dispositions. Consume only actual matching-candidate receipts; if either required cell lacks a gate, retain an explicitly pending report with no unsupported qualified row. The first accepted Rust list is earned later by task .1 after fn-232.10; this phase never fabricates that proof.
Document and produce the candidate-bound `gomad.wasm-support-list/v1` through the existing qualification owner. Require registered cells' final success/canary Results and PublicationReceipts, a zero exit for every cell gate, exact canonical argument vectors and authenticated stdout proving the named test actually passed without skip or no-tests suppression. The first accepted list must include both the mandatory smoke selection `./tests` `TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds` and the supplementary `./tests/gomadfunctional` `TestFrontendSystemInfo` cell. Those preparation/measured commands describe the historical Go producer path and run only when authorized within that phase. Task .1 owns mandatory final Rust preparation, measurements and independently validated Results for both cells after fn-232.10. Retain any failing or unrun gate; this preparatory task can complete with an explicit pending disposition and no qualified rows, while a smoke-only list cannot complete final R2 acceptance. Record candidate/module identities, gate commands/exits and retained references per row, explicitly state unlisted tests are unsupported, and store JSON/Markdown unchanged next to checkpoint records with a QUALIFICATION.md reference. Sweep passes alone never add rows. Changed candidate identities must re-earn every row.

**Touches:** tools/gomad/internal/backends/wasm/qualification/**, tools/gomad/docs/wasm/QUALIFICATION.md, .flow/artifacts/fn222-*.json and .flow/artifacts/fn222-*.md (completed preparatory phase).

**Size:** S
**Files:** tools/gomad/internal/backends/wasm/qualification/**, tools/gomad/docs/wasm/QUALIFICATION.md, .flow/artifacts/fn222-*.md
**Ordering (owner decision 2026-10-20):** this preparatory Go phase supersedes the old dependency on fn-222.1. The installed CLI has no dependency-removal setter. The conductor may use supported `flowctl start --force` solely for this obsolete edge after verifying the task is todo and unclaimed; do not mark .1 complete to bypass it.

## Acceptance
- [ ] Support list shape documented in QUALIFICATION.md with per-test, candidate-bound wording
- [ ] Go-candidate report contains only actually gate-passing rows, or explicitly records pending gates without a support claim; task .1 retains final R2 acceptance for both required Rust cells
- [ ] A sweep pass alone cannot add a row (documented and enforced by the producer)

## Done summary
Completed the authorized Go preparatory phase: documented the existing candidate-bound support-list producer and both registered cells, retained an authenticated historical disposition report with zero qualified rows, and preserved final R1/R2 Rust acceptance under fn-222.1 after fn-232.10. No guest run, support-list JSON or qualification pass was fabricated. Existing producer behavior and rejection tests remain unchanged.

Integrated source is 9a5980335f; root repeated both qualification package tests with pinned Go and test_dep (exit 0 each). Worker focused tests/build/lint and all 172 historical input hashes passed. Independent correctness, contracts and integration reviewers all returned SHIP with no findings; finalized receipt is /tmp/impl-review-receipt-657da2bc4466-fn-222-gomad-qualify-the-one-box-cell-and.2.json, RID f89d9b157aa34b94b79600eadedfaff9, gpt-6.1-sol high (same family). The initial fanout was invalidated/refunded because root committed unrelated custody pointers before finalization; the replacement reviewed the current head and finalized successfully.

Handover: /home/agent/wt/evidence/fn222-support/handover/summary.md and evidence.json. Both required Rust cells and the first accepted Rust list remain pending; no whole-spec closure is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9a5980335f54d52d3f137b6727be9009db08aeec
- Tests: Pinned stock Go 1.27.1: go test -tags test_dep -count=1 ./internal/backends/wasm/qualification ./internal/evidence/qualification (root integrated candidate, exit 0), Worker focused TestSupportList tests, qualification package build and lint-code-fast: exit 0; /home/agent/wt/evidence/fn222-support/handover/evidence.json, 172 historical input hashes verified by worker and independent reviewers: exit 0, Three independent review draws finalized SHIP, no findings; RID f89d9b157aa34b94b79600eadedfaff9; /tmp/impl-review-receipt-657da2bc4466-fn-222-gomad-qualify-the-one-box-cell-and.2.json
- PRs:
