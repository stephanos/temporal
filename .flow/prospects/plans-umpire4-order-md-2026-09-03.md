---
title: Next important specs in UMPIRE4_ORDER
date: "2026-09-03"
focus_hint: .plans/UMPIRE4_ORDER.md
volume: 20
survivor_count: 10
rejected_count: 10
rejection_rate: 0.5
artifact_id: plans-umpire4-order-md-2026-09-03
promoted_ideas: []
status: active
---

## Focus

.plans/UMPIRE4_ORDER.md

## Grounding snapshot

focus_hint: .plans/UMPIRE4_ORDER.md
focus_kind: path
focus_path: .plans/UMPIRE4_ORDER.md

git_log_30d: 8265 files modified
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

open_specs: 26
  - fn-5: Umpire discovery promotion and artifact
  - fn-14: Milestone A pilot baseline and Lean-first usability decision
  - fn-15: Standalone API and config input catalogs
  - fn-17: Bounded semantic exploration and coverage
  - fn-21: Nexus duplicate observation control
  - fn-22: Deterministic replay, semantic minimization, and reviewed promotion
  - fn-23: Veil toolchain compatibility and adoption gate
  - fn-24: Lean-native verification receipts and canonical replay
  - fn-25: Optional CallerClosure Veil binding and canonical replay
  - fn-26: Local qualification receipts and staged profile contract
  - fn-28: Portable evaluation contract and disposable-cluster qualification
  - fn-29: Bounded production canary execution and qualification
  - fn-30: Release evidence graph and manual authorization
  - fn-33: Run serial bounded semantic exploration with umpire-fuzz
  - fn-40: Centralize PlannerPolicy constructors and default seed
  - fn-42: Centralize configuration authoring with ConfigUseSpec
  - fn-43: Deepen ordinary Property, Behavior, and Query authoring
  - fn-44: Seal Observation traces and centralize semantic coordinates
  - fn-45: Index and reconcile Umpire plan authority
  - fn-46: Export Lean model module impact index
  - fn-47: Generate Umpire semantic outcome and Known Gap inventory
  - fn-48: Canonicalize Known Gaps as a checked set
  - fn-49: Centralize Observation field and structural contracts
  - fn-50: Migrate System CallerClosure to FiniteMachine
  - fn-51: Shorten ordinary model authoring
  - fn-52: Caller-neutral gRPC portable test plans

changelog_recent: scanned: none (no CHANGELOG.md)
memory_matches: scanned: skipped (no concept focus)
memory_audit_stale: scanned: none (audit not run)
strategy: scanned: none (no STRATEGY.md signal)

## Survivors

### High leverage (1-3)

#### 1. Finish fn-51 documentation and verification
**Summary:** Complete the sole in-progress task and close the nearly finished authoring-simplicity spec.
**Leverage:** Small-diff lever because only one documentation-and-verification task remains; impact lands on the ordinary authoring surface and the current in-progress queue.
**Size:** S
**Affected areas:** model/Umpire, .flow/specs/fn-51-shorten-ordinary-model-authoring.md
**Risk notes:** Aggregate verification may expose preserved-byte or documentation gaps.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 2. Close completed fn-40 PlannerPolicy work
**Summary:** Verify and close fn-40 so the P3 exploration runner no longer reports it as a blocker.
**Leverage:** Small-diff lever because all three implementation tasks are already done; impact lands on fn-33 readiness and the P3 critical path.
**Size:** S
**Affected areas:** .flow/specs/fn-40-centralize-plannerpolicy-constructors.md, model/Umpire/Query
**Risk notes:** Artifact drift checks may need regeneration before closure.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 3. Close completed fn-5 promotion work
**Summary:** Verify and close fn-5 so deterministic replay and reviewed promotion can start.
**Leverage:** Small-diff lever because all seven implementation tasks are already done; impact lands on fn-22 readiness and the duplicate-violation lifecycle.
**Size:** S
**Affected areas:** .flow/specs/fn-5-umpire-discovery-promotion-and-artifact.md, tools/umpire
**Risk notes:** Promotion-source sealing must still satisfy the current architecture rules.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

### Worth considering (4-7)

#### 4. Close completed fn-17 exploration foundation
**Summary:** Complete spec-level review for the bounded exploration foundation required by fn-33.
**Leverage:** Small-diff lever because all eight implementation tasks are already done; impact lands on the exploration runner and retained P3 sequencing.
**Size:** S
**Affected areas:** .flow/specs/fn-17-bounded-semantic-exploration-and.md, model/Umpire/Exploration
**Risk notes:** Flow state may reveal unrecorded completion evidence.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 5. Close completed fn-47 semantic inventory
**Summary:** Verify the semantic outcome inventory and remove its stale blocker edge from fn-33.
**Leverage:** Small-diff lever because all six inventory tasks are already done; impact lands on fn-33 dependency clearance and semantic drift confidence.
**Size:** S
**Affected areas:** .flow/specs/fn-47-generate-umpire-semantic-outcome-and.md, model/generated
**Risk notes:** Generated inventory drift could require a narrow follow-up.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 6. Close completed fn-21 duplicate control
**Summary:** Complete spec closure for the exact negative control consumed by fn-22.
**Leverage:** Small-diff lever because all seven negative-control tasks are already done; impact lands on fn-22 replay admission and the prototype violation path.
**Size:** S
**Affected areas:** .flow/specs/fn-21-nexus-duplicate-observation-control.md, model/Temporal/Feature/Nexus
**Risk notes:** The live proof may need rerunning under current fixtures.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 7. Validate the retained dependency graph after closures
**Summary:** Run full Flow-Next validation and confirm fn-33 and fn-22 depend only on retained work.
**Leverage:** Small-diff lever because it reuses the existing structural validator after status updates; impact lands on both prototype-gating P3 branches.
**Size:** S
**Affected areas:** .flow/specs, .flow/meta.json
**Risk notes:** Validation can identify structural errors but not missing architectural requirements.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

### If you have the time (8+)

#### 8. Close completed fn-43 authoring foundation
**Summary:** Reconcile fn-43's open status with its fully completed task set and fn-51 dependency claim.
**Leverage:** Small-diff lever because all seven foundation tasks are already done; impact lands on roadmap consistency and fn-51 provenance.
**Size:** S
**Affected areas:** .flow/specs/fn-43-deepen-ordinary-property-behavior-and.md, model/Umpire
**Risk notes:** Closure could uncover a stale cross-spec acceptance reference.
**Persona:** first-time-user
**Next step:** /flow-next:interview

#### 9. Close superseded fn-14
**Summary:** Apply the roadmap's explicit supersession decision so fn-14 stops appearing as open work.
**Leverage:** Small-diff lever because the supersession decision is already recorded; impact lands on backlog clarity and cold-session briefs.
**Size:** S
**Affected areas:** .flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md, .plans/UMPIRE4_ORDER.md
**Risk notes:** Closure must preserve its historical architecture record.
**Persona:** first-time-user
**Next step:** /flow-next:interview

#### 10. Keep fn-52 behind P3 completion
**Summary:** Treat caller-neutral gRPC portability as a successor track until fn-33 and fn-22 prove the prototype.
**Leverage:** Small-diff lever because the roadmap already marks fn-52 non-prototype-gating; impact lands on engineering focus across the remaining vertical slice.
**Size:** S
**Affected areas:** .plans/UMPIRE4_ORDER.md, .flow/specs/fn-52-caller-neutral-grpc-portable-test-plans.md
**Risk notes:** Delays a reusable protocol seam needed by later production canary work.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

## Rejected

- Close completed fn-28 portable evaluation work — insufficient-signal: Closing fn-28 only unlocks fn-52, which is non-prototype-gating and premature.
- Start fn-33 with the bounded Lean bridge — duplicates-open-epic: The proposed Lean bridge is already the subject of open fn-33 and remains blocked by unclosed prerequisites.
- Start fn-22 with strict replay admission — duplicates-open-epic: Strict replay admission is already tracked by open fn-22 and cannot start before fn-5 and fn-21 close.
- Start fn-52 with authority reconciliation — duplicates-open-epic: Authority reconciliation is already tracked by open fn-52 and would divert effort from prototype-gating P3 work.
- Reconcile roadmap completion wording with Flow-Next state — duplicates-open-epic: Reconciliation between roadmap authority and Flow-Next state materially overlaps open fn-45.
- Add a roadmap-to-Flow-state drift check — duplicates-open-epic: A bespoke roadmap drift checker duplicates the plan-authority indexing and reconciliation scope of open fn-45.
- Encode deferred prototype specs in tracker state — insufficient-signal: No canonical deferred tracker state is established, so changing dependencies or statuses could misrepresent the roadmap.
- Run one completion-review wave for all task-complete specs — other: Bundling unrelated completion reviews obscures per-spec failures and duplicates the more precise closure candidates.
- Choose fn-33 as the first prototype-gating P3 branch — insufficient-signal: The snapshot provides no evidence that fn-33 should precede fn-22 beyond a preference for earlier campaign visibility.
- Choose fn-22 as the first prototype-gating P3 branch — insufficient-signal: The snapshot provides no evidence that fn-22 should precede fn-33 beyond a preference for lifecycle completion.
