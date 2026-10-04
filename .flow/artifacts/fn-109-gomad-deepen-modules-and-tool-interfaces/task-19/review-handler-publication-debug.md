# CLI handler metadata diagnostic

The original CLI handler published initial metadata successfully in two isolated runs, one on `/tmp` and one on the project filesystem. No exception appeared at the metadata boundary. The historical `sidecar_publish_failed` remains unreproduced.

The shell used the installed wrapper's exact interpreter selection order and minimum-version probe. It selected `/usr/bin/python3`, Python 3.14.4, matching the first diagnostic. `PYTHON_BIN`, `FLOW_STATE_DIR`, and `REVIEW_RECEIPT_PATH` were unset in the current environment. The wrapper executes `flowctl.py` directly for this multiargument command; its optional bootstrap file is absent. Shell SHA-256 checks before and after the diagnostic gave `1cffb34a9b0ea75f1b4ea904042c995db4845a89796b5af371a59aaccfc2e540`. The failed invocation's interpreter and environment were not independently retained, so current selection alone cannot prove their historical values.

The scratch script is `/tmp/review-handler-publication-diagnostic.McZ6AX/probe.py`. It executed the installed source with `runpy.run_path(..., run_name='__main__')`, so the original `main()` parsed the task-mode CLI arguments and called the original wrapper and `_codex_impl_review_fanout`. The script installed these explicit process-local replacements at entry to `main()`.

| Replaced function | Isolated behavior |
| --- | --- |
| `get_repo_root` | Returns the selected fresh scratch root. |
| `_capture_review_snapshot` | Returns the retained base and head strings. |
| `_gather_review_scope` | Returns a scratch scope string without Git. |
| `_gather_review_identity_diff` | Returns a scratch identity string without Git. |
| `_review_fanout_build_prompts` | Returns one inert prompt string per axis. |
| `_review_artifact_hash_or_warn` | Returns the retained artifact hash string. |
| `enforce_and_increment_review_cap` | Returns `(1, retained_reservation_id)` without reserving anything. |
| `_review_fanout_journal_refund_intent` | Returns `None` without writing a journal. |
| `record_review_attempt` | Raises a diagnostic stop if reached. |
| `subprocess.run` and `subprocess.Popen` | Raise a diagnostic stop if any process launch is attempted. |

The original scope resolver, task canonicalizer, first-round guard, backend-spec parser, draw parser, primary-axis selection, caller argument construction, replay-result handling, sidecar-directory creation, `_review_fanout_write_meta`, and `_review_fanout_publish` executed unchanged. Scratch task/spec files were created through `apply_patch`. The receipt argument retained its CLI string type and used an absent scratch path. Original sidecar creation wrote only scratch directories and their managed `.gitignore`.

`sys.settrace` observed one metadata call per run. `sidecar` was a `PosixPath`; task, base, SHAs, artifact hash, reservation ID, rid, primary axis, and receipt were strings; `standalone` was `False`; results were the three original pending draw dictionaries; focus and claim token were `None`. These types match the earlier direct-helper fixture. The tracer watched exception events in the caller and both publication helpers. It recorded zero such exceptions and stopped at the `_review_fanout_dispatch` call before any original dispatch statement executed.

Both `/tmp/review-handler-publication-diagnostic.McZ6AX/.flow/review-fanout/28918d38c5ed438ea2933f6402f7a28d/meta.json` and `/Users/stephan/Workspace/skunkworks/review-handler-publication-diagnostic.z37mQP/.flow/review-fanout/28918d38c5ed438ea2933f6402f7a28d/meta.json` were published. Each run reached the stop boundary, and the shell command exited 0. Thus the result is two of two handler-path successes, in addition to the earlier 40 helper successes.

This test omitted actual Git evidence gathering, prompt assembly, artifact assembly, reservation persistence, and refund-intent persistence. It did not reproduce the production directory state or the historical process environment. No concrete cause can be assigned to those omitted steps from this result.

The next justified diagnostic step, if an authorized real invocation recurs, is exception tracing at `_review_fanout_write_meta` and `_review_fanout_publish` before the caller's blanket catch discards the type, errno, message, and traceback. Additional identical mocked replays have no demonstrated value. This report grants no dispatch or state-mutation authority.

The first report remains unchanged. No actual repository state, ledger, reservation, receipt, sidecar, source, installed tool, or Git/index was changed. No reviewer process, verdict, model switch, or additional judgment occurred. Requested routing remains Codex thinking scout `gpt-6.1-sol` at high; actual child model metadata is unavailable. All commands are terminal; no terminal handle remains.
