---
satisfies: [R2, R3, R4, R5, R6]
---
# fn-13-deterministic-go-regression-projections.4 Publish the caller-closure projections and wire regression checks

## Description
Land the visible C5 pilot and repository workflow for R2 through R6. Generate and check in the caller-closure Go/Markdown pair, expose focused generation and clean-check targets, include the check in the stable Umpire regression gate, and document ownership and truthfulness. This is the final integration task; it does not add CI workflow files.

**Size:** M
**Files:** `Makefile`, `tools/umpire/regression/catalog_generated_test.go`, `model/Temporal/Tool/Generated/Regressions.md`, `model/README.md`
**Touches:** [Makefile, tools/umpire/regression/catalog_generated_test.go, model/Temporal/Tool/Generated/Regressions.md, model/README.md]

### Approach
- Add `umpire-gen-regression-projections` and `umpire-check-regression-projections` beside the existing model generator/regression targets. The check builds the inspector, regenerates into a fresh temporary root, byte-diffs both files, and runs focused generator/verifier tests.
- Make the stable `umpire-check-regression` target depend on or invoke the focused projection check while preserving its current registry fixtures and exact structured negative diagnostics.
- Generate the production wrapper and Markdown only through the new command. Verify the wrapper package with `-tags test_dep`, and keep the generated files small enough to serve as navigation aids.
- Extend the model README's semantic authoring/planning section with commands, owned paths, fingerprint definition, regeneration workflow, and explicit projection-only/no-runtime language. Do not absorb fn-11 architecture cleanup or fn-5 glossary/promotion docs.

### Investigation targets
**Required** (read before coding):
- `Makefile:1014-1126` — stable Umpire regression gate and clean inspector fixture checks
- `Makefile:1236-1238` — current Umpire phony-target declarations
- `model/README.md:68-105` — current authoring, inspector, and no-runtime documentation
- `model/Temporal/Tool/Inspect.lean:69-77` — production identity used by the generated pilot

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/CallerClosure.lean:11-16` — authoritative Lean source provenance
- `.plans/UMPIRE_COMPONENTS.md:215-236` — roadmap's Go/docs projection requirements

### Key context
A checked-in generated Go source must use the exact standard generated marker. `go generate` directives and GitHub Actions are not required; Make remains the repository interface. Preserve all existing Makefile and README comments while editing.

### Acceptance

## Acceptance
- [ ] The checked-in Go wrapper and Markdown are generated from the same caller-closure projection and agree on identity, format, sources, fixture, properties/requirements, and semantic fingerprint.
- [ ] The wrapper is an ordinary discoverable Go test, calls only `RequireProjection`, passes against the canonical fixture without Lean or Temporal, and contains no copied semantic procedure.
- [ ] `make umpire-gen-regression-projections` replaces the pair transactionally and `make umpire-check-regression-projections` detects either missing, renamed, or byte-modified output without rewriting it.
- [ ] `make umpire-check-regression` preserves all current Lean builds, fixture/determinism checks, and structured inspector negative cases while adding the focused projection check.
- [ ] The model README accurately documents commands, owned outputs, fingerprint derivation, stable-only scope, and that projection success is not runtime execution/evidence/conformance.
- [ ] No CI/GitHub Actions, generated Lean API drift gate, fn-5 glossary/promotion surface, fn-11 showcase artifact, or Umpire3 dependency is added.
- [ ] Focused Go tests and `make umpire-check-regression` pass with `-tags test_dep` where required.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
