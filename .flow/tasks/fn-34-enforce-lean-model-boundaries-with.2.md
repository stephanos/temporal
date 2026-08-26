---
satisfies: [R5]
---
# fn-34-enforce-lean-model-boundaries-with.2 Align normative and supporting architecture documentation

## Description
Update the normative index and supporting architecture documents for R5 after the executable rule
matrix is proven. Keep each rule in one owning section and avoid creating a second normative table.

**Size:** M
**Files:** `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_SPEC_MODEL_ARCH.md`, `.plans/UMPIRE4_SPEC_COMPS.md`, `model/ARCHITECTURE.md`, `model/README.md`
**Touches:** [.plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_SPEC_MODEL_ARCH.md, .plans/UMPIRE4_SPEC_COMPS.md, model/ARCHITECTURE.md, model/README.md]

## Approach

- Replace the normative `Model architecture` grouping with `Enforced module boundaries` and `Module
  design`. Place every existing MOD rule once, preserve every rule ID, and allocate new MOD IDs for
  newly explicit base-System/refinement and lint-coverage contracts rather than overloading or
  renumbering IDs.
- Keep MOD-04's independently understandable/testable design requirement under `Module design`;
  move its import-composition constraint into the new fully lintable rule with exact
  qualified-module classification and exception language.
- Give the existing `Shared.*` independence invariant a new stable MOD ID under `Enforced module
  boundaries` and align the executable rule matrix and supporting dependency descriptions to it.
- State that the existing model lint command checks transitive reachability and complete first-party
  inventory. Keep semantic altitude, deep modules, narrow responsibilities, and isolated testability
  normative without claiming graph enforcement.
- Use fully qualified Lean module, namespace, and type names in backticks throughout every touched
  normative rule. Preserve unrelated prose and comments, including review markers.
- Align the model architecture and component design with the single executable mechanism, Shared
  independence, and the exact refinement/Verify consumer exceptions. Supporting documents may
  explain the rule but must cite the normative MOD IDs instead of redefining them.
- Update model developer docs so `make lint-model` owns import boundaries; keep the focused
  regression command's domain-purity, artifact, and diagnostic responsibilities accurate after its
  duplicate Feature/System grep is removed.

## Investigation targets

**Required** (read before editing):

- `.plans/UMPIRE4_SPEC.md:176-193` — stable normative MOD IDs and current mixed architecture section
- `.plans/UMPIRE4_SPEC_MODEL_ARCH.md:78-94,421-440` — qualified module layout, refinement/Verify exceptions, and mechanical-enforcement claim
- `.plans/UMPIRE4_SPEC_COMPS.md:809-826,855-869` — supporting dependency matrix and testing strategy
- `model/ARCHITECTURE.md:95-99,197-200` — current high-level direction and regression-gate description
- `model/README.md:111-123` — developer-facing focused-check description

## Key context

- GOV-01 forbids renumbering or reusing rule IDs.
- The normative core remains `.plans/UMPIRE4_SPEC.md`; adjacent documents are explanatory and must
  not compete with it.
- Preserve existing comments while editing and do not fold the unrelated ubiquitous-language
  regrouping into this task.

## Acceptance

- [ ] `Enforced module boundaries` contains only graph-decidable rules and identifies the model lint gate; `Module design` contains the remaining normative design judgments.
- [ ] Existing MOD IDs remain stable and new contracts receive new IDs.
- [ ] `Shared.*` independence has explicit normative ownership and matches the implemented direct/transitive check.
- [ ] Every touched Lean reference is fully qualified and enclosed in backticks.
- [ ] Refinement and opt-in verification exceptions match the implemented exact policy without a broad wildcard bypass.
- [ ] Supporting architecture/component docs cite the normative ownership and the single lint mechanism without duplicating authority.
- [ ] Model developer docs accurately divide responsibilities between `make lint-model` and the focused regression target.
- [ ] Unrelated prose, rule text, comments, and review markers are preserved.
- [ ] Markdown/reference checks and `make lint-model` pass after the documentation alignment.
- [ ] R5 is satisfied across the normative index and every affected supporting model document.
- [ ] Rule IDs, fully qualified vocabulary, comments, and command ownership remain consistent.

## Done summary
Separated the normative import-boundary rules from module-design judgments, assigned new stable MOD IDs to Shared/System/lint coverage, and aligned every supporting model document with the exact implemented policy and command ownership. Fully qualified references, exact exception identities, and the single normative authority remain consistent; the configured Codex review returned SHIP with zero findings.

baseline: green (`cd model && mise exec -- lake build modelLintTests modelLint`; `make lint-model`); inherited/tooling red (`make lint` failed pre-edit because `/home/agent/.cache/go-build` is a broken cross-workspace symlink)

verify: green (documentation consistency/reference checks; `cd model && mise exec -- lake build modelLintTests modelLint`; `make lint-model`); inherited/tooling red (`make lint` failed at the same broken Go-cache symlink before evaluating task changes; log: `.flow/tmp/fn34-task2-verify-make-lint.log`)

review: SHIP with zero introduced or pre-existing findings; R5 met and exact documentation policy matched `ModelLint.ImportGraph.defaultPolicy`.

memory capture: skipped because the review was clean on its first pass.

stage: impl-review - ran
## Evidence
- Commits: f4b3fb225e0dc5ecd9ea277967a77edb321e4336
- Tests: documentation consistency/reference checks (pass: unique MOD-01 through MOD-11 ownership, exact policy identities, non-normative supporting additions, git diff --check), cd model && mise exec -- lake build modelLintTests modelLint (pass), make lint-model (pass), make lint (inherited/tooling red: broken /home/agent/.cache/go-build cross-workspace symlink)
- PRs: