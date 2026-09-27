# Temporal integration for Gomad v3

This directory owns Temporal-specific use of the application-neutral Gomad v3
module. It contains the `test_dep` wrapper fixture, the bounded representative
Temporal qualification manifest, and outside-in tests of the root Make targets.

Run the wrapper contract and representative qualification with:

```sh
make gomad3-integration-test
make gomad3-qualification
```

The v3 manifest owns 16 tier 2 package workloads, the tier 3
`frontend-system-info` functional probe (`./tests/gomadfunctional`, guarded
capability mode), the tier 3 `user-timers-workflow` functional suite
(`./tests`, `TestUserTimersTestSuite`, closure mode under the `gomad` build tag
with the schema directory mounted read-only), and two fixed seeds. Gomad analyzes
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
The user-timers suite's expectation is `intermittent`: it runs the one-box
cluster to a successful exit with exact I/O evidence, and its same-seed choice
evidence reproduces on some seeds and repetitions but not all (seed 11 eight of
eight, seed 17 five of eight on 2026-09-27) because of the collector-side
divergence GOMAD_MILESTONES.md F5 records; the report keeps whichever of
`qualified`, `nondeterministic`, or `replay_divergence` the run produced.

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
