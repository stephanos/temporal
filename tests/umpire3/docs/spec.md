# SPECIFICATION — Umpire3 platform specification

This document indexes the current high-level functionality of the Umpire3 platform. Each leaf section defines one independently verifiable product requirement under a stable identifier; **MUST** denotes required current behavior.

## SPECIFICATION.PURPOSE — Platform purpose

Umpire3 MUST provide a coherent path from behavioral intent to bounded verification evidence without overstating what that evidence proves.

## SPECIFICATION.CURRENT-BEHAVIOR — Current behavior

This specification MUST describe supported current behavior rather than planned or experimental behavior.

## SPECIFICATION.IDENTIFIER-STABILITY — Requirement identity

Each requirement identifier MUST remain unique and stable so that code, tests, and evidence can refer to the requirement independently of its prose title.

## SPECIFICATION.ARTIFACT-FLOW — Artifact lifecycle

Umpire3 MUST preserve the lifecycle from Scenario through Experiment and Execution to Result, with Replay bundles and Qualification receipts retaining the bindings needed for later verification.

## VOCABULARY — Glossary

**Semantic catalog** — The authoritative, versioned vocabulary and meaning of supported resources, actions, properties, evidence, faults, capabilities, modules, and verification targets.

**Scenario** — Sparse author intent describing setup, threats, actions, and properties.

**Experiment** — A validated, versioned, executable semantic artifact compiled from a Scenario.

**Regression** — A reusable test that compiles a Scenario and executes every resulting Experiment.

**Execution** — One attempt to run one Experiment in one Environment.

**Environment** — The boundary that realizes actions, observes evidence, and performs cleanup.

**Deployment profile** — A contract describing an Environment's capabilities, maximum authority, isolation, attestation, and evidence behavior.

**Participant program** — A bounded program that realizes declared actions without deciding whether properties hold.

**Result** — The evidence and semantic claim produced by one Execution.

**Replay bundle** — A redacted, digest-bound package containing an Experiment, Result, and reproduction metadata.

**Qualification receipt** — Authenticated external-execution evidence binding a candidate release to an exact Experiment, Result, profile, build, and configuration.

## SEMANTICS — Semantic authority

### SEMANTICS.SINGLE-AUTHORITY — Single semantic source

Umpire3 MUST derive its supported resources, actions, properties, evidence, faults, capabilities, and verification targets from one versioned semantic authority.

### SEMANTICS.COMPOSITION — Composable guarantees

Umpire3 MUST compose declared guarantees, requirements, and omissions without allowing an adapter or runner to redefine their meaning.

### SEMANTICS.EXECUTABLE-EQUIVALENCE — Executable meaning

When a semantic contract has an executable view, Umpire3 MUST check that generated executable behavior remains equivalent to the declared meaning within its stated scope.

### SEMANTICS.CLAIM-BOUNDARIES — Claim separation

Umpire3 MUST distinguish semantic proof, bounded exploration, implementation validation, real-environment execution, and deployment qualification as different classes of claim.

## AUTHORING — Scenario authoring

### AUTHORING.SPARSE-INTENT — Sparse author intent

Umpire3 MUST let authors state setup, threats, actions, and properties without spelling out compiler-derived dependencies or transition mechanics.

### AUTHORING.TYPED-VOCABULARY — Typed vocabulary

Umpire3 MUST expose supported resources, actions, properties, faults, and observations as typed authoring concepts derived from the semantic authority.

### AUTHORING.STRUCTURAL-ORDERING — Structural ordering

Umpire3 MUST let authors express path choice, ordering, and concurrency structurally rather than through timing sleeps or adapter-local state.

### AUTHORING.RUNTIME-IDENTITY — Runtime identity

Umpire3 MUST let authors refer to logical resources while deferring concrete runtime identities to compilation and execution.

### AUTHORING.ENVIRONMENT-SEPARATION — Environment separation

Umpire3 MUST keep environment setup, credentials, and deployment-specific connection details outside Scenario intent.

## COMPILATION — Scenario compilation

### COMPILATION.SEMANTIC-COMPLETION — Semantic completion

Umpire3 MUST validate a Scenario and complete its dependencies, capabilities, observations, monitors, cleanup obligations, and runtime identity bindings from declared semantics.

### COMPILATION.DETERMINISTIC-PATHS — Deterministic paths

Umpire3 MUST compile the same Scenario and inputs into the same complete, stably ordered set of execution paths.

### COMPILATION.BOUNDED-FAILURE — Explicit bounds

Umpire3 MUST reject compilation when path or artifact bounds are exceeded instead of silently truncating behavior.

### COMPILATION.SOURCE-DIAGNOSTICS — Source diagnostics

Umpire3 MUST report invalid, conflicting, or unsupported intent with diagnostics tied to the authoring source that caused the problem.

### COMPILATION.CAPABILITY-DISCOVERY — Capability discovery

Umpire3 MUST identify required execution and evidence capabilities before any Environment is allocated.

## EXPERIMENT — Executable semantic artifact

### EXPERIMENT.SELF-CONTAINED — Self-contained artifact

An Experiment MUST contain the validated actions, dependencies, monitors, assumptions, bounds, capability requirements, evidence requirements, cleanup obligations, and identity projections needed for execution.

### EXPERIMENT.STRICT-VERSION — Strict versioning

An Experiment MUST use a strict versioned format that rejects unknown or incompatible structure.

### EXPERIMENT.MODEL-BINDING — Semantic binding

An Experiment MUST bind the exact semantic and generated artifacts that define its meaning.

### EXPERIMENT.DIGEST-IDENTITY — Content identity

An Experiment MUST have a deterministic content identity suitable for evidence, replay, qualification, and release bindings.

## EXECUTION — Experiment execution

### EXECUTION.PREFLIGHT — Preflight validation

Umpire3 MUST validate Experiment structure, required capabilities, Environment identity, and authority before performing actions.

### EXECUTION.ENVIRONMENT-PORTABILITY — Environment portability

Umpire3 MUST execute the same Experiment through conforming local, test-cluster, remote, black-box, and canary Environments when their declared capabilities satisfy its requirements.

### EXECUTION.LIFECYCLE — Execution lifecycle

An Execution MUST prepare the Environment, realize the Experiment actions, collect evidence, evaluate monitors, and attempt cleanup within declared bounds.

### EXECUTION.OUTCOME-CLASSIFICATION — Outcome classification

Umpire3 MUST distinguish conformance, violation, unsupported capability, incomplete or conflicting evidence, and cleanup failure in the Result.

### EXECUTION.RESULT-COMPLETENESS — Complete result

A Result MUST bind the Experiment, Environment identity, action receipts, observations, monitor evaluations, capability facts, cleanup status, and final claim outcome for one Execution.

## PARTICIPANT — Programmable participants

### PARTICIPANT.VALIDATED-PROGRAM — Validated program

Umpire3 MUST support bounded, validated participant programs that realize declared actions through permitted system operations.

### PARTICIPANT.PROPERTY-INDEPENDENCE — Property independence

A participant program MUST NOT decide whether a property holds or substitute its own state for semantic evidence.

## EVIDENCE — Evidence collection

### EVIDENCE.TYPED-NORMALIZATION — Typed normalization

Umpire3 MUST normalize observations and action receipts into the typed evidence vocabulary required by the Experiment.

### EVIDENCE.IDENTITY-LINEAGE — Identity lineage

Umpire3 MUST preserve the lineage between logical resources, runtime identities, observed resources, and the Environment that reported them.

### EVIDENCE.CAUSAL-ORDERING — Causal ordering

Umpire3 MUST retain enough ordering and timing information to evaluate declared monitors without inventing unobserved causality.

### EVIDENCE.QUALIFIED-FACTS — Qualified facts

Umpire3 MUST treat a monitor result as conclusive only when its required evidence is present, attributable, and within the declared observation scope.

### EVIDENCE.AMBIGUITY — Evidence ambiguity

Umpire3 MUST report missing, contradictory, stale, or identity-ambiguous evidence as inconclusive rather than as conformance.

## FAULT — Fault injection

### FAULT.FIRST-CLASS — First-class faults

Umpire3 MUST represent supported faults as typed, bounded Experiment actions with explicit capability requirements.

### FAULT.SCOPED-AUTHORITY — Scoped authority

Umpire3 MUST realize a fault only through an Environment capability and authority explicitly scoped to that fault and target.

### FAULT.OCCURRENCE-EVIDENCE — Occurrence evidence

Umpire3 MUST require evidence that a requested fault actually occurred before using the fault in a behavioral claim.

### FAULT.FOOTPRINT — Fault footprint

Umpire3 MUST record the affected resources, duration, recovery state, and cleanup obligations of every realized fault.

### FAULT.CLEANUP — Fault cleanup

Umpire3 MUST classify an Execution as inconclusive when it cannot establish that fault effects were removed within the cleanup bound.

## EXPLORATION — Unknown-behavior exploration

### EXPLORATION.BOUNDED-DISCOVERY — Bounded discovery

Umpire3 MUST explore generated schedules, action variants, and typed mutations only within explicit finite bounds.

### EXPLORATION.DETERMINISTIC-SEED — Deterministic seed

Umpire3 MUST reproduce the same exploration choices from the same Experiment, configuration, and seed.

### EXPLORATION.COVERAGE-GUIDANCE — Coverage guidance

Umpire3 MUST use retained semantic or implementation coverage signals to prioritize further exploration without changing property meaning.

### EXPLORATION.BUDGET-OMISSIONS — Budget omissions

Umpire3 MUST disclose unexplored work caused by schedule, time, mutation, or execution budgets.

### EXPLORATION.VIOLATION-PRESERVATION — Violation preservation

Umpire3 MUST preserve a discovered violation as reproducible evidence and support bounded minimization that retains the same violation.

### EXPLORATION.PROMOTION — Regression promotion

Umpire3 MUST support promoting a minimized discovered violation into a deterministic known-regression Scenario.

## REPLAY — Reproduction artifacts

### REPLAY.BUNDLE — Replay bundle

Umpire3 MUST retain a versioned Replay bundle for a violating or inconclusive Execution when retention is requested or required.

### REPLAY.REDACTION — Redacted content

A Replay bundle MUST exclude credentials and redact or digest sensitive payloads, endpoints, and runtime identifiers.

### REPLAY.STRICT-BINDING — Strict binding

A Replay bundle MUST bind the exact Experiment, Result, semantic artifacts, execution choices, and reproduction metadata it represents.

### REPLAY.REPRODUCTION — Reproduction

Umpire3 MUST reject replay when required bindings are invalid and otherwise reproduce the recorded execution choices within the selected Environment's capabilities.

### REPLAY.DRIFT — Drift classification

Umpire3 MUST distinguish semantic, realization, schedule, observation, evidence, and footprint drift when replay does not match the recorded Execution.

## GENERATION — Derived artifacts

### GENERATION.SEMANTIC-DERIVATION — Semantic derivation

Umpire3 MUST derive executable models, typed authoring vocabulary, monitors, dependency data, and verification metadata from the semantic authority.

### GENERATION.WIRE-SELECTION — Selected wire structure

Umpire3 MUST derive only the selected wire structures needed by supported semantics rather than treating all available protocol structure as meaningful.

### GENERATION.FIELD-DISPOSITION — Field disposition

Every selected wire field MUST have an explicit semantic, opaque, ignored, or rejected disposition.

### GENERATION.DETERMINISTIC-OUTPUT — Deterministic output

Umpire3 MUST generate stable artifacts from the same semantic and wire inputs and detect committed artifact drift.

## VERIFICATION — Verification portfolio

### VERIFICATION.EVIDENCE-LAYERS — Evidence layers

Umpire3 MUST keep proof, bounded model exploration, executable seam checks, synthetic implementation tests, real-environment tests, and qualification evidence independently identifiable.

### VERIFICATION.DEPENDENCY-PORTFOLIO — Dependency portfolio

Umpire3 MUST select the required verification checks from the semantic dependencies of the behavior under test.

### VERIFICATION.KNOWN-REGRESSIONS — Known regressions

Umpire3 MUST compile and execute every Experiment produced by a known-regression Scenario.

### VERIFICATION.TRUST-BOUNDARY — Trust boundary

Every verification result MUST disclose the trusted components and unsupported assumptions on which its claim depends.

### VERIFICATION.BOUND-DISCLOSURE — Bound disclosure

Every bounded verification result MUST disclose the state, schedule, depth, time, and resource bounds relevant to its claim.

### VERIFICATION.WITNESS-REPLAY — Witness replay

Umpire3 MUST replay retained counterexamples or violation witnesses against the applicable executable or real Environment before treating them as implementation evidence.

### VERIFICATION.NEGATIVE-CONTROL — Negative control

Umpire3 MUST verify that a known violating observation is detected as a violation rather than counting an evidence or infrastructure failure as success.

### VERIFICATION.FAIL-CLOSED — Verification failure

Umpire3 MUST fail a verification gate when required checks, evidence, bindings, or generated artifacts are missing, stale, duplicated, or contradictory.

## PROFILE — Deployment profiles

### PROFILE.PORTABILITY — Portable contract

A Deployment profile MUST describe an Environment through capabilities, isolation, authority, attestation, and evidence contracts rather than through test-specific assumptions.

### PROFILE.MAXIMUM-AUTHORITY — Maximum authority

A Deployment profile MUST declare the maximum actions and fault scopes an Environment is permitted to realize.

### PROFILE.CAPABILITY-INTERSECTION — Capability intersection

Umpire3 MUST execute only the intersection of Experiment requirements, profile capabilities, and granted authority.

### PROFILE.OBSERVATION-MODES — Observation modes

Deployment profiles MUST support implementation-aware and black-box evidence contracts without changing the semantic properties under evaluation.

### PROFILE.ATTESTED-IDENTITY — Attested identity

External Deployment profiles MUST provide an attributable build, configuration, isolation, authority, and evidence identity suitable for qualification.

### PROFILE.SECRET-EXCLUSION — Secret exclusion

A Deployment profile MUST describe authentication requirements without embedding authentication secrets in portable artifacts.

## QUALIFICATION — Deployment qualification

### QUALIFICATION.EXTERNAL-EXECUTION — External execution

Deployment qualification MUST be based on Execution evidence produced by the target external profile rather than by a synthetic substitute.

### QUALIFICATION.EXACT-BINDING — Exact binding

A Qualification receipt MUST bind the candidate release, Experiment identity, exact Result bytes, evidence digest, build identity, configuration identity, and Deployment profile.

### QUALIFICATION.SIGNED-AUTHORITY — Signed authority

A Qualification receipt MUST be authenticated by the authority assigned to the target Deployment profile.

### QUALIFICATION.COMPLETE-EVIDENCE — Complete evidence

Qualification MUST reject missing, duplicated, unsigned, mismatched, drifted, mixed-candidate, inconclusive, or incomplete-cleanup evidence.

### QUALIFICATION.COMMON-EXPERIMENT — Common experiment

Cross-profile qualification MUST compare executions of one content-identical Experiment.

### QUALIFICATION.PROMOTION — Qualification promotion

Umpire3 MUST promote a candidate release to qualified only after every required Deployment profile has supplied a valid receipt.

## CANARY — Production canary execution

### CANARY.SIGNED-APPROVAL — Signed approval

A production canary Execution MUST be authorized by a signed immutable approval bound to the Experiment, semantic catalog, Deployment profile, and recovery intent.

### CANARY.ALLOWLISTS — Allowlists

A canary approval MUST constrain permitted tenants, namespaces, resources, actions, faults, and destinations through explicit allowlists.

### CANARY.HARD-BUDGETS — Hard budgets

A canary approval MUST impose hard count, rate, concurrency, duration, evidence, output, and cleanup budgets.

### CANARY.WORKER-CONTAINMENT — Worker containment

Umpire3 MUST run canary work in a killable, resource-bounded worker with an explicit operating environment and no inherited authority beyond the approved scope.

### CANARY.RECOVERY-INTENT — Recovery intent

Umpire3 MUST durably record approved cleanup and recovery intent before starting canary work that can affect production resources.

### CANARY.CLEANUP-RESUMPTION — Cleanup resumption

Umpire3 MUST resume interrupted canary cleanup only under the same valid approval, authority, profile, and resource bindings while quarantining affected resources until cleanup is established.

## SECURITY — Security boundaries

### SECURITY.HOSTILE-INPUT — Hostile input

Umpire3 MUST treat Experiments, descriptors, participant programs, corpora, worker output, Replay bundles, approvals, and receipts as untrusted input.

### SECURITY.STRICT-DECODING — Strict decoding

Umpire3 MUST reject unknown fields, invalid structure, incompatible versions, and disallowed vocabulary at every portable artifact boundary.

### SECURITY.BOUNDED-ALLOCATION — Bounded allocation

Umpire3 MUST enforce input size, nesting, count, and work bounds before allocating or executing attacker-controlled content.

### SECURITY.SENSITIVE-DATA — Sensitive data

Umpire3 MUST reject typed sensitive fields and redact or digest sensitive values in diagnostics, evidence, replay, receipts, and published artifacts.

### SECURITY.CREDENTIAL-SEPARATION — Credential separation

Umpire3 MUST keep authentication material outside portable artifacts by obtaining it through an environment-owned channel.

### SECURITY.LEAST-AUTHORITY — Least authority

Umpire3 MUST select the least authoritative eligible Environment, with explicit opt-in for restricted faults or production effects.

## OPERATIONS — Operator workflows

### OPERATIONS.UNIFIED-SURFACE — Unified surface

Umpire3 MUST provide one supported operator surface for inspection, compilation, execution, exploration, replay, qualification, and release assurance with consistent diagnostics.

### OPERATIONS.ALLOCATION-FREE-INSPECTION — Allocation-free inspection

Operators MUST be able to inspect author intent, compiled paths, capability requirements, bounds, and claim scope without allocating an Environment.

### OPERATIONS.REMOTE-EXECUTION — Remote execution

Umpire3 MUST support secure remote Execution with unique resource scope, explicit authentication, Environment attestation, capability preflight, and retained Result evidence.

### OPERATIONS.FAIL-CLOSED — Operational stop conditions

Umpire3 MUST stop or classify the Execution as inconclusive on unsupported capability, evidence loss, identity ambiguity, contradiction, configuration drift, budget exhaustion, or cleanup uncertainty.

### OPERATIONS.CLOCK-SAFETY — Clock safety

Umpire3 MUST preserve safety claims when Environment clocks differ by using declared clock assumptions and evidence that does not require unjustified global ordering.

## RELEASE — Release assurance

### RELEASE.CANDIDATE-STATE — Candidate state

Umpire3 MUST distinguish a locally validated candidate release from a deployment-qualified release.

### RELEASE.ASSURANCE-GRAPH — Assurance graph

Umpire3 MUST evaluate release readiness as a dependency graph of semantic, generated, executable, benchmark, mutation, resilience, operational, documentation, and qualification evidence.

### RELEASE.RETAINED-EVIDENCE — Retained evidence

Release evidence MUST be attributable, content-bound, complete for the claimed scope, and retained with the candidate it evaluates.

### RELEASE.SELF-VALIDATION — Self-validation

Umpire3 MUST validate its own release graph, receipts, and required-evidence inventory before reporting release status.

### RELEASE.QUALIFIED-STATE — Qualified state

Umpire3 MUST report a release as qualified only when all required local evidence and external Qualification receipts are valid for the same candidate.

## MIGRATION — Prior-suite migration

### MIGRATION.INDEPENDENCE — Independent operation

Umpire3 MUST establish its claims independently of prior Umpire suites and their runtime artifacts.

### MIGRATION.BEHAVIOR-INVENTORY — Behavior inventory

Migration into Umpire3 MUST track each prior behavior by its semantic replacement, evidence class, and remaining gap rather than by file-for-file translation.

### MIGRATION.FIDELITY — Migration fidelity

A migrated behavior MUST retain its essential setup, threat, property, and evidence intent while adopting Umpire3 claim boundaries.

### MIGRATION.SIDE-BY-SIDE — Side-by-side evidence

Umpire3 MUST support comparing independent prior-suite and Umpire3 Results without treating agreement as deployment qualification.

## SUPPORT — Compatibility and support boundaries

### SUPPORT.STRICT-FORMATS — Strict formats

Umpire3 MUST strictly validate versioned Experiments, semantic manifests, Replay bundles, approvals, receipts, and release evidence.

### SUPPORT.COMPATIBLE-EVOLUTION — Compatible evolution

Umpire3 MUST preserve the meaning of supported artifact and semantic versions by assigning incompatible changes an explicit new version.

### SUPPORT.UNIFIED-ENTRY — Unified operator entry point

Umpire3 MUST expose supported operator workflows through the unified command surface.

### SUPPORT.EXPERIMENTAL-QUARANTINE — Experimental quarantine

Umpire3 MUST isolate experimental verification integrations from required product claims and release gates until they become supported behavior.

### SUPPORT.PRIOR-SUITE-OWNERSHIP — Prior-suite ownership

Prior Umpire suites MUST retain ownership of their existing behavior until migration evidence explicitly transfers that behavior to Umpire3.
