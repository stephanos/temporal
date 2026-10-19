# Temporal upstream candidates

This inventory groups changes outside `tools/gomad` relative to recorded upstream
commit `48a1e5ff025e55b3d4d0f44247423d4c279ff636`. It describes branch contents,
including changes that predate the latest fn-187/fn-189 work.

“Candidate” means worth proposing separately, not verified merge-ready. Newer
upstream `main` has not been checked for equivalent fixes. No upstream merge,
commit, push or pull request is authorized by this inventory.

## Server behavior

| # | Change | Effect | Upstream recommendation |
| --- | --- | --- | --- |
| 1 | [Eager workflow-task retries](service/history/api/startworkflow/api.go) | Stops retried starts from returning an already-expired eager workflow task. | Candidate; add a focused regression test. |
| 2 | [Activity timer ordering](service/history/workflow/timer_sequence.go) | Makes ScheduleToClose win when activity deadlines coincide. Includes a regression test. | Candidate; confirm intended timeout precedence. |
| 3 | [Workflow update ordering](service/history/workflow/update/registry.go), [admission sequence](service/history/workflow/update/update.go) | Adds an admission sequence to break equal-timestamp ties. | Candidate; review process-wide sequence ownership. |
| 4 | [Matching metadata persistence](service/matching/db.go) | Uses an explicit dirty flag so same-clock-tick changes aren't missed. | Candidate; add a clock-tie regression test. |
| 5 | [Enhanced DescribeTaskQueue cache](service/matching/matching_engine.go) | Includes report flags in cache keys and prevents double-counting cached results during partial-cache refreshes. Includes regression tests. | Strong candidate for a standalone fix. |
| 6 | [Tracing initialization](temporal/fx.go) | Skips resource detection when no span processors are configured. Includes a test. | Review separately for other resource consumers. |

## Tests and test infrastructure

| # | Change | Effect | Upstream recommendation |
| --- | --- | --- | --- |
| 7 | [Nexus endpoint isolation](tests/nexus_api_test.go) | Generates endpoint names per subtest invocation, avoiding parallel collisions while retaining endpoint assertions. | Candidate. |
| 8 | [Nexus reset synchronization](tests/nexus_workflow_test.go) | Uses UpdateWithStart and explicit history barriers to establish the ordering asserted by the test. | Candidate. |
| 9 | [Activity fairness readiness](tests/priority_fairness_test.go) | Waits until all activity tasks reach matching before measuring fairness. | Candidate. |
| 10 | [Schedule migration idempotency](tests/schedule_migration_test.go) | Holds the migration task and explicitly checks pending, repeated and completed migration behavior. | Review separately; expands and clarifies the tested contract. |
| 11 | [Workflow-task heartbeat test](tests/workflow_task_test.go) | Sets an explicit heartbeat deadline and deliberately exercises two expirations. | Review timing/sleep assumptions first. |
| 12 | [Versioning poller readiness](tests/versioning_test.go) | Waits for both workflow and activity pollers before checking report flags. | Bundle with #5. |
| 13 | [In-memory SQLite test option](tests/testcore/test_env.go), [cluster setup](tests/testcore/test_cluster.go) | Adds dedicated SQLite-backed environments and persistence-option plumbing; also adds optional pprof disabling. | Discuss as a general test-harness improvement. |
| 14 | [Persistent test clusters](tests/testcore/persistent_cluster.go) | Preserves file-backed SQLite across restarts, with stable addresses and initialize/reopen validation. | Discuss a standalone crash/recovery testing API. |
| 15 | [SQL plugin import decoupling](common/persistence/persistence-tests/setup.go), [server registration checks](cmd/server/main_test.go) | Uses neutral plugin-name constants instead of importing concrete drivers; adds server registration checks. | Extract generic cleanup; keep Gomad-specific selection separate. |

## Platform support and repository tooling

| # | Change | Effect | Upstream recommendation |
| --- | --- | --- | --- |
| 16 | [WASI SQLite driver](common/persistence/sql/sqlplugin/sqlite/driver_wasi.go) | Adds ncruces SQLite support, timestamp compatibility, memory-database mapping and connection reuse handling. | Separate platform-support proposal. |
| 17 | [Inspection hooks](common/gomaddebug/README.md) | Adds debug registration and shard/mutable-state snapshot hooks. | Design discussion before upstreaming. |
| 18 | Build-tagged platform alternatives in `common/`, `service/worker/` and `temporal/` | Splits membership, signals, host IP, cloud providers, password commands, connection errno handling and worker-component wiring. | Keep downstream unless upstream accepts the Gomad/platform integration. |
| 19 | Gomad test selections in `tests/` | Adds frontend fixtures, excludes unsupported suites under `gomad`, and moves shared deployment helpers. | Keep exclusions/fixtures downstream; consider helper cleanup independently. |
| 20 | [Lint routing](cmd/tools/lintcode/main.go) | Adds multi-module/native/WASM routing, policy tests and mixedbrain lint CI. | Extract generic multi-module improvements; separate Gomad exceptions. |
| 21 | [Dependency changes](go.mod), [mixedbrain pins](tests/mixedbrain/go.mod) | Adds WASI SQLite dependencies and updates Prometheus/`golang.org/x` dependencies, including mixedbrain pins. | Split routine upgrades from platform dependencies. |
| 22 | Gomad CI, documentation and housekeeping | Adds workflows, Makefile targets, agent guidance, migration/roadmap/research files and formatter/ignore rules. | Keep Gomad-specific material downstream or move it with Gomad. |

## Suggested first batch

Start with **#1-5, #7-9 and #12**, split into focused fixes. Keep each regression
test with its corresponding fix. Review #6, #10 and #11 separately before
including them. Treat the remaining rows as independent API, platform or tooling
decisions rather than prerequisites for the bug-fix batch.

The working tree also contains uncommitted lint-policy test, milestone and Flow
planning changes. Keep those distinct from the committed branch delta when
preparing patches. This document is a triage inventory, not a completion receipt
for fn-189 or fn-187.
