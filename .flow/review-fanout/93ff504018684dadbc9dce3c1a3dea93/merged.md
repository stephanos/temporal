# Merged review — fn-93-simplify-the-lean-model.1

Three concurrent draws (correctness, contracts, integration), each on `gpt-6-astra` via codex,
reviewed `a8a044d754..HEAD` (the two commits: the negative fixture and the schema/catalog import
fix, plus the added `StartWorkflowExecutionRequest` schema root). All three returned SHIP with zero
findings.

## Findings

No findings from any draw. No same-defect dedupe was needed.

Suppressed findings: none.

## Reviewer-reported verification gaps (sandbox-limited)

Each draw's own sandbox is read-only, so none could run the golden comparison or the model-wide
lint, and two of three could not complete a fresh `lake build` (they fell back to direct
elaboration of the new fixture against already-compiled dependencies). The coordinator (this
worker) ran the full verification matrix outside the reviewer sandbox, with green results and
`flowctl gate receipt` entries recorded against commit `cb40e961d1`:

- `make umpire-build-model` — full model build, 1346/1346 jobs, zero failures (surfaced and fixed
  the two latent `Workflow.Start.Model` / `Workflow.Outage.Model` schema errors this task's receipt
  lists).
- `make umpire-check-goldens` — clean, no diff.
- `make canary-check-case` — clean, no diff.
- `make umpire-check-case-runtime-conformance` — clean, no diff; both Go packages `ok`.
- `go test ./tools/umpire/vocabulary/... ./tools/umpire/authoring/... ./tools/umpire/regression/...`
  — all `ok` (includes `spec_names_test.go`).
- `LEAN_NUM_THREADS=1 make lint-model-builtin LINT_MODEL_MODULES="Temporal.Case.Syntax
  Temporal.Case.Schema Temporal.Case.Tests.ProductionImports TemporalModelTests
  Temporal.Feature.Workflow.Start.Model Temporal.Feature.Workflow.Outage.Model"` — scoped lint,
  zero findings for every named module.

This closes the R10/R12 "partial" notes the contracts draw raised: golden and case-runtime
byte-identity are confirmed green outside the sandbox, and the receipt (Phase 5 evidence) records
the build-cost and baseline-split numbers R12 asks the first fn-93 task to capture.

## Requirements coverage (merged)

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | `Temporal.Case.Syntax` imports both `Temporal.Case.Schema` and `Temporal.Case.Catalog`; the negative fixture in `Temporal.Case.Tests.ProductionImports` pins both rejections and is wired into `TemporalModelTests`; every production Model (400 files) elaborates under `make umpire-build-model`, including the Worker Model fn-92 added. The two latent `StartWorkflowExecutionRequest` schema-resolution failures this surfaced were fixed by adding `startWorkflowExecution` as a sixth admitted root in `Temporal.Case.Schema`, and are listed in the task receipt. |
| R10 | met | `make umpire-check-goldens` and `make canary-check-case` are byte-identical; `make umpire-check-case-runtime-conformance` reports no diff. |
| R12 (partial, this task's share) | met | The task receipt records the start-baseline split (generated/test/production) after fn-88/fn-89/fn-92 landed, and the cold-build cost of the wider import closure. |
| R2–R9, R11, R13–R19 | deferred | Out of this task's scope (later fn-93 lanes). |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - all three draws SHIP with zero findings; the coordinator's own
out-of-sandbox verification (build, goldens, canary, case-runtime conformance, scoped lint, go
tests) confirms what the sandboxed draws could not run themselves.
