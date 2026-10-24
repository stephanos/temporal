---
satisfies: [R12, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.28 Bind all simulation integration fixtures to their compiled role

## Description
Touches: tools/gomad/rust/wasmhost/tests/linked_simulation_corpus.rs.

Repair actual C8 diagnostic command0045 (101,2PASS4FAIL): prepared_cluster_fueled and live_peer_cluster set conditions only under determinism-fixture and retain Ordinary in diagnostic builds. Follow existing positive integration fixtures by unconditionally selecting linked::Conditions::compiled(), removing only obsolete conditional unused_mut/role gates. Preserve production admission, WAT, bounds, assertions, watchdog16 repeats/200ms and intentional incompatible-role tests.

Diagnosis/inventory/exact five static-plan Cargo recipes: /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/cycle8-diagnostic-integration-fixture-diagnosis.{json,md}. Audit covers15 integration targets. Prior .26 library/streaming proof does not cover this target; this task must execute COMPLETE target discovery/run matrices for all five compiled profiles, not only --lib or selected tests. Parent .13 retains full96/final145.

## Acceptance
- [x] Retain actual original diagnostic RED and a focused unchanged-source reproduction before the two-helper fix. No production admission changes.
- [x] Execute complete five-profile Cargo executable discovery/list/ignored-list/run matrices using the actual frozen reference-helper profile recipes and release/locked/offline/no-default-features context. Run every discovered library/integration executable in each profile, continue collecting after failures, retain separate doctests and original cfg exclusions/ignored dispositions. Include allocation_receiving embedded library tests and every linked_simulation_corpus test. No --lib/target-only substitute.
- [x] Full current driver suite and mandatory worker/conductor merge gates pass; independently reviewed on a green tree with SHIP. Preserve all ignored status, bounds, production semantic/engine/host checks and deliberately incompatible diagnostic_roles fixtures.
- [x] Integrate minimal reviewed source change and retain full raw profile/target commands, numeric exits, inventories and actual source identities. Full frozen qualification remains pending under .13.
## Done summary
Bound both simulation integration fixture helpers to Conditions::compiled(); removed only obsolete cfg/unused-mut gates (2 insertions,6 deletions). Actual original/focused RED retained. Complete clean five-profile matrix ran93 discovered executables (5x17 wasmhost plus8 default Gomad) with list/ignored-list/run, discoveries and doctests all0; original ignored status/assertions/watchdogs preserved. Worker, separate conductor and combined canonical gates passed;589 driver tests passed. Worker4795738 and ef836cf trees identical; backupref retained. Integrated33121111dd. Three independent Codex gpt-6.1-sol/high draws SHIP, zero findings, same-family disclosed. Durable review: /home/agent/wt/evidence/fn232-simulation-fixtures/formal-review/receipt.json. Full qualification remains pending under .13.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 33121111dd7e5307128496a3130522193c1c0e64
- Tests: Complete clean five-profile plus default Gomad matrix: /home/agent/wt/evidence/fn232-simulation-fixtures/matrix-clean.json, make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism, conductor: /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/conductor-repair28-gate.json, combined: /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/conductor-repaircombined-gate.json, SHIP: /home/agent/wt/evidence/fn232-simulation-fixtures/formal-review/receipt.json
- PRs: