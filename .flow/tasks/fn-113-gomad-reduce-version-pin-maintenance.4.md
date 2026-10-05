---
satisfies: [R5, R6]
---
# fn-113-gomad-reduce-version-pin-maintenance.4 Document and measure the bump procedure; run Darwin gates

## Description

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
