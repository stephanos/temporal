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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
