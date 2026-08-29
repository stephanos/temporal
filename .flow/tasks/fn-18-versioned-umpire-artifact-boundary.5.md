---
satisfies: [R1, R3, R8]
---
# fn-18-versioned-umpire-artifact-boundary.5 Define the bounded RawEvidence transport

## Description
Persist bounded typed raw records without interpreting them as Model Facts.


**Size:** M
**Files:** `model/Umpire/Artifact/Evidence.lean`, tests, and `tools/umpire/artifact/evidence.go`
**Touches:** [model/Umpire/Artifact/Evidence.lean, model/Umpire/Artifact/Tests/Evidence.lean, tools/umpire/artifact/evidence.go, tools/umpire/artifact/evidence_test.go]

### Approach
- Implement exactly the parent `umpire-raw-evidence/v2` field/nested-record order,
  ArtifactBinding/provenance/checksum rules, three capture/source statuses, and closed fact/field
  value grammar.
- Preserve source identity, source-local order, causal links, typed fields, dispositions, closure, and exact Artifact bindings.
- Enforce 64 sources, 4,096 facts, 128 fields/fact, 1 MiB/fact, and 16 MiB aggregate decoded
  payload before allocation/append and reject ordinal gaps, forward/dangling references, or cycles.
- Keep Observation Evaluation, Model Trace construction, Property evaluation, and Claim Assessment absent.

### Investigation targets
**Required:** the parent RawEvidence contract and fn-4 Evidence input boundary.

## Acceptance
- [ ] Canonical cross-language fixtures preserve exact raw facts and bindings.
- [ ] Malformed types, ordering, closure, causality, disposition, Limit, and checksum mutations reject.
- [ ] RawEvidence cannot encode an accepted Model Fact or Property result.
- [ ] N/N+1 fixtures cover every evidence ceiling without truncation, and field disposition/value
  combinations reject prohibited raw values.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRawEvidence`

## Done summary
Defined the exact canonical `umpire-raw-evidence/v2` transport in Lean and Go, including byte-identical pretty fixtures/checksums, closed typed field grammar, bounded pre-allocation admission, source-local causality, exact Artifact bindings, and one-to-one ExperimentRun control receipts. The implementation remains transport-only and reuses the existing runtime binding, provenance, and checksum modules without adding semantic evaluation or recovery machinery.

Baseline: green (`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRawEvidence`, before task tests existed). Final exact Quick, full artifact/artifactio/internal Go, Lean Evidence/Codecs/UmpireTests, N/N+1 ceiling and closure matrices, regression, model lint, vet, canonical golangci-lint, race, fuzz, formatting, diff, and fixture-parity checks passed. The first regression run exposed transient cold-build inspector stderr; a component diagnostic returned status 0 with empty stderr and the captured final regression gate passed. The root `umpire-check-artifact` and `umpire-check-artifact-set` targets remain inherited deferred fn18.11 work per conductor direction, and the unittest gate receipts were not writable because the protected inherited `config/development.yaml` false-symlink stat keeps the worktree dirty. Codex review's one stable-error-precedence finding was fixed and re-reviewed to SHIP; review-fix memory capture was attempted but repository memory is not initialized.

stage: impl-review - ran [2026-08-29T06:02:28Z..2026-08-29T06:10:58Z] (model: gpt-5.6-sol)
## Evidence
- Commits: c6943d22e551f7536d6a5e945aef3b1fbf90750a, 3b440ba11406bddf2c48e849f860ee889cbc06d4
- Tests: baseline: green (mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRawEvidence), mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRawEvidence, mise exec -- go test -count=1 ./tools/umpire/artifact/... -run 'TestRawEvidenceV2(WrongContainersAreMalformedValues|EvidenceCeilings)', go test -count=1 ./tools/umpire/artifact/..., go test -count=1 ./tools/common/artifactio/..., mise exec -- go test -count=1 ./tools/umpire/internal/artifactv2/..., cd model && mise exec -- lake build Umpire.Artifact.Tests.Evidence, cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs, cd model && mise exec -- lake build UmpireTests, make umpire-check-regression, make lint-model, mise exec -- go vet ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/..., ./.bin/golangci-lint-v2.13.1 run ./tools/umpire/artifact/..., mise exec -- go test -count=1 -race ./tools/umpire/artifact/..., mise exec -- go test -count=1 ./tools/umpire/artifact/... -run '^$' -fuzz '^FuzzStrictJSONNoPanicOrPermissiveSuccess$' -fuzztime=5s, gofmt -d tools/umpire/artifact/evidence.go tools/umpire/artifact/evidence_test.go tools/umpire/internal/artifactv2/evidence.go, git diff --check, cmp -s model/Umpire/Artifact/Tests/Fixtures/RawEvidenceV2.json tools/umpire/artifact/testdata/raw-evidence-v2.json, INHERITED_DEFERRED: make umpire-check-artifact ARTIFACT=model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json FAMILY=umpire-experiment/v2 - target owned by fn18.11, INHERITED_DEFERRED: make umpire-check-artifact-set SET=tools/umpire/artifact/testdata/valid-run-evaluation-set - target owned by fn18.11, GATE_RECEIPT_NOT_WRITTEN:unittest - protected inherited config/development.yaml false symlink stat kept worktree dirty
- PRs: