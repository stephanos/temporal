---
satisfies: [R2, R21]
---
# fn-115-make-the-scala-model-the-model-and.9 Rename and regenerate the Umpire IR namespace

## Description
Rename and regenerate the Umpire IR namespace. Implements R2, R21 using the reviewed parent contracts.

**Size:** M
**Files:** proto/internal/temporal/server/api/umpire/v1; api/umpire/v1; lifter generated-class imports/packaging; Go consumers and generation dependencies
**Touches:** [proto/internal/**, api/modelir/**, api/umpire/**, model/**, tools/umpire/**, tests/**, Makefile, .plans/umpire-migration-*.json]

### Approach
- Rename the IR proto package/options and buffer generation configuration together, then regenerate Go and existing Java/JVM classes through the established owners. Do not hand-edit generated code or introduce ScalaPB early.
- Update all live schema consumers, lifter class imports and packaging inputs. Keep field numbers/types, enum values, presence, oneofs and wire encoding fixed; add structural descriptor and representative wire compatibility checks.
- Verify schema-input invalidation prevents reuse of an old jar. Run the full semantic/artifact golden set, keeping type-name changes separate from behavioral results.
- If an unavoidable change exceeds R21's permitted identity/type-name transformation, follow its explicit fallback with evidence and map revision; do not silently normalize it.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `proto/internal/temporal/server/api/modelir/v1/ir.proto:12`
- `proto/internal/buf.yaml:17`
- `model/scalav2/gen.sh`
- `model/scalav2/lifter/Lift.scala`
- `model/scalav2/goir/load.go`
- `Makefile`

### Quick commands
make proto; the renamed model generation/check commands; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/...; make lint-scala; make lint-code-fast

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.

## Acceptance
- [ ] Live proto/Go/generated JVM classes use umpire/v1, with regenerated owners and complete import/packaging updates.
- [ ] Descriptor structure and representative wire bytes remain compatible; stale schema-generated jars are invalidated.
- [ ] Only the reviewed naming/path transformation affects artifacts; all semantic goldens and consumers pass, or the explicit R21 fallback is evidenced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
