---
status: draft
---

# Plan: Deepen the Agentworkflow Durable Run Module

## Context

Agentworkflow's Store implementation owns locking, integrity, atomic persistence, crash recovery,
and legacy inspection, but Engine and Workflow still sequence those mechanics through manifests,
artifact paths, generations, raw JSON, recorder finalization, and explicit recovery calls. This
refactor makes the durable Run interface the test surface and concentrates the complete persistence
protocol inside `internal/store`.

The public Engine, Backend, Request, Result, CLI, and v1/v2 artifact contracts remain unchanged. The
candidate Workspace, qualification policy, project admission, and provider execution modules are
explicitly deferred.

## Pattern Survey

### Analogous Features

- `tools/gomad3/runner/internal/campaign/recovery.go:45` — `RecoverCampaign` acquires the recovery lock, re-inspects durable state under that lock, performs recovery, and returns domain-level before/after projections.
- `tools/gomad3/runner/internal/campaign/campaign_journal.go:333` — `CampaignJournal.Publish` owns lifecycle transition, encoding, atomic publication, syncing, and terminal state rather than exposing their ordering to callers.
- `tools/gomad3/runner/internal/campaign/campaign_journal.go:487` — `ExecutionJournal.Transition` validates legal state transitions at the durable module interface.
- `tools/gomad3/artifact/store.go:65` — `PublishArtifact` stages, validates, hashes, syncs, and atomically publishes an entire artifact as one operation.
- `tools/agentworkflow/internal/workspace/workspace.go:56` — `Reopen` returns a semantic `Prepared` projection after validating filesystem layout and admitted-source identity, keeping scanning mechanics behind the Workspace module.
- `tools/agentworkflow/internal/store/legacy.go:27` — `inspectLegacyRun` adapts v1 files into the current inspection model without mutating legacy artifacts.
- `tools/agentworkflow/internal/store/recovery.go:89` — `recoverAttempt` already concentrates event-prefix validation and completed-versus-interrupted reconciliation inside Store.
- `tools/agentworkflow/internal/store/store_test.go:334` — Store tests exercise the published-output crash window and recovered completed attempt using real durable files.
- `tools/agentworkflow/engine_test.go:614` — The corresponding Engine test verifies that a completed mutation attempt is reused when checkpoint publication was interrupted.

### Reusable Utilities

- `tools/agentworkflow/internal/store/store.go:592` — `verifyArtifact` — validates containment, declared size, bounded reads, and domain-separated integrity.
- `tools/agentworkflow/internal/store/store.go:613` — `atomicWrite` — publishes a synced temporary file by rename and syncs its parent directory.
- `tools/agentworkflow/internal/store/store.go:642` — `createExclusiveJSON` — provides exclusive, synced JSON publication for locks.
- `tools/agentworkflow/internal/store/store.go:675` — `readBounded` — enforces evidence capacity while joining read and close failures.
- `tools/agentworkflow/internal/store/store.go:694` — `strictDecode` — rejects unknown fields and trailing JSON data.
- `tools/agentworkflow/internal/store/store.go:728` — `digest` — provides the existing domain-separated SHA-256 artifact identity.
- `tools/agentworkflow/internal/store/recovery.go:141` — `validateEventPrefix` — validates bounded normalized lifecycle evidence before recovery publication.
- `tools/agentworkflow/internal/store/legacy.go:93` — `verifyLegacyArtifact` — preserves the distinct v1 integrity contract.
- `tools/agentworkflow/engine.go:553` — `mapStoreError` — preserves the public error contract while Store retains its internal sentinel errors.

### Convention Anchors

- Durable operation ownership: repository journal modules own validation, encoding, transition ordering, syncing, and publication behind one semantic operation (`tools/gomad3/runner/internal/campaign/campaign_journal.go:333`, `tools/gomad3/artifact/store.go:65`).
- Lock-and-reinspect recovery: recovery decisions are recomputed after lock acquisition, so callers never act on stale pre-lock state (`tools/gomad3/runner/internal/campaign/recovery.go:45`).
- Compatibility locality: old artifact schemas are isolated in a dedicated adapter file and projected into the current read model without migration (`tools/agentworkflow/internal/store/legacy.go:11`, `tools/agentworkflow/internal/store/legacy.go:27`).
- Typed outer seam, private persistence shapes: Workspace exposes `Prepared` and `Change` while keeping inventory entries and scanning mechanics private (`tools/agentworkflow/internal/workspace/workspace.go:24`, `tools/agentworkflow/internal/workspace/workspace.go:41`).
- Layered error handling: Store identifies capacity, corruption, and locking with sentinel errors; Engine maps them at the public seam (`tools/agentworkflow/internal/store/store.go:19`, `tools/agentworkflow/engine.go:553`).
- Real-filesystem test split: Store's same-package tests directly mutate files for corruption, truncation, stale-lock, and legacy fixtures, while Engine tests assert user-visible run behavior (`tools/agentworkflow/internal/store/store_test.go:58`, `tools/agentworkflow/internal/store/store_test.go:99`, `tools/agentworkflow/internal/store/store_test.go:262`, `tools/agentworkflow/engine_test.go:614`).
- Current shallow interface: Engine explicitly sequences `Inspect`, `Acquire`, `RecoverAttempts`, another `Inspect`, raw checkpoint decoding, and attempt reconciliation (`tools/agentworkflow/engine.go:99`, `tools/agentworkflow/engine.go:146`, `tools/agentworkflow/engine.go:169`).
- Current persistence leakage: Engine reads manifest paths from `Store.Root`, marshals checkpoint/result JSON, and drives `Recorder` finalization (`tools/agentworkflow/engine.go:324`, `tools/agentworkflow/engine.go:343`, `tools/agentworkflow/engine.go:355`, `tools/agentworkflow/workflow.go:788`).

### Proposed Alignment

Blend the existing Agentworkflow integrity and v1 adapter utilities with the deeper CampaignJournal pattern: Store should own recovery, typed publication, and complete attempt recording at its Run seam. This increases locality and leverage while preserving the established split between Run-interface behavior tests and Store-internal filesystem corruption tests.

## Implementation Steps

1. **Pin the desired Run interface through failing tests**
   - Add focused tests in `tools/agentworkflow/internal/store/store_test.go` for typed create, checkpoint transition, inspection, terminal result loading, and invalid transition rejection.
   - Add tests showing acquisition returns only after abandoned attempts are recovered and completed attempt output is reusable without exposing artifact paths.
   - Add tests for scoped attempt success, failure, interruption, capacity exhaustion, and unconditional terminal publication.
   - Name the production symbols that make each test fail before adding their implementation, and verify every red state independently.

2. **Introduce run-domain projections while retaining private persistence shapes**
   - In `tools/agentworkflow/internal/store/store.go`, add compact domain values for run position, inspection, attempt summary, recovery bounds, and attempt completion.
   - Make manifest and attempt-manifest structs implementation details; project only state, phase, outcome, timestamps, recoverability, stage, status, and session across the Run seam.
   - Validate legal state transitions at publication time, following `ExecutionJournal.Transition` in `tools/gomad3/runner/internal/campaign/campaign_journal.go`.
   - Keep `ErrCapacity`, `ErrCorrupt`, and `ErrLocked` as Store sentinels for `mapStoreError`.

3. **Move typed payload ownership into Store**
   - Change `Store.Create` to encode a typed request internally using the existing v2 request integrity domain.
   - Replace raw checkpoint methods with typed load and transition operations that own encoding, strict decoding, bounds, generation selection, atomic publication, and manifest updates.
   - Add Store-level typed result loading for both v1 and v2 layouts, and replace raw result publication with a terminal operation that owns encoding and state transition.
   - Reuse `atomicWrite`, `verifyArtifact`, `readBounded`, `strictDecode`, and `digest`; do not add another serialization or filesystem seam.

4. **Combine acquisition, reinspection, and attempt recovery**
   - Refactor `Store.Acquire`, `Run.RecoverAttempts`, and follow-up `Store.Inspect` into one acquisition operation taking recovery bounds and the current time.
   - Re-read and validate the manifest after lock acquisition before choosing recovery actions, mirroring `RecoverCampaign` in `tools/gomad3/runner/internal/campaign/recovery.go`.
   - Return the acquired Run with a post-recovery domain inspection so Engine cannot observe or act on the pre-recovery state.
   - Keep v1 runs read-only and reject acquisition without mutating their layout.

5. **Make attempt recording a scoped durable operation**
   - Replace exported `Recorder` construction and caller-driven `Finish` ordering with a Run operation that owns attempt creation, evidence bounds, syncing, terminal status, output publication, and cleanup.
   - Give its execution callback only the minimal evidence append and session-identification interface needed by `recordingSink` in `tools/agentworkflow/workflow.go`.
   - Keep provider lifecycle validation in Workflow and retain `validateEventPrefix` as the recovery validator.
   - Replace raw completed-attempt output reads with typed destination decoding and sanitized attempt summaries.

