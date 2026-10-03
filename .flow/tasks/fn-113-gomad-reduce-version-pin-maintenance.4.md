---
satisfies: [R5, R6]
---
# fn-113-gomad-reduce-version-pin-maintenance.4 Document the bump procedure, measure it against the baseline, and run both-platform gates

## Description
Documentation, the measured reduction in manual steps, and final gates (R5, R6).

**Size:** S
**Files:** `tools/gomad3/README.md`, `SPEC.md`, `CLI.md`, `ARCHITECTURE.md`, `tools/gomad3/toolchain/version/descriptor.go` (upgrade guide template), `.plans/GOMAD_NEXT.md`, `MILESTONES.md`
**Touches:** [tools/gomad3/*.md, tools/gomad3/toolchain/version/**, tools/gomad3/deterministicio/boundary/*.md, .plans/GOMAD_NEXT.md, MILESTONES.md, AGENTS.md]

### Approach
- Describe the bump procedure with the three new commands in README (compatibility-pack development and upgrade sections), SPEC `COMMAND.GOMADTOOL` and the maintenance requirements, CLI.md step 9 and command index, and ARCHITECTURE maintenance gates.
- The upgrade guide `deterministicio/boundary/upgrade-go1.27.1.md` is generated; edit the template in `descriptor.go` and run `make -C tools/gomad3 generate`.
- Walk one bump with the new commands and count manual steps (one command invocation or one hand edit each) against the task 1 baseline. Report both numbers.
- Update COMPAT-8 in the roadmap and the milestones Maintenance cost section.
- Run the gates on both platforms; coordinate with fn-110 task 5, which also edits the upgrade guide.

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
- [x] README, SPEC, CLI.md, ARCHITECTURE, and the generated upgrade guide describe the bump procedure with the new commands
- [x] Manual steps per bump are reported before and after against the task 1 baseline
- [x] Roadmap COMPAT-8 and the milestones section reflect the delivered state
- [ ] `make -C tools/gomad3 validate` and `test`, compatibility-pack qualification, and the core set pass on darwin/arm64 and linux/amd64; a missing linux run leaves this task open
- [x] Links and command inventories checked

Local progress verified on 2026-10-03: R5 and the architecture correction received a SHIP verdict in `task-4/working-tree-review.json`. Darwin validation and the literal full test gate pass, all eight mapped pack requests qualify, and all seven core workloads qualify and replay exactly. Source and gate identities are retained in `task-4/source-binding.json`; actual and normalized step counts are distinguished in `task-4/measurement.json` and `walkthrough.json`. Native linux/amd64 execution evidence is missing, so R6 and this task remain incomplete. No changes were committed.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
