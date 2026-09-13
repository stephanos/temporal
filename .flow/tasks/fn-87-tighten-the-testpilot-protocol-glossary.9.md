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
A Slot is now the only opaque-handle encoding, the natural Value arm is gone in favour of `unsigned_integer_value` typed `UINT64`, and the wire `EntrypointKind` is replaced by a Go classification in the `contract` leaf. The wire-encoding scalar kinds are kept, with the reason in a `ScalarKind` comment. No fixture changed, and the oracle passes with three new declared steps.

**Decisions** (recorded in the fn-87 Planning decisions as "decided in .9")
- **Opaque handle.**
  - `SingularType.opaque_handle` is removed.
  - `ir` binds a handle Slot through `Catalog.OpaqueHandleType()`, a type with no schema. `owns`, `Equal` and `checkLiteral` accept it, so inspecting, reading or writing a literal into an opaque slot still rejects (existing path, expression and snapshot tests).
  - The Observation opaque check and the capture-type `opaque handle` test row are gone, because neither case can be written any more.
- **Natural removed.**
  - No Producer needs a value above 2^64-1: every runtime natural comes from a protobuf integer field. A probe showed the checked Property already rejects an oversized integer literal, as `"property"`, before lowering.
  - The evidence lift, correlated validation and literal typing use `unsigned_integer_value` / `UINT64`, checked canonical and within 64 bits. So do the Lean decoder (`unsigned integer overflow`) and the Umpire Producer (`protobuf unsigned overflow`). The Producer guard is defensive and no test reaches it.
  - `Value` arms and `ScalarKind` are renumbered so their numbers stay dense.
- **EntrypointKind.**
  - `contract.EntrypointKind` has the constants `ControllerEntrypoint`, `WorkflowEntrypoint`, `ActivityEntrypoint`, `NexusHandlerEntrypoint` and `MaxEntrypointKind`.
  - One classifier, `EntrypointKindOf`, is used by preparation and by the Temporal profile.
  - The facade re-exports the type and constants by alias and the classifier through a wrapper function.
- **Scalar kinds kept.** Admission types a field read by the field's declared kind and requires it to equal the declared type.

**Tests**
- `TestEntrypointKindOfClassifiesEachActivation`
- `protocol_test`: descriptor assertions on the `SingularType` and `Value` arms, and that `SCALAR_KIND_NATURAL` is absent
- Correlated `oversized-unsigned-literal` rejection
- Lean `#guard`s for the unsigned integer overflow and non-canonical rejections
- Oracle step cases for natural value and kind (by name and number), oversized and non-canonical values, and opaque singular type refusal. A mutation of the natural step made the oracle fail.
- `retiredvocabulary`: positive and negative lines

**Oracle and vocabulary**
- New steps:
  - `natural_value` → `unsigned_integer_value`, validated as canonical uint64
  - `ScalarKind` NATURAL → UINT64, with later numbers shifted down
  - refusal of an opaque `SingularType`
- Retired tokens: `NaturalValue` / `naturalValue`, `natural_value`, `SCALAR_KIND_NATURAL`, the `ENTRYPOINT_KIND_*` prefix, and `EntrypointKind` in the protocol vocabulary test.

**Outside declared Touches**
- `tools/umpire/internal/retiredvocabulary/check.go` and `check_test.go`: the retired tokens.
- `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go`: its crossed-oneof input named the removed kind.
- `.flow` spec: the decision note.

**Gates**
- Baseline was green: the oracle ran, and the regression receipt at 64480064 was honored.
- `make umpire-check-regression`:
  - Run 1: failed on the known flake (c). An async-nexus Run ended INCONCLUSIVE in `TestTestpilotAsyncNexusCase` and in the umpire-run test.
  - Run 2, at b32e986756: exit 0 with 9 live identities. Receipt written.
- `lint-code`: 161 after `go clean -cache`, none in changed files. `lint-model`: 163. Both are the baselines.
- `make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance` and `umpire-check-retired-vocabulary` passed.

**Review**
- Claude impl-review returned SHIP.
- Its P2 finding was valid: the enum swap had left `require.NotEqual(t, 0, …)` comparing an untyped int. It was fixed in b32e986756 with `require.NotZero`.
- A suppressed confidence-25 note was not acted on: `rewriteScalarKind` shifts an out-of-range baseline number instead of rejecting it, and no frozen fixture carries one.

stage: impl-review - ran (claude backend, SHIP on first round; P2 applied in follow-up commit b32e986756)
## Evidence
- Commits: 2a469e1163c179d4df74f9b29b41e25152cf0897, b32e98675614f81d9fa66de4897fe5b81407aeb3
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/..., make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance, make umpire-check-retired-vocabulary, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (run 1 at 2a469e1163+b32e986756: FAIL, known flake (c) async-nexus INCONCLUSIVE in TestTestpilotAsyncNexusCase and TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint; run 2 at b32e986756: exit 0, 9 live identities, receipt written), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 issues, baseline; none in changed files), make lint-model (163, baseline)
- PRs: