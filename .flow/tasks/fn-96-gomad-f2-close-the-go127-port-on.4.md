---
satisfies: [R3, R4]
---
# fn-96-gomad-f2-close-the-go127-port-on.4 Record the F2 darwin result in the milestone status

## Description
Update F2 status.

## Acceptance
- F2 status updated

## Done summary
Recorded the F2 darwin/arm64 outcome in `.plans/GOMAD_MILESTONES.md`. F2's Status now has a dated 2026-09-27 section covering five things. First, the dossier result: every gate passes except the root-only host-clock-escape, and the clock-audit fixture and `.d` path were restored in 68d36aadfe. Second, the approved boundary diff digest `sha256:86f18fc8cda31fe234d345f70384e8d5ae94e9cbb883beb5f8399e73f6d4798f`, with the reviewed entries and a note that it is the `GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256` repository variable value (setting it needs a repo admin). Third, the DTrace audit did not run, and `sudo make -C tools/gomad3 clock-audit` completes it. Fourth, the darwin Temporal corpus result: 5 supported, 11 unsupported with matching blockers, 2 tier-3 intermittent, 0 infrastructure errors. Fifth, the three a8777f5d73 fixes and the probe's darwin expectation change to `intermittent`.

F4's status no longer says the darwin packs and adapter pins are unverified. It now records that the darwin `./tests` closure is closed at a8777f5d73. The tracking table marks F2 done, noting that the clock audit needs a root run. `.plans/GOMAD3_NEXT.md` has no statement this makes false, so it is unchanged.

stage: impl-review - ran [2026-09-27] triage_skip SHIP (docs-only .plans/GOMAD_MILESTONES.md)
## Evidence
- Commits: 80b7bc4736d44e104af5729ada23b3f3cbe6b9e3
- Tests: baseline: none (spec defines no Quick commands), flowctl gate classify: docs-only tier-B (.plans/GOMAD_MILESTONES.md)
- PRs: