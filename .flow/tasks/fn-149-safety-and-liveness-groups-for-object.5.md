---
satisfies: [R3, R4, R5, R6]
---
# fn-149-safety-and-liveness-groups-for-object.5 Document grouped authoring and close integrated gates

## Description
Prepare grouped author docs and the final meaning-preserving source seal before fn-123's fault changes; link the shared batch close later. Advances R3, R4, R5, R6.

**Size:** M
**Files:** `model/README.md`, `model/SEMANTICS.md`, `tools/umpire/README.md`, `model/irgen/testdata/layout/**`; managed trees and functional/canary pins publish once through fn-123.8's shared close.
**Touches:** [model/README.md, model/SEMANTICS.md, tools/umpire/README.md, model/irgen/testdata/layout/**, .flow/tmp/fn149/task5/**]

### Approach
- Update the author layout/template and examples with safety postconditions, bounded progress and the distinction between verification and witness finding. Document supported composition safety and link composition progress work to fn-150.
- Integrate .3's reports and .4's complete migration seal. Include final documentation and executable layout changes in an independent scratch grouping/assessment-equivalence comparison against the frozen fn-140.6 baseline. Preserve exact authorized identity/provenance mapping and changed functional/canary pin disposition, with historical recorded Run companions untouched. Commit this source and complete final seal before root releases fn-123.1.
- Source completion and the grouping seal do not complete this task/spec. Hold its integrated acceptance open until fn-123.8 provides the single production regeneration, full model/Case/fixture/lint/Go gates, independent review and live run for the whole inserted Batch 2. Link the prior witness and grouping seals with composed mappings; changed functional/canary pins are published once at that boundary.
- At the shared close, use actual `fcntl`/`flock` on `/tmp/umpire-heavy-gates.lock` and the existing skip-go-checks split; capture full Go JSON terminal events and wall time. Reuse applicable passing receipts, repeat only invalidated checks, and retain complete fixture/assertion populations. More than one hour cumulatively stuck across attempts defers validation unless every other work path is blocked; record the command, elapsed attempts, evidence and revisit condition. OOM/ENOSPC, unexecuted assertions and strict semantic failures earn no green credit; fn-154/fn-157 resource debt and Batch 5 failures keep their existing separate obligations.
- Close requirement coverage only after the shared boundary evidence establishes each required result. Do not add a broad generated-API drift gate, new CI workflow or live composition Case capability.

### Investigation targets
**Required:**
- `model/README.md` - authoring and gate instructions.
- `model/SEMANTICS.md:164` - safety claims; Claims at :434 and bounded Progress at :506.
- `model/irgen/testdata/layout` - executable author layout.
- `MILESTONES.md` - verification and regeneration serialization.
- `tools/umpire/README.md` - current reporting documentation.

### Key context
Root owns the possible Batch 2 insertion and source gates. The final grouping seal, including this task's executable-layout changes, precedes fn-123.1. This task remains pending shared fn-123.8 close; never add a within-batch whole-spec dependency that prevents that close from running. Re-anchor actual fn-155/fn-156/fn-140 inputs at execution; fn-141 runs later.

### Quick commands
These are required shared-close commands, run once by fn-123.8 rather than as a separate fn-149 gate campaign.
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-cases umpire-check-fixtures canary-check-case
make lint-model
make lint-code-fast
go test -tags test_dep -p 2 -timeout 30m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
```

## Acceptance
- [ ] Author documentation and executable layout agree on grouping and all R6 examples; composition progress is not advertised as shipped.
- [ ] Complete independent final source/layout grouping seal and its authorized mapping are committed before fn-123.1, with safety/progress/negative-control and expected/live-replay assessment meaning preserved.
- [ ] At fn-123.8's shared boundary, integrated model, Case/fixture, applicable lint and Go gates pass against the explained migration diff; required review/live and complete coverage evidence are linked before this task/spec closes.
- [ ] Record focused and shared full gate evidence, changed artifact disposition and unresolved unsupported/resource/semantic results without weakening expected verdicts or treating deferral as a pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
