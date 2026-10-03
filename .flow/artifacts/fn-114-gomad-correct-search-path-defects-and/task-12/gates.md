# fn-114 task 12: gate evidence (darwin/arm64, 2026-10-02)

Host: darwin/arm64, stock go1.27.1 first on PATH
(`~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin`).
Base commit `c532a6e0611a60a861209e50adfe9583fb027ab0`. Toolchain build key
`2008ea81459afbd1ee019beb4a231968c1af8a24f4b46d4a8426b3188b1a9b87`, unchanged:
this task edits no runtime, overlay, or wire source. Load average 3.7 to 8.4
during the gates. linux/amd64 was not run: no native host in this session.

## Baseline

`baseline: green via handoff`: all tiers verified at 7c93c665 by task 11 on the
same build key; only `.flow` commits landed since, so no gate was rerun before
the edit.

## Red first

`../select-reduction/red-first.txt`: the three engine tests and the runner test
fail with the no-op skip disabled in `expandCandidate`, and the fixture's
reduced-versus-full comparison fails when a two-ready shape is listed as a
no-op with the reduction widened to it (mutations, reverted).

## After the edit, in this order

| command | exit | elapsed |
| --- | ---: | --- |
| `go test -tags test_dep -count=1 ./runner/internal/exploration/choice/` | 0 | 0:01 |
| `go test -tags test_dep -count=1 -run ChoiceExploration ./runner/` | 0 | 0:27 |
| `go test -tags test_dep -count=1 ./runner/internal/campaign/` (after the journal re-encoding) | 0 | 0:28 |
| `env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT GOMAD3_RUNTIME_REPRODUCTION_DIR=<scratch> go test -tags test_dep -count=1 -run 'TestRuntimeSearchFixtures$' ./internal/gomadtool/conformance` (retained to `../select-reduction/search-reproduction.json`) | 0 | 0:54 |
| `gofmt -l runner/ internal/gomadtool/conformance/` (no output); `go vet -tags test_dep ./runner/... ./internal/gomadtool/conformance/` | 0 | |
| `make -C tools/gomad3 validate` | 0 | 0:04 |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` (45 packages ok) | 0 | 3:17 |
| `make -C tools/gomad3 test-runtime` | 0 | 8:03 |
| `make -C tools/gomad3 test-toolchain overlay-test` | 0 | 0:32 |

Every gate passed on its first run. The spec's literal `go -C tools/gomad3 test
-tags test_dep ./runner/...` was not run with the stock go, which fails the
helper-target tests regardless of the change; `test-host` covers `./runner/...`,
`./artifact/...`, `./target/...`, and `./cmd/...` with the patched toolchain.
`make -C tools/gomad3 core-qualification` and `make gomad3-smoke-qualification`
were not run: the toolchain identity is unchanged and task 14 qualifies the
combined candidate.

## Signal suite run

`gomad explore --seeds 11 --choices --choice-bytes=64MiB --keep-successes=all
--success-limit=1 --success-bytes=1GiB --build-tag disable_grpc_modules
--build-tag gomad --build-tag test_dep --io-ro-mount <root>/schema=/go.temporal.io/server/schema
--execution-timeout 10m --overall-timeout 30m --parallel 1 --artifacts <scratch>
--working-dir <root> --json go-test ./tests -- -test.run=^TestSignalWorkflowTestSuiteChasm$`:
exit 0, 1:08 including the target build, one success retained. Counts in
`../select-reduction/`.

## Review round 1 fixes (same base, toolchain unchanged)

The review (claude-fable-5-1 at high) returned NEEDS_WORK with two findings:
the new counter stopped at the engine (`runner.ChoiceExplorationSummary`, its
projection, and the `explore`/`inspect` exploration lines lacked it), and the
explorer's shape list and the fixture's `noOp` rows were two hand-maintained
lists. Fixed by carrying `omitted_by_select_readiness` through the projection
and both lines, moving the list to `choice.NoOpSelectShapes`, and having the
fixture hold its proven rows and that list to the same set
(`requireNoOpShapesListed`, red-first in `../select-reduction/red-first.txt`).
No CLI output test pins those lines; `go test ./cmd/...` passed unchanged.

| command | exit | elapsed |
| --- | ---: | --- |
| `go test -tags test_dep -count=1 ./runner/internal/exploration/choice/ ./choice/ .` and `-run ChoiceExploration ./runner/` | 0 | 0:20 |
| `go test -tags test_dep -count=1 ./cmd/...` | 0 | 0:29 |
| `TestRuntimeSearchFixtures` (retained again to `../select-reduction/search-reproduction.json`; select-shape rows identical to the first run, timer identities differ with the rebuilt fixture binary) | 0 | 0:19 |
| `gofmt -l` (no output); `go vet -tags test_dep ./choice/ ./runner/... ./cmd/... ./internal/gomadtool/conformance/` | 0 | |
| `make -C tools/gomad3 validate` (build key unchanged) | 0 | 0:05 |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` (45 packages ok) | 0 | 2:42 |
| `make -C tools/gomad3 test-runtime` | 0 | 7:12 |

`test-toolchain` and `overlay-test` were not rerun: no runtime, overlay, or
generator input changed in this round.
