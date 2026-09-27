---
satisfies: [R1]
---
# fn-89-one-contract-rule-per-entity.1 Protocol and Authoring: Rule instance messages and the instance-value reference

## Description
Add the spec's API Contracts to the wire and to Lean Authoring (R1): `ContractRule.instance_values = 8`, `ContractRule.instances = 9`, the three new messages, and `Reference.instance_value_id = 13`. Split first because every later task (Go preparation, Evaluator, Producer fold, conformance Case) consumes these generated types and constructors. No behavior changes here: no Producer emits the new fields yet, and Go preparation learns them in the next task.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/contract.proto`, `proto/internal/temporal/server/api/testpilot/v1/expression.proto`, `api/testpilot/v1/contract.pb.go`, `api/testpilot/v1/expression.pb.go` (regenerated), `model/Testpilot/Authoring.lean`, `model/Testpilot/Tests/Authoring.lean`, `common/testing/testpilot/protocol_test.go` (`TestProtocolEncodesExpressionAndStateScopes` pins the Reference arm list), `common/testing/testpilot/internal/ir/expression.go` (only if the compiler's reference switch needs an explicit rejection arm to stay exhaustive)
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/contract.proto, proto/internal/temporal/server/api/testpilot/v1/expression.proto, api/testpilot/v1/**, model/Testpilot/Authoring.lean, model/Testpilot/Tests/Authoring.lean, common/testing/testpilot/protocol_test.go, common/testing/testpilot/internal/ir/expression.go]

## Approach
- Follow the README's extension order (`common/testing/testpilot/README.md:109-160`, and `### A new expression reference` at `:226-242`): proto, then `make proto`, then Authoring. Field numbers are appended densely; the oneof arm is appended last.
- Leading comments on every new message and field, worded with the spec's vocabulary (Rule, Rule instance, instance value; never monitor). Reword `contract.proto:22` and `:52` ("Rule-local") and `expression.proto:88` (`capture_id`, "the rule captured") to "local to one Rule instance" to match the SEM-17 draft. Extend the `Reference` comment (`expression.proto:72-75`) with the new arm's admitted context.
- Lean protocol types elaborate from the protos at build time (`model/Testpilot/Protocol.lean`); there is no codegen step, but `Testpilot.Protocol` and `Testpilot.Carried` rebuild (about 159 s for `Carried`). Confirm `Built Testpilot.Protocol` appears if the build looks stale (`model/lakefile.lean:14-52`).
- Authoring: add constructors beside `Contract.rule` (`model/Testpilot/Authoring.lean:615`) for an instance value declaration, a Rule instance and an assignment, give `Contract.rule` optional `instanceValues`/`instances` arguments defaulting to empty (so every existing call site and every rendered fixture is unchanged), and add `Expr.instanceValue` beside `Expr.capture` (`:272`). Guard them in `model/Testpilot/Tests/Authoring.lean` the way neighbouring constructors are guarded.
- Byte stability: the Lean ProtoJSON writer omits empty repeated fields (`printOptions`, `emitFieldsWithoutPresence := false`), so a Rule with no instances renders exactly as today. Do not regenerate fixtures in this task; confirm stability with `make umpire-check-case-runtime-conformance`.
- Go: if `ir` compilation of `Reference` is a type switch with a default rejection, the new arm already rejects with an unsupported-reference error; otherwise add an explicit rejection arm now. Admission is the next task.

## Investigation targets
**Required**:
- `proto/internal/temporal/server/api/testpilot/v1/contract.proto:17-27` and `expression.proto:76-99`
- `common/testing/testpilot/README.md:109-160,226-242` — extension checklist
- `model/Testpilot/Authoring.lean:272,615` — `Expr.capture`, `Contract.rule`
- `common/testing/testpilot/protocol_comments_test.go:25`, `protocol_test.go:143`

**Optional**:
- `model/Testpilot/ProtoJSON.lean` — key order follows descriptor order

## Key context
- fn-88.12 may move the model toolchain from Lean 4.33.1 to 4.32.0 (spec Dependencies). Use no 4.33-only API; land the Lean part wholly on one toolchain.
- Regenerate only with `make proto`; never hand-edit `.pb.go`.

### Carried (2026-09-27)
- A first attempt is saved as `.flow/tmp/fn-89.1-wip.patch` (proto, regenerated api, Lean Authoring constructors, protocol_test and ir/expression_test entries). Start from it. After the wire change, run `make umpire-rerecord-pinned-runs` (fn-89.7) and commit the refreshed records with this task.

## Acceptance
- [ ] the five protocol additions exist with the spec's names and numbers, each with a leading comment; the rule-local wording is updated
- [ ] `protocol_test.go`'s Reference arm list includes `instance_value_id`
- [ ] `make proto`, `make umpire-check-testpilot-protocol`, `make umpire-check-testpilot-authoring` pass
- [ ] Authoring constructors and `Expr.instanceValue` exist with tests; existing `Contract.rule` call sites compile unchanged
- [ ] `make umpire-check-case-runtime-conformance` passes with no fixture byte changed
- [ ] `go build ./...` and `go vet -tags test_dep ./common/testing/testpilot/...` pass

## Done summary
Added the R1 protocol surface: `ContractRule.instance_values = 8` / `instances = 9`, the `ContractInstanceValue`, `ContractRuleInstance` and `ContractInstanceAssignment` messages, and `Reference.instance_value_id = 13`, all with leading comments in the spec's vocabulary; the rule-local wording now reads "local to one Rule instance". Lean Authoring gains `Expr.instanceValue`, `Contract.instanceValue`, `Contract.instanceAssignment` and `Contract.ruleInstance`, and `Contract.rule` takes `instanceValues`/`instances` that default to empty, with guards in `Tests/Authoring.lean`. The Go `ir` admission is an allow-list, so the new arm is rejected everywhere until fn-89.2 (probed in `ir/expression_test.go`), and `protocol_test.go` pins the arm list. The pinned Runs and receipts were re-recorded via `make umpire-rerecord-pinned-runs`: only the catalog and run identities changed, and the conformance fixtures are byte-identical.

Follow-up: re-recorded Runs carry the local worker hostname (`...@Stephans-MacBook-Air-2.local@`, previously `...@vm@`). This comes from the fn-89.7 target, not this task. `lint-model` was inconclusive in the shared tree (see evidence).

stage: impl-review - ran (claude:opus:high, first-pass SHIP)
## Evidence
- Commits: 4dfc90dedf4f2f142233cd02bc0a14f1be199748, 964b57e54d2a533603354e98ca426f9ce4fd4865
- Tests: baseline: green (go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/evaluation/... pre-edit), make lint-protos protoc proto-codegen (generated api/testpilot/v1 byte-identical to the patch), make umpire-rerecord-pinned-runs (re-recorded 2 Runs + 3 receipts; only catalog/run identities changed), go build ./..., go vet -tags test_dep ./common/testing/testpilot/..., go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/evaluation/..., make lint-code-fast, make umpire-check-testpilot-protocol, make umpire-check-testpilot-authoring, make umpire-check-case-runtime-conformance (no fixture byte changed), make canary-check-case, INCONCLUSIVE: LEAN_NUM_THREADS=1 make lint-model - fails only on concurrent fn-88.4 uncommitted Umpire.Search.Product/MonitorProofs WIP and a stale Temporal/API/Types.olean build artifact from overlapping builds; neither touched by this diff
- PRs: