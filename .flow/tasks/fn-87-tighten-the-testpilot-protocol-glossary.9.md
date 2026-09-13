---
satisfies: [R6]
---
# fn-87-tighten-the-testpilot-protocol-glossary.9 One opaque-handle encoding, natural values, wire EntrypointKind and scalar kinds

## Description
Finish R6 on the Program/value side: keep one opaque-handle encoding, remove `Value.natural_value` unless a Producer needs a value no other arm represents, remove the wire `EntrypointKind` (the Go runtime keeps its own classification), and fold the wire-encoding scalar kinds or record them as kept with the reason.

**Size:** M
**Files:** `proto/.../v1/{value,program}.proto`, `api/testpilot/v1/*`, `common/testing/testpilot/internal/ir/{type.go,type_test.go,path.go,expression.go,evaluate.go}`, `common/testing/testpilot/internal/execution/{prepare.go,projection.go,dataflow.go,carrier.go,values.go}`, `common/testing/testpilot/internal/verification/{correlated.go,correlated_prepare.go}`, `common/testing/testpilot/contract/{driver.go,profile.go}`, `common/testing/testpilot/{driver.go,contract.go?}` (facade aliases), `common/testing/testpilot/temporal/{profile.go,driver.go}`, `common/testing/testpilot/temporal/internal/{delivery/ledger.go,activation/activation.go}`, `common/testing/testpilot/temporal/{server/driver.go,worker/carrier.go}`, `model/Testpilot/Authoring.lean` (`Value.natural`, `Types.opaqueHandle`), Lean Producers using naturals, fixtures, mapping
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, common/testing/testpilot/**, model/Testpilot/**, model/Umpire/Case/**, model/Temporal/**, tests/testcore/testpilot/**, tests/testpilot_*_test.go]

### Approach
- Opaque handle: keep `Slot.opaque_handle` (the encoding fixtures use) and remove `SingularType.opaque_handle`, which would otherwise admit handles inside repeated fields, maps, Observations and captures, each needing its own rejection. `ir/type.go:146` sets the opaque flag from the type arm, synthesized from the slot arm at `execution/prepare.go:314-315`; carry the opaque flag from the Slot declaration directly. `temporal/server/driver.go:199` reads the slot arm and stays.
- Natural: the runtime emits naturals for correlated evidence (`execution/projection.go:268-279`) and correlated typing reads them (`verification/correlated.go:171,249-250`, `correlated_prepare.go:212,239-240`, `ir/type.go:312`, `ir/evaluate.go:156`, `ir/expression.go:486,502`). Recommended: remove `natural_value` and `SCALAR_KIND_NATURAL`; the runtime projection and Producers use `unsigned_integer_value` typed `UINT64` (same canonical base-10 text), and `ir` type equality treats them as one kind. A comparison between a former natural and an unsigned integer must not start rejecting anywhere. Keep it only if a Lean Producer needs an unbounded natural above 2^64-1 that it cannot bound-check (see the `.flow/memory` entry "Check unbounded Lean numbers before protobuf narrowing"); record the decision and reason.
- Wire `EntrypointKind`: add a Go `EntrypointKind` in `common/testing/testpilot/contract` (the Driver-contract leaf), re-exported by alias from the facade as the MOD-14 restatement requires, with a single classifier from the activation oneof; replace the ~114 wire-enum uses across 25 Go files (`contract/driver.go:129`, `contract/profile.go:36`, `driver.go:61`, `execution/dataflow.go:41`, `execution/carrier.go:37-104`, `execution/values.go:74-78`, `temporal/profile.go:58-186`, `temporal/driver.go:121`, `temporal/internal/delivery/ledger.go:252-365`, `temporal/internal/activation/activation.go:36`, `temporal/worker/carrier.go:65,74`). The Go constant names must not spell `ENTRYPOINT_KIND_` so the retired prefix can be gated.
- Scalar kinds: `ir/type.go:44-45` compares types with `proto.Equal`, so a slot typed `INT32` does not match a read from a `sint32` field; `ir/path.go:194-221` maps protoreflect kinds one-to-one; `ir/type.go:317-354` checks bit widths. Per the spec's Edge Cases, fold only if descriptor admission does not need the kinds to match a field's declared type. Recommended outcome: kept, with the reason recorded in the done summary and as a comment on `ScalarKind` in `value.proto`. If you fold, prove no admission outcome changes in any test or fixture.
- Mapping steps: `naturalValue` → `unsignedIntegerValue` and `SCALAR_KIND_NATURAL` → `SCALAR_KIND_UINT64` (if removed); drop `singular.opaqueHandle` types (none expected in fixtures). Retire `ENTRYPOINT_KIND_`, `natural_value`/`naturalValue`/`NaturalValue` and `SCALAR_KIND_NATURAL` (if removed).

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/ir/type.go:40-160,300-360`, `path.go:190-225`
- `common/testing/testpilot/internal/execution/projection.go:260-285`, `prepare.go:300-320`
- `common/testing/testpilot/contract/profile.go`, `common/testing/testpilot/temporal/profile.go:50-190`
- `.plans/UMPIRE4_SPEC.md` MOD-14 restatement

**Optional:**
- `model/Testpilot/Authoring.lean:20-95` — value and type constructors
- `common/testing/testpilot/temporal/internal/delivery/ledger.go:250-370`

### Key context
- The facade/leaf split from fn-84 .2 is enforced by tests that compile unedited facade code; add aliases, do not duplicate declarations.

## Acceptance
- [ ] one opaque-handle encoding remains (`Slot.opaque_handle`); inspecting an opaque slot still rejects
- [ ] `natural_value` removed (or kept with a recorded Producer need); runtime correlated evidence uses the remaining arm; correlated Verdict pins unchanged
- [ ] wire `EntrypointKind` removed; the Go classification lives in the `contract` leaf with facade aliases; no Go identifier spells `ENTRYPOINT_KIND_`
- [ ] scalar-kind decision recorded with its reason (and as a proto comment if kept)
- [ ] equivalence test passes with declared steps; retired tokens added; `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
