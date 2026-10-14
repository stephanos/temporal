---
satisfies: [R2]
---
# fn-146-adopt-cel-for-runtime-predicates-and.2 Admit canonical CEL through one restricted environment

## Description
Introduce the canonical CEL AST and the pinned restricted environment for R2. This task owns syntax admission, variable and function authority, source mapping, cost limits and the isolated canonical-to-engine AST bridge.

**Size:** M
**Files:** CEL proto inputs and generated APIs, `proto/internal/temporal/server/api/testpilot/v1/expression.proto`, `common/testing/testpilot/internal/ir/expression.go`, `model/check/Gate.scala`, `go.mod`, `go.sum`
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/expression.proto, api/testpilot/v1/expression*.go, common/testing/testpilot/internal/ir/expression.go, common/testing/testpilot/internal/ir/*cel*, model/check/Gate.scala, go.mod, go.sum]

### Approach
- Store canonical `cel.expr` messages and isolate the pinned engine's legacy AST conversion at one checked boundary.
- Define site-specific variables, functions, types and AST subsets over one environment contract.
- Retain structural, depth, work and cancellation bounds around parsing, admission and evaluation.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/expression.proto:1-162` - current expression contract
- `common/testing/testpilot/internal/ir/expression.go:190-335` - current admission and evaluation kernel
- `.plans/UMPIRE_CEL_RUNTIME_RESEARCH.md:62-84` - dependency and accounting constraints
- `.plans/UMPIRE_CEL_RUNTIME_RESEARCH.md:106-143` - prototype bridge and remaining decisions
- `model/check/Gate.scala` - `schemaClosure` and `generateIr` own the complete imported Umpire ScalaPB schema inputs; retain the Umpire/Testpilot import boundary


### Quick commands

```bash
go test -tags test_dep ./common/testing/testpilot/internal/ir/...
```

Record a site-specific outcome matrix for missing binding, evaluator error, cancellation and cost exhaustion. Cover nested presence, map misses, mixed wildcard absence, float32 widening, message/Any equality, enum aliases and existential multi-fact matching.

## Acceptance
- [ ] R2's canonical schema, restricted environments and error locations are implemented.
- [ ] Unsupported fields are rejected before canonical-to-engine conversion.
- [ ] Depth, work, cancellation and environment-reference negative tests pass.
- [ ] Focused IR expression and generation tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
