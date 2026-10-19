---
satisfies: [R2]
---
# fn-216-mini-activity-vertical-slice-the-model.2 Oracle judge/answer/reset with the minimal Temporal extension; guest and Go host

## Description
Rename the Model interface: `umpire.oracle.v1` becomes `umpire.model.v1` (`model/umpire/proto/umpire/model/v1/model.proto`, Go package, Rust module, host paths; ABI, method IDs and message bytes unchanged, proven by the existing parity tests). Add `reset`, `judge(Observation) -> Judgment`, `answer(Callback) -> Answer`. `model/temporal/proto/umpire/temporal/v1/temporal.proto` holds exactly three envelopes, `Observation { method, response: Any }`, `Callback { task: Any }`, `Reply { request: Any }`, over `temporal.api.*` messages (`DescribeActivityExecutionResponse`, `PollActivityTaskQueueResponse`, `RespondActivityTaskCompletedRequest`, `RespondActivityTaskFailedRequest` for the slice); the Temporal guest compiles the API types with prost from `proto/api.binpb` (prost-build's descriptor-set input; no `.proto` sources needed), Go uses `go.temporal.io/api`. Nothing is copied from Testpilot's proto; nothing Temporal enters `model/umpire` (owner). `judge`/`answer` carry `google.protobuf.Any`; the Temporal guest's registry decodes, unknown type URLs are a `Refusal`. `temporal.rs` (from .1) maps RPC messages to actions and answers with partial requests (only the fields the Model decides), stated in its doc comment. `umpire-wasm` becomes a library (ABI, buffers, core dispatch, a hook for a system guest's Models and registry); the Temporal guest `model/temporal/wasm` links it and hosts the mini activity Model with a per-run instance, replays observations through `Machine::step` for `judge` and answers callbacks from its state; the Go host (`model/umpire/hosts/go`) gains `Reset`, `Judge`, `Answer` over the core messages only; both generated with the existing generators, byte parity tests as for explore/rows. A Temporal-extension message other than the three envelopes is a review finding. After .1. Add the gate check (R5) to `umpire-testing` and `make model-check`. **Touches:** `model/umpire/proto/**`, `model/umpire/wasm/**`, `model/umpire/hosts/go/**`, `model/umpire/testing/**`, `model/temporal/proto/**`, `model/temporal/wasm/**`, `Makefile` model section (exact minimal replacement), `model/README.md`, `MIGRATE.md` (umpire and temporal sides named). **Feedback loop (R30):** paste `make model-time`.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
