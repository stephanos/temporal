# Gates for the fn-112.5 review fixes (linux/arm64 development harness, harness ON)

Toolchain key (harness on) `8af6857b…`; base `c2fd6d2f0`. Compared with the harness baseline
(BASELINE.md, gomad-next at bce040728).

| Gate | Result | vs baseline |
| --- | --- | --- |
| `make validate` | exit 0 | same |
| `make test-toolchain` | FAIL only `TestPatchedRuntimeHostClockReferencesAreReviewed` (no linux/arm64 clock inventory) | same; a scratch run of the same inventory restricted to darwin/arm64 + linux/amd64 scanned 25 keys with 0 problems |
| `make test-runtime` | pass (22m22s) | same |
| `make overlay-test` | pass | same |
| `make test-simulation` | pass | same |
| `make test-live-capability` | pass | same |
| `make intercept-test` | FAIL only `interception-report-unsupported-platform` | same |
| `GOFLAGS='-tags=test_dep -count=1' make test-host` | baseline failures only, plus `TestRunHardCrashesAndReapsSimulationNodeProcess` once (exit 49 under load; passed 5/5 on rerun) | first run without the Makefile `GOEXPERIMENT` fix failed every test that runs its own test binary as a seeded target (the new Green Tea refusal); fixed in `test-host` |
| focused: `TestDiagnosticDrawCheckStopsHostTimedSeededDraw`, `TestPatchedRuntimeSeededDrawReferencesAreReviewed`, `TestSeededDrawInventoryRejectsUnclassifiedReference` | pass | |

Fixture projection (fn-110 task-2 `fixture-compare.py`, seeds 0/1/7/42 twice each plus disabled,
all built with `GOEXPERIMENT=nogreenteagc`), toolchain `275f92ed` (c2fd6d2f0) vs `8af6857b`: 125
cases, every seeded case identical output/status and same-seed repeatable in both builds; the only
difference is `select_readiness blocking-two-ready disabled` (unseeded upstream select order, which
varies run to run). Choice records are byte-identical across builds for 40/104 traced cases;
caller-site text offsets differ between builds, so this is informational.

Owed on darwin/arm64 and linux/amd64: `make -C tools/gomad3 validate test-toolchain test-runtime
overlay-test test-simulation test-live-capability intercept-test` and
`GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`.
