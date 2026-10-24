---
satisfies: [R12, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.30 Repair inspection polling, measured canary budget and retained replay

## Description
Touches cooperative guest polling, the full inspection canary budget, and the authenticated retained-incarnation replay readers reached by that canary. Preserve the complete original TestReplayInspectionWorkflowFailure/TestPrototypeWorkflow workload, both one-minute timers, activity retries, all required markers, deterministic scheduling and exact replay.

### Diagnosis and measured budget decision

Task .29 remains complete historical diagnosis. Its original 1 MiB refusal at virtual471133454ns correctly charged2136 calls/6408 transitions. Retain actual-diagnosis-r1, the scope addendum and all original failures unchanged. Active native per-owner recording and exact canonical-byte charging remain binding.

The bounded non-consuming readiness probe and positive-hint refinement on worker22ae55f5ab reduced redundant individual pending polls, but two full original1MiB attempts still refused capacity. The separately labelled16MiB diagnostic used unchanged uninstrumented product, original timers/scenario/output bounds and genuine higher-bound preparation. It completed the original workload at virtual3m0.008999999s with the exact required completion markers. Broker history retained9692 calls/29076 transitions and4718416 canonical native transition bytes;2242 genuine idles alone charged1060466 bytes. The endpoint network World charged272851 bytes. Its detached replay then exposed the separate reader contract defect below; this diagnostic is not a qualification pass.

On2026-10-24 the owner asked, "by how much? should we raise it?" The conductor recommended measuring the full run and selecting the canary budget with margin, under the standing instruction to follow recommendations. Select8MiB (8388608 bytes) for this onebox inspection canary, approximately78percent headroom over measured4718416 bytes. This explicitly supersedes this task's earlier unchanged1MiB canary requirement. Preserve1MiB generic capacity/refusal regressions, other workload budgets, model accounting, old artifact settings and failed attempts. No trace-format redesign, counter discount, timer change or hidden operation is selected. Retain measurement/provenance under /home/agent/wt/evidence/fn232-inspection-poll-batch/measure-retained-world-16m.json and onebox-diagnostic-16m-outcome.json.

### Supported polling repair

Retain separately versioned net_stream v2 op14 readiness_probe_v1 for id0 with exact64+8N bounded frames and0..128 unique local pending IDs. Preserve legacy invalid-op14/nonzero-id behavior. Probe ordinary Broker::check command draining and register/ready/quiesce accounting, authoritative owner time/readiness, and non-consuming results. Collect ready IDs through existing individual polls in existing list/wake order. Positive hints only request authoritative collection; before skipping a formerly-zero slot after an intervening visible operation, reprobe that suffix. Never infer safety from equal virtual time or a guest-only dirty cache.

Retain cross-language strict validation, empty/full/chunked/malformed/duplicate/foreign/stale cases, nonconsumption, same-scan cascades, cancellation/deadline/close/delayed delivery/partition/heal/revoke/restart and queued Link/Revoke boundary tests. The actual linked many-pending-read RED/GREEN and exact replay remain meaningful under the original1MiB regression budget, independently of the corrected full-workload canary budget. Preserve genuine Go oracle generation and verify old corpus behavior bytes remain unchanged apart from authentic source bindings; retain the genuinely added named case and source-identity comparison evidence. No identity normalization or old behavior regeneration.

### Retained-incarnation replay repair

The completed diagnostic retained a valid13935648-byte choice payload whose descriptor.limit was1210056704 bytes (1154MiB), the authenticated aggregate evidence budget. Producer and existing retained tests deliberately assign that evidence bound to stream descriptors. Three cluster replay/admission/inspection callers instead supply fixed1GiB to descriptor.read, refusing the declared bound before reading the small valid payload. Correct those callers to the matching authenticated incarnation.limits.evidence_bytes, consistent with existing typed readers. Preserve producer descriptors, generic Payload::bind/read identity and caller-bound checks, the512MiB module bound and actual model limits.

First retain focused RED with tiny authentic retained data declaring the supported1154MiB evidence budget, avoiding GiB allocation. GREEN must cover all three affected readers, and a descriptor declaring evidence_bytes+1 must still refuse. Do not rewrite an old artifact or its identities to claim cross-source replay; use focused bound tests and fresh final-candidate artifacts with genuine source identities for integration proof.

Update only the actual onebox8MiB command/assertion mirrors. Authenticate fresh affected source/helper/model/runtime/compiler/module/install identities. Then run the complete fresh8MiB canary including detached/stopped/reopened inspection/replay and the required offline shim harness. Pass canonical worker and conductor merge gates and independent green-tree SHIP review. This task does not replace .13 full96/final145, .22-.25 receiving or .14 reach obligations.
## Acceptance
- [ ] Preserve the supported bounded non-consuming polling contract, ordinary native accounting, invalidation/wakeup semantics and existing wake order; retain meaningful linked1MiB RED/GREEN and deterministic exact replay.
- [ ] Retain the full-workload native-byte measurement and explicit8MiB onebox-only budget rationale. Preserve original failed1MiB/diagnostic16MiB attempts and all generic capacity/refusal tests, model bounds and other workload budgets.
- [ ] Fix all three retained-incarnation caller bounds from authenticated evidence limits with focused RED/GREEN, overdeclared descriptor refusal and unchanged identity/module/generic payload checks. No artifact or identity rewriting.
- [ ] Pass the complete original inspection workload at8MiB with unchanged timers, retries, scenarios, assertions and expected results, plus detached/stopped/reopened replay/inspection; retain actual raw commands, numeric exits and genuine final-candidate identities.
- [ ] Pass offline shim coverage, canonical worker/conductor merge gates and independent green-tree SHIP review before completion; preserve old corpus behavior and authenticate genuine changed source/model/runtime preparations.
- [ ] Keep .13 full96/final145 and .22-.25 receiving pending until their actual fresh proof; .14 remains after .13.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
