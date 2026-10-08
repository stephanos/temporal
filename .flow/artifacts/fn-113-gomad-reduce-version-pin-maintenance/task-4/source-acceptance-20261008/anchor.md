# Worker anchor bundle - fn-113-gomad-reduce-version-pin-maintenance.4 (spec fn-113-gomad-reduce-version-pin-maintenance)

Each section is the verbatim output of the command it is labeled with, in fixed order, untruncated. The bundle is a floor, not a ceiling - memory keyword-search and every further read remain available.

===== [1/11] task_show: `flowctl show fn-113-gomad-reduce-version-pin-maintenance.4 --json` =====
{
  "success": true,
  "assignee": "stephanos@users.noreply.github.com",
  "claim_note": "",
  "claimed_at": "2026-10-08T20:04:26.264683Z",
  "created_at": "2026-10-02T03:43:39.095023Z",
  "depends_on": [
    "fn-113-gomad-reduce-version-pin-maintenance.2",
    "fn-113-gomad-reduce-version-pin-maintenance.3"
  ],
  "id": "fn-113-gomad-reduce-version-pin-maintenance.4",
  "priority": null,
  "spec": "fn-113-gomad-reduce-version-pin-maintenance",
  "spec_path": ".flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.4.md",
  "status": "in_progress",
  "title": "Document and measure the bump procedure; run Darwin gates",
  "updated_at": "2026-10-08T20:04:26.264471Z",
  "impl": null,
  "review": null,
  "sync": null,
  "status_source": "flow-state"
}

===== [2/11] task_md: `flowctl cat fn-113-gomad-reduce-version-pin-maintenance.4` =====
---
satisfies: [R5, R6]
---
# fn-113-gomad-reduce-version-pin-maintenance.4 Document and measure the bump procedure; run Darwin gates

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Platform-aware pin/pack behavior, unavailable-platform refusal/unknown handling, measured steps, source reconciliation, Darwin gates and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Documentation, the measured reduction in manual steps, and final gates (R5, R6).

