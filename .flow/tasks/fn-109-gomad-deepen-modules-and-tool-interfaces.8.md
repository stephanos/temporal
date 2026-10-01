---
satisfies: [R4]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.8 Move analysis and compatibility review onto the preparation owner's inspection operation

## Description
Stage 3, second half of R4. Analysis and compatibility review each create a private temporary root, prepare adapters and hand a cleanup obligation to their caller. Add the capability-inspection operation to the preparation owner from task 7 and migrate both, so no ordinary caller selects adapters or manages a preparation workspace.

**Size:** M
**Files:** `tools/gomad3/internal/preparation/`, `tools/gomad3/cmd/gomad/internal/cli/analyze.go`, `tools/gomad3/qualification/analysis/{prepared_review.go,analysis.go}`, `tools/gomad3/cmd/gomadtool/compatibility_pack.go`, tests.
**Touches:** [tools/gomad3/internal/preparation/**, tools/gomad3/cmd/gomad/internal/cli/analyze.go, tools/gomad3/qualification/analysis/**, tools/gomad3/cmd/gomadtool/compatibility_pack.go, tools/gomad3/architecture_test.go]

### Approach
- Callers to migrate: `prepareAnalysisTarget` (`cli/analyze.go:53-67`: `os.MkdirTemp("gomad3-analysis-")`, chmod 0700, `PrepareTargetBuildAdapters`, returns a cleanup func) and `PrepareCapabilityReview` (`qualification/analysis/prepared_review.go:21-50`, `Close` `:53-60`: `gomad3-compatibility-review-` root, rejects a caller-supplied `PreparationRoot`, calls `target.ReviewCapabilities`). `cmd/gomadtool/compatibility_pack.go:69,223` consumes `PreparedCapabilityReview` including `BuildAdapters`.
- Inspection returns complete validated review evidence with adapter identities, and owns its workspace through an explicit close whose error is returned. Decide whether the result is a handle with `Close` or a call that completes within the operation by what `compatibility_pack.go` needs after review (it reads rewritten module/overlay inputs); record the reason.
- Mode guarantees are the contract: closure inspection must not compile or execute; linked inspection compiles without launching the target; guarded execution keeps its policy. Prove the closure case with a test whose Go command fails on `build` and still passes.
- `cmd/gomadtool` is owner `developer`; extend `ownerMayImport` only as far as this migration needs.
- Keep `IsInvalidCapabilityReview` (`target/capability.go:54`) and `UnsupportedCapabilityError` classifications reachable with `errors.As` through the new seam.

### Investigation targets
**Required:**
- `tools/gomad3/cmd/gomad/internal/cli/analyze.go` (232 lines)
- `tools/gomad3/qualification/analysis/prepared_review.go`, `prepared_review_test.go`
- `tools/gomad3/cmd/gomadtool/compatibility_pack.go:55-90,210-240`
- `tools/gomad3/target/capability.go:184-292` (`ReviewCapabilityClosure`, `ReviewCapabilities`)
- the task 7 preparation package

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./internal/preparation/... ./qualification/analysis/... ./cmd/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture'
make validate-compatibility
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
- [ ] Analysis and compatibility review use the preparation owner's inspection operation; neither creates a preparation root nor calls `PrepareTargetBuildAdapters` directly.
- [ ] Workspace cleanup is explicit and its failure is surfaced; a caller-supplied preparation root for review is still rejected.
- [ ] Closure inspection is shown not to compile or execute; linked inspection compiles without launching the target.
- [ ] Unsupported closure, malformed linked records and invalid sums keep their classifications (`UnsupportedCapabilityError`, invalid capability review) and CLI exit statuses.
- [ ] `gomad analyze` text/JSON output and `gomadtool compatibility-pack` behaviour are byte-identical for fixed fixtures; `make validate-compatibility` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
