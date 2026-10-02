---
satisfies: [R8]
---
# fn-112-gomad-determinism-assurance-and-test.7 Compare the filesystem and TCP models with the host OS on generated sequences

## Description
Model conformance (R8): run the same generated operation sequence against the in-memory model and natively against the host, and compare results.

**Size:** M
**Files:** new fixture programs `testdata/model_fs` and `testdata/model_net` under `tools/gomad3/internal/gomadtool/conformance/`, new `model_conformance_*_test.go` files under `tools/gomad3/runner/internal/execution/`
**Touches:** [tools/gomad3/runner/internal/execution/model_conformance_*_test.go, tools/gomad3/internal/gomadtool/conformance/testdata/model_fs/**, tools/gomad3/internal/gomadtool/conformance/testdata/model_net/**]

### Approach
- One fixture program per model takes a generator seed, performs a bounded sequence of standard-library `os` or `net` operations, and prints a canonical result log (operation, result, error class).
- The test runs the fixture twice: under Gomad through the existing toolchain-test launcher, and natively with stock Go in a scratch directory or on loopback. It compares logs after removing declared differences.
- Generator: the standard library's seeded source with fixed seeds and bounded lengths; no new dependency.
- Generate only operations the model declares supported. Error results compare by class, since darwin and linux errnos differ; declared differences are per platform, live in one Go table in the test source, and each has a test. Task 10 references that table from the docs.
- An undeclared difference is a model defect to record, or a new declared difference with a reason.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/runner/internal/execution/io_filesystem_toolchain_test.go:17-84` — launcher and hand-coded expectations
- `tools/gomad3/runner/internal/execution/io_network_toolchain_test.go:15-90` — network counterpart
- `tools/gomad3/internal/gomadtool/conformance/testdata/io_filesystem/main.go` — fixture shape
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/fs.go:149-201` — model entry points
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go:102-145` — listen and dial

**Optional** (reference as needed):
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/fs_test.go` — existing model tests

### Key context
- fn-109 tasks 14, 17, and 18 reshape the backend handles behind these models. The tests go through the standard library, so they survive that change; check for overlap before starting.
- Native loopback can be flaky; keep native sequences single-connection and bounded, with a logical timeout.
## Acceptance
- [ ] Filesystem and loopback TCP conformance tests run generated sequences for fixed seeds against the model and the host, on darwin/arm64
- [ ] Declared differences are listed in one Go table, per platform, each with a test
- [ ] A failing seed prints the seed and the shortest diverging prefix found
- [ ] Undeclared differences found are fixed or recorded as findings
- [ ] `make -C tools/gomad3 test-host` passes; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
