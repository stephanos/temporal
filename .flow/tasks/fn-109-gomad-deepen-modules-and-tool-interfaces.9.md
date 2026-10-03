---
satisfies: [R10]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.9 Give target Go commands one private host-command seam with bounded output

## Description
Stage 3, R10 (F9). Target compilation, Go identity queries and package listing each start processes their own way; the build captures unbounded combined output. Route them through one private Go-command adapter over the existing `hostexec` primitives while keeping the two output contracts distinct.

**External ordering:** after `fn-108-gomad-reduce-code-size-without-removing.2` (local cleanup; it removes `validateGoCapabilityClosure`, `target/capability.go:184`). Re-anchor `capability.go` line references after it lands.

**Size:** M
**Files:** `tools/gomad3/target/target.go`, `target/capability.go`, `target/internal/capabilityreview/list.go`, a new private adapter under `target/internal/`, tests.
**Touches:** [tools/gomad3/target/**]

### Approach
- Call sites: `target.go:379` (`go env GOVERSION GOOS GOARCH CGO_ENABLED`, no context), `:405` (`go env GOMODCACHE`), `:442` (`go mod download -json`), `:736-739` (build, `exec.CommandContext` + unbounded `CombinedOutput`, then `cacheUse.Release()` `:740`), `capability.go:881` (`go list std`), `capabilityreview/list.go:104` with its own `runBounded` (`:150`).
- Reuse `hostexec.Run` (`internal/hostexec/command.go:17-58`): it needs a working directory, a positive timeout and an output limit, and gives process-group termination and head/tail capture with full hashes (`output.go`). Owner `target` may already import `hostexec` (`architecture_test.go` `ownerMayImport`). Follow the private `dependencies{run: hostexec.Run}` shape at `toolchain/build.go:63-82`.
- Two contracts, one mechanism: structured output (`go list -json`, `go mod download -json`, `go env`) requires complete bounded data and rejects overflow instead of decoding a truncated prefix; diagnostics (compiler output) may keep bounded head/tail plus hashes. Do not merge them into one "output" type that makes truncated structured data acceptable.
- Timeouts: derive from the caller context deadline; `hostexec` rejects a non-positive timeout, so decide the bound used when the context has no deadline (`ReadToolchainIdentity` currently has no context) and keep cancellation returning `ctx.Err()` as `list.go:108-110` does.
- Error text and wrapping are behaviour: `"prepare %s target: %w: %s"` and `linkedCapabilityBuildError` (`target.go:744-748`), `CommandError{Stderr, InvalidInput}` (`list.go:111-114`), `"release target build cache: %w"` precedence over the build error (`:740-742`), `"query pinned Go command"` (`:383`). Characterize before changing.
- The fake adapter should let fresh/cache preparation tests run without real compilation.

### Investigation targets
**Required:**
- `tools/gomad3/target/target.go:350-460,640-760`
- `tools/gomad3/target/internal/capabilityreview/list.go` and `list_test.go`
- `tools/gomad3/target/capability.go:870-905`
- `tools/gomad3/internal/hostexec/{command.go,command_unix.go,output.go}`
- `tools/gomad3/toolchain/build.go:59-82,200-215`

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./target/... ./internal/hostexec
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./target -run 'Prepare|Cache|Capability'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./qualification/analysis/...
make test-live-capability
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [x] Target compilation, Go identity queries and listing run through one private Go-command adapter built on `hostexec`; `target` no longer calls `exec.Command` directly for them.
- [x] Structured-output overflow is rejected and never decoded; compiler diagnostics are bounded head/tail with full hashes.
- [x] Tests through the adapter cover cancellation and process termination, long diagnostics, structured overflow, malformed listing, build failure and build-cache lock release on every path.
- [x] Timeout, unsupported capability, invalid-input diagnostics and cleanup failure remain distinct errors with their existing wrapping and precedence.
- [x] Fresh and cached preparation produce the same `Prepared` value and provenance as before for a fixed fixture.
## Done summary
Target compilation, Go identity/cache queries, module download and standard/package listing now use one private hostexec adapter. Structured data rejects overflow before parsing; compiler diagnostics retain bounded head/tail and full stream hashes. Caller cancellation, process-group termination, existing public target APIs, error classifications and build-cache release precedence are preserved. Fresh/cache fake preparation and actual original/final fixtures match whole Prepared and provenance values (Path alone normalized).

All five task9 acceptance checks and R10 are met. Review reproduced a relative-toolchain-root exec preparation regression; absolute command resolution fixes it. The public Prepare regression failed before the fix and passes afterward, comparing complete relative/absolute Prepared values. Final affected target tests, architecture, vet, scoped task lint and CLI rebuild/smoke pass. The full Darwin host gate passed actual make exit0, 47 packages, 238.54s on round1; the narrow path fix used affected verification rather than another full gate. Live-capability and all baseline Quick gates passed. Native linux/amd64 and final spec qualification remain open under task21; root/global lint limitations are retained.

Formal same-receipt review reached SHIP with zero introduced findings and R10 met at 2026-10-03T15:21:25.887103Z. The unchanged pre-existing linked-capability identity query without its caller context remains recorded and nonblocking. Eight final source hashes, five originals and original comments, exact patch, logs, fixture bytes and installed CLI hash were independently verified; post-review source mismatch0. Original fixture/lint failure raw logs were overwritten by the writer; retained terminal excerpts disclose that limitation. Missing-symbol adapter red is a compile failure, not a behavioral regression.

MILESTONES.md now excludes completed-work rows, delivered tables and completion narratives, retains open work and blockers, and preserves the overthinking reminder. Local links and source/document diff checks pass. Unified patch context payloads cause whitespace-only artifact findings in unfiltered diff check; source/document checks exclude only patch payloads. User-requested force push with an explicit lease succeeded; later external WIP commit/push already matches the remote. No agent-created commit or staging; final fix, cleanup and lifecycle evidence remain local. No task10 started; milestone execution stops after this task.

stage: impl-review - ran [2026-10-03T15:06:21.462874+00:00..2026-10-03T15:21:25.887103Z] (model: gpt-6-sol at high; same-family fresh context, same receipt fix loop)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive).

Evidence: handover.md/json, source-freeze.json, preimage.json, task-only.patch, parent-source-verification.json, post-review-verification.json, working-tree-review.json, review-round-1/, review-fix-*.json/log, final-full-host.json/log, milestones-cleanup-check.json and force-push.json/remote-verification.json.
## Evidence
- Commits:
- Tests: .toolchain/bin/go test -count=1 -tags test_dep ./target/internal/gocommand => actual exit 1, 0.391s; adapter-red.json adapter-red.log, make test-live-capability => actual exit 0, 14.383s; baseline-live-capability.json baseline-live-capability.log, .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./qualification/analysis/... => actual exit 0, 159.287s; baseline-overlap.json baseline-overlap.log, .toolchain/bin/go test -count=1 -tags test_dep ./target -run Prepare|Cache|Capability => actual exit 0, 14.375s; baseline-preparation.json baseline-preparation.log, .toolchain/bin/go test -count=1 -tags test_dep ./target/... ./internal/hostexec => actual exit 0, 23.554s; baseline-target.json baseline-target.log, make runner => actual exit 0, 5.076s; cli-build.json cli-build.log, .bin/gomad doctor --json --toolchain-root=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.toolchain => actual exit 0, 0.862s; cli-smoke.json cli-smoke.log, .toolchain/bin/go test -count=1 -tags test_dep . -run TestPackageArchitecture => actual exit 0, 5.156s; final-architecture.json final-architecture.log, make -C tools/gomad3 test-host => actual exit 0, 238.543s; final-full-host.json final-full-host.log, ../../.bin/golangci-lint-v2.13.0 run --config ../../.github/.golangci.yml --build-tags test_dep --timeout 10m --new-from-patch=../../.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/task-only-module.patch . ./target/... ./internal/hostexec/... => actual exit 0, 1.875s; final-lint.json final-lint.log, make test-live-capability => actual exit 0, 20.173s; final-live-capability.json final-live-capability.log, .toolchain/bin/go test -count=1 -tags test_dep ./target -run Prepare|Cache|Capability => actual exit 0, 15.101s; final-preparation.json final-preparation.log, .toolchain/bin/go test -count=1 -tags test_dep ./target/... ./internal/hostexec => actual exit 0, 38.259s; final-target.json final-target.log, .toolchain/bin/go vet -tags test_dep ./target/... ./internal/hostexec/... => actual exit 0, 0.191s; final-vet.json final-vet.log, .toolchain/bin/go run /tmp/gomad-task9-old-fixture.go => actual exit 0, 0.677s; new-prepared-receipt.json , .toolchain/bin/go run /tmp/gomad-task9-old-fixture.go => actual exit 0, 1.636s; old-prepared-receipt.json , .toolchain/bin/go test -count=1 -tags test_dep . -run TestPackageArchitecture => actual exit 0, 1.494s; review-fix-architecture.json review-fix-architecture.log, make runner => actual exit 0, 7.853s; review-fix-cli-build.json review-fix-cli-build.log, .bin/gomad doctor --json --toolchain-root=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.toolchain => actual exit 0, 0.374s; review-fix-cli-smoke.json review-fix-cli-smoke.log, .toolchain/bin/go test -count=1 -tags test_dep ./target -run TestPrepareExecRequiresMatchingProvenance/relative_toolchain_root -v => actual exit 0, 3.49s; review-fix-green.json review-fix-green.log, ../../.bin/golangci-lint-v2.13.0 run --config ../../.github/.golangci.yml --build-tags test_dep --timeout 10m --new-from-patch=../../.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/task-only-module.patch . ./target/... ./internal/hostexec/... => actual exit 0, 2.734s; review-fix-lint.json review-fix-lint.log, .toolchain/bin/go run /tmp/gomad-task9-old-fixture.go => actual exit 0, 1.109s; review-fix-prepared-receipt.json , .toolchain/bin/go test -count=1 -tags test_dep ./target -run TestPrepareExecRequiresMatchingProvenance/relative_toolchain_root -v => actual exit 1, 4.548s; review-fix-red.json review-fix-red.log, .toolchain/bin/go test -count=1 -tags test_dep ./target => actual exit 0, 22.748s; review-fix-target.json review-fix-target.log, .toolchain/bin/go vet -tags test_dep ./target/... => actual exit 0, 0.171s; review-fix-vet.json review-fix-vet.log, Full host gate47packages exit0 covers round1; final narrow Abs fix affected checks0 and formal SHIP., Whole original/final Prepared/provenance bytes equal; all eight frozen sources and comments match after review., Native Linux not run; task21 remains open. Scoped lint0 does not imply historical root/global lint clean.
- PRs: