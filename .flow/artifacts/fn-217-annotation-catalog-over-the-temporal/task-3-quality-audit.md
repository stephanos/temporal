# fn-217.3 quality audits

Scope: `1b06b2ff66a5926f731e29fe85661e04caed4a52..3d6d5b4a7d717213b782839d363fb2d135222bb1`, source checkpoint `7e5eb61c036f8299a1c7b4d16424a7f054104ddd`. The two read-only audits ran concurrently. Reports follow verbatim.

## Correctness report

## Quality Audit — Correctness axis: fn-217.3

### Summary

- Files changed: 32 · Critical 0 · Should Fix 0 · Consider 0 · Ship: ✅ Ship, subject to outstanding gates.
- No correctness findings. Reviewed `1b06b2f..3d6d5b4`; writer and reviewer are the same Codex family.

### Evidence

- Executed: diff inventory, `git diff --check`, frozen HEAD and clean working-tree checks.
- Walked: SDK retry interception, server validators, nested projections, redaction, protocol compatibility and freeze/allocation controls. Tests and full gates were not rerun by this auditor.

### Test Budget

- Raw added lines: 996 test : 1,376 implementation; excluding setup lowers the ratio further.
- Existing admission, catalog, generation, allocation and slice tests retain their assertions; slice changes remove sensitive diagnostic payloads.

### What's Good

- Walked: `tests/testpilot_model_annotations_test.go:37` validates before invocation, separates RPC errors and validates only successful responses; line 110 asserts retained background failures after shutdown.
- Walked: `model/temporal/realization/src/annotations.rs:25` dispatches annotations independently of frozen Judge state; `model/temporal/features/activity/api.rs:680` preserves response bytes. Diagnostics contain paths and constraint names without secret values.

## Standards report

## Quality Audit — Standards axis: fn-217.3

### Summary

- Files changed: 32 · Should Fix 1 · Consider 0 · Blocking: none possible (standards axis)

### Should Fix

- **model/temporal/features/activity/api.rs:1001** (Conf 100): New file crosses 1,000 lines (0 → 1,080), combining catalog declarations with transport dispatch and traversal — extract transport/traversal helpers into a sibling module while retaining the specified feature catalog in `api.rs`.

### Evidence

- **executed:** Read-only diff and line-count checks; exactly one file crosses the threshold. No tests, builds, gates, or repository writes performed.
- **walked:** Generic/Temporal boundaries, workspace dependencies, transport routing and modified tests; existing assertions remain intact. Default/required-TLC gates and timing remain pending; task completion is not established.

### What's Good

- **walked:** Rust owns annotation rules; Go transports envelopes and records failures. Dependencies reuse established workspace crates, and catalog reasons make infrastructure exclusions explicit.
- Reviewer and writer are from the same Codex family.

## Conductor disposition

Correctness: zero findings, worst tier none. Standards: one Should Fix, worst tier Should Fix. The conductor accepts the file-layout finding. The worker will retain the single feature catalog in api.rs and extract transport/traversal helpers after its frozen-source full gates and timing finish. No assertion, catalog entry, source-field obligation or live scenario may be removed. Formal implementation review and integrated verification remain pending; this report does not close the task.

## Standards follow-up report

## Quality Audit — Standards follow-up: fn-217.3

- Files reviewed: 2 · Should Fix 0 · Consider 0 · Previous layout finding: resolved.

**executed:** Compared `7e5eb61c03..56aa9bd`; `api.rs` is now 630 lines and `api_transport.rs` 483. Catalog declarations and supplemental catalog factories compare unchanged. Extracted transport/helper bodies differ only in three `pub(super)` visibility changes and formatting. No test or snapshot files changed; no assertions weakened. Integrated files match the reviewed fix.

**walked:** Imports, module wiring and public re-exports preserve existing call paths.

No Cargo/Go tests or gates run by this auditor. Worker regates and timing remain pending; this report does not establish task completion.

## Additional conductor finding

Should Fix. The supplemental `start_request_id_intent`, `cancel_request_id_intent` and `worker_failure_presence` factories construct request constraints/rules through raw builders, outside the macro's compile-time admission. Removing their expectation still compiles and fails only during runtime admission, contrary to task .3's compile-admission requirement. The worker confirmed this gap. The conductor accepted a private checked constructor/macro requiring a typed expectation and nonempty literal covers, with negative compile controls. This keeps UUID Accepts intent separate from length Rejects semantics and preserves the conditional worker-failure context. No new public DSL or duplicate descriptor declarations are authorized by this correction. Resolution and final verification remain pending.

## Supplemental correctness follow-up report

Supplemental-admission finding resolved. Critical 0 · Should Fix 0 · Consider 0.

- Walked: `model/temporal/features/activity/api_requests.rs:18` requires a typed expectation and literal provenance for both constructor forms; line 14 rejects blank provenance during compilation. Field accessors enforce `String` or `Option<Vec<u8>>`.
- Walked: `model/temporal/features/activity/api.rs:565`, line 582 and line 598 preserve UUID `Accepts` and conditional failure-presence expectations. Existing byte-limit `Rejects` metadata and transport semantics remain unchanged. The macro is private and adds no descriptor declarations.
- Cited: `.flow/tmp/fn2173-supplemental-final.log:53` records all three negative controls passing; the same log records 25 native tests and Clippy completing successfully.
- Executed: scoped diff inspection, `git diff --check`, frozen HEAD and clean working-tree checks. No tests, gates or writes performed by this reviewer. Writer/reviewer remain the same Codex family.

## Resolved findings at final source checkpoint

Worker source `55a27ddd024a20ea94c40f0209eeb7d6408794a4` is integrated at `53d51ac71cdd52365a86e4e029ee741a47b8ad07`. The layout finding and supplemental-admission finding are resolved by their focused follow-ups above. The conductor independently read the normal, non-overwrite compile-control rerun and its three passing fixtures, alongside all 25 native tests and Clippy. Final live testing also passed on that source. Final full gates, integrated verification and formal review remain outstanding; no task closure is claimed.

### Correctness follow-up: formal review failure-field coverage fix

Focused correctness follow-up: P2 finding resolved. Critical 0 · Should Fix 0 · Consider 0.

- Walked: SDK v1.48.0 `FailureToError` and `DecodeCommonFailureAttributes` against the catalogs. Consumed failure fields are typed, and nested payloads, causes, activity types and workflow values are traversed in `model/temporal/features/activity/api_transport.rs:268` and line 424.
- Walked: `model/temporal/features/activity/api.rs:398`, line 406 and line 418 validate nested retry enums. Stack trace, operation token and fallback operation ID use actual field setters for copy redaction. Response validation does not redact or mutate transport bytes; request expectations remain unchanged.
- Cited: recorded RED then native GREEN evidence covers nested discriminant findings, exact response-byte retention and secret-free diagnostics. Clippy and Go compilation logs completed successfully.
- Executed: four-file diff inspection, `git diff --check`, frozen HEAD and clean-tree checks. No builds, tests or writes performed here; live/full gates remain outside this follow-up’s evidence.

Writer and reviewer are the same Codex family.

Correctness follow-up: 0 findings; no remaining tier. The conductor accepts the focused resolution at source `5c9d19cca65f6920ceb8fd988d107f4eb866618f`, integrated as `99527a1f3b4e3a4bd27b2412b045019f8a7a0e99`. Final full gates and single-session re-review are still required.
