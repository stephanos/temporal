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
TBD
## Evidence
- Commits:
- Tests:
- PRs:
