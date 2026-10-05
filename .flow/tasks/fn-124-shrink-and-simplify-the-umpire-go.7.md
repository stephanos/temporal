---
satisfies: [R7]
---
# fn-124-shrink-and-simplify-the-umpire-go.7 Retire the migration harness and frozen snapshots

## Description
Implements R7. Retire the migration harness and frozen snapshots.

Ordering (host for the owner, 2026-10-05): runs beside fn-126.5, before fn-126.6. Since fn-126.4, fn-126's R5 equality is proved by the reader projection (tables and every Check receipt, `.flow/tmp/fn-126/fn126-4/projtool`, plus `project4.py`), which the review confirmed is a true comparison; the harness no longer carries that proof, and retiring it before fn-126.6 saves re-capturing a baseline decision 23 would rewrite entirely. fn-126.5 may still add `config.json` entries meanwhile; whichever lands second resolves (deletion wins).

### Approach: inventory to delete
- **Harness:** `tools/umpire/internal/golden/` in full, 4,586 Go lines: `golden.go`, `original.go`, `waits.go`, `job.go` and their tests, `config.json` (34 KB), `original.json` (11 KB), `testdata/original` (264 KB).
- **Reader migration tests:**
  - `tools/umpire/model/migration_golden_test.go` (1,218 lines) and `original_migration_test.go` (391).
  - `model/testdata/migration/` (22.7 MB), including `semantics/ir/nexus-close.json.gz` (19.35 MB).
  - Keep `testdata/schema`.
- **Lowering migration tests:**
  - `tools/umpire/lower/{migration_golden,migration_job,migration_projection,migration_fixture,migration_fixture_export,migration_oracles,original_migration}_test.go` (1,968 lines).
  - `lower/testdata/migration/{original,mapped,oracles}` (6.2 MB on disk).
- **Pins that check frozen values:**
  - `TestALoweredCaseIsTheComparativeGoModelsCase` (`lower/lower_test.go:255-320+`, compared to the archived comparative Model);
  - `lower/producer_contract_test.go:20`;
  - `model/schema_test.go:250-270` (frozen `inputs`);
  - `model/nexus_close_baseline_test.go`, unless rewritten as below.
- **Docs and gates:**
  - `MILESTONES.md:45-49` (the original-baseline command);
  - `tools/umpire/README.md:19,61`;
  - the module map lines that mention golden/migration;
  - the `layout_test.go:55,89` exemptions;
  - the `golden.go` rows in `model/ownership_test.go:355-360`.

### Approach: rewrite against live IR first, in its own commit before deletion
- **`model/activity_properties_test.go`:** `frozenPropertyRows` (l.88-133), `pathDisagreements` (l.348-366) and `twinDisagreements` (l.370-385) compare live answers with frozen receipts. Restate them as explicit expectations over `model/ir/activity.json`: per-property row counts, the counterexample rows and each path Query's answer kind and witness, as `activity_pins_test.go` already does.
- **`model/activity_parity_test.go:34-41`** (evidence in catalog order): assert the order against the live IR's catalog.
- **`model/nexus_close_baseline_test.go`:** replace `frozenReaderModel/Meaning("nexus-caller")` (l.59, 85, 107) with the live `model/ir/nexus-caller.json` built by `Build`/`Check`.
- **`model/choices_test.go:79,115-117,177`:** compare two live encodings with `sha256` inline instead of `golden.Digest`/`migrationMeaning`.
- **Relocate the generic helpers:**
  - `golden.Root` (used by `model/{ownership,replay}_test.go` and `cmd/umpire-ir-bridge/protocol_test.go`) becomes a small `tools/umpire/internal/testroot` or a local function.
  - `golden.JobModel` moves into `lower/internal/producer/occurrence_test.go` and the lower job tests that stay.
  - `golden.Read` serves `model/schema_test.go:161` (`testdata/schema/before-rename`), so it moves with that test.
- Keep `model/table_support_test.go`'s row helpers and drop its `readerBaseline` (l.99-127).
## Acceptance
- [ ] Before anything is deleted, each test listed under "rewrite" reads only live `model/ir` (or fixtures it builds) and passes. The commit is separate, and each rewritten test is shown to fail on a seeded change to the live IR.
- [ ] `tools/umpire/internal/golden`, `model/testdata/migration`, `lower/testdata/migration` and the migration, original-baseline and frozen-pin tests listed above are gone. `grep -rn "internal/golden\|testdata/migration\|OriginalBaseline\|OriginalDelta" tools tests common model MILESTONES.md .plans/UMPIRE_MODULES.md` is empty.
- [ ] The done summary reports Go test lines and git-tracked testdata bytes before and after for `tools/umpire`, using `git ls-files | xargs wc -c`. 1,457 files are tracked under those three directories today.
- [ ] `layout_test.go` and `ownership_test.go` no longer exempt or list the harness, and still enforce their other rules.
- [ ] `MILESTONES.md`'s verification instructions and `tools/umpire/README.md` describe the gate without the original-baseline check.
- [ ] Model gate, Go tooling suite (`-p 2`, timed with `-json` per MILESTONES), Testpilot tests, `make lint-code-fast` and `make umpire-check-cases` pass with Case bytes unchanged.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
