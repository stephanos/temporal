---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.38 Preserve target capability and cache digests while repairing lint

## Description
Repair the five target compatibility-import findings and three prepared-cache SHA-256 bookkeeping findings under R18/R19. This corrective owner does not widen tasks 9, 10, 11, 19 or 23.

**Size:** M
**Files:** the five capability import files, prepared_cache.go and a focused digest test file listed below.
**Touches:** [tools/gomad3/target/capability.go, tools/gomad3/target/capability_collection.go, tools/gomad3/target/capability_evaluation.go, tools/gomad3/target/capability_golden_test.go, tools/gomad3/target/capability_review_test.go, tools/gomad3/target/prepared_cache.go, tools/gomad3/target/prepared_cache_digest_test.go]

### Approach

- Admit source work only after the current predecessor candidate is integrated, independently source-reviewed and committed. Revalidate the target research bindings from .flow/tmp/fn109-target-lint-research.md against HEAD. Root owns lifecycle, review, evidence and commits; one shared-checkout source/cache writer runs at a time.
- Before production edits, retain actual unfiltered pinned target lint and add literal BASE digest controls. Five explicit compatibility aliases preserve all import uses and public projections. Replace exactly the three unchecked fmt.Fprintf calls using the existing infallible SHA-256 Write pattern at tools/gomad3/internal/sourceinventory/inventory.go:82; preserve each format and argument, without collecting the complete input stream or adding an impossible failure branch.
- Pin overlay empty/multiple replacement inputs, sorted original keys and map-order independence, replacement read failure, module-file present/absent/empty states, argument order and basename framing. Use fixed fictitious absolute original names with real temporary replacement files; originals need not exist and replacement paths stay outside the digest. Expected hashes are literal independent BASE values, never computed by the candidate under test. Preserve %s\x00%x\n and %s\x00absent\n, lowercase hexadecimal and sha256: exactly.
- Preserve all original production and fixture logic/comments/assertions outside the eight admitted statements and new controls. Retain module/source/profile identities, admission errors and go.mod/go.sum immutability. Source code, runtime overlays, pins, lint rules and compatibility grants outside Touches stay unchanged.

### Investigation targets

**Required:**
- tools/gomad3/target/prepared_cache.go:225-268
- tools/gomad3/target/capability_collection.go:188
- tools/gomad3/internal/sourceinventory/inventory.go:82
- tools/gomad3/target/capability_golden_test.go:24
- tools/gomad3/target/capability_projection_test.go:13-132

**Optional:** tools/gomad3/target/prepared_cache_test.go for native cache integration requirements.

### Quick commands

From tools/gomad3 with pinned stock Go1.27.1 first on PATH, GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS= and both Gomad seed variables unset:

```sh
go test -count=1 -tags test_dep ./target -run 'TestPreparedCacheDigest|TestCapabilityReviewGoldenCanonicalBytes|TestCompatibilityPackProjectionPreserves'
go test -count=1 -tags test_dep . -run '^TestPackageArchitecture$'
/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./target
```

Run pinned errortype, formatting and generator validation where input ownership requires it. Capture the unfiltered lint RED before edits and its exact source-bound final diagnostic delta, including residuals. The expected eight repairs and nine residual cleanup findings are hypotheses until the actual command establishes them. No filtered gate, suppression, whole-scope subtraction or native qualification claim is allowed.

Root commits reviewed source progress before admitting the cleanup writer. Original R18/R19, task21/predecessor acceptance, matched first-baseline identities, complete/full/default/functional/affected-consumer/formal and native Darwin gates remain open wherever unproved. Native Linux execution remains with fn-128 and does not block this source owner.

## Acceptance
- [ ] Literal digest and error controls pass on unchanged BASE and final source, covering every named case with independent expectations; all five capability alias changes preserve public projection/canonical behavior.
- [ ] Exactly three hash-write statements change, with unchanged formats/arguments/order and no new possible failure, full-stream copy or module mutation; original logic/comments/assertions outside scope remain intact.
- [ ] Actual unfiltered pinned target lint reproduces the eight mapped findings before edits and removes exactly those afterward; every residual and unexpected finding is retained. Focused controls, architecture, errortype, formatting and applicable generator checks pass on frozen sources.
- [ ] A fresh independent source review finds no actionable introduced defect; root commits source, controls and owned Flow evidence before another writer. Source progress is distinguished from formal qualification.
- [ ] Required original R18/R19 and predecessor/full/default/functional/affected-consumer/formal/native Darwin acceptance is proved before task completion. Task21 consumes this evidence; task21 completion is not a prerequisite for starting this corrective source work. Missing or red source-owned gates keep acceptance open; transferred Linux proof is nonblocking.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
