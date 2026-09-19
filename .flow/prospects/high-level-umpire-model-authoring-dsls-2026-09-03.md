---
title: High-level Umpire model authoring DSLs
date: "2026-09-03"
focus_hint: High-level Umpire model authoring DSLs
volume: 20
survivor_count: 8
rejected_count: 12
rejection_rate: 0.6
artifact_id: high-level-umpire-model-authoring-dsls-2026-09-03
promoted_ideas: []
status: active
---

## Focus

High-level Umpire model authoring DSLs

## Grounding snapshot

focus_hint: Simplify model/ into high-level developer DSLs that hide deeper Lean mechanics inside Umpire, informed by fn-51, fn-52, fn-33, and fn-22
focus_kind: concept

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

open_specs: 12
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

changelog_recent: scanned: none (no CHANGELOG.md)
memory_matches: scanned: none (memory not initialised)
memory_audit_stale: scanned: none (audit not run)
strategy: scanned: none (no STRATEGY.md signal)

## Survivors

### High leverage (1-3)

#### 1. Compile-time checked declaration commands
**Summary:** Add thin commands that define checked Property, Behavior, Query, Observation, and Link values with located errors.
**Leverage:** Small-diff lever because each command delegates to one existing typed checker; impact lands on every ordinary checked model declaration and its diagnostics.
**Size:** M
**Affected areas:** model/Umpire/Property, model/Umpire/Behavior, model/Umpire/Query, model/Umpire/Observation
**Risk notes:** Custom elaboration can obscure trust and make diagnostics harder if it does more than invoke existing checkers.
**Persona:** first-time-user
**Next step:** /flow-next:interview

#### 2. Typed model vocabulary handles
**Summary:** Let one kind-indexed definition own its ID, source, metadata, meaning, values, and exact patterns.
**Leverage:** Small-diff lever because one additive handle can project existing records unchanged; impact lands on Target vocabularies, Property patterns, and repeated Nexus identities.
**Size:** L
**Affected areas:** model/Umpire/Core.lean, model/Umpire/Target, model/Temporal/Feature/Nexus
**Risk notes:** Type-indexing may complicate inference or accidentally create a second semantic representation.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

#### 3. Purpose-named Query constructors
**Summary:** Add verify, witness, counterexample, and select constructors that own ordinary target, limits, and policy wiring.
**Leverage:** Small-diff lever because the constructors wrap the existing QueryDeclaration and QueryForm values; impact lands on every ordinary Query call site and future exploration inputs.
**Size:** S
**Affected areas:** model/Umpire/Query/Language.lean, model/Temporal/Feature/Nexus
**Risk notes:** Defaults may silently change identity-bearing policy or Limits if not entirely explicit.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

### Worth considering (4-7)

#### 4. Action-centered Property clause constructors
**Summary:** State one selected action once, then attach expected state, outcome, observation, and bounded facts.
**Leverage:** Small-diff lever because focused constructors return existing PropertyClause values; impact lands on repeated action-pattern wiring in ordinary Nexus Properties.
**Size:** M
**Affected areas:** model/Umpire/Property/Language.lean, model/Temporal/Feature/Nexus/Operations
**Risk notes:** Grouping clauses must still emit independent stable clause IDs and preserve canonical order.
**Persona:** first-time-user
**Next step:** /flow-next:interview

#### 5. Behavior occurrence sequence builders
**Summary:** Build named occurrences, exact bounds, ordering, and actionsExactly from one explicit action sequence.
**Leverage:** Small-diff lever because sequence projections can return the existing Behavior fields; impact lands on exact multi-action scenarios and Variation Space models.
**Size:** M
**Affected areas:** model/Umpire/Behavior/Language.lean, model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean
**Risk notes:** Automatic derivation could hide occurrence identity or make malformed declarations unrepresentable.
**Persona:** first-time-user
**Next step:** /flow-next:interview

#### 6. Scoped Definition ID authoring
**Summary:** Replace repeated long prefixes with a checked namespace scope that expands explicit local Definition ID suffixes.
**Leverage:** Small-diff lever because scope expansion still produces ordinary DefinitionId values; impact lands on nearly every model vocabulary and diagnostic path.
**Size:** M
**Affected areas:** model/Umpire/Core.lean, model/Temporal/Feature/Nexus, model/Temporal/System/Nexus
**Risk notes:** Source-level shorthand must preserve exact IDs and make the fully qualified value obvious in diagnostics.
**Persona:** first-time-user
**Next step:** /flow-next:interview

#### 7. Source-aware declaration constructors
**Summary:** Capture stable source data at declaration sites while keeping semantic source identity explicit and deterministic.
**Leverage:** Small-diff lever because Target already has compiler-occurrence capture machinery; impact lands on diagnostics for Property, Behavior, Query, Observation, and Link declarations.
**Size:** M
**Affected areas:** model/Umpire/Core.lean, model/Umpire/Target/Language.lean, model/Temporal
**Risk notes:** Compiler locations must not enter fingerprints or make checked artifacts change after code motion.
**Persona:** first-time-user
**Next step:** /flow-next:interview

### If you have the time (8+)

#### 8. Observation mapping assembly API
**Summary:** Assemble profiles, typed field handles, rules, ordering, closures, and dispositions through a cohesive mapping builder.
**Leverage:** Small-diff lever because the builder can project the existing mapping declaration; impact lands on the largest remaining System Nexus authoring modules.
**Size:** L
**Affected areas:** model/Umpire/Observation, model/Temporal/System/Nexus/Observation.lean
**Risk notes:** A fluent builder can conceal missing lists or introduce a parallel Observation language.
**Persona:** senior-maintainer
**Next step:** /flow-next:interview

## Rejected

- Checked model bundle composer — insufficient-signal: The grounding shows no composition failure beyond existing checked facades and local helpers that justifies a cross-language bundle with an AUT-07 boundary risk.
- Table-driven finite target constructor — other: Completed FiniteMachine work and AUT-08 deliberately require independent ordered domains plus closure and executable-action evidence, which transition-row derivation would reopen.
- Implementation Link correspondence table — insufficient-signal: Completed fn-43 and fn-51 already centralize forward simulation and explicit mapping-pair construction, and the snapshot supplies no fresh duplication requiring another table abstraction.
- Matrix-oriented Space authoring — insufficient-signal: One concrete two-by-two Nexus Variation Space after fn-51's leaf constructors does not justify an L-sized matrix representation that could become a second Space language.
- Exploration campaign model facade — duplicates-open-epic: The proposed facade materially overlaps fn-33's open bounded semantic-exploration orchestration across Space, policy, Limits, pinned tests, and session startup.
- Lean PortableTestPlan authoring builder — duplicates-open-epic: Fn-52 already owns the successor PortableTestPlan vocabulary, Lean lowering, execution program, and verification program.
- Replay reduction authoring DSL — duplicates-open-epic: Fn-22 already owns replay signatures, semantic reduction coordinates, and reviewed promotion rules.
- Generated Go authoring facade — out-of-scope: A Go model-authoring facade would violate SCP-03 and SEM-01 by creating a non-Lean behavior-authority path.
- General feature block macro — out-of-scope: A monolithic feature grammar conflicts with SEM-04 and AUT-07, which require separate Property, Behavior, Query, and Observation authoring languages.
- Generate behavior DSLs from protobuf — out-of-scope: Generating states, outcomes, or default Properties from protobuf descriptors violates SEM-03 because Generated Data cannot define product behavior.
- Public-facade import enforcement — insufficient-signal: MOD-11 already enforces architectural imports, and the grounding shows no concrete low-level-API misuse that warrants broader allowlist enforcement before the proposed facades exist.
- Readability budgets and canonical examples — insufficient-signal: Recent authoring specs already require canonical examples and documentation, while no measured readability failure supports adding line-count ceremony budgets now.
