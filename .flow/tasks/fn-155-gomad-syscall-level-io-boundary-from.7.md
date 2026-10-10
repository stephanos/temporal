---
satisfies: [R7, R8]
---
# fn-155-gomad-syscall-level-io-boundary-from.7 Measure the boundary's size, record the decision and annotate dependent work

## Description
Measure patch and overlay size against the fn-110.1 baseline, write the go/no-go decision for files and DNS citing the evidence, and annotate dependent tasks. Fold the documentation of the opt-in boundary and adapter exclusion into this task.

**Size:** M
**Files:** `docs/research/gomad/<date>-syscall-boundary-decision.md` + `docs/research/gomad/README.md` index; `docs/research/gomad/GOMAD_PATCH_SIZE.md` (measurement section); `tools/gomad3/README.md`, `ARCHITECTURE.md`, `SPEC.md`, `CLI.md` for the opt-in options; `MILESTONES.md`; dated notes in `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.{13,14,15,16,17,18}.md`, `.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.{3,4,5}.md`, `.flow/specs/fn-154-gomad-simpler-virtual-network-with.md`.
**Touches:** [docs/research/gomad/**, tools/gomad3/README.md, tools/gomad3/ARCHITECTURE.md, tools/gomad3/SPEC.md, tools/gomad3/CLI.md, MILESTONES.md, .flow/tasks/fn-109-*, .flow/tasks/fn-110-*, .flow/specs/fn-154-*]

### Approach
- Measure with the fn-110.1 method (`docs/research/gomad/GOMAD_PATCH_SIZE.md:9-11, 290+`; `gomadtool patch-materialize`).
- Decision record options: move files and DNS to the syscall boundary, keep them at the current level, or abandon the syscall boundary; cite .3–.6 and .8 results, the escape inventory from .6, and size. Weigh whether .8's capability admission brings back per-library review in another form.
- Annotate fn-109 .13–.18, fn-110 .3–.5 and fn-154 with the decision's effect; fn-109.17, fn-109.18 and fn-110.3 are blocked with notes pointing at this task (flowctl has no cross-spec dependencies); unblock them as the decision directs.
- Document the boundary and adapter-exclusion options as experimental where users find them (README deterministic I/O L536, ARCHITECTURE transparent deterministic I/O L714-771, SPEC [INTERACTION.BOUNDARY]/[INTERACTION.ADAPTERS]).

### Investigation targets
**Required:**
- `docs/research/gomad/GOMAD_PATCH_SIZE.md`
- `docs/research/gomad/README.md`
- `tools/gomad3/ARCHITECTURE.md:714-771`

## Acceptance
- [ ] Size measurement against the fn-110.1 baseline is recorded with its method.
- [ ] The decision record exists, is indexed, states one of the three outcomes and cites R2–R7 and R10 evidence.
- [ ] fn-109 .13–.18, fn-110 .3–.5 and fn-154 carry dated notes on the decision's effect.
- [ ] Docs describe the opt-in boundary and adapter exclusion; MILESTONES.md fn-155 table lists all tasks.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
