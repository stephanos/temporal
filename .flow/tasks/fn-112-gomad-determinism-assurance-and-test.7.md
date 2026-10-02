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
Task fn-112.7 implements bounded filesystem and single-connection loopback TCP comparisons through the existing standard-library launcher, and corrects the model defects these comparisons exposed. Sources are frozen; independent review and Flow completion belong to the conductor. No staging, commits, pushes, worktrees, dependency additions or deletion of existing builds/caches occurred.

Each fixture generates all 64 operations from a standard-library seeded RNG before executing a requested prefix. Five fixed seeds (0, 1, 7, 42, 89) run against Gomad and stock Go 1.27.1. Logs compare operation, result and error class, including byte counts, payloads and supported metadata. On failure, fresh executions inspect every ascending shorter prefix and report the seed and shortest reproduced divergence; nonmonotonic mismatch controls verify this search. Representative successful prefixes also prove changing the execution bound preserves generated operations and results. TCP uses one connection, logical deadlines and bounded execution, without sleeps. Address results record endpoint relationships and network names; host-assigned ephemeral port values are omitted from the fixture schema and require no normalization rule.

The sole declared difference is directory allocation size on darwin/arm64 and linux/amd64. Task10 should reference `modelDeclaredDifferences` in tools/gomad3/runner/internal/execution/model_conformance_test.go. Normalization accepts only successful `stat-dir workspace/dir` rows with complete directory metadata, the expected name and nonnegative size; all other fields stay exact. Tests prove it cannot hide file sizes, payloads, errors, other operations/directories, malformed or extra metadata, or unequal operation counts. Native Linux is unverified on this host.

Baseline6b evidence reproduced lost os.ErrClosed at filesystem prefix15, lost net.ErrClosed at TCP prefix60, and self-rename data loss in full sequences. Closed backend handles now preserve EBADF through a distinct private marker; the os adapter translates that marker to os.ErrClosed only after raw transcript recording, preserving wrong-access EBADF. Process handles share the marker. Network closed errors reuse internal/poll.ErrNetClosing. Valid self-renames return without mutations after source, parent, read-only and mount checks; focused tests preserve file/directory metadata, contents and volume durability snapshots. None of these defects became declared differences.

The first corrected build d15 exposed a real libc consumer regression: reading a captured *os.File after descriptor close returned os.ErrClosed, which the errno converter mapped to EIO. Baseline6b returns EBADF. The final narrow libcErrno conversion restores EBADF and retains the captured-descriptor regression in existing anonymous_test.go. First-build failure/proof, source manifest and patch remain separately labeled. The second build d49ef0309636e2301c46cfb36cda8a9b21323d87601c93c429260b3eefd4e37d remains intermediate evidence: its runtime tier passed, but the unseeded overlay gate exposed incorrect closed-directory expectations and a remaining modeled wrapper/sentinel difference. Stock directory reads return PathError wrapping poll.ErrFileClosing (empty operation on Darwin, readdirent on Linux), while Chdir retains os.ErrClosed. A narrow post-record closed-marker branch now matches that directory contract; nonclosed paths and Chdir remain unchanged. Exact wrapper/path/operation/sentinel regressions pass stock, unseeded and seeded scratch checks. The final corrected build is 56e4a2f0c5514d43b9a0682d030964588dd3d5a58978ba0b989459bb556843a3; baseline6b and both intermediate builds remain present.

Exactly two test-only overlay allowlist entries were added: src/os/gomad_model_conformance_test.go and src/internal/gomadfs/export_test.go. Generated consumers were regenerated/validated and remain byte-identical. Runtime patch hunks and capability boundary files remain unchanged. The second rebuild started with 7.6 GiB free and the third with 6,618,044KiB (about 6.31 GiB), under the conductor's documented >=6 GiB measured exception (first build peak additional use about 1.5 GiB); all existing builds and caches were preserved.

Final3 verification passes: generate/validate; toolchain rebuild; three focused model repetitions (30 complete comparisons, 1,920 operations per side), prefix checks and mismatch controls; seeded public os and backend/process/libc regressions; unseeded patched os and pinned stock os regressions; formatting and focused vet for driver and both fixtures. Full test-host passes; complete overlay-test and test-runtime also pass on this final identity (the conductor executed parent-overlay-runtime-final3; exact argv, exit and raw output are retained). All tests use test_dep. Parent independently verifies all 20 frozen sources, all 30 complete comparisons, the 41 protected prior-task sources, all 9 actual built overlay file contents and reconstruction of all 15 changed files from task-only.patch plus actual before copies.

Root make lint-code-fast was run on final sources: default reference main is absent (exit 2); HEAD override reaches golangci-lint but cannot load nested gomad3 packages (including qualification/workload and both new fixtures) from the root module (exit 2, underlying lint Error 7). These are recorded limitations, not a passing lint claim. Focused vet and formatting pass. No native Linux qualification is claimed.

Review inputs are task-only.patch (actual before-copy delta, excluding prior changes), beforecopies/manifest.json, before-task-spec.md, frozen-source-hashes.json, final-source-hashes.json, findings.md, commands.jsonl and bound raw logs. Parent proof artifacts are parent-conformance-validation.json, parent-conformance-final2-validation.json, parent-protected-source-check.json, parent-patch-validation.json, parent-conformance-final3-validation.json, parent-patch-final3-validation.json and parent-built-sources-final3-validation.json. Intermediate d15/d49 manifests and patches remain separately labeled. handover-evidence.json and evidence-bindings.json bind completed final gates to this source/build identity.

Independent implementation review returned SHIP with no blocking findings or unaddressed task R-IDs. Reviewer gpt-6-astra at high, session 01a0fc62-9e53-7a20-b037-30a415fd7ab9, timestamp 2026-10-02T11:35:57.547495Z. R8 is met for this task on Darwin; global two-platform acceptance remains open. The reviewer independently reconstructed the15 changed files, checked157 parent bindings/20 frozen sources/nine built overlay files, and validated30 complete comparison logs. A non-blocking long-method heuristic was not a correctness finding and required no source change.

Review reservation f0bf6706785c4010bb942eea23d0eca0 is finalized with one consumed verdict and zero refunds. Domain artifact SHA256 0bedd7deb617992c1160e286a7d0315a6ad2163e0509e377b8a7c2de7b6f1bcb binds the exact task-only patch. Source and evidence bindings were rechecked unchanged after review. Final completion uses this separate conductor summary; the reviewed handover remains immutable. Commits and PRs are empty because the user owns commits.

stage: impl-review - ran (model: gpt-6-astra at high; verdict: SHIP)
stage: plan-sync - skippedconfig
stage: wave - skippedpolicy (shared workspace; no worktrees; one writer)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: make generate validate (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3), make toolchain (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3), env GOMAD3_CHILD_SEED=7 .toolchain/bin/go test -exec /Users/stephan/Workspace/temporal/gomad/tools/gomad3/internal/gomadtool/conformance/scripts/exec.sh -tags test_dep -count=1 -run '^TestFilesystem(SelfRename|Closed|ReadOnlySelf|VolumeSelf)|^TestCapturedLibcDescriptor' -v internal/gomadfs internal/gomadio (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3), env GOMAD3_CHILD_SEED=7 .toolchain/bin/go test -exec /Users/stephan/Workspace/temporal/gomad/tools/gomad3/internal/gomadtool/conformance/scripts/exec.sh -tags test_dep -count=1 -run '^TestGomad(Closed|Wrong)' -v os (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3), .toolchain/bin/go test -tags test_dep -count=1 -run '^TestGomad(Closed|Wrong)' -v os (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3), env GOMODCACHE=/tmp/gomad-task7-empty-modcache /Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go test -overlay=/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-7/directory-stock-final3-overlay.json -tags test_dep -count=1 -run '^TestGomad(Closed|Wrong)' -v os (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3), .toolchain/bin/go test -tags test_dep -count=3 -run '^TestModel' -v ./runner/internal/execution (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3), make test-host (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3), /Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/gofmt -l tools/gomad3/runner/internal/execution/model_conformance_test.go tools/gomad3/runner/internal/execution/model_conformance_compare_test.go tools/gomad3/internal/gomadtool/conformance/testdata/model_fs/main.go tools/gomad3/internal/gomadtool/conformance/testdata/model_net/main.go tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/fs.go tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/process_volume.go tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/fs_test.go tools/gomad3/toolchain/runtime/overlay/src/os/gomad.go tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go tools/gomad3/toolchain/runtime/overlay/src/os/gomad_model_conformance_test.go tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/export_test.go tools/gomad3/toolchain/version/generated.go tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/libc.go tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/anonymous_test.go (cwd /Users/stephan/Workspace/temporal/gomad), .toolchain/bin/go vet -tags test_dep ./runner/internal/execution (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3), /Users/stephan/Workspace/temporal/gomad/tools/gomad3/.toolchain/bin/go vet -tags test_dep ./model_fs ./model_net (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3/internal/gomadtool/conformance/testdata), make overlay-test test-runtime (cwd /Users/stephan/Workspace/temporal/gomad/tools/gomad3)
- PRs: