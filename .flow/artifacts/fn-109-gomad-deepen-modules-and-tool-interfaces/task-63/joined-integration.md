# Joined task 63 and task 10 source progress

This checkpoint joins the independently reviewed local-campaign extraction, the Runner coverage inventory and a fresh refresh-help observation. It is source progress, not task completion or full-host qualification. The primary base is `e5cbeb22b6`; the joined candidate before this note is `c741cb50fc8b2f333b9142cf99502e3f027527f7` on `gomad-fn10963-task11210-joined-20261010`.

## Checkpoint provenance

| Original worker checkpoint | Joined checkpoint | Scope |
| --- | --- | --- |
| `2d5a19f0b55664f534a853a3cd011b0d90cac1cb` | `e7cf32740509c82ae6ff5fadaad0cca51cbb3c75` | Task 63 source extraction and retained evidence |
| `1af406663d1304453a70b370c22fef92778808c9` | `ddf8a602e5` | Disjoint Runner inventory admission |
| `fd76e5697b85d7512897862ecf037669f5ccd37e` | `92ef91d7120e4104c3e61e023d0e8af52e88aafe` | Runner inventory and independent reviews |
| `48a8b8e4d9b920a26a9bdee65e724d019bc93765` | `c741cb50fc8b2f333b9142cf99502e3f027527f7` | Fresh refresh-help capture and independent review |

The existing task 10 source-only reconciliation was already integrated in primary at `9a9ca290700ee81584163c3b109c1296b2f327fd`, with a separate primary-source binding at `056a2fb807`. Normalization preserves worker file bytes and history; historical receipts still name their actual execution candidates, not these joined commits. Worker handovers saying “uncommitted” describe their capture-time state, superseded only for checkpoint status by the table above.

## Joined source identity

The product diff against the primary base is exactly these three paths under `tools/gomad3/runner`:

| File | SHA-256 |
| --- | --- |
| `runner.go` | `1593fdbad9c3cd318e064fe4d7977ac7cfbd5044e4b72cbffc09a10dc1cfa46c` |
| `runner_local.go` | `1e11c499ed4b274e77aca0261f9c94d6a8ea05b5587fdf43baa8ddacc408b432` |
| `runner_local_test.go` | `6a8db7341cc00b1ca39e595bb0d4fe087d6e84099844f48154bb111f440ac320` |

The joined `tools/gomad3` tree is byte-identical to the reviewed task 63 worker checkpoint. All 1,069 other entries in `../combined-60-62/sources-fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297.json` match their retained hashes. Replacing its old `runner.go` entry and adding the two new files yields 1,072 paths. This is a new candidate, not the old 1,070-input execution fingerprint. The old aggregate verifier and inventory execution binding cannot qualify the changed Runner by excluding its required input.

Task 63's original `packet.sha256` remains unchanged (`d81e18651997aaeb236a3c86358bf7f43e024e47f9985aa53c080142d0d2f2fb`). Its successful verification does not bind the subsequently added independent-source review or this integration note. Those are separately committed records.

## Evidence interpretation

Task 63 extracts the original local orchestration into 17 private functions, largest 96 lines. The original Runner named outcomes remain 345 pass, 288 fail and 12 skip; 13 new phase-control outcomes pass. Original CLI outcomes remain 432 pass and three fail. Required original-base lint remains red with 53 findings. Diff-filtered fast lint, source/static controls and passing phase controls do not replace full ordinary acceptance or native qualification.

The task 10 inventory accounts for all 123 failed top-level Runner tests in the retained pre-extraction run. Its final SHA-256 is `cca3d8d4e39bfc4168d7f2d15973e24fdcbe0d2bd0c384b58a42660050f1c165`; the historical first review binds the earlier JSON bytes and the separate whitespace review binds the final one-LF correction. The earlier 109-fixture/14-exclusion audit was a hypothesis, superseded by the source inventory: 12 intended seeded-target cases, three compiler/preparation cases, five mixed default/private cases, one ordinary crash subprocess and 102 other scripted cases. Portable assertions stay source-owned; real execution and native qualification are separate dispositions. This classification is not new test execution or authorization for blanket fake preparation.

The fresh refresh-help observation used one stock Go build (exit 0) and one `compatibility-pack refresh -h` invocation (exit 2), with empty stdout and 616-byte stderr SHA-256 `42c826d83aea0bcd282e9cab26f5026bdd076f2c551cba950280536e9a2e5443`. It does not retroactively supply the two missing historical raw observations. Generated compiler postimages and incomplete C/system-header prebinding remain disclosed. Five space-before-tab diagnostics belong to exact Go-generated raw stderr; its bytes are preserved, not normalized or treated as a source lint-policy exception. The source-only reconciliation and older observations remain immutable.

All worker execution handles are terminal. Joining evidence did not rerun Go, tests or lint. Fresh independent integration readers `joined63_source_identity` and `joined112_evidence_identity` verified the candidate at `c741cb50fc8b2f333b9142cf99502e3f027527f7` in parallel, with no integration findings. Their read-only Git comparisons, SHA-256 checks and packet inspection confirm the source and artifact identities above. Task 63's sealed 146 entries pass, with the later review outside that seal. The evidence reader independently reproduced the whole refresh commit whitespace check's exit 2 and exactly five raw-stderr diagnostics. No failed source gate was rerun or relabeled. Requested reviewer routing was `gpt-6.1-sol` at high effort, in a fresh context and the same GPT family as the writers; actual model telemetry was not exposed.

The earlier independent reviews remain bounded source-progress reviews, not formal SHIP verdicts. Task 63 and task 10 remain in progress. Native fn-128/fn-149 owners remain deferred and qualification unverified; no push, PR, CI or native execution is authorized by this checkpoint.

The primary checkout's unrelated owner amendments, new specs and Turbo documents are not part of this join. The isolated conductor was used while primary's index lock was present; no lock was removed or bypassed.
