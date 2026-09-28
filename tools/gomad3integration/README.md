# Temporal integration for Gomad v3

This directory owns Temporal-specific use of the application-neutral Gomad v3
module. It contains the `test_dep` wrapper fixture, the bounded representative
Temporal qualification manifest, and outside-in tests of the root Make targets.

Run the wrapper contract and representative qualification with:

```sh
make gomad3-integration-test
make gomad3-qualification
```

A full run retains about 11 GiB of Campaigns under
`tools/gomad3/.toolchain/temporal-qualification`. Where disk is short,
`make gomad3-qualification GOMAD3_QUALIFICATION_PRUNE=1` deletes each qualified
seed's Campaigns after its exact replay and keeps its qualification report; the
set report then marks those seeds `artifacts_pruned` (see
`tools/gomad3/README.md`).

The v3 manifest owns 15 tier 2 package workloads, the tier 3
`frontend-system-info` functional probe (`./tests/gomadfunctional`, guarded
capability mode), the tier 3 `user-timers-workflow` functional suite
(`./tests`, `TestUserTimersTestSuite`, closure mode under the `gomad` build tag
with the schema directory mounted read-only), the tier 3
`activity-batch-cancel-boundary` functional suite (`./tests`,
`TestActivityAPIBatchCancelClientTestSuite`, built and mounted the same way),
the ten tier 3 `functional-*` suites of the GOMAD_MILESTONES.md F6 slice
(`./tests` activity, cancel, child workflow, continue-as-new, cron, query,
CHASM signal, timer, update, and workflow suites, built and mounted the same
way), and two fixed seeds. Gomad analyzes
the complete corpus first, executes only supported workloads, retains and
replays every successful repetition with bounded choice coverage, and writes a
path-free `gomad3.qualification-set-report/v1` to
`tools/gomad3/.toolchain/temporal-qualification-set.json`. Expected unsupported
boundaries are exact analyzer dispositions, not claims of support; the report
keeps actual supported and unsupported counts separate from expectation
matching.
The probe's linux/amd64 expectation is `unrepeatable`: it runs to a successful
exit under guarded mode, but its same-seed evidence does not yet reproduce in
every run, and the report records `nondeterministic` or `replay_divergence`
as observed. Its darwin/arm64 expectation is `qualified`: since darwin targets
re-execute with ASLR disabled, both seeds reproduce their evidence across
repetitions and replay with exact choice replay.
The user-timers and activity batch cancel suites' darwin/arm64 expectation is
`qualified`: since the runtime greys the scheduler structures at mark start
(GOMAD_MILESTONES.md F5), both seeds reproduce their evidence across
repetitions and replay with exact choice replay, and all 28 darwin workloads
are supported. Their linux/amd64 expectation stays `intermittent`, as last
measured there before that runtime change: the suites run the one-box cluster
to a successful exit with exact I/O evidence, and the report keeps whichever of
`qualified`, `nondeterministic`, or `replay_divergence` the run produced until
a linux/amd64 run re-measures them.
The F6 slice follows the same split. On darwin/arm64 all ten suites qualify
on both seeds with exact replay, including four fresh repetitions of the six
suites that had diverged on linux. Their linux/amd64 expectation is
`intermittent` because linux last measured them before the environment-filter
and mark-start greying fixes, when six of twenty seed runs qualified; those
fixes are platform-neutral, but until a linux run re-measures the slice the
manifest does not claim `qualified` there, and each of those expectations names
`GOMAD_MILESTONES.md#f6-a-package-level-functional-slice` as its `finding`. The
other non-qualified expectations name their milestone sections the same way.
Each slice suite inherits the
two-minute `run_timeout` (the longest measured darwin execution took 7 s of
wall time; the 20-minute overall timeout covers the cold `./tests` build) and
requires the modeled probes every measured run observed:
`stdlib.os.openfile` (the read-only schema mount lookup), `stdlib.os.getwd`,
`stdlib.os.newfile`, and `stdlib.net.interfaces`, so a boundary change that
silently drops one of those operations fails the set. The set rejects an
unknown `required_probes` name when it loads the manifest.
Nine package workloads build with the `gomad` tag, which cuts the cloud
credential providers that import `os/exec`. On darwin/arm64 they and
`temporal-cache-concurrent` qualify through the `temporal-leaf-*` packs; on
linux/amd64 their expectation is the amd64 xxhash assembly, which no linux
pack admits for these closures.

## The generated `./tests` manifest

`qualification/tests.json` qualifies every top-level test of `./tests`, one
tier 3 workload per test. It is generated, never edited:
`make -C tools/gomad3 generate` (or `tests-qualification-generate`) lists the
tests `go test -list` reports under the `disable_grpc_modules`, `gomad`, and
`test_dep` tags on darwin/arm64 and linux/amd64 and applies
`qualification/tests.generator.json`, which holds the manifest and workload
defaults (the F6 slice's closure mode, tags, schema mount, 64 MiB choices,
retention, and timeouts), per-test overrides, and exclusions.
`make -C tools/gomad3 validate`, which CI runs, fails when the checked-in
manifest is stale relative to `./tests` or the spec.

A new test therefore lands expecting `qualified` on both platforms and fails
the set when it is not. An override changes a test's `expectation`,
`platform_expectations`, or `required_probes`; a failure expectation names its
`finding` as elsewhere. A test leaves the set only by an exclusion with an
`owner`, a `date` (`YYYY-MM-DD`), and a `reason`; an exclusion missing any of
them, or an override or exclusion naming a test `./tests` no longer declares,
is refused. `temporal.json` stays the representative corpus.

```sh
make gomad3-tests-qualification
```

regenerates the manifest and runs the set with
`GOMAD3_QUALIFICATION_PRUNE=1`, writing
`tools/gomad3/.toolchain/tests-qualification-set.json`.

## Capability closure of `./tests`

The functional test package itself has a closed capability closure on
linux/amd64 and darwin/arm64 under the `gomad` build tag:

```sh
tools/gomad3/.bin/gomad analyze --capability-mode=closure --format=json \
  --build-tag disable_grpc_modules --build-tag gomad --build-tag test_dep \
  go-test ./tests
```

reports `supported` with zero blockers. The tag drops the server's host-only
providers (signal handlers, the persistence password command, the cloud
archivers, AWS request signing, ringpop membership, the auto-scaled-workers
component, and the MySQL and PostgreSQL drivers) and the fx, Temporal SDK,
otel/sdk, and gRPC adapters remove the remaining `os/signal`, `os/user`, and
`syscall` imports; `temporal-functional-tests-linux-amd64` admits the assembly,
linknames, and procfs reads that stay. On darwin/arm64,
`temporal-functional-compute-darwin-arm64` admits the arm64 assembly and
`temporal-functional-tests-darwin-arm64` admits the Prometheus client's darwin
process-collector `syscall` and `golang.org/x/sys/unix` imports. The manifest's tier 2 suites keep their
untagged expectations because their darwin/arm64 counterparts have not been
observed; GOMAD_MILESTONES.md F4 records the tagged linux results per suite.
