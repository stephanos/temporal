---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.9 Migrate CLI JSON delivery and retain the public upgrade facade

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** cmd/gomad/internal/cli/{analyze,compare_support,qualify_set}.go; upgrade/maintenance_compat.go and pinimpact/render.go with focused tests (5 core callers).
**Touches:** [tools/gomad3/cmd/gomad/internal/cli/analyze*.go, tools/gomad3/cmd/gomad/internal/cli/compare_support*.go, tools/gomad3/cmd/gomad/internal/cli/qualify_set*.go, tools/gomad3/cmd/gomad/internal/cli/cli_test.go, tools/gomad3/upgrade/maintenance_compat*.go, tools/gomad3/upgrade/upgrade_test.go, tools/gomad3/upgrade/pinimpact/**]

### Approach

- Use direct stdlib encoding at delivery boundaries and keep semantic report projection/encoding error classification.
- Retain upgrade.PinImpact.CanonicalJSON as an exposed facade using stdlib internals; package removal does not authorize an API rename.
- Behavior pin: table-driven command/flag/default, stdout/stderr/status and infrastructure/report-delivery error outcomes. Replace only ordinary-byte goldens with decoded semantic comparisons.
- Leave upgrade's independent canonicalBoundaryJSON approval-normalization helper and directory/adapter transactions with their owners.
- Migrate shared CLI fixture encodes in cli_test.go and upgrade_test.go here. Boundary-diff tests use their unchanged domain normalizer; ordinary delivery fixtures use stdlib with semantic assertions.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/cmd/gomad/internal/cli/analyze.go:220`
- `tools/gomad3/cmd/gomad/internal/cli/compare_support.go:65`
- `tools/gomad3/cmd/gomad/internal/cli/qualify_set.go:169`
- `tools/gomad3/upgrade/maintenance_compat.go:93`
- `tools/gomad3/upgrade/pinimpact/render.go:14`
- `tools/gomad3/upgrade/upgrade.go:433`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomad/internal/cli ./upgrade/pinimpact ./upgrade

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.
## Acceptance
- [ ] CLI grammar/defaults, public streams/status/error classifications and report-delivery outcomes match the frozen pin while JSON ordering may change.
- [ ] The exposed CanonicalJSON method still compiles for external callers with current semantic output; no forwarding/API cleanup broadens scope.
- [ ] Invalid typed strings and encoder/delivery failures retain owned errors; independent boundary approval semantics remain unchanged.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
