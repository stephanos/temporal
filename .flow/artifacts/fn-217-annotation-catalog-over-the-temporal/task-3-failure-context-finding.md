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
