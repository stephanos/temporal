---
satisfies: [R7, R8, R9]
---
# fn-19-bounded-local-temporal-execution-and.8 Integrate generated Go tests and synchronize runtime documentation

## Description
### Umpire4 reconciliation (normative)

There is no public `umpire-local-run`, `umpire-run-local`, or separate run-tests CLI. Generate ordinary Go tests through `umpire-gen-tests` / `umpire-gen-tests-go`; those tests call the reusable `tools/umpire/runner` directly and preserve normal `go test` discovery, filtering, breakpoints, and failures. An internal harness may prove adapters but is not user-facing.

Complete R7/R8/R9 with the reusable runner handoff, one deterministic generated Go test, public usage documentation, and honest roadmap status.

**Size:** M
**Files:** `tools/umpire/runner/**`, `tools/umpire/cmd/umpire-gen-tests-go/**`, `tools/umpire/temporal/nexus/*runner*`, `tools/umpire/runtime/README.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [tools/umpire/runner, tools/umpire/cmd/umpire-gen-tests-go, tools/umpire/temporal/nexus, tools/umpire/runtime/README.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Add a domain-neutral `tools/umpire/runner` handoff that consumes only an exact admitted two-member set, verifies generated digest literals before adapter construction, runs in memory, and returns the admitted four-member output without publishing.
- Bind the sole System-owned caller-closure authority/program through the current local and Nexus adapters; do not reconstruct setup, program, order, observation, termination, or cleanup intent in Go.
- Add only the smallest generation-only `umpire-gen-tests-go` seam needed to deterministically render one ordinary test that calls the runner directly and remains discoverable/filterable/debuggable through `go test`.
- Document the two-member input, fixed profile/budgets, four-member output, failed-run artifact behavior, evidence dispositions, local-only authority, and downstream Run Evaluation boundary.
- Update C6/C9 and pilot-step implementation status only after the live test is implemented; keep Milestone B unintegrated until live semantic Run Evaluation.

### Investigation targets
**Required** (read before coding):
- Task `.7` live API/publication result
- `Makefile:988-1118` Umpire target conventions
- `.plans/UMPIRE4_COMPONENTS.md:303-327,394-416,613-640,706-716`
- parent runner/generated-test and boundary contracts

### Acceptance
- [ ] The generated test is byte-deterministic, embeds the exact admitted pretty-plus-one-LF input set, retains literal set/member digest binding, and calls `tools/umpire/runner` directly.
- [ ] Incomplete input or digest drift fails before adapter construction; the runner does not read bytes, publish, evaluate evidence, or add authority options.
- [ ] Documentation gives one copy-paste ordinary `go test` run and does not imply semantic Run Evaluation or remote support.
- [ ] No model-local Makefile, CI, remote adapter, or prohibited legacy reference/use is added.
## Acceptance
- [ ] R7/R8/R9 generated-test runner UX, docs, and implementation-time roadmap status are complete.
- [ ] All focused suites and the one bounded live command pass.
- [ ] Existing comments remain preserved.

## Done summary
Implemented the reusable digest-bound local runner, closed Nexus adapter, deterministic generation-only Go seam, and ordinary generated live test, with exact two-member input and four-member output boundaries documented honestly. The review fix made generator tests fresh-checkout-safe, safely constrained embed directives, and completed the exact budget and evidence-disposition documentation.

Verification is green for every reconciled parent Quick command plus race, vet, and diff checks. The pre-edit baseline inherited three stale legacy Quick failures (the prohibited local-run CLI, Feature-owned Lean target, and Make wrapper); this task replaced them with the normative runner/generated-test commands. Memory capture was skipped because flow memory is enabled but not initialized.

baseline: green for implemented dependency surfaces; inherited red for the three task-owned stale legacy Quick entries described above

stage: impl-review - ran [2026-08-29T19:43:28Z..2026-08-29T19:48:17Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 507a8626e9e46a4637f762128e512c43e7e65376, 2dabcde445694955fff47c741e857cd528a742e8
- Tests: go test -count=1 ./tools/umpire/runtime/..., go test -count=1 ./tools/umpire/runner/..., go test -count=1 ./tools/umpire/temporal/local/..., go test -count=1 ./tools/umpire/temporal/nexus/..., go test -count=1 ./tools/umpire/cmd/umpire-gen-tests-go/..., go test -count=1 ./temporaltest/..., cd model && mise exec -- lake build Temporal.System.Execution.LocalProfileTests, cd model && mise exec -- lake build Temporal.NexusExecutionIntegrationTests, go test -count=1 ./tools/umpire/temporal/nexus/... -run '^TestGeneratedWorkflowNexusQueryExactActionCallerClosureExecutesLocally$', go test -race -count=1 ./tools/umpire/runner/... ./tools/umpire/cmd/umpire-gen-tests-go/... ./tools/umpire/temporal/nexus/..., go vet ./tools/umpire/runner/... ./tools/umpire/cmd/umpire-gen-tests-go/... ./tools/umpire/temporal/nexus/..., git diff --check, INHERITED_RED:go test -count=1 ./tools/umpire/cmd/umpire-local-run/... - prohibited package absent before edit, INHERITED_RED:cd model && mise exec -- lake build Temporal.Feature.Nexus.ExecutionTests - stale Feature-owned target absent before edit, INHERITED_RED:make umpire-run-local SET=tools/umpire/temporal/nexus/testdata/caller-closure-input-set OUTPUT_ROOT=/tmp/umpire-local-runs RUN_ID=umpire.local.caller-closure.run-1 - prohibited Make wrapper absent before edit
- PRs:
