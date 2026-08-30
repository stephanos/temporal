---
satisfies: [R1, R2]
---
# fn-45-index-and-reconcile-umpire-plan.1 Build the strict read-only plan index checker

## Description
Implement the reusable parser and pure validation core for R1/R2. Keep repository discovery and diagnostics at a thin command boundary.

**Size:** M
**Files:** `tools/planindex/main.go`, `tools/planindex/index.go`, `tools/planindex/check.go`, `tools/planindex/index_test.go`, `tools/planindex/check_test.go`
**Touches:** [tools/planindex/**]

### Approach
- Follow small command/package separation used under `tools/umpire/cmd/` while keeping the validation core independently testable.
- Decode the closed v1 schema token-by-token so duplicate keys and unknown fields fail; use sorted slices for all rendered output.
- Validate confined normalized paths, complete document coverage, authority references/cycles, Markdown links/anchors, and Flow JSON against fixture roots.
- Never call a mutating command or infer lifecycle/disposition from prose.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md:3-14` — normative authority.
- `.plans/UMPIRE4_ORDER.md:28-31,223-261` — reduced scope, gate, and dispositions.
- `.flow/specs/fn-23-veil-toolchain-compatibility-and.json` — tracked readiness shape.
- `tools/umpire/cmd/umpire-check-legacy-vocabulary/main.go` — focused checker command pattern.

**Optional** (reference as needed):
- `tools/common/artifactio/` — repository confinement/error test patterns; reuse only if it fits without expanding scope.

### Quick commands
`go test -count=1 -tags test_dep ./tools/planindex/...`
## Acceptance
- [ ] Strict parser rejects malformed/duplicate-key/unknown-field/version/enum/type/nullability input with deterministic diagnostics.
- [ ] Pure checks cover complete document and Flow-spec registration, graph integrity, links/anchors, confined paths, exact Flow state/dependencies, and cross-field invariants.
- [ ] Success and multi-error output are byte-stable across reordered fixture input.
- [ ] All checks are read-only and use no new third-party library.
- [ ] Go tests use `require` and whole-value comparisons.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
