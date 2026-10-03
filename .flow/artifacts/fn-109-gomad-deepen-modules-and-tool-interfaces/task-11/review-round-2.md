**Verification caveat:** this session still has no shell, so the Quick commands and the golden test were not run. All resolutions below were verified by reading the current files on disk and the new diff receipt.

Prior finding #1: fixed
Prior finding #2: fixed
Prior finding #3: fixed
Prior finding #4: not-fixed

**How each was verified**

- **Finding 1, double evaluation.** `evaluateCollectedCapabilities` now runs `evaluateCapabilityPolicy` once, builds the closure from that selection, and hands the same evaluation to the new shared `reviewEvaluatedClosure`, which does the identity-equality, structure and main-package checks. The recorded-closure path keeps its own single evaluation. Error order is unchanged from the base sequence.
- **Finding 2, duplicated bounded reader.** The reader now lives once as `hostfs.ReadBounded` with its own unit test. The copy in `target/target.go` is deleted, and every former caller in `capability_collection.go`, `prepared_cache.go` and `sourceinventory` calls the hostfs function. `target.go` still uses `hostfs.OpenPath` elsewhere, so its import stays valid.
- **Finding 3, purity walker only saw calls.** `checkPureFile` now inspects every selector through an import regardless of position (type, conversion, value, call) and every identifier against the cross-file function map, including shadowing. I checked the walker against the files it inspects: every `compatibility.X` and `capabilitypolicy.X` reference in the evaluation file and the policy package is in the updated allowlists, and no identifier in the evaluation file collides with a package-level function declared in another `target` file. The test should pass as written, though I could not execute it.
- **Finding 4, pre-existing variadic `replacementSets`.** Still present in `collectCapabilityPackages` and `projectCapabilityReview`. It was carried over from the base implementation and is non-blocking.

**New hunks introduced by the fixes**

The `prepared_cache.go` and `target.go` changes are mechanical call-site substitutions plus the deletion of the old reader. No behaviour or error text changes. Nothing new rises to Major.

## Pre-existing issues (not blocking this verdict)

- [P3, confidence 100, introduced=false] `tools/gomad3/target/capability_collection.go:62` — `collectCapabilityPackages` keeps the variadic `replacementSets ...[]AdapterReplacement` with a runtime error for more than one set. Speculative generality carried over from the base `projectCapabilityReview`; a plain slice parameter would do.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R17 | met | Separate private owners for collection, pure evaluation and linked projection behind the unchanged review contract; evaluator reuses `SelectPacksForPlatform`; `internal/sourceinventory` is the single registered inventory owner consumed by `target` and `deterministicio`; golden canonical reviews and the inventory digest pin are present; fail-closed error paths are moved verbatim. Tests not executed in this session. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 1 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":1},"unaddressed":[]}
```

<verdict>SHIP</verdict>

VERDICT=SHIP
