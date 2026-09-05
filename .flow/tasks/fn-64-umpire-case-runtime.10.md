---
satisfies: [R2, R4, R10]
---
# fn-64-umpire-case-runtime.10 Reconcile Umpire documentation, artifacts, and regression gates

## Description
Finish R10 after code removal by updating normative/current documentation, regenerating
owner-managed artifacts, installing the small public-facade conformance corpus, and making the
complete Case Runtime suite the regression boundary.

**Size:** M
**Files:** `.plans/UMPIRE4_*`, `tools/umpire/CONTEXT.md`, Umpire/model package READMEs and
architecture docs, Makefile/regression tests/workflow command surface, generated regression and
inventory artifacts
**Touches:** [.plans/UMPIRE4_*, tools/umpire/**/*.md, tools/umpire/CONTEXT.md, model/**/*.md,
model/SEMANTIC_INVENTORY.md, tools/umpire/testdata/**, tools/umpire/regression/**, Makefile,
.github/workflows/umpire.yml]

## Approach
- Update the normative specification using stable rule IDs: add/retire, never renumber. Reconcile
  component, CLI, model, package, and glossary documentation with the public boundaries, bounded
  rule captures, expiry-before-transition semantics, activation-level cancellation, and immutable
  closed Runs. Describe private Slot opacity without implying general response secrecy.
- Mark genuinely historical references superseded; active docs/help/targets describe only the new
  vocabulary and architecture.
- Regenerate regression views and semantic inventory only through their owning generators.
- Add exactly six facade-level conformance classes: satisfied, violated, inconclusive, static
  preparation rejection, cleanup failure after proved violation, and cross-Run isolation. Keep
  concurrency, cancellation, descriptor/path grammar, fuzzing, cardinality, and lifecycle tests
  focused rather than converting them to goldens.
- Compare deterministic Lean-produced Case/Contract fixtures byte-for-byte. Compare intentional
  Run-time values only through a closed named stable projection, validate excluded dynamic fields
  structurally, and prohibit generic normalization. Source expected results from Lean or fixed
  hand-authored tables that never invoke the Go runtime under test.
- Generate a complete validated fixture tree under a temporary root and diff it against the
  checkout. Keep promotion a separate reviewed action; ordinary Go tests invoke neither Lean nor a
  fixture rewrite.
- Make the existing regression workflow select every new IR/execution/verification/server/worker/
  integration layer, run the complete `-run '^TestUmpire'` selector, and compare the complete
  inherited failure-identity set.
- Add structural guards for the two-call root facade and internal execution imports, and ensure
  fn-5's scenario-neutral checked-promotion types and validation build without caller-closure
  imports.
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
- Normative rules, component diagrams, package READMEs, model/compiler docs, command help, and
  glossary agree on the new vocabulary and server/worker authority split.
- Active docs/help/targets contain no legacy runtime recommendation; historical names remain
  only under an explicit superseded marker.
- All managed views and semantic inventory are regenerated through their owners with focused
  checks passing.
- The six-class facade corpus uses independent expected results, exact deterministic bytes or a
  closed named stable projection, structurally validates dynamic fields, and has no generic ignore
  or normalization mechanism.
- Fixture verification writes a complete temporary tree before diffing; interruption cannot
  partially update the checkout, promotion is separate, and ordinary Go tests invoke neither Lean
  nor rewrite mode.
- `make umpire-check-regression` selects the complete migrated suite and fails on additions or
  deletions in the inherited failure-identity set; the live tagged selector remains
  `-run '^TestUmpire'`.
- Structural tests reject public scheduler/recorder/Slot/Monitor-factory construction, imports
  of `tools/umpire/internal/execution` by external packages, and drift from the two-call root facade.
- The migration ledger is complete, and fn-5's generic promotion seam remains buildable without
  caller-closure imports.
- No broad generated-Lean API drift gate or new GitHub Actions coverage is added.
- Focused generation/unit/integration commands, `make umpire-build-model`,
  `make umpire-check-regression`, and `make fmt-imports` pass; literal `make lint-code` is executed,
  task-owned files are lint-clean, and the finding set adds zero findings relative to the recorded
  pre-task baseline. The inherited repository lint backlog remains unchanged and is reported without
  being claimed green.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
