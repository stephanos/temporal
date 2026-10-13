---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.16 Delete the generic canonical package and its obsolete owner entries

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** S
**Files:** internal/canonicaljson implementation and two tests deleted; architecture production owner/edge tables and root owner assertion (6 core files).
**Touches:** [tools/gomad3/internal/canonicaljson/**, tools/gomad3/internal/gomadtool/architecture/architecture.go, tools/gomad3/internal/gomadtool/architecture/edges.go, tools/gomad3/architecture_test.go]

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Revival trigger (2026-10-13)

Revived after fn153.15 completed with a SHIP implementation review, proving that the last retained architecture fixtures no longer advertise or import the generic canonical-JSON owner. Every migration dependency is now done, so this task may delete the package only after a fresh integrated-tree inventory proves zero surviving production, test, fixture, or ownership consumers. fn165 retired the original module-root analyzer and its `internal/gomadtool/architecture` owner tables; use the live `internal/archcheck` and `internal/puritycheck` successors for preserved-boundary evidence, and treat any necessary successor assertion edit as a declared scope correction rather than reviving deleted analyzer code. Final approval and generated-output rotation remains owned by fn153.17. The forced local claim is required only because Flow chain admission still sees completed parent work on an unpushed remote branch; pushing remains outside the requested scope.
