**Verification caveat first:** this session has no shell, so I could not run the Quick commands or the golden test. Everything below is from reading the diff, the current files, and the compatibility, livecap, hostfs and architecture helpers the change depends on. The `make validate-compatibility` acceptance item is therefore unverified here.

**What I confirmed by tracing**

- The moved code is byte-for-byte the old code in every function I compared: collection (`projectCapabilityPackage`, `loadBuildOverlay`, `validateAdapterReplacementInputs`, `matchAdapterReplacement`), structure validation, linked projection, and the inventory digest including its limits and error strings.
- `loadCompatibilityPolicy` plus `SelectPacksForPlatform(packs, pkgs, GOOS/GOARCH)` is exactly what the old `compatibility.Select` did internally (`v2_selection.go:183-185`), so pack selection is unchanged and the `select target compatibility packs:` prefix is preserved on both load and select failures.
- Error order for fresh reviews (source errors, then pack load, then identity, then structure, then main-package) and for recorded closures (schema/identity, then pack load, then structure) matches the old sequence.
- The simulation-bridge check is equivalent: `MainModule` mirrors `Module.Main`, `Policy.Module.Replaced` mirrors `Replacement != nil`, and a nil module yields an empty path and is rejected as before.
- Finding construction, source name/digest attachment per kind, and final sort are equivalent to the old `collectCapabilityFindings`.
- `AdapterCapacityError` mapping is kept in both consumers and the typed error survives the `%w` wrap in `matchAdapterReplacement`.
- `DigestAdapterSourceInventory` has no remaining callers outside the preimage artifact and the architecture assertion that it is gone.
- `TestCapabilityEvaluationHasNoHostEffect` is consistent with the evaluation file's actual imports and calls, and every helper it calls (`listHostPackages`, `packageExports`, `go/ast` imports) already exists.
- All identifiers the golden test uses (`livecap.FactKind*`, `Disposition*`, `record.SHA256`, `canonicaljson.CanonicalJSON`, `requireTestNoError`) exist.

## Findings

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `tools/gomad3/target/capability_evaluation.go:213`
- **R-IDs**: [R17]
- **Problem**: `evaluateCollectedCapabilities` runs `evaluateCapabilityPolicy` to obtain the selection identities, then calls `evaluateCapabilityClosure`, which runs the same selection and the full findings pass again over the same packages and discards the first result. The old code selected twice but computed findings once.
- **Suggestion**: Compute the evaluation once, derive the closure identities from it, run the identity and structure checks, and project from that same evaluation. `reviewRecordedClosure` can keep its own path since it must check identity before loading packs.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `tools/gomad3/internal/sourceinventory/inventory.go:97`
- **R-IDs**: [R17]
- **Problem**: `readBoundedRegularFile` is copied verbatim from `tools/gomad3/target/target.go:935`. The neutrality rule (sourceinventory may import only hostfs) forces this copy as written.
- **Suggestion**: Move the bounded reader into `internal/hostfs`, which both `target` and `sourceinventory` already import, and have both call it. Duplicated Code smell.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `tools/gomad3/architecture_test.go:187`
- **R-IDs**: [R17]
- **Problem**: `checkPureFile` only inspects direct call expressions. A function value taken without a call (`reader := os.ReadFile`) or a package-level function from another file passed as a value would pass the purity gate. It also treats any identifier call as a cross-file call by name, so a future local variable that shadows a package-level function name would be a false positive.
- **Suggestion**: Also inspect `*ast.SelectorExpr` and `*ast.Ident` nodes outside call position for imported and cross-file function references, or at minimum note the limitation in the helper's comment. The recorded mutation check covers the current code, so this is hardening, not a defect.

## Pre-existing issues (not blocking this verdict)

- [P3, confidence 100, introduced=false] `tools/gomad3/target/capability_collection.go:1771` — `collectCapabilityPackages` keeps the variadic `replacementSets ...[]AdapterReplacement` with a runtime error for more than one set. This is carried over from the old `projectCapabilityReview` and is speculative generality; a plain `[]AdapterReplacement` parameter would do.

**FYI, suppressed:** the design record states `TestBuiltInSimulationLinknamesPinCurrentFirstPartySources` and `TestClosureReviewSupportsSimulationFixtureAndRefusesHarnessTests` fail on the base revision because `runtime_time_toolchain.go` gained a directive without a pin update. I could not run them here. The task explicitly declines to repin, which matches the acceptance criterion that pins stay unchanged.

**Smells noted, not filed:** `capabilitypolicy.Package` carries `ImportPath` both at top level and inside `Policy`, and `MainModule` beside `Policy.Module`. A small data clump, but the split keeps `compatibility.Package` as the only policy-matching identity.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R17 | met | Separate private owners in `capability_collection.go`, `capability_evaluation.go` plus `target/internal/capabilitypolicy`, and `capability_linked.go` behind unchanged `ReviewCapabilities` / `ReviewCapabilityClosure`. Evaluator reuses `SelectPacksForPlatform`, no new evaluator. `internal/sourceinventory` registered as owner and imported by both `target` and `deterministicio`. Golden canonical reviews and inventory digest pin added. Fail-closed error paths are moved verbatim. Tests not executed in this session. |

Unaddressed R-IDs: []

Suppressed findings: 1
Classification counts: 3 introduced, 1 pre_existing.

```json
{"suppressed_count":{"50":1},"classification_counts":{"introduced":3,"pre_existing":1},"unaddressed":[]}
```

<verdict>SHIP</verdict>

VERDICT=SHIP
