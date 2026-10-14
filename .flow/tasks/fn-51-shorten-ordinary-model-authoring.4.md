---
satisfies: [R3, R5, R6]
---
# fn-51-shorten-ordinary-model-authoring.4 Add and migrate Space leaf constructors

## Description
Add focused inert constructors for ordinary Space leaves and migrate the production example/fixtures (R3, R5, R6).

**Size:** M
**Files:** `model/Umpire/Space/Language.lean`, `model/Umpire/Space/Tests/Fixtures.lean`, `model/Umpire/Space/Tests/Intent.lean`, `model/Umpire/Space/Tests/Validation.lean`, `model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean`, `model/Temporal/Feature/Nexus/Experimental/VariationSpaceTests.lean`
**Touches:** [model/Umpire/Space/Language.lean, model/Umpire/Space/Tests/Fixtures.lean, model/Umpire/Space/Tests/Intent.lean, model/Umpire/Space/Tests/Validation.lean, model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean, model/Temporal/Feature/Nexus/Experimental/VariationSpaceTests.lean]

### Approach
- Add plain constructors for baseline, bound value, selected fault, fault axis, fault at occurrence, and seek goal beside their existing records.
- Fill only established inert defaults; require every semantic identity/source/input explicitly.
- Preserve negative tests as raw records and prove checked Space, point order, fault requests, and metadata are unchanged.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Space/Language.lean:7-70` — existing leaf record shapes and defaults
- `model/Umpire/Space/Tests/Fixtures.lean:33-88` — minimal repeated authoring
- `model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean:127-190` — production baseline/fault axes and goals
- `model/Umpire/Space/Tests/Intent.lean` — request-only and seek-only semantics
- `model/Umpire/Space/Tests/Validation.lean` — invalid leaf diagnostics
- `model/Temporal/Feature/Nexus/Experimental/VariationSpaceTests.lean` — canonical production behavior

## Acceptance
- [ ] Each leaf constructor is documented, inert, explicit about semantic inputs, and equal to its prior record shape.
- [ ] Production Variation Space and shared positive fixtures use the constructors; raw negative fixtures remain direct.
- [ ] Existing Space errors for baseline/effect/reference/fault/coverage/bound failures are unchanged.
- [ ] Checked Space identity, point order, fault intent, coverage goals, artifacts, and focused tests are unchanged.

## Done summary
Added six documented inert Space leaf constructors with direct record-equivalence coverage, then migrated ordinary shared fixtures and production VariationSpace/CallerClosureFault declarations. Explicit-role, incompatible-fault, negative, mutation, boundary, and equivalence-oracle records remain raw; Space diagnostics, identity, ordering, intent, coverage, comments, and artifacts remain unchanged.
## Evidence
- Commits: 7f229aa5887f5d339df00eb3a1d49e456bf8dd5d
- Tests: cd model && mise exec -- lake build Umpire.Space.Tests.Compilation Umpire.Space.Tests.Determinism Umpire.Space.Tests.Intent Umpire.Space.Tests.Metadata Umpire.Space.Tests.Validation Temporal.Feature.Nexus.Experimental.VariationSpaceTests Temporal.Feature.Nexus.Experimental.CallerClosureFaultTests, make lint-model, CGO_ENABLED=0 mise exec -- go test -count=1 -tags test_dep ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$'
- PRs: