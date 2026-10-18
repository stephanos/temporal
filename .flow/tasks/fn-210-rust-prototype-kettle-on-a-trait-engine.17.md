---
satisfies: [R43]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.17 Move the generic Go wasm host inside model/umpire

## Description
Owner-approved mechanical relocation: the generic Go wasm host belongs beside Umpire's wasm guest and protobuf contract. Move `model/hosts/go/` to `model/umpire/hosts/go/` after .15, before .6/.7, running alone because paths, generated-output destinations and shared build/doc files change.

**Touches:** `model/hosts/**`, `model/umpire/hosts/**`, `model/scripts/proto-go.sh`, `Makefile`, `AGENTS.md`, `model/README.md`, `model/DESIGN.md`, `model/umpire/proto/README.md`, `MIGRATE.md`

Use a history-preserving move, then update Makefile Go-host gate paths, proto-go.sh generation/freshness output paths, Go fixture/artifact relative paths and current docs/agent instructions. Keep UMPIRE_WASM/CARGO_TARGET_DIR overrides and stock direct guest builds working. Preserve Go module identity umpire.local/oracle, schema, generated Go bytes, method IDs and ABI ownership semantics. Run generator freshness; this task changes location, not schema. The public module-path change belongs to the later standalone extraction plan, not here.

Read MIGRATE.md; it is a plan only, not authority to extract, publish or push. Correct its current-path statements if needed while retaining its historical survey as dated source evidence. Do not alter archived Scala, unrelated Go tooling, protocol messages, fixtures or goldens. MILESTONES.md is conductor-owned.

**Quick commands:** `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check`; `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-time`; from the new host module, `UMPIRE_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_wasm.wasm CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 ./...`.

R30: paste actual warm timing output; inner <=3s and full gate <=30s, fixing any breach or obtaining an owner exception. Use task-unique logs and return a clean checkpoint; no reviews, Flow lifecycle writes, pushes or unrelated history edits from the isolated implementation lane.

## Acceptance
- The host and its tracked history reside at model/umpire/hosts/go; the old model/hosts/go is absent.
- Protocol/schema/generated Go bytes, module identity, goldens and guest exports are unchanged.
- Build/generation commands and default/overridden wasm and fixture paths work at the new location; current docs and AGENTS agree.
- The actual full Rust gate, direct host tests and protobuf freshness pass; actual model-time satisfies R30.
- No standalone extraction, remote changes or Scala/cache changes occur.


## Done summary
Moved the six-file Go host module from `model/hosts/go` to `model/umpire/hosts/go` with `git mv`; Make, protobuf generation/freshness, host artifact/fixture paths and current documentation follow the new location. MIGRATE.md remains a draft extraction plan and its pre-move survey is explicitly dated historical evidence.

Task: fn-210-rust-prototype-kettle-on-a-trait-engine.17
Status: in_progress; conductor owns integration, review and completion.
Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn21017-host
Base: 0a169c2779034f3864526d18bacde6bf83983785
Commit: d905b5b906c4df73aa5927a8fbbd47f8f3ce1d9f
Tier: project implementer gpt-6.1-sol medium, judge no_key retainprojecttier.
stage: impl-review - ran (three concurrent correctness/contracts/integration draws; first-pass SHIP, no findings) (model: gpt-6.1-sol high)

### Preservation and scope

Before the move, every source ancestor and destination ancestor was checked against symlinks, and the destination was absent. The complete source inventory was six tracked files and four directories, without symlinks, ignored or untracked files. The immediate post-move inventory retained every inode and size, including root inode 8837814. No source cache was present or removed. Git records all six files as renames; `git log --follow` on the moved host reaches 6264f11760 and 0fc76f7805 before this checkpoint.

Byte comparisons against the base passed for `host.go`, `go.mod`, `go.sum` and `umpire/oracle/v1/oracle.pb.go`. The module remains `umpire.local/oracle`. Schema, Rust guest, fixture/golden groups, Cargo lockfile, Scala archive and unrelated Go tooling have no diff. Stock direct Cargo build, default host artifact lookup, explicit UMPIRE_WASM lookup and explicit CARGO_TARGET_DIR lookup all passed. The new default Cargo cache created by the stock build is ignored and remains available in this worktree.

