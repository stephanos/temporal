# fn-245-gomad-typed-boundaries-for-the-rust-host.7 One cluster-run entry point for scenario callers

## Description
Complexity review item 3 (owner, 2026-10-24). Five callers each assemble `linked_cluster` `Request`/`BrokerConfig`/`WorldLimits`/`NetworkConfig` and call `prepare` → `execute_prepared`: `fixture_cluster`, `campaign/scenario`, `live_cluster`, `replay/cluster`, `stress/cluster` (plus the `fixture_*`/`live_*` wiring, ~4.2k lines at 2026-10-24). Add one deep module that takes a scenario description and returns the typed outcome and evidence, with callers supplying only what differs (replay tape, stress jitter, crash groups, cleanup mode). Delete the duplicated assembly. Measure before choosing the seam; if fewer than three callers share real logic, record why and stop.

## Acceptance
- [ ] All cluster callers go through the shared entry point, or a recorded reason explains why one does not.
- [ ] Existing cluster, replay, stress and fixture e2e tests pass unchanged; before/after line counts recorded with revisions in `.flow/artifacts/fn-245/counts.md`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
