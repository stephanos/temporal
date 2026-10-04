# Task 13 acceptance remains open after the D26 pin repair

Task 13's verified 22-file time-wire source checkpoint is committed at
`58b718565044ab3bc3385d3323ee908a6d54328e`. Its historical reports remain
unchanged. The inherited first-party bridge-pin mismatch they found was repaired
separately by its D26 owner in
`5350185a3601921c0a5f9ba07e1f05bdad7df81f`; it is no longer a current source
blocker. See the retained D26 `task-31/bridge-pin-repair/conductor-checkpoint.md`
and `conductor-verification.json`. The repair grants no generic I/O capability
and changes no runtime bridge source or task-13 codec behavior.

The conductor freshly verified the current bridge SHA-256
`211c01f57125ba62115b1ffce5d2479d3c22116d51a41aefcfb1a576e8b393a9`,
the exact policy SHA-256
`6e8e072c7d47aa73f7e9f05d938ebe625e0e6ba56f317381978e539b30342bd8`,
and the rejection-test SHA-256
`88a4b3046dc1d9103f18c1206c817f05bdec5d49e875a2a74297fc8c9b41b728`.
Policy binds that bridge and the ordered Advance, Current and TakeArrivals
directives. From `tools/gomad3`, these fresh stock Go1.27.1 commands exited 0:

```text
go test -count=1 -tags test_dep ./target -run '^TestBuiltInSimulationLinknamesPinCurrentFirstPartySources$'
go test -count=1 -tags test_dep ./target/internal/capabilitypolicy
```

The exact executable was
`/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`;
GOWORK=off, GOTOOLCHAIN=local, GOENV=off, GOFLAGS empty and GOMAXPROCS=2,
with GOMADSEED and GOMAD3_CHILD_SEED unset. Package times were 0.014s and
0.011s; the complete batch exited 0. An initial path check used repository-relative
paths from the nested-module directory and stopped before either test; the
corrected path check and both tests then succeeded.

Task 13 still requires source-bound patched-toolchain rebuild, runtime vectors,
real process transport, gomad3sim execution and full quiescence/nosplit checks
on native darwin/arm64 and linux/amd64. Exact commands remain in
`task-13/evidence.json`'s `native_commands` and the task's Quick section.
This linux/arm64 host supplies neither qualification. The earlier empty-range
formal review did not accept the committed candidate. Keep task 13 blocked
and R7 open; resolving the pin mismatch closes none of those gates.

Commit verified progress under MILESTONES instruction 5, preserve historical
evidence and unrelated changes, and push only when authorized.
