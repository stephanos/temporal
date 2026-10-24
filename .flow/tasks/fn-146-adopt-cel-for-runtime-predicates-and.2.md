---
satisfies: [R2]
---
# fn-146-adopt-cel-for-runtime-predicates-and.2 Admit canonical CEL through one restricted environment

## Description
**Note 2026-10-18 (one-language series):** Depends on fn-202 (Testpilot protos move under the framework) and then fn-203's single Case regeneration.

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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: