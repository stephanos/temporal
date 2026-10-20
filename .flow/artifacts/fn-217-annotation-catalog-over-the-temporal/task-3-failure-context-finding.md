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
