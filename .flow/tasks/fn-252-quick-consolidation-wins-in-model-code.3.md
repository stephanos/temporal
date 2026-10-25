---
satisfies: [R3]
---
# fn-252-quick-consolidation-wins-in-model-code.3 Delete committed spike output; no copied Kettle in the spike

## Description
Implement **Spike output (A6)**: delete `design/native-symbolic/out/` and `out-big/`, ignore them, and include the shared sample_kettle model by `#[path]` instead of the copies.

## Focused source-lane contract

Owner-authorized early source lane (2026-10-25), ahead of whole fn248. Parent fn252 keeps its whole-fn248 prerequisite, and no other fn252 task is admitted by this work.

**Touches:** [model/design/native-symbolic/**, model/umpire/dev/fixtures/fixtures/sample_kettle/model.rs]

Reuse the actual shared sample_kettle source without changing its baseline threshold100. Generate the big threshold1_000_000 variant privately, with missing/ambiguous declaration guards. Preserve distinct claim99/heat2e9 controls and strict native witness refusal. Necessary current frontend/witness API repairs and standalone manifest/lock reconciliation are within scope. Regenerate actual baseline/control outputs and retain source/solver evidence; incomplete/timeout is not solver agreement. Focused checks: `bash -n model/design/native-symbolic/run.sh` and locked offline Cargo check/test for `model/design/native-symbolic/Cargo.toml`, with formatting. Retain prior actual solver runs with their exact source limitations rather than repeating big exploration or running a full Model gate. Whole-spec R9/current guest publication remain deferred.

## Acceptance
- [ ] No committed spike output; `run.sh` still regenerates it.


## Done summary
# fn-252.3 source handover

Original worker handover: source checkpoint was in_progress before conductor integration and authentic review. No Flow lifecycle, review, milestone, remote or shared index writes performed by this lane.

Commit: `badd05d62150406b6054c21eb98733c7bf3f62cd` on `work/fn2523-spike`, base `4c4e99ca7e89d87f6c64e7b30cdbbb764294d758`. Clean worktree retained at `.worktrees/fn2523-spike` pending conductor integration; remove it and integrated branch at that boundary.

Removed 45 committed regenerable output files (12,895 lines) and both copied Kettle sources; out/out-big are ignored. Baseline includes the actual shared authored fixture, unchanged. build.rs scales its unique threshold declaration to 1_000_000 in OUT_DIR, refusing missing/ambiguous declarations and tracking the actual shared source. Standalone facade dependency and lockfile were updated; no backward compatibility support added.

Necessary existing spike drift repairs: current Value variants now refuse unsupported native witness values explicitly; transparent ExprKind::Check is emitted and traversed for mutations. Default regeneration gives claim99 and heat2e9 separate directories. Reporting separates observed disagreement from incomplete evidence, so timeout/unknown never implies full agreement.

Focused evidence: locked/offline cargo check; six Rust tests covering native baseline/big boundaries, lifted constants/emission, claim99 mutation firing, unsupported witness refusal and incomplete reporting; bash -n; unique threshold guard controls; Git ignore/no-tracked-output/clean-tree checks. The initial locked check exposed a stale separate-workspace lockfile; it was refreshed offline, and final locked checks passed. Existing two dead-code warnings persist.

Actual run.sh baseline/claim99/heat2e9 with rIC3 1.5.2 and Z3 5.3.1 regenerated artifacts. Baseline IC3 safety proofs and a 62-step never_beeps counterexample replay confirm. claim99 produces native-Holds nonconfirmation and disagreement; heat2e9 produces unsafe arithmetic witnesses rejected by native stepping. A final default-selection IC3 rerun confirms distinct directories on final source. Earlier SMT evidence is retained, not represented as typed replay proof.

Actual big run regenerated from the real scaled Model: 2,000,083 native states, depth 999,962, no cut; four IC3 safety proofs. Its never_beeps IC3 check timed out at 10 seconds: NO verdict/proof/agreement for that property. This run preceded the reporting-only fix, so its old final true summary is explicitly invalid as an agreement claim. A focused delayed-rIC3 timeout run on final source proves fullagreement=false, observed disagreement=false, evidence incomplete=true; no duplicate big exploration was run.

Evidence lives at `.flow/tmp/fn2523-evidence`: prior output SHA/path inventory; final regenerated reports, BTOR2, solver logs and replay paths; SMT and big runs; generated scaled source; final release executable; retained solver version/SHA/source provenance; control-checks.json; manifest.json (116 files, SHA256 e8cacf22abb23646ebaa4f7f6933194185beb78e570dcc236c9835125fd1c0a9). Shared solver caches were read/executed, never modified or cleaned.

No complete model gate was run in this lane. The separate Cargo workspace focused checks are complete; conductor owns integration, authentic review, final shared gate, Flow completion and cleanup. No lane commands/processes remain running. Private target `.flow/tmp/fn2523-target` is idle; retained final proof executable is copied to evidence before any later cache reclamation.

Conductor actual premerge base 8dfb781724d9d05a52f19bddf78da166dbe7a9d4, merge4a2df700ab and joined lock/format/evidence commit6a8d3ee660. The standalone lockfile initially refused --locked after joined native dependency changes; offline refresh adds no package/version, removes the enum-iterator pair and reconciles already-resolved local edges. Final locked check and six focused controls pass. Rustfmt initially refused inherited layout in spike sources; layout-only normalization and final focused test/fmt checks pass. Shell syntax passes. Two inherited dead-code warnings remain.

All116 original proof-manifest hashes were verified;114 text reports/logs/BTOR2/witness/provenance files are durable under task3/evidence, and two retained ELF controls are referenced by SHA/path in task3/binary-references.json. Actual worker baseline/control regeneration and the two-million-state big run retain their original source/solver provenance. The big never_beeps IC3 timeout is inconclusive: no property verdict/proof/full agreement. Its old pre-report-fix final true flag is invalid as an agreement claim; final-source delayed-solver control demonstrates incomplete=true and disagreement=false. No duplicate big exploration or current joined solver run is inferred from six unit controls.

stage: impl-review - ran (codex:gpt-6.1-sol:high, same family; all three fresh readonly draws SHIP, no findings; authentic receipt in task3/review)

Parent fn252 retains the whole-fn248 dependency; whole-spec R9, full Model pipeline and current guest publication remain deferred. Only the explicitly admitted .3/.4 lanes ran early. This is the last assigned task; no new task or parent closure follows.

Secondary reviewers independently ran the retained six-test executable successfully and verified all114 archived text artifact hashes. Fresh reviewer Cargo attempts remained blocked before execution by readonly build-cache locks; conductor current-source checks exited0.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: badd05d62150406b6054c21eb98733c7bf3f62cd, 4a2df700ab7efd8951a60a4c52f6379df27da35c, 6a8d3ee660a394898f85fd7f2d8813b6cbe9eddf
- Tests: bash -n model/design/native-symbolic/run.sh, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2523-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo check --locked --offline --manifest-path design/native-symbolic/Cargo.toml, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2523-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test --locked --offline --manifest-path design/native-symbolic/Cargo.toml, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2523-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo fmt --manifest-path design/native-symbolic/Cargo.toml --check, git diff --check
- PRs: