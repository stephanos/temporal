# Keep Start-only limits out of nested failure metadata

Conductor follow-up at source `1691aa14ce849d628a752bafcec34212248ef0c3`
identified an unintended constraint introduced by the failure-field coverage fix.
`ActivityFailureInfo.activity_type` traversed the shared ActivityType request
catalog, whose name limits cite the CHASM Start validator. Those limits do not
apply to a type name carried as nested failure metadata.

The frozen-source probe crossed the real Wasm host and pinned SDK converter.
Names of zero and 1,001 bytes decoded in the SDK but produced `string.min_len`
and `string.max_bytes` findings in both RespondActivityTaskFailed requests and
PollActivityExecution responses. A two-byte control produced no findings;
every byte envelope remained unchanged. The probe itself does not claim live
server RPC success. Server source separately confirms that the worker handler
checks the top-level application-failure alternative and recursively validates
application retry delays, not nested activity-type names.

Disposition: use a typed optional String projection of the actual nested name
as observed Text in this failure context. Keep the shared ActivityType catalog,
its Start min/max constraints and qualified expectations unchanged. Preserve
all failure enum checks, secret handling and exact request/response bytes.
Add regression cases for both nested contexts and retained Start enforcement.
No ignored aggregate, duplicate descriptor catalog, protocol change or gap
waiver is needed. Final-source gates and the existing single-session re-review
remain required; the task is still in progress.

Active evidence: `.flow/tmp/fn2173-reviewfix-type-context-probe.go` and its
`.log`, plus the pinned SDK/server source trace returned by the worker.

## Focused correctness follow-up

Focused correctness follow-up: context finding resolved. Critical 0 · Should Fix 0 · Consider 0.

- Walked: `model/temporal/features/activity/api.rs:396` projects the actual nested name as `Option<String>` with unconstrained Text. Only `ActivityFailureInfo`’s shared-type traversal was removed. Start’s Identifier/1,000-byte constraints remain at line 183.
- Walked: consumed failure values remain typed; retry enum checks, secret setters, request expectations and transport byte preservation remain intact.
- Cited: `model/temporal/realization/tests/catalog.rs:205` covers empty and 1,001-byte names in both failure requests and responses, asserts unchanged bytes, and retains Start rejection checks. Recorded RED/GREEN logs show the regression failing before the fix and passing afterward, alongside 27 native tests, three compile-failure controls and Clippy.
- Executed: scoped three-file diff inspection, `git diff --check`, frozen HEAD and clean-tree checks. No tests, builds or writes performed here. Final live/full gates are not established by this follow-up.

Writer and reviewer are the same Codex family.

Conductor disposition: accepted resolution at source `a23540b5446e834df6fc1e478aae093d991fb38f`, integrated as `7f5479c34690ebd2d06da34f41035134906a7684`. This follow-up has zero findings and no remaining tier; task closure still requires final verification and formal re-review.

## Related request-enum acceptance finding

Before final re-review, a real-guest diagnostic found another request/response
distinction: an undefined Activity retry-state reaches the SDK converter but
produces annotation findings in both directions. Source inspection identified
related Child retry-state, Nexus retry-behavior and timeout enum cases for the
worker to verify separately. The inspected server failure validator checks
application retry delays, not nested enum membership. Request-side expectations
must faithfully describe accepted values while response validation retains its
findings. The existing per-field acceptance handling distinguishes directions;
no blanket constraint filter or invented server rejection is appropriate.

The diagnostic completed before disk cleanup; source remained frozen at
`a23540b5446e834df6fc1e478aae093d991fb38f`. Evidence is retained in
`.flow/tmp/fn2173-reviewfix-nested-request-enum-probe.go` and its `.log`.
This finding is open pending the worker's narrow fix and regression controls.

## Focused enum follow-up

Focused enum follow-up: resolved. Critical 0 · Should Fix 0 · Consider 0.

- Walked: all four fields retain enum validation with qualified `Accepts` expectations. `api_transport.rs:173` suppresses only request findings explicitly marked `Accepts`; response findings remain. Start/shared constraints are unchanged.
- Walked: `service/frontend/workflow_handler.go:1835` recursively checks application retry delays, while the failed-request handler requires top-level application info without rejecting these nested enum values. CHASM’s failure path introduces no nested enum rejection.
- Cited: `model/temporal/realization/tests/catalog.rs:150` covers all four request acceptances, response findings, exact byte preservation and secret-safe diagnostics. The focused log records 27 native tests, three compile-failure controls, five API/descriptor tests and Clippy passing. RED evidence demonstrates only the Activity retry-state case.
- Executed: two-file diff inspection, `git diff --check`, frozen HEAD and clean-tree checks. No builds, tests or writes performed here; live/full gates remain outside this evidence.

Writer and reviewer are the same Codex family.

Conductor disposition: resolved at source `9baf1f772e79f4332fb20906fc5b7af0432224f4`, integrated as `f65a1f356851fc3885be946f6c136e31b64c9ceb`. This follow-up has zero findings and no remaining tier. Final full gates, timing and single-session formal re-review remain required.
