---
satisfies: [R12, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.29 Diagnose primary inspection polling capacity refusal

## Description
Diagnose the actual C8 primary Temporal inspection failure without changing its requirements. The original onebox explore exited3 instead of expected1 with transition-bytes limit1048576 used1048560 delta188. Preserve C8 raw evidence unchanged; separately labelled diagnostics do not qualify the candidate.

Two fresh bounded reproductions now retain byte-identical failed World prefixes, capacity records and guest output. Evidence identifies wasm-broker ownership,2136 visible calls/6408 transitions, virtual time471133454ns, and correct charging:2124 network calls at491bytes plus12 idle calls at473bytes equals1048560. The network trace shows1474 pending results among1647 polls; before/after-idle scans poll every pending linked-list entry. This establishes polling amplification, not arithmetic overcount or a demonstrated accounting defect.

Evidence: /home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/actual-diagnosis-r1.{json,md}. Diagnostic instrumentation stays isolated and uncommitted. Independent review and evidence authentication remain required before diagnosis is closed.

Repair and full unchanged canary RED/GREEN obligations have been transferred explicitly to .30. This split does not discharge the actual capacity failure, .13 full qualification, .22-.25 receiving, or .14 reach.
## Acceptance
- [x] Authenticate both actual diagnostic reproductions under original1MiB/scenario/assertions and exact source/compiler/runtime/model/host bindings; preserve failed World/network prefix and guest output, with original C8 raw attempt unchanged.
- [x] Verify selected World owner, exact accounting/op counts, virtual time and guest progress. Distinguish observed polling amplification from unsupported accounting-error or DNS-loop claims.
- [x] Independently review the diagnosis and its limits; retain raw evidence and instrumentation provenance. Claim no production repair or qualification pass from diagnostics.
- [x] Transfer repair, behavioral RED/GREEN and full original canary obligations into .30, and make .13 closure depend on .30. All original runtime/replay/capacity/receiving obligations remain intact.
## Done summary
Diagnosed the unchanged primary inspection refusal using two fresh separately labelled executions with byte-identical failed prefixes/guest streams. wasm-broker accepted6408 transitions/2136 visible calls totaling1048560B; next188B refused correctly at virtual471133454ns.1474 of1647 network polls remained pending; runtime scans explain amplification, not arithmetic overcount or demonstrated DNS causation. Independently authenticated raw hashes/counts/scenario/settings and diagnostic identity distinctions. Historical Go had no active recorder for these owners; current Rust explicitly activates per-owner native bounds, which remain required. No production repair committed and no qualification pass inferred. Repair/fullcanary obligations transferred intact to.30; .13 closure depends.30 and full96/final145 plus existing receiving. Three formal Codexgpt-6.1-sol/high draws SHIP,0findings,same-family disclosed. Durable receipt: /home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/diagnosis-review/receipt.json. OriginalC8/r1 unchanged; r3 scope addendum and archive custody preserve provenance.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c3e7a89f29e37e8254310ed01bf2528e53cfb8cf, be8f2726c5238ffdae878f02c82493862a959acd
- Tests: Actual diagnostic reproductions02/03: /home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/actual-diagnosis-r1.json, Raw authentication: /home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/parent-diagnosis-authentication.json, Historical scope: /home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/actual-diagnosis-scope-addendum-r3.md, Green unchanged code gate: /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/conductor-repaircombined-gate.json, Independent SHIP: /home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/diagnosis-review/receipt.json
- PRs: