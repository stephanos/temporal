---
satisfies: [R1, R2, R3, R4]
---
# fn-34-enforce-lean-model-boundaries-with.1 Build and integrate the import-graph linter

## Description
Implement the pure import-graph checker and thin Lake/Lean adapter for R1-R4. Keep graph policy
isolated from metadata and process I/O so the rule engine is exhaustively testable.

**Size:** M
**Files:** `model/ModelLint/ImportGraph.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/ModelLint.lean`, `model/lakefile.toml`, `Makefile`
**Touches:** [model/ModelLint/ImportGraph.lean, model/ModelLint/ImportGraphTests.lean, model/ModelLint.lean, model/lakefile.toml, Makefile]

## Approach

- Add `ModelLint.ImportGraph.check` as the pure boundary: module records plus an explicit
  qualified-name policy in, deterministically ordered violations out. Keep traversal cycle-safe and
  select a lexically stable shortest path.
- Encode all current first-party module classes and exact exception sets in one policy table.
  Enforce `Shared.*` independence. Authorize `Temporal.System.Nexus.Refinement` exactly for
  refinement and authorize only `TemporalVerify`, `TemporalVeilTests`,
  `Temporal.Tool.VerifyVeil`, and `Temporal.Feature.Nexus.CallerClosure.VeilTests` as opt-in
  Verify/Veil consumers; do not infer exemptions from broad suffixes or paths.
- Extend the existing driver adapter to make owned modules fresh through Lake, read direct imports
  from `Lean.ModuleData.imports`, and compare the loaded graph with every Lean source contained by
  the canonical model package root. Forcibly prune `.lake` and build/runtime state, reject duplicate
  module identities, and do not follow symlinks outside the root. External package modules are
  excluded; traversal/permission failures and unknown, missing, or unclassified first-party modules
  fail closed.
- Preserve the existing Batteries declaration-linter flow and combine both result classes so neither
  suppresses the other. Render all architecture findings with rule identity and fully qualified
  path, then return non-zero.
- Add a non-default `modelLintTests` executable. Its normal mode runs the synthetic suite; a fixed
  controlled-violation mode reuses the adapter's rendering and exit composition and exits non-zero.
  Make `make lint-model` execute both modes, assert the expected-failure status and deterministic
  stderr/path, and then run the live custom lint. Remove only the duplicate Feature/System grep block
  from the regression target; preserve domain-vocabulary and other unrelated checks and comments.
- Cover Shared/Umpire/Feature/System/Verify direct and transitive rejection, allowed ordinary imports,
  exact refinement and opt-in aggregate/tool/test exceptions, near-miss exceptions, inventory and
  containment failures, external leaves, equal shortest paths, multiple findings, and cycles.

## Investigation targets

**Required** (read before coding):

- `model/ModelLint.lean:8-60` — current driver, root build, declaration lint, and exit-status composition
- `model/lakefile.toml:1-35` — existing custom lint driver and target declarations
- `model/.lake/packages/batteries/scripts/check_imports.lean:37-43,71-79` — local dependency example reading module import metadata
- `.bin/lean-4.33.1/lean-4.33.1-linux_aarch64/src/lean/Lean/Environment.lean:107-128` — pinned toolchain definition of `Lean.ModuleData.imports`
- `Makefile:1046-1088,1263-1278` — existing domain/import grep checks and model lint entrypoint
- `.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md:7,144,314-318` — exact opt-in aggregate, executable, and ordinary-path isolation contract

**Optional** (reference as needed):

- `model/Umpire/Behavior/ImportTests.lean:3-15` — checked import/visibility regression style

## Key context

- `Temporal.Tool.Inspect` is outside the current Temporal lint aggregate, so aggregate reachability
  alone is not a complete inventory.
- No Verify or Veil module exists yet; synthetic tests must freeze their policy before fn-24/fn-25
  add those paths.
- A recursive model-root scan would include cached dependencies under `.lake`; source containment and
  forced excludes are part of correctness, not an optimization.
- Preserve existing comments and add module/API documentation required by the Lean authoring
  guidelines.

## Acceptance

- [ ] `ModelLint.ImportGraph.check` is pure, cycle-safe, deterministic, and returns all violations with shortest qualified import paths.
- [ ] The explicit policy enforces Shared independence and every other R2 boundary with only the exact approved refinement and opt-in aggregate/tool/test exceptions.
- [ ] The adapter uses fresh authoritative Lean metadata and a canonical, symlink-safe, `.lake`-excluding inventory, failing on traversal, containment, duplicate, uncovered, or unclassified first-party modules.
- [ ] Executed synthetic tests cover every positive, negative, transitive, exception, inventory/containment, ordering, and cycle case listed in R4.
- [ ] `make lint-model` executes the test suite, asserts the controlled violation's non-zero status and deterministic stderr/path, then runs the live graph and existing declaration linters.
- [ ] The current repository passes, while the fixed controlled forbidden-edge mode exercises the same rendering and exit composition as the live driver.
- [ ] Only the superseded Feature/System grep block is removed; unrelated regression checks and comments remain intact.
- [ ] `cd model && mise exec -- lake build modelLintTests modelLint`, `make lint-model`, and the applicable full lint gate pass.
- [ ] R1-R4 are satisfied by the pure checker, complete live adapter, deterministic diagnostics, and executable regressions.
- [ ] Focused and full verification commands pass without new dependencies or default targets.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
