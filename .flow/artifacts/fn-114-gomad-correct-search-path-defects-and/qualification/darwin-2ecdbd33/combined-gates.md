# Combined Darwin gate on 2ecdbd33

The runtime candidate is `2ecdbd330bb5928f360739fdd80cb3eb0f0764e513a5cc208cd83714fff0a457`, on rebased HEAD `d635e23f00d926a43b942f25a9d05bd0ccb72025` with uncommitted implementation changes. `combined-source-hashes.json` binds current changed production, test, generated, and guide inputs. The candidate contains all fn-114 runtime changes and fn-112.5 stream isolation; no qualification disposition was weakened.

All components of the nested Makefile `test` gate passed. They ran once per stable input set, with completed components reused after the integration-only fixture fix:

| Component | Evidence |
| --- | --- |
| Harness | `remaining-test-tiers.log`: three packages passed |
| Toolchain | fn-112 task-5 `required-gates.log`; after the test-only cross-file alias fix, its full toolchain suite and scoped vet passed again |
| Interception | `remaining-test-tiers.log`: interception tier passed |
| Host | `remaining-test-tiers.log`: all 45 packages passed; Runner 229.482 s, CLI 116.080 s, execution 171.599 s |
| Overlay | `remaining-test-tiers.log`: all nine packages passed |
| Simulation | `remaining-test-tiers-final.log`: both the self-seeded root tests and integration transport selection passed; integration package 4.999 s |
| World race | `remaining-test-tiers-final.log`: all World packages passed with `-race` |
| Builder | `remaining-test-tiers-final.log`: builder tier passed |
| Live capability | `remaining-test-tiers-final.log`: live-capability tier passed |
| Runtime | fn-112 task-5 `required-gates.log`: runtime tier passed |
| Upstream | `remaining-test-tiers-final.log`: upstream compatibility tier passed |

The first remaining-tier run exited 2 after the simulation integration target was rejected by the new seeded Green Tea guard. Its raw builder lacked `GOEXPERIMENT=nogreenteagc`; the builder now owns that profile and filters an inherited experiment setting. Assertions and the existing simulation selection were preserved. The succeeding command reused the already green components and ran simulation through upstream, exiting 0. It did not rerun the entire literal `make test` command; the table records every reused component explicitly. The outer commands unset `GOEXPERIMENT`, so the nested host and simulation Make recipes establish their seeded profile themselves. No production or runtime input changed in this fixture repair.

Core: 7/7 workloads qualified and replayed, exit 0, 192.040 s. Smoke: 4/4 qualified and replayed, exit 0, 402.453 s. Both reports have no failed, unsupported, timed-out, infrastructure-error, or replay-diverged result and bind this build key. `core.json` and `smoke.json` here record commands, timings, and input hashes; the set reports are retained alongside them.

The representative set's first run stopped at its unchanged 2 GiB free-space bound after 1/28 workloads. This is infrastructure evidence, not a target regression or a pass. `representative-space-bound-report.json` and the first log retain that result. Only rebuildable Go caches were cleared; toolchains, prepared targets, campaigns, artifacts, and evidence remain. The unchanged manifest passed on retry with the already built CLI and toolchain: 28/28 workloads supported and completed, 56 exact replays, every expectation met, and zero failed, unsupported, timed-out, cancelled, infrastructure-error, or replay-diverged results. `representative-final.json` records exit 0 in 1,737.872 s; `representative-qualification-set.json` retains the report and its complete identities. The 112 unpruned artifacts retain 1,154,449,408 bytes on disk; private-target counting gives 11,372,320,431 bytes. `retained-measurement.json` records count-once bytes and excludes the four preserved campaigns from the first infrastructure-failed attempt. Task 12 already measured its Signal suite on build `2008ea81` and did not defer that measurement; those historical counts are not claimed for this new identity.

The current documentation check passed: 123 semantic identifiers, 25 original glossary terms, 30 commands, 71 documented flags accepted, and 17 links, with no errors. Native Linux qualification is unavailable, so fn-114 task 14 and R12 remain incomplete. The Linux/amd64 classic runtime cross-build passed, but proves compilation only. The existing root lint failure cannot analyse the nested module; prior `task-13/integrated-root-lint.log` records that environment limitation. Current scoped vet and diff checks passed.
