---
satisfies: [R1, R2, R7]
---
# fn-19-bounded-local-temporal-execution-and.2 Build the domain-neutral checked runtime and participant contracts

## Description
Implement R1/R2's deep reusable Go boundary for checked requests, participant programs, commands, receipts, resources, and preflight.

**Size:** M
**Files:** `tools/umpire/runtime/runtime.go`, `tools/umpire/runtime/request.go`, `tools/umpire/runtime/participant.go`, `tools/umpire/runtime/errors.go`, `tools/umpire/runtime/request_test.go`
**Touches:** [tools/umpire/runtime/runtime.go, tools/umpire/runtime/request.go, tools/umpire/runtime/participant.go, tools/umpire/runtime/errors.go, tools/umpire/runtime/request_test.go]

### Approach
- Consume only fn-18 admitted typed sets; require exact two-member ExperimentSpec/RuntimeConfiguration input and never parse persisted bytes.
- Define immutable CheckedRunRequest, closed participant program/command/receipt/resource vocabulary, adapter/environment interfaces, stable error kinds, and bounded identity/value types.
- Validate profile/config/program/target/action/occurrence/participant/protocol/capability/budget/run/seed/attempt relations before invoking a factory.
- Use constructor/private-field discipline so callbacks, arbitrary maps, alternate semantic values, and unchecked receipts cannot enter the engine.
- Prove every preflight mutation returns no request and never calls an IO-counting fake factory.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/artifact` admitted set/types after fn-18
- Task `.1` exact profile values
- parent spec participant and preflight contracts

### Acceptance
- [ ] The package contains no Temporal/Nexus import or vocabulary.
- [ ] Every exact preflight failure is typed, deterministic, and side-effect free.
- [ ] One valid fixture produces an immutable checked request; reordered/duplicate/drifted inputs reject rather than normalize.
- [ ] No byte decoder, writer, mapping, evaluator, or general plugin surface exists.

## Acceptance
- [ ] R1/R2 checked runtime/participant boundary is deep, inert, and domain-neutral.
- [ ] Focused Go request/contract tests pass.
- [ ] Every public value enforces its documented Limits.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
