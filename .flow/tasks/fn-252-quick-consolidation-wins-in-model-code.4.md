---
satisfies: [R4, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.4 Native fixtures reuse Tiny; one SpaceBinding impl

## Description
Implement **Native fixtures (A7)** and **SpaceBinding (A10)** from the spec.

## Focused source-lane contract

Owner-authorized early parallel source lane (2026-10-25), ahead of whole fn248 completion. Parent fn252 still depends on whole fn248; this lane neither satisfies nor removes that dependency. Only admitted tasks .3/.4 may run early; other work remains unstarted.

**Touches:** [model/umpire/backends/native/tests/**, model/umpire/author/suite/src/space.rs, model/umpire/author/suite/tests/**]

Preserve distinct fixture transitions/outcomes/facts, action filters, shared starts and vacuity failures. The single binding implementation retains selected Rc allocation identity and registered World construction. Required focused checks are `cd model && mise exec -- cargo test -p umpire-checker -p umpire-suite --locked` and `cd model && mise exec -- cargo clippy -p umpire-checker -p umpire-suite --all-targets --locked -- -D warnings`, plus formatting. Existing independent controls cover ownership, construction and negative fixtures; review any changed snapshots.

The bounded wrap-up explicitly defers the complete Model pipeline, guest publication and whole-spec R9 acceptance. The R9 obligation below remains open at parent integration; focused source closure must not claim it passed. No snapshots changed in this lane.

## Acceptance
- [ ] The four fixtures reuse `tests/support`; one generic `SpaceBinding` impl.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
# fn-252.4 source handover

Source commit: `fe1fe5384df454402cdb55ab7aadfc60083cffaa` on `work/fn2524-fixtures`, based on admitted `4c4e99ca7e89d87f6c64e7b30cdbbb764294d758`.

Four native fixture wrappers reuse tests/support Tiny, Value and Action. Explicit Choices mode retains all three successors (including separate false/true outcomes to the same target), facts and terminal-state action refusal. Vacuity mode retains the failing claim scoped to never-enabled Action::Unused. Existing Edges and SharedStarts modes retain their independent failures and start deduplication.

SpaceBinding now has one implementation over a private minimal WorldSource trait. Unit storage still constructs a registered World on each binding request; selected Rc storage clones the same allocation. Space layout and public query signatures are unchanged. Net reduction: 273 lines across six source files.

Validation: baseline and post-change `cargo test -p umpire-checker -p umpire-suite --locked` passed: 53 tests plus one compile-fail doctest, with one existing ignored example. This includes the native fixture golden runner, independent brute-force oracle, explicit_world, declared_world, retained and world_domains tests. `cargo clippy -p umpire-checker -p umpire-suite --all-targets --locked -- -D warnings`, focused fmt check and git diff --check passed. No assertions or snapshots were changed; reviewed the four independent expected reports and retained native fixture output exactly. Logs: `.flow/tmp/fn2524-tests.log`, `.flow/tmp/fn2524-clippy.log`.

No full make model-check was run in this lane. Joined fn-252 R9 validation remains open for eventual task .8. No review, Flow completion, MILESTONES, remote or history mutation was performed by this worker beyond its isolated source commit. Conductor owns authentic implementation review and task completion after integration.

No live commands remain. Worktree `.worktrees/fn2524-fixtures` is clean and retained pending integration; branch is unintegrated at handover. Private Cargo target: `.flow/tmp/fn2524-target`.

Conductor merged source3447c056d6 from actual premergebase 3b11e230f2475d68cdc544f036e2b58cb0dc6f85; evidence/operational contract commitaca114eab9. All six source blobs match the worker commit. The current joined checker/suite suite passed56 tests and1 compile-fail doctest with1 inherited ignored example; affected all-target Clippy and focused formatting exited0. The source changes retain distinct choices/facts/vacuity semantics and registered/selected World ownership. No snapshots changed.

stage: impl-review - ran (codex:gpt-6.1-sol:high, same family; all three fresh readonly draws SHIP, no findings)

Receipt and raw draws are durable in task4/review. Worker logs/admission/source bindings are durable in task4. One triage probe used an invalid SHA and returned an error; it did not skip or fabricate a review and fell through to the full backend review on the actual premerge base. Whole-spec R9 and the fn252 parent dependency on whole fn248 remain open. No complete Model gate, guest publication, parent closure or new task was performed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fe1fe5384df454402cdb55ab7aadfc60083cffaa, 3447c056d6f271e59bcda5491d65d306382013eb, aca114eab95a27a88073669f419b27f8017d4b28
- Tests: cd /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2524-fixtures/model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2524-target mise exec -- cargo test -p umpire-checker -p umpire-suite --locked, cd /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2524-fixtures/model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2524-target mise exec -- cargo clippy -p umpire-checker -p umpire-suite --all-targets --locked -- -D warnings, cd /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2524-fixtures/model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2524-target mise exec -- cargo fmt -p umpire-checker -p umpire-suite -- --check, cd /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2524-fixtures && git diff --check, cd model && BASH_ENV=/dev/null INSTA_UPDATE=no CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-checker -p umpire-suite --locked, cd model && BASH_ENV=/dev/null INSTA_UPDATE=no CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo clippy -p umpire-checker -p umpire-suite --all-targets --locked -- -D warnings, cd model && BASH_ENV=/dev/null INSTA_UPDATE=no CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2481-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo fmt -p umpire-checker -p umpire-suite --check
- PRs: