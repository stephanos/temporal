---
satisfies: [R2, R8]
---
# fn-18-versioned-umpire-artifact-boundary.1 Adopt the deterministic pretty v2 Artifact baseline

## Description
Place DrivePlan and ExperimentSpec v2 behind the vertical Artifact facade with deterministic pretty
JSON as their one exact Lean/Go byte representation and no second format.


**Size:** L
**Files:** `model/Umpire/Artifact.lean`, `model/Umpire/Artifact/**`, retained v2 fixtures and their Lean consumers, `tools/umpire/internal/artifactv2/**`, regression Generated View readers/tests, and active Artifact docs
**Touches:** [model/Umpire/Artifact.lean, model/Umpire/Artifact/**, model/UmpireTests.lean, model/Umpire/Examples/Fixtures/*.json, model/Umpire/Examples/testdata/*.json, model/Umpire/Examples/*Tests.lean, model/Temporal/Feature/Nexus/Fixtures/*.json, model/Temporal/Feature/Nexus/OperationsTests.lean, model/Temporal/Feature/Nexus/Experimental/testdata/*.json, model/Temporal/Feature/Nexus/Experimental/*Tests.lean, model/Temporal/Tool/GenerateTests*.lean, tools/umpire/internal/artifactv2/**, tools/umpire/cmd/umpire-gen-regression-views/**, tools/umpire/regression/**, model/README.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Preserve the existing declarations and comments while moving implementation behind focused modules.
- Retain fn-37's v2 schemas, Definition IDs, Behavior Fingerprints, Limits, and Known Gaps while
  replacing its compact byte spelling with one deterministic pretty representation.
- Treat this as the explicit pre-release baseline correction authorized by the parent spec: it
  supersedes fn-37's compact canonical-form/checksum-preimage sentences and regenerates every v2
  checksum/fixture/view atomically; it does not introduce v3 or a compact compatibility reader.
- Share exact field order, escaping, number spelling, two-space indentation, no trailing spaces, and
  one terminal LF across Lean and Go.
- Derive each domain-separated Artifact Checksum from that document's exact pretty checksum preimage:
  omit only its own `artifactChecksum`, retain one terminal LF, and seal the nested DrivePlan before
  deriving the outer ExperimentSpec checksum.
- Treat the checked-in pretty fixtures as exact byte goldens; Generated View and fixture consumers
  use the same strict decoder rather than a semantic-equality or whitespace-normalizing adapter.
- Keep `umpire-drive-plan/v2` and `umpire-experiment/v2` as the sole supported current formats.
- Reject compact JSON, alternate whitespace/indentation, reordered keys, alternate escaping or
  number spelling, and missing/extra terminal LF as noncanonical.
- Remove the obsolete compact golden and add no compact reader, migration, alias, or fallback.

### Investigation targets
**Required:** the parent deterministic-pretty v2 contract, the committed pretty formatter baseline
`fd84945b8`, current Lean/Go codecs and checksum formulas, all retained v2 fixtures, and Generated
View ingestion.

## Acceptance
- [ ] Lean and Go emit and admit exactly the same deterministic pretty v2 bytes for DrivePlan and
  ExperimentSpec, including fixed order/escaping/number spelling, two-space indentation, no trailing
  spaces, and one terminal LF.
- [ ] Nested and outer Artifact Checksums are independently recomputed from exact pretty checksum
  preimages and every canonical pretty fixture is an exact byte golden.
- [ ] Compact JSON and every alternate whitespace/order/escaping/number/LF form reject through the
  strict production decoder; no fixture or Generated View path normalizes them.
- [ ] Public imports expose one vertical Artifact package with comments preserved.
- [ ] No earlier-format reader, alternate writer, compatibility alias, or inferred missing intent exists.
- [ ] Active Artifact documentation records that the pretty-v2 correction supersedes fn-37's
  compact spelling and that no external or immutable published v2 compatibility set exists.

### Quick command

```bash
cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs
go test -count=1 ./tools/umpire/internal/artifactv2/... ./tools/umpire/cmd/umpire-gen-regression-views/... ./tools/umpire/regression/...
make umpire-check-regression
```

## Done summary
Implemented the deterministic two-space pretty JSON v2 boundary across Lean and Go: exact pretty checksum preimages, strict byte admission, refreshed fixtures/generated views, exact-byte consumers, and active documentation. Review-driven follow-up aligned control/Unicode escaping across languages and preserved the existing public semantic comparison declaration while keeping it out of strict Artifact paths.

- Baseline: Lean Codecs and task-.1 Go suites were green; exact `make umpire-check-regression` was red pre-edit because Go rejected the compact Lean inspector output, then green after the correction. The parent-spec `go test ./tools/umpire/artifact/...` remains an inherited future-task gate because task `.2` creates that package.
- Final gates: focused Lean Codecs, task-.1 Go packages, exact `make umpire-check-regression`, affected Lean aggregates, `make lint-model`, scoped golangci-lint, and scoped Go vet all passed at reviewed head `5d16f3c52`.
- Inherited lint: full `make lint-code` remains red only for the existing case-insensitive import collision between `Tools/gomad1/api/ext-lib/nettrace` and `tools/gomad1/api/ext-lib/nettrace`; task-owned Go packages are lint/vet clean.
- Owned file set: `.plans/UMPIRE4_COMPONENTS.md`; active Artifact docs; Lean JSON/Artifact codecs and exact-byte tests/consumers; all seven retained v2 fixtures; regenerated Umpire/Temporal views; and Go `artifactv2` plus Generated View/regression tests. The obsolete compact golden was deleted.
- Review: SHIP on round 2; receipt `/tmp/fn18_task1_pretty_impl_review_receipt_20260828.json`. Review memory capture was attempted and skipped because flow-next memory is not initialized.

stage: impl-review - ran [2026-08-28T16:01:51Z..2026-08-28T16:15:45Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9dda3a6b21792feed10ba453be92780a687adca6, 5d16f3c52873156249a60443ab6779661870559a
- Tests: baseline: green (cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs), baseline: green (go test -count=1 ./tools/umpire/internal/artifactv2/... ./tools/umpire/cmd/umpire-gen-regression-views/... ./tools/umpire/regression/...), baseline: red (make umpire-check-regression failed pre-edit because strict Go admission rejected compact Lean inspector bytes), INHERITED_FUTURE_GATE: go test -count=1 ./tools/umpire/artifact/... (package is created by task .2), cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs, go test -count=1 ./tools/umpire/internal/artifactv2/... ./tools/umpire/cmd/umpire-gen-regression-views/... ./tools/umpire/regression/..., make umpire-check-regression, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect umpire-gen-tests, make lint-model, .bin/golangci-lint-v2.13.1 run --build-tags test_dep --timeout 10m --new-from-rev 45dc9c494d0e7008f97adb27ab321b389a4c2740 --config=.github/.golangci.yml ./tools/umpire/internal/artifactv2/... ./tools/umpire/cmd/umpire-gen-regression-views/... ./tools/umpire/regression/..., go vet -tags test_dep ./tools/umpire/internal/artifactv2/... ./tools/umpire/cmd/umpire-gen-regression-views/... ./tools/umpire/regression/..., INHERITED_RED: make lint-code (case-insensitive import collision: Tools/gomad1/api/ext-lib/nettrace vs tools/gomad1/api/ext-lib/nettrace)
- PRs:
