# Agentworkflow Durable Run Module Design

## Goal

Deepen `internal/store` so Agentworkflow's durable Run interface owns persistence, integrity,
locking, attempt recovery, and publication ordering. Engine and Workflow should express run-domain
transitions without knowing manifest fields, artifact paths, checkpoint generations, raw JSON, or
recovery sequencing.

This is an internal architecture refactor. Public Engine behavior and the existing v1 read-only and
v2 read/write artifact contracts remain compatible.

## Scope

This change deepens only the durable Run module. It does not also deepen the candidate Workspace,
evidence qualification, project admission, or provider execution modules. Those remain independent
follow-up candidates.

The refactor includes:

- replacing storage-mechanical return values with run-domain projections;
- moving JSON encoding and strict decoding behind the Run interface;
- combining lock acquisition and attempt recovery into one operation;
- making complete attempt recording one operation so callers cannot omit finalization;
- removing Engine knowledge of manifest paths and generations;
- moving crash-window and publication tests to the Run interface; and
- retaining low-level integrity and filesystem tests inside `internal/store`.

## Non-goals

- No public `agentworkflow.Engine`, `Backend`, `Request`, `Result`, or CLI contract changes.
- No new run artifact schema or migration.
- No filesystem port or adapter; temporary directories already provide a local substitute.
- No generic stage interface or pluggable stage graph.
- No merge between Store and Workspace in this change.
- No façade that preserves the existing shallow storage protocol.

## Current friction

The Store implementation already earns its existence under the deletion test: removing it would
spread locking, hashing, atomic publication, corruption detection, recovery, and legacy inspection
through Engine. Its interface is nevertheless shallower than it needs to be.

Engine and Workflow currently know:

- `Manifest`, `Inspection`, and `AttemptManifest` persistence shapes;
- result and checkpoint artifact paths;
- checkpoint generation numbers;
- the `Acquire` then `RecoverAttempts` then `Inspect` ordering;
- raw JSON marshaling and strict decoding around store calls;
- `StartAttempt` then `Emit`/`SetSession` then `Finish` ordering; and
- which attempt fields are needed to reconcile a resumed mutation.

`Engine.readResult` bypasses `Run.ReadResult` and opens the result path from the manifest directly.
Apply backup naming reads the current manifest generation. Recovery tests construct persistence
state through `engine.store`, `workspace.Prepare`, `saveCheckpoint`, and `Recorder`, so the intended
Run interface is not their test surface.

## Selected design

### Store owns durable admission and inspection

`Store.Create` accepts a typed request value and publishes its v2 representation internally.
`Store.Inspect` returns a run-domain snapshot containing only state, phase, outcome, timestamps,
recoverability, and sanitized attempt summaries. Manifest paths, byte counts, digests, and
generations stay inside the implementation.

Store-level result loading accepts a destination value and handles v1 and v2 layout differences
internally. Engine does not join store roots, run identifiers, or manifest paths.

### Acquisition includes recovery

Acquiring an existing Run takes the evidence bounds and current time required for recovery. The
module acquires or recovers the lock, validates the run, reconciles abandoned attempts, and returns
the acquired Run plus its post-recovery snapshot as one operation.

There is no externally observable state in which Engine holds a Run but has not yet applied required
attempt recovery. Recovery stays idempotent so terminal commands such as `diff` and `apply` can use
the same acquisition path safely.

### Run owns typed transitions

Checkpoint and result operations accept typed values. The Run implementation owns JSON encoding,
strict decoding, size checks, integrity domains, immutable checkpoint generations, manifest updates,
and atomic publication.

Transition metadata is passed as a small run-domain value containing state, phase, and outcome. It
replaces independent string parameters and makes invalid publication combinations testable at the
Run seam.

Engine continues to own the checkpoint and public result domain types. Store treats their encoded
form as integrity-bound payloads and does not import the parent `agentworkflow` package, avoiding an
import cycle.

### Run owns complete attempt recording

Attempt recording becomes a scoped operation. The caller supplies the stage, bounds, and agent
execution callback. The callback can append normalized evidence and identify a resumable session;
it returns the terminal session, structured output, and failure. The Run module always closes and
publishes the attempt as completed, failed, or interrupted before returning.

The callback seam does not interpret provider events. Lifecycle validation remains in Workflow,
while persistence ordering, evidence limits, hashing, syncing, and terminal attempt publication stay
inside Run.

Completed-attempt recovery returns a sanitized summary and decodes structured output into a caller
destination. Workflow can reuse a completed mutation without seeing output artifact paths or raw
store manifests.

### Transitional workspace locality

`Run.Directory` remains temporarily because Workspace still creates base, candidate, review, and
backup material beneath the run directory. This is the only retained filesystem location in the Run
interface. The separate candidate Workspace deepening can remove it later without coupling both
refactors.

Engine stops reading `Store.Root`, `Run.Manifest`, result paths, and checkpoint generations. Apply
uses a stable backup stem beneath the Run directory; Workspace already allocates a unique directory
when that stem exists.

## Data flow

### New run

1. Engine normalizes the public request and backend identity.
2. Store creates and locks a Run from the typed request.
3. Run publishes typed checkpoint transitions.
4. Workflow records each provider attempt through the scoped attempt operation.
5. Run publishes the typed terminal result and releases the lock.

### Resume

1. Store inspects the Run without exposing persistence metadata.
2. Terminal Runs return their typed result immediately.
3. Store acquires a nonterminal Run and completes mandatory attempt recovery atomically from the
   caller's perspective.
4. Engine loads the typed checkpoint and reconciles mutation intent from sanitized attempt summaries.
5. Workflow continues from the durable transition.

### Inspect, diff, and apply

Inspection uses the run-domain snapshot and typed result reader. Diff and apply acquire a recovered
Run through the same path, load the typed checkpoint and result, and then delegate candidate behavior
to the existing Workspace module.

## Error handling

- Store capacity, corruption, and locking errors remain recognizable through existing root errors.
- Failed typed encoding leaves the current manifest unchanged.
- Failed checkpoint or result publication never exposes a manifest pointing at incomplete evidence.
- Attempt callbacks always produce terminal attempt evidence, including cancellation, timeout,
  output exhaustion, and infrastructure failure.
- Recovery validates the retained event prefix before publishing interrupted or completed evidence.
- v1 artifacts remain read-only; attempts to acquire a legacy Run remain unsupported.
- Closing errors continue to join the primary error.

## Testing strategy

The interface is the test surface for durable behavior.

New Run-interface tests cover:

- typed create, checkpoint transition, inspection, and terminal result loading;
- acquisition that returns only after abandoned-attempt recovery;
- completed-attempt reuse after the checkpoint publication crash window;
- scoped attempt recording on success, agent failure, cancellation, and capacity exhaustion;
- rejection of invalid transition combinations;
- v1 and v2 result loading without artifact-path knowledge; and
- unique apply backup allocation without manifest-generation knowledge.

Existing Store-internal tests retain direct filesystem access for corruption, truncation, digest,
atomic-write, stale-lock, symlink, and legacy-layout fixtures. Engine tests retain public behavior
coverage but stop constructing checkpoints or attempts through Store mechanics once equivalent
Run-interface coverage exists.

All tests use real temporary directories. No filesystem mock or port is added because one adapter
would create a hypothetical seam.

## Migration sequence

1. Add failing tests for the desired Run interface and recovery ordering.
2. Add run-domain projections and typed payload helpers inside `internal/store`.
3. Replace inspection and typed result reads in Engine.
4. Replace acquisition plus recovery sequencing.
5. Replace raw checkpoint publication and loading.
6. Replace exposed Recorder ordering with scoped attempt recording.
7. Remove obsolete exported persistence types and methods after all callers move.
8. Move reach-through recovery tests to the Run interface and delete superseded tests.
9. Run focused tests, the full race suite, vet, build, lint, and coverage reporting.

## Success criteria

- Engine and Workflow do not reference Store manifests, artifact paths, generations, raw checkpoint
  or result JSON, Recorder, or explicit attempt-recovery ordering.
- `internal/store` presents a smaller run-domain interface with more implementation behind it.
- Existing public behavior and v1/v2 artifact compatibility remain unchanged.
- Recovery and publication tests exercise the same Run seam used by Engine.
- The complete Agentworkflow test, race, vet, build, and lint gates pass.
