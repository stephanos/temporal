---
title: Next important specs in UMPIRE4_ORDER
date: "2026-09-04"
focus_hint: .plans/UMPIRE4_ORDER.md
volume: 23
survivor_count: 5
rejected_count: 18
rejection_rate: 0.78
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

## Previous survivors — 2026-09-03

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

## Previous rejected — 2026-09-03

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

## Extension — 2026-09-04

The hard-cut Case Runtime in fn-64 supersedes the earlier PortableTestPlan execution roadmap. This
extension ranks work against the new Case, Program, Contract, PreparedCase, Run, and Verdict
boundaries.

### Grounding snapshot

- The active queue is fn-64, fn-59, fn-60, and fn-62.
- Fn-61 and fn-63 remain open in Flow state even though `UMPIRE4_ORDER.md` marks them superseded.
- Fn-22, fn-26, fn-29, and fn-33 remain open but explicitly require replanning after fn-64.
- Fn-22, fn-29, and fn-33 still carry dependency edges to superseded fn-61.
- Fn-64 has ten unstarted tasks and deliberately excludes a replacement service, CLI, production
  canary controller, durable replay, generic fault framework, and extra SDK instructions.
- The vision still requires approachable authoring, deterministic regression execution,
  exploration, first-class faults, programmable workers, black-box execution, and clock-skew-safe
  distributed use.

## Survivors

### High leverage (1-3)

#### 1. Reconcile superseded specs and stale Flow dependencies
**Summary:** Close fn-61 and fn-63 as superseded and remove their stale edges from downstream specs.
**Leverage:** Small-diff lever because only Flow metadata and roadmap edges change; impact lands on the entire visible critical path.
**Size:** S
**Affected areas:** .flow/specs, .plans/UMPIRE4_ORDER.md
**Risk notes:** Status edits must preserve history and avoid implying successor work is complete.
**Persona:** senior-maintainer
**Next step:** perform as tracker reconciliation, not as a new spec

#### 2. Replan fn-33 around Case, Run, and Verdict
**Summary:** Replace PortableTestPlan campaign assumptions with PreparedCase execution and semantic coverage.
**Leverage:** Small-diff lever because one existing spec can be rewritten against fn-64's approved vocabulary; impact lands on the first exploration workflow and the project's primary unknown-bug goal.
**Size:** M
**Affected areas:** .flow/specs/fn-33-run-serial-bounded-semantic-exploration.md
**Risk notes:** Replan only after fn-64's preparation and execution APIs survive their early proof.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 3. Replan fn-22 around a Case-native violation lifecycle
**Summary:** Redesign replay, reduction, and promotion over Case, Run, and Verdict identities.
**Leverage:** Small-diff lever because the existing lifecycle and promotion contracts can be retained while replacing one obsolete artifact boundary; impact lands on deterministic regression reproduction and reviewed promotion.
**Size:** M
**Affected areas:** .flow/specs/fn-22-deterministic-replay-semantic.md
**Risk notes:** The retired caller-closure fixture leaves the concrete violated Case unresolved.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

### Worth considering (4-7)

#### 4. Replan fn-26 for Case-native qualification receipts
**Summary:** Bind local qualification and staged profiles to Case, PreparedCase, Run, and Verdict.
**Leverage:** Small-diff lever because one existing receipt boundary can be rebound after runtime identities settle; impact lands on local-to-CI-to-canary portability and later Claim Assessment.
**Size:** M
**Affected areas:** .flow/specs/fn-26-local-qualification-receipts-and-staged.md
**Risk notes:** Provenance and claim scope are premature until Case identity and compiler metadata land.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 5. Make Contract diagnostics explainable
**Summary:** Render transition paths, pending horizons, and supporting Run Events without changing Verdict semantics.
**Leverage:** Small-diff lever because rendering can sit on the isolated verification result boundary; impact lands on every failed, violated, or pending Case diagnosis.
**Size:** M
**Affected areas:** tools/umpire/verification, tools/umpire/cmd
**Risk notes:** Design only after fn-64 reveals concrete diagnostic gaps; output formats can become accidental contracts.
**Persona:** first-time-user
**Next step:** /flow-next:interview

## Rejected

- Replan fn-29 for PreparedCase production canaries — duplicates-open-epic: Open fn-29 already owns this direction and should wait on fn-26 and fn-64.
- Prove one Lean-produced violated Case end to end — duplicates-open-epic: Violation behavior is core acceptance evidence for fn-64; if its integration scope intentionally remains success-only, select the concrete violation while replanning fn-22.
- Prove Case Runtime generality with a non-Nexus behavior — insufficient-signal: No second Temporal domain or concrete behavior establishes the generality gap.
- Add first-class bounded fault injection to Cases — insufficient-signal: No retained concrete Case currently requires runtime fault injection.
- Build a bounded programmable Temporal worker behavior library — duplicates-open-epic: Fn-64 already owns the first generic worker interpreter; broaden it only from a concrete later Case.
- Add a Case inspect, validate, and run CLI — duplicates-open-epic: Fn-64 defines preparation and diagnostics while fn-33 owns the first command workflow.
- Publish Host and Monitor conformance test kits — insufficient-signal: One first-party Host does not yet justify a supported conformance surface.
- Define Case provenance and model-bound claim scope — duplicates-open-epic: Fn-26 and fn-30 already own qualification and release claim boundaries.
- Add causal multi-source Observations resilient to clock skew — insufficient-signal: The initial single authoritative server-history source does not exercise this need.
- Add content-addressed Run replay and audit digests — duplicates-open-epic: Fn-22 and fn-24 already own replay and receipt lineage; durability is explicitly deferred.
- Replace fn-63 with a post-cutover golden Case corpus cleanup — duplicates-open-epic: Resolve fn-63 as superseded first and reassess test shape after fn-64 lands.
- Add a remote black-box Temporal Host profile — duplicates-open-epic: Remote execution overlaps the retained production canary track in fn-29.
- Add Activity entrypoints and instructions — insufficient-signal: No concrete activity-backed Case justifies expanding the opcode surface.
- Add a reusable Case catalog and bounded batch runner — duplicates-open-epic: Case selection and bounded batches overlap fn-33's exploration campaign.
- Define Case schema evolution and migration policy — insufficient-signal: Designing migration before real v1 usage would freeze mistakes prematurely.
- Add adaptive prioritized exploration and corpus retention — duplicates-open-epic: It extends fn-33 before the serial bounded baseline exists.
- Add durable canary fleet orchestration and recovery — out-of-scope: Fleet scheduling, leases, and recovery are explicitly outside the runtime and current bounded canary scope.
- Add external Producer conformance fixtures — insufficient-signal: No external Producer exists to ground the portability contract.
