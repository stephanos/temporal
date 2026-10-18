---
satisfies: [R5]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.4 umpire-dsl-macros: derive(State) with #[starts], derive(Action) with #[internal]

## Description
**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model2-check` at 30 s or less, warm; paste `make model2-time` output into the done summary; a breach is fixed here or excepted by the owner.

**Shared fixtures 2026-10-18 (R29):** end-to-end fixtures live in `umpire/tests/fixtures/<name>/` after .13; their expected files select the consumers (checker, TLC, Apalache).

**Conventions 2026-10-18 (R25, R26):** `derive(State)` and `derive(Action)` also generate `Spell` in the R25 form, using the author's names (R26); replace the hand-written `Spell` impls in DSL-owned fixtures with the derive and require byte-identical goldens. Compile Kettle-shaped state/action declarations in this lane; canonical shared Kettle migration belongs to .6 so this lane does not modify .3's shared fixture surface.

**Touches:** model2/umpire/dsl/build.rs, model2/umpire/dsl/macros/build.rs, model2/umpire/dsl/src/**, model2/umpire/dsl/tests/**, model2/umpire/dsl/macros/src/**, model2/umpire/dsl/macros/tests/**

**Parallel boundary:** shared Cargo wiring and empty macro crate are prepared by .11; no manifests or lockfiles change in this lane.

**Layout (R22):** expansion cases go in `umpire/dsl/macros/tests/fixtures/<name>/`; refusals in `dsl/macros/tests/ui/`.

**Deps 2026-10-18 (R18):** proc-macro crate on `syn` 3, `quote`, `proc-macro2`, `darling` 0.24 (attribute parsing) and `manyhow` (multiple errors with spans). `derive(Action)` may delegate enumeration to `enum-iterator`'s `Sequence` (syn 2 duplicate accepted). Expansion goldens: insta snapshots of the derive's implementation function, formatted by `prettyplease`, not macrotest.

**Note 2026-10-18 (R17):** derive only what Kettle needs (unit variants, `#[starts]` ranges), but keep `derive(Action)` and `derive(State)` shaped so variants with data and a `Finite` domain trait slot in at Bank without breaking Kettle.

**Note 2026-10-18 (R8, golden files):** refusals are `trybuild` `.stderr` goldens and the derived output is an insta snapshot formatted by `prettyplease`; no unit tests of the derive.

New proc-macro crate `umpire/dsl/macros` re-exported from `umpire-dsl`: `#[derive(State)]` adds Clone, Eq, Hash and `fn starts() -> Vec<Self>` from `#[starts(expr)]` field attributes (cartesian over annotated fields, defaults for the rest); `#[derive(Action)]` yields the catalog, names and `internal()` from `#[internal]` variants. Use `syn`, `quote`, `darling`; pin messages with `trybuild`.

**Files:** model2/umpire/dsl/macros/{Cargo.toml,src/lib.rs}, model2/umpire/dsl/Cargo.toml, tests/ui/.

**Quick commands:** `mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros`
## Acceptance
- Kettle-shaped struct and enums compile with the derives in DSL-owned coverage; insta expansion snapshots for `derive(State)` and `derive(Action)` checked in (prettyplease-formatted).
- `ui/` goldens for a non-range `#[starts]` and a non-unit `#[internal]` variant.
## Done summary
DSL authors can derive state and action values with canonical Spell output, ordered starts and action catalogs. State generates Clone, PartialEq, Eq and Hash, takes the cartesian product of bounded #[starts] ranges in field order, and defaults unannotated fields. Action generates the same value traits, enumerates unit variants in declaration order, and exposes their internal flags.

Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)
stage: impl-review - ran (codex:gpt-6.1-sol:high; round 1 NEEDS_WORK, same-scope round 2 SHIP; same-family reviewer)

At worker handover, task fn-210-rust-prototype-kettle-on-a-trait-engine.4 remained in_progress for conductor-owned integration, review and completion. Its isolated workspace was .worktrees/fn2104-derives on work/fn2104-derives. The worker base was 44580891599e0e37ceffe7ffee4dc21fddf589b4 and its implementation commit was 1d05204219cac5420008f18b1791fd64b2ae44f1.

baseline: green via handoff (44580891599e0e37ceffe7ffee4dc21fddf589b4; mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros; make model2-check). No paths changed between the verified base and this worktree's initial HEAD. Root logs are .flow/tmp/fn210-4-rolling-baseline.log and fn210-rolling-baseline.log; supplied receipt is 44580891-unittest. No gate skip is claimed.

The new DSL derive fixture first failed because State, Action and their generated methods were absent. Its final golden pins Kettle-shaped declarations, two-field cartesian order, inclusive and exclusive ranges, defaults including a generic String field, an empty start domain, clone/equality/hash deduplication, author names and internal flags. The existing leaf Spell fixture had no manual record/action implementations to migrate and stays byte-identical. Log paths below are relative to this worktree.

Macro fixtures call the actual state::expand and action::expand implementation functions through source modules, format their output with prettyplease and snapshot expansion.rs through the supported umpire-testing runner. Generated output contains only implementations; Rust retains the original declaration. One macro fixture binary runs both the expansion fixture test and the trybuild refusal test. The DSL keeps one fixture binary and its existing UI binary. Shared discovery, determinism checks, aliases, pending snapshots and stale-output checks apply to all new expansion and behavior goldens.

The two trybuild files cover non-range and unbounded starts, and tuple/record payload variants marked internal. manyhow reports both offending spans in each file. Darling validates internal markers. UI diagnostics were read from the first real compiler run before acceptance. No arbitrary derive unit assertions or cargo-expand dependency was added.

Validation passed with CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target.

- mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros. Log .flow/tmp/fn2104-focused-final.log. Both fixture binaries and all four UI cases pass.
- mise exec -- cargo clippy -p umpire-dsl -p umpire-dsl-macros --all-targets --all-features -- -D warnings. Log .flow/tmp/fn2104-clippy-final.log.
- make model2-check. Initial cache-warming log .flow/tmp/fn2104-warmup-gate.log. The final source also passed the full gate inside the actual make model2-time command below, including capability/export controls, protobuf freshness, fmt, Clippy, docs, workspace tests, WASI core builds and CGO-disabled Go parity.
- git diff --check and preservation comparison against the base passed. Existing DSL leaf goldens, shared fixtures, manifests and Cargo.lock are unchanged.

Red evidence is .flow/tmp/fn2104-red.log. Initial real expansion/UI observations are .flow/tmp/fn2104-expansion-ui-red.log. Accepted golden generation is .flow/tmp/fn2104-update.log. All recorded suite statuses came from their command exit codes, with output captured once per invocation.

The first warm macro edit/test loop measured 3.214 s, exceeding R30, with a 2.422 s full gate. A diagnostic repetition measured 2.825 s. Cargo timings showed macro-library compilation, fixture-binary compilation and trybuild's separate macro rebuild. Four redundant Darling structural parser derives projected an already parsed syn AST. Reading that AST directly removed the duplicate parser generation while retaining Darling marker validation, manyhow error aggregation and byte-identical expansion/behavior/UI goldens. The next actual edit/test measurement was 2.220 s and the full gate was 7.534 s. The first breach is retained in .flow/tmp/fn2104-time.log, the diagnostic in .flow/tmp/fn2104-inner-diagnostic.log, and final output in .flow/tmp/fn2104-time-optimized.log. Initial target-cache warming is excluded from warm timings.

Final command was CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target UMPIRE_INNER_CRATE=umpire-dsl-macros UMPIRE_INNER_SOURCE=umpire/dsl/macros/src/lib.rs make model2-time. Exit code was 0. Full actual output follows, with terminal color escapes omitted.

```text
Time the model2 feedback loop (fn-210 R30)...
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model2-check
make[1]: Entering directory '/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2104-derives'
Check model2 (fmt, clippy, docs, tests)...
timing-controls                   0.009s
capability-controls               0.102s
export-controls                   0.072s
protobuf-freshness                0.102s
fmt                               0.078s
clippy                            0.367s
docs                              0.630s
tests                             2.223s
wasm32-wasip1                     0.714s
core-wasm-clippy                  0.648s
wasm-guest                        1.309s
go-host                           1.158s
make[1]: Leaving directory '/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2104-derives'
inner source: umpire/dsl/macros/src/lib.rs
inner command: mise exec -- cargo test -p umpire-dsl-macros --quiet
inner loop (edit+test)            2.220s
gate total                        7.534s  (budget 30s)
inner loop                        2.220s  (budget 3s)
```

Integration seam for .5/.8 is /Users/stephan/Workspace/skunkworks/umpire/temporal/.git/flow-notes/fn-210-20261018T203953Z-root/fn2104-derive-seam.md. It records StateType::starts(), ActionType::actions() and action.internal(), and the need for future lifting to retain the original derive input's fields, start-domain expressions and variants beyond machine impl tokens. Task .4 implements no machine attribute or lift metadata. Payload-domain enumeration and Finite remain future work. Canonical shared Kettle migration remains task .6.

Review fix pass after the conductor's first NEEDS_WORK

The conductor advanced this worktree to 48c18288fab91fb0c1b11a9e9e98945c7ec47500, containing the integrated progress and derive lanes plus their lifecycle records. Finding finding-fcc97c672a161881c438d442ef1055a6 identified accepted empty action enums whose generated Clone and Spell matches failed with E0004. The worker confirmed that exact failure with a new DSL-owned empty_actions fixture and committed the failing reproduction as 333cc0dd967996dd9065937c7df8016a0baa7a2e. Log .flow/tmp/fn2104-review-empty-red.log records exit 101 and the compiler's explanation that references are always considered inhabited.

Fix 59b5c5953e4017622910317255cce5d9b54b89e8 changes the two generated matches to match *self. The new behavior fixture compiles Clone, Eq, Hash, Spell, catalog enumeration and internal classification for an empty enum, and snapshots an empty catalog and zero unique actions. Existing behavior, state expansion and refusal goldens remain byte-identical. The action expansion golden changes exactly those two match expressions. No validation restriction, dependency or public API was added. The fix-pass range is 48c18288fab91fb0c1b11a9e9e98945c7ec47500..59b5c5953e4017622910317255cce5d9b54b89e8. The evidence JSON retains the full oldest-first history since the original base, including conductor-supplied integration and task .3 commits; this worker authored only the original derive commit and the two fix-pass commits.

Final normal focused tests passed with exit 0 in .flow/tmp/fn2104-review-empty-green.log. All-target, all-feature DSL and macro Clippy with -D warnings passed with exit 0 in .flow/tmp/fn2104-review-empty-clippy.log. Snapshot generation is .flow/tmp/fn2104-review-empty-update.log. All log paths in this section are relative to the retained derive worktree. The full gate inside the actual macro timing run passed with exit 0, and both R30 budgets passed. There are no running commands and the worktree is clean. The conductor owns the existing review invocation and task completion; this worker issued no review verdict or lifecycle mutation.

Final fix-pass timing command was CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target UMPIRE_INNER_CRATE=umpire-dsl-macros UMPIRE_INNER_SOURCE=umpire/dsl/macros/src/lib.rs make model2-time. Exit code was 0. Full actual output is .flow/tmp/fn2104-review-empty-time.log and follows with terminal color escapes omitted.

```text
Time the model2 feedback loop (fn-210 R30)...
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model2-check
make[1]: Entering directory '/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2104-derives'
Check model2 (fmt, clippy, docs, tests)...
timing-controls                   0.010s
capability-controls               0.155s
export-controls                   0.085s
protobuf-freshness                0.108s
fmt                               0.115s
clippy                            0.596s
docs                              0.624s
tests                             3.226s
wasm32-wasip1                     0.714s
core-wasm-clippy                  0.647s
wasm-guest                        2.581s
go-host                           1.544s
make[1]: Leaving directory '/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2104-derives'
inner source: umpire/dsl/macros/src/lib.rs
inner command: mise exec -- cargo test -p umpire-dsl-macros --quiet
inner loop (edit+test)            2.505s
gate total                       10.538s  (budget 30s)
inner loop                        2.505s  (budget 3s)
```

The conductor integrated source and the empty-enum fix into umpire at 0492f5dc2f1ab439485b8d962230c2ac759e9d7e. The existing review resumed against base 400bd9062cffd32f1fa7558dccc2a6f1eaf7d0fc and that integrated head. Round 2 returned SHIP with finding finding-fcc97c672a161881c438d442ef1055a6 fixed and no new blocking findings. Review metadata is c5e23473e7f136ff75f4fa23b92093e5eaeab9ee. The canonical receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.4.json; the initial three-lens sidecars are archived at .flow/review-fanout/8091f1e684ce45cc8166ecb8f9903fe9/.

The conductor reran `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros` from model2 at the integrated review-metadata head. Exit 0 is recorded in .flow/tmp/fn210-4-closure-focused.log. The reviewer's own fresh test attempt was inconclusive because its Cargo lock was read-only; this conductor run supplies actual integrated verification.

stage: wave-dispatch - ran (isolated derive lane beside progress; disjoint declared Touches)
stage: wave-join - ran (source, regression fix and review metadata integrated into umpire; focused integrated tests passed)
stage: memory-capture - ran (bug/build-errors/match-enum-values-rather-than-2026-10-18)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1d05204219cac5420008f18b1791fd64b2ae44f1, 333cc0dd967996dd9065937c7df8016a0baa7a2e, 59b5c5953e4017622910317255cce5d9b54b89e8, 0492f5dc2f1ab439485b8d962230c2ac759e9d7e, c5e23473e7f136ff75f4fa23b92093e5eaeab9ee
- Tests: baseline: green via handoff (44580891599e0e37ceffe7ffee4dc21fddf589b4; mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros; make model2-check), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target mise exec -- cargo clippy -p umpire-dsl -p umpire-dsl-macros --all-targets --all-features -- -D warnings, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target make model2-check, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target UMPIRE_INNER_CRATE=umpire-dsl-macros UMPIRE_INNER_SOURCE=umpire/dsl/macros/src/lib.rs make model2-time (final inner 2.220s; gate 7.534s), R30 initial breach: inner 3.214s; fixed by removing redundant structural parser generation; .flow/tmp/fn2104-time.log and fn2104-time-optimized.log, git diff --check, git diff --exit-code 44580891599e0e37ceffe7ffee4dc21fddf589b4 -- model2/umpire/dsl/tests/fixtures/spell model2/umpire/tests model2/Cargo.toml model2/Cargo.lock, review fix red: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target mise exec -- cargo test -p umpire-dsl --test fixtures (exit 101; E0004 for empty action enum), review fix green: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros (exit 0; .flow/tmp/fn2104-review-empty-green.log), review fix Clippy: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target mise exec -- cargo clippy -p umpire-dsl -p umpire-dsl-macros --all-targets --all-features -- -D warnings (exit 0; .flow/tmp/fn2104-review-empty-clippy.log), review fix timing: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target UMPIRE_INNER_CRATE=umpire-dsl-macros UMPIRE_INNER_SOURCE=umpire/dsl/macros/src/lib.rs make model2-time (exit 0; inner 2.505s; full gate 10.538s; .flow/tmp/fn2104-review-empty-time.log), git diff --exit-code 48c18288fa -- model2/umpire/dsl/tests/fixtures/derives/expected model2/umpire/dsl/tests/fixtures/spell/expected model2/umpire/dsl/macros/tests/fixtures/state/expected model2/umpire/dsl/macros/tests/ui (exit 0), Integrated final focused verification at c5e23473e7: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros (exit 0; .flow/tmp/fn210-4-closure-focused.log), Same-scope implementation re-review round 2 SHIP; codex:gpt-6.1-sol:high; canonical receipt /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.4.json
- PRs: