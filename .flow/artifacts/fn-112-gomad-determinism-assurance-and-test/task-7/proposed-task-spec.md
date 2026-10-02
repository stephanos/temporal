---
satisfies: [R8]
---
# fn-112-gomad-determinism-assurance-and-test.7 Compare the filesystem and TCP models with the host OS on generated sequences

## Description
Model conformance (R8): run the same generated operation sequence against the in-memory model and natively against the host, and compare results.

**Size:** M
**Files:** new fixture programs `testdata/model_fs` and `testdata/model_net` under `tools/gomad3/internal/gomadtool/conformance/`, new `model_conformance_*_test.go` files under `tools/gomad3/runner/internal/execution/`
**Touches:** [tools/gomad3/runner/internal/execution/model_conformance*_test.go, tools/gomad3/internal/gomadtool/conformance/testdata/model_fs/**, tools/gomad3/internal/gomadtool/conformance/testdata/model_net/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/fs.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/process_volume.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/*_test.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/libc.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/anonymous_test.go, tools/gomad3/toolchain/runtime/overlay/src/os/gomad.go, tools/gomad3/toolchain/runtime/overlay/src/os/*_test.go, tools/gomad3/toolchain/version/version.json, tools/gomad3/toolchain/version/generated.go, tools/gomad3/version_generated.mk, tools/gomad3/deterministicio/boundary/upgrade-go1.27.1.md]

### Approach
- One fixture program per model takes a generator seed, performs a bounded sequence of standard-library `os` or `net` operations, and prints a canonical result log (operation, result, error class).
- The test runs the fixture twice: under Gomad through the existing toolchain-test launcher, and natively with stock Go in a scratch directory or on loopback. It compares logs after removing declared differences.
- Generator: the standard library's seeded source with fixed seeds and bounded lengths; no new dependency.
- Generate only operations the model declares supported. Error results compare by class, since darwin and linux errnos differ; declared differences are per platform, live in one Go table in the test source, and each has a test. Task 10 references that table from the docs.
- An undeclared difference is a model defect to record, or a new declared difference with a reason.

### Reproduced model corrections (2026-10-02)

The generated comparisons exposed three defects on baseline build
`6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee`:
closed modeled files lose `os.ErrClosed`, closed modeled TCP connections lose
`net.ErrClosed`, and renaming a file to itself deletes it. Retain baseline logs
under task-7 artifacts and fix these defects without treating them as declared
differences. This is the task's existing model-defect correction acceptance.

- Give closed filesystem handles a distinct internal marker that preserves backend
  `EBADF` handling. Translate it to `os.ErrClosed` at the os adapter after recording
  raw transcript results; preserve wrong-access `EBADF` and cover process handles.
- Reuse the standard network closed sentinel, preserving error text and transcript
  classification while restoring `errors.Is(err, net.ErrClosed)`.
- Make valid self-renames a no-op, preserving source-existence, read-only, mount,
  parent, data and metadata invariants. Do not widen capability admissions.
- Preserve comments and baseline sources/evidence. The conductor serializes these
  overlay edits; no other runtime task runs concurrently. No patch hunks or schemas
  change. The overlay allowlist gains exactly two test-only entries:
  `src/os/gomad_model_conformance_test.go` and
  `src/internal/gomadfs/export_test.go`, with required generated version consumers.
  The external os tests avoid the testing/os import cycle; the internal test-only
  export constructs a closed process handle without starting a process backend.
- Check at least 8 GiB free before rebuilding and preserve the baseline and active
  builds. Record each new identity; never relabel old artifacts. The initial
  corrected build `d15ad896e542b9f64381c2f184567355bac8ffd62ae2347501fe8fcdd96f6eba`
  exposed a libc consumer regression in a captured-descriptor close/read probe:
  public `os.ErrClosed` was converted to `EIO`. Translate that sentinel to `EBADF`
  in `gomadio.libcErrno`, preserve all other errno extraction, and retain the
  deterministic captured-descriptor regression in `gomadio/anonymous_test.go`.
  This reproduced regression requires a second rebuild and refreshed final gates;
  the first corrected identity is intermediate evidence, not final qualification.
  The second rebuild may start with at least 6 GiB free: the first build measured
  about 1.5 GiB peak additional use and more than 7 GiB remains. Record fresh disk
  evidence after the prior gate drains; this measured exception avoids deleting
  any baseline, active build, or shared cache. The same measured >=6 GiB floor
  applies to the third corrective rebuild below.
- The unseeded overlay gate on `d49ef0309636e2301c46cfb36cda8a9b21323d87601c93c429260b3eefd4e37d`
  exposed incorrect directory-read expectations and modeled behavior. Pinned stock
  Go returns `PathError` wrapping `poll.ErrFileClosing` for closed `ReadDir`,
  `Readdir`, and `Readdirnames`; its operation is empty on Darwin and `readdirent`
  on Linux. `Chdir` retains `os.ErrClosed`. Correct only the closed-marker branch
  of `gomadFileReaddir` after transcript recording; preserve nonclosed paths.
  Retain native/seeded probes, assert wrapper/path/operation/sentinel identity, and
  pass both native and seeded focused tests before the third rebuild. Keep the
  d49 runtime success and overlay failure as intermediate evidence.
- After focused regressions, run `make -C tools/gomad3 generate validate`, rebuild
  the toolchain, then `test-host test-runtime overlay-test` with `test_dep` enabled.
  Root lint and focused vet remain required; native Linux remains unverified here.

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
- [ ] Closed-file and closed-network public sentinel identities match stock Go; wrong-access filesystem errors retain `EBADF` without `os.ErrClosed`; process handles and backend errno consumers, including libc captured-descriptor close/read, remain covered; closed directory reads match stock PathError/poll.ErrFileClosing while Chdir retains os.ErrClosed
- [ ] File/directory self-renames preserve contents and metadata, while source-existence and read-only/mount checks remain enforced
- [ ] Baseline failure evidence is retained; the corrected overlay rebuilds under a new identity and `generate validate test-host test-runtime overlay-test` pass with `test_dep`; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
