---
satisfies: [R1, R5]
---
# fn-71-standalone-lean-testpilot-protocol.1 Prototype the exact Testpilot schema with Lean protobuf

## Description
Prototype the approved `Lean-zh/protobuf` package against the repository's exact Testpilot schema before committing the migration. Pin one reviewed revision, exercise Lean 4.33.1 and the repository-supported `protoc`, and record cold/warm build time plus executable-size impact. This is a hard adoption gate: tasks 2-7 stop for re-planning if any required schema, resolver, determinism, or Go-interoperability behavior is missing or unreliable.

**Size:** M
**Files:** `model/lakefile.toml`, `model/lake-manifest.json`, `model/Testpilot/ProtocolPrototype.lean`, `model/TestpilotTests/ProtocolPrototype.lean`, `model/TestpilotTests/ProtocolPrototypeMain.lean`, `Makefile`, `tests/testcore/testpilot/protobuf_lean_interop_test.go`
**Touches:** [`model/lakefile.toml`, `model/lake-manifest.json`, `model/Testpilot/ProtocolPrototype.lean`, `model/TestpilotTests/ProtocolPrototype*.lean`, `Makefile`, `tests/testcore/testpilot/protobuf_lean_interop_test.go`]

### Approach
- Pin the reviewed package revision and use its supported `PROTOC` override rather than relying on an ambient compiler.
- Load or generate only the eight-file Testpilot import closure rooted at `proto/internal/temporal/server/api/testpilot/v1/case.proto` plus required well-known types.
- Construct exact generated values covering recursive Program/Contract expressions, oneofs, proto3 message presence, enums, empty and non-UTF-8 bytes, integer boundaries, and `google.protobuf.Any`.
- Serialize repeatedly with `Protobuf.Json` using explicit print options and the generated-pool resolver; feed the emitted file to a focused Go test using `testpilot.DecodeCaseProtoJSON`. Ordinary Go tests must not invoke Lean.
- Record package revision, generation mode, `protoc` version contract, cold/warm build timings, and executable sizes in the task evidence. Remove or promote prototype-only modules during task 2.

### Investigation targets
**Required** (read before coding):
- `model/lean-toolchain`
- `model/lakefile.toml`
- `proto/internal/temporal/server/api/testpilot/v1/case.proto` and its seven local imports
- `common/testing/testpilot/case.go`
- `tests/testcore/testpilot/artifact_test.go`
- Lean-zh/protobuf v0.4.0 README and ProtoJSON API at reviewed revision

**Optional** (reference as needed):
- `Makefile:1000-1105` generation/check patterns
## Acceptance
- [ ] A pinned Lean protobuf revision builds the exact Testpilot closure with Lean 4.33.1 and the selected repository `protoc`.
- [ ] The prototype covers recursion, oneofs, presence, enums, bytes, signed/unsigned boundaries, and resolved `Any` without handwritten wire types.
- [ ] Equal generated values emit byte-identical ProtoJSON on repeated runs, and the focused Go test strictly decodes the actual Lean output.
- [ ] Unknown `Any` type URLs and malformed payloads return errors; no field or value is silently omitted.
- [ ] Cold/warm build time and executable-size deltas are recorded, with an explicit adopt or re-plan result.
## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: