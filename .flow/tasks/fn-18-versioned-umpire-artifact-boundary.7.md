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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
