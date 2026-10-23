---
satisfies: [R2]
---
# fn-228-runner-and-run-the-one-go-host-loop.2 After .1: Propose on the protocol, Runtime, Realize and Host; activity Realization proposes from the world

## Description
Add METHOD_PROPOSE, ProposeRequest { seed }, ProposeResponse { request, action }, Response.propose and REFUSAL_CODE_QUIESCENT to umpire.model.v1; regenerate Rust and Go. Add Realize::propose returning Proposal { action, request } and the Runtime's pending-proposal state (STATE refusal on a second Propose, cleared by the explaining Judge, frozen on Unexplained). Deterministic for the same state and seed; internal actions and Deadline never proposed. Implement it in the activity Realization: start request filled from the world (fn-227), cancel, and worker responses; wire goldens for the method; negative control that a disabled-action proposal is caught. Host.Propose in Go with parity tests.

### Required integration contract

Implement the parent bootstrap/admission, pending-state, incremental Judge completion and stateful control-guest contracts. Reserve METHOD_PROPOSE=6 and Response.propose=9; preserve reserved Response tag 7 and Annotate tag 8. Add ProposalKind and JudgeResponse.continue_polling at fresh tag 3. Realize defaults preserve existing bindings. Negative controls cover failed publication, failed/crossed Start, pre-admission callbacks, disabled actions, intermediate polls and second pending proposals. No Start Model action.


### Conductor settlement (2026-10-23)

Use additive ResetRequest.configuration Any at tag1 for a typed StartActivityExecutionRequest; absent retains the old zero-byte Reset wire. Configuration explicitly opts into bootstrap behavior. Once opted in, Reset without configuration reuses the selected configuration while clearing admission, receipt, pending, freeze and callback context for a new Run; configured Reset replaces configuration atomically and clears the same run effects. Unconfigured legacy realizations keep existing Reset behavior. Receipt immutability applies within a Run.

Use a default-refusing Realize::matches(request, observation) correlation hook. Generic Runtime keeps payloads opaque; Temporal checks actual invocation and receipt identities, including correlated Describe reads. A typed Temporal Realization context may retain validated callback/task-token data only after successfully published Answer, reusing the existing admission result. Check run, attempt and current Model state before proposal; invalidate stale context on reset, re-admission or corresponding final observation. Do not fabricate tokens or introduce a Go chooser. Preserve immediate Answer reply behavior and ensure proposal support does not require duplicate worker-response RPCs.

All configuration, capability, receipt, context and pending changes share existing prepare/commit publication atomicity. Add failed-publication controls, callback-between-polls controls and explicit Start correlation controls. This implements the approved parent contract without changing the canonical activity Model or its behavioral goldens.

Focused baselines only: the predecessor's default and required full gates are already verified at committed closure ac2fd68e0e. Final task gates still include explicit post-edit default make model-check and timeout3600 UMPIRE_BACKENDS=require make model-time, fresh guest/live/privacy proof, review and paired conductor audits. The global model capacity error interrupted planning before implementation; preserve any unrecoverable baseline observation as inconclusive and recover only after confirming its process terminal.
## Acceptance
- [ ] Propose on proto, Runtime, Realize, Host with wire goldens
- [ ] Activity Realization proposes start/cancel/responses deterministically per seed
- [ ] QUIESCENT and pending STATE refusals pinned; disabled-action control
- [ ] Gate and Go host parity pass

## Done summary
Implemented R2 Propose across the protocol, reusable Runtime/Realize, Temporal realization and Go Host. Configured bootstrap admission and operation-correlated polling now commit atomically with publication, while canonical Model semantics remain authoritative.

Propose never steps the Model. Explicit Start configuration opts into bootstrap; subsequent empty Reset retains that configuration while clearing the prior Run's admission, receipt, pending request/action, freeze and callback context. Only actual matching Start evidence admits a Run. Published Answer replies supply validated callback context; observation instructions never resend those replies. Intermediate reads preserve pending work, completed operation evidence clears it, and Unexplained freezes. Pending Cancel requires its exact invocation; delivery completion includes a validated later attempt when a read missed Started.

Validation covers publication rollback, disabled/internal proposals, quiescence, pre-admission callbacks, stale/crossed evidence, reset reuse, callback-between-polls, completion races, and native/Wasm/Go parity. Existing wire vectors remain unchanged; the new Judge bool/oneof representation pins each legal encoder order and cross-decodes semantic equality. The canonical activity Model, DSL/checker source and protected semantic goldens are unchanged across all 1002 protected paths.

Baseline: focused Rust and Go suites passed before source edits. The first broader Rust baseline's lost host handle is preserved as inconclusive; the first Go launcher failed before Go because of its cwd, and the corrected launcher passed. The conductor explicitly reused the predecessor's full baseline; no fabricated green observation or gate skip is claimed.

The CLI review found two operation-completion defects; both reproduced red in native and Wasm controls, were corrected, and passed re-review. Independent correctness audits have no remaining findings. The standards snapshot-size finding is resolved by exhaustive complete-item snapshots, preserving every expansion item. Final focused tests, strict Rust formatting/clippy/docs, protobuf freshness and both nested Go lint/parity suites passed. Stock Kettle Propose remains Unsupported.

The rebuilt canonical guest SHA-256 is 23dceb5e96baf46db32f4496a0db0cf1c7420d10e4847b1ef67899b03f97f1d4. Fresh actual slice and gaps/privacy proof passed (5.875s and 359.201s; 2336 probe passes). Review, audit, protected-source and live verification records are preserved under .flow/artifacts/fn-228.2/.

Final full gates passed once each after source/review settlement: `UMPIRE_BACKENDS=require make model-time` (all 18 stages, rc0, 877s elapsed) and `env -u UMPIRE_BACKENDS make model-check` (rc0, 392s elapsed). The timed gate measured 870.575s against the 60s soft target; the inner loop measured 6.109s against the 10s target. Both green receipts were recorded. Detailed stage timings are preserved in .flow/artifacts/fn-228.2/final-gates.json; no full-gate repetition or gate weakening was used.

Tier session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-23T08:01:05Z..2026-10-23T08:18:39Z] (codex gpt-6.1-sol high; same family; NEEDS_WORK -> SHIP)

Captured the nontrivial correlation/completion fix in bug/integration/correlate-pending-proposals-with-2026-10-23. Runner/Run schema and commands remain subsequent task scope; no production Runner or Go chooser was introduced.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0a91a7b99df2e43a73eb79209a67208bc12539bb, d207e950283bdece61c9a4f09d50f8487d74b7fe, 9e30e3602d312d908d2ccef0d058fcd4f96f732f, deb9de2ed45008b9311a3b8b977a52ba651f8c0f
- Tests: UMPIRE_BACKENDS=require make model-time, env -u UMPIRE_BACKENDS make model-check, make umpire-check-model-slice && make umpire-check-model-gaps, CGO_ENABLED=0 mise exec -- go -C /Users/stephan/Workspace/skunkworks/umpire/temporal/model/umpire/hosts/go run /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-228.2/stock-propose.go
- PRs: