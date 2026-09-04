---
satisfies: [R1, R2, R3, R4, R5, R6]
---
# fn-61-simplify-the-umpire-go-execution-surface.7 Lock down the simplified surface and full regression gates

## Description
Document and mechanically guard the supported execution surface, then run the complete focused-to-aggregate verification for R1-R6. This final task removes stale architecture descriptions and catches public-import regressions.

**Size:** S
**Files:** `tools/umpire/README.md`, `tools/umpire/portableevaluation/README.md`, `tools/umpire/runevaluation/README.md`, `tools/umpire/runtime/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`, `tools/umpire/regression/ci_workflow_test.go`, `tools/umpire/*_test.go`, `Makefile`
**Touches:** [tools/umpire/README.md, tools/umpire/portableevaluation/README.md, tools/umpire/runevaluation/README.md, tools/umpire/runtime/README.md, model/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md, tools/umpire/regression/ci_workflow_test.go, tools/umpire/*_test.go, Makefile]

### Approach
- Add one concise root package guide showing direct and gRPC use, ownership, runtime independence from Lean, eventual completion, and the distinction between resident execution and offline Run Evaluation.
- Delete stale HTTP/legacy diagrams and package instructions; update architecture text to the actual facade-to-internal pipeline.
- Update the regression workflow's pinned package command and required-document/string matrix so it checks the simplified package layout and retained offline Run Evaluation path rather than deleted runner/runtime/local constructors.
- Add an import/API-surface guard following existing Umpire structural tests so code outside `tools/umpire` cannot regress to removed execution packages or recreate adapter/binding construction.
- Record the before/after execution package, exported-construction, orchestration-layer, and production-line counts; require fewer supported surfaces and no production-line increase without turning counts into a fragile exact golden.
- Run focused unit tests, generated drift, tagged integration, full Umpire regression, formatting, and code lint.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/portableevaluation/README.md:111-172,228-242` — stale legacy and current gRPC descriptions
- `model/ARCHITECTURE.md:377-399,531-552` — execution architecture and verification commands
- `tools/umpire/regression/ci_workflow_test.go:22,126-177` — pinned package command and documentation matrix
- `tools/umpire/runevaluation/README.md:125-140` — offline/live integration description
- `.plans/UMPIRE4_COMPONENTS.md:350-395,493-500` — component map for runner, HTTP, and attached factory
- `tools/umpire/runner/runner_test.go:291-315` — existing structural surface guard pattern

**Optional** (reference as needed):
- `.plans/UMPIRE4_SPEC.md` — canonical component and scope language

### Key context
The full live gate must continue selecting the complete `^TestUmpire` suite. For repository lint, compare against the pre-edit inherited baseline and require zero task-scoped findings rather than fixing unrelated debt.

### Acceptance
- [ ] Documentation presents one normal path: construct root executor, execute portable plan directly or through gRPC, inspect `ExecutionResult`; internal bindings/participants/Evidence mechanics are not setup steps.
- [ ] A structural test prevents imports of removed execution packages from outside `tools/umpire` and confirms no legacy HTTP serving surface.
- [ ] The regression workflow test, package-local CI command, runtime/portable/offline guides, architecture document, and Umpire component map name only paths and constructors that still exist.
- [ ] Before/after evidence shows fewer supported packages, exported construction concepts, and orchestration layers, with no production Go-line increase in the migrated execution stack.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/...`, generated drift, tagged `^TestUmpire` live tests, and `make umpire-check-regression` pass under the approved inherited-failure policy.
- [ ] `make fmt-imports` and `GOLANGCI_LINT_FIX=false make lint-code` run; any inherited failure is recorded against the pre-edit baseline and the task introduces zero scoped findings.

## Acceptance
- [ ] Docs and structural guards make the one supported execution path explicit.
- [ ] Complete generated, unit, live, regression, format, and lint gates satisfy R1-R6.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
