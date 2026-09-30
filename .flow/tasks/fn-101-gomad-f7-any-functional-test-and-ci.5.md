---
satisfies: [R3, R4]
---
# fn-101-gomad-f7-any-functional-test-and-ci.5 Make the gomad3 workflow pass on a clean checkout (linux and macOS runners)

## Description
Fork CI run 36453715669 (workflow_dispatch on stephanos/temporal gomad at d6a6d5d-ish) failed three jobs on clean runners: (1) host-tools-linux tests nine tools/gomad3/internal/* packages that no longer exist (moved) — update the step to the current package set; (2) core-linux: TestRunInterceptionExecutesManifestDrivenCompilerCases fails 'create interception test workspace: stat .../tools/gomad3/.toolchain: no such file or directory' when test-harness runs before any toolchain build — the test must create what it needs (or the tier must order correctly) without relying on local state; (3) core (macos-15): compatibility-pack-qualification fails 'resolve pinned go.opentelemetry.io/otel/sdk module: lstat /Users/runner/go/pkg/mod/go.opentelemetry.io: no such file or directory' — adapter/pack resolution must download pinned modules itself (e.g. go mod download of the exact pinned version, checksum-verified) or the Makefile must ensure it, so a clean checkout works. Reproduce each locally as far as possible (fresh GOMODCACHE/ fresh clone in scratchpad for (3); remove .toolchain in a scratch clone for (2)), fix, then dispatch the fork workflow (`gh workflow run gomad3.yml --repo stephanos/temporal --ref gomad` after the conductor pushes) and iterate until the jobs that can pass on hosted runners pass. Record anything that cannot run on hosted runners (e.g. root DTrace).

## Acceptance
- a fork workflow_dispatch run of gomad3.yml on the branch passes host-tools-linux, core-linux, core (macOS), and temporal-integration, or each remaining failure is recorded with cause

## Done summary
The gomad3 workflow now passes on clean hosted runners. Fork run 36485805062 at 5f720a9aa7 passed host-tools-linux, core (macos-15, including the root-DTrace clock audit, which does run on hosted macOS), and temporal-integration (28/28 supported, exact replay). core-linux passed every step up to the Linux Temporal requalification. That step fails on two expectation findings, listed below, which go to the conductor.

Fixed clean-checkout defects:
- host-tools-linux tested nine package paths that moved. It now tests `./internal/hostexec ./internal/hostfs ./toolchain ./internal/gomadtool/... ./cmd/gomadtool`.
- The interception test workspace failed when `.toolchain` did not exist yet. The driver now creates it first.
- Adapter preparation read pinned modules from a module cache that might never have fetched them. `PrepareTargetBuildAdapters` first checks the target's sums, then runs `target.DownloadModule` for each selected adapter, checksum-verified and outside any module, before preparing. All four callers now use it.
- darwin `kill(-pgid)` returns EPERM when a process group holds only zombies. That failed the toolchain build. `SignalGroup` now accepts EPERM, and callers still confirm the group is gone.
- The clock audit never saw its activation marker, because the darwin ASLR re-exec discards probes placed by `dtrace -c`. The probe now stops itself at the top of main, after the re-exec. DTrace attaches with `-p`, reports ready from BEGIN, and the audit then resumes the probe.
- The linux libc pack request was derived, not observed, and carried the wrong libc replacement inventory (325d10... instead of 94c69a...). It was rediscovered on a runner, then reviewed, approved (sha256:38596e0c...), and regenerated.
- A qualification set's analysis was killed by the analyzer's 2-minute guarded default on a cold build cache. It now gets `--timeout` set to the workload's overall timeout, capped at 30m.
- temporal-integration used Go 1.27.0 from the root go.mod and had a 30-minute limit. It now uses tools/gomad3/go.mod and 90 minutes, because the darwin requalification takes about 41 minutes.
- The reflect2 capability test now downloads its module first.

Tests: `TestDownloadModuleFetchesPinnedModuleIntoCleanCache` (pinned, checksum mismatch, missing version), `TestPrepareTargetBuildAdaptersRejectsMissingSumBeforeDownloading`, `TestClassifyGroupSignal`, `TestAnalysisCommandBoundsAnalysisByWorkloadBudget`. Each was confirmed red without its fix.

Findings for the conductor (linux/amd64, expectations not relaxed):
1. All 10 `functional-*` suites fail with `semantic_coverage_failure`: "required semantic probes were not observed: stdlib.os.getwd", on both seeds and in 3 runs. Root cause: `os.Getwd` fires only through darwin's `os/executable_darwin.go` package init (`var initCwd, initCwdErr = Getwd()`). The required_probes list in temporal.json / tests.generator.json was derived on darwin and is platform-specific. The fix belongs to the manifest owner, via per-platform required probes or dropping getwd.
2. `frontend-system-info` expects `unrepeatable`, which rejects qualified seeds. Across runs it measured 11:qualified,17:nondeterministic, then 11:qualified,17:qualified twice. It needs an expectation decision (`intermittent` fits every observation); that is F3's call.
- The non-gating linux host tier still fails darwin-assumption tests: toolchain/version TestGenerateRendersDescriptorConsumers and the upgrade Run* tests ("qualification host linux/amd64 is unsupported"). These are known and documented in the workflow.

baseline: none (spec defines no Quick commands); verified with the focused suites plus fork CI.

stage: impl-review - ran [codex fan-out NEEDS_WORK (download inside target rewrote go.sum, bypassing sum checks) -> fixed -> re-review SHIP]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0234c6db988ba666f17edab54117271f2ccf6aa1, ed2d90231af8a49747515ad2477144fd3ebbf945, c6f8809af067fe34b68fc5cb33c4a11775bfbdaf, d70eed5bd11dadaf4d3497c9c8c490004cd59003, 05df037063a3754ccbf4facce651d9047686cb8b, 9cc8fda6633f3c9ee383089ac6f1c9c92ea086c9, 4473921d0664724bad9e677b24c1490f2162a6b6, 5622db1fec0ad9e7da9f0239976dc298fb0d045e, faac67ba0deca22d6824aab363a5a8f688f992cd, 88a9480cc84beff3dba44399020804a5551ffec9, 5f720a9aa7ef55fee0562d3ab6ccfe6b675d1194
- Tests: baseline: none (spec defines no Quick commands), make -C tools/gomad3 validate, go test -count=1 -tags test_dep ./internal/hostexec ./internal/hostfs ./toolchain ./internal/gomadtool/... ./cmd/gomadtool ./qualification/set ./internal/compatibilitypack/..., .toolchain/bin/go test -count=1 -tags test_dep ./deterministicio ./target ./runner ./runner/internal/execution ./qualification/analysis ./cmd/gomad/..., fork run 36485805062 at 5f720a9aa7: host-tools-linux success, core (macos-15) success incl. clock audit, temporal-integration success (28/28 supported), core-linux failure only at Temporal requalification (findings)
- PRs: