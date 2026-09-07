---
satisfies: [R1, R7]
---
# fn-73-explicit-environment-binding-for.1 Evolve the Testpilot protocol and Lean authoring surface

## Description
Extend the owned Testpilot protocol closure for symbolic environment bindings and Case 1.1 (R1). Keep wire declarations generated and add only producer-neutral authored Lean helpers and compatibility coverage.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/{program,expression}.proto`, generated `api/testpilot/v1/*`, `model/Testpilot/{Protocol,Authoring}.lean`, `model/Testpilot/Tests/{Compatibility,ProtoJSON,Authoring}.lean`, `tests/testcore/testpilot/protobuf_lean_authoring_test.go`
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/program.proto, proto/internal/temporal/server/api/testpilot/v1/expression.proto, api/testpilot/v1/**, model/Testpilot/Authoring.lean, model/Testpilot/Tests/**, tests/testcore/testpilot/protobuf_lean_authoring_test.go]

### Approach
- Add the environment definition, environment expression and role-reference fields to the authoritative protobufs without adding a Contract expression variant.
- Regenerate Go outputs through `make proto`; do not hand-edit generated Go or generated Lean declarations.
- Extend `Testpilot.Authoring` beside the existing Program expression, role and Program builders with pleasant constructors for the new protocol shapes.
- Add representative ProtoJSON and compatibility cases that pin presence/default behavior needed for exact 1.0/1.1 admission in the next task.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/program.proto:22-84` — role and Program wire ownership
- `proto/internal/temporal/server/api/testpilot/v1/expression.proto:23-69` — Program-only expression oneof
- `model/Testpilot/Protocol.lean:13-23` — generated descriptor closure
- `model/Testpilot/Authoring.lean:140-175,219-236,336-342` — authored facade patterns
- `model/Testpilot/Tests/ProtoJSON.lean:12-62` — public codec fixture pattern

**Optional** (reference as needed):
- `tests/testcore/testpilot/protobuf_lean_authoring_test.go:11-39` — Go/Lean cross-language check
- `model/lakefile.lean:14-65` — schema and library target ownership

### Key context
The eight checked-in protobufs remain the only Lean protocol source. `TestpilotTests` stays a separate Lake target while its modules remain colocated under `model/Testpilot/Tests`.

## Acceptance
- [ ] The protobuf contract contains only the specified Program environment, role binding and ProgramExpression environment-reference fields; ContractExpression remains structurally unable to carry them.
- [ ] `make proto` regenerates the owned Go outputs without hand edits or unrelated generated churn.
- [ ] Public Lean authoring helpers construct binding definitions, direct environment assignments and bound roles without exposing raw generated record assembly.
- [ ] Compatibility and ProtoJSON tests pin present-empty references, deterministic field encoding and literal 1.0 fixture stability.
- [ ] `make umpire-check-testpilot-protocol`, `make umpire-check-testpilot-authoring`, the focused cross-language Go test with `-tags test_dep`, and the relevant `Testpilot`/`TestpilotTests` Lake builds pass.

## Done summary
Implemented the Case 1.1 protocol surface for symbolic environment binding while preserving literal Case 1.0 compatibility. Added authoritative protobuf fields for Program environment definitions, Program-only environment references, and role namespace/resource references; regenerated the four owned Go mirrors through `make proto`.

Extended the producer-neutral Lean authoring facade with environment, role-binding, and direct assignment helpers. Focused Lean and Go coverage pins descriptor numbers, Contract-expression exclusion, present-empty oneof behavior, deterministic ProtoJSON, and the separate literal 1.0 compatibility path.

stage: wave-dispatch - ran (model: gpt-5.6-sol medium)
stage: impl-review - ran; SHIP on the task-scoped synthetic before/after range after the uncommitted real-tree range correctly refused an empty commit comparison (model: gpt-5.6-sol medium)
stage: plan-sync - skipped(config: planSync.enabled != true)

No unresolved implementation issue remains. Repository-wide lint is owned by the final fn-73 gate task; it was not run here with less than 1 GiB free.
## Evidence
- Commits:
- Tests: make proto, make umpire-check-testpilot-protocol, make umpire-check-testpilot-authoring, cd model && mise exec -- lake build Testpilot TestpilotTests, gofmt -d tests/testcore/testpilot/protobuf_lean_authoring_test.go, git diff --check, impl-review: SHIP (codex:gpt-5.6-sol:medium; /tmp/impl-review-receipt-fn-73-explicit-environment-binding-for.1.json)
- PRs: