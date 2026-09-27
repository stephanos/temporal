---
satisfies: [R5]
---
# fn-95-gomad-f1-restore-the-checkout-on.4 Record the F1 darwin/arm64 result in the milestone status

## Description
Update the F1 status in `.plans/GOMAD_MILESTONES.md` with the darwin results and any fixes.

## Acceptance
- F1 status names the darwin/arm64 outcome

## Done summary
F1's Status in `.plans/GOMAD_MILESTONES.md` now has a dated 2026-09-27 paragraph with the darwin/arm64 outcome: the toolchain build and doctor result, all ten `make -C tools/gomad3 test` tiers with the five host-tier repairs (45e6788d97), `make gomad3-integration-test`, the core set (5/5/0, all `choice_replay_exact`), the v041 fixture and pack repair (7ea97c052e), and the host clang prerequisite. The wrong "Nothing references a v041 fixture" sentence now says the Makefile's non-Linux branch qualifies it, and the Work tracking row for F1 is `done`, because R1-R5 are met. The runner fake-preparer fix has not been rerun on linux/amd64, and the status says so.

stage: impl-review - ran (triage_skip SHIP: docs-only diff)
## Evidence
- Commits: 66a8816d914a5c17ae6c9268cc1eb6255046ec4f
- Tests: baseline: none (spec defines no Quick commands), flowctl gate classify: docs-only tier-B (.plans/GOMAD_MILESTONES.md only)
- PRs: