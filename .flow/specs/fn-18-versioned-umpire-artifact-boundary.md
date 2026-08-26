# Versioned Umpire artifact boundary

> HTML render lens: local file `.flow/artifacts/fn-18-versioned-umpire-artifact-boundary/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview

Create one strict versioned persistence boundary for current Umpire artifacts. Lean remains authoritative for semantic structures and canonical projections; one deep Go package, `tools/umpire/artifact`, performs bounded byte admission, exact version dispatch, structural and cross-document validation, deterministic named migrations, and atomic complete-set publication.

The boundary covers the existing `umpire-drive-plan/v1` and `umpire-experiment/v1`, plus the minimal runtime configuration, run, raw-evidence, semantic-evidence, Result, coverage-report, and coverage-checkpoint formats required by the next runtime, conformance, exploration-resume, and replay slices. It does not plan, execute, collect or interpret evidence, score coverage, compute verdicts, replay, minimize, promote, or qualify a release.

## Goal & Context
<!-- scope: business -->

An inspectable `ExperimentSpec` is not executable infrastructure until every consumer can prove it read exactly the artifact Lean wrote. Success means malformed, stale, ambiguous, oversized, noncanonical, partially published, or cross-boundary-inconsistent artifacts fail before runtime behavior begins, while valid artifacts round-trip byte-for-byte and publish as one recoverable immutable set.

## Architecture & Data Models
<!-- scope: technical -->

```text
Lean semantic values + canonical encoders
                  |
                  v
          canonical JSON + LF
                  |
                  v
 tools/umpire/artifact strict boundary
 parse -> version -> shape -> invariants -> identity -> bindings
                  |
          +-------+--------+
          |                |
          v                v
      admitted value   immutable ArtifactSet
                           |
                           v
                 later runtime/conformance/replay
```

`model/Umpire/Artifact.lean` stays the public facade but becomes a vertical package over `Umpire.Artifact.Experiment`, `Runtime`, `Evidence`, `Result`, `Coverage`, and `Canonical`. Existing declarations and comments move without compatibility aliases. New Lean records are inert schema values and canonical encoders; they do not contain callbacks, IO, runtime adapters, or evidence interpretation.

`tools/umpire/artifact` is the single persisted-reader authority. Its public surface is a small set of typed `Decode*`, `Encode*`, `ValidateSet`, `MigrateSet`, and `PublishSet` operations over admitted wire values. Go wire structs mirror canonical Lean projections and carry no target transition logic, Property evaluator, Observation mapper, exploration scorer, or promotion behavior.

### Format family

| Exact format | Meaning and boundary |
| --- | --- |
| `umpire-drive-plan/v1` | Existing environment-independent requested plan; target-owned model outcomes remain model facts, not realized outcomes. |
| `umpire-experiment/v1` | Existing Lean-selected executable specification and semantic identity, with fn-16's populated intent arrays and unchanged wire shape. |
| `umpire-runtime-configuration/v1` | Authorized resolved profile/protocol/participant/evidence-profile identities and explicit execution/observation/cleanup bounds; no endpoint, credential, namespace, executable, or arbitrary environment option. |
| `umpire-experiment-run/v1` | One operational realization binding exact experiment/configuration artifacts, seed/attempt, phase outcomes, control attempts/receipt fact IDs, source capture summary, omissions, and cleanup outcome; no satisfied/violated claim. |
| `umpire-raw-evidence/v1` | Typed implementation records with exact experiment/run/config bindings, sources, source-local ordinals, causal links, explicit gaps/closure, typed bounded fields or opaque payload digest; no semantic interpretation. |
| `umpire-semantic-evidence/v1` | Fn-4's qualified/unknown/conflict/unsupported outcome, mapping/profile identities, optional qualified-trace projection, derivations, dispositions, and diagnostics bound to exact raw evidence; no Property verdict. |
| `umpire-result/v1` | Exact experiment/run/raw/semantic-evidence bindings, independent operational/qualification/semantic statuses, per-property verdicts and query summary, omissions, provenance, and mechanically checked `qualifiedOutcomeIdentity`. |
| `umpire-coverage-report/v1` | Fn-17's exact canonical report; fn-18 reads and validates but never scores or reinterprets it. |
| `umpire-coverage-checkpoint/v1` | Persisted envelope for the exact fn-17 immutable `CoverageState` plus bindings for selected and pinned ExperimentSpecs; no second exploration state model. |
| `umpire-artifact-set/v1` | Canonical manifest of exact member bindings and relationship closure used for validation and atomic publication. |

Every artifact-to-artifact reference is an `ArtifactBinding` with exact format version, artifact identity, SHA-256 of canonical persisted bytes, schema-defined semantic identity/digest when applicable, SHA-256 of canonical provenance, and one nullable validated bundle-relative path. Content, semantic, and provenance identities are distinct and all provided components are verified. Query, Property, Observation-program, mapping, profile, and capability identities are `SemanticReference`s, not artifact families: they resolve against the exact query/property fields embedded in the bound ExperimentSpec or the profile/program/mapping/capability fields embedded in the bound RuntimeConfiguration. Set closure therefore never requires a nonexistent query, Property, or Observation-program document.

Existing ExperimentSpec v1 is not redesigned. The decoder recomputes the embedded DrivePlan and ExperimentSpec semantic identities from the same canonical semantic views as Lean and rejects disagreement. Non-empty `selectedChoices`, `selectedVariants`, and `requestedFaults` are valid request intent. Choices are unique/canonical by axis; fault IDs are unique/canonical and reference occurrences in the linear extension; selected variants remain semantic values; fault capabilities appear in requirements. Planned action/outcome/state/checkpoint lengths and positions, occurrences/actions, bindings, properties, bounds, requirements, omissions, and provenance must agree.

### Normative v1 wire contract

The field lists below are the schema and canonical field order. Every listed field is required. Arrays are non-null (including when empty); nullable scalars/objects are encoded as explicit `null`, never omitted. Arrays described as sets are strictly increasing by the named identity and contain no duplicate; sequence arrays retain stated semantic order. Unless a tighter bound is stated, lists contain at most 4096 items, strings are valid UTF-8, closed identities are namespaced Umpire `DeclarationId`s, and unsigned values use the common numeric ceiling.

Common records are exact:

- `ArtifactBinding = {formatVersion, artifactIdentity, contentSha256, semanticIdentity, semanticDigest, provenanceSha256, path}`. `semanticIdentity`, `semanticDigest`, and `path` are nullable. `path` is null for standalone admission and non-null inside a set. The two SHA fields are exactly 64 lowercase hex characters. The binding is checked against decoded canonical member bytes, semantic fields, provenance, and manifest path.
- `SemanticReference = {kind, identity, version, semanticDigest}` where `kind` is `query|property|observation-program|mapping|profile|capability`, `version` is positive, and the other fields reproduce an embedded semantic declaration. It is never followed as a file reference.
- `ArtifactProvenance = {sourceIdentities, sources}` reuses the current ExperimentSpec v1 definition. Each `source = {path, line, column, provenance}`; identities and sources use the existing canonical order and deduplication rules.
- `Omission = {code, subject, detail}`. `code` and `subject` are closed identifiers; `detail` is nullable and may contain only non-secret diagnostic text.

`umpire-runtime-configuration/v1` is exactly `{formatVersion, profile, profileRequiredCapabilities, participantBindings, evidenceProfile, observationProgram, mapping, phaseBudgets, capabilities, omissions, semanticIdentity, provenance}`:

- `profile`, `evidenceProfile`, `observationProgram`, and `mapping` are `SemanticReference`s of their matching kinds. `profileRequiredCapabilities` is the complete sorted capability-reference projection committed by `profile.semanticDigest`; it is persisted specifically so admission is inert and needs no external profile lookup.
- `participantBinding = {identity, protocolIdentity, protocolVersion, protocolSemanticDigest, roleBindings, capabilities}`. `protocolVersion` is positive; role bindings use the existing ExperimentSpec v1 `RoleBinding` wire shape; capabilities are sorted `SemanticReference`s of kind `capability`. There are 1–64 participant bindings sorted by identity.
- `phaseBudget = {phase, timeMilliseconds, attempts, records, bytes}`. There are exactly five records in `preparation|realization|observation|isolation|cleanup` order, each with at least one positive bound and no zero-only record.
- `capabilities` is exactly the sorted union of every participant capability and `profileRequiredCapabilities`; both omissions and extras reject locally. `profile.semanticDigest` must equal `umpire-semantic/v1:` plus the canonical JSON object `{kind, identity, version, profileRequiredCapabilities}` in that order, using the profile reference's first three fields and deliberately excluding `semanticDigest` itself. Changing the profile identity/version or required capabilities therefore invalidates the reference without a self-reference or external lookup. No field can encode addresses, credentials, namespaces, executable names, filesystem roots, headers, or arbitrary key/value options.
- `semanticIdentity` is recomputed from the canonical object above excluding itself and `provenance`.

`umpire-experiment-run/v1` is exactly `{formatVersion, runIdentity, experiment, runtimeConfiguration, seed, attempt, phases, controlAttempts, sourceClosures, omissions, artifactIdentity, provenance}`:

- `experiment` and `runtimeConfiguration` are `ArtifactBinding`s. `seed` is an unsigned 64-bit decimal value and `attempt` is positive.
- `phaseOutcome = {phase, status, startedAt, finishedAt, code, evidenceFactIdentities}`. Exactly five appear in phase order. Status is `not-started|succeeded|failed|timed-out|canceled`; timestamps are nullable UTC RFC3339Nano strings, both null for `not-started`, and both non-null with `startedAt <= finishedAt` otherwise. `code` is null exactly for `not-started|succeeded`; evidence fact identities are sorted.
- `controlAttempt = {identity, occurrenceIdentity, faultIdentity, participantIdentity, status, receiptFactIdentity}`. `faultIdentity` and `receiptFactIdentity` are nullable. Status is `not-attempted|accepted|rejected|unsupported|failed`; every planned occurrence/requested fault has exactly one consistent record, sorted by identity.
- `sourceClosure = {sourceIdentity, status, lastOrdinal, gaps}` where status is `closed|partial|failed`, `lastOrdinal` is nullable, and `gap = {first, last, reason}` uses `missing|dropped|redacted|unsupported`. Gaps are disjoint, increasing, within `lastOrdinal`, empty exactly for `closed`, and closures are sorted by source identity.
- `runIdentity` is stable caller input; `artifactIdentity` is recomputed from the canonical identity view excluding itself and `provenance`. Neither identity asserts semantic success.

`umpire-raw-evidence/v1` is exactly `{formatVersion, experiment, runtimeConfiguration, run, captureStatus, sources, facts, omissions, artifactIdentity, provenance}`:

- The first three reference fields are exact `ArtifactBinding`s. `captureStatus` is `closed|partial|failed` and agrees with every source closure.
- `source = {identity, kind, schemaIdentity, schemaVersion, schemaDigest, closureStatus, lastOrdinal, gaps}`. There are 1–64 sources, sorted by identity. Kind is `participant-output|history|control-receipt|log|metric|span|cleanup`; schema version is positive; closure fields obey the run gap rules.
- `fact = {identity, sourceIdentity, ordinal, kind, subject, causes, correlations, fields, payload}`. There are at most 4096 facts, ordered by `(sourceIdentity, ordinal, identity)`, with unique identity and strictly increasing zero-based source-local ordinals. Kind is `output|history-event|control-receipt|log-record|metric-sample|span-record|cleanup-record`; causes/correlations contain 0–64 sorted existing fact identities; causes form a DAG.
- `field = {name, kind, disposition, value}`. There are at most 128 fields sorted by name. Kind is `bool|uint|string|digest`; disposition is `retain|redact|hash`; `value` has the named JSON scalar type for `retain`, is null for `redact`, and is a 64-character lowercase digest string for `hash`.
- `payload` is null or `{mediaType, schemaIdentity, encoding, digest, size, content}`. Encoding is `utf8|base64|digest-only`; digest and size cover decoded content, content is null exactly for `digest-only`, and decoded payload is at most 1 MiB.
- `artifactIdentity` is recomputed from experiment/config/run bindings, capture status, source/fact identities, ordinals, relationships, dispositions, and content digests, excluding retained payload bytes, itself, and provenance.

`umpire-semantic-evidence/v1` is exactly `{formatVersion, experiment, runtimeConfiguration, run, rawEvidence, observationProgram, mapping, qualificationStatus, qualifiedTrace, derivations, dispositions, diagnostics, omissions, semanticIdentity, provenance}`:

- The first four reference fields are `ArtifactBinding`s. `observationProgram` and `mapping` are `SemanticReference`s that exactly equal the corresponding embedded RuntimeConfiguration references.
- `qualifiedTrace` is null or `{identity, initialState, steps, sourceProfileIdentity, closedSources, appliedEvidenceBound}`. It reuses the existing `SemanticValue`, `SemanticTraceStep`, and `TypedBound` wire shapes; `semanticTraceStep = {selectedAction, modelOutcome, resultingState, observations}` and semantic step order is preserved.
- `derivation = {coordinateIdentity, mappingIdentity, evidenceFactIdentities, bindings, orderingFactIdentities, closureSourceIdentities, dispositionIdentities, appliedEvidenceBound}`. Derivations are sorted by the fn-4 coordinate order and form a bijection with every qualified trace slot.
- `disposition = {identity, evidenceFactIdentity, fieldName, policy, retainedValue, digest}`. Policy is `retain|redact|hash|reject`; `retainedValue` is non-null only for `retain`, `digest` only for `hash`, and both are null otherwise. Dispositions are sorted by identity and cover every consumed field exactly once.
- `diagnostic = {kind, subject, evidenceFactIdentities, alternatives, missingDiscriminators}` where kind is `unknown|conflict|unsupported`; all reference arrays are sorted.
- Qualification is `qualified|unknown|conflict|unsupported`. `qualified` requires one non-null trace, coordinate-complete derivations/dispositions, and no diagnostics. Every other status requires a null trace, empty derivations, and at least one matching diagnostic. `semanticIdentity` is recomputed from the stable semantic projection excluding transport artifact identities, timestamps, paths, itself, and provenance.

`umpire-result/v1` is exactly `{formatVersion, experiment, runtimeConfiguration, run, rawEvidence, semanticEvidence, query, properties, operationalStatus, qualificationStatus, semanticStatus, propertyVerdicts, querySummary, diagnostics, omissions, qualifiedOutcomeIdentity, artifactIdentity, provenance}`:

- The first five reference fields are `ArtifactBinding`s. `query` and sorted `properties` are `SemanticReference`s that exactly reproduce the query and complete property set embedded in ExperimentSpec; there are no query/property artifact members.
- `propertyVerdict = {property, status, clauses, responsibleClauseIdentities, spans, appliedBounds, derivationIdentities}`. `property` is its matching reference; status is `satisfied|violated|unknown|conflict|unsupported`; clauses and responsible identities reproduce fn-4 evaluation. `clause = {identity, status, coordinateIdentities}`, `span = {firstCoordinateIdentity, lastCoordinateIdentity}`, and bounds reuse exact `TypedBound` wire values. Verdicts are sorted by property identity and exactly one exists per required property.
- `querySummary = {status, satisfied, violated, incomplete}` where status is `satisfied|violated|incomplete`; the three arrays partition the complete sorted property set and agree with verdict statuses. `unknown|conflict|unsupported` verdicts enter `incomplete`.
- `diagnostic` reuses the semantic-evidence shape and is sorted. `artifactIdentity` is recomputed from the entire identity view excluding itself and provenance.

Result validity is the following complete matrix; operational status is independently any of `succeeded|failed|incomplete` in every row and never changes the semantic row:

| qualification | allowed property verdicts | query/semantic status | diagnostics | qualified outcome |
| --- | --- | --- | --- | --- |
| `unknown|conflict|unsupported` | no evaluation: all properties have the matching non-resolved status with empty clause/span/derivation arrays | `incomplete` / `incomplete` | at least one matching diagnostic | null |
| `qualified` | every verdict `satisfied` | `satisfied` / `satisfied` | empty | required |
| `qualified` | every verdict resolved and at least one `violated` | `violated` / `violated` | empty | required |
| `qualified` | at least one `unknown|conflict|unsupported` | `incomplete` / `incomplete` | at least one matching diagnostic | null |

For qualified rows, resolved verdicts require non-empty clause projections and exact derivation references; non-resolved verdicts require empty responsible clauses/spans and a diagnostic. `qualifiedOutcomeIdentity` is exactly `umpire-qualified-outcome/v1:` plus the canonical stable semantic view of ExperimentSpec query/plan, RuntimeConfiguration Observation-program/mapping, the sorted Property set, qualified trace, semantic status, property verdicts, clauses/spans/bounds, and allowed derivation semantics. It excludes run/raw/semantic artifact identities, timestamps, environment/namespace, participant/source/fact/receipt/cleanup identities, paths, omissions, and provenance.

`umpire-coverage-report/v1` retains fn-17's canonical wire unchanged: `{formatVersion, sourceDigest, policyDigest, stateIdentity, universeDigest, selected, omitted, directInteractions, equivalentInteractions, coordinateHits, goalCredits, pinned, exploratory, seed, selectionBudgetCeiling, termination, reportIdentity, provenance}`. Its nested candidate/omission/interaction/hit/credit values and their order are exactly the fn-17 canonical encoder; fn-18 may mirror but not redefine, sort, or repair them.

`umpire-coverage-checkpoint/v1` is exactly `{formatVersion, spaceDigest, universeDigest, policyCompatibilityDigest, goalDigest, symmetryDigest, pinnedSetDigest, selectionBudgetCeiling, selectedIdentities, omittedIdentities, coordinateHitSets, goalCreditSets, cursor, stateProvenance, stateIdentity, report, selectedSpecs, pinnedSpecs, artifactIdentity, provenance}`. Identity sets are sorted/disjoint, hit/credit maps use fn-17 key order, `cursor` is the exact fn-17 cursor, `report` is one `ArtifactBinding`, and spec bindings are sorted by semantic identity. All state/report/partition/count/digest relations are recomputed.

`umpire-artifact-set/v1` is exactly `{formatVersion, members, relationships, artifactIdentity, provenance}`. `members` are `ArtifactBinding`s sorted by `(formatVersion, artifactIdentity)` with non-null unique paths. `relationship = {kind, fromArtifactIdentity, toArtifactIdentity}` is sorted by all fields; its closed artifact-only kinds are `run-experiment|run-runtime-configuration|raw-experiment|raw-runtime-configuration|raw-run|semantic-experiment|semantic-runtime-configuration|semantic-run|semantic-raw-evidence|result-experiment|result-runtime-configuration|result-run|result-raw-evidence|result-semantic-evidence|checkpoint-report|checkpoint-selected-spec|checkpoint-pinned-spec`. The set identity is recomputed from the canonical member/relationship view excluding itself and provenance.

Identity formulas are uniform and literal: existing DrivePlan/ExperimentSpec identities remain `umpire-semantic/v1:` plus their unchanged canonical semantic JSON; new `semanticIdentity` fields use the same prefix plus their specified semantic view; new transport `artifactIdentity` fields use `umpire-artifact/v1:` plus their specified canonical identity view; `qualifiedOutcomeIdentity` uses its prefix above. SHA-256 values are lowercase hex SHA-256 of exact persisted bytes including LF, except `provenanceSha256`, which hashes the canonical `ArtifactProvenance` object without LF. No implementation may substitute a hash for an inline semantic identity or vice versa.

### Runtime and evidence schemas

`RuntimeConfiguration` contains profile identity/version/digest, participant protocol/binding identities, evidence-profile, Observation-program and mapping references, exact phase budgets, and declared capabilities. It deliberately cannot carry connection or authority material; fn-19 supplies those out of band from an authorized local profile.

`ExperimentRun` has one stable run identity and artifact identity, exact ExperimentSpec/RuntimeConfiguration bindings, seed and attempt number, one closed phase outcome for preparation/realization/observation/isolation/cleanup, canonical control attempts bound to planned occurrences/requested faults, referenced receipt fact identities, source closure summary, omissions, and provenance. Phase outcome is `not-started`, `succeeded`, `failed`, `timed-out`, or `canceled`. It never carries a Property or semantic verdict.

`RawEvidence` contains 1–64 sources and at most 4096 facts. Sources have closed kinds, schema/media identity, version/digest, exact closure status, and declared gaps. Facts have unique ID, source, nonnegative strictly increasing source-local ordinal, closed kind, subject, zero to 64 causal/correlation references, and zero to 128 typed fields. A field is one closed bool, unsigned integer, string, or payload-digest value with explicit retain/redact/hash disposition; retained opaque payload is separately bounded and digest-checked. Causal references must exist and form a DAG. Partial capture is representable but cannot be silently called closed.

`SemanticEvidence` mirrors fn-4's checked outcome rather than defining another interpretation. `qualified` requires exactly one qualified trace and coordinate-complete derivations/dispositions; `unknown`, `conflict`, and `unsupported` forbid a qualified trace and require canonical diagnostics. Raw field values prohibited by dispositions cannot reappear. Fn-18 checks representation and identity relations only; fn-4/fn-20 produce the meaning.

`Result` records independent operational (`succeeded|failed|incomplete`), qualification (`qualified|unknown|conflict|unsupported`), and semantic (`satisfied|violated|incomplete`) statuses plus exact fn-4 verdict/clause/span/bound projections. The normative matrix above determines all valid combinations. Fn-20 computes `qualifiedOutcomeIdentity`; fn-18 only enforces the exact formula.

Fn-17 remains the authority for coverage meaning and report bytes. `CoverageCheckpoint` persists its exact space, universe, policy-compatibility, goal, symmetry, pinned-set and state identities/digests; recorded selection-budget ceiling; selected/omitted identities; per-coordinate hit sets; per-goal distinct credited-spec sets; cursor/provenance; report binding; and selected/pinned ExperimentSpec bindings. Structural validation enforces canonical disjoint partitions, count/set equality, reference closure, identity recomputation, and report/state agreement. Resume compatibility and monotonic selection behavior remain fn-17 responsibilities after admission.

## Strict Byte Contract
<!-- scope: technical -->

All readers share one recursive streaming parser and family-specific validators:

- bounded `io.Reader` with N+1 overflow detection before unbounded allocation;
- valid UTF-8, no BOM, exactly one top-level object, and exactly one terminal LF;
- no leading whitespace, CRLF, trailing whitespace/token, concatenated value, or truncation;
- duplicate, case-variant, and case-fold-colliding keys rejected at every depth;
- every unknown field rejected at every schema level; v1 has no extension bag or alias;
- exact format strings and closed enums; unknown majors reject before family decoding;
- unsigned integers only, canonical decimal spelling, range 0–9,007,199,254,740,991; no negative, fraction, exponent, or leading zero;
- canonical schema field order, string escaping, array order, set uniqueness, and identity order are admission requirements; readers never sort or normalize accepted input; and
- decode → validate → canonical re-encode must equal the exact input bytes.

Common limits are depth 32, 128 object members, and 131,072 JSON tokens. The exact string-limit table is:

| Field class | Maximum UTF-8 bytes |
| --- | ---: |
| declaration/run/source/fact/coordinate/member identifiers, enum/code/media/schema fields | 512 |
| `umpire-semantic/v1:` inline identities, including current ExperimentSpec `semanticIdentity` and DrivePlan `plan.semanticIdentity` | 524,288 |
| new `umpire-artifact/v1:` inline identities and `qualifiedOutcomeIdentity` | 524,288 |
| embedded `semanticDigest` fields, including the current 71-KiB Nexus fixture values | 524,288 |
| bundle-relative path | 1,024 |
| retained evidence field or diagnostic/detail value | 65,536 |
| retained decoded opaque payload | 1,048,576 |

The 512-byte limit therefore applies only to compact identifiers, never to existing inline semantic identities. SHA fields are exactly 64 ASCII bytes. Family byte ceilings are 1 MiB for DrivePlan/ExperimentSpec, 4 MiB for RuntimeConfiguration/ExperimentRun, 16 MiB for RawEvidence/SemanticEvidence, 8 MiB for Result, 16 MiB for CoverageReport/CoverageCheckpoint, and 2 MiB for an ArtifactSet manifest. A set contains at most 512 members and 320 MiB of member bytes, processed one member at a time. Schema-specific tighter list bounds above override generic limits. Exceeding any limit returns its stable name, ceiling, and observed N+1 value; no truncation occurs.

Canonical in-memory encoders return a JSON document without LF, preserving the existing Lean API. Canonical persisted/stdout bytes are exactly that document plus one LF, matching current fixtures and inspector output. Semantic identities exclude this transport LF; content SHA-256 includes it.

## Artifact Sets, Migrations, and Publication
<!-- scope: technical -->

An `ArtifactSet` manifest names every member binding and required artifact relationship. Validation admits each member independently and then requires closure: run→experiment/configuration; raw evidence→run/experiment/configuration; semantic evidence→raw/run/experiment/configuration; Result→all prior artifacts; report/checkpoint→the same space/universe/policy/state and all named selected/pinned specs. Semantic references are instead resolved against fields embedded in those closed artifacts: semantic evidence program/mapping→RuntimeConfiguration, and Result query/properties→ExperimentSpec. Missing, extra, duplicate, stale, mixed-version, path-mismatched, unresolved-semantic-reference, or cross-boundary-inconsistent members reject the whole set.

The migration engine is a closed registry of exact source/target formats and unique names. Each edge strictly validates source, deterministically transforms, and strictly validates target. Downgrades, guessing, aliases, ambiguous routes, skipped unknown versions, semantic reinterpretation, and partial set migration reject. Writers emit current versions only. Because no superseded production format exists, the production migration registry is intentionally empty; a private fixture family proves deterministic one-way multi-step and ambiguity rejection until a real predecessor exists. No fake product version is invented.

Publication reuses and, only where necessary, deepens `tools/common/artifactio` locking, safe-path, staging, fsync, validation, rollback, and interruption recovery. The exact destination is an immutable directory keyed by the SHA-256 of canonical `umpire-artifact-set/v1` manifest bytes. Member paths are normalized bundle-relative paths under that directory; absolute paths, traversal, symlinks, non-regular files, or collisions reject. Files are mode 0600 and directories 0700. A staged set is fully decoded and cross-validated before install, then re-opened and revalidated before success. Concurrent identical publication is idempotent; conflicting existing bytes reject. A failed/interrupted migration or publication leaves every prior complete set unchanged and recoverable. Sets are never overwritten in place and no mutable `current` pointer is created.

## API Contracts
<!-- scope: technical -->

- `Decode*` accepts canonical persisted bytes for one expected family and returns one typed admitted value or a structured `ArtifactError` classified as read-limit, syntax, family/version, schema, canonicality, identity, provenance, binding, relationship, migration, path, publication, or recovery.
- `ValidateSet` requires exact manifest/member closure and returns an inert admitted set; it never follows external paths or fetches missing artifacts.
- `MigrateSet` produces a new admitted set in memory; it never modifies source bytes or publishes a partial destination.
- `PublishSet` accepts only an admitted current-version set and a caller-provided root. It owns safe deterministic layout and atomic visibility, but not retention/deletion policy.
- `umpire-artifact check --family <exact-format> <path>`, `check-set <directory>`, and `publish-set --root <directory> <staged-directory>` are the exact CLI forms. `migrate-set --to <exact-format-set> --output <directory> <source-directory>` exists but rejects every production v1 input with `no-migration-route` until a real route is registered. Success writes one canonical summary JSON line; failure writes one canonical error line to stderr, no stdout, and status 1.
- Root Make exposes `make umpire-check-artifact ARTIFACT=<path> FAMILY=<exact-format>` and `make umpire-check-artifact-set SET=<directory>`. Publication and migration remain direct explicit CLI operations and are never implicit in a check target.

## Edge Cases & Constraints
<!-- scope: technical -->

- A single artifact may decode independently, but a binding is only established when its referenced artifact is provided to set validation. Orphan references never become an admitted set.
- Provenance may differ without changing an existing semantic identity only where the Lean semantic projection already excludes it; its separate digest and binding must still match exact bytes.
- Noncanonical but semantically equivalent JSON is rejected rather than rewritten. Migration begins only after strict source admission.
- Empty, partial, duplicate, reordered, or unknown collections are handled exactly by each schema; omission is never inferred.
- Run/evidence/result content never mutates or replaces the ExperimentSpec semantic identity.
- Coverage checkpoint admission does not prove that resuming with a new ceiling is compatible; fn-17 checks that semantic operation.
- Atomic publication prevents partial visibility; it is not a runtime scheduler, coordination, lease, authorization, cleanup, or retention system.
- Existing comments are preserved when moving Artifact declarations into vertical modules.

## Quick commands
<!-- scope: technical -->

```bash
go test -count=1 ./tools/umpire/artifact/...
go test -count=1 ./tools/common/artifactio/...
cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs
cd model && mise exec -- lake build Umpire.Exploration.PersistenceTests
make umpire-check-artifact ARTIFACT=model/Temporal/Feature/Nexus/testdata/nexus-caller-closure-experiment-spec.json FAMILY=umpire-experiment/v1
make umpire-check-artifact-set SET=tools/umpire/artifact/testdata/valid-conformance-set
make umpire-check-regression
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** One bounded recursive strict JSON kernel and closed version dispatcher reject malformed, truncated, duplicate/case-colliding, unknown-field, noncanonical, oversized, trailing, wrong-family, or unsupported-version bytes before returning a value. Exact limits and structured failures are deterministic. [paraphrase]
- **R2:** Existing DrivePlan/ExperimentSpec v1 values decode with byte-identical canonical round trips and complete structural, semantic-identity, provenance, bounds, occurrence, checkpoint, property, and requirement validation. Fn-16's non-empty choice/variant/fault intent arrays retain their request-only meaning and unchanged wire shape. [user]
- **R3:** RuntimeConfiguration, ExperimentRun, and RawEvidence have minimal exact versioned schemas and artifact bindings. Runtime config carries no authority material, Run carries no semantic verdict, RawEvidence carries no interpretation, and cross-document invariant violations fail set admission. [paraphrase]
- **R4:** SemanticEvidence is the persisted projection of fn-4 qualification and derivations, not a second mapper; Result binds the full chain and keeps operational/qualification/semantic status independent. `qualifiedOutcomeIdentity` is present/recomputed only for qualified satisfied/violated semantics and excludes run-specific transport facts. [user]
- **R5:** Fn-17's report bytes decode strictly, and exact immutable exploration state persists only as `umpire-coverage-checkpoint/v1` with report/spec bindings and structural/digest validation. Fn-18 never scores coverage or decides resume compatibility. [paraphrase]
- **R6:** Named migrations are current-version-only writers, deterministic one-way exact routes, strict before/after every step, and complete-set-only. The production registry is honestly empty until a predecessor exists; private fixtures prove the engine without inventing a product version. [paraphrase]
- **R7:** Exact `umpire-artifact-set/v1` relationship validation and immutable atomic publication reject unsafe paths/files, partial/mixed/stale sets, concurrent conflicts, and interrupted installation while preserving prior complete sets. Identical republish is idempotent and read-after-write is strictly revalidated. [paraphrase]
- **R8:** Cross-language golden/mutation fixtures, fuzz tests, exact CLI/root commands, public facades, architecture docs, and roadmap status prove a transport-only boundary. No runtime driving, evidence collection/interpretation, coverage scoring, replay/minimization/promotion, qualification, CI/release claim, model-local Makefile, or Umpire3 use is introduced. [user]

## Early proof point
<!-- scope: technical -->

Task `.3` is the compatibility proof gate. It must admit the current 164-KiB caller-closure fixture and a Lean-generated synthetic v1 artifact with non-empty choices, variants, and faults; recompute DrivePlan and ExperimentSpec identities independently in Go; re-encode exact bytes; and reject one-at-a-time mutations of every identity, bound, occurrence/action, intent, property, observation requirement, omission, and provenance relation. It must also prove byte-limit N+1 behavior and canonical rejection for every JSON token class. Tasks `.4`–`.11` cannot proceed if the Go/Lean projections disagree; the fix is the one existing v1 contract, never a permissive decoder or alternate IR.

## Boundaries
<!-- scope: business -->

- No planning, target semantics, Property evaluation, Observation interpretation, coverage scoring, or artifact repair.
- No runtime action/fault realization, evidence collection, environment authority, credentials, endpoints, namespaces, executables, or arbitrary options.
- No replay, minimization, promotion, generated regression, catalog/glossary, or release qualification.
- No fake product migration, compatibility alias, permissive extension bag, unknown-major fallback, or mutable in-place artifact rewrite.
- No model-local Makefile or CI workflow.
- No Umpire3 inspection, import, invocation, dependency, compatibility, or migration path.

## Decision Context
<!-- scope: both -->

A transport DTO in Go is unavoidable for a Go runtime, but it is not a semantic IR: Lean defines the fields, ordering, identities, and canonical examples; Go admits those bytes and exposes exact typed access. Cross-language golden and mutation tests prevent either side from drifting silently.

SemanticEvidence remains separate from Result because observation qualification and Property verdicts are different user-approved layers. Persisting qualified trace derivations gives Result a precise input binding and lets evidence corruption be tested without recomputing or hiding the interpretation boundary.

RuntimeConfiguration is portable policy identity, not authority material. Keeping addresses, credentials, namespaces, and executables out of it lets fn-19 bind one authorized local environment without turning persisted artifacts into ambient execution authority.

An empty production migration registry is more honest than inventing a predecessor. The framework and complete-set atomicity are implemented now; the first real schema change must add one named reviewed route.

## References
<!-- scope: technical -->

- `.plans/UMPIRE4_COMPONENTS.md:118-140,253-347,700-711` — artifact inventory, compiler/runtime/evidence boundaries, and pilot sequence.
- `.plans/UMPIRE4_DSL.md:220-243,291-324` — artifact meaning, strict versioning, and component seams.
- `.flow/specs/fn-4-umpire-observation-and-semantic-verdicts.md` — qualification, derivation, and semantic verdict authority.
- `.flow/specs/fn-16-authored-variation-spaces-and.md` — populated v1 intent and unchanged format contract.
- `.flow/specs/fn-17-bounded-semantic-exploration-and.md` — report, state identity, and persistence handoff.
- `model/Umpire/Artifact.lean:36-91,93-241,311-382` — existing canonical structures, encoders, omissions, and lowering.
- `tools/umpire/regression/projection.go:201-292` — duplicate/trailing JSON precedent.
- `tools/common/artifactio/artifact.go:10-40` and `set.go:16-110,475-645` — fsync, lock, staging, rollback, and recovery.
- `tools/umpire/internal/generate/regression/generate.go:116-176` — validate-before-publish pattern.

## Requirement coverage
<!-- scope: both -->

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Strict bounded JSON/version core | `.2`, `.3`, `.4`–`.9` | — |
| R2 | Existing ExperimentSpec compatibility | `.1`, `.3` | — |
| R3 | Runtime config/run/raw evidence | `.1`, `.4`, `.5`, `.8` | — |
| R4 | Semantic evidence/Result/outcome identity | `.1`, `.6`, `.8` | — |
| R5 | Coverage report/checkpoint | `.1`, `.7`, `.8` | — |
| R6 | Named migrations | `.2`, `.9` | — |
| R7 | Artifact sets/atomic publication | `.2`, `.8`, `.10` | — |
| R8 | Fixtures, commands, docs, boundaries | `.1`–`.11` | — |
