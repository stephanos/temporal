---
satisfies: [R1, R4]
---
# fn-180-gomad-replay-time-instrumentation-for.1 Debug side-channel import, replay request options and pre-execution refusals

## Description
Add the dedicated debug import and replay request plumbing for R1/R4. Resolve target/stop coordinates and names from authenticated retained metadata before launching execution. Extend CLI/provider/Runner reports with requested/resolved stops, instance/incarnation and a dedicated `inspected_prefix` result; normal full replay retains complete-tape validation. Inspection must not use `ModePrefix`, which continues with seeded choices.

Tasks .1–.3 are one coupled runtime/library/prefix batch. Implement the spec's bounded, versioned registration descriptors, exact tape-decision/trace-record/shared-order mapping, per-incarnation and World prefix material/cuts, and compulsory allocation/runtime-state witness independently of optional diagnostics. Existing final WASI/network/storage hashes are insufficient. Include relevant stdout/stderr, lifecycle and storage/recovery state; preserve clock-run folding and existing network payload guarantees. All metadata is retained while probes/dumps are dormant, authenticated in existing artifact/source identities and strictly bounded/decoded/cloned. Integrate against fn-178's final incarnation-aware lifecycle/client without changing delayed starts, holds, streamed terminals or Finish.

Own `wasmhost/src/{abi,protocol,engine,model,runtime_typed}.rs`, `wasi/{runtime,process}.go`, cluster retained/provider/replay/model evidence, artifact capture, `internal/runner/replay_operation.go` and CLI plumbing as needed. Dedicated debug imports bypass ordinary runtime dispatch and `close_clock_run`, use already-held clock/coordinate state and never alter canonical counters, transcripts, clocks or scheduling. Metadata exhaustion/incompleteness and unknown names fail before debug execution. Keep guest library implementation with .3 and runtime handoff with .2, developing their shared seams in this batch.
## Acceptance
- [ ] Debug import never yields or reads clock and remains outside transcript/choice/evidence identity
- [ ] Single-guest inference and required cluster instance/incarnation selectors follow R1; missing, unknown and ambiguous selectors refused
- [ ] Preflight validates local ordinal/range, recorded time/failure boundary and target-scoped dump/probe names; missing metadata/evidence refused without executing
- [ ] CLI and reports carry target, requested and resolved stop; debug bytes bounded with truncation markers outside the artifact
- [ ] Recordings never enable probes/dumps; replay without debug options preserves evidence
- [ ] Bound/authenticate retained registry, exact decision/trace/global coordinates and prefix material; final hashes alone never authorize arbitrary-prefix verification
- [ ] All executed incarnations, World/lifecycle order, output and relevant storage/recovery state have verified cuts; clock-folding and composed network guarantees preserved
- [ ] Compulsory allocation/runtime-state witness is available without `--diagnostics`; incomplete/older/corrupt metadata is refused before execution
- [ ] Dedicated debug import bypasses `close_clock_run`; clock-run fixture proves debug calls cannot perturb canonical WASI bytes
- [ ] `inspected_prefix` is distinct from full reproduction, incomplete tapes and seeded-continuation `ModePrefix`; ordinary exact-replay consumption checks unchanged
- [ ] Metadata/protocol/source identity inventories, strict decoders, cloning and byte bounds updated together; final fn-178 client/lifecycle semantics preserved
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
