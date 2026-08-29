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
Admitted only the exact two-member executable, four-member execution, and six-member evaluation Artifact v2 closures at their prescribed ordered safe paths. The new Lean and Go deep modules decode members independently, validate canonical transport and complete retained relationships, reject partial, extra, duplicate, mixed, stale, unresolved, unsafe, path-inconsistent, or non-canonical input atomically, and remain inert, bounded, fetch-free, and publication-independent.

Lean and Go independently reproduce the exact deterministic 2-space pretty manifest bytes with one terminal LF, member rows, set identity, set checksum, and raw manifest SHA-256. TDD began from the inherited green Quick baseline, then drove the missing set admission and manifest APIs plus unresolved-link rejection. The configured Codex review found one Lean transport-parity gap for checksum-preserving invalid planning collections; focused DrivePlan and ExperimentSpec validators and regression proofs closed it, all scoped and aggregate gates pass, and the same-receipt re-review reached SHIP with no remaining findings.

stage: impl-review - ran [2026-08-29T08:54:34Z..2026-08-29T09:04:34Z] (model: gpt-5.6-sol)
## Evidence
- Commits: 240a50cf66b58ce285a295cae70c725b37e86bee, f70b3716f9bc285767b0070b8387f33bc3861ab0
- Tests: baseline: green (mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestArtifactSet; no matching tests before task implementation), TDD RED: mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestArtifactSet (missing AdmitSet and SetMember API), TDD RED: mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestArtifactSet (missing AdmitSetManifest API), TDD RED: exact Go and Lean unresolved Result implementation-link source target rejected after resealing the affected outcome and checksum, review RED: cd model && mise exec -- lake build Umpire.Artifact.Tests.Set (checksum-preserving duplicate observationRequirementDefinitionIds accepted before transport validation fix), mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestArtifactSet, mise exec -- go test -count=1 ./tools/umpire/artifact/..., cd model && mise exec -- lake build Umpire.Artifact.Tests.Set, cd model && mise exec -- lake build UmpireTests, make lint-model, make umpire-check-regression, make umpire-check-legacy-vocabulary, mise exec -- go vet ./tools/umpire/artifact/..., ./.bin/golangci-lint-v2.13.1 run --timeout 10m --new-from-rev=b3d9c949f00169de3c99b951f5899b0d5459f4ce --config=.github/.golangci.yml ./tools/umpire/artifact/... (0 issues), mise exec -- go test -race -count=1 ./tools/umpire/artifact/..., mise exec -- go test -count=1 ./tools/umpire/artifact/... -run '^$' -fuzz '^FuzzArtifactSetAdmission$' -fuzztime=5s, gofmt -d tools/umpire/artifact/set.go tools/umpire/artifact/set_test.go, git diff --check, impl-review: SHIP at f70b3716f9bc285767b0070b8387f33bc3861ab0 (codex:gpt-5.6-sol:high; /tmp/impl-review-receipt-fn-18-versioned-umpire-artifact-boundary.8.json), GATE_RECEIPT_NOT_WRITTEN:unittest - protected inherited config/development.yaml and schema/elasticsearch/visibility/index_template_v7.json false-symlink status entries kept the worktree dirty
- PRs: