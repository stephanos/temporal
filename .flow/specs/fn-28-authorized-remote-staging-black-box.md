# Black-box staging execution

> HTML render lens (local): open `.flow/artifacts/fn-28-authorized-remote-staging-black-box/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 architecture reconciliation

Fn-28 is a portability proof, not a Claim Assessment platform. It consumes the byte-identical v2
`ExperimentSpec`, the existing runner, and the canonical Run Evaluation authority. The named
operational owner must already provide the fixed staging profile and harness, fail-closed authority,
target preflight, concurrency one, fixed Limits, isolation, cleanup verification, and postflight
target identity. If any control is unavailable, the run is blocked before remote mutation; Umpire
does not build a replacement control plane.

## Overview

Run the same normal caller-closure `ExperimentSpec` used locally and in CI against one controlled
nonproduction Temporal endpoint. Observe only public gRPC Evidence and participant-owned Execution
Receipts, perform cleanup before Run Evaluation, and retain an ordinary fn-18 run/evidence/result set.
Also provide a canary dry-run fixture that proves the same Artifact bytes and model bindings can be
bound to an owner-supplied canary profile without granting production Execution authority.

## Goal & Context
<!-- scope: business -->

Developers need one inspectable answer to whether the portable Test Plan and Run Evaluation contract
survive a public remote boundary. Operators remain responsible for the staging environment and its
existing controls. This slice supplies no release signal, environment Claim Assessment, or generalized
remote management machinery.

## Architecture & Data Models
<!-- scope: technical -->

```text
byte-identical v2 ExperimentSpec
  -> owner-supplied fixed staging binding + preflight
  -> one bounded public-gRPC Execution
  -> participant/public Evidence + verified cleanup/postflight
  -> existing Observation Evaluation / Implementation Link / Property path
  -> ordinary Run Evaluation Result

same Artifact + canary binding -> dry-run validation only -> no production authority
```

The staging adapter owns only the narrow binding to the existing harness. It does not add an
Evaluation Profile or Evaluation Receipt, does not evolve `ArtifactSet`, and does not introduce a
new persisted authority, lease, recovery, progress, provenance, or Claim Assessment schema. Target
coordinates and credentials remain owner-supplied runtime inputs and never enter portable Artifacts,
Behavior Fingerprints, Generated Views, logs, or retained Evidence.

Public Evidence is admitted through the existing fn-18/fn-20 boundaries. Environment-specific facts
may establish operational preflight, cleanup, and postflight outcomes, but cannot establish Feature
meaning or alter Observation Evaluation. The final Result keeps operational, Observation Evaluation,
Implementation Link, Property, cleanup, and tooling outcomes independent.

## API Contracts
<!-- scope: technical -->

- The production entry point accepts the admitted v2 input set and one compiled fixed staging binding;
  it exposes no endpoint, namespace, credential, participant, retry, checker, Property, or target
  selector.
- Before any mutation, the adapter asks the existing harness to validate authority, target identity,
  isolation, concurrency one, and sufficient remaining wall/API Limits. A missing or failed control
  blocks the run and performs no remote mutation.
- The adapter dispatches one bounded caller-closure operation, records public history plus
  participant-owned Execution Receipts, and never reads server-internal state.
- Cleanup and postflight identity verification complete before the evidence set reaches Run
  Evaluation. Uncertain cleanup remains an honest non-success.
- The canary dry-run validates only Artifact bytes, format version, Artifact Checksum, Behavior
  Fingerprints, required public Evidence capabilities, and the absence of production Execution
  authority. It sends no production RPC.
- Publication reuses the ordinary fn-18 complete-set path. No new member family or set version is
  created by this spec.

## Edge Cases & Constraints
<!-- scope: technical -->

- Authority, target, isolation, cleanup, or postflight uncertainty blocks success and cannot be
  converted to a Property result or Claim Assessment.
- Concurrency is exactly one because the owner-supplied harness enforces it; Umpire adds no lease or
  scheduler.
- Retry and ambiguity behavior comes from the existing harness. The adapter never implements a
  recovery controller, redispatch loop, or crash-resume protocol.
- Public Evidence at the exact Limit is admitted; Limit plus one fails at the responsible boundary.
- The same semantic facts may yield the same Run Evaluation outcome while environment/run identities
  remain distinct.
- Existing comments are preserved in reused source and documentation.

## Quick commands

```bash
cd model && mise exec -- lake build TemporalModelTests
mise exec -- go test ./tools/umpire/staging/...
mise exec -- go test ./tools/umpire/cmd/umpire-run-fixed-staging/...
mise exec -- make umpire-check-regression
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** One compiled fixed staging binding names the owner-supplied harness and required authority,
  target, isolation, concurrency-one, Limit, cleanup, and postflight controls without adding portable
  environment or Claim Assessment vocabulary.
- **R2:** The adapter consumes the byte-identical local v2 `ExperimentSpec` and preserves its format,
  Artifact Checksum, Behavior Fingerprints, and all admitted source bytes.
- **R3:** One bounded public-gRPC Execution records only public Evidence and participant-owned
  Execution Receipts, with no server-internal Evidence or semantic interpretation in Go.
- **R4:** Cleanup and postflight complete before the existing Run Evaluation path; operational,
  Observation Evaluation, Implementation Link, Property, cleanup, and tooling outcomes remain
  independent.
- **R5:** A canary dry-run fixture proves the same Artifact/model bindings and required Evidence
  capabilities without credentials, production RPC, or production Execution authority.
- **R6:** Negative fixtures prove missing owner controls, authority/target drift, concurrency or Limit
  breach, cleanup uncertainty, secret leakage, and attempted target selection fail closed. No
  protected workflow, lease system, recovery controller, Evaluation Receipt/Profile, new ArtifactSet,
  or Claim Assessment platform is introduced.

## Early proof point

The first integration test must bind the existing harness to the exact v2 Artifact and complete
preflight without broadening the command surface. If the owner-supplied controls are unavailable, the
spec is blocked rather than implementing substitutes in Umpire.

## Boundaries
<!-- scope: business -->

- No general remote target selector, protected workflow, lease system, recovery controller, or
  progress/checkpoint service.
- No Evaluation Profile, Evaluation Receipt, Claim Assessment, release eligibility, canary Execution,
  deployment, routing, or production mutation.
- No server-internal Evidence, staging-specific semantic mapper, second Run Evaluation authority, or
  new Artifact family/set version.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

A narrow black-box run answers the portability question without turning a prototype test tool into an
environment control plane.

### Implementation Tradeoffs
<!-- scope: technical -->

Requiring an existing owner-supplied harness can block delivery, but it keeps authority, isolation,
and cleanup with the operational owner and makes Umpire's interface substantially smaller.

## References

- `.plans/UMPIRE4_ORDER.md` — retained fn-28 portability scope and deferred control-plane work.
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — v2 Artifact admission/publication.
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — shared runner lifecycle.
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — canonical Run Evaluation authority.
- `.flow/specs/fn-27-hermetic-ci-execution-and-qualification.md` — byte-identical CI portability proof.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Fixed owner-supplied binding and controls | `.1`, `.2` | — |
| R2 | Byte-identical v2 Artifact admission | `.1`, `.3` | — |
| R3 | Public-gRPC Execution and Evidence | `.3`, `.4` | — |
| R4 | Cleanup then shared Run Evaluation | `.5`, `.6` | — |
| R5 | Canary dry-run binding proof | `.7` | — |
| R6 | Fail-closed integration and boundaries | `.8`–`.11` | — |
