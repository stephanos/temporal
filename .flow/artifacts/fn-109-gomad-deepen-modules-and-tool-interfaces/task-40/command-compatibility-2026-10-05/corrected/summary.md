# Task 40 corrective source progress

Task fn-109-gomad-deepen-modules-and-tool-interfaces.40 remains in_progress. The accepted C1 and C2 findings now have real failing-before and passing-after controls. Root retains review, commits, evidence archiving and lifecycle ownership. This lane created no commits and issued no review verdict.

Tier: session (jev-unavailable(no_key)); explicit AGENTS implementer gpt-6.1-sol/high retained.
stage: impl-review - skipped(policy: host-deferred; root owns review and original owned gates remain red or unproved)

Base and HEAD remain 29e12b5e473de0aa4e7b3df4db8626d421ed8891. The original handover at /tmp/fn109-task40-handover.n9ferk remains unchanged. Root archived its evidence under .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-40/command-compatibility-2026-10-05/initial. The corrective changes affect only command_unix.go and additional command_test.go test code within the existing five-file task surface.

## Corrections

C1 now returns actual non-exit Wait copying failures through Run's infrastructure-error return. The original cause object remains the returned error; complete captured outputs remain in Result so Compatibility retains bounded stderr while rejecting stdout. Infrastructure therefore wins over physical output overflow. The failing-reader test makes a real shell read EOF, emit over-capacity JSON plus a fitting diagnostic, and exit 0. Both an independent read-error sentinel and a reader returning context.Canceled while the caller context stays live exercise this contract. Before the fix, Run returned nil infrastructure error and Compatibility projected stdout overflow. The same cases now return the exact reader error with no decodable stdout.

Go CommandContext has a separate non-exit Wait case. A successful Cancel kill can race with exit 0, causing watchCtx to return the actual caller context sentinel. Blanket classification by error type would mistake that valid legacy command outcome for copying infrastructure. The opted-in Cmd.Cancel now wraps its original callback without changing the call or returned error. A local boolean records whether the original cancellation actually returned nil. After command.Wait returns through waits, classification retains the exact Wait error as command data only when that recorded cancellation succeeded and waitErr equals ctx.Err. All other non-exit Wait errors are infrastructure. The boolean is per command, with no global hook or added process framework. Wait receives watchCtx's result before sending waits, establishing the happens-before relation for the local read. The focused race run passed. Existing pre-start lookup/context precedence and acknowledged after-start SIGKILL/deadline controls also pass.

This provenance branch preserves the pinned watchCtx rule at Go1.27.1 src/os/exec/exec.go lines 816-836. The rare successful-Kill plus successful-exit interleaving itself was assessed from that source; no deterministic real reproduction of that race is claimed. The live-context stdin sentinel supplies executed evidence that a context-shaped copying error is not classified by a context-state heuristic.

C2 now clears watchdog/cancellation attribution when Process.Kill reports os.ErrProcessDone, even if the actual Wait outcome is nonzero. The real control blocks the stdin copier, waits for a PID-marked child to exit 7 and be reaped, holds copying beyond its 100 ms watchdog using a bounded timer, then releases EOF. Before the fix, the actual exec.ExitError for exit 7 was wrapped in WatchdogError. After the fix, Compatibility returns the same actual Wait error object, with its real ProcessState/PID and exit 7. Existing genuine watchdog SIGKILL and TERM-ignoring descendant tests remain passing. The reproduction uses no sleep, fabricated termination result, global production hook, or replacement subprocess owner.

## Commands and results

All commands ran from tools/gomad3 with /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin first on PATH. GOENV=off, GOWORK=off, GOTOOLCHAIN=local, GOPROXY=off, GOSUMDB=off and empty GOFLAGS were exported. GOMADSEED, GOMAD3_CHILD_SEED and GOMAD3_SEED were unset. Each gate ran once per observation in a foreground timeout 600 call with captured output. Every command handle is terminal.

| Command | Exit and seconds | Log |
| --- | --- | --- |
| go test -count=1 -tags test_dep ./internal/hostexec ./target/internal/gocommand ./target/internal/capabilityreview | baseline 0, 4; final 0, 4 | baseline-focused.log; final-focused.log |
| go test -count=1 -tags test_dep . -run '^TestPackageArchitecture$' | baseline 0, 2; final 0, 1 | baseline-architecture.log; final-architecture.log |
| /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./internal/hostexec ./target/internal/gocommand ./target/internal/capabilityreview | baseline 1, 1; final 1, 1 | baseline-lint.log; final-lint.log |
| go test -count=1 -tags test_dep ./target/internal/gocommand -run '^TestCompatibility(StdinFailure\|CompletedExit)' | RED 1, 0; GREEN 0, 0 | red.log; c1-c2-green.log |
| go test -count=1 -tags test_dep ./target/internal/gocommand -run '^TestCompatibilityCompletedExit' | 0, 0 | c2-green.log |
| go vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype ./internal/hostexec ./target/internal/gocommand ./target/internal/capabilityreview | 0, 0 | final-errortype.log |
| go test -race -count=1 -tags test_dep ./target/internal/gocommand -run '^TestCompatibility(StdinFailure\|CompletedExit\|CallerLifetime)' | 0, 5 | final-race.log |
| make validate | 0, 4 | final-validation.log |
| go test -count=1 -tags test_dep ./toolchain -run '^(TestBuild\|TestValidate\|TestMaterialize)' | 0, 1 | consumer-toolchain.log |
| go test -count=1 -tags test_dep ./internal/gomadtool/conformance -run '^(TestRun(Upstream\|Builder\|LiveCapability\|Accepts\|Rejects\|Reports\|Interception)\|TestResolve\|TestRequireStockCompatibilitySelectsPinnedToolchain$)' | 0, 0 | consumer-conformance.log |
| go test -count=1 -tags test_dep ./qualification/set ./qualification/soak | 0, 1 | consumer-set-soak.log |
| go test -count=1 -tags test_dep ./upgrade -run '^TestRun(Publishes\|KeepsPriorDossier\|ReplacesExistingDossier)' | 0, 1 | consumer-upgrade.log |
| go test -count=1 -tags test_dep ./upgrade/pinimpact -run '^TestGoResolver' | 0, 0 | consumer-pinimpact.log |
| go test -count=1 -tags test_dep ./upgrade/adapterregen -run '^(TestDefaultGeneratorsAreTheModuleLocalMakeGenerateSteps\|TestUnifiedDiff\|TestStalePacksNameBindingsOfThePreviousIdentity)$' | 0, 0 | consumer-adapterregen.log |

The Markdown table escapes pipe characters for rendering. evidence.json holds the literal executed commands. Final gofmt -l on the five admitted files exited 0 with empty output. git diff --check exited 0. No new lint findings remain. Actual unfiltered lint retains exactly four inherited unchecked pipe Close defers and the historical test-helper sleep. Its red baseline and final result are preserved without suppression or filtered lint.

Both new regressions failed behaviorally before their fixes. red.log retains the original cause/output-precedence error and the real completed exit 7 with WatchdogTimeout true. c2-green.log isolates the attribution fix before the C1 correction. c1-c2-green.log verifies both fixes together. Final focused and race logs include the existing cancellation/deadline/watchdog protections and no empty test selection.

## Frozen inputs and preservation

source-freeze.sha256 binds all five final admitted files. tool-config.sha256 binds pinned go/gofmt, lint and errortype binaries, .github/.golangci.yml and nested go.mod/go.sum. Both checksum files were rechecked with exit 0 after all commands. full-diff.patch retains the complete current two-file diff relative to committed BASE, including the original task source progress. Tool versions remain pinned as recorded in the immutable initial handover.

protected-hostexec-tests.diff and protected-gocommand-tests.diff are empty. The entire original test-function suffixes, including their helper bodies and comments, are byte-identical to BASE. protected-source.diff is empty for .github, nested module/sums, pinned descriptors and the public AdapterPreparedSourceSetSHA256 helper. No public target API, pin, generator output, grant, module or publication policy changed. Root's MILESTONES.md and artifact writes and unrelated .turbo notes remain intact.

The default execution branch, Structured and Diagnostic retain their previous behavior. The additional Cancel wrapper and Wait classification run only under PreserveCommandError. The six applicable consumer controls passed on the corrected shared source. No native or patched-toolchain gate was substituted with those controls.

## Acceptance still open

Task40 remains source progress. Green original unfiltered lint, predecessor/matched full baseline, complete original patched-toolchain/full/default/functional/smoke/affected-native gates, native Darwin qualification, exact-input full adapterregen controls, independent source review and formal root implementation review remain unproved. The host is developmental linux/arm64 stock Go and has no patched toolchain. Unchanged missing-toolchain/native failures were not rerun. The initial 29-case unchanged public helper proof remains retained with its empty-source limitation; no task9 integration, source-set/pin qualification or aggregate R10 success is claimed. Task9/fn113/task21 retain their original requirements; fn128 retains transferred Linux qualification.

Defect route:
- prior fixes: root accepted the frozen review's C1/C2 findings and assigned this existing source lane; no competing writer or lifecycle mutation ran.
- diagnosis: real RED confirmed both stdin infrastructure demotion and completed exit 7 watchdog misattribution. Pinned watchCtx explains why proven caller context errors need separate preservation.
- introduced by: the initial uncommitted task40 source candidate; no history change or worktree was required or authorized.
- base: both regressions failed on that candidate; head: both pass on frozen corrected source, with existing lifetime controls and race checks passing.
- live: no live app surface; real stdin and child process controls exercised the shared executor.
