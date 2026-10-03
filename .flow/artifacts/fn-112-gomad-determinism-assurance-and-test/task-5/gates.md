# Gates (linux/arm64 development harness, 2026-10-03)

Working tree: task-5 changes on top of `d5330bb77`, harness on, build key `275f92ed…`.
Host load from other agents was 15-24 on 12 cores during these runs.

| Command (from `tools/gomad3`) | Exit | Time | Result |
| --- | --- | --- | --- |
| `make toolchain` | 0 | 240 s | key `275f92ed3f906dae5ce5563d0d9c693b511350316c9dfe55bc7c5434edb7d7c9` |
| `make validate` | 0 | 4 s | pass |
| `make test-toolchain` | 2 | 10 s | only `TestPatchedRuntimeHostClockReferencesAreReviewed` fails (baseline harness failure: no linux/arm64 clock inventory); both draw-inventory tests pass |
| `make overlay-test` | 0 | 31 s | pass (includes `TestDiagnosticTrace`) |
| `make test-runtime` (first run) | 2 | 500 s | `scheduler-seed-0-repeatable-83` killed with status 0 and no output (watchdog of a `go run` with a 1-minute bound) under load ~24 |
| direct loop of the `scheduler` fixture binary, `GOMADSEED=0`, 1000 runs | 0 | 150 s | 1000/1000 identical output, no hang or non-zero exit |
| `make test-runtime` (rerun) | 0 | 1097 s | `gomad3 runtime tier passed` (includes `runq_user_choice` with the tightened bounds) |
| `GOFLAGS='-tags=test_dep -count=1' make test-host` | 2 | 438 s | exactly the 21 baseline failures in BASELINE.md, no new failure |
| `.toolchain/bin/go test -tags test_dep -run SeededDraw ./toolchain` | 0 | 5 s | pass |
| `go test -tags test_dep -count=10 -run 'TestDiagnosticDrawCheckStopsHostTimedSeededDraw\|TestDiagnosticLauncherLocalizesInjectedDraw' ./runner/internal/execution` | 0 | 14 s | 10/10 pass each |
| `go vet` on `./toolchain`, `./runner/internal/execution`, `./internal/gomadtool/conformance`; `gofmt -l` | 0 | | clean |

Not run: `core-qualification-set` and the smoke set. Their acceptance item applies only if a site
was rerouted, and none was.

Owed on darwin/arm64 and linux/amd64: `make -C tools/gomad3 validate test-toolchain test-runtime
overlay-test` and `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`.
| `make validate` with the harness off | 2 | | fails only on `syscall.Dup2` in the runner (harness limitation); passes with the harness on |