6. **Migrate Engine and Workflow to the deepened seam**
   - In `tools/agentworkflow/engine.go`, replace manifest inspection, raw result path reads, JSON checkpoint encoding, generation-based backup naming, and explicit recovery ordering with the new Store and Run operations.
   - Simplify `Engine.Resume` and remove `recoverRun` once acquisition returns recovered inspection and typed checkpoint state.
   - Update `Engine.Inspect`, `readResult`, `publishAndClose`, `saveCheckpoint`, `Diff`, and `Apply` without changing public outcomes or error mapping.
   - In `tools/agentworkflow/workflow.go`, migrate `invokeTyped`, `recoveredInvocation`, and `recordingSink` to scoped attempt recording while preserving backend lifecycle and capacity semantics.
   - Remove the unused run-directory argument from `checkpoint.result`; retain `Run.Directory` only for existing Workspace placement.

7. **Remove superseded persistence surface and reach-through tests**
   - Make or remove obsolete exported manifest, artifact-path, generation, raw-payload, recorder, and standalone recovery symbols after all production callers migrate.
   - Rewrite recovery tests in `tools/agentworkflow/engine_test.go` to use public Engine behavior or move durable mechanics to Store-interface tests.
   - Retain same-package direct-file tests for corruption, truncation, stale locks, digest validation, legacy fixtures, and atomic publication.
   - Verify no production reference remains to Store roots, manifests, result paths, raw checkpoint/result payloads, recorder finalization, or explicit recovery ordering.

8. **Polish the module and update architecture documentation**
   - Split `tools/agentworkflow/internal/store/store.go` only where file placement improves locality, keeping one Run interface rather than introducing sibling modules.
   - Update `tools/agentworkflow/README.md` only if internal development guidance needs to describe the deeper Run seam; leave user-facing commands unchanged.
   - Update `docs/superpowers/specs/2026-08-24-agentworkflow-durable-run-module-design.md` only for implementation discoveries that sharpen, rather than broaden, the approved contract.

## Verification

- `GOWORK=off go test -count=1 -tags test_dep ./internal/store` — all Run-interface, corruption, recovery, and compatibility tests pass.
- `GOWORK=off go test -count=1 -tags test_dep .` — Engine behavior, resume crash windows, inspection, diff, and apply remain unchanged.
- `GOWORK=off go test -count=1 -tags test_dep ./...` — every Agentworkflow package passes.
- `GOWORK=off go test -count=1 -tags test_dep -race ./...` — no races in attempt evidence, locking, progress, or concurrent review behavior.
- `GOWORK=off go vet -tags test_dep ./...` — static analysis passes.
- `GOWORK=off go build ./cmd/agentworkflow` — the CLI binary builds.
- `../../.bin/golangci-lint-v2.12.2 run --build-tags test_dep --timeout 10m --fix=false --config=../../.github/.golangci.yml --disable=staticcheck ./...` — the nested module reports zero lint findings.
- `make agentworkflow-check` from the Temporal root — the repository's Agentworkflow gate passes.
- `GOWORK=off go test -count=1 -tags test_dep -cover ./...` — coverage is reported for the refactored Store and root package, with no regression in the durable paths.
- Inspect a newly created v2 run and the existing v1 fixture through `Engine.Inspect`; neither caller reads an artifact path, and the v1 fixture remains unchanged.

## Context Files

- `docs/superpowers/specs/2026-08-24-agentworkflow-durable-run-module-design.md` — approved scope, compatibility contract, and selected deepening.
- `tools/agentworkflow/internal/store/store.go` — current Store and Run interface, persistence shapes, integrity helpers, and lock ownership.
- `tools/agentworkflow/internal/store/recovery.go` — existing attempt reconciliation and event-prefix validation.
- `tools/agentworkflow/internal/store/legacy.go` — v1 read-only compatibility adapter.
- `tools/agentworkflow/engine.go` — current Store protocol sequencing and public error projection.
- `tools/agentworkflow/workflow.go` — attempt recording, lifecycle validation, and completed-attempt reuse.
- `tools/agentworkflow/internal/store/store_test.go` — real-filesystem integrity and recovery fixtures.
- `tools/agentworkflow/engine_test.go` — public behavior and current crash-window reach-through tests.
- `tools/gomad3/runner/internal/campaign/recovery.go` — lock-and-reinspect recovery precedent.
- `tools/gomad3/runner/internal/campaign/campaign_journal.go` — deep durable transition and publication precedent.
