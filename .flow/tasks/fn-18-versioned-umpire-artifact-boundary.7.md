---
satisfies: [R1, R5, R8]
---
# fn-18-versioned-umpire-artifact-boundary.7 Prove cross-language Artifact identities and closure

## Description
Prove exact Lean/Go agreement for every retained family before complete-set admission depends on it.


**Size:** M
**Files:** `model/Umpire/Artifact/Tests/Goldens.lean`, `tools/umpire/artifact/golden_test.go`, and retained fixtures
**Touches:** [model/Umpire/Artifact/Tests/Goldens.lean, tools/umpire/artifact/golden_test.go, tools/umpire/artifact/testdata/**]

### Approach
- Emit authoritative canonical fixtures from Lean and recompute every Artifact Checksum independently in Go.
- Pin deterministic pretty bytes, provenance checksums, exact domain tags/preimages, and every
  family format/field sequence from the parent normative tables.
- Independently compare every Evidence/Result nested projection and the ExperimentRun
  receipt-fact link, not only top-level checksums.
- Mutate each Definition ID, Behavior Fingerprint, Artifact Checksum, Limit, Known Gap, binding, and terminal-LF relation one at a time.
- Keep coverage checkpoints, replay bundles, and generic receipt envelopes outside the fixture set.

### Investigation targets
**Required:** tasks `.3`–`.6` and fn-37's golden/mutation pattern.

## Acceptance
- [ ] Every retained family has one exact canonical positive fixture and focused mutation coverage.
- [ ] Every compact/alternate-whitespace form rejects and no fixture test substitutes semantic JSON
  equality for exact bytes.
- [ ] Cross-domain fingerprint/checksum substitution and stale relationship mutations reject.
- [ ] No deferred Artifact family appears in the production manifest.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestCrossLanguageGoldens`

## Done summary
Proved exact Lean/Go identity and closure for the six retained v2 Artifact families. The new golden suites pin byte-for-byte 2-space pretty JSON with exactly one terminal LF, top-level field order, Behavior Fingerprints, Artifact and provenance checksums, full chain closure, every Evidence/Result nested projection, and the ExperimentRun receipt-to-RawEvidence fact relationship. Compact and alternate-whitespace forms, cross-domain substitutions, stale relationships, and focused identity/checksum mutations reject without adding deferred Artifact families or runtime machinery.

TDD started from the inherited green Quick baseline, then failed because the Go mirrors for the authoritative Lean Evidence and Result fixtures were absent. Adding exact-byte mirrors made the six-family cross-language test green. The configured Codex review found one shallow-copy mutation-isolation defect; each mutation now reloads fresh golden documents, all scoped and aggregate gates pass, and the final-head re-review reached SHIP with no remaining findings.

stage: impl-review - ran [2026-08-29T08:17:06Z..2026-08-29T08:19:41Z] (model: gpt-5.6-sol)
## Evidence
- Commits: f67efe2caa8fb993fc2f0ecfd018f850d9ff8f54, f81b16b98ccc6e89358787e671522275ed331c0d
- Tests: baseline: green (mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestCrossLanguageGoldens; no matching tests before task implementation), TDD RED: mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestCrossLanguageGoldens (missing Go Evidence/Result fixture mirrors), mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestCrossLanguageGoldens, mise exec -- go test -count=1 ./tools/umpire/artifact/..., cd model && mise exec -- lake build Umpire.Artifact.Tests.Goldens, cd model && mise exec -- lake build UmpireTests, make lint-model, make umpire-check-regression, mise exec -- go vet ./tools/umpire/artifact/..., ./.bin/golangci-lint-v2.13.1 run --timeout 10m --new-from-rev=281a7102b17877d633a7db87a1b466b761964cc7 --config=.github/.golangci.yml ./tools/umpire/artifact/..., mise exec -- go test -race -count=1 ./tools/umpire/artifact/..., gofmt -d tools/umpire/artifact/golden_test.go, git diff --check, git diff --no-index -- model/Umpire/Artifact/Tests/Fixtures/EvidenceV2.json tools/umpire/artifact/testdata/evidence-v2.json, git diff --no-index -- model/Umpire/Artifact/Tests/Fixtures/ResultV2.json tools/umpire/artifact/testdata/result-v2.json, impl-review: SHIP at f81b16b98ccc6e89358787e671522275ed331c0d (codex:gpt-5.6-sol:high; /tmp/impl-review-receipt-fn-18-versioned-umpire-artifact-boundary.7.json), GATE_RECEIPT_NOT_WRITTEN:unittest - protected inherited config/development.yaml and schema/elasticsearch/visibility/index_template_v7.json false-symlink status entries kept the worktree dirty
- PRs: