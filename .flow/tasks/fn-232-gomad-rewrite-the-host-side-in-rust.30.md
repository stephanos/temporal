---
satisfies: [R12, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.30 Repair inspection polling capacity under unchanged canary bound

## Description
Touches the cooperative guest network polling/runtime seam identified by task .29. Establish the smallest correct repair before selecting implementation files; readiness-aware or generation-aware polling is a hypothesis, not an approved design.

Task .29 reproduced the original primary inspection refusal twice with identical failure-prefix bytes: wasm-broker budget 1048576, used 1048560, rejected delta188 at virtual time471133454ns. There were2136 visible calls and6408 correctly charged transitions;1474 of1647 network polls remained pending. Every-pending-request before/after-idle scans explain amplification. Accounting matches the existing contract, so weakening charging or the bound is not a repair.

Use /home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/actual-diagnosis-r1.{json,md}. First establish safe invalidation/wakeup semantics across network mutation, time advancement, other incarnations, new requests, cancellation, close and deadlines. Implement only the smallest evidence-backed correct change, retain behavioral RED/GREEN regression, and prove the complete original TestReplayInspectionWorkflowFailure/TestPrototypeWorkflow canary with both one-minute timers/retries and required results under the original1MiB bound. Mere duplicate-scan elimination or a short prefix is insufficient. Preserve original failed attempts and identify diagnostic instrumentation separately.

This task receives the repair and full unchanged canary obligations previously included in .29. It does not replace .13 full qualification, .22-.25 receiving, or .14 reach. Required source/compiler/runtime/model/host identities and any preparation invalidated by the final repair must be authenticated afresh.

### Selected repair design

Contract decision after source/history review: retain explicitly active native per-owner bounds. Retired Go did not activate recording on either internal owner, but Rust93be185443 and current wasmhost/protocol.md:1017-1035 explicitly broadened that scope. Restoring inactive recording for new preparations would waive the current contract. A guest-only time/dirty/epoch cache is also unsafe because each visible Broker::check drains queued Link/Revoke commands.

Implement a separately declared/versioned, bounded, non-consuming batch readiness operation. The guest supplies pending request IDs in existing pending-list order, in bounded chunks. Each probe follows ordinary broker command processing and register/ready/quiesce accounting; the endpoint attaches and advances the authoritative network owner at broker time before answering. Reuse existing network owner generations/ready completions and per-incarnation delivery. Preserve every existing poll/query operation and its semantics. The guest collects only reported-ready results with existing individual polls in its existing wake order, asking the host again on every scan. Never infer unchanged readiness from equal virtual time or repeated payload bytes.

A controller command queued after a probe remains observable at the next ordinary poll/probe/idle boundary. Keep those boundaries explicit and test their consequences. Bound request/response sizes, validate empty/full/chunked/malformed/duplicate/foreign/stale IDs, and prove repeated probes do not consume completion results. Compare legacy scanning and batching for bytes/errors/wake order/virtual time across write, dial/accept/listen, close/half-close, cancel, deadline changes/expiry, delayed delivery, partition/heal, other-incarnation activity, revoke and restart. Use synchronization at declared boundaries.

Meaningful RED/GREEN: an actual linked cooperative guest with many pending reads, repeated scans, eventual completions and exact replay must fail under the original fixed budget before repair and reach expected results afterward. Assert remaining visible calls, including probes, retain ordinary three-transition accounting. Preserve active-bound refusal/retained-prefix/reproduction tests and old absent-field artifact semantics. Then run the complete original onebox canary; prefix savings alone cannot finish this task. Interception shim changes additionally require the existing offline shim harness per MILESTONES.

Authenticate all affected runtime-overlay/module/helper/model/source and preparation/install identities afresh. No old identity normalization or stale admissions. Preserve diagnostic binaries/evidence separately. Design rationale and exact source seams are retained in /home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/repair-contract-decision-r1.md and actual-diagnosis-scope-addendum-r3.md. The goal owner's instruction to follow recommendations supplies design-decision authorization; no additional user approval round is required.
## Acceptance
- [ ] Establish and document the correctness contract for avoiding redundant pending polls, including network mutations, virtual-time advance, other incarnations, new requests, cancellation, close, deadlines and wake order. Do not infer safety from repeated payload bytes alone.
- [ ] Retain behavioral RED/GREEN coverage for the supported repair; preserve broker transition accounting/conformance, modeled scheduling, deterministic replay and failure semantics.
- [ ] Run the complete original inspection canary under the unchanged1MiB transition bound, original timers/retries/assertions and expected exit/results. Retain actual commands, raw output and exact identity bindings. No budget increase, scenario shortcut, expected-error relabeling or unsupported disposition.
- [ ] Pass canonical worker and conductor merge gates and independent green-tree SHIP review before completion; record all changed identity/preparation implications. Temporary diagnostic instrumentation is not silently integrated.
- [ ] Keep .13 full96/final145 and .22-.25 receiving obligations pending until their actual fresh proof; existing .14 remains after .13.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
