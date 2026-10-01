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
- [ ] Target compilation, Go identity queries and listing run through one private Go-command adapter built on `hostexec`; `target` no longer calls `exec.Command` directly for them.
- [ ] Structured-output overflow is rejected and never decoded; compiler diagnostics are bounded head/tail with full hashes.
- [ ] Tests through the adapter cover cancellation and process termination, long diagnostics, structured overflow, malformed listing, build failure and build-cache lock release on every path.
- [ ] Timeout, unsupported capability, invalid-input diagnostics and cleanup failure remain distinct errors with their existing wrapping and precedence.
- [ ] Fresh and cached preparation produce the same `Prepared` value and provenance as before for a fixed fixture.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
