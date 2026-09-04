---
satisfies: [R10]
---
# fn-64-umpire-case-runtime.10 Reconcile Umpire documentation, artifacts, and regression gates

## Description
Finish R10 after code removal by updating normative/current documentation, regenerating
owner-managed artifacts, and making the complete Case Runtime suite the regression boundary.

**Size:** M
**Files:** `.plans/UMPIRE4_*`, `tools/umpire/CONTEXT.md`, Umpire/model package READMEs and
architecture docs, Makefile/regression tests/workflow command surface, generated regression and
inventory artifacts
**Touches:** [.plans/UMPIRE4_*, tools/umpire/**/*.md, tools/umpire/CONTEXT.md, model/**/*.md,
model/SEMANTIC_INVENTORY.md, tools/umpire/regression/**, Makefile, .github/workflows/umpire.yml]

## Approach
- Update the normative specification using stable rule IDs: add/retire, never renumber. Reconcile
  component, CLI, model, package, and glossary documentation with the public boundaries.
- Mark genuinely historical references superseded; active docs/help/targets describe only the new
  vocabulary and architecture.
- Regenerate regression views and semantic inventory only through their owning generators.
- Make the existing regression workflow select every new IR/execution/verification/server/worker/
  integration layer and compare the complete failure-identity set.
- Preserve focused generator checks; do not add broad Lean API drift enforcement or new GitHub
  Actions coverage.

## Investigation targets
**Required** (read before coding):
- `Makefile:1063-1167` — legacy generation and regression gate
- `tools/umpire/regression/ci_workflow_test.go:21-178` — enforced command/package/doc contract
- `.plans/UMPIRE4_SPEC.md:1-161` — normative terminology/rule governance
- `.plans/UMPIRE4_COMPONENTS.md:280-418` — legacy component map
- `model/README.md:373-556` — old handoff/regression docs
- `model/ARCHITECTURE.md:301-438` — old compiler/runtime ownership
- `tools/umpire/CONTEXT.md` — approved replacement glossary

**Optional** (reference as needed):
- `.flow/memory/bug/integration/full-integration-gates-must-select-the-2026-09-04.md` — full-suite
  gate lesson

## Key context
Generated documentation is changed through its owner. Broad generated-Lean drift verification and
new workflow coverage are explicitly declined.

## Acceptance
- [ ] Normative rules, component diagrams, package READMEs, model/compiler docs, command help, and
  glossary agree on the new vocabulary and server/worker authority split.
- [ ] Active docs/help/targets contain no legacy runtime recommendation; historical names remain
  only under an explicit superseded marker.
- [ ] All managed views and semantic inventory are regenerated through their owners with focused
  checks passing.
- [ ] `make umpire-check-regression` selects the complete migrated suite and fails on additions or
  deletions in the inherited failure-identity set.
- [ ] No broad generated-Lean API drift gate or new GitHub Actions coverage is added.
- [ ] Focused generation/unit/integration commands, `make umpire-build-model`,
  `make umpire-check-regression`, `make fmt-imports`, and `make lint-code` pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
