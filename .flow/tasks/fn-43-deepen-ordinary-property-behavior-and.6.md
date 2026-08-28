---
satisfies: [R3, R6, R7]
---
# fn-43-deepen-ordinary-property-behavior-and.6 Introduce ordered CanonicalJson and migrate artifact codecs

## Description
Build the ordered JSON deep module required by R6 and prove it by migrating the retained planning artifact codecs. Field semantics and schema ownership stay in Artifact/Planning modules.

**Size:** M
**Files:** `model/Umpire/Json.lean`, `model/Umpire/Artifact/Codecs.lean`, `model/Umpire/Planning/Types.lean`, `model/Umpire/Artifact/Tests/Codecs.lean`, `model/Umpire/Planning/Tests/Artifacts.lean`
**Touches:** [model/Umpire/Json.lean, model/Umpire/Artifact/Codecs.lean, model/Umpire/Planning/Types.lean, model/Umpire/Artifact/Tests/Codecs.lean, model/Umpire/Planning/Tests/Artifacts.lean]

### Approach
- Deepen `Umpire.Json` with a documented ordered `CanonicalJson` value API for null, strings, naturals, arrays, ordered objects, required scalars, compact rendering, pretty rendering, and exactly-one-LF persisted bytes.
- Provide an option-to-null constructor/combinator so absent fields remain typed values rather than raw `"null"` fragments.
- Reuse Lean JSON string escaping; preserve caller-supplied object order and avoid maps or key sorting.
- Retain compatibility helpers needed by current callers, but make typed construction the path used by migrated artifact codecs.
- Migrate KnownGap `subject`/`detail` and PlannedOccurrence `authoredDefinitionId` optional fields through the typed null path, then migrate DrivePlan and ExperimentSpec while keeping field selection/order owner-local and reusing Task 1 Definition ID canonicalization.
- Prove byte equality against current fixtures, both optional-field families, independent checksum preimages, large naturals, escaping cases, compact-vs-pretty output, and terminal-newline behavior before removing string-concatenation helpers.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Json.lean:1-57` — current exact pretty/prettyBytes compatibility boundary.
- `model/Umpire/Artifact/Codecs.lean:7-180` — retained planning artifact schemas, optional authored Definition ID, field order, checksums, and string plumbing.
- `model/Umpire/Planning/Types.lean:109-123` — owner-local Known Gap JSON and optional subject/detail fields.
- `model/Umpire/Artifact/Tests/Codecs.lean:1-30` — exact stored artifact fixtures.
- `model/Umpire/Planning/Tests/Artifacts.lean:219-230` — newline, checksum, and large-natural contracts.
- `model/Umpire/Target/Tests/Compatibility/CanonicalMetadata.lean:2-15` — broader canonical metadata byte precedent.

### Key context
- fn-18 and fn-24 own artifact/receipt envelope semantics; this task owns only ordered construction/rendering and must remain byte-compatible with their contracts.
- `prettyBytes` adds exactly one LF. Do not normalize field order, parse raw JSON, or broaden `Umpire` umbrella imports.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs Umpire.Planning.Tests.Artifacts UmpireTests
```
## Acceptance
- [ ] CanonicalJson has module/public docstrings and typed constructors/renderers for null, strings, naturals, arrays, ordered objects, and option-to-null values; fields preserve supplied order, strings use Lean JSON escaping, naturals render in decimal, and persisted bytes add exactly one LF.
- [ ] Migrated Known Gap, PlannedOccurrence, DrivePlan, and ExperimentSpec codecs no longer assemble object/array/null syntax through local string concatenation while field semantics remain owner-local.
- [ ] Exact tests cover absent/present KnownGap subject/detail and PlannedOccurrence authoredDefinitionId through the typed null path.
- [ ] Exact compact/pretty JSON, fixture bytes, field order, escaping, values above machine-word range, checksum preimages, fingerprints, and terminal-newline assertions are byte-identical to the baseline.
- [ ] The new API adds no parser, unordered map, schema engine, duplicate keys in migrated codecs, or broad umbrella import.
- [ ] Focused Artifact/Planning suites and `UmpireTests` pass; existing JSON comments are preserved or expanded in place.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
