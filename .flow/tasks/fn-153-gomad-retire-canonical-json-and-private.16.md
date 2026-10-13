---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.16 Delete the generic canonical package and its obsolete owner entries

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** S
**Files:** internal/canonicaljson implementation and tests deleted; live architecture owner/edge tables, static source inventory, and owner assertions corrected.
**Touches:** [tools/gomad3/internal/canonicaljson/**, tools/gomad3/internal/archcheck/**]

### Approach

- Reinventory every production/test/fixture import on the integrated survivor tree; compare the original 44-caller/49-test projection, resolve every remaining caller under its existing owner before deletion.
- Delete the package only when all semantic rejection and wire controls have migrated. Remove its production owner/edge registration and root owner-exists assertion while retaining strictjson's pure owner.
- Behavior pin: post-delete architecture/module/public-signature/effect controls must preserve all surviving package boundaries. Final golden/generated validation follows on this final source tree in task17.
- Search independent canonical-named helpers by import/ownership, not name alone; retain approved boundary-diff and deterministic-I/O stdlib encoding.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/internal/canonicaljson/canonical.go`
- `tools/gomad3/internal/canonicaljson/canonical_test.go`
- `tools/gomad3/internal/canonicaljson/canonical_characterization_test.go`
- `tools/gomad3/internal/gomadtool/architecture/architecture.go:296`
- `tools/gomad3/internal/gomadtool/architecture/edges.go:14`
- `tools/gomad3/architecture_test.go:202`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 -run '^Test(PackageArchitecture|Architecture|PureModules|ExactModuleEdges|PublicPackages)' .

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Zero surviving imports/source fixtures require internal/canonicaljson; its implementation and obsolete tests/owner entries are removed.
- [ ] Strictness, invalid original-string rejection, complete identities and wire controls remain under the named migrated owners.
- [ ] Architecture/purity/public signatures and both static source sets retain their constraints; independent canonical-named domain helpers stay intact.

## Done summary
Deleted `tools/gomad3/internal/canonicaljson` after proving the integrated tree had no surviving callers. Removed its live architecture owner, permitted edges, owner assertion, and pure-source inventory entries while preserving `strictjson`, World binary-wire controls, and independent canonical-named domain helpers. The source diff removes 1,753 lines across the retired implementation/tests and makes only the bounded live archcheck corrections.

All-platform architecture and purity tests, their race/vet variants, World, strictjson, the full WASI survivor suite, and the repository-wide zero-reference proof pass. Aggregate `make -C tools/gomad3 validate` reaches only the known fn153.17-owned stale `golang-x-sys-v047-darwin-arm64` approval.

The first three-axis implementation review found one stale `internal/canonicaljson` entry in the newly amended fn166 identity inventory. That entry was removed, the focused gates reran green, and the required single-session re-review returned SHIP with no surviving findings.

stage: impl-review - ran [c8435fec78117f3daa583b1e1bebca86031cedda..5504eb0057] (model: codex:gpt-5.6-sol:xhigh)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2a93fee443067283d68615a2bbe3cf3cfd809ac8, f559c97f1872cf64fcd0fdf3a391fe0c1ee47e2d, cad9e6a86b62b00cf8540b006889fc534211f8c9, 5504eb00577f8a14f2228fdddde361b08485ec90
- Tests:
- PRs:
## Revival trigger (2026-10-13)

Revived after fn153.15 completed with a SHIP implementation review, proving that the last retained architecture fixtures no longer advertise or import the generic canonical-JSON owner. The task may delete the package only after a fresh integrated-tree inventory proves zero surviving production, test, fixture, or ownership consumers. fn165 retired the original module-root analyzer and its `internal/gomadtool/architecture` owner tables; use the live `internal/archcheck` and `internal/puritycheck` successors for preserved-boundary evidence, and treat any necessary successor assertion edit as a declared scope correction rather than reviving deleted analyzer code. Final approval and generated-output rotation remains owned by fn153.17. The forced local claim is required only because Flow chain admission still sees completed parent work on an unpushed remote branch; pushing remains outside the requested scope.

## Dependency correction (2026-10-13)

The first integrated-tree inventory found live canonical-JSON consumers in World compression, World implementation identity, Native replay, their tests, and live archcheck ownership assertions. fn153.19 explicitly owns the compressed World profile plus Native replay/inspection consumers, so fn153.16 now depends on fn153.19 and returned to Todo without implementation changes. Revive fn153.16 again only after fn153.19 completes, then rerun the zero-reference inventory before deletion.

## Second revival trigger (2026-10-13)

fn153.19 is complete with a SHIP re-review and all former World, Native replay, WASM, and simulator consumers migrated to the generated binary World envelope. A fresh integrated-tree search now finds canonicaljson only inside its own package and the live `internal/archcheck` ownership, allowed-edge, negative-control, and static-source fixtures. Delete the package and correct those live architecture fixtures; do not recreate the retired module-root analyzer. The implementation-review Phase 0 for this task already ran before the dependency correction and must not be repeated.
