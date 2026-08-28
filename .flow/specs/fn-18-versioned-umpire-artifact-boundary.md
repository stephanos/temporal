# Versioned Umpire Artifact boundary

> HTML render lens (local): open `.flow/artifacts/fn-18-versioned-umpire-artifact-boundary/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 v2 baseline

Fn-37 owns the only supported DrivePlan and ExperimentSpec families: `umpire-drive-plan/v2` and
`umpire-experiment/v2`. Fn-18 retains those schemas, Definition IDs, Behavior Fingerprints, Limits,
and Known Gaps while establishing deterministic pretty JSON as their one canonical byte
representation. Canonical objects preserve fixed field order, escaping, and number spelling, use
two-space indentation with no trailing spaces, and end in exactly one LF. Compact JSON and every
other whitespace form are noncanonical input, not compatibility formats. Fn-18 adds no reader,
migration, alias, or fallback for any earlier prototype format. Unsupported majors fail before
field-level validation.

This is an explicit pre-release baseline correction to the canonical-form and checksum-preimage
sentences in fn-37 and `.plans/UMPIRE4_ORDER.md`; it does not preserve fn-37's compact bytes or the
checksums derived from them. No supported external or immutable published v2 set predates this
prototype correction, so the cut is completed in place rather than inventing a v3 compatibility
family. All checked-in v2 fixtures, Generated Views, and consumers move atomically to the pretty
bytes and newly derived checksums. Downstream fn-19 must consume `umpire-experiment/v2`; its stale
`umpire-experiment/v3` sentence is a planning defect that task `.11` synchronizes before fn-19 may
start.

## Overview

Create the minimal strict persistence boundary required by the prototype. Lean remains authoritative
for Artifact structure and deterministic pretty canonical bytes. A small Go admission package emits
and accepts those same exact v2 bytes plus the additional `RuntimeConfiguration`, `ExperimentRun`,
bounded `RawEvidence`, interpreted `Evidence`, and `Result` families needed by local Execution and
Run Evaluation.

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

- Decode operations require one expected family and exact deterministic pretty canonical bytes with
  two-space indentation, no trailing spaces, and one terminal LF.
- The v2 DrivePlan and ExperimentSpec codecs share one Lean/Go field order, escaping, number
  spelling, indentation, and terminal-LF contract. Each Artifact Checksum is derived from the exact
  domain-separated pretty checksum preimage for that document: the same canonical object with only
  its own `artifactChecksum` field omitted and one terminal LF. The outer ExperimentSpec preimage
  contains its already-sealed DrivePlan.
- Unknown keys, duplicate or case-colliding keys, malformed fingerprints/checksums, wrong family,
  unsupported version, compact JSON, alternate indentation or whitespace, noncanonical
  order/escaping/numbers, checksum drift, and missing or extra LF reject without normalization.
- Recursive input has explicit byte, token, depth, collection, string, and payload Limits with N+1
  failure evidence before unbounded allocation.
- Artifact-set admission requires exact member uniqueness, paths, checksums, version agreement, and
  reference closure. It never fetches or synthesizes a missing member.
- Publication validates the complete candidate before replacement, uses safe contained regular
  files and one lock, and never exposes a partial or mixed set.
- Go validates transport and closure only. Lean-owned Observation Evaluation, Implementation Link,
  Run Evaluation, and Property modules remain the sole semantic authorities.

### Canonical checksum and binding rules

Every checksum is `sha256:` plus 64 lowercase hexadecimal digits. Artifact checksums hash the UTF-8
bytes `domain + "\n" + preimage`, where `preimage` is the document's deterministic pretty encoding
with only its own `artifactChecksum` field omitted and exactly one terminal LF. The domains are
`umpire.drive-plan/v2`, `umpire.experiment-spec/v2`, `umpire.runtime-configuration/v2`,
`umpire.experiment-run/v2`, `umpire.raw-evidence/v2`, `umpire.evidence/v2`, and `umpire.result/v2`.
The outer ExperimentSpec preimage contains the already-sealed DrivePlan. `provenanceChecksum` uses
domain `umpire.provenance/v2` over the exact pretty provenance subobject plus one LF; for retained
DrivePlan/ExperimentSpec it is a computed binding value, not a new JSON field.

An `ArtifactBinding` has the exact field order
`{formatVersion, artifactChecksum, behaviorFingerprint, provenanceChecksum}`. It never carries a
path. Every new family persists its own `behaviorFingerprint`, `provenance`,
`provenanceChecksum`, and `artifactChecksum`; the behavior fingerprint identifies the checked
producer/profile/program closure and excludes run-local facts, while the Artifact Checksum covers
the complete document. A binding is valid only when all four values independently match the bound
member.

### Normative retained transport schemas

The following rows freeze top-level field order. Nested records use the field order shown inline;
all arrays have the stated canonical order, duplicate identities reject, nullable fields are
present as JSON `null`, and no unlisted field is legal.

| Family | Format and exact top-level field order | Closed nested rules |
| --- | --- | --- |
| RuntimeConfiguration | `umpire-runtime-configuration/v2`: `{formatVersion, configurationDefinitionId, behaviorFingerprint, experiment, authorityProfile, phaseLimits, observation, participantBindings, knownGaps, provenance, provenanceChecksum, artifactChecksum}` | `experiment` is an ArtifactBinding. `authorityProfile` is `{definitionId, version, behaviorFingerprint, requiredCapabilityDefinitionIds}`. `phaseLimits` is ordered `preparation, realization, observation, isolation, cleanup`, each `{phase, durationMilliseconds, maxAttempts, maxRecords, maxBytes}`. `observation` is `{profileDefinitionId, profileBehaviorFingerprint, programDefinitionId, programBehaviorFingerprint, mappingDefinitionId, mappingBehaviorFingerprint}`. Participant bindings sort by Definition ID and are `{participantDefinitionId, protocolDefinitionId, protocolVersion, programDefinitionId, programBehaviorFingerprint, capabilityDefinitionIds}`. |
| ExperimentRun | `umpire-experiment-run/v2`: `{formatVersion, runIdentity, behaviorFingerprint, experiment, runtimeConfiguration, attempt, operationalStatus, phaseOutcomes, controlAttempts, sourceClosures, cleanup, limits, knownGaps, provenance, provenanceChecksum, artifactChecksum}` | `attempt` is positive. `operationalStatus` is `succeeded|failed|incomplete`. Phase outcomes occur once in the fixed five-phase order and are `{phase, status, startedAtUnixMillis, finishedAtUnixMillis, code}`, with status `not-started|succeeded|failed|timed-out|canceled`; non-started timestamps/code are null, started terminal rows have both timestamps and a closed nullable code. Control attempts sort by occurrence/attempt and are `{occurrenceDefinitionId, actionDefinitionId, attempt, receiptFactDefinitionId, status, code}`, status `accepted|rejected|unsupported|failed|canceled|not-attempted`. `not-attempted` requires null receipt/code; every attempted status requires one non-null receipt fact ID, which must resolve exactly once in RawEvidence to source `umpire.evidence.source.control-receipt` and kind `umpire.evidence.kind.control-receipt`, with the same occurrence/action/attempt/status. Source closures sort by source identity and are `{sourceDefinitionId, status, recordCount, byteCount}`, status `closed|partial|failed`. Cleanup is `{status, openHandleCount, code}`, status `complete|incomplete|failed`. |
| RawEvidence | `umpire-raw-evidence/v2`: `{formatVersion, runIdentity, behaviorFingerprint, experiment, runtimeConfiguration, run, captureStatus, sources, facts, knownGaps, provenance, provenanceChecksum, artifactChecksum}` | The three inputs are ArtifactBindings. `captureStatus` is `closed|partial|failed`. Sources sort by Definition ID and are `{sourceDefinitionId, status, factCount, byteCount}` with the same three statuses. Facts sort by `(sourceDefinitionId, ordinal, factDefinitionId)` and are `{factDefinitionId, sourceDefinitionId, ordinal, kindDefinitionId, causalFactDefinitionIds, fields}`. Causal IDs precede the fact and form a DAG. Fields sort by Definition ID and are `{fieldDefinitionId, disposition, value}`, where disposition is `plain|redacted|sha256|rejected`; rejected values are null, sha256 values have checksum spelling, and plain values are only JSON null/bool/canonical integer/string. |
| Evidence | `umpire-evidence/v2`: `{formatVersion, runIdentity, behaviorFingerprint, experiment, runtimeConfiguration, run, rawEvidence, observationProgram, mapping, observationEvaluationStatus, evidenceBackedModelTrace, evidenceLinks, dispositions, diagnostics, knownGaps, provenance, provenanceChecksum, artifactChecksum}` | The first four references are ArtifactBindings. Observation program and mapping are `{definitionId, behaviorFingerprint}`. Status is `accepted|unknown|conflict|unsupported`. Only `accepted` has a non-null complete Evidence-backed Model Trace and a coordinate-sorted Evidence Link for every established Model Fact; it has no diagnostics. Every non-accepted row has null trace, empty links, and at least one canonical closed diagnostic. Dispositions retain one entry for every consumed or rejected raw field and never copy a prohibited raw value. |
| Result | `umpire-result/v2`: `{formatVersion, runIdentity, behaviorFingerprint, experiment, runtimeConfiguration, run, rawEvidence, evidence, operationalStatus, observationEvaluationStatus, implementationLink, implementationLinkStatus, propertyVerdicts, querySummary, semanticStatus, limits, knownGaps, cleanupStatus, evaluationOutcomeChecksum, provenance, provenanceChecksum, artifactChecksum}` | The five inputs are ArtifactBindings. `implementationLink` is the full `ImplementationLinkRecord` defined below, and status is `applied|invalid|unknown|conflict|unsupported|not-evaluated`; its embedded diagnostic must agree with that status. Property verdicts sort by Property Definition ID and use `satisfied|violated|unknown|conflict|unsupported`. `semanticStatus` is `satisfied` only for a complete all-satisfied required set, `violated` only for a complete resolved set with at least one violation, and otherwise `incomplete`. Verdicts exist only after accepted Observation Evaluation and an applied link. `evaluationOutcomeChecksum` is non-null only for a complete resolved `satisfied|violated` outcome; otherwise it is null. Operational, observation, link, semantic, Known Gap, Limit, and cleanup fields remain independent. |

`evaluationOutcomeChecksum` uses domain `umpire.evaluation-outcome/v2` and the deterministic pretty
Generated View of the stable trace, observation mapping/program, Implementation Link, query,
Properties, verdict clauses/spans/Limits, and allowed Evidence Links. It excludes run-local
timestamps, operational diagnostics, paths, and raw transport facts. The Result admission layer
recomputes it but never performs the evaluation.

### Exact Evidence and Result nested projections

These are transport projections of the named Lean values, not encodings of Lean constructor names.
Fields occur exactly in the shown order; an `Option` is always an explicit value or `null`.

- `ModelValue` is `{definitionId, value}`. `ModelCoordinate` is `{kind, step, position}` where kind is
  `initial-state|selected-action|model-outcome|resulting-state|observation`; `initial-state` requires
  both numbers null, the three step kinds require a positive step and null position, and observation
  requires positive step and position. Coordinates order by that kind order, then step, then position.
- `ModelTrace` is `{traceId, initialState, steps}`. Each contiguous one-based step is
  `{position, selectedAction, modelOutcome, resultingState, observations}`; observations retain
  their list position and every value is a `ModelValue`.
- `SourceLocation` is `{path, line, column, provenance}`. `Provenance` is
  `{sourceDefinitionIds, sourceLocations}` with definition IDs sorted and locations sorted by
  `(path,line,column,provenance)`. `Limit` is `{value, unit}`. `EvidenceLimit` has unit
  `evidence-records`. `QueryLimits` is `{behavior, search}`, where behavior is
  `{transitions, selectedActions}` and each child is a Limit.
- `ObservationPlanReference` is `{definitionId, behaviorFingerprint}`. `MeaningProvision` is
  `{definitionId, kind, canonicalBehavior}`. `FieldReference` is
  `{kindDefinitionId, fieldDefinitionId}`. `FieldDispositionRecord` is
  `{field, disposition, digestPolicyDefinitionId}` with disposition `retain|redact|hash|reject`;
  only hash may have a non-null policy.
- `EvidenceOrderingFact` is
  `{factDefinitionId, kindDefinitionId, ordinal, causalFactDefinitionIds}` and
  `EvidenceClosureFact` is `{kindDefinitionId, lastOrdinal}`. `AppliedFieldDisposition` is
  `{field, kind, normalizedValue, digestPolicyDefinitionId, digestToken}` with kind
  `retained|redacted|digest-token`: retained has only normalizedValue, redacted has no variant
  values, and digest-token has policy and token. Raw/rejected-material Lean sentinels have no wire
  form and reject.
- `EvidenceLink` is
  `{coordinate, mappingDefinitionId, mappingVersion, mappingBehaviorFingerprint,
  profileDefinitionId, profileVersion, evidenceDefinitionIds, ruleDefinitionId,
  bindingDefinitionIds, orderingSupport, closureSupport, appliedDispositions, appliedLimit,
  meaningBehaviorFingerprint}`. Identity lists are sorted; ordering support sorts by fact ID;
  closure support sorts by kind ID; dispositions sort by field reference. Links sort by coordinate
  and are a bijection with established Model coordinates.
- `EvidenceBackedModelTrace` is the explicit projection
  `{traceId, observationPlan, mappingDefinitionId, mappingVersion, mappingBehaviorFingerprint,
  source, profileDefinitionId, profileVersion, sourceClosed, vocabulary, appliedLimit,
  evidenceDefinitionIds, trace}` from Lean `EvidenceBackedTrace`. Vocabulary sorts by Definition ID;
  evidence IDs sort; `trace.traceId` equals the outer trace ID. Evidence Links and disposition
  declarations live in their sibling Evidence fields and are not duplicated in this projection.
- `ObservationDiagnostic` is
  `{kind, observationPlanDefinitionId, relatedDefinitionIds, appliedLimit, observedCount,
  alternatives, missingDiscriminatorDefinitionId}`. Kind is the exact kebab-case projection of
  `ObservationFailureKind`; identity arrays sort. Accepted Evidence has null diagnostics represented
  by an empty array; each non-accepted Evidence has exactly one diagnostic whose derived status
  equals `observationEvaluationStatus`.
- `ImplementationTargetReference` is `{definitionId, kind, behaviorFingerprint}`.
  `ImplementationLinkRecord` is
  `{definitionId, behaviorFingerprint, sourceTarget, destinationTarget, diagnostic}`. Diagnostic is
  null for `applied|not-evaluated`; otherwise it is
  `{kind, coordinate, relatedDefinitionIds, sourceSetupBehaviorFingerprint, appliedLimit,
  observedCount, knownGapCode, knownGapReason, unsupportedVocabularyKind,
  evidenceLinkBehaviorFingerprint, identity}` and its kind is the kebab-case projection of
  `ImplementationLinkFailureKind` whose class exactly equals `implementationLinkStatus`.
- `SemanticVerdictDiagnostic` is
  `{kind, relatedDefinitionIds, observationDiagnostic}`. `SemanticClauseVerdict` is
  `{propertyDefinitionId, clauseDefinitionId, status, coordinates, queryLimits, propertyLimit,
  evidenceLimit, provenanceDefinitionIds, evidenceLinks}`. `PropertyVerdict` is
  `{queryDefinitionId, propertyDefinitionId, propertyBehaviorFingerprint, traceId, status,
  queryLimits, evidenceLimit, provenanceDefinitionIds, clauses, diagnostic}`. Clauses sort by clause
  ID, coordinates by ModelCoordinate order, and links by coordinate; resolved satisfied/violated
  verdicts have a trace ID, evidence limit, nonempty clauses, and null diagnostic, while every
  non-resolved verdict has an exact diagnostic and no invented resolved clause.
- `QuerySummary` is
  `{queryDefinitionId, status, queryLimits, requiredPropertyDefinitionIds, propertyVerdicts,
  missingPropertyDefinitionIds, duplicatePropertyDefinitionIds,
  unexpectedPropertyDefinitionIds, divergentPropertyDefinitionIds,
  wrongQueryResultDefinitionIds, traceIds}` with status `satisfied|violated|incomplete` and every ID
  list sorted. Its embedded verdicts are byte-identical to Result `propertyVerdicts`; satisfied and
  violated require empty structural-error lists and one shared non-null trace ID, otherwise status
  is incomplete.
- Result `limits` is an array of `{stage, limit}` sorted by the closed stage order
  `observation-evaluation, implementation-link, query, property`; absent stage limits are omitted.
  `knownGaps` uses fn-37's exact record/order. `cleanupStatus` is
  `complete|incomplete|failed` and must equal the bound ExperimentRun cleanup status.

All Definition-ID arrays use ascending ASCII spelling (the admitted identifier alphabet is ASCII),
all enum names are the existing public kebab-case names, and all digests use the checksum/fingerprint
wire spelling. The Go admission layer validates these projections and relationships but never
constructs them from raw Evidence or reevaluates their meaning.

### Bounded admission constants and precedence

The scanner counts UTF-8 input bytes including the terminal LF; one JSON punctuation token or scalar
is one token; depth counts the root as one; collection cardinality is checked before allocating the
next element. The hard ceilings are 32 MiB/document, 1,048,576 tokens, depth 32, 4,096 array items,
256 object members, 1 MiB/string, six set members, 64 evidence sources, 4,096 evidence facts,
128 fields/fact, 1 MiB decoded payload/fact, and 16 MiB decoded RawEvidence payload in aggregate.
Definition IDs, run IDs, codes, and checksum/fingerprint strings are additionally 1--512 UTF-8 bytes;
diagnostic/detail strings are at most 4,096 bytes. Exactly N passes and N+1 returns the matching
stable Limit error before the N+1 allocation or append.

Stable admission error precedence is `byte-limit`, `syntax`, `token-limit`, `depth-limit`,
`duplicate-key`, `case-collision`, `unsupported-format`, `wrong-family`, `unknown-field`,
`collection-limit`, `string-limit`, `payload-limit`, `malformed-value`, `noncanonical`,
`provenance-checksum`, `artifact-checksum`, then `closure`. Unsupported format therefore precedes
all field-value, canonicality, checksum, and relationship checks after a bounded structural scan.

### Artifact-set manifest and atomic visibility

Only three closures exist: the two-member executable set (ExperimentSpec, RuntimeConfiguration),
the four-member execution set (plus ExperimentRun and RawEvidence), and the six-member evaluation
set (plus Evidence and Result). DrivePlan remains nested in ExperimentSpec. Exact member paths are
`artifacts/experiment.json`, `artifacts/runtime-configuration.json`,
`artifacts/experiment-run.json`, `artifacts/raw-evidence.json`, `artifacts/evidence.json`, and
`artifacts/result.json`; absent later-stage paths are omitted.

`manifest.json` is deterministic pretty `umpire-artifact-set/v2` with exact field order
`{formatVersion, artifactSetIdentity, members, artifactSetChecksum}`. Members occur in the closure
order above and are `{path, formatVersion, artifactChecksum, behaviorFingerprint,
provenanceChecksum}`. `artifactSetIdentity` is `umpire.artifact-set.` plus the lowercase hex digest
of domain `umpire.artifact-set-identity/v2` over the ordered member rows. `artifactSetChecksum` uses
the ordinary checksum rule with domain `umpire.artifact-set/v2`, omitting only itself. The returned
`manifestSha256` is the raw SHA-256 of the complete manifest bytes including its checksum.

`PublishSet(root, admittedSet)` fully encodes and validates a private sibling staging directory,
fsyncs each regular file and the staging directory, then atomically renames it to the previously
absent immutable destination `root/sets/<manifestSha256-hex>` and fsyncs `root/sets`. It never
replaces a mutable current pointer or multiple managed roots. A root lock serializes publishers;
an existing identical destination is reopened, fully revalidated, and returned idempotently, while
any conflicting/non-regular/symlinked entry rejects. `LoadSet(destination)` accepts only that exact
digest directory, opens `manifest.json` and every safe relative member without following symlinks,
checks regular-file mode and all declared sizes/checksums/bindings/closure, rehashes the manifest,
and returns an immutable admitted value. Readers can therefore observe a complete digest directory
or absence, never an old/new mixture; interrupted staging is unreachable and may be cleaned under
the publisher lock.

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
make umpire-check-artifact ARTIFACT=model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json FAMILY=umpire-experiment/v2
make umpire-check-artifact-set SET=tools/umpire/artifact/testdata/valid-run-evaluation-set
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** One bounded strict JSON kernel rejects malformed, compact, alternate-whitespace,
  noncanonical, oversized, duplicate, unknown-field, wrong-family, unsupported-version, and
  checksum-invalid bytes before returning a value. [paraphrase]
- **R2:** DrivePlan and ExperimentSpec v2 are the sole baseline and round-trip byte-for-byte in the
  one deterministic pretty representation. Lean and Go agree on field order, escaping, number
  spelling, two-space indentation, one terminal LF, and independently recomputed Artifact Checksums
  derived from exact pretty checksum preimages, with complete Definition ID, Behavior Fingerprint,
  Limit, Known Gap, occurrence, Property, and provenance validation. [user]
- **R3:** RuntimeConfiguration, ExperimentRun, and RawEvidence have minimal exact schemas and
  Artifact bindings while preserving the separation between configuration, Execution, and raw
  capture. [paraphrase]
- **R4:** Evidence persists Observation Evaluation and Evidence Links without remapping raw facts;
  Result keeps operational, Observation Evaluation, Implementation Link, Property, Known Gap, and
  cleanup statuses independent and performs no Claim Assessment. [user]
- **R5:** Cross-language deterministic pretty goldens prove exact canonical bytes, Behavior
  Fingerprints, Artifact Checksums, and closed references for every retained family. [paraphrase]
- **R6:** Unsupported majors and every pre-v2 current Artifact reject without a reader, migration,
  alias, fallback, or repaired output. [user]
- **R7:** Complete-set admission and immutable atomic publication reject unsafe paths, partial or
  mixed sets, stale references, and interrupted installation while preserving prior complete
  state. [paraphrase]
- **R8:** Public facades, root commands, tests, and architecture docs expose only the transport
  boundary and introduce no runtime, semantic, replay, coverage, verification-receipt, release, or
  Umpire3 machinery. [user]

## Early proof point

Task `.3` must prove that the canonical pretty v2 caller-closure fixture is accepted and re-encoded
byte-for-byte while compact JSON, alternate whitespace, representative earlier-format, legacy-key,
noncanonical, and checksum mutations reject. Later families do not proceed until Lean and Go agree
on those exact bytes and checksum preimages.

## Boundaries
<!-- scope: business -->

- No pre-v2 reader, migration, compatibility alias, deprecated forwarding module, or dual-format
  output.
- No compact compatibility input, fixture-only normalization adapter, semantic-equality substitute
  for byte goldens, or alternate pretty writer.
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
Evaluation. Retaining fn-37's v2 schemas and identities while deliberately replacing its pre-release
compact spelling with one readable deterministic pretty representation avoids carrying a
compatibility problem or a second writer when no external or immutable published v2 consumer exists.

### Implementation Tradeoffs
<!-- scope: technical -->

A deep admission module centralizes deterministic pretty byte generation, exact-byte admission,
Limit, checksum-preimage, and closure enforcement while keeping each Artifact family inert. One
immutable complete-set path is sufficient for the prototype; general migration and
Artifact-management machinery waits for a real post-v2 format change.

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
| R2 | Deterministic pretty v2 baseline | `.1`, `.3` | — |
| R3 | Runtime, Run, and RawEvidence | `.4`, `.5` | — |
| R4 | Evidence and Result | `.6`, `.8` | — |
| R5 | Cross-language pretty-byte identity goldens | `.3`, `.7` | — |
| R6 | Unsupported-format hard cut | `.3`, `.9` | — |
| R7 | Set closure and atomic publication | `.8`, `.10` | — |
| R8 | Integration and boundaries | `.1`–`.11` | — |
