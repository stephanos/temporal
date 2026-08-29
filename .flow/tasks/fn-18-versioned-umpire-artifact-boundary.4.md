---
satisfies: [R1, R3, R8]
---
# fn-18-versioned-umpire-artifact-boundary.4 Define RuntimeConfiguration and ExperimentRun transports

## Description
Define exact inert transports for one runtime configuration and one bounded Execution Run.


**Size:** M
**Files:** `model/Umpire/Artifact/Runtime.lean`, tests, and `tools/umpire/artifact/runtime.go`
**Touches:** [model/Umpire/Artifact/Runtime.lean, model/Umpire/Artifact/Tests/Runtime.lean, tools/umpire/artifact/runtime.go, tools/umpire/artifact/runtime_test.go]

### Approach
- Implement exactly the parent RuntimeConfiguration and ExperimentRun format strings, top-level
  field order, nested record order, ArtifactBinding/provenance/checksum rules, closed enums, and
  nullability; add no arbitrary map or inferred field.
- Bind both families to the exact v2 Test Plan through Artifact Checksum and Behavior Fingerprint references.
- Keep RuntimeConfiguration free of endpoints, credentials, namespaces, and authority material.
- Record Run attempts, phase outcomes, Execution Receipts, source closure, cleanup, Limits, and Known Gaps without evaluating the Run.
- Enforce the five fixed phases, terminal timestamps/codes, control/source/cleanup status matrices,
  positive attempt, canonical ordering, and cross-binding rules from the parent contract.
- Give every attempted control row exactly one `receiptFactDefinitionId`; task `.5` must resolve it
  to the sole matching RawEvidence control-receipt fact with identical occurrence/action/attempt/status.

### Investigation targets
**Required:** the parent retained-family schemas and fn-19's execution handoff.

## Acceptance
- [ ] Canonical Lean/Go values agree and reject unknown, stale, malformed, or cross-boundary-inconsistent data.
- [ ] Each Limit is scoped to one stage and Known Gaps use the exact closed fn-37 record.
- [ ] RuntimeConfiguration performs no authorization and ExperimentRun carries no Property or Claim Assessment.
- [ ] Exact format/order/null/status/checksum mutations reject one at a time, including every
  invalid phase/control/source/cleanup combination.
- [ ] Not-attempted control rows have no receipt/code, every attempted row has one receipt ID, and
  set closure rejects missing, duplicate, crossed, or mismatched receipt facts.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRuntime`

## Done summary
Defined the exact canonical v2 RuntimeConfiguration and ExperimentRun transports in Lean and Go, including deterministic pretty-byte checksums, Artifact bindings, fixed phase/control/source/cleanup matrices, operational-status precedence, bounded admission, provenance parity, and task-boundary closure checks. RawEvidence receipt resolution remains intentionally deferred to task .5; no semantic evaluation, authorization, endpoint, credential, namespace, Property, or Claim Assessment surface was added.

Baseline: green (`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRuntime`). Final exact Quick, full artifact Go, Lean Runtime/Codecs/UmpireTests, fixture/preimage parity, regression, model lint, vet, golangci-lint, race, gofmt, and diff checks passed. The unittest gate receipt was not writable because the protected inherited `config/development.yaml` false symlink stat keeps the worktree dirty; the task gate itself passed. Review-fix memory capture was attempted but repository memory is not initialized.

stage: impl-review - ran [2026-08-29T03:02:13Z..2026-08-29T05:17:57.929549Z] (model: gpt-5.6-sol)
## Evidence
- Commits: 4f5ad19c64a921999c6837787870e972e8e0da82, e74929561ce590be2a29994a8769fe904079e2f7, 93f4c8b2eed3ce8e709b5eda07eeb65dde02477e, 2589c47c2d51b451bff9df265e96ca1107f71d8a, 7be518b9585735981801dd70902f394f573c1cc2
- Tests: baseline: green (mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRuntime), mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRuntime, mise exec -- go test -count=1 ./tools/umpire/artifact/..., mise exec -- go test -count=1 ./tools/umpire/artifact -run 'Test(RuntimeConfigurationV2|ExperimentRunV2)CanonicalFixtureRoundTrip|TestRuntimeV2IndependentPrettyPreimageChecksums', cd model && mise exec -- lake build Umpire.Artifact.Tests.Runtime Umpire.Artifact.Tests.Codecs UmpireTests, make umpire-check-regression, make lint-model, mise exec -- go vet ./tools/umpire/artifact/..., ./.bin/golangci-lint-v2.13.1 run ./tools/umpire/artifact/..., mise exec -- go test -count=1 -race ./tools/umpire/artifact/..., gofmt -d tools/umpire/artifact/runtime.go tools/umpire/artifact/runtime_test.go tools/umpire/internal/artifactv2/runtime.go, git diff --check, GATE_RECEIPT_NOT_WRITTEN:unittest - protected inherited config/development.yaml false symlink stat kept worktree dirty
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)
