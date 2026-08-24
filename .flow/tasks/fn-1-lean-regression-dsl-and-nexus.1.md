---
satisfies: [R1, R2, R3, R4, R5, R7]
---
# fn-1-lean-regression-dsl-and-nexus.1 Define Lean regression compiler contracts

## Description
Build the closed pure-Lean compiler foundation for R1-R5 and R7. This is the early proof task: it settles the reusable contract and structural validation before any Temporal-specific binding is added.

**Size:** M
**Files:** `model/Temporal/Experiment/DSL.lean`, `model/Temporal/Experiment/Compiler.lean`, `model/Temporal/Experiment/Json.lean`, `model/Temporal/ExperimentTests.lean`, `model/lakefile.toml`
**Touches:** [model/Temporal/Experiment/DSL.lean, model/Temporal/Experiment/Compiler.lean, model/Temporal/Experiment/Json.lean, model/Temporal/ExperimentTests.lean, model/lakefile.toml]

### Approach
- Define typed resource/action/property identities, a resolved setup, declaration-size bounds, named precedence edges, a non-empty expected-property collection, `Regression`, `ModelTarget`, `ExperimentSpec`, and a closed `CompileError` kind.
- Make each target action projection setup-dependent: a mapped projection returns a model-owned outcome or a stable `impossibleAction` error. Keep `compile` pure, validate in deterministic order, and preserve action attempts separately from projected outcomes.
- Derive model identity from the canonical compiled target slice—the declaration, resolved setup, projected outcomes, and property observation contracts consumed by the regression—without accepting a caller hash. Canonicalize collection and JSON field order without relying on map iteration or `Repr`.
- Add synthetic positive and negative fixtures that exercise every structural error class independently, including an empty expectation collection and a mapped-but-inapplicable action.
- Register only the focused test target needed to prove the modules elaborate; defer the Nexus binding and public inspector to task `.2`.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Proto/Core.lean:3-80` — small current-model declaration style and derivations.
- `tools/umpire/internal/generate/api/render.go:37-95` — established deterministic sorting and provenance pattern outside Umpire3.
- `tools/umpire/internal/generate/api/main_test.go:58-109` — ordering-equivalence and conflict-error test shape.
- `model/lakefile.toml:1-2` — current Lean library and build targets.

**Optional** (reference as needed):
- `model/Temporal.lean:1` — current authored model entrypoint.

### Key context
- Resource/action/property names are unique within their own kind; precedence rejects duplicate edges, self edges, and cycles.
- The expected-property collection is non-empty. Resource and action bounds are positive; the edge bound may be zero.
- Semantic identity covers canonical contract data consumed by compilation, not arbitrary Lean source text; proof-only refactors preserving the contract intentionally keep the same identity.
- Do not inspect, cite, search, import, copy, adapt, or depend on any Umpire3 source or artifact.
- Preserve all existing comments in touched files.

### Task-scoped verification
- The baseline and completion command for this task is `make -C model check`, plus the focused `ExperimentTests` build.
- `make umpire-check-regression` is a final spec Quick command whose top-level target is created by task `.3`; its absence is expected until that task and must not block `.1`.

## Acceptance
- [ ] `compile` exposes the planned pure Lean success/error contract with no file or runtime side effects.
- [ ] A synthetic valid declaration with multiple expectations resolves into one complete `ExperimentSpec` with canonical JSON, preserving attempts separately from projected outcomes.
- [ ] Tests assert stable error kind and subject for missing/duplicate identities, empty expectations, unresolved references, target mismatch, unmapped action, mapped-but-inapplicable action, duplicate/self/cyclic ordering, and exceeded bounds.
- [ ] Equivalent declarations with different incidental input order serialize identically; changing a consumed projected outcome or property observation contract changes model identity and output.
- [ ] The focused Lean test target elaborates without new third-party dependencies.
- [ ] New compiler/test/build files contain no Umpire3 import, reference, copied contract, or dependency.

## Done summary
Defined the pure Lean regression DSL, deterministic compiler, semantic identity, and canonical JSON contract, with setup-dependent action projection and bounded linear ordering validation. Focused fixtures cover the complete synthetic artifact plus missing/duplicate identities, empty expectations, typed unresolved references, target mismatch, unmapped/impossible actions, duplicate/self/cyclic ordering, invalid/exceeded bounds, incidental-order equivalence, dense acyclic ordering, and independent outcome/property identity drift; baseline was green for both task-scoped commands.

stage: impl-review - ran [2026-08-24T16:16:08Z..2026-08-24T16:24:48Z]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5d821ccaee6ca3f138c7d439f6df346704a1c09d, 803e3859e4deb60334c492446d1223ae679b9c9e
- Tests: make -C model check, make umpire-check-api
- PRs:
