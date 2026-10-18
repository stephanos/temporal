---
satisfies: [R42]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.16 Name the wasm ABI explicitly and remove export rewriting

## Description
Rename the application exports to `umpire_buffer_alloc`, `umpire_buffer_release` and `umpire_call`, using direct Rust exports on the stock toolchain. The owner approved these names after the ABI explanation on 2026-10-18. Remove the obsolete linker wrapper, export-table rewrite implementation and rewrite-only tests/configuration. Retain actual compiled-module export/signature validation and its negative controls.

This is a transport naming change. Preserve protobuf messages and method IDs, the packed length/pointer return, live-allocation ownership, size limits, typed refusals, trap invalidation and the same native/Go report and row goldens. Update Go lookups, diagnostics, test controls and ABI/host/workspace/DESIGN documentation. Rebuild the real guest and prove host parity with the direct exports. Remove documentation describing export rewriting as part of the Rust recommendation.

**Touches:** Makefile, model2/.cargo/config.toml, model2/umpire/wasm/**, model2/umpire/proto/README.md, model2/hosts/go/**, model2/scripts/**, model2/README.md, model2/DESIGN.md

**Scheduling:** runs alone after .5 closes and before .15 moves the workspace. Shared Makefile and build configuration make this an always-serial lane. No dependency, schema, generated protobuf or Scala changes. Root owns MILESTONES and Flow closure.

**Quick commands:** `make model2-check`; `make model2-time`; `cd model2 && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked`

**R30:** capture the actual warm timing output. Inner loop at most 3 s and full gate at most 30 s; fix a breach or obtain an owner exception. Do not claim historical module size or host latency measurements describe the new binary; record its new size and identify older measurements honestly until .7 refreshes them.

## Acceptance
- The actual built guest directly exports the three approved names with the existing signatures. Old alloc/free/call and umpire_free application exports are absent.
- No custom linker or post-link export rewriting remains; direct locked Cargo guest build succeeds.
- Go resolves the new names, rejects missing/unexpected/wrong-signature exports, and all allocation/refusal/trap/parity controls pass with unchanged report and row goldens.
- The full gate and warm timing budgets pass; documentation describes the direct ABI and removes the obsolete workaround.


## Done summary
The guest directly exports umpire_buffer_alloc, umpire_buffer_release and umpire_call with the existing byte ABI signatures. The Go host resolves those names, validates the real compiled guest, and rejects missing, unexpected and mis-typed application exports; the obsolete linker wrapper, export normalizer, rewrite-only tests, target linker configuration and rewrite gate step are deleted.

Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)
stage: impl-review - ran (codex:gpt-6.1-sol:high; correctness/contracts/integration all first-pass SHIP; same-family reviewer)

At worker handover, task fn-210-rust-prototype-kettle-on-a-trait-engine.16 remained in_progress for conductor review and closure. Its workspace was .worktrees/fn21016-abi. Source range was ac1c322e714b8f17b0a292e6800612d06f27e2bb..19bf7b6d4a0a21cf88131f6d5c10604f28f9051d. The source commit is 19bf7b6d4a0a21cf88131f6d5c10604f28f9051d.

baseline: green. Before implementation, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model2-check, make model2-time and the task's direct locked Cargo guest build all exited 0. The baseline gate receipt check returned RUN (no honorable receipt). Baseline warm timing was 2.904 s for the gate and 1.279 s for the default checker edit/test loop. Logs: .flow/tmp/fn21016-baseline-check.log, fn21016-baseline-time.log and fn21016-baseline-direct.log in the workspace.

R42 red-to-green proof: TestGuestExports failed against the baseline built guest because all three approved exports were missing and alloc/free/call remained. The host controls using the new names also failed before production changes because the host still required the old exports. Logs: .flow/tmp/fn21016-red-guest.log and fn21016-red-host.log. After the change, the direct stock-linker release build and the complete CGO_ENABLED=0 Go suite exited 0 (.flow/tmp/fn21016-direct.log and fn21016-host.log). TestRejectsApplicationExports covers a missing export and wrong signature for each required function, plus each retired name and an unrelated extra export. TestTrapIdentifiesABIAndLogicalMethod now checks the new ABI names in trap diagnostics. Existing allocation bounds/ownership/double-release, typed refusal, response exhaustion/restoration, concurrent calls and report/ordered-row golden parity tests still pass.

Final CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model2-check exited 0; log .flow/tmp/fn21016-verify-check.log includes nonempty fixture/UI/property/native dispatch suites and Go parity. The gate now passes -tags test_dep to the nested Go module per AGENTS.md. gate classify returned FULL for the executable Makefile diff. No GATE_SKIPPED observations were recorded. CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model2-time then exited 0 on the committed source and ran the full gate again as part of the required actual timing command. No test was rerun merely to discover its exit code. Every command was captured once, and all launched commands exited before return.

R30 final actual warm timing output (.flow/tmp/fn21016-verify-time.log; terminal color escapes omitted):

```text
Time the model2 feedback loop (fn-210 R30)...
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model2-check
make[1]: Entering directory '/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn21016-abi'
Check model2 (fmt, clippy, docs, tests)...
timing-controls                   0.008s
capability-controls               0.137s
protobuf-freshness                0.066s
fmt                               0.088s
clippy                            0.092s
docs                              0.094s
tests                             0.610s
wasm32-wasip1                     0.065s
core-wasm-clippy                  0.074s
wasm-guest                        0.074s
go-host                           1.146s
make[1]: Leaving directory '/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn21016-abi'
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            0.791s
gate total                        2.565s  (budget 30s)
inner loop                        0.791s  (budget 3s)
```

The worker's guest is 3,370,765 bytes including debug information. DESIGN records this measurement and identifies fn-210.9's earlier 2,872,544-byte module and latency/memory measurements as historical, pending fn-210.7's refresh. No protobuf/schema/generated code, dependencies, lockfiles, DSL/checker source, Scala Model, goldens or feature map changed. git diff --check passed and the source worktree was clean. Advisory for .15/.7 is in the shared notes directory at fn21016-abi-seam.md.

The conductor fast-forwarded source and review metadata into umpire. All three first-round draws returned SHIP with zero findings and zero unaddressed requirements against exact source head 19bf7b6d4a0a21cf88131f6d5c10604f28f9051d and base ac1c322e714b8f17b0a292e6800612d06f27e2bb. Metadata commit is d12f1179ed166384bc5d807f645a0f17d7d41902. Canonical receipt is /tmp/impl-review-receipt-96c93c67dffe-fn-210-rust-prototype-kettle-on-a-trait-engine.16.json. Raw sidecars are archived at .flow/review-fanout/aee816f5b50b4b41b8bb2fdc43383c2e/ and worker logs at .flow/tmp/fn21016-archived/.

The reviewers' fresh test attempts were sandbox-blocked and are inconclusive. The conductor ran actual integrated `make model2-check`, `make model2-time`, and the direct locked release Cargo guest build, all with CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target and exit 0. Logs are .flow/tmp/fn210-16-integrated-check.log, fn210-16-integrated-time.log and fn210-16-integrated-direct.log. The integrated timing run measured 3.255 s for the full gate and 1.930 s for the checker edit/test loop, both within R30. No executable source changed after verification.

stage: wave-dispatch - ran (isolated serial ABI lane; shared build configuration)
stage: wave-join - ran (source and review metadata fast-forwarded; integrated full gate and direct build passed)
stage: memory-capture - skipped(policy: first-pass SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 19bf7b6d4a0a21cf88131f6d5c10604f28f9051d, d12f1179ed166384bc5d807f645a0f17d7d41902
- Tests: baseline: green; CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model2-check (exit 0), baseline: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model2-time (exit 0; gate 2.904s, inner 1.279s), baseline: cd model2 && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked (exit 0), red: cd model2/hosts/go && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 -run '^TestGuestExports$' ./... (expected exit 1: missing approved exports, obsolete exports present), red: cd model2/hosts/go && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 -run '^TestRejectsApplicationExports$|^TestBoundaryControls$' ./... (expected exit 1: host still requires old exports), cd model2 && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked (exit 0; stock linker, no post-link rewrite), cd model2/hosts/go && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 ./... (exit 0), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model2-check (exit 0), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model2-time (exit 0; full gate 2.565s <=30s, checker inner 0.791s <=3s), git diff --check (exit 0), Integrated make model2-check at 19bf7b6d4a (exit 0; .flow/tmp/fn210-16-integrated-check.log), Integrated CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model2-time at d12f1179ed (exit 0; full gate 3.255s, checker inner 1.930s; .flow/tmp/fn210-16-integrated-time.log), Integrated cd model2 && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked (exit 0; .flow/tmp/fn210-16-integrated-direct.log), First-round correctness/contracts/integration codex:gpt-6.1-sol:high all SHIP; zero findings; /tmp/impl-review-receipt-96c93c67dffe-fn-210-rust-prototype-kettle-on-a-trait-engine.16.json
- PRs: