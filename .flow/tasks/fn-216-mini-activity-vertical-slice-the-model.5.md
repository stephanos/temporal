---
satisfies: [R2]
---
# fn-216-mini-activity-vertical-slice-the-model.5 Reusable stateful realization core and ABI library

## Description
Land R2's generic realization, per-run state and reusable ABI library. This consumes
the core protocol checkpoint and leaves all system payload decoding to .6.

**Size:** M
**Files:** reusable WASM library, stock Kettle binding and generic stateful controls.
**Touches:** [model/umpire/wasm/**, model/umpire/tests/fixtures/shared/**, model/umpire/hosts/go/**, model/Cargo.toml, model/Cargo.lock, Makefile, model/scripts/**]

### Approach
- Separate the embedded Kettle binding from system-agnostic ABI/buffer/dispatch mechanics, retaining the current stock guest and its byte-identical reports/rows. A system guest must link the reusable library without duplicate exports or mutable instance state.
- Put the generic Realize interface and stateful engine beside the reusable core unless an existing module provides the same seam. Consume Machine generically; introduce no Temporal vocabulary or imports and no second transition interpreter.
- Represent positive current-state projection separately from transition candidates and unsupported input. Judge applies actual Machine::step results; zero actions explain an observation only after a positive projection match. Expected actions reflect enabled Model actions.
- Make callback handling transactional. Realization validation and response selection must succeed before callback-evidenced transitions commit. Answer cannot claim an unobserved terminal result. Keep each instance's state and first Unexplained marker private; freeze future advancement/Answer until Reset.
- Preserve state on typed refusal, trap and response encoding/allocation failure. Follow existing exploration rollback and buffer ownership behavior rather than weakening it for new operations.
- Use generic test Models/opaque synthetic payloads for reusable engine controls. Pin repeated current observations, unknown projections, impossible transitions, zero-step encoding, reset/frozen behavior, instance isolation and failure rollback. Keep system-policy tests for .6.
- Wire focused native and WASM tests into the existing model gate. Serialize manifest/lockfile/Make edits and keep all prior ABI/refusal/parity controls green.

### Investigation targets
**Required:**
- `model/umpire/wasm/src/lib.rs:12` - embedded binding and generic extraction seam.
- `model/umpire/wasm/src/lib.rs:107` - buffer owner and rollback controls.
- `model/umpire/wasm/tests/dispatch.rs` - native protocol dispatch harness.
- `model/umpire/dsl/src/machine.rs` - canonical Machine transition interface.
- `model/umpire/hosts/go/host_test.go:359` - exhaustion preservation controls.
- `Makefile:1424` - model gate and stock guest build.

### Quick commands
From model/, run the reusable core's focused tests through mise. Run CGO_ENABLED=0 mise exec -- go test ./... in the generic host module, then make model-check and make model-time from the root.

## Acceptance
- [ ] A system guest links the reusable ABI/dispatch library while the stock Kettle guest retains its exact exports and canonical report/rows bytes.
- [ ] Generic Judge/Answer/Reset replay Machine transitions, distinguish positive zero-step matches, and preserve the parent contract's per-run/frozen-state semantics.
- [ ] Native/WASM controls pin malformed/refused/unexplained inputs, reset/isolation and response-allocation failure without partial mutation or ownership leaks.
- [ ] Generic code contains no system vocabulary; focused controls, host parity and the full model gate pass with recorded timing.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
