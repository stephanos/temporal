---
satisfies: [R3, R4, R5, R6, R7, R8]
---
# fn-28-authorized-remote-staging-black-box.8 Compose the remote qualification controller and closed run mode

## Description
Implement R3-R8 behind one production-fixed staging controller and the qualification binary's closed run mode.

**Size:** M
**Files:** `tools/umpire/staging/**`, `tools/umpire/qualification/**`, `tools/umpire/cmd/umpire-qualify-remote-staging/**`, `model/lakefile.toml`
**Touches:** [tools/umpire/staging/**, tools/umpire/qualification/**, tools/umpire/cmd/umpire-qualify-remote-staging/**, model/lakefile.toml]

### Approach
- Compose ordered input/pilot/profile/workflow-context admission, protected authority and target preflight, lease, runtime execution, cleanup/reconciliation, postflight target verification, final Run/RawEvidence/provenance closure, conformance, offline qualification, v4 construction, and exactly one publication behind a narrow production API.
- Keep package-private injection at each operational seam; production fixes authority source, profile/checker siblings, program, limits, action, statuses, and publisher.
- Implement the exact `run` arguments and canonical secret-free summary/error/status 0/1/2 contract, including dispatch/cleanup/publication booleans and reporting-after-publication recovery.
- Expose narrow recovery-record and progress-sink interfaces for Task `.9`; run mode records state after lease acquisition but cannot select or weaken either production path.
- Preserve every constructible failed/incomplete run and all independent status dimensions; pre-dispatch tooling failures publish nothing, post-dispatch non-success publishes when structurally possible, and no path redispatches or republishes automatically.
- Register only required sibling executables in the primary Lake package; do not add a model-local Make target.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.5.md` — deep orchestration and stage/status preservation
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — runtime API and cleanup dominance
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — conformance API and tooling errors
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — pilot/policy/receipt command contract
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.10.md` — sole publisher behavior

### Key context
Cleanup/postflight facts are inputs to final evidence, conformance, provenance, and qualification; none may be finalized before those facts close. The remote adapter owns authority/public Temporal handles, while the controller owns stage order and filesystem publication.

### Acceptance
- [ ] Every malformed/preflight input performs no remote or publication IO.
- [ ] One valid injected run closes cleanup/postflight before conformance and publishes exactly one admitted v4 set.
- [ ] Every failure/cancel/non-success/cleanup/reporting/publication row preserves exact facts and status without redispatch.
- [ ] Run mode accepts only set, pilot-evidence, output-root, and run-id and emits no secret or arbitrary remote error.

## Acceptance
- [ ] R3-R8 controller, run mode, sibling handshake, stage ordering, status, and publication contracts are complete.
- [ ] Independent stage fakes and command integration tests pass.
- [ ] Existing orchestration comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