Similar code search reused the existing `TestKettleParity` and host suite, extended the existing Make recipe and generator script, and found no need for a new helper or abstraction. After the pure move and before path adaptations, `TestKettleParity` failed because `../../umpire/tests/fixtures/kettle_trait/expected/report.txt` no longer existed; the same test passes after the path correction. The destination-directory assertion also failed before the move and passed afterward. No tests, protocol assertions or gates were weakened.

### Baseline and final evidence

All canonical logs below live under `/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/`.

- Baseline green: `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-time` exited 0, running the complete actual model-check gate; log `fn210-17-baseline-time.log`. Warm full gate 2.743s, inner loop 1.517s. Baseline explicit-artifact Go tests in the old module exited 0; log `fn210-17-baseline-go.log`.
- Baseline diagnostics: the initial nested Bash ran in the wrong directory and returned make exit 2 (`No rule to make target 'model-check'`), retained in the worktree's `.flow/tmp/fn210-17-baseline-check.log`; this is inconclusive and never counted green. A corrected explicit-directory standalone model-check ran to completion in canonical `fn210-17-baseline-check.log`, but its session/exit metadata was accidentally omitted from the tool output, so its observation also remains inconclusive. The subsequent captured baseline model-time exit 0 supplies authoritative full-gate baseline evidence; no suite was rerun just to recover the lost exit observation.
- Relocation red: focused parity exited 1 for the expected stale fixture path; log `fn210-17-relocation-red.log`.
- Final full gate: `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check` exited 0; log `fn210-17-final-check.log`. Native tests include nonzero collected suites, fixtures, UI, property and ownership/export controls; Go host passed.
- Final direct host suite, from the new module: `UMPIRE_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_wasm.wasm CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 ./...` exited 0; log `fn210-17-final-go.log`.
- Stock direct guest build from model/: `env -u CARGO_TARGET_DIR mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked` exited 0 in 11.38s; log `fn210-17-stock-default-build.log`.
- Default artifact lookup from the new module, clearing both overrides, passed `TestKettleParity` with exit 0; log `fn210-17-default-path.log`. Clearing UMPIRE_WASM while supplying the shared CARGO_TARGET_DIR also passed with exit 0; log `fn210-17-target-override.log`.
- Both protobuf freshness and normal regeneration exited 0 using the absolute script path, with generated bytes unchanged. An initial relative-script invocation under this host's nested Bash directory behavior could not locate the script; it performed no regeneration and is not counted as evidence. The corrected absolute script calls succeeded.
- Gate classification required full verification (`FULL: unmatched: AGENTS.md`); no gate was skipped. Shared gate receipts and lifecycle writes are deferred to the conductor. `git diff --check` passed; committed workspace status is clean and the task still reports in_progress. Every command session has exited.

Actual final warm timing output (`fn210-17-final-time.log`, exit 0):

```text
timing-controls                   0.009s
capability-controls               0.129s
protobuf-freshness                0.071s
fmt                               0.101s
clippy                            0.098s
docs                              0.100s
tests                             0.666s
wasm32-wasip1                     0.048s
core-wasm-clippy                  0.065s
wasm-guest                        0.067s
go-host                           1.103s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            0.718s
gate total                        2.582s  (budget 30s)
inner loop                        0.718s  (budget 3s)
```

Both R30 budgets pass. No remote write, extraction, Flow lifecycle mutation, review dispatch, MILESTONES update or worktree teardown was performed.

The preceding implementation record describes the worker's actions. The conductor integrated d905b5b906 without conflict. Review metadata 479387fa4918d9b43e70b1b0d1a36619f2e2ed15 was written directly on canonical umpire, whose reviewed source HEAD was exactly d905b5b906; the implementation worktree retained that same source HEAD. RID 7383a9ad378143e2a5af11a5282ff8d0 is already archived under canonical .flow/review-fanout/. The receipt /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.17.json has SHA256 a0ffce2d8e659f887c4596c999c5f14767e48a13c271144b7001783773432927. All three actual draws were gpt-6.1-sol high, same family as the implementer, SHIP with no findings. Their sandbox-blocked runtime attempts remain inconclusive; actual worker/conductor gates supply runtime evidence.

Canonical integrated make model-check exited 0 (.flow/tmp/fn210-17-integrated-check.log). Subsequent actual make model-time exited 0 (.flow/tmp/fn210-17-integrated-time.log); the combined wrapper later failed only because it tried copying review sidecars from the worktree although they already resided in canonical root. That copy error performed no deletion and is not counted as a failed model gate. Final timing output:

```text
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            2.169s
gate total                        2.895s  (budget 30s)
inner loop                        2.169s  (budget 3s)
```

Canonical checks confirm the old host path is absent, new host path exists, module/generated bytes are identical, and schema/guest/fixtures/lockfile/archive have no diff. git log --follow reaches the earlier source. Memory capture is skipped for first-pass SHIP with no fix. Review and full integrated verification are complete; task closure is conductor-owned. No repository extraction, remote write or release occurred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d905b5b906c4df73aa5927a8fbbd47f8f3ce1d9f, 479387fa4918d9b43e70b1b0d1a36619f2e2ed15
- Tests: Canonical integrated make model-check exit0; .flow/tmp/fn210-17-integrated-check.log, Canonical integrated make model-time exit0; full2.895s inner2.169s; .flow/tmp/fn210-17-integrated-time.log; later wrapper cp failed because sidecars already lived in canonical root, not a model gate failure, Canonical oldhost absent/newhost exists; module/generated bytes identical; schema/guest/fixtures/lock/archive no diff; git log --follow reaches pre-move source, Three concurrent actual gpt-6.1-sol high review draws allSHIP/no findings; receiptSHA256 a0ffce2d8e659f887c4596c999c5f14767e48a13c271144b7001783773432927; RID7383a9ad378143e2a5af11a5282ff8d0 canonical sidecars, baseline: green via actual CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-time, exit 0; canonical .flow/tmp/fn210-17-baseline-time.log, baseline: explicit UMPIRE_WASM CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 ./... from old host module, exit 0; canonical .flow/tmp/fn210-17-baseline-go.log, INCONCLUSIVE: initial wrong-directory model-check, make exit 2; worktree .flow/tmp/fn210-17-baseline-check.log, INCONCLUSIVE: corrected standalone baseline model-check completed but tool result omitted session/exit metadata; canonical .flow/tmp/fn210-17-baseline-check.log, RED: moved TestKettleParity failed for expected stale ../../umpire/tests fixture path, exit 1; canonical .flow/tmp/fn210-17-relocation-red.log, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check (exit 0; canonical .flow/tmp/fn210-17-final-check.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-time (exit 0; full gate 2.582s, inner loop 0.718s; canonical .flow/tmp/fn210-17-final-time.log), from model/umpire/hosts/go: UMPIRE_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_wasm.wasm CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 ./... (exit 0; canonical .flow/tmp/fn210-17-final-go.log), from model: env -u CARGO_TARGET_DIR mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked (exit 0; canonical .flow/tmp/fn210-17-stock-default-build.log), from model/umpire/hosts/go: env -u UMPIRE_WASM -u CARGO_TARGET_DIR CGO_ENABLED=0 mise exec -- go test -tags test_dep -run TestKettleParity -count=1 ./... (exit 0; canonical .flow/tmp/fn210-17-default-path.log), from model/umpire/hosts/go: env -u UMPIRE_WASM CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target CGO_ENABLED=0 mise exec -- go test -tags test_dep -run TestKettleParity -count=1 ./... (exit 0; canonical .flow/tmp/fn210-17-target-override.log), /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn21017-host/model/scripts/proto-go.sh --check (exit 0), /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn21017-host/model/scripts/proto-go.sh (exit 0; generated Go bytes identical to base), cmp against base for host.go, go.mod, go.sum, generated oracle.pb.go (exit 0); no diff in schema, wasm source, fixtures, Cargo.lock or archived tree, git diff --check (exit 0); git status --porcelain empty after commit; flowctl show status in_progress
- PRs: