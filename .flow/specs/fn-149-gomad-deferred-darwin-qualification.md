# Gomad deferred Darwin qualification

## Goal & Context

The owner approved deferring the remaining native Darwin gates so the six source specs can finish their retained source acceptance independently of native hardware. This spec owns the outstanding darwin/arm64 execution, native reports, platform-specific pack qualification, exact replay, soak and qualification guidance inherited from fn-105, fn-109, fn-110, fn-112, fn-113 and fn-114. It grants no current-candidate qualification or support claim.

Keep this spec deferred and not ready. Revival requires an explicit owner request for Darwin qualification, native darwin/arm64 execution, the supported pinned toolchain/profile and an identified frozen candidate. Linux qualification and Linux CI work remain deferred under fn-128. The owner instructed on 2026-10-07, "no PR for Lnux CI; defer that work". Creating a PR, pushing or triggering CI is outside this handoff.

## Acceptance Criteria

- **R1:** Build and launch the native Darwin patched runtime for an exact source/toolchain/profile closure. Retain the inherited runtime, upstream, live-capability, process, overlay, clock/draw, actual time-wire consumption, native baseline/candidate and U3/U1 controls, including Darwin sandbox and DTrace requirements. Preserve strict results, removed-skip seeds 1-24 and forward workloads at traced seeds 11/17 wherever the original ledgers require them. Errors: unavailable or unsupported execution, missing controls, failed build or stale identities leave qualification open. A source-check pass cannot substitute for runtime execution.
- **R2:** Retain the inherited native full-host, model, lifecycle, network, filesystem, process, race, built-CLI, default/integration, core, smoke, representative and affected-consumer evidence, including platform-bound version/pin/pack discover, review, approved generation and qualification. Preserve dispositions and required exact replay. Errors: omitted workloads, stale packs, mixed source closures, unexplained regressions or relaxed dispositions leave qualification open. Portable tests remain source-owned even when a native aggregate command includes them.
- **R3:** Qualify the real downstream consumer on native Darwin against the reviewed shared D8-D10 implementation and actual checkout. Retain closure/linked analyses, reachable adapters, exact consumer packs, refusal/drift controls, seeds 11 and 17 twice each, retained exact replay and evidence-backed Darwin guidance. Errors: absent checkout or source implementation, identity drift, classified failure or missing native execution leaves qualification open; no stand-in consumer qualifies it.
- **R4:** Retain an actual completed scheduled or dispatched native Darwin soak with its report and execution artifacts, as fn-112.10 requires. Preserve the ledger, per-cohort cross-batch comparisons, clean counts, diagnostics/load controls, overflow/infrastructure separation and measured platform-specific bound. Reconcile every transferred command in a final combined-candidate matrix with source/toolchain/profile/module identities, outcomes, qualification review and documentation. Errors: local-only stand-ins, historical or partial cohorts, absent reports, mixed revisions, stale bindings or missing required results prevent a current bound or overall Darwin qualification. Refresh invalidated results or justify unaffected coverage explicitly, without carrying invalid pack/profile identities. CI execution stays deferred until separately authorized.

## Boundaries

The source owners retain implementation, ordinary host-source coverage, both-source-set static checks, lint, generated-output validation, byte equivalence, fixed-identity characterization, matched first-baseline preservation, non-native measurements, source reviews, documentation consistency, consumer checkout requirements and source integration/review dependencies. Native execution of the full-host aggregate transfers here because its patched test driver and seeded child targets require a supported native runtime. Partial stock-Go coverage cannot be reported as a full native test-host pass or excuse portable failures.

Source completion does not depend on this deferred spec or fn-128. No source task is marked complete by this ownership transfer. Historical completed tasks and evidence stay unchanged. D6 and D15 remain conditional source work; the closed fn-107/fn-108 scopes and waivers stay unchanged. Only the owner's separately recorded current 642-byte U3 excess waiver applies to size.

This handoff changes no production code, supported platform, compatibility grant, replay disposition or collector/assembly restriction. Repairs found during later qualification return to their existing source owner and require their own admitted scope and verification. PR, push and CI actions require separate authority; qualification revival alone grants none.

## Decision Context

Maintainability (plan review): duplication - the dated ownership amendment appears in both Description and Acceptance of 69 donor tasks and in six donor specs to supersede their older local native blockers; structure - none identified.

Original Darwin requirements and command ledgers remain recoverable at Git commit d28d67c40ce74dd8886cf11b36fe7d2ddaf23675. The dated native transfer manifest maps the current open donor tasks, including fn-109 correction owners 38-49, to these four tasks. The October 4 Linux transfer and its older pinned snapshot remain historical and unchanged.

Task 1 establishes native runtime capability and initial outcomes. Tasks 2 and 3 consume that identity; task 3 also needs the reviewed downstream candidate and actual checkout. Task 4 consumes all three and owns actual soak plus finalization. Serialize execution that shares toolchain or qualification output directories; parallel runs require disjoint output directories and a frozen candidate. Source mutation invalidates affected evidence.

Native qualification is unverified until these owners retain current evidence. Developmental linux/arm64, emulation, cross-compilation, static coverage, other-platform reports and historical receipts cannot satisfy Darwin execution. Findings from the first three tasks remain open in the final matrix until their required results pass; collecting failures supplies diagnostics, never an overall pass.

## Early proof point

Task 1 must build and launch the supported native patched runtime with recorded identities. Host absence or an unsuccessful build blocks dependent native execution while source work continues independently.

## Quick commands

Use each donor's pinned command ledger. Flow validation checks the ownership records only. Native test commands always retain the required test_dep tag and original integration/race/workload controls.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Native baseline, runtime and clock controls | fn-149-gomad-deferred-darwin-qualification.1 | - |
| R2 | Native model, pack and integration qualification | fn-149-gomad-deferred-darwin-qualification.2 | - |
| R3 | Actual downstream native qualification | fn-149-gomad-deferred-darwin-qualification.3 | - |
| R4 | Scheduled/dispatched soak and combined final matrix | fn-149-gomad-deferred-darwin-qualification.4 | - |

## Execution waves

Task 1 precedes tasks 2 and 3; task 4 follows all three. Tasks 2 and 3 are dependency-eligible together, but shared toolchain/qualification resources require serialized writers and disjoint output directories for read-only runs. All four tasks remain blocked until the recorded revival prerequisites are met.
