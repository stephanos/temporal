---
satisfies: [R2, R3, R10, R11]
---
# fn-228-gomad-helper-owned-replay-tape.2 Helper preflight: accept trace frames, verify digests and counts, refuse before guest start

**Size:** M
**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/model/guest_state.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/src/namespace/transport.rs, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/cluster.go, tools/gomad/internal/backends/wasm/wasmhost/src/cluster/execution/transport.rs, tools/gomad/internal/backends/wasm/wasmhost/src/cluster/execution/mod.rs, tools/gomad/internal/backends/wasm/wasmhost/src/cluster/execution/manual.rs, adjacent standalone/cluster transport and replay tests

## Description
Replace `tape_chunks` in the helper model config with a versioned replay request: `{schema, mode, limit, execution_identity, trace_sha256, tape_sha256, decisions, trace_bytes, trace_frames}`. Implement the spec's new pipe framing explicitly; current transport is LF JSON only. Use one buffered reader, checked byte/count admission, four-byte big-endian frame lengths, 64-KiB chunks and the required JSON end marker. Read this block at the declared replay-input point before instantiation, then hash and compare against `trace_sha256`, project through the module from task 1, compare tape digest and decision count, and on failure return a preflight reply `{error:"replay_preflight", check}` before any instantiation. Keep `mode` 1 (record) rejecting any trace. On the Go side, `helperConfig` emits the new request and `process.go` sends the trace payload as frames instead of JSON decision chunks; unknown schema versions are refused with both versions named. Implement exact-versus-prefix admission and helper-side cutoff/final-rank controls per the spec; exact mode rejects controls. Replace cluster initial and start/restart runtime `tape_chunks` with this admission data. Remove the LF split/re-tag of binary tape bytes in `wasi/cluster.go`; adapt `TaggedInput` and `read_instance_tape` to consume the attributed transaction atomically, with node/incarnation in declaration and end marker. Keep the per-decision comparison in `imports.rs` unchanged after admission.

## Acceptance
A replay with a tampered trace byte, a wrong expected tape digest, a wrong decision count, an oversized, zero-length, truncated or extra frame, wrong total/count/end marker, arithmetic overflow, or a mismatched schema version fails preflight with the named check and the Runner's existing status code; a valid replay starts the guest; record mode with a trace is refused. Test payloads containing LF and arbitrary binary bytes, exact chunk boundaries and zero-byte replay; none of the invalid cases may instantiate or start a guest. Exercise actual standalone and cluster exact/prefix replay, including manual start/restart, shortened prefixes and forced rank, LF bytes inside binary payloads, wrong/stale attribution and malformed frames.

## Done summary
R2/R3/R10/R11: Added helper-owned preflight for raw trace framing, digest/count/tape/schema checks, typed exact/prefix/rank admission and atomic node/incarnation transactions. Per-decision comparison unchanged. Retained22actual-helper refusal cases,17terminal negatives, fivecluster transaction cases and fourfresh nonzero oracle comparisons pass; malformed input cannot start guest. Combined source/helper Go gates at0561 passed24packages/1133top-level tests. Helper71inputs/digest remain unchanged by later Go-only fixes. Independent review SHIP07ebfdb4ee. Final Rust runtime qualification pending232R12/.13.

Evidence index: /home/agent/wt/evidence/fn232/fn228-replay-inputs-2026-10-20/FINAL_CUSTODY.md; original producer handovers and narrow review-fix appends remain authenticated independently. Source acceptance only; spec closure requires separate completion review.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1ef7bd73b5, 0561e32514, 95caeab345
- Tests: actual standalone/cluster preflight and terminal negative suites, retained raw receipts, combined go test -tags test_dep ./internal/runner/... ./internal/backends/wasm/... exit0, canonical Quick and serial read-only changed-source lint exit0
- PRs: