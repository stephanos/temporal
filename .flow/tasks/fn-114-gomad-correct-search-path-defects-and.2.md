---
satisfies: [R1]
---
# fn-114-gomad-correct-search-path-defects-and.2 Reproduce C2 and E3 with runtime fixtures on the unmodified toolchain

## Description
Second half of R1: fixture programs that reproduce C2 and E3 against the toolchain as it is before any fn-114 runtime edit, with their output retained as files. Tasks 5, 11, and 13 change identities, so this evidence cannot be regenerated later.

**Size:** M
**Files:** new fixture directories under `tools/gomad3/internal/gomadtool/conformance/testdata/` (one for timer-callback identity, one for select readiness), `tools/gomad3/internal/gomadtool/conformance/runtime_scheduling.go`, `registry.go`, `runtime_test.go`, `testdata/README.md`, retained output under `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/runtime-reproduction/`
**Touches:** [tools/gomad3/internal/gomadtool/conformance/**, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/runtime-reproduction/**]

### Approach
- Record the toolchain build key the fixtures ran on.
- C2 fixture: two goroutines each arm a context deadline (or `time.AfterFunc`) due at the same virtual instant, so the firing order follows the schedule. Record the Choice Trace under two schedules that fire them in opposite orders. Show that the callback goroutine identities differ between the two, and that a forced prefix that swaps the order ends in an alternative-set divergence. State in the fixture how each order is produced (seed choice or forced prefix).
- E3 fixture: a finite program covering these select shapes: blocking with zero ready cases, blocking with one ready, blocking with two or more ready, non-blocking with a default, a timer-channel case, a closed-channel case, and a nil-channel case. Record how many select-poll decisions each shape emits and how many of them had fewer than two ready cases.
- E3 baseline for task 12: run an unreduced choice exploration of the fixture to exhaustion and retain its set of outcomes and deadlocks.
- Register the fixtures the way the existing scheduling fixtures are registered. Assertions describe today's behavior; tasks 5 and 12 change them.
- Each E3 shape gets a stable name. Tasks 11 and 12 use the same names for the shape evidence and the eligibility rule.
- If a fixture cannot reproduce its finding, record the finding as refuted or changed per the task 1 closure rule.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/internal/gomadtool/conformance/runtime_scheduling.go:12` — scheduling behavior checks to extend
- `tools/gomad3/internal/gomadtool/conformance/testdata/select/main.go` — existing select fixture
- `tools/gomad3/internal/gomadtool/conformance/testdata/choice_replay/main.go` — forced-prefix fixture pattern
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:363-371` — timer tie-break draw that decides same-instant order
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:766-770` — the counter the C2 fixture exposes

**Optional** (reference as needed):
- `tools/gomad3/internal/gomadtool/conformance/testdata/choice_exploration/main.go` — exhaustive exploration fixture
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:689-721` — select-poll decision and select-result observation
- `tools/gomad3/internal/gomadtool/conformance/registry.go` — fixture registration

### Key context
- fn-112 tasks 2 and 6 add fixtures in the same conformance package. Check their state and rebase onto whichever landed.
- Must finish before task 5 edits the runtime, or the unmodified-toolchain evidence is lost.
## Acceptance
- [ ] The C2 fixture shows different callback goroutine identities under the two firing orders and a divergence on the swapped forced prefix, on the unmodified toolchain
- [ ] The E3 fixture reports select-poll decisions per shape and the count with fewer than two ready cases, for all seven listed shapes
- [ ] The unreduced exploration's outcome and deadlock set for the E3 fixture is retained as a file
- [ ] Fixture output, the toolchain build key, and the commands are retained under `runtime-reproduction/` in the spec's artifacts directory
- [ ] A fixture that fails to reproduce its finding is recorded as refuted or changed and the owning task annotated
- [ ] `make -C tools/gomad3 test-runtime overlay-test` pass on darwin/arm64 with no runtime or overlay file changed; linux status recorded
## Done summary
Registered two runtime conformance fixtures and their scheduling checks on the unchanged patched toolchain. C2 is changed: callback identity instability reproduces under opposite firing schedules, including a valid same-seed prefix, while all 16 one-decision alternatives of its seed-6 parent complete successfully. The original supported swapped-prefix alternative-set-divergence premise did not reproduce. A cross-seed full-prefix experiment ends at a select-site divergence; it is not a supported seed-bound replay failure. Parent applied the narrow C2 spec/task-5 correction and preserved R3 acceptance; stable callback IDs remain task 5's obligation.

E3 is confirmed across seven stable names. Each shape emits one poll decision; six have fewer than two initially ready cases. Unreduced seed-1 exploration expands every recorded decision, including runtime-owned Runnable choices, and exhausts each frontier after 732 total executions with eight outcomes and no deadlocks. Explicit execution/decision bounds fail instead of pretending to exhaust. Task 12 retains suppression soundness ownership.

Native darwin/arm64 evidence is retained in runtime-reproduction/final-approved: source snapshots, target identities, exact child commands/exits, 781 losslessly compressed raw traces, terminal frames, prefixes, callback association table, and baseline configuration/outcome/deadlock sets. Early yielding/child probes are marked exploratory and excluded from the authoritative identity proof. Parent independently verified all 781 trace archives against original SHA and length. The toolchain build key remains 6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee; runtime, overlay, version descriptor, existing comments, and prior task sources are preserved. No toolchain source rebuild, commit, staging, push, worktree, or Flow completion was performed by this worker.

Pre-edit conformance baseline, frozen focused conformance tests, behavior grouping test, focused vet, formatting and whitespace checks pass. Both the initial and final frozen make -C tools/gomad3 test-runtime overlay-test gates pass with exit 0. The frozen test capture has 783 expected-passing command cases, including the intentional cross-seed exit 125. Root make lint-code-fast fails for unavailable main, then nested-module discovery with GOLANGCI_LINT_BASE_REV=HEAD; this is a documented standards limitation, not clean lint. Linux remains unverified.

Independent implementation review returned SHIP with no findings; R1 task-2 scope is met. The reviewer verified the exact patch and evidence bindings, callback associations, all 781 trace archives, and the complete E3 frontier. Receipt: task-2/review-round1.json; session 01a0fbf9-368f-7752-8709-2fad022226bb; task-only patch SHA-256 1052e7e321f4c2872ac1fbc526c80cb69e984e7d2288cdcf15e44dccaff44057. Work remains uncommitted at the user's instruction.

stage: impl-review - ran (model: gpt-6-astra at high)
stage: plan-sync - skipped(config: planSync.enabled=false; evidence-driven C2 owning-task correction applied directly)
stage: wave-dispatch - skipped(policy: shared conformance files and no worktrees; sequential worker)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep ./internal/gomadtool/conformance => exit 0, env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT GOMAD3_RUNTIME_REPRODUCTION_DIR=/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/runtime-reproduction/final-approved tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep ./internal/gomadtool/conformance -count=1 -v => exit 0, env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOFLAGS=-tags=test_dep make -C tools/gomad3 test-runtime overlay-test => exit 0, env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT PATH=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH GOMAD3_STOCK_GO=/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go GOFLAGS=-tags=test_dep make -C tools/gomad3 test-runtime overlay-test => exit 0, env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT tools/gomad3/.toolchain/bin/go -C tools/gomad3 vet -tags test_dep ./internal/gomadtool/conformance => exit 0, env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep . -run TestConformanceRuntimeIsGroupedByBehavior -count=1 => exit 0, /Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/gofmt -l tools/gomad3/internal/gomadtool/conformance/registry.go tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go tools/gomad3/internal/gomadtool/conformance/runtime_choice.go tools/gomad3/internal/gomadtool/conformance/runtime_scheduling.go tools/gomad3/internal/gomadtool/conformance/runtime_test.go tools/gomad3/internal/gomadtool/conformance/testdata/timer_callback_identity/main.go tools/gomad3/internal/gomadtool/conformance/testdata/select_readiness/main.go => exit 0, make lint-code-fast => exit 2, GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast => exit 2, git diff --check -- tools/gomad3/internal/gomadtool/conformance .flow/specs/fn-114-gomad-correct-search-path-defects-and.md .flow/tasks/fn-114-gomad-correct-search-path-defects-and.5.md => exit 0, Independent review: SHIP; task-2/review-round1.json; no introduced or pre-existing findings, Parent post-review verification: 41 bound files unchanged; git apply --reverse --check task-only.patch passed, flowctl validate --spec fn-114-gomad-correct-search-path-defects-and --json => valid true
- PRs: