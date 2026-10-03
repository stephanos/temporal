---
satisfies: [R1]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.2 Give local and isolated campaigns one normalized options owner

## Description
Stage 1, second half of R1 (F1 options ownership). After task 1 the transport is correct but still a hand-maintained copy of 36 fields. Give the serializable campaign intent one owner that `runLocal` and the coordinator envelope both consume, so a new option cannot be dropped again.

**External ordering:** start only after the fn-108 R6/R7 tasks are done and verified: `fn-108-gomad-reduce-code-size-without-removing.5` (shared assessment, R6) and `.6` (retention and artifact-input composition, R7) in `tools/gomad3/runner`. Re-anchor the line references below against the post-fn-108 source first. flowctl cannot record a cross-spec task edge, so check `flowctl tasks --spec fn-108-gomad-reduce-code-size-without-removing` before `flowctl start`.

**Size:** M
**Files:** `tools/gomad3/runner/runner.go` (`CampaignSpec` `:126-177`, `validateConfig` `:1194-1404`), `runner/coordinator.go`, `runner/resume.go:91-110`, `runner/campaign_plan.go:40-50`, `runner/campaign_shard_execution.go:80-95`, a new private options file and its test.
**Touches:** [tools/gomad3/runner/*.go]

### Approach
- One private serializable options value grouped by invariant (target intent, search settings, resource limits, observation, retention). Callbacks (`Progress`), injected dependencies (`Preparer`, `Executor`, `Replayer`), resolved child commands (`SupervisorCommand`, `CoordinatorCommand`), `RunnerBuild` and private resume state stay outside it.
- One normalization function produces it from `CampaignSpec`; `runLocal` and `runIsolated` both use its output, and the coordinator envelope is its serialization. The two conversion functions from task 1 disappear. Strategy defaulting (`runner.go:1211-1214`), shard normalization (`:1215`) and `resumeRequestDefaults` (`resume.go`) are the normalization inputs to unify.
- `record` campaign plans (`campaign_plan.go:47-48`) and resume plans (`resume.go:103-104`) remain separate versioned contracts. Map to and from them; never make the mutable request type the recorded schema.
- Keep the exported `CampaignSpec` field set source-compatible in this task. Regrouping exported fields is a public Go change that needs the consumer inventory owned by the executor-injection task; record any regrouping wish in `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md` instead of doing it here.
- Equivalence harness: before editing, capture for a table of specs (each strategy, guided, sharded, resume, every `validateConfig` rejection) the returned `SeedSelection`, base environment, error text and the coordinator request bytes; the same table must produce identical values afterwards. Request bytes may change only by field grouping, and the test then pins the new bytes with decode equality.

### Investigation targets
**Required:**
- `tools/gomad3/runner/runner.go:126-177,362-400,1194-1450`
- `tools/gomad3/runner/coordinator.go` (post task 1)
- `tools/gomad3/runner/resume.go`, `runner/campaign_plan.go`, `runner/campaign_shard_execution.go:40-95`
- `tools/gomad3/runner/coordinator_transport_test.go` (task 1 tests; they must keep passing unchanged in intent)

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'Coordinator|IsolatedRunner|ValidateConfig|Resume|CampaignPlan|Shard'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./cmd/gomad/... ./qualification/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture|TestExactModuleEdges'
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
- [x] Local and isolated execution consume the same normalized options value; no second hand-written field list exists for coordinator transport.
- [x] Callbacks, injected dependencies, resolved child commands and Runner identity are outside the serialized intent; campaign-plan and resume records keep their own schemas and bytes.
- [x] The characterization table yields identical selections, environments, error messages and precedence before and after; task 1's real isolated-execution tests still pass for all three strategies.
- [x] Unknown fields, trailing data and malformed coordinator requests remain rejected.
- [x] Exported CampaignSpec fields are unchanged.

Verified on darwin/arm64 with final source hashes, full host gate plus the scoped selector-fix rerun, and a fresh read-only SHIP review. Native linux/amd64 remains unverified and is required at fn-109.21 before overall spec completion.
## Done summary
# fn-109.2 implementation handoff

The task implementation and Darwin verification received a SHIP verdict from the configured fresh gpt-6-sol high reviewer on 2026-10-03. This is a same-family review in a fresh read-only context. No files were staged or committed.

The Runner now maps the unchanged 48-field public `CampaignSpec` once into private grouped `campaignOptions` plus separate runtime dependencies. Local execution, portable planning, shard execution, and coordinator execution consume this request. Strategy and shard defaults are cached in the options value after public ingress and coordinator decode; raw values stay in the transport for compatibility. Resume preflight defaults enter at the same point before the local/isolated choice. The coordinator serializes the options value directly, retains only its required supervisor command and Runner build identity in the envelope, and no longer has either full-field conversion function. Existing campaign and resume record types remain distinct.

The pre-edit characterization captured 75 executed rows in `campaign-options-before.json` (7 valid, 68 rejection/precedence cases), including selection, base environment, exact errors, and old request bytes. `campaign-options-before-complete.json` adds the missing `CoverageChoice` plus required-probes rejection as a historical row reconstructed from the captured old wire ordering and unchanged validation branch; the original 75-row artifact remains immutable. The 76-row checked-in fixture now passes against the new implementation. `campaign-options-request-after.json` pins exact new request bytes; tests also compare every old/new top-level request field and strictly decode every new request. Existing coordinator malformed, unknown-field, trailing-data, and actual isolated seed/choice/simulation tests pass.

The frozen Darwin host gate passed with `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` (`test-host.log`). The subsequent source change was exactly the two selector simplifications in `post-host-selector-fix.patch`, whose reconstructed preimages match the frozen gate hashes. On the final source, `go test -count=1 -tags test_dep ./runner` passed (`runner-post-lint-fix.log`, 121.504s), architecture tests passed, and `go vet -tags test_dep ./runner/...` passed. `make -C tools/gomad3 runner` and the `gomadtool` build passed; binary hashes are in `cli-binaries.sha256`. `git diff --check` and `gofmt -l` are clean.

Scoped `golangci-lint` finished exit 1 with 430 findings in the already lint-heavy Runner subtree (`runner-lint-post-fix.log`); neither new options file has a reported finding after the selector fix. The repository root `make lint-code-fast` has an existing exit-2 nested-module loading limitation recorded at `.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/root-lint-retained.json`; this unchanged failure was not rerun. Native `linux/amd64` verification remains unavailable on this Darwin host.

`own-work.patch` is the exact task-only patch, including four new files and the parent-verified three hunks in the two previously dirty tests; `git apply --reverse --check` passes. `source-pre-complete.json` and `source-post-complete.json` bind all 20 changed/new paths. No collector source was changed. Commits: `[]`.

Parent completion evidence: `working-tree-review.json` reports zero introduced findings and R1 met. `parent-verification.json` binds the final twenty paths, task-only patch, binaries, and gate logs; `parent-public-contract.json` verifies the complete public CampaignSpec declaration is unchanged byte for byte. The parent independent characterization run passed, and the final parent scoped lint records 427 findings with only the two redundant selectors removed and no added diagnostic locations. Lint is not clean. Native Linux verification and full spec completion remain pending. Tracker sync: n/a (bridge inactive).
## Evidence
- Commits:
- Tests: env -u GOROOT -u GOBIN -u GOMADSEED -u GOMAD3_CHILD_SEED PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOEXPERIMENT=nogreenteagc GOWORK=off GOTOOLCHAIN=local .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'CampaignOptionsCharacterization|Coordinator|IsolatedRunner|ValidateConfig|Resume|CampaignPlan|Shard' [cwd tools/gomad3; exit 0; focused-final.log], env -u GOROOT -u GOBIN -u GOMADSEED -u GOMAD3_CHILD_SEED PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOEXPERIMENT=nogreenteagc GOWORK=off GOTOOLCHAIN=local .toolchain/bin/go test -count=1 -tags test_dep ./runner -run '^TestIsolatedRunnerExecutesSeed|^TestIsolatedRunnerExecutesChoice|^TestIsolatedRunnerCompletesSimulation' [cwd tools/gomad3; exit 0; isolated-three-strategies.log], env -u GOROOT -u GOBIN -u GOMADSEED -u GOMAD3_CHILD_SEED PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOEXPERIMENT=nogreenteagc GOWORK=off GOTOOLCHAIN=local GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host [cwd repo root; exit 0; test-host.log], env -u GOROOT -u GOBIN -u GOMADSEED -u GOMAD3_CHILD_SEED PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOEXPERIMENT=nogreenteagc GOWORK=off GOTOOLCHAIN=local .toolchain/bin/go test -count=1 -tags test_dep ./runner [cwd tools/gomad3; exit 0 on final source; runner-post-lint-fix.log], env -u GOROOT -u GOBIN -u GOMADSEED -u GOMAD3_CHILD_SEED PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOEXPERIMENT=nogreenteagc GOWORK=off GOTOOLCHAIN=local .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture|TestExactModuleEdges' [cwd tools/gomad3; exit 0 on final source; architecture-post-lint-fix.log], env -u GOROOT -u GOBIN -u GOMADSEED -u GOMAD3_CHILD_SEED PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOEXPERIMENT=nogreenteagc GOWORK=off GOTOOLCHAIN=local .toolchain/bin/go vet -tags test_dep ./runner/... [cwd tools/gomad3; exit 0 on final source; runner-vet-post-lint-fix.log], env -u GOROOT -u GOBIN -u GOMADSEED -u GOMAD3_CHILD_SEED PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOEXPERIMENT=nogreenteagc GOWORK=off GOTOOLCHAIN=local make -C tools/gomad3 runner [cwd repo root; exit 0 on final source; runner-build-final.log], env -u GOROOT -u GOBIN -u GOMADSEED -u GOMAD3_CHILD_SEED PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOEXPERIMENT=nogreenteagc GOWORK=off GOTOOLCHAIN=local .toolchain/bin/go build -trimpath -o .bin/gomadtool ./cmd/gomadtool [cwd tools/gomad3; exit 0 on final source; gomadtool-build-final.log], git diff --check -- tools/gomad3/runner; gofmt -l <changed Go files> [cwd repo root; exit 0 on final source], Parent independent .toolchain/bin/go test -count=1 -tags test_dep ./runner -run '^TestCampaignOptionsCharacterization$' with documented stock/classic-GC environment; exit 0, parent-characterization.json, git apply --reverse --check task-2/own-work.patch; final twenty source hashes, binary hashes, complete CampaignSpec declaration, and frozen host-gate reconstruction verified; parent-verification.json
- PRs: