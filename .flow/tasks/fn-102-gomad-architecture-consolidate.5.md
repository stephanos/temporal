---
satisfies: [R5]
---
# fn-102-gomad-architecture-consolidate.5 Enforce package coverage, purity, and public signature visibility

## Description
R5. listHostPackages currently enumerates only chosen roots, so ownerless new roots can escape. Discover all host packages while explicitly excluding patched stdlib overlays, hidden toolchain builds and independent fixture modules. Inspect darwin/arm64 and linux/amd64 source sets via go list; this is static inspection, not cross-platform execution. Preserve owner/import restrictions and add narrowly scoped host-effect restrictions to pure World, seed controller and exploration engines; pure files inside a mixed package need file-level checks rather than banning legitimate campaign storage effects. Use typed exported API inspection or an equivalent correct traversal to detect inaccessible nested-internal signature types; permit types legally accessible to the intended consumer (avoid a blanket /internal/ ban). Add negative fixtures for ownerless package, forbidden host dependency, forbidden internal edge and inaccessible public type. Task 4 removes the current violation first. Preserve existing comments. No new dependencies. Run focused commands from tools/gomad3 with GOWORK=off and -tags test_dep; use the patched toolchain for runtime-consumer tests.

**Size:** M

**Touches:** [tools/gomad3/architecture_test.go, tools/gomad3/architecture_*_test.go, tools/gomad3/testdata/architecture/**]

**Files:** `tools/gomad3/architecture_test.go`; small checker test fixtures/helpers; architecture documentation input for task 6.

### Quick commands

`go test -tags test_dep .`

## Acceptance
- [ ] R5 covers all host packages for both source platforms with justified exclusions.
- [ ] Negative fixtures fail for each named architectural violation and positive platform fixtures pass.
- [ ] Pure code cannot acquire selected host effects unnoticed; mixed storage packages keep legitimate I/O.
- [ ] Visibility checker traverses intended public method/field signatures and allows legal internal-package access.

## Done summary
NOT IMPLEMENTED. Moved to fn-105-gomad-follow-ups-deferred-scope.4 (D4) on 2026-09-29 as a scope cut; the task text above remains the implementation brief.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: