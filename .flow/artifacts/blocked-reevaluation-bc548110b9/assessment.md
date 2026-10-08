# Blocked Gomad task reassessment, 2026-10-07

The remaining milestone source work can advance without Darwin or Linux qualification. This assessment covers all 65 live blocked tasks outside fn-149 at `bc548110b9321df59d757d0e0e5c0fea464c002b`. Fifty-eight retain source work; seven Linux tasks remain deliberately deferred. No source task is held by the absence of transferred native proof alone.

No task is certified complete. The reevaluation found two explicit source-contract conflicts, executable verification/review backlogs, preserved predecessor waits, concrete documentation/cleanup work, and local setup gaps. Native deferral and the exact 642-byte waiver do not waive those requirements.

## Findings that change the next action

- The official Go 1.27.1 source archive is reachable now. A streamed GET matched descriptor SHA-256 `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`. Cache absence is setup work, so fn-110's pinned regeneration/equivalence and materialized static checks have a local route. This assessment downloaded no cache file. [Archive evidence](patch.md#missing-input-and-the-minimal-route).
- D27/fn-105.32 has an obsolete native-only blocker. Its inventory source is unchanged from its implementation; the collector-stamp overwrite remains declined. A bounded source evidence/review reconciliation can advance now. [D27 evidence](followups.md#fn-10532--d27).
- Fn-112.10's actual scheduled soak moved to the native owners. Finish its current harness/docs/lint acceptance and .5/.16 source dependencies without starting CI. [Assurance evidence](assurance.md).
- Fn-109.21/R18 still demands aggregate first-baseline preservation while fn-114.11/.12 explicitly require trace/controller format migrations. Those owners must reconcile the exact conflict before anyone restores a decoder or changes recorded behavior. The actual first baseline and reconstructed 670-file tree remain available; missing scratch state is not the blocker. Existing 10/100 measurements need current-candidate attribution, not a claim that no measurements exist. [Runtime evidence](runtime.md#fn-10921---aggregate-preservation-and-final-matrix).
- Fn-109.28 must preserve two invariant panics while its retained lint policy forbids them. Its bounded correction prohibits suppression and semantic redesign. It needs an explicit compatible owner/scope disposition. The other remaining panic, fixed-error, load-spin and CLI-output findings also require contract-aware ownership. [Correction evidence](corrections.md).
- Concrete source work remains in fn-113.2's unchecked lock-release paths and portable exhaustive source test setup. Five exact adapter modules are absent from the offline cache; their network availability was not tested. Fn-114.14 must correct the stale all-implementation-tasks-done sentence in `.plans/GOMAD_CMP.md:44`. The selected v041 fixture and pack inputs are already restored. [Maintenance evidence](maintenance.md), [documentation evidence](assurance.md).

## Classification counts

These are primary classifications, not acceptance verdicts. Every source row retains applicable portable coverage, generated validation, both-source-set static checks, preservation, standards, review and existing dependency/admission rules. A dependency wait permits preparation and reconciliation but does not declare its predecessor accepted.

| Primary classification | Tasks |
| --- | ---: |
| Verification/review backlog | 28 |
| Dependency wait | 24 |
| Owner-contract reconciliation | 2 |
| Documentation reconciliation | 2 |
| Source closure reconciliation | 1 |
| Source correction/setup | 1 |
| Deliberately deferred | 7 |
| Total outside Darwin qualification | 65 |

## Current integrated lint evidence

The applicable retained full report has 265 findings across 55 Gomad host packages and Make exit 2. Its classes are 208 errcheck, 2 exhaustive, 11 forbidigo and 44 staticcheck. The full errortype stage was not reached. A recent-change fast gate passed because its reporting base differs; it does not establish the original-base gate.

Root freshly hashed `root-full-lint.log` to `c9a8008eba9d4aaa202e4eb44e113ebe99322d65e02dfee9c64da1edc5468bd5` and checked the source diff from `d28d67c40c` to this candidate. Gomad code, tests, module inputs and lint configuration are unchanged; only its README changed. Root therefore reused the source-equivalent full receipt rather than rerunning an unchanged gate. This is a revalidated retained result, not a newly executed full gate. No source correction in this reevaluation removed a finding.

The latest adapter-cache correction added zero findings and removed one against its 266-finding predecessor. Residual debt predates that correction, with mixed origins before and during fn-109. Temporal's original main baseline lacks this Gomad module, so calling every finding pre-existing Temporal debt would be incorrect. Preserve baseline `951c5516e9e7b3066e7e069adda9565cfd68844c`, pinned rules and `FIX=false` for eventual original-base acceptance. [Full conductor receipt](../fn-109-gomad-deepen-modules-and-tool-interfaces/task-49/conductor-checks.json), [lint provenance](../fn-109-gomad-deepen-modules-and-tool-interfaces/lint-linux-feasibility-20261007.md).

## Task-by-task disposition

All 65 tasks remain live `blocked` during this assessment. The linked domain reports give inspected implementation, acceptance, exact evidence boundaries and bounded next checks for each source task. No row requests a second implementation of already delivered work.

| Task | Primary classification | Next retained action | Evidence |
| --- | --- | --- | --- |
| fn-109.2 | Verification/review backlog | Rebind options characterization and changed caller receipts against the actual first baseline. | [campaign](campaign.md) |
| fn-109.8 | Dependency wait | Reconcile predecessor source acceptance (fn-109.7), current coverage, preservation, lint and review. | [campaign](campaign.md) |
| fn-109.9 | Dependency wait | Reconcile predecessor source acceptance (fn-109.8, fn-109.40), current coverage, preservation, lint and review. | [campaign](campaign.md) |
| fn-109.10 | Dependency wait | Reconcile predecessor source acceptance (fn-109.9), current coverage, preservation, lint and review. | [campaign](campaign.md) |
| fn-109.11 | Dependency wait | Reconcile predecessor source acceptance (fn-109.10), current coverage, preservation, lint and review. | [campaign](campaign.md) |
| fn-109.12 | Dependency wait | Reconcile predecessor source acceptance (fn-109.11), current coverage, preservation, lint and review. | [campaign](campaign.md) |
| fn-109.13 | Dependency wait | Reconcile predecessor source acceptance (fn-109.12), current coverage, preservation, lint and review. | [runtime](runtime.md) |
| fn-109.14 | Dependency wait | Reconcile predecessor source acceptance (fn-109.13), current coverage, preservation, lint and review. | [runtime](runtime.md) |
| fn-109.15 | Dependency wait | Reconcile predecessor source acceptance (fn-109.14), current coverage, preservation, lint and review. | [runtime](runtime.md) |
| fn-109.16 | Dependency wait | Reconcile predecessor source acceptance (fn-109.15), current coverage, preservation, lint and review. | [runtime](runtime.md) |
| fn-109.17 | Dependency wait | Reconcile predecessor source acceptance (fn-109.16), current coverage, preservation, lint and review. | [runtime](runtime.md) |
| fn-109.18 | Dependency wait | Reconcile predecessor source acceptance (fn-109.17), current coverage, preservation, lint and review. | [runtime](runtime.md) |
| fn-109.19 | Dependency wait | Reconcile predecessor source acceptance (fn-109.18), current coverage, preservation, lint and review. | [runtime](runtime.md) |
| fn-109.20 | Dependency wait | Reconcile predecessor source acceptance (fn-109.19), current coverage, preservation, lint and review. | [runtime](runtime.md) |
| fn-109.21 | Owner-contract reconciliation | Resolve aggregate R18 treatment of fn-114's approved trace/controller migrations; attribute existing 10/100 evidence. | [runtime](runtime.md) |
| fn-109.23 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.24 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.25 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.26 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.27 | Documentation reconciliation | Append current v041 restoration evidence and reconcile the still-real format/controller differences. | [corrections](corrections.md) |
| fn-109.28 | Owner-contract reconciliation | Resolve the bounded invariant-panic/lint conflict without changing protected semantics or suppressing policy. | [corrections](corrections.md) |
| fn-109.29 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.30 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.31 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.32 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.33 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.34 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.35 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.36 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.37 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.38 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.39 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.40 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.41 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.42 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.43 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.44 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.45 | Verification/review backlog | Rebind the retained scoped correction to current sources, matched first-baseline preservation, integrated lint and source review. | [corrections](corrections.md) |
| fn-109.46 | Dependency wait | Reconcile predecessor source acceptance (fn-109.45), current coverage, preservation, lint and review. | [corrections](corrections.md) |
| fn-109.47 | Dependency wait | Reconcile predecessor source acceptance (fn-109.46), current coverage, preservation, lint and review. | [corrections](corrections.md) |
| fn-109.48 | Dependency wait | Reconcile predecessor source acceptance (fn-109.47), current coverage, preservation, lint and review. | [corrections](corrections.md) |
| fn-109.49 | Dependency wait | Reconcile predecessor source acceptance (fn-109.48), current coverage, preservation, lint and review. | [corrections](corrections.md) |
| fn-110.2 | Verification/review backlog | Reuse exact compact source receipts and the narrow 642-byte waiver; finish retained source review. | [patch](patch.md) |
| fn-110.3 | Dependency wait | After .2 source acceptance, verify relocation, exact source sets, pristine rand and generated identities. | [patch](patch.md) |
| fn-110.4 | Dependency wait | After .3, cache the reachable checksum-pinned archive and rerun real pinned equivalence/regeneration tests. | [patch](patch.md) |
| fn-112.5 | Verification/review backlog | Reuse unchanged both-source-set inventories; rebind generated identities, source coverage and review. | [assurance](assurance.md) |
| fn-112.9 | Verification/review backlog | Reconcile removed-behavior mappings with current portable Runner/CLI ownership and retained before/after evidence. | [assurance](assurance.md) |
| fn-112.10 | Dependency wait | Finish .5/.16 source acceptance and harness/docs/lint review; native scheduled soak stays deferred. | [assurance](assurance.md) |
| fn-112.16 | Verification/review backlog | Bind collision/noncollision publication preservation and current store/Runner/CLI source evidence. | [assurance](assurance.md) |
| fn-114.13 | Verification/review backlog | Reconcile current scheduler through the compact preservation chain; finish source validation and review. | [assurance](assurance.md) |
| fn-114.14 | Documentation reconciliation | After .13, correct the stale all-tasks-done sentence and reconcile all ten dispositions with current Flow. | [assurance](assurance.md) |
| fn-105.3 | Dependency wait | Close the private-executor migration once fn-109.6 has retained source acceptance. | [followups](followups.md) |
| fn-105.4 | Dependency wait | Close by reference to fn-109.19; preserve current architecture/static/review obligations. | [followups](followups.md) |
| fn-105.5 | Dependency wait | Close by reference to fn-109.20 after current-guide and preservation reconciliation. | [followups](followups.md) |
| fn-105.32 | Source closure reconciliation | Bind unchanged clock inventory, current contract, static evidence and review; keep the stamp-overwrite refusal. | [followups](followups.md) |
| fn-113.1 | Verification/review backlog | Bind the original maintenance baseline and current pin/build comparator coverage; finish standards/review. | [maintenance](maintenance.md) |
| fn-113.2 | Source correction/setup | After .1, address two unchecked lock releases and the portable test's driver/cache prerequisite; retain approvals. | [maintenance](maintenance.md) |
| fn-113.3 | Dependency wait | After .1/.2, reconcile refresh approvals and restored selected v041/v047 evidence against the original baseline. | [maintenance](maintenance.md) |
| fn-128.1 | Deliberately deferred | Owner revival request, native linux/amd64 execution and frozen pinned candidate/runtime are required. | [linux](#linux-owner-evidence) |
| fn-128.2 | Deliberately deferred | After .1 and authorized revival, reproduce the causal divergence and strict replay regression; no CI action now. | [linux](#linux-owner-evidence) |
| fn-128.3 | Deliberately deferred | Retain the D21 trigger decision; eventually require the audit or explicit owner-approved not-triggered disposition. | [linux](#linux-owner-evidence) |
| fn-128.4 | Deliberately deferred | After .1 and revival, retain native model/workload/pack evidence with exact source and approval identities. | [linux](#linux-owner-evidence) |
| fn-128.5 | Deliberately deferred | After .1 and revival, retain the actual scheduled/dispatched native soak; no bound follows from source tests. | [linux](#linux-owner-evidence) |
| fn-128.6 | Deliberately deferred | Requires revival, native execution, the actual downstream checkout and shared D8 source candidate. | [linux](#linux-owner-evidence) |
| fn-128.7 | Deliberately deferred | After .1-.6, reconcile a frozen final native matrix and fresh review; source tasks do not depend on it. | [linux](#linux-owner-evidence) |

## Linux owner evidence

Root read current `flowctl show/cat` state for fn-128 and its seven tasks. [The Linux owner](../../specs/fn-128-gomad-deferred-linux-qualification-and.md) still requires an explicit qualification request, native linux/amd64 execution and a pinned frozen candidate. This reevaluation supplies neither revival nor CI authority. The source/native split is authoritative in the [October 7 manifest](../native-scope-transfer-2026-10-07.md); the [October 4 Linux manifest](../linux-scope-transfer-2026-10-04.md) preserves its original command obligations.

Fn-128.3 additionally retains the D21 audit-trigger or owner-approved not-triggered decision. Fn-128.6 additionally needs the actual downstream checkout and shared fn-105.8 source implementation. Fn-128.7 waits for all six predecessor obligations. No source spec depends on unfinished Linux qualification. Native linux/arm64 development, stock-Go checks, cross-compilation and Darwin evidence cannot qualify Linux/amd64.

## Recommended source sequence

1. Follow milestone group 1. Cache the verified pinned archive, reconcile the combined D26/fn-110 candidate, fn-114.13/.14, fn-112.5 and D27. Reuse unchanged compact/inventory receipts and run only uncovered source checks.
2. Follow group 2. Reconcile fn-112.16/.9, fn-113.1-.4 and fn-109.2-.6. Resolve actual source cleanup/static/dependency setup and retain current source reviews. Preserve selected v041 and original measurement baselines.
3. Continue the existing fn-109 and fn-110 source dependency chain. Use the admitted correction receipts once; route remaining lint failures to bounded contract-aware owners. Resolve the R18 migration and invariant-panic/lint conflicts without a blanket waiver, then finish current-source preservation and evidence matrices.
4. Leave both native owners deferred. Resume D8-D10's separate source work only with the actual downstream checkout. Keep conditional D6/D15 workload triggers unchanged.

This is an assessment snapshot, not a replacement tracker or a new implementation plan. Current task state and acceptance remain in Flow. No lifecycle reset, start, done, dependency change or acceptance edit occurred. In particular, `flowctl task reset` clears historical completion/evidence fields, so this report does not use a blanket reset to replace stale native blocker wording.

## Verification scope and limits

Seven requested `gpt-6-astra/high` research agents inspected disjoint task domains; root inspected Linux ownership and integrated lint provenance. The tier judge returned unavailable `no_key`; the explicit project research model remained in the host dispatch. Actual executing-model metadata is unavailable. These reports supply research classifications, not formal implementation-review verdicts.

Focused stock-Go source diagnostics cover options characterization, Artifact behavior, command helpers, simulation-time/progress, ownership/selectors, architecture negative controls, soak harness, collision controls, CLI forwarding, pin impact, refresh approvals, source-checksum cleanup and selected pack evidence. Domain reports record exact commands, exits, skips and unsupported-host stops. Such controls provide no native full-host/runtime/pack/soak qualification. Historical evidence is reused only at its matching input scope; changed manifests need successor reconciliation rather than an automatic pass or an automatic defect claim.

Only this assessment and its seven domain reports were authored. No product source, generated output, task record, milestone index, commit, PR, push, CI or native qualification changed. Darwin fn-149.1-.4 are excluded from this reevaluation. The two unrelated untracked `.turbo` documents remain untouched.

Root's final live-state comparison confirmed 65 distinct report rows match all blocked tasks outside Darwin, with all 65 statuses and dependency lists unchanged and four Darwin tasks excluded. `flowctl validate --all` returned valid across 22 specs and 198 tasks, with zero errors and two warnings. `git diff --check` returned exit 0; tracked and staged diffs are empty. These checks verify assessment coverage and the absence of tracked mutations, not source acceptance.
