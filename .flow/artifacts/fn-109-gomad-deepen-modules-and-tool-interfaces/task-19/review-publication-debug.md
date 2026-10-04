# Predispatch metadata publication diagnostic

The original cause remains unreproduced. The installed full metadata helper passed 40 isolated calls with the failed attempt's retained inputs. The original caller discarded its exception, so `sidecar_publish_failed` does not identify a filesystem operation or errno.

The failed attempt used reservation `28918d38c5ed438ea2933f6402f7a28d`, base `b7c26a2a5cf15fb3626b8114abdd44ccb2dd7005`, head `4a646710d15148f1a3cf75bdeba7bdfd2fd2edf7`, and artifact hash `6370c01ecf1f544291552c381b1b6aceb5313ffa38a0572f48bea532db3fcb62`. The spec records a transport failure with `round_consumed=false`; pending rounds and reservations are empty. The original sidecar currently contains only the root's two harmless diagnostic files. It contains no `meta.json`, draw files, or temporary publication files.

In `/home/agent/.codex/scripts/flowctl.py`, `_review_fanout_write_meta` at line 47502 constructs the metadata and serializes it with `json.dumps` before `_review_fanout_publish` runs. The initial caller at lines 48653-48673 catches every `Exception`, replaces all three pending rows' failure classes with `sidecar_publish_failed`, and refunds the reservation. It neither prints nor persists the exception type, message, errno, or traceback. The label can therefore cover metadata serialization as well as exclusive create, write, flush, fsync, and replace. The later `_review_fanout_dispatch` call at line 48674 was never reached.

Source inspection found JSON-compatible inputs throughout this invocation. The CLI parses base, receipt, focus, and task as strings; resolved task IDs and backend names are strings; the captured SHAs and artifact hash are strings; task-mode claim token and focus are `None`; pending rows contain strings, booleans, and `None`. This inspection supplies no concrete serialization defect.

I ran `python3 /tmp/review-publication-diagnostic.BXG7P3/probe.py` from the repository. The script loads the installed source with `runpy`, then calls the original `_review_fanout_write_meta` with the retained task, base, head, hash, reservation, receipt path, primary axis, and three initial `dispatch_interrupted` rows. It creates each isolated directory with mode 0700 immediately before the call, validates the published JSON, and checks that no temporary file remains.

- `/tmp/review-publication-diagnostic.BXG7P3` passed 20 of 20 calls.
- `/Users/stephan/Workspace/skunkworks/review-publication-diagnostic.k6TYH4` passed 20 of 20 calls on the project filesystem.
- Python was 3.14.4. Installed source SHA-256 was `1cffb34a9b0ea75f1b4ea904042c995db4845a89796b5af371a59aaccfc2e540`.

These calls tested metadata construction and the original exclusive-create/write/flush/fsync/replace sequence. They did not invoke the CLI handler, reserve rounds, write actual review state, or dispatch draws. They confirm present operation; they cannot recover the discarded historical exception. A filesystem transient, capacity error, permission error, serialization error, or temporary-file collision is not established by the retained evidence.

The next diagnostic action should capture the original exception type, message, errno, and traceback at the initial metadata boundary if the failure recurs. Repeating only the installed CLI's generic failure output cannot resolve the cause. The scratch helper replay above is presently green; this diagnostic does not authorize a formal retry or a tool modification.

Only this report and isolated scratch files were created. Installed tools, source, Git/index, ledger, receipts, and existing sidecars were not changed. No review verdict or additional judgment was issued. The requested thinking-scout route was Codex `gpt-6.1-sol` at high; actual child model metadata was unavailable. All commands exited; no terminal handle remains.
