---
satisfies: [R1, R4]
---
# fn-130-model-views.3 Render signatures and composition syncs

## Description
Add the signature and composition view families behind the established renderer. Keep their ownership disjoint from refinement and derived comparisons.

**Size:** M
**Files:** `tools/umpire/render/signature.go`, `tools/umpire/render/signature_test.go`, `tools/umpire/render/composition.go`, `tools/umpire/render/composition_test.go`
**Touches:** [tools/umpire/render/signature*, tools/umpire/render/composition*]

### Approach
- Use current Action.actor/on/creates/schema/example/input metadata and existing stable IDs; include timers/internal actors and entities without declaration-side configuration.
- Resolve composition members and paired sync actions with current replaces metadata. Attribute only available composition and Action positions; Sync has no independent source position.
- Test repeated labels, source characters, two identical action names from different owners, missing references, empty member/action sets and declared substitution metadata. Register through the existing renderer's established convention.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/umpire/v1/machine.proto:18` - Action metadata.
- `proto/internal/temporal/server/api/umpire/v1/machine.proto:167` - members, replaces and Sync carriers.
- `tools/umpire/check/compose.go` - current member/sync identities.

### Quick commands
```bash
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/render -run 'Test(Signature|Composition)'
```

## Acceptance
- [ ] Signature and sync golden views retain actor/entity/input/schema/example/substitution data and truthful positions.
- [ ] Empty, malformed, duplicate-label and owner-collision controls are attributed and deterministic.
- [ ] Both D2/SVG families reproduce byte-identically using .2's rendering settings.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
