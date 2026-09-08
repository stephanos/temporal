---
satisfies: [R1, R2, R3, R4]
---
# fn-70-scheduled-canary-proof-of-concept-as-a.4 Build pinned check catalog and validate complete selections before mutation

## Description
Build pinned check catalog and validate complete selections before mutation.

**Size:** M
**Files:** tools/canary/catalog.go; tools/canary/catalog_test.go; tools/canary/config.go; tools/canary/config_test.go; tools/canary/dependencies_test.go
**Touches:** [tools/canary/catalog.go, tools/canary/catalog_test.go, tools/canary/config.go, tools/canary/config_test.go, tools/canary/dependencies_test.go]

### Approach
- Define the sole supported Nexus3 success entry with exact packaged digest, checked provenance identity and bounded environment configuration. Production catalog owns deployment ProfileSpec using the public catalog helper and existing binding IDs; never import test fixture helpers or generators.
- Bound-read exact bytes, DecodeCaseProtoJSON, compare digest/source provenance, Prepare and ValidateDriver for every selected entry before returning an immutable admitted selection. Unknown/duplicate names, malformed/stale artifact, unsupported Case/limits and incompatible namespace/queue/endpoint reject as a whole.
- Define serializable immutable measurement identity envelope consumed by sink/Workflow: installation/check/artifact and binding/Profile/catalog identity, orchestration workflow/run identity, later Testpilot Run identity; bounded provenance reference/summary. No credentials, opaque authority or runtime objects.
- Validate explicit config limits and callback/transport settings independently of Case semantics; one check and concurrency1 default, binding snapshots immutable. Empty selection is valid removal intent. Add production import-closure test excluding tests, generator and private Testpilot execution/verification packages.

### Investigation targets
**Required:**
- tests/testcore/testpilot/async_nexus_fixture.go:10,21 — policy/reference only.
- common/testing/testpilot/temporal/catalog.go:12 — reusable public catalog.
- common/testing/testpilot/profile.go:113,132 — immutable environment fingerprint.
- common/testing/testpilot/case.go:11 — strict decode.

### Quick commands
`mise exec -- go test -count=1 -tags test_dep ./tools/canary ./common/testing/testpilot/temporal`
Create TestCanaryCatalogSelectionAdmission and TestCanaryProductionDependencies in the new package and run explicitly after creation.

### Execution constraints
Read native fn70 R1–R10 and repository guides. Preserve comments and unrelated dirty source. Do not stage, commit, or push; the user owns commits. Fn77 Producer edits finish first for source serialization only: re-anchor exact delivered source/artifact before fn70 edits, without introducing a semantic prerequisite. No fn79 operation cancellation, replacement Driver/evaluator, runtime Lean invocation, general recovery/lease framework, production deployment/config mutation, or fn29 machinery. Proposed new file owners may reuse an established equivalent; record actual paths. Run Lean jobs serially. Baseline existing focused tests before edits; new named tests apply after creation and must be wired into the actual package/module roots. No silent skips, unmatched test regexes, fixture expectation weakening, or inferred passing gates.

## Acceptance
- [ ] One supported pinned success artifact is admitted without runtime Lean, copied assertions or private/test imports.
- [ ] Full selection negatives reject before schedule mutation or target execution; empty selection remains valid.
- [ ] Both supported binding configurations preserve exact source/Contract/provenance bytes and have distinct binding identities.
- [ ] Immutable bounded identity envelope and concurrency/limit configuration are covered by focused tests.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
