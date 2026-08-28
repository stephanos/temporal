---
satisfies: [R3, R4, R5, R7, R8]
---
# fn-18-versioned-umpire-artifact-boundary.8 Admit complete Artifact sets with exact closure

## Description
Admit one exact closed set of the v2 Test Plan and retained execution/evaluation Artifacts.


**Size:** M
**Files:** `model/Umpire/Artifact/Set.lean`, tests, and `tools/umpire/artifact/set.go`
**Touches:** [model/Umpire/Artifact/Set.lean, model/Umpire/Artifact/Tests/Set.lean, tools/umpire/artifact/set.go, tools/umpire/artifact/set_test.go]

### Approach
- Admit only the exact two-member executable, four-member execution, or six-member evaluation
  closures and their exact ordered safe paths from the parent contract; DrivePlan stays nested.
- Decode members individually, then validate unique safe paths, exact checksums, version agreement, bindings, and complete relationship closure.
- Resolve model references only through the appropriate retained Artifact fields; never invent pseudo-Artifact documents.
- Reject partial, extra, duplicate, mixed, stale, unresolved, or path-inconsistent sets atomically.

### Investigation targets
**Required:** tasks `.3`–`.7` and the parent complete-set contract.

## Acceptance
- [ ] Every retained relationship closes exactly once and no deferred family is required.
- [ ] Missing, extra, duplicate, mixed-version, stale, unsafe-path, and unresolved-reference sets reject without partial output.
- [ ] Admission remains inert, bounded, fetch-free, and independent of publication.
- [ ] The exact deterministic pretty manifest member rows, set identity, set checksum, and raw
  manifest SHA-256 are independently reproduced; every other member count/family/path/order rejects.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestArtifactSet`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
