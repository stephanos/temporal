# Umpire3 walkthrough

Umpire3 checks whether selected Temporal behaviors match explicit semantic models. Lean owns the
meaning of those behaviors and their safety properties. Go turns an authored Scenario into a
versioned Experiment, runs it through an Environment, and qualifies the resulting evidence without
reimplementing the model's state machine.

This guide starts with a read-only inspection command, then walks through the normal development
loop. You do not need a Temporal cluster to begin.

## The five things that move through Umpire3

The shortest useful mental model is:

```text
Scenario  --compile-->  Experiment  --execute in an Environment-->  Result
                            |                                      |
                            +------------- Replay bundle <---------+
```

The names have precise meanings:

1. A **Scenario** is sparse author intent: the resources, behavior, ordering, and property a test
   cares about.
2. An **Experiment** is the validated, versioned, executable artifact produced from a Scenario. It
   includes completed dependencies, bounds, model identity, and evidence requirements.
3. An **Execution** is one Experiment run against one **Environment**. The Environment is the adapter
   that realizes actions, observes evidence, and cleans up.
4. A **Result** records the environment identity, realized actions, normalized evidence, cleanup,
   and semantic claim from that Execution.
5. A **Replay bundle** retains a redacted, digest-bound Experiment and Result with enough metadata to
   attempt the same Execution again.

A **Regression** is the test that compiles a Scenario and executes every resulting Experiment. A
**Deployment profile** describes the maximum authority and capabilities allowed for a deployment
kind; it is not the Environment or proof that a deployment was tested. See [CONTEXT.md](CONTEXT.md)
for the complete vocabulary.

### Know what a passing check claims

Umpire3 deliberately keeps different evidence boundaries separate:

| Evidence | What it establishes |
| --- | --- |
| Lean proof | The declared theorem holds for the modeled semantics and its stated assumptions. |
| Bounded exploration or an external checker | No counterexample was found, or a witness was found, within the recorded bounds and trust mode. |
| Go validation | Compilation, transport, adapters, evidence normalization, cleanup, and artifact handling behave as tested. |
| Real-cluster Execution | One checked Experiment produced the recorded evidence in the identified Environment. |
| Qualification receipt | An authorized deployment owner signed a complete, conforming Result for the exact candidate release, Experiment, and profile. |

None of these silently upgrades another. Missing identity, causality, ordering, capability, cleanup,
or source evidence remains unsupported or inconclusive. One implementation run is not a proof of
the whole implementation. [IMPLEMENTATION_VERIFICATION.md](IMPLEMENTATION_VERIFICATION.md) describes
the boundary in detail.

## Before you begin

Run commands below from the repository root. What you need depends on how far you go:

- The first `explain` exercise needs the repository's Go toolchain and dependencies. It does not
  connect to Temporal, allocate an Environment, or require credentials.
- Model generation and model checks need the pinned Lean toolchain. The setup lives in
  `tests/umpire3/model/mise.toml` and `tests/umpire3/model/lean-toolchain`.
- `make umpire3-integration` uses the repository's cluster-capable test harness. The `run`, `replay`,
  and `campaign` commands need an eligible Temporal deployment plus an isolated namespace and task
  queue.
- Production canaries and release qualification additionally need deployment-owner approval,
  attestation, least-authority credentials, and the relevant signing authority.

All Go commands and tests in this tree use the `test_dep` build tag. Integration tests add the
`integration` tag. The Make targets shown here supply those tags for you.

## 1. Inspect an Experiment without running it

Start with a checked Workflow Update Experiment already in the repository:

```sh
make umpire3-explain EXPERIMENT=tests/umpire3/testdata/update-lifecycle.json
```

The command strict-decodes the file and prints an `umpire3/diagnostic/v1` JSON document. A successful
response has `command: "explain"` and `status: "ok"`. Its `data` describes:

- `experimentDigest` and `experimentID`: the exact semantic artifact you inspected;
- `property` and `modelModules`: the claim and Lean sources to which it is bound;
- `requiredCapabilities`: what an Environment must provide before allocation is allowed;
- `resources`, `actions`, `faults`, `order`, and `checkpoints`: what will be realized and observed;
- `seed` and `bounds`: the deterministic choices and limits that constrain the run.

This is the safest way to answer “what would Umpire3 do?” for an existing Experiment. The equivalent
direct command is:

```sh
go run -tags test_dep ./tests/umpire3/cmd/umpire3 explain \
  -experiment tests/umpire3/testdata/update-lifecycle.json
```

## 2. Choose the right verification loop

Use the narrowest check that covers your change while iterating.

For one model family:

```sh
make umpire3-check-family FAMILY=nexus-cancellation
```

This reads the generated family dependency graph and selects the family's Lean modules, mutation
tests, exact views, proof and fixture drift checks, and the native, Lean-temporal, or Veil checks the
family actually owns. Veil runs only for a family that imports a Veil declaration. TLA+, TLC, and
Apalache are never part of this target.

To check that generated files still match their sources:

```sh
make umpire3-check-generated
```

Before handing off an Umpire3 change, run the complete local gate:

```sh
make umpire3-check
```

The complete gate checks generated artifacts, validates the retained 10-replica native benchmark,
runs a fresh benchmark without comparing nondeterministic timing values, checks the Lean model, and
tests every Umpire3 Go package. It is intentionally much broader than the family loop. Replace the
retained performance measurement with `make umpire3-record-native-benchmark` only when that is the
purpose of the change and the resulting evidence has been reviewed.

## 3. Write a regression

An ordinary regression says only:

- which user-visible resources exist;
- which behavior threatens the contract; and
- which modeled property must hold.

Cluster setup and credentials belong in an Environment factory. Here is the typed Nexus example:

```go
func TestNexusCancellationRetry(t *testing.T) {
	operation := nexus.Operation("operation")
	authored := nexus.Regression("nexus-cancellation", operation,
		scenario.OnePath(
			operation.CancelWithRetry(),
			operation.CancellationSafety(),
		),
	)

	umpire3test.RequireRegression(t, authored,
		umpire3test.WithEnvironment(newTestNexusFactory(t)))
}
```

Read it from the inside out:

1. `nexus.Operation("operation")` creates a domain handle. It hides protocol records and binds the
   operation identity learned during execution.
2. `operation.CancelWithRetry()` expands to the model-backed Nexus actions for the lifecycle under
   test.
3. `operation.CancellationSafety()` requires the generated
   `nexus.cancellation.won-excludes-success` property.
4. `scenario.OnePath(...)` asks for one intended ordering. The compiler adds catalog-declared
   dependencies and semantic edges rather than requiring the test to restate transition mechanics.
5. `nexus.Regression(...)` supplies the family target and resources, producing the sparse Scenario.
6. `RequireRegression` compiles all paths, rejects missing capabilities before Environment
   allocation, executes each Experiment, checks the expected claim, and requires cleanup.
7. `WithEnvironment(...)` is the only environment-specific part. Swap this factory to run the same
   Scenario against another eligible Environment.

The imports for those author-facing pieces are:

```go
import (
	"testing"

	"go.temporal.io/server/tests/umpire3/scenario"
	"go.temporal.io/server/tests/umpire3/scenario/nexus"
	"go.temporal.io/server/tests/umpire3/umpire3test"
)
```

`newTestNexusFactory` represents the Environment factory owned by the surrounding test harness; it
is intentionally not part of the Scenario.

### Add ordering only when the contract needs it

Use the structural vocabulary instead of sleeps or adapter-local state:

- `scenario.OnePath(...)` keeps one intended trace.
- `scenario.AllPaths(...)` executes every bounded valid path.
- `scenario.AnyOrder(...)` introduces bounded alternative orderings, usually inside `AllPaths`.
- `scenario.Before(a, b)` adds an author-required ordering edge.
- `scenario.During(fault, body)` scopes a generated fault around behavior.

Compilation fails rather than truncating silently when its path, action, state, memory, or time
limits are exhausted. To inspect an authored Scenario before providing an Environment, call
`umpire3test.Explain(authored, limits)`. It returns the completed paths, inferred semantic and causal
edges, identity plan, omissions, catalog digest, and enumeration bounds without allocation.

Regressions expect conformance by default. A deliberate negative-control adapter can use
`umpire3test.ExpectViolation()`. The Result must still contain a qualified violating observation and
the flagged outcome; missing evidence cannot make the negative control pass.

## 4. Read a failure and retain it

Read a regression failure from the top down:

1. A compilation category points back to the authored Scenario and usually identifies invalid or
   ambiguous intent.
2. `unsupported` means the selected Environment lacks a required capability. This is decided before
   allocation.
3. `unknown` or `conflict` means evidence, identity, or another observation obligation could not be
   resolved. It is not a property violation.
4. A violation names the path, checkpoint, and supporting facts that qualified the claim.
5. Cleanup is evaluated independently. A conforming property with incomplete cleanup is
   inconclusive.

To retain failures, give the test facade a file corpus:

```go
umpire3test.RequireRegression(t, authored,
	umpire3test.WithEnvironment(newTestNexusFactory(t)),
	umpire3test.WithCorpus(replay.NewFileCorpus(corpusDirectory)),
)
```

Choose `corpusDirectory` as a restricted artifact location appropriate for the test environment.
The saved Replay bundle is strict, size-bounded, redacted, and bound to the Experiment digest. It
contains the original seed and bounds but no API keys.

Replay the bundle against an equivalent least-authority Environment:

```sh
go run -tags test_dep ./tests/umpire3/cmd/umpire3 replay \
  -bundle <bundle.json> -address <host:port> -namespace <namespace> \
  -task-queue <isolated-queue> -build-id <build>
```

Replay does not reduce every difference to “the test changed.” It classifies:

- **semantic drift**: the Experiment digest changed;
- **realization drift**: the Environment, capabilities, or realized actions changed;
- **schedule drift**: source ordering changed;
- **observation drift**: normalized checkpoint values changed;
- **evidence drift**: evidence availability, sources, references, or counts changed; and
- **footprint drift**: the learned runtime fault footprint changed.

## 5. Change a model

For model work, keep this ownership rule in view: Lean owns semantics; Go consumes generated
semantic artifacts and handles execution mechanics.

A typical change follows this sequence:

1. Put user-visible state and contracts in `model/Temporal/Product`. Feature-scale compositions live
   in `model/Temporal/Feature`; follow the neighboring family when deciding whether both are needed.
2. Add the smallest distributed-system mechanism that threatens the contract under
   `model/Temporal/System`, with explicit assumptions and finite domains.
3. Connect system and product behavior under `model/Temporal/Refinement`.
4. When the behavior is executable, prove the executable transition view equivalent to the
   relational `Step` rather than creating another semantics in Go.
5. Add permitted examples and a negative mutation that fails a theorem or yields a counterexample.
6. Declare dependencies, projections, properties, monitors, capabilities, and targets in the Lean
   catalog and composition sources.
7. If live observations decide the property, add the observation program and its evidence mapping.

For a protobuf-backed behavior, begin at `model/Temporal/API/selection.json`. Selection imports the
transitive descriptor closure. Every selected field needs an explicit semantic, opaque, ignored, or
rejected disposition plus an interpretation where applicable. Never copy generated protobuf
messages or field numbers into Lean or maintain a parallel Go allowlist.

After an intentional Lean semantic-source, descriptor-selection, or root-test inventory change:

```sh
make umpire3-gen
```

Generation updates the semantic catalog, typed Go identifiers, regression facade, Experiment
schema, proof manifests, checked Experiments, observation and monitor programs, composition and
coverage data, family dependency graph, selected API projection, migration ledger, and candidate
release bindings. These files are semantic review surfaces: review their diffs and do not hand-edit
them.

Then run the affected family check, the generated drift check, and finally the complete gate:

```sh
make umpire3-check-family FAMILY=nexus-cancellation
make umpire3-check-generated
make umpire3-check
```

Replace `nexus-cancellation` with the affected catalog target.

See [MODELING.md](MODELING.md) for the model boundary and [GENERATION_GATE.md](GENERATION_GATE.md) for
the generated-artifact contract.

## 6. Run against Temporal

The earlier exercises do not require a cluster. The workflows in this section do.

For the repository's real-cluster Nexus and Workflow Task acknowledgement integration tests:

```sh
make umpire3-integration
```

The target creates its test Environment through the repository harness; it does not require the
remote connection flags used below.

To execute a checked Experiment against an eligible deployment, use the unified command:

Replace the angle-bracket placeholders in the remaining command examples with deployment-owned
values and artifact paths.

```sh
go run -tags test_dep ./tests/umpire3/cmd/umpire3 run \
  -experiment <experiment.json> -address <host:port> -namespace <namespace> \
  -task-queue <isolated-queue> -build-id <build> -profile remote-deployment \
  -output <result.json> -bundle-output <bundle.json>
```

Use a unique task queue, attest the build and configuration, and choose the least-authoritative
eligible Deployment profile. Supply Nexus endpoint, service, and operation together or omit all
three. If authentication is required, set `UMPIRE3_TEMPORAL_API_KEY` in the process environment; the
CLI does not accept credentials in semantic input, Result, or Replay bundle files.

The generic Temporal Environment advertises no fault authority. A faulted Experiment is unsupported
unless the Environment provides a deployment-owned, scoped fault realizer with positive occurrence
and cleanup evidence.

### Explore with a bounded campaign

`campaign` deterministically mutates typed values, schedules, fault occurrence and scope, and
bounded topology, then runs selected candidates through the same Environment adapter:

```sh
go run -tags test_dep ./tests/umpire3/cmd/umpire3 campaign \
  -experiment <seed-experiment.json> -seed 1 -max-candidates 16 \
  -address <host:port> -namespace <namespace> \
  -task-queue <isolated-queue> -build-id <build> -profile remote-deployment
```

Campaign budgets and omissions remain explicit. A promoted result must minimize and replay while
preserving the exact qualified violation, then compile as an ordinary regression without
campaign-only support. `make umpire3-mutation-gate` checks the retained approved mutation audit. The
unified CLI also has an `audit-mutation` subcommand used by that gate; it is not the normal author or
operator entry point.

## 7. Qualify and promote a release

Execution and release qualification are different operations. The checked
`testdata/umpire3-1.2.json` manifest is a candidate until deployment owners supply all required
external evidence for these profiles:

- `local-in-process`;
- `ci-test-cluster`;
- `remote-deployment`;
- `grpc-only-black-box`; and
- `production-canary`.

Each owner runs the unchanged released Experiment in the named deployment, retains the complete
Result and attestation, and signs a qualification receipt with the reviewed Ed25519 authority for
that profile:

```sh
go run -tags test_dep ./tests/umpire3/cmd/umpire3 qualify \
  -release tests/umpire3/testdata/umpire3-1.2.json \
  -experiment <experiment.json> -result <result.json> \
  -profile <external-profile> -signing-key <authority.pem> \
  -output <receipt.json>
```

Keep the PKCS#8 private key outside the repository and pass only its path inside the authorized job.
Qualification requires exact release and Experiment binding, a conforming claim, complete evidence
and cleanup, profile-appropriate attestation, and complete fault realization when faults are
declared.

After every required profile has produced one valid receipt for the same Experiment digest:

```sh
go run -tags test_dep ./tests/umpire3/cmd/umpire3 promote \
  -release tests/umpire3/testdata/umpire3-1.2.json \
  -receipt <local-receipt.json> -receipt <ci-receipt.json> \
  -receipt <remote-receipt.json> -receipt <grpc-receipt.json> \
  -receipt <canary-receipt.json> -output <qualified-release.json>
```

Promotion fails closed on missing, duplicate, unsigned, wrong-authority, drifted, or mixed-Experiment
receipts. Production canaries add a signed immutable approval, hard resource budgets, allowlists,
durable recovery intent, a killable worker boundary, and independently bounded cleanup. Follow
[OPERATIONS.md](OPERATIONS.md), [SECURITY.md](SECURITY.md), and
[INCIDENT_RECOVERY.md](INCIDENT_RECOVERY.md) before running either workflow.

The supported operator surface is the unified `cmd/umpire3` command with `explain`, `run`, `replay`,
`campaign`, `qualify`, and `promote`. It emits `umpire3/diagnostic/v1` JSON for every subcommand.
Single-purpose commands under `cmd/` remain compatibility or build-pipeline entry points.

## Other gates and experimental work

`make umpire3-root` runs the retained Umpire2 root tests and the separately implemented Umpire3
copies. The Umpire3 copies use generated participant programs, public Temporal behavior and history,
and scoped fault realization; adapter unit tests remain negative controls rather than deployment
qualification.

`make umpire3-resilience-gate` verifies the retained control-plane audit and exercises hostile
decoding, resource and artifact limits, secret redaction, transactional publication, child-process
termination, and resumable canary cleanup.

The retained TLA/TLC/Apalache prototype is non-qualifying experimental work. Its Go packages require
the explicit `umpire3_tla_experiment` build tag and are absent from normal discovery, CI, release,
generation, family checks, and the complete gate. The manual `umpire3-experimental-tla-*` Make
targets are only for developers who already provide those external tools; Umpire3 does not install
or maintain them.

## Where to go next

- [CONTEXT.md](CONTEXT.md): canonical terms and words to avoid.
- [AUTHORING.md](AUTHORING.md): compact regression-authoring and failure-diagnosis rules.
- [MODELING.md](MODELING.md): the Lean/Go ownership boundary and model workflow.
- [GENERATION_GATE.md](GENERATION_GATE.md): generated source ownership and review expectations.
- [IMPLEMENTATION_VERIFICATION.md](IMPLEMENTATION_VERIFICATION.md): what each evidence class does and
  does not establish.
- [OPERATIONS.md](OPERATIONS.md): deployment execution, qualification, promotion, and authority.
- [SECURITY.md](SECURITY.md): hostile-input, redaction, credential, canary, and signing boundaries.
- [INCIDENT_RECOVERY.md](INCIDENT_RECOVERY.md): production-canary cleanup after interruption.
- [SUPPORT.md](SUPPORT.md): supported formats, profiles, command surfaces, and compatibility policy.
- [model-checkers/veil/README.md](model-checkers/veil/README.md): Veil setup, commands, evidence, and
  trust modes.

## Package and folder tour

All paths in this table are relative to `tests/umpire3`. Generated files are derived and must not be
edited by hand. Retained result files are checked evidence and should change only through their
recording targets. `.lake/` directories are ignored local build caches.

| Package or folder | What it is for |
| --- | --- |
| Root Markdown files and `documents.go` | The published vocabulary, authoring, modeling, verification, operations, security, recovery, support, and generation guides plus the digest-based documentation audit. |
| `campaign/` | Discovers, ranks, executes, minimizes, replays, and promotes bounded semantic mutations. `campaign/results/` contains the retained cross-layer mutation audit. |
| `canary/` | Runs approved Experiments inside the durable, killable production safety envelope, including signed approval and recovery state. |
| `clockskew/` | Audits how qualification receipts and canary approval/recovery behave across wall-clock skew. |
| `cmd/` | Executable entry points. `cmd/umpire3` is the supported unified CLI; the other commands serve generation, checking, canary workers, participants, and compatibility workflows. |
| `developerux/` | Audits the supported commands, examples, generated facade, package documentation, and failure guidance as a developer-facing surface. |
| `evidence/` | Builds bounded causal graphs from typed facts and action receipts, rejecting ambiguous identity, lineage, ordering, and contradictions. |
| `execution/` | Executes validated Experiments through the Environment seam and produces Results, including assurance and cleanup status. |
| `explore/` | Performs bounded deterministic discovery from compiled Experiments and promotes only replayable catalog-vocabulary paths. |
| `familycheck/` | Resolves and runs the generated transitive checker portfolio for `make umpire3-check-family`. |
| `fault/` | Defines scoped faults, their deployment-owned realization interface, and reconciliation of declared versus observed causal footprints. |
| `internal/` | Non-public implementation details: `artifact/` publishes files transactionally and `command/` implements unified CLI dispatch and compatibility entry points. |
| `migration/` | Generates and validates the typed Umpire2-to-Umpire3 behavior-fidelity ledger. `migration/ledger.json` is retained release evidence. |
| `model/` | The primary Lean project. `Umpire3/` is the reusable modeling/exploration framework, `Temporal/` contains product, feature, system, refinement, observation, target, experiment, and API models, and `Umpire3Tests/` contains Lean tests. `Temporal/API/Generated/` is generated; `model/.lake/` is an ignored local cache. |
| `model-checkers/` | Checker adapters and evidence: `canonical/` replays canonical Lean traces, `native/` runs the finite native checker and retains scale evidence, `lean-temporal/` handles temporal/lasso checking, `veil/` binds and normalizes Veil, and quarantined `tla/` contains the experimental TLA/TLC/Apalache path. `results/` and `bindings/` are retained; nested `.lake/` is ignored. |
| `mutationaudit/` | Validates semantic mutations across proof and checker layers. `mutationaudit/results/` is retained audit evidence. |
| `observation/` | Defines typed raw facts and evaluates generated observation programs. `observation/generated/` is generated from Lean declarations. |
| `participant/` | Validates and executes concrete Temporal command programs that realize semantic actions without deciding modeled properties. |
| `process/` | Runs bounded child processes with explicit environments, process-group termination, output limits, and distinct failure classes. |
| `profile/` | Defines Deployment profile authority and binds it to the capabilities and identity realized by an Environment without broadening either. |
| `protocol/` | Owns strict, versioned wire documents and generated semantic identifiers. `protocol/generated/` holds the catalog, schemas, manifests, views, dependency graph, and release evidence generated from semantic sources. |
| `qualification/` | Validates external execution evidence, signs profile-specific receipts, and promotes a candidate only after every required receipt passes. |
| `release/` | Builds and validates the release assurance graph against current generated artifacts, audits, documentation, and qualification state. |
| `replay/` | Encodes, stores, strict-decodes, reproduces, and compares redacted digest-bound Replay bundles. |
| `resilience/` | Runs and records hostile-input, isolation, resource-bound, publication, termination, and recovery checks. `resilience/results/` is retained audit evidence. |
| `scenario/` | The authoring language and deterministic compiler. `catalog.gen.go` is the generated typed facade; `scenario/nexus/` and `scenario/workflow/` are higher-level domain handles. |
| `temporal/` | Adapts Temporal clients, SDK workers, participant programs, public history, and evidence to the Environment seam. `temporal/internalhistory/` isolates explicitly privileged server-history corroboration. |
| `testdata/` | Checked fixtures: released Experiments, the candidate release manifest, and the empty manifest used by generation tests. |
| `umpire3test/` | The test-facing facade containing `RequireRegression`, `Explain`, Environment/corpus/limit options, and deliberate negative-control configuration. |
| `wirecase/` | Exercises selected protobuf fields through typed bounded transport fixtures so descriptor presence cannot be mistaken for semantic interpretation. |