**Size:** S
**Files:** `tools/gomad3/README.md`, `SPEC.md`, `CLI.md`, `ARCHITECTURE.md`, `tools/gomad3/toolchain/version/descriptor.go` (upgrade guide template), `.plans/GOMAD_NEXT.md`, `MILESTONES.md`
**Touches:** [tools/gomad3/*.md, tools/gomad3/toolchain/version/**, tools/gomad3/deterministicio/boundary/*.md, .plans/GOMAD_NEXT.md, MILESTONES.md, AGENTS.md]

### Approach
- Describe the bump procedure with the three new commands in README (compatibility-pack development and upgrade sections), SPEC `COMMAND.GOMADTOOL` and the maintenance requirements, CLI.md step 9 and command index, and ARCHITECTURE maintenance gates.
- The upgrade guide `deterministicio/boundary/upgrade-go1.27.1.md` is generated; edit the template in `descriptor.go` and run `make -C tools/gomad3 generate`.
- Walk one bump with the new commands and count manual steps (one command invocation or one hand edit each) against the task 1 baseline. Report both numbers.
- Update COMPAT-8 in the roadmap and the milestones Maintenance cost section.
- Run the Darwin gates here; run the Linux gates under fn-128; coordinate with fn-110 task 5, which also edits the upgrade guide.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/README.md:628-669`, `:837-883` — upgrade and pack sections
- `tools/gomad3/SPEC.md:472` — `COMMAND.GOMADTOOL` table
- `tools/gomad3/toolchain/version/descriptor.go:287` — `renderUpgradeGuide`
- `tools/gomad3/CLI.md:487`, `:602` — pack step and command index

**Optional** (reference as needed):
- `.plans/GOMAD_NEXT.md:99` — COMPAT-8

### Key context
- fn-111's link and command-inventory checks are manual; rerun them for the edited docs.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [x] README, SPEC, CLI.md, ARCHITECTURE, and the generated upgrade guide describe the bump procedure with the new commands
- [x] Manual steps per bump are reported before and after against the task 1 baseline
- [x] Roadmap COMPAT-8 and the milestones section reflect the delivered state
- [ ] `make -C tools/gomad3 validate` and `test`, compatibility-pack qualification, and the core set pass on native darwin/arm64; Linux execution belongs to fn-128; missing required Darwin evidence leaves this task open; Linux evidence is owned by fn-128
- [x] Links and command inventories checked

Historical progress receipt (before the Linux ownership transfer). Local progress verified on 2026-10-03: R5 and the architecture correction received a SHIP verdict in `task-4/working-tree-review.json`. Darwin validation and the literal full test gate pass, all eight mapped pack requests qualify, and all seven core workloads qualify and replay exactly. Source and gate identities are retained in `task-4/source-binding.json`; actual and normalized step counts are distinguished in `task-4/measurement.json` and `walkthrough.json`. Native linux/amd64 execution evidence is missing, so R6 and this task remain incomplete. No changes were committed.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

===== [3/11] spec_show: `flowctl show fn-113-gomad-reduce-version-pin-maintenance --json` =====
{
  "success": true,
  "branch_name": "fn-113-gomad-reduce-version-pin-maintenance",
  "completion_review_status": "unknown",
  "completion_reviewed_at": null,
  "created_at": "2026-10-02T03:34:29.349673Z",
  "default_impl": null,
  "default_review": null,
  "default_sync": null,
  "depends_on_epics": [],
  "id": "fn-113-gomad-reduce-version-pin-maintenance",
  "impl_review_rounds": {
    "fn-113-gomad-reduce-version-pin-maintenance.1": 0,
    "fn-113-gomad-reduce-version-pin-maintenance.2": 0,
    "fn-113-gomad-reduce-version-pin-maintenance.3": 0
  },
  "next_task": 1,
  "plan_review_rounds": 0,
  "plan_review_status": "ship",
  "plan_reviewed_at": "2026-10-02T04:00:32.611166Z",
  "review_hash_epoch": {
    "impl:fn-113-gomad-reduce-version-pin-maintenance.1": 2,
    "impl:fn-113-gomad-reduce-version-pin-maintenance.2": 1,
    "impl:fn-113-gomad-reduce-version-pin-maintenance.3": 1,
    "plan": 1,
    "plan#completion": 0
  },
  "review_pending_rounds": {},
  "review_reservations": {},
  "review_transport_failures": {
    "impl:fn-113-gomad-reduce-version-pin-maintenance.1": 0,
    "impl:fn-113-gomad-reduce-version-pin-maintenance.2": 0,
    "impl:fn-113-gomad-reduce-version-pin-maintenance.3": 0,
    "plan": 0
  },
  "spec_path": ".flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md",
  "status": "open",
  "title": "Gomad reduce version-pin maintenance",
  "updated_at": "2026-10-08T19:54:14.004107Z",
  "tasks": [
    {
      "id": "fn-113-gomad-reduce-version-pin-maintenance.1",
      "title": "Baseline the pins and add the pin impact report",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": []
    },
    {
      "id": "fn-113-gomad-reduce-version-pin-maintenance.2",
      "title": "Regenerate adapter anchors for a new module version behind an approval digest",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-113-gomad-reduce-version-pin-maintenance.1"
      ]
    },
    {
      "id": "fn-113-gomad-reduce-version-pin-maintenance.3",
      "title": "Refresh invalidated packs in one command and remove unselected variants",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-113-gomad-reduce-version-pin-maintenance.1",
        "fn-113-gomad-reduce-version-pin-maintenance.2"
      ]
    },
    {
      "id": "fn-113-gomad-reduce-version-pin-maintenance.4",
      "title": "Document and measure the bump procedure; run Darwin gates",
      "status": "in_progress",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-113-gomad-reduce-version-pin-maintenance.2",
        "fn-113-gomad-reduce-version-pin-maintenance.3"
      ]
    }
  ],
  "ready": false,
  "no_plan": false
}

===== [4/11] spec_md: `flowctl cat fn-113-gomad-reduce-version-pin-maintenance` =====
# Gomad: reduce version-pin maintenance

## Native qualification ownership amendment (2026-10-07)

The owner approved deferring remaining native Darwin qualification to [fn-149](fn-149-gomad-deferred-darwin-qualification.md). The [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md) maps every affected open task, requirement slice and exact command ledger. Linux qualification and Linux CI work remain deferred under fn-128; the owner explicitly instructed "no PR for Lnux CI; defer that work". Missing transferred native proof no longer blocks this spec or its mapped source tasks.

This dated owner decision supersedes older missing-Darwin, no-renewed-deferral and native-first admission clauses only for transferred native execution, reports, packs/replay, qualification measurements, soak and platform-specific qualification guidance. The mapped task amendments are authoritative over their older criteria. Source integration, predecessor source review and retained source acceptance remain required, including fn-110.5's predecessor rule. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, documentation consistency and actual consumer checkout prerequisites stay source-owned. Native aggregate test-host execution transfers; embedded portable coverage and portable failures do not. Partial portable results cannot be labeled a full native gate pass.

All other original criteria, dependency edges, completed tasks and historical evidence remain unchanged. This transfer completes no task and creates no qualification pass, soak bound, PR, push or CI run. Qualification stays unverified under its native owner until retained current-candidate proof exists. fn-128's Linux revival trigger remains unchanged; fn-149 additionally requires an explicit owner request and native darwin/arm64 execution. Neither revival grants publication or CI authority.


## Linux ownership amendment (2026-10-04)

The owner transferred all remaining native linux/amd64 execution and Linux-only deferred work to [fn-128](fn-128-gomad-deferred-linux-qualification-and.md) on 2026-10-04. Missing transferred Linux evidence does not block this spec or its retained tasks. Darwin, shared implementation, static coverage of both supported source sets, preservation, size, full-host, review and other independent requirements remain here. Historical reports and completed-task evidence remain unchanged and do not establish current-source qualification.

Linux owners: [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). The [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md) maps each affected task and requirement to its owner. Native execution clauses below apply to Darwin within this spec; references to both platforms retain static/API behavior and historical scope, with outstanding Linux execution owned by fn-128. This explicit owner decision supersedes older no-renewed-deferral and unavailable-Linux completion rules only for the transferred obligations.


**Plan date:** 2026-10-01

## Goal & Context

Cut the manual work a dependency or Go version bump causes, without loosening
any exact pin. The [2026-10-01 quality assessment](../../MILESTONES.md#maintenance-cost)
counted the pins: a 1010-line runtime patch with a 57-file overlay, 131
interception fingerprints, 15 dependency adapters anchored by 129 SHA-256
literals, and 12 compatibility packs covering 19 module versions. Upstream
`go.mod` changed in 73 commits over six months, including 4 `go` directive
bumps. Each pin fails closed on a bump, which is intended. The cost is that
adapter repair has no command, pack repair takes four commands, and nothing
reports ahead of time which pins a given bump breaks.

Counts above come from the assessment's source reading and are re-measured as
the baseline before work starts.

### Relationship to existing work

- fn-110 owns reducing the runtime patch. This spec does not change the patch
  or overlay content.
- fn-107 owns the downstream cell, including the six adapters for modules the
  server does not import.
- fn-112 owns determinism assurance and the test suite shape.
- [COMPAT-8](../../.plans/GOMAD_NEXT.md#compat-8-dependency-and-go-upgrade-impact-reports)
  is the roadmap item this spec draws from. Rollback bundles and release
  attestations stay on the roadmap.

## Architecture & Data Models

### Pin impact report

One command takes a candidate `go.mod` and `go.sum` and
reports every pin it invalidates: adapters by module and version, packs by
rule and source-set digest, interception fingerprints, and clock-inventory
references. The report is path-free canonical JSON with a human rendering, and
exits nonzero when any pin is invalidated. It reads the same descriptors the
build reads, so it cannot disagree with the build's fail-closed checks.

### Adapter regeneration

One governed command re-derives an adapter's rewrite and digest anchors for a
new exact module version. It applies each rewrite by its existing structural
exact-occurrence anchor, fails when an anchor no longer matches exactly once, and prints the changed
upstream source for review. It writes the new anchors only with an explicit
approval digest, the same control compatibility packs use. A changed upstream
file never produces a silently shifted rewrite.

### Pack refresh

One command runs `discover`, `review`, and `generate` for every request a bump
invalidates and stops at the review approval. Stale variants that no qualified
module version selects are removed, with the evidence that nothing selects
them.

## API Contracts

Pins, pack and adapter identities, fail-closed behavior, approval digests,
target identity, and replay compatibility are unchanged. The new commands are
`gomadtool` subcommands; the `gomad` CLI grammar is unchanged. Regenerated
adapters and packs carry new identities, and retained artifacts keep theirs.

## Edge Cases & Constraints

- The [milestone constraints](../../MILESTONES.md#constraints) apply. No pin is
  widened to a version range, and no generic capability is granted.
- Regeneration never approves on its own. Review of changed upstream source by
  a person stays in the flow.
- The impact report must name a pin it cannot evaluate as unknown, never as
  unaffected.
- Module downloads use the exported proxy settings and fail as infrastructure
  errors when unavailable.
- Both platforms' packs and adapters are covered; a host that cannot evaluate
  the other platform's pins says so in the report.

## Acceptance Criteria

- **R1:** A re-measured baseline records every pin class, its count, and the
  commands and manual steps a bump of each currently needs. Errors: a count
  that differs from the assessment is corrected in the milestones.

- **R2:** The pin impact report lists every invalidated pin for a candidate
  `go.mod`, and a fixture bump of one adapted and one packed
  module shows the expected entries. Errors: a pin the build later rejects
  that the report called unaffected fails this criterion.

- **R3:** Adapter regeneration re-derives anchors for a new exact version
  behind an approval digest, and a negative fixture with a moved anchor fails
  without writing. One real adapter is regenerated across a version bump with
  the command and qualifies.

- **R4:** Pack refresh runs the authoring steps for every invalidated request
  up to approval in one command. Unselected stale pack variants are removed
  with retained evidence.

- **R5:** The README and upgrade guide describe the bump procedure with the new
  commands, and the measured manual steps per bump are reported against the
  R1 baseline.

- **R6:** `make -C tools/gomad3 validate` and `test`, compatibility-pack
  qualification, and the core set pass on native Darwin. Linux execution belongs to fn-128.4/.7. Errors: missing required Darwin execution leaves this spec's acceptance incomplete.

## Boundaries

- A Go-version candidate as report input is excluded; the upgrade dossier
  covers Go bumps.
- Patch and overlay reduction, feature removal, version-range pins, automatic
  approval, and release bundles are excluded.
- Automating the runtime patch rebase across Go releases is excluded; the
  existing `patch-regenerate` command and upgrade dossier keep that scope.

## Decision Context

The assessment ranked adapter anchors as the second-largest recurring cost
after the Go rebase, and the only one with no repair command. A report that
runs before a bump lets the owner batch repairs, where today the first signal
is a failed build.

## Planning decisions (2026-10-01)

Task breakdown settled the points below. Each is a default the owner can
change before the task that uses it starts.

- **R2 source of truth.** Adapter anchors stay Go constants. The report reads
  them by importing the adapter registry, so no data migration and no adapter
  identity change.
- **R2 input** is one candidate `go.mod` with its `go.sum`, by default the
  repository root module. Exit status follows the existing convention: 0 for
  no invalidated pin, 1 for at least one, 2 for invalid input, 3 for
  infrastructure failure. An unknown pin counts as invalidated.
- **R2 side effects.** Module resolution runs outside the target module with a
  private module cache, because `go mod download` inside a target module
  rewrites its `go.sum`.
- **R3 anchors** are the existing exact-occurrence byte anchors. An anchor
  that matches zero or more than one time fails. The approval digest covers
  the changed upstream source and the proposed new anchors, comes from a dry
  run, and is passed back on the command line. The transaction covers the
  generated outputs: the full set is staged and verified in a scratch copy,
  then published under an exclusive lock after revalidating the checkout, with
  a marker that lets the next run complete or roll back an interrupted
  publication.
- **R3 real adapter** is the first adapted module the root `go.mod` has moved
  past when the task starts. If none has moved, the fixture bump stands in and
  the criterion says so.
- **R4 inputs.** Refresh runs on a checkout where the bump is applied. Each
  request is discovered in its own target module through one checked-in
  request-to-directory table shared with the Makefile qualification list.
- **R4 approval** is per request. A request is done when its stored approval
  matches the review digest of freshly discovered evidence, so an approval of
  older evidence never counts and partial progress survives a rerun. Variants are removed only on a host that can evaluate their
  platform; the others are reported and left.
- **R5 manual step** means one command invocation or one hand edit of a
  checked-in file.

## Open Questions

1. fn-109 tasks 10 and 11 change the adapter registry and source-inventory
   owner, and fn-105 task 8 adds adapters. Should R3 wait for them, or land
   first and be rebased?

## Quick commands

```bash
make -C tools/gomad3 validate
go -C tools/gomad3 test -tags test_dep ./cmd/gomadtool ./deterministicio ./internal/compatibilitypack/...
```

## Early proof point

Task fn-113-gomad-reduce-version-pin-maintenance.1 validates the core approach:
the impact report names exactly the pins the build rejects for a fixture bump.
If the report and the build disagree, re-evaluate reading anchors through the
registry before tasks 2 and 3.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Re-measured pin baseline | .1 | — |
| R2 | Pin impact report | .1 | — |
| R3 | Adapter regeneration behind approval | .2 | — |
| R4 | Pack refresh and stale variants | .3 | — |
| R5 | Bump procedure documented and measured | .4 | — |
| R6 | Darwin gates pass; Linux execution owned by fn-128.4/.7 | .4 | — |

===== [5/11] git_status: `git status --short --branch` =====
## gomad...origin/gomad [ahead 9]
?? .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/conductor-preparation-20261008/
?? .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-4/source-acceptance-20261008/
?? .turbo/plans/gomad3-glossary-update.md
?? .turbo/technical-debt.md

===== [6/11] git_log: `git log -5 --oneline` =====
da1e726eab gomad: admit retained bump documentation source acceptance
9663e4c1ba gomad: complete retained pack refresh source acceptance
46a42ec94b gomad: verify retained pack refresh source candidate
273ec2af11 gomad: record pack refresh worker dispatch
2b84decef8 gomad: admit retained pack refresh source acceptance

===== [7/11] git_branch: `git rev-parse --abbrev-ref HEAD` =====
gomad

===== [8/11] memory_enabled: `flowctl config get memory.enabled --json` =====
{
  "success": true,
  "key": "memory.enabled",
  "value": true
}

===== [9/11] glossary: `flowctl glossary list --json --match "<task title + description>"` =====
(no glossary entry matches the task title or description - skipped)

===== [10/11] memory_index: `flowctl memory list` =====
bug/integration/
  go-mod-download-inside-a-target-module-2026-09-28 — "go mod download inside a target module rewrites its go.sum and bypasses sum chec" (module: tools/gomad3/target)
  shard-merge-and-prepared-target-cache-2026-09-29 — "Shard merge and prepared-target cache must bind source identity, not go.mod" (module: tools/gomad3/qualification/set)
  profile-adapter-changes-leave-libc-2026-10-01 — "Profile adapter changes leave libc-bound compatibility packs stale" (module: tools/gomad3/internal/compatibilitypack)
  engine-summary-fields-need-the-runner-2026-10-03 — "Engine summary fields need the runner projection and both CLI exploration lines" (module: tools/gomad3/runner/runner.go)
  recompute-mapping-summaries-after-final-2026-10-08 — "Recompute mapping summaries after final row classification" (module: tools/gomad3 test-preservation evidence)

bug/runtime-errors/
  gobuild-cross-platform-matchfile-keeps-2026-09-28 — "go/build cross-platform MatchFile keeps host arch feature ToolTags" (module: tools/gomad3/qualification/set/manifestgen)

===== [11/11] dependencies: ids, titles, statuses, done summaries =====
- fn-113-gomad-reduce-version-pin-maintenance.2 [done] - Regenerate adapter anchors for a new module version behind an approval digest
    Retained R3 source acceptance is verified on `8f292989713ff55b97016e5156eb92667b29102d`. Approval/no-write, anchor refusal, complete publication/recovery, drift/concurrency, stale-pack and real Sprig default-pipeline controls pass. Root adapter pins remain nine selected, six absent, zero moved. Diagnostic writes and lock release are checked with preserved status/error identity, publication warnings and release ordering.
    
    The [final handover](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/conductor-source-acceptance-20261008/final-handover.md) and [worker evidence](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/source-acceptance-20261008/evidence.json) bind all 17 final gates, source/tools/logs, actual portable assertions, retained overlay and exact helper preservation. Root independently verified them and replayed the four unchanged cache controls: 17 passes, no failures/skips, real permission faults. The two filesystem selections are complementary; no full single-environment or native aggregate pass is claimed. Scoped lint resolves all 14 owned sites and retains 86 unchanged OTHER findings without suppression.
    
    Independent [review](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/conductor-source-acceptance-20261008/review-receipt.json) of `ca6fd855..8f292989` returned SHIP with R3 met, no findings and no unaddressed requirements. All three fresh Codex gpt-6.1-sol/high draws independently verified retained source/log evidence; their own Go reruns were blocked before execution by the read-only sandbox. Same requested GPT family, not cross-family review.
    
    Native qualification/workloads/reports/replay/soak remain deferred and unverified under fn-149/fn-128. External supplied Memberlist TCP checkout and fn-105 SDK Git checkout are not claimed. Historical summaries/evidence and both unrelated user files remain preserved. No PR, push or CI action.
    
    stage: worker - ran
    stage: impl-review - ran [2026-10-08T18:40:30.377811Z..2026-10-08T18:45:27.483643Z] (model: gpt-6.1-sol at high)
    stage: memory-capture - skipped(policy: no NEEDS_WORK to SHIP fix transition)
    stage: plan-sync - skipped(config: planSync.enabled=false)
    Tracker sync: n/a (bridge inactive)
    Shipped: 0 (no PR/push authority)
- fn-113-gomad-reduce-version-pin-maintenance.3 [done] - Refresh invalidated packs in one command and remove unselected variants
    Retained R4 source acceptance is complete at candidate `46a42ec94bcb5ed40efd9f72f1fe51d1b01755a4`.
    Eight checked stderr sites preserve primary statuses and all unaffected source
    bytes. Real mapped-module discovery, partial approval, stale-approval refusal,
    continuation and both selected variants are verified. No variant is retired.
    
    The [current handover](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-3/conductor-source-acceptance-20261008/final-handover.md)
    and [source assertion map](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-3/source-acceptance-20261008/assertion-mapping.md)
    bind 910 unique parent-inclusive passing identities, one unchanged native skip,
    mandatory source gates and exact preservation. The broad/mixed commands and
    unfiltered lint remain RED as recorded; the exact original publication parent
    uses narrowly admitted private tmpfs, with unchanged assertions. Both earlier
    portable failure causes remain unknown. All eight owned lint findings disappear;
    78 OTHER blocks remain exact. No native or whole-broad-command pass is claimed.
    
    Three fresh read-only `codex:gpt-6.1-sol:high` draws returned SHIP with R4 met,
    no findings and no unaddressed requirements on the nonempty base-to-candidate
    range. The [actual receipt](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-3/conductor-source-acceptance-20261008/review-receipt.json)
    binds their sessions and finalizer-derived verdict. Reviewer runtime reruns
    were blocked before execution; retained source/log evidence supports acceptance.
    
    Native Darwin fn149 and Linux fn128 remain deferred/unverified. Actual checkout
    prerequisites and unrelated user files remain untouched; no PR/push/CI occurs.
    
    Tier: session (jev-unavailable(no_key))
    stage: impl-review - ran [2026-10-08T19:50:56.284893Z..2026-10-08T19:54:13.974862Z] (three fresh same-GPT-family draws; SHIP, R4 met)
    stage: memory-capture - skipped(policy: clean first-pass SHIP)
    stage: plan-sync - skipped(config: planSync.enabled=false)
    Tracker sync: n/a (bridge inactive).

