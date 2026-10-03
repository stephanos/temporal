---
satisfies: [R6]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.4 Resolve CLI installation and private child modes through one application construction path

## Description
Stage 3, first half of R6 (F5 CLI construction). Explore, qualify, resume, shard execution, replay, minimize, analyze and doctor each resolve the installation, executable hash and `__supervisor`/`__coordinator` commands themselves. Give that one private owner in the CLI and pin the command grammar before the parsing refactor in the next task.

**External ordering:** start only after the fn-108 R6/R7 tasks are done and verified: `fn-108-gomad-reduce-code-size-without-removing.5` (shared assessment, R6) and `.6` (retention and artifact-input composition, R7) in `tools/gomad3/runner`. Re-anchor the line references below against the post-fn-108 source first. flowctl cannot record a cross-spec task edge, so check `flowctl tasks --spec fn-108-gomad-reduce-code-size-without-removing` before `flowctl start`.

**Size:** M
**Files:** `tools/gomad3/cmd/gomad/internal/cli/{cli.go,qualify.go,resume.go,recover.go,campaign_shards.go,analyze.go,doctor.go}`, a new private construction file, `cli_test.go` plus a new characterization test file.
**Touches:** [tools/gomad3/cmd/gomad/internal/cli/**]

### Approach
- Current duplication: `localIdentity` (`cli.go:1041-1063`) resolves the installation and hashes the executable; `runDoctor` resolves it again (`cli.go:370`); private-mode argv is built at `cli.go:626`, `qualify.go:143,163`, `resume.go:57` and in the replay/minimize/shard paths. Private modes dispatch at `cli.go:101-102` to `runner.DispatchPrivateMode`.
- One private application value built once per invocation carries the resolved toolchain root, executable, Runner build identity and the child-mode commands. Operations receive it; none calls `os.Executable`, `toolchain.ResolveInstallation` or hashes the binary on their own.
- Keep the existing test seams (`analyzeDependencies` `analyze.go:42-50`, `minimizeDependencies` `cli.go:913`, the qualify `run`/`replay` functions `qualify.go:26-27`): they become fields fed by the application value, not new globals.
- A public tool constructor is out of scope unless the consumer inventory in the executor-injection task establishes a need (spec "API Contracts").
- Characterization first (new test file): documented grammar from `tools/gomad3/CLI.md` for every command, explicit-zero and irrelevant-flag rejection (`resolveExploreStrategy` `cli.go:718-800`), `--env`, `--build-tag`, argv after `--`, text and JSON output, exit statuses 0/1/2/3, and output-writer failure. These tests must pass before and after.

### Investigation targets
**Required:**
- `tools/gomad3/cmd/gomad/internal/cli/cli.go:95-146,347-420,560-692,871-1063`
- `tools/gomad3/cmd/gomad/internal/cli/{qualify.go,resume.go,campaign_shards.go,analyze.go,doctor.go}`
- `tools/gomad3/CLI.md`
- `tools/gomad3/toolchain/installation.go:18-110`
**Optional:**
- `tools/gomad3/cmd/gomad/internal/cli/cli_test.go:180-220,1150-1170`

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./cmd/... ./qualification/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture'
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
- [x] Installation, executable identity, Runner build and private child-mode commands are resolved in exactly one construction path; no other CLI file calls `os.Executable` or `toolchain.ResolveInstallation`.
- [x] Characterization tests cover documented grammar, explicit zero and irrelevant flags, environment/tags/argv, text and JSON output and exit statuses for every command, and pass unchanged before and after.
- [x] Malformed input and stdout/stderr writer failures keep their classification and exit status; no flag default changes.
- [x] No mutable package-global hook is introduced; existing dependency-struct seams remain the test entry points.
## Done summary
CLI invocations now share one lazy private application value for executable discovery, installation/root resolution, Runner identity, private child commands and private-mode dispatch. Operations consume it after input validation. Existing dependency-struct seams remain usable; doctor retains its report and executable validation. Shared plan/explore parsing and semantic normalization remain task 5.

Characterization covers all public commands' missing operands, explicit-zero/irrelevant flags, writer errors, environment/build tags/argv, and text/JSON output, alongside the unchanged command tests. The same characterization passes against captured original CLI bytes using a Go overlay and against final source. Parent independently runs both (0.707s original; 0.422s final including the application cache test) and verifies 21 original snapshots, ten final hashes, existing comments and the task-only patch.

The frozen Darwin full host gate passes (Runner 154.831s; execution 95.231s), including cmd/qualification and root architecture packages. Final affected tests, vet, changed-line root-config lint, formatting/diff checks and CLI rebuild pass after small moved-code error-handling/lint fixes; the final comment-only rationale edit is rebuilt. Filtered lint zero does not mean full lint cleanliness: the unfiltered CLI run retains legacy findings, and root lint's known nested-module loading failure is unchanged. See handover.json for exact commands and logs.

Independent codex:gpt-6-sol:high review returned SHIP at 2026-10-03T11:52:32.369780Z with zero introduced findings. It records one pre-existing P2: doctor's JSON stdout write ignores its error, preserving the old 0/1 status. The construction refactor preserves that behavior; this receipt does not claim a fix. R6 is partial until task 5; full-spec R18/R19 and native linux/amd64 qualification remain open in task 21. Review uses fresh context in the same model family.

Nothing was staged, committed or pushed; the user owns commits.

stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/...; before production edits; exit 0; baseline-evidence.json, .toolchain/bin/go test -count=1 -tags test_dep -overlay <captured-preimages> ./cmd/gomad/internal/cli -run "TestPublicCommand|TestQualifyGrammar" -v; before-refactor bytes with characterization; exit 0; baseline-overlay/characterization.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/...; refactor source before lint-only fixes; exit 0; quick-cmd-gomad.log, GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host; frozen refactor source before lint-only fixes; exit 0; full-host.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/internal/cli -run "TestApplication|TestRunDoctor|TestPublicCommand|TestQualifyGrammar"; final lint-only fixes; exit 0; post-lint-focused-test.log, .toolchain/bin/go vet -tags test_dep ./cmd/gomad/internal/cli; final; exit 0; scoped-vet.log, env -u GOROOT -u GOBIN -u GOMADSEED -u GOMAD3_CHILD_SEED PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOEXPERIMENT=nogreenteagc GOTOOLCHAIN=local GOWORK=off /Users/stephan/Workspace/temporal/gomad/.bin/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --new-from-rev=HEAD ./cmd/gomad/internal/cli (working directory: tools/gomad3); final scoped delta; exit 0; scoped-lint-new.log, make -C tools/gomad3 runner; final; exit 0; runner-build.log, gofmt -l <ten changed Go files>; git diff --check -- tools/gomad3/cmd/gomad/internal/cli; final; exit 0; final-source-manifest.json
- PRs: