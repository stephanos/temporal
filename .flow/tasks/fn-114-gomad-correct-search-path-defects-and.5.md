---
satisfies: [R3]
---
# fn-114-gomad-correct-search-path-defects-and.5 Derive timer-callback goroutine identity from the timer's creator and inventory parentless creations

## Description
C2 (R3): a goroutine started for a timer callback takes a schedule-independent identity, and every remaining parentless creation site is inventoried. First of the three runtime edits (5, 11, 13) that task 14 qualifies as one toolchain identity.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, `tools/gomad3/toolchain/runtime/go1.27.1.patch`, a new inventory file and test beside `tools/gomad3/toolchain/clock_inventory_test.go`, the C2 fixture from task 2 and `runtime_scheduling.go`, `tools/gomad3/toolchain/version/` only if the source set changes
**Touches:** [tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/toolchain/**, tools/gomad3/internal/gomadtool/conformance/**, tools/gomad3/choice/**, tools/gomad3/expected-intercepts-go1.27.1.txt]

### Approach
- At timer creation, capture what the callback goroutine's identity needs: the creating goroutine's identity, an ordinal taken from that goroutine at creation, and the creation site. Carry it on the timer and use it when the callback goroutine starts.
- The ordinal is consumed at creation whether or not the timer ever fires. Decide whether it shares the creator's child ordinal or is a separate per-goroutine timer ordinal, and state the effect on existing goroutine identities.
- A timer whose callback can start more than once (`Reset` after firing) hashes a per-timer firing ordinal, so two live callback goroutines never share an identity.
- A timer created on one goroutine and reset on another keeps its creator's identity. A timer created with no identified goroutine stays on the parentless path and is an inventory entry.
- Use new label strings for the changed derivation; the existing `/v1` labels are the scheme version.
- Inventory: every path that reaches the parentless branch (runtime-owned workers, collector workers started from whichever goroutine triggers a cycle, finalizer and cleanup goroutines, timers with no creator). Each entry is either given a schedule-independent derivation or listed as a declared exception with its reason.
- Check the inventory in and add a toolchain-tier test that fails when the pinned runtime source contains a creation site that is in neither list. Follow the clock inventory test.
- Update task 2's characterization to require schedule-independent callback identities under the opposite firing schedules, including the valid same-seed prefixes that currently swap identities. Preserve those prefixes' successful execution; task 2 already established that all 16 alternatives of its seed-6 parent succeed. This preservation check does not fix a reproduced same-seed alternative-set divergence. Cross-seed experiments remain separate evidence.
- Regenerate the patch with the repository's patch tooling. Hunks must stay within the patch policy; a derivation that needs a prohibited collector file becomes a declared exception and goes to the patch-policy owner.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:45`, `:745-776` — counter, root identity, assignment
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:338-342` — `newproc1` call site with the caller goroutine
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:750-836` — existing hunk in the runtime timer file
- `tools/gomad3/toolchain/clock_inventory_test.go` — inventory test pattern
- `tools/gomad3/toolchain/buildkey.go:48-57` — what enters the build key
- `tools/gomad3/toolchain/patch.go:37`, `:106`, `:191` — patch policy and allowlist checks

**Optional** (reference as needed):
- `tools/gomad3/toolchain/version/descriptor.go:33-36`, `:131`, `:158` — source-set allowlists
- `tools/gomad3/toolchain/patch_regenerate.go` — patch regeneration
- `tools/gomad3/choice/trace.go:100` — choice implementation identity derived from the build key

### Key context
- fn-110 tasks 2 to 4 move scheduler code into the overlay and re-emit the patch; fn-112 tasks 3 and 5 and fn-109 task 13 edit the same overlay file. Check their state first and rebase onto whichever landed. Never run concurrently with another overlay or patch task.
- Any runtime edit changes the toolchain build key and the choice implementation identity, so every retained tape is rejected by identity. That is expected; retained artifacts are not rewritten.
- The callback goroutine is created on the scheduler's stack, which is why its parent has no identity today.
- Task 2 confirmed schedule-dependent callback identities, but its valid same-seed swapped prefixes succeed. `BuildRankPrefix` truncates the suffix after the changed choice. A cross-seed full-prefix failure is not a supported seed-bound replay failure; the stable-ID requirement remains open, and the no-divergence requirement preserves existing behavior. See task 2's retained final fixture evidence and C2 correction.

## Acceptance
- [ ] In the task 2 fixture each callback goroutine has the same identity under both firing orders
- [ ] A forced prefix that swaps the two callbacks replays without an alternative-set divergence
- [ ] A timer that never fires and a stopped timer leave later identities of the creating goroutine equal across schedules
- [ ] Two callback goroutines started by one reset timer have different identities
- [ ] The inventory of remaining parentless creations is checked in with a reason per entry
- [ ] The toolchain tier fails when a creation site outside the inventory is added, shown by a test with a seeded extra site
- [ ] No hunk touches a prohibited collector file; any exception that needs one is listed and referred to the patch-policy owner
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime overlay-test` pass on darwin/arm64; linux status recorded


## Done summary
Timer-callback goroutines now take a schedule-independent identity from the timer's creator. `newTimer` binds each package-time timer to its creating goroutine (creator identity, a per-goroutine timer ordinal, the `time` entry site; label `gomad3-choice-timer/v1`), `unlockAndRun` publishes the timer and its firing ordinal while its function runs, and a goroutine started from the scheduler's stack inside that window hashes `gomad3-choice-goroutine-timer/v1` over the creator digest and firing ordinal. The timer ordinal is separate from the child ordinal, so existing child identities are unchanged; a timer with no identified creator stays on the runtime ordinal path. Every go statement of the pinned runtime and time packages is inventoried in `tools/gomad3/toolchain/goroutine_inventory_test.go` (one derived site, nine declared exceptions with reasons; the collector, finalizer, cleanup, and signal sites need prohibited files and are referred to the patch-policy owner), and a seeded extra site fails the toolchain tier. The task-2 fixture now requires one identity per callback under both firing orders and across all 16 same-seed one-decision prefixes, including the swapping ones (red on the unmodified toolchain, green on the rebuilt one); the cross-seed experiment completes with the parent's transcript and is retained as evidence only. New `timer_creator_identity` (none/stopped/pending) and `timer_reset_identity` fixtures cover a stopped or never-firing timer and the two callbacks of one reset timer.

Toolchain build key 56e4a2f0 -> 245141dc. `protocol-generate` refreshed the identity mirrors (`choice/internal/wire`, overlay `gomadchoicewire`, and the live-capability producer/guard identities in `target/internal/livecap/protocol_generated.go` and overlay `gomadcap/protocol_generated.go`), and the runner diagnostics-off identity golden (`runner/testdata/diagnostic-identity-choices.json`) was refreshed for the new implementation identity: only `implementation_sha256` and the digests derived from it changed. Those two files sit outside the declared Touches; they are generated or pinned consequences of the identity change, not implementation edits. No hunk touches a prohibited collector file.

Gates on darwin/arm64: `make -C tools/gomad3 validate test-toolchain overlay-test` exit 0 and `make -C tools/gomad3 test-runtime` exit 0 (7:47), run as two commands to fit the foreground window; focused inventory, search-fixture, choice, and gomadtool tests exit 0. Baseline: the toolchain gate was green (the first `test-runtime` attempt failed on a vanished host go-build cache entry; the rerun passed); the Runner Quick command was red before any edit with 13 helper-target failures in `runner` and `runner/internal/execution` ("choice trace unterminated", "I/O transcript terminal is absent"), identical after the edit and outside this task's files. linux/amd64 was not run. Evidence: `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-5/` (gates.md, red-first.txt, search-reproduction.json, toolchain-build-key.txt).

Follow-ups from the review (P3, not blocking): guard the timer slot save/restore in `gomadChoiceTimerFire`/`gomadChoiceTimerFired` with `gomadChoiceEnabled`; say in the `gomadChoiceTimerCreated` comment that the hashed site is the `time` entry point, not the user call site; let the inventory also scan package-level function literals and direct `newproc` calls; consider requiring both labels to lead in the creator and reset checks.

stage: impl-review - ran (backend claude, model claude-fable-5-1 at high, same family as the writer; SHIP on the first round with four P3 findings recorded above)
## Evidence
- Commits: b98c501853609d4569f07461366080b6151dee04
- Tests: baseline: green (make -C tools/gomad3 validate test-toolchain overlay-test; make -C tools/gomad3 test-runtime green on rerun after a host go-build cache miss), baseline: red (go -C tools/gomad3 test -tags test_dep ./runner/... ./artifact/... ./target/... ./cmd/gomad/... - 13 pre-existing helper-target failures in runner and runner/internal/execution, identical after the edit), make -C tools/gomad3 toolchain, make -C tools/gomad3 validate test-toolchain overlay-test, make -C tools/gomad3 test-runtime, tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep -count=1 -run TestRuntimeSearchFixtures ./internal/gomadtool/conformance, .toolchain/bin/go test -tags test_dep -count=1 -run 'TestPatchedRuntimeGoroutineCreationsAreReviewed|TestGoroutineCreationInventoryRejectsSeededSite|TestPatchedRuntimeHostClockReferencesAreReviewed' ./toolchain, go -C tools/gomad3 test -tags test_dep -count=1 ./choice/... ./internal/gomadtool/... ./cmd/gomadtool, go -C tools/gomad3 test -tags test_dep -count=1 -run TestDiagnosticsOffPreservesExistingCanonicalIdentities ./runner, go -C tools/gomad3 test -tags test_dep ./runner/... ./artifact/... ./target/... ./cmd/gomad/... (exit 1: the same 13 pre-existing failures as baseline), go vet -tags test_dep ./toolchain ./toolchain/version ./internal/gomadtool/... ./choice/... ./target/internal/livecap/..., linux/amd64: not run (no native host)
- PRs: