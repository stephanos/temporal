# Versioned Umpire Artifact boundary

> HTML render lens (local): open `.flow/artifacts/fn-18-versioned-umpire-artifact-boundary/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 v2 baseline

Fn-37 owns the only supported DrivePlan and ExperimentSpec formats: `umpire-drive-plan/v2` and
`umpire-experiment/v2`. Fn-18 begins from those exact canonical bytes, Definition IDs, Behavior
Fingerprints, Artifact Checksums, Limits, and Known Gaps. It adds no reader, migration, alias, or
fallback for any earlier prototype format. Unsupported majors fail before field-level validation.

## Overview

Create the minimal strict persistence boundary required by the prototype. Lean remains authoritative
for Artifact structure and canonical bytes. A small Go admission package accepts the fn-37 v2
baseline and the additional `RuntimeConfiguration`, `ExperimentRun`, bounded `RawEvidence`,
interpreted `Evidence`, and `Result` families needed by local Execution and Run Evaluation.

One admitted Artifact set closes every cross-document reference, and one immutable publication/load
path exposes either a complete consistent set or nothing. The boundary is transport-only: it does
not execute Temporal, interpret raw Evidence, perform Observation Evaluation, apply an
Implementation Link, evaluate a Property, or produce a Claim Assessment.

## Goal & Context
<!-- scope: business -->

The same environment-independent v2 Test Plan can be admitted byte-for-byte, bound to one runtime
configuration, executed into one Run, interpreted into Evidence and a Result, and loaded atomically
without any component inventing model meaning or silently repairing stale input.

## Architecture & Data Models
<!-- scope: technical -->

```text
Lean-owned canonical v2 Artifacts
  ── strict Go admission ──▶ typed inert values
  ── exact closure ────────▶ admitted Artifact set
  ── immutable publish ────▶ atomic load

RawEvidence ── Observation Evaluation ──▶ Evidence + Evidence Links
Evidence + Model Trace + Properties ────▶ Run Evaluation Result
```

The additional persisted families are deliberately small:

- `RuntimeConfiguration` binds the v2 Test Plan to participant, protocol, Observation, and phase
  Limit identities without carrying endpoints, credentials, namespaces, or authority material.
- `ExperimentRun` records bounded attempts, outcomes, Execution Receipts, source closure, cleanup,
  and Known Gaps without deciding what the Run proves.
- `RawEvidence` preserves typed source-local and causal facts without interpretation.
- `Evidence` is the persisted output of Observation Evaluation. An accepted result contains one
  complete Evidence-backed Model Trace and an Evidence Link for every established Model Fact.
- `Result` keeps operational status, Observation Evaluation, Implementation Link, Property status,
  Limits, Known Gaps, and cleanup separate. Claim Assessment remains downstream.

Every Artifact carries its own Artifact Checksum. Model references carry Definition IDs and
Behavior Fingerprints. The two types are never interchangeable.

## API Contracts
<!-- scope: technical -->

- Decode operations require one expected family and exact canonical bytes with one terminal LF.
- The v2 DrivePlan and ExperimentSpec codecs reuse fn-37's field order, canonical escaping, number
  spelling, and independently recomputed Artifact Checksum formulas.
- Unknown keys, duplicate or case-colliding keys, malformed fingerprints/checksums, wrong family,
  unsupported version, noncanonical order/whitespace/escaping/numbers, checksum drift, and missing
  or extra LF reject without normalization.
- Recursive input has explicit byte, token, depth, collection, string, and payload Limits with N+1
  failure evidence before unbounded allocation.
- Artifact-set admission requires exact member uniqueness, paths, checksums, version agreement, and
  reference closure. It never fetches or synthesizes a missing member.
- Publication validates the complete candidate before replacement, uses safe contained regular
  files and one lock, and never exposes a partial or mixed set.
- Go validates transport and closure only. Lean-owned Observation Evaluation, Implementation Link,
  Run Evaluation, and Property modules remain the sole semantic authorities.

## Edge Cases & Constraints
<!-- scope: technical -->

- Earlier prototype formats always return the stable unsupported-format classification before
  field validation. There is no migration registry in the prototype.
- A Behavior Fingerprint mismatch is distinct from an Artifact Checksum mismatch.
- A Limit belongs to one stage and cannot be reused to bound another stage.
- Known Gaps remain the exact closed records from fn-37; a free-form gap or conflicting duplicate
  rejects.
- Missing, ambiguous, conflicting, unsupported, redacted, or causally unrelated Evidence cannot
  establish an accepted Model Fact.
- A Run that operationally succeeds may still have unknown or conflicting Observation Evaluation,
  a failed Implementation Link, a violated Property, incomplete cleanup, or unresolved Known Gaps.
- Failed admission or publication leaves every previously admitted complete set unchanged.

## Quick commands

```bash
go test -count=1 ./tools/umpire/artifact/...
go test -count=1 ./tools/common/artifactio/...
cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs
make umpire-check-artifact ARTIFACT=model/Temporal/Feature/Nexus/testdata/nexus-caller-closure-experiment-spec.json FAMILY=umpire-experiment/v2
make umpire-check-artifact-set SET=tools/umpire/artifact/testdata/valid-run-evaluation-set
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** One bounded strict JSON kernel rejects malformed, noncanonical, oversized, duplicate,
  unknown-field, wrong-family, unsupported-version, and checksum-invalid bytes before returning a
  value. [paraphrase]
