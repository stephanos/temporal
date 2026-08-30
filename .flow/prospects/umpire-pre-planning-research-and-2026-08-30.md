---
title: Umpire pre-planning research and indexing
date: "2026-08-30"
focus_hint: Umpire pre-planning research and indexing
volume: 24
survivor_count: 9
rejected_count: 15
rejection_rate: 0.62
artifact_id: umpire-pre-planning-research-and-2026-08-30
promoted_ideas: []
status: active
---

## Focus

Prepare bounded planning, research, and generated indexes that let spare AI-agent capacity reduce risk for current and future Umpire work without duplicating open specs.

## Grounding snapshot

focus_hint: pre-plan, research, or index work that can use spare AI-agent capacity around .plans/ and model/
focus_kind: concept

git_log_30d: 8122 files modified
top:
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

open_specs: 19
  - fn-5: Umpire discovery promotion and artifact evolution
  - fn-14: Milestone A pilot baseline and Lean-first usability decision
  - fn-15: Standalone API and config input catalogs
  - fn-17: Bounded semantic exploration and coverage
  - fn-20: Local execution semantic conformance
  - fn-21: Nexus duplicate observation control
  - fn-22: Deterministic replay, semantic minimization, and reviewed promotion
  - fn-23: Veil toolchain compatibility and adoption gate
  - fn-24: Lean-native verification receipts and canonical replay
  - fn-25: Optional CallerClosure Veil binding and canonical replay
  - fn-26: Local qualification receipts and staged profile contract
  - fn-27: Hermetic CI execution and qualification
  - fn-28: Authorized remote staging black-box execution and qualification
  - fn-29: Bounded production canary execution and qualification
  - fn-30: Release evidence graph and manual authorization
  - fn-33: Run resumable semantic exploration campaigns with umpire-fuzz
  - fn-40: Centralize PlannerPolicy constructors and default seed
  - fn-42: Centralize configuration authoring with ConfigUseSpec
  - fn-43: Deepen ordinary Property, Behavior, and Query authoring

changelog_recent: scanned: none (no CHANGELOG.md)
memory_matches: scanned: none (memory not initialised)
memory_audit_stale: scanned: none (audit not run)
strategy: scanned: none (no STRATEGY.md signal)

## Survivors

### High leverage (1-3)

#### 1. Reconcile tracker state with the prototype order
**Summary:** Classify open specs as retained, deferred, superseded, or newly added and repair roadmap drift before more dispatch.
**Leverage:** Small-diff lever because it reconciles existing tracker metadata with one ordering document; impact lands on every future Flow-Next dispatch.
**Size:** S
**Affected areas:** .plans/UMPIRE4_ORDER.md, .flow/specs
**Risk notes:** Status changes require human agreement because they affect work selection.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 2. Index plan authority and supersession
**Summary:** Create a compact map of which .plans documents are authoritative, historical, normative, or superseded.
**Leverage:** Small-diff lever because it adds one compact authority map over existing documents; impact lands on every agent reading .plans/.
**Size:** S
**Affected areas:** .plans, model/README.md, model/ARCHITECTURE.md
**Risk notes:** A wrong authority label can misroute future agents more than no index.
**Persona:** first-time-user
**Next step:** /flow-next:interview

#### 3. Export a model module and test-impact index
**Summary:** Reuse source inventory and import metadata to map each module to facades, reverse dependencies, and test roots.
**Leverage:** Small-diff lever because existing source-inventory and import-graph machinery supplies most inputs; impact lands on task scoping and focused test selection across the Lean model.
**Size:** M
**Affected areas:** model/Tools/LeanSourceInventory.lean, model/ModelLint, model/lakefile.toml
**Risk notes:** A checked-in snapshot would stale unless generation and verification ownership are explicit.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

### Worth considering (4-7)

#### 4. Map Umpire rules to specs, code, and tests
**Summary:** Trace every UMPIRE4 rule ID to owning specs, enforcement code, focused tests, and any uncovered obligation.
**Leverage:** Small-diff lever because rule IDs and owning paths already exist; impact lands on review coverage and unimplemented-obligation detection across Umpire work.
**Size:** M
**Affected areas:** .plans/UMPIRE4_SPEC.md, .flow/specs, model
**Risk notes:** Manual mappings can imply enforcement where only narrative coverage exists.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 5. Build an artifact blast-radius manifest
**Summary:** Map each canonical fixture and checksum to its Lean producer, Go reader, generated view, and refresh command.
**Leverage:** Small-diff lever because the fixtures, generators, and refresh commands already exist; impact lands on every identity-bearing artifact migration.
**Size:** M
**Affected areas:** model/Umpire/Artifact, model/Temporal/Feature/Nexus/Fixtures, tools/umpire, Makefile
**Risk notes:** The manifest must distinguish generated ownership from authored fixtures.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 6. Normalize the stage-outcome crosswalk
**Summary:** Document every stage status and forbidden implication across planning, execution, observation, links, properties, and results.
**Leverage:** Small-diff lever because it is a distinction-preserving view over existing enums; impact lands on Run Evaluation, qualification, replay, and review reasoning.
**Size:** S
**Affected areas:** model/Umpire/Observation, model/Umpire/ImplementationLink, model/Umpire/Artifact, model/Temporal/Tool/RunEvaluation.lean
**Risk notes:** A unified table must preserve distinctions rather than invent a shared enum.
**Persona:** first-time-user
**Next step:** /flow-next:interview

#### 7. Establish a proof-trust and axiom baseline
**Summary:** Record axiom inventories for public facades and load-bearing declarations before parallel Lean refactors begin.
**Leverage:** Small-diff lever because Lean already exposes transitive axiom inventories; impact lands on every queued refactor that uses proof-taking facades or native_decide.
**Size:** M
**Affected areas:** model/Umpire.lean, model/Temporal.lean, model/UmpireTests.lean, model/TemporalModelTests.lean
**Risk notes:** Compiler-version-specific native_decide behavior makes the baseline toolchain-bound.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

### If you have the time (8+)

#### 8. Audit Definition ID collisions and stability
**Summary:** Extract semantic IDs from checked declarations and artifacts, flag collisions, and baseline identity-bearing changes.
**Leverage:** Small-diff lever because checked metadata already carries the authoritative IDs; impact lands on catalog, replay, promotion, and artifact compatibility work.
**Size:** M
**Affected areas:** model/Umpire/Core.lean, model/Umpire/Target, model/Temporal/Tool/Inspect.lean, model fixtures
**Risk notes:** Regex extraction would be unsound; the audit must use checked model metadata.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 9. Inventory Known Gaps and unsupported paths by stage
**Summary:** Extract every declared or synthesized Known Gap and map where it propagates, blocks claims, or becomes unsupported.
**Leverage:** Small-diff lever because Known Gap types and propagation paths are already explicit; impact lands on honest results, replay, qualification, and Claim Assessment.
**Size:** M
**Affected areas:** model/Umpire/Planning/Types.lean, model/Umpire/Artifact, model/Umpire/Observation, model/Temporal/Tool/RunEvaluation.lean
**Risk notes:** The index must distinguish authored gaps from runtime diagnostics and test-only fixtures.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

## Rejected

- Benchmark build and test shards for agent dispatch — insufficient-signal: The snapshot provides no evidence that build latency or shard selection currently blocks agent dispatch, while the measurements would be machine-specific.
- Precompute the positive and negative-control mutation matrix — duplicates-open-epic: Defining mutation expectations for the normal and duplicate-observation slices is implementation planning already owned by fn-20 and fn-21.
- Research staging harness and profile readiness — duplicates-open-epic: Establishing staging authority, isolation, limits, and cleanup is core readiness work already assigned to fn-28.
- Research black-box Nexus evidence sufficiency — duplicates-open-epic: Determining sufficient black-box Nexus evidence materially overlaps the execution and qualification contract of fn-28.
- Build a semantic coverage-coordinate glossary — duplicates-open-epic: Selecting and defining semantic coverage coordinates is a central design responsibility of fn-17.
- Mine GOMAD exploration and reduction lessons for Umpire — duplicates-open-epic: Translating exploration, replay, coverage, and minimization lessons into Umpire decisions materially overlaps fn-17 and fn-22.
- Generate agent context packs per prototype track — insufficient-signal: No grounding evidence shows agents are blocked by missing context packs, and the proposed packs would duplicate 19 actively maintained specs.
- Plan a documentation link and path drift gate — insufficient-signal: The prospect snapshot does not establish enough current-path drift to justify a dedicated gate before classifying historical documents.
- Review deep-module hotspots before parallel refactors — duplicates-open-epic: Reviewing missing semantic APIs in Property and Behavior authoring materially overlaps the deepening work already tracked by fn-43.
- Catalog generator ownership and change impact — duplicates-open-epic: Cataloging generated API and configuration ownership is part of the standalone input catalogs already specified by fn-15.
- Threat-model canary and release authority now — duplicates-open-epic: Authorization, fencing, evidence retention, veto, revocation, and cleanup are substantive design obligations already owned by fn-29 and fn-30.
- Preflight the optional Veil toolchain matrix — duplicates-open-epic: Toolchain candidates, compatibility, supply-chain closure, and adoption costs are already the explicit scope of fn-23 and fn-25.
- Implement the generic semantic catalog — duplicates-open-epic: Implementing the broad semantic catalog duplicates fn-5 and disregards the prototype order that defers most of it.
- Implement PlannerPolicy constructors immediately — duplicates-open-epic: PlannerPolicy constructors are already fully tracked by fn-40 and are blocked on fn-17 strategy work.
- Implement ConfigUseSpec immediately — duplicates-open-epic: ConfigUseSpec implementation is exactly the existing work item fn-42 and should be executed rather than prospected again.
