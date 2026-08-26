---
satisfies: [R1, R2, R4, R5, R6, R7, R8]
---
# fn-15-standalone-api-and-config-input-catalogs.6 Add domain adapters and the standalone input catalog executable

## Description
Compose distinct API/config catalog definitions over the shared core and implement the exact standalone command for R1/R2/R4-R8.

**Size:** M
**Files:** `model/Temporal/Tool/Catalog/API.lean`, `model/Temporal/Tool/Catalog/Config.lean`, `model/Temporal/Tool/InputCatalog.lean`, `model/Temporal/Tool/InputCatalogTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Tool/Catalog/API.lean, model/Temporal/Tool/Catalog/Config.lean, model/Temporal/Tool/InputCatalog.lean, model/Temporal/Tool/InputCatalogTests.lean, model/TemporalModelTests.lean]

### Approach
- Build separate `CatalogDefinition` adapters with API `all|current-model` and config `all|semantic-uses` views.
- Preserve complete generated API facts/selection provenance and generated config facts/six-use overlay in domain payloads.
- In production `Temporal.Tool.Catalog.Config`, import the two owner modules, concatenate their exact four-use and two-use erased lists, validate the combined registry, and derive all config selectors from that one checked aggregate.
- Map domain validation failures to stable generic envelopes without discarding the domain error/subject.
- Parse the exact CLI grammar and expose one effect-thin `IO UInt32` main.
- Pin canonical list/explain/check output and all invalid argument/selector paths; prove unclassified setting explain behavior.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Tool/Inspect.lean:23-88` and `InspectTests.lean:10-71` — current CLI boundary.
- Generated `model/Temporal/API.lean` catalog from Task 3.
- `model/Temporal/DynamicConfig/Settings.lean:6859,7546-7548` — config source catalog.
- `model/Temporal/System/API/Catalog.lean` and the separate Callback/Matching owner lists from Tasks 4/5.
- `.flow/tasks/fn-5-umpire-discovery-promotion-and-artifact.2.md` — distinct semantic catalog adapter/executable.

### Quick command
`cd model && lake env lean Temporal/Tool/InputCatalogTests.lean && lake build TemporalModelTests temporal-input-catalog`

## Acceptance
- [ ] Both domains use one generic engine but retain distinct exact payloads and format versions.
- [ ] API explain exposes projection plus selection provenance; config explain exposes generated facts plus checked-use overlay.
- [ ] The production config adapter is the only cross-owner aggregate: `all` contains the current 685 settings plus six classifications and six uses (697 selectors), while `semantic-uses` contains exactly the six named setting/classification/use triples (18 selectors) in canonical order.
- [ ] Combined validation rejects duplicate use IDs, classification keys, or setting keys across Callback and Matching before a queryable config catalog exists.
- [ ] Repeated calls and reordered inputs produce identical bytes.
- [ ] Exact valid/invalid command, view, selector, arity, stdout, stderr, status, and one-LF cases pass.
- [ ] An unclassified setting explains successfully while its absent classification is an unknown subject.
- [ ] Check consumes compiled values only and performs no filesystem, generation, source comparison, semantic evaluation, or mutation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