- **R2:** Fn-37's v2 DrivePlan and ExperimentSpec are the sole baseline and round-trip byte-for-byte
  with independently recomputed Artifact Checksums and complete Definition ID, Behavior
  Fingerprint, Limit, Known Gap, occurrence, Property, and provenance validation. [user]
- **R3:** RuntimeConfiguration, ExperimentRun, and RawEvidence have minimal exact schemas and
  Artifact bindings while preserving the separation between configuration, Execution, and raw
  capture. [paraphrase]
- **R4:** Evidence persists Observation Evaluation and Evidence Links without remapping raw facts;
  Result keeps operational, Observation Evaluation, Implementation Link, Property, Known Gap, and
  cleanup statuses independent and performs no Claim Assessment. [user]
- **R5:** Cross-language goldens prove exact canonical bytes, Behavior Fingerprints, Artifact
  Checksums, and closed references for every retained family. [paraphrase]
- **R6:** Unsupported majors and every pre-v2 current Artifact reject without a reader, migration,
  alias, fallback, or repaired output. [user]
- **R7:** Complete-set admission and immutable atomic publication reject unsafe paths, partial or
  mixed sets, stale references, and interrupted installation while preserving prior complete
  state. [paraphrase]
- **R8:** Public facades, root commands, tests, and architecture docs expose only the transport
  boundary and introduce no runtime, semantic, replay, coverage, verification-receipt, release, or
  Umpire3 machinery. [user]

## Early proof point

Task `.3` must prove that the existing fn-37 v2 caller-closure fixture is accepted and re-encoded
byte-for-byte while representative earlier-format, legacy-key, noncanonical, and checksum mutations
reject. Later families do not proceed until Lean and Go agree on those exact bytes.

## Boundaries
<!-- scope: business -->

- No pre-v2 reader, migration, compatibility alias, deprecated forwarding module, or dual-format
  output.
- No coverage report/checkpoint, replay bundle, verification receipt family, generic receipt
  envelope, Artifact-management platform, or management CLI.
- No live Execution, Evidence collection, Observation Evaluation, Implementation Link, Property
  evaluation, replay, promotion, or Claim Assessment.
- No environment credentials, endpoints, namespaces, or authority material in portable Artifacts.
- No generated drift framework, CI workflow, model-local Makefile, or Umpire3 path.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

The prototype needs one trustworthy handoff from checked model intent to bounded Execution and Run
Evaluation. Reusing fn-37's v2 baseline avoids carrying a compatibility problem that has no external
consumer.

### Implementation Tradeoffs
<!-- scope: technical -->

A deep admission module centralizes canonical byte, Limit, checksum, and closure enforcement while
keeping each Artifact family inert. One immutable complete-set path is sufficient for the prototype;
general migration and Artifact-management machinery waits for a real post-v2 format change.

## References

- `.plans/UMPIRE4_SPEC.md` — authoritative Umpire4 language and separation rules.
- `.plans/UMPIRE4_ORDER.md` — reduced prototype scope and v2 baseline.
- `.flow/specs/fn-37-hard-cut-umpire-vocabulary-and-current.md` — current Artifact contract.
- `.flow/specs/fn-4-umpire-observation-and-semantic-verdicts.md` — Observation Evaluation authority.
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — future Run Evaluation consumer.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Strict bounded admission | `.2`, `.3`, `.4`–`.9` | — |
| R2 | V2 baseline | `.1`, `.3` | — |
| R3 | Runtime, Run, and RawEvidence | `.4`, `.5` | — |
| R4 | Evidence and Result | `.6`, `.8` | — |
| R5 | Cross-language identity goldens | `.3`, `.7` | — |
| R6 | Unsupported-format hard cut | `.3`, `.9` | — |
| R7 | Set closure and atomic publication | `.8`, `.10` | — |
| R8 | Integration and boundaries | `.1`–`.11` | — |
