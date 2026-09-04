---
satisfies: [R4, R5, R6]
---
# fn-61-simplify-the-umpire-go-execution-surface.6 Retire the legacy HTTP and non-portable executor path

## Description
Delete the superseded HTTP resident executor and its `ExecuteRequest`/`ExecuteResponse` implementation, then make portable evaluation an internal stage of the sole facade (R4). Preserve the evaluator rather than rewriting its proven semantics.

**Size:** M
**Files:** `tools/umpire/executorhttp/**`, `tools/umpire/executor/executor.go`, `tools/umpire/executor/executor_test.go`, `tools/umpire/evaluationcontract/**`, `tools/umpire/portableevaluation/**`
**Touches:** [tools/umpire/executorhttp/**, tools/umpire/executor/executor.go, tools/umpire/executor/executor_test.go, tools/umpire/evaluationcontract/**, tools/umpire/portableevaluation/**]

### Approach
- Remove the HTTP handler, legacy resident executor, and duplicate gate after Task `.2` has migrated the repository integration test; do not add a redirect or compatibility server.
- Retain one execution gate in the root facade and one evaluator path from authorized portable plan plus closed Evidence to `ExecutionResult`.
- Internalize evaluation-contract packing/admission and interpreter machinery still needed to preserve semantic parity; delete legacy-only public entry points, requests, status conversion, tests, and limits.
- Keep generated legacy proto messages unchanged and inert so this task does not trigger schema or generated-source churn.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/executor/executor.go:21-180` — duplicate legacy lifecycle owner
- `tools/umpire/executorhttp/handler.go` — superseded transport
- `tests/umpire4_portable_executor_test.go:24-45,339-350` — only repository HTTP caller
- `tools/umpire/portableevaluation/portable.go:14-63,229-288` — portable adapter over the interpreter
- `tools/umpire/portableevaluation/evaluator.go:15-84` — evaluator core to preserve

**Optional** (reference as needed):
- `tools/umpire/portableevaluation/parity_test.go` — Lean/Go semantic parity corpus

### Key context
`EvaluationContract` may remain as an internal interpreter representation because portable evaluation currently projects into it. R4 removes it as a resident caller contract, not the semantic checks it carries.

### Acceptance
- [ ] `executorhttp`, the legacy resident `Executor`, and all serving/test code for `ExecuteRequest`/`ExecuteResponse` are removed with no replacement compatibility layer.
- [ ] Exactly one resident execution gate and one portable evaluation path remain.
- [ ] Evaluation-contract/interpreter code still required for semantic parity is internal; unused legacy packing, limits, statuses, and tests are deleted.
- [ ] Portable result bytes, limits, diagnostics, property decisions, status mappings, and Lean/Go parity remain exact.
- [ ] No proto, generated code, HTTP route, new transport, or existing comment changes.

## Acceptance
- [ ] The obsolete HTTP/non-portable serving path is deleted.
- [ ] One private evaluator preserves exact portable semantics and parity.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
