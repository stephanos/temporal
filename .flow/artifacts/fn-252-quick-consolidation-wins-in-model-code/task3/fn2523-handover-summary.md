# fn-252.3 source handover

Status: in_progress; source checkpoint ready for conductor integration and authentic review. No Flow lifecycle, review, milestone, remote or shared index writes performed by this lane.

Commit: `badd05d62150406b6054c21eb98733c7bf3f62cd` on `work/fn2523-spike`, base `4c4e99ca7e89d87f6c64e7b30cdbbb764294d758`. Clean worktree retained at `.worktrees/fn2523-spike` pending conductor integration; remove it and integrated branch at that boundary.

Removed 45 committed regenerable output files (12,895 lines) and both copied Kettle sources; out/out-big are ignored. Baseline includes the actual shared authored fixture, unchanged. build.rs scales its unique threshold declaration to 1_000_000 in OUT_DIR, refusing missing/ambiguous declarations and tracking the actual shared source. Standalone facade dependency and lockfile were updated; no backward compatibility support added.

Necessary existing spike drift repairs: current Value variants now refuse unsupported native witness values explicitly; transparent ExprKind::Check is emitted and traversed for mutations. Default regeneration gives claim99 and heat2e9 separate directories. Reporting separates observed disagreement from incomplete evidence, so timeout/unknown never implies full agreement.

Focused evidence: locked/offline cargo check; six Rust tests covering native baseline/big boundaries, lifted constants/emission, claim99 mutation firing, unsupported witness refusal and incomplete reporting; bash -n; unique threshold guard controls; Git ignore/no-tracked-output/clean-tree checks. The initial locked check exposed a stale separate-workspace lockfile; it was refreshed offline, and final locked checks passed. Existing two dead-code warnings persist.

Actual run.sh baseline/claim99/heat2e9 with rIC3 1.5.2 and Z3 5.3.1 regenerated artifacts. Baseline IC3 safety proofs and a 62-step never_beeps counterexample replay confirm. claim99 produces native-Holds nonconfirmation and disagreement; heat2e9 produces unsafe arithmetic witnesses rejected by native stepping. A final default-selection IC3 rerun confirms distinct directories on final source. Earlier SMT evidence is retained, not represented as typed replay proof.

Actual big run regenerated from the real scaled Model: 2,000,083 native states, depth 999,962, no cut; four IC3 safety proofs. Its never_beeps IC3 check timed out at 10 seconds: NO verdict/proof/agreement for that property. This run preceded the reporting-only fix, so its old final true summary is explicitly invalid as an agreement claim. A focused delayed-rIC3 timeout run on final source proves fullagreement=false, observed disagreement=false, evidence incomplete=true; no duplicate big exploration was run.

Evidence lives at `.flow/tmp/fn2523-evidence`: prior output SHA/path inventory; final regenerated reports, BTOR2, solver logs and replay paths; SMT and big runs; generated scaled source; final release executable; retained solver version/SHA/source provenance; control-checks.json; manifest.json (116 files, SHA256 e8cacf22abb23646ebaa4f7f6933194185beb78e570dcc236c9835125fd1c0a9). Shared solver caches were read/executed, never modified or cleaned.

No complete model gate was run in this lane. The separate Cargo workspace focused checks are complete; conductor owns integration, authentic review, final shared gate, Flow completion and cleanup. No lane commands/processes remain running. Private target `.flow/tmp/fn2523-target` is idle; retained final proof executable is copied to evidence before any later cache reclamation.
