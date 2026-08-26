---
satisfies: [R6, R7]
---
# fn-30-release-evidence-graph-and-manual.5 Implement two-role authorization, veto, and revocation

## Description
Implement the append-only role-decision boundary, protected role-decision signer mode, ReleaseAuthorizationSet publisher, and closed verifier/controller CLI modes for R6 and R7. Authorization consumes an exact still-current qualified set and never gains deployment capabilities.

**Size:** M
**Files:** authorization/controller files and tests under `tools/umpire/release/`, verifier command under `tools/umpire/cmd/umpire-release/`, role-decision mode under `tools/umpire/cmd/umpire-release-sign/`
**Touches:** [tools/umpire/release/authoriz*.go, tools/umpire/release/controller*.go, tools/umpire/release/publish*.go, tools/umpire/cmd/umpire-release/**, tools/umpire/cmd/umpire-release-sign/**]

### Approach
- Have the protected signer derive candidate, graph, policy, trust, omission, invocation, predecessor head, role/key, issued time, and bounded expiry from the current set plus fixed protected context; only approve/deny/revoke is selectable.
- Strictly admit signed decisions, enforcing distinct active role keys, exact bindings, predecessor heads, evaluation freshness, and signer-role separation. Gate refusal/timeout emits no record and can never mean deny.
- Require both owner roles for approval; allow either role to deny or revoke; make identical retries idempotent and crossed/stale/conflicting sequences fail closed.
- Keep qualify/authorize/deny/revoke behind one verifier/controller with typed terminal states and one final immutable publication point per mode; separate canonical status output from process exit mapping.

### Investigation targets
**Required** (read before coding):
- `tools/common/artifactio/set.go:65-103` — lock-scoped recovery and final install
- `tools/common/artifactio/artifact.go:10-40` — durable atomic artifact write
- `tools/umpire/cmd/umpire-gen-regression-projections/main.go` — focused Umpire command boundary pattern
- `tools/umpire/cmd/umpire-gen-regression-projections/main_test.go` — command-level test pattern

**Optional** (reference as needed):
- `tools/common/artifactio/set_test.go` — concurrent conflict and recovery fixtures

### Acceptance
- [ ] Two distinct current approvals create one bounded authorization; one explicit valid veto/revocation prevents or supersedes it without mutating history.
- [ ] Signer tests cover fixed-secret acquisition, derived role/bindings/head, issued/expiry caps, explicit decision modes, and refusal/timeout producing no record.
- [ ] Wrong-role, copied, expired, crossed, stale-head, conflicting, post-denial, and post-revocation decisions fail with canonical non-authorizing outcomes.
- [ ] All verifier CLI modes preserve status 0/1/2 semantics, publish at most once, and accept no target, credential, private key, policy, clock, repository, URL, executable, or deployment option.
- [ ] Capability and secret scans prove that neither signer, controller, nor command can deploy, promote, route, configure, roll back, or leak protected material.
## Acceptance
- [ ] The protected signer creates exact role-decision artifacts only after its role gate.
- [ ] The two-role approval and one-role veto/revocation state machine is immutable and race-safe.
- [ ] Closed verifier modes preserve canonical outcomes and never expose deployment authority.
- [ ] Authorization, signing, adversarial-ordering, command, and race tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
