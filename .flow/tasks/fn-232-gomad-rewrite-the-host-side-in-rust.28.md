---
satisfies: [R12, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.28 Bind all simulation integration fixtures to their compiled role

## Description
Touches: tools/gomad/rust/wasmhost/tests/linked_simulation_corpus.rs.

Repair actual C8 diagnostic command0045 (101,2PASS4FAIL): prepared_cluster_fueled and live_peer_cluster set conditions only under determinism-fixture and retain Ordinary in diagnostic builds. Follow existing positive integration fixtures by unconditionally selecting linked::Conditions::compiled(), removing only obsolete conditional unused_mut/role gates. Preserve production admission, WAT, bounds, assertions, watchdog16 repeats/200ms and intentional incompatible-role tests.

Diagnosis/inventory/exact five static-plan Cargo recipes: /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/cycle8-diagnostic-integration-fixture-diagnosis.{json,md}. Audit covers15 integration targets. Prior .26 library/streaming proof does not cover this target; this task must execute COMPLETE target discovery/run matrices for all five compiled profiles, not only --lib or selected tests. Parent .13 retains full96/final145.

## Acceptance
- [ ] Retain actual original diagnostic RED and a focused unchanged-source reproduction before the two-helper fix. No production admission changes.
- [ ] Execute complete five-profile Cargo executable discovery/list/ignored-list/run matrices using the actual frozen reference-helper profile recipes and release/locked/offline/no-default-features context. Run every discovered library/integration executable in each profile, continue collecting after failures, retain separate doctests and original cfg exclusions/ignored dispositions. Include allocation_receiving embedded library tests and every linked_simulation_corpus test. No --lib/target-only substitute.
- [ ] Full current driver suite and mandatory worker/conductor merge gates pass; independently reviewed on a green tree with SHIP. Preserve all ignored status, bounds, production semantic/engine/host checks and deliberately incompatible diagnostic_roles fixtures.
- [ ] Integrate minimal reviewed source change and retain full raw profile/target commands, numeric exits, inventories and actual source identities. Full frozen qualification remains pending under .13.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
