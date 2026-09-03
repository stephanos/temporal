---
title: Open-ended prospect
date: "2026-09-03"
focus_hint: ""
volume: "20"
survivor_count: "5"
rejected_count: "15"
rejection_rate: "0.75"
artifact_id: open-ended-2026-09-03
promoted_ideas: [1, 2, 3, 4, 5]
promoted_to: {"1": [fn-54-decompose-the-observation-evaluator], "2": [fn-55-migrate-and-separate-the-local-temporal], "3": [fn-56-split-lean-api-generator-planning-from], "4": [fn-57-partition-the-observation-authoring], "5": [fn-58-partition-the-property-language]}
status: active
---

## Focus

_(open-ended; fresh artifact informed by the three active prospect artifacts)_

## Grounding snapshot

focus_hint: (none — open-ended)
focus_kind: open-ended
git_log_30d: 8297 files modified
  - .codex/agents/agents-md-scout.toml
  - .codex/agents/build-scout.toml
  - .codex/agents/docs-gap-scout.toml
  - .codex/agents/docs-scout.toml
  - .codex/agents/env-scout.toml
  - .codex/agents/flow-gap-analyst.toml
  - .codex/agents/github-scout.toml
  - .codex/agents/memory-scout.toml
  - .codex/agents/observability-scout.toml
  - .codex/agents/plan-sync.toml
code_hotspots:
  - model/Umpire/Observation/Evaluation.lean (2126 lines); tools/umpire/testplan/validate.go (1891); tools/umpire/internal/artifactv2/result.go (1870)
  - model/Temporal/Tool/RunEvaluation.lean (1438); model/Umpire/Property/Language.lean (1259); model/Umpire/ImplementationLink/Language.lean (1206)
  - tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog/project.go (1234); tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go (1219)
  - model/Umpire/Observation/Language.lean (1107); model/Temporal/System/Configuration/Core.lean (1100); tools/umpire/internal/runtimeengine/engine.go (917)
open_specs: 13
  - fn-14-milestone-a-pilot-baseline-and-lean: Milestone A pilot baseline and Lean-first usability decision
  - fn-15-standalone-api-and-config-input-catalogs: Standalone API and config input catalogs
  - fn-22-deterministic-replay-semantic: Deterministic replay, semantic minimization, and reviewed promotion
  - fn-23-veil-toolchain-compatibility-and: Veil toolchain compatibility and adoption gate
  - fn-24-lean-native-verification-receipts-and: Lean-native verification receipts and canonical replay
  - fn-25-optional-callerclosure-veil-binding-and: Optional CallerClosure Veil binding and canonical replay
  - fn-26-local-qualification-receipts-and-staged: Local qualification receipts and staged profile contract
  - fn-29-bounded-production-canary-execution-and: Bounded production canary execution and qualification
  - fn-30-release-evidence-graph-and-manual: Release evidence graph and manual authorization
  - fn-33-run-serial-bounded-semantic-exploration: Run serial bounded semantic exploration with umpire-fuzz
  - fn-46-export-lean-model-module-impact-index: Export Lean model module impact index
  - fn-52-caller-neutral-grpc-portable-test-plans: Caller-neutral gRPC portable test plans
  - fn-53-extract-local-isolation-collection: Extract local isolation collection state machine
changelog_recent: scanned: none (no CHANGELOG.md)
memory_matches: scanned: skipped (no concept focus)
memory_audit_stale: scanned: none (audit not run)
strategy: scanned: none (no STRATEGY.md signal)
prior_prospects: fresh artifact informed by active artifacts
  - High-level authoring DSLs: checked declarations, typed vocabulary, query/property/behavior builders, scoped IDs, source-aware constructors, Observation mapping assembly
  - UMPIRE4 order: close completed specs, validate dependency graph, finish fn-51, retain fn-52 ordering
  - Pre-planning research: tracker/order reconciliation, plan-authority index, model/test impact, rule mapping, artifact blast radius, stage crosswalk, trust baseline, ID and Known Gap audits
conversation_research:
  - fn-53 now owns local isolation collection extraction and must not be duplicated
  - Observation evaluator decomposition remains an unplanned stable-facade cleanup candidate
  - Artifact Evidence/Result validation split is useful only after fn-52 stops touching artifactv2
  - Local Temporal authority implementation extraction remains a lower-risk navigation/testability candidate

## Survivors

### High leverage (1-3)

#### 1. Decompose the Observation evaluator behind its facade
**Summary:** Split public contracts, structural analysis, raw evaluation, and accepted-trace admission without changing behavior.
**Leverage:** Small-diff lever because the stable Observation facade confines the split to one module family; impact lands on evaluator navigation, isolated testing, and diagnostic-preserving maintenance.
**Size:** L
**Affected areas:** model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation.lean
**Risk notes:** Diagnostic precedence and import direction must remain byte-for-byte compatible.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 2. Separate the local Temporal authority implementation
**Summary:** Move temporaltest server, client, and worker mechanics behind the existing private temporalAuthority seam.
**Leverage:** Small-diff lever because one concrete authority implementation already sits behind a private seam; impact lands on local Temporal lifecycle readability and fake-based testing.
**Size:** M
**Affected areas:** tools/umpire/temporal/local/environment.go, tools/umpire/temporal/local
**Risk notes:** A file move must not create a new adapter surface or alter resource ownership.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 3. Split Lean API generator planning from rendering
**Summary:** Isolate descriptor-to-plan normalization from deterministic Lean source emission.
**Leverage:** Small-diff lever because one generator hotspot already owns both normalization and rendering; impact lands on deterministic Lean API generation and focused tests.
**Size:** M
**Affected areas:** tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go, tools/umpire/cmd/umpire-gen-lean-api
**Risk notes:** Prior generator cleanup specs may already have chosen a different seam.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

### Worth considering (4-7)

#### 4. Partition the Observation authoring language
**Summary:** Split declaration data, source mapping checks, and ergonomic constructors behind Umpire.Observation.
**Leverage:** Small-diff lever because the existing Observation facade can preserve callers while internals move; impact lands on authoring-language clarity and constructor testing.
**Size:** L
**Affected areas:** model/Umpire/Observation/Language.lean, model/Umpire/Observation.lean
**Risk notes:** Must preserve the seams established by completed Observation cleanup specs.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 5. Partition the Property language implementation
**Summary:** Keep the public Property facade while separating declaration data, checking, and constructor helpers.
**Leverage:** Small-diff lever because the existing Property facade can remain stable across a physical split; impact lands on property authoring maintenance and import locality.
**Size:** L
**Affected areas:** model/Umpire/Property/Language.lean, model/Umpire/Property.lean
**Risk notes:** Physical splitting can accidentally expose internal constructors or create import cycles.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

### If you have the time (8+)

_(none)_

## Rejected

- Split artifact Evidence and Result validation after fn-52 — duplicates-open-epic: Fn-52 currently owns the same artifactv2 validation surface and must finish before this split is independently actionable.
- Decompose portable test-plan validation — duplicates-open-epic: Fn-52 owns portable test-plan contracts and directly overlaps the proposed validator decomposition.
- Decompose evaluation-contract validation — insufficient-signal: The snapshot provides no evidence that evaluation-contract validation is currently causing defects, maintenance friction, or blocked work.
- Extract the runtime engine phase ledger — duplicates-open-epic: Open qualification, canary, and release-evidence specs own the runtime stage and receipt sequencing this extraction would restructure.
- Split portable Observation evaluation mechanics — duplicates-open-epic: Fn-24 and fn-52 already own portable verification and plan-admission behavior within this evaluator surface.
- Split dynamic-config catalog project loading from emission — duplicates-open-epic: Fn-15 owns standalone dynamic-config catalog work adjacent to this generator's discovery and rendering pipeline.
- Separate Implementation Link language from application mechanics — duplicates-open-epic: Open replay, verification-receipt, and CallerClosure binding specs already constrain Implementation Link declaration and projection semantics.
- Split Temporal configuration core by ownership — duplicates-open-epic: Fn-15 owns adjacent configuration catalog contracts that this ownership split would reorganize.
- Decompose the experimental Nexus AutoClose model — insufficient-signal: The snapshot contains no hotspot, active requirement, or reported maintenance problem for the experimental AutoClose model.
- Consolidate portable-evaluation test fixtures — insufficient-signal: The snapshot contains no evidence that duplicated portable-evaluation fixtures materially impede maintenance or defect detection.
- Extract Nexus evidence test builders — insufficient-signal: The snapshot contains no evidence that Nexus Evidence setup repetition is a meaningful source of defects or contributor friction.
- Add an indexed Umpire command catalog — insufficient-signal: The snapshot identifies no contributor-navigation problem that justifies introducing another command catalog with ongoing drift risk.
- Retire the Umpire3 Makefile surface — backward-incompat: Removing legacy Make targets would break downstream scripts without any snapshot evidence that their consumers have migrated.
- Consolidate artifact JSON and set canonicalization — duplicates-open-epic: Fn-52 currently owns canonical artifact membership and admission rules across the proposed canonicalization surface.
- Add local receipt assertion helpers — duplicates-open-epic: Fn-53 already owns local isolation receipt coverage and should own any assertion helpers required by that work.
