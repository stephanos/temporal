---
satisfies: [R4]
---
# fn-242-gomad-shrink-the-qualification.6 Remove duplicated measurement checking

## Description
Using the .1 inventory, remove logic in tools/gomad/rust/gomad/src/measurement/ (about 25k lines) that re-checks what qualify, report and the Result/receipt readers already prove; keep producers (host-owner-profile, runtime-cost-profile, stress, warm, cluster, retention) and their own checks. Each removal cites its inventory row. Fixture tests stay green; no bound or assertion weakened.

**Touches:** tools/gomad/rust/gomad/src/measurement/**

## Acceptance
- [ ] Every removed check cites an inventory row marked duplicate
- [ ] Producers and their non-duplicate checks unchanged in behavior; existing fixture tests pass
- [ ] Line count of measurement/ recorded before and after

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
