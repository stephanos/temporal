---
satisfies: [R2, R3, R4, R5, R6]
---

# fn-13-deterministic-go-regression-projections.4 Publish the caller-closure projections and wire regression checks
## Description
Land the visible C5 pilot and repository workflow for R2 through R6. Generate and check in the caller-closure Go/Markdown pair, expose focused generation and clean-check targets, include the check in the stable Umpire regression gate, and reconcile the model and component-status documentation. This is the final integration task; it does not add CI workflow files.

**Size:** M
**Files:** `Makefile`, `tools/umpire/regression/catalog_generated_test.go`, `model/Temporal/Tool/Generated/Regressions.md`, `model/README.md`, `.plans/UMPIRE_COMPONENTS.md`
**Touches:** [Makefile, tools/umpire/regression/catalog_generated_test.go, model/Temporal/Tool/Generated/Regressions.md, model/README.md, .plans/UMPIRE_COMPONENTS.md]

## Approach
- Add `umpire-gen-regression-projections` and `umpire-check-regression-projections` beside the existing model generator/regression targets. The check builds the inspector, regenerates into a fresh temporary root, byte-diffs both files, and runs focused generator/verifier tests.
- Make the stable `umpire-check-regression` target depend on or invoke the focused projection check while preserving its current registry fixtures and exact structured negative diagnostics.
- Generate the production wrapper and Markdown only through the new command. Verify the wrapper package with `-tags test_dep`, and keep the generated files small enough to serve as navigation aids.
- Extend the model README's semantic authoring/planning section with commands, owned paths, provenance-root conversion, fingerprint definition, regeneration workflow, and explicit projection-only/no-runtime language.
- Reconcile the C5 component-status section with the landed one-regression surface, actual Make interface, checked-in outputs, and remaining limitations. Preserve the document's broader current-status changes; do not absorb fn-11 architecture cleanup or fn-5 glossary/promotion docs.

## Investigation targets
**Required** (read before coding):
- `Makefile:1014-1126` — stable Umpire regression gate and clean inspector fixture checks
- `Makefile:1236-1238` — current Umpire phony-target declarations
- `model/README.md:68-105` — current authoring, inspector, and no-runtime documentation
- `.plans/UMPIRE_COMPONENTS.md:215-236` — current C5 status and proposed interface to reconcile
- `model/Temporal/Tool/Inspect.lean:69-77` — production identity used by the generated pilot

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/CallerClosure.lean:11-16` — authoritative model-root-relative Lean source provenance

## Key context
A checked-in generated Go source must use the exact standard generated marker. `go generate` directives and GitHub Actions are not required; Make remains the repository interface. Preserve all existing Makefile, roadmap, and README comments and unrelated current-status edits while editing.

## Acceptance
- [ ] The checked-in Go wrapper and Markdown are generated from the same caller-closure projection and agree on identity, format, canonical and repository-facing sources, fixture, properties, observation requirements, and semantic fingerprint.
- [ ] The wrapper is an ordinary discoverable Go test, calls only `RequireProjection`, passes against the canonical fixture without Lean or Temporal, and contains no copied semantic procedure.
- [ ] `make umpire-gen-regression-projections` replaces the pair transactionally and `make umpire-check-regression-projections` detects either missing, renamed, or byte-modified output without rewriting it.
- [ ] `make umpire-check-regression` preserves all current Lean builds, fixture/determinism checks, and structured inspector negative cases while adding the focused projection check.
- [ ] The model README accurately documents commands, owned outputs, provenance roots, fingerprint derivation, stable-only scope, and that projection success is not runtime execution/evidence/conformance.
- [ ] The C5 roadmap status names the actual Make interface and checked-in pilot outputs, no longer says the generator is unimplemented, and retains remaining gaps and the projection-only boundary.
- [ ] No CI/GitHub Actions, generated Lean API drift gate, fn-5 glossary/promotion surface, fn-11 showcase artifact, or Umpire3 dependency is added.
- [ ] Focused Go tests and `make umpire-check-regression` pass with `-tags test_dep` where required.

## Done summary
Published the stable caller-closure regression as one transactionally generated Go/Markdown projection pair, added focused generation and clean-check targets to the root Makefile and stable regression gate, and documented ownership, provenance, fingerprinting, and the projection-only boundary. Reconciled C5 in the live renamed `.plans/UMPIRE4_COMPONENTS.md` roadmap without recreating its obsolete predecessor.

baseline: green (`go test -count=1 -tags test_dep ./tools/umpire/...` passed pre-edit; `make umpire-check-regression` reused receipt e1a635b2)
GATE_SKIPPED:unittest:green-receipt e1a635b2 - baseline reused from prior post-gate pass
stage: impl-review - ran [2026-08-26T05:28:28Z..2026-08-26T05:31:45Z] (SHIP)
## Evidence
- Commits: 222a9447c7af154630ff98a58f2b2c927e241f0d
- Tests: go test -count=1 -tags test_dep ./tools/umpire/..., make umpire-check-regression-projections, make umpire-check-regression, make umpire-gen-regression-projections (silent success; repeated generation byte-identical), GATE_SKIPPED:unittest:green-receipt e1a635b2 - baseline reused from prior post-gate pass
- PRs: