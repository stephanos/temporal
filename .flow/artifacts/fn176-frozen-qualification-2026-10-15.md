# fn-176 frozen qualification checkpoint

## Accepted current frozen run

The fresh frozen run passed at `2026-10-15T09:02:41Z`: execution session 4485
returned exit 0, all 84 unique sequential START/EXIT stage pairs have numeric
exit 0 and exactly one final `PASS frozen-qualification module=gomad` is retained.
The conductor independently parsed the ledger and confirmed driver PID/PGID/SID
13965 was absent after completion. Task 7 implementation review and whole-spec
completion review remain separate gates.

- Qualified source: `bfb53b2c4585f39b70e0e361c45f3c3b2af49e5e`.
- Evidence root: `/Users/stephan/Workspace/skunkworks/gomad-fn176-qualification-live-20261015.1hg1RG`.
- Frozen worktree: `/Users/stephan/Workspace/skunkworks/gomad-fn176-frozen-20261015`.
- Source digest: `sha256:952dd5dec7f1190861ecae3c6994a0af378654826b948fb1870bcd4631b2e9cb`.
- Driver SHA256: `5f75b4ff9bd227abfe547ea7587789e84a7d5a9bdc3ed59f26bfcf6dc3b181cd`.
- Authenticated producer/admitted helper SHA256: `de2bbf6c1a9c4fec75b8e9a8d3a71027e3fc0d6df71500560c1c056c52764f99`.
- Go model: `sha256:37a249b97a6fdd3c716032769fa0836a64957cea585633e071be92d9ae46efe4`.
- Stock Go executable SHA256: `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`.
- Rust compiler SHA256: `0af7dded06df00663679b7088aaf375eafe42b475f0ce3a86d7f40074b753267`.
- Host: Linux `my-codex-build-v3`, kernel `7.0.14`, aarch64; actual CPU and terminal
  host admission evidence is retained in the root's host/binding files.
- Native temporary directory: `/tmp/gomad-fn176-qualification-live.Uy7SIz`.

Exact commands and working directories are retained in `<stage>.command`, raw
observations in `<stage>.log`, and numeric exits/timestamps in `stages.log`.
`revision`, `source.sha256`, `source.digest`, `compiler.sha256`, `helper.sha256`,
`terminal-host-binding.json` and `DISPOSITION.md` retain provenance and the final
conductor checks. `RUNNING.md` is retained only as the historical launch record.

| Required gate | Current frozen observation |
| --- | --- |
| Generated source, architecture/static, portable and genuine compile-only | All passed, including retained native source/pin/preservation controls; compile-only uses `go test -exec /bin/true`. |
| Router/workflow/root/module lint, Rust fmt, both release test and clippy configurations | All passed. Fixed lint base `a1f0be7e96e2746cb9c51598766a5ae96d8552c2`. |
| Authenticated release-helper producer and ordinary `make test-wasm` | Passed; ordinary suite exited 0 at 08:15:36Z and includes both refused-run parity and strict partial-envelope controls. |
| Fresh retained durable restart and cluster replays | Both passed on new helper-produced evidence; corruption controls remain fail-closed. |
| Actual `./tests` terminal producer, host admission and terminal-to-reap | Passed; success 56,566,959 ns and post-load authorization mismatch 5,387,708 ns, both reaped within 1 second. |
| Runtime qualification | 800 complete fresh recordings and 32 fresh forced exact replays passed; module `11aca2dfc37bd17918531bb0e5bcef2d6b50283bedda31d0c6a1c530c4013603`. |
| Retained integrated failure artifact and installed operator | Passed, including source-less discovery/inspect/replay. |
| Temporal committed workflow crash/reopen | Passed in 320.278 seconds; complete enumeration and all fresh reopens match stdout, storage evidence and interrupted choices. Images 6,513,151 B at crash and 7,460,217 B after recovery. |
| Temporal named network reset/retry and independent clusters | Both passed. |
| Ten primary cells and corpus aggregate | All passed their expected dispositions: five qualified successes and five intentionally unqualified canaries. Passing canary controls do not qualify their workload results. |
| Four actual WASM SQLite selected states | All passed on authenticated current-candidate source results: indices 0–3 exited 0 at 09:01:03Z, 09:01:34Z, 09:02:04Z and 09:02:34Z. |
| Relocated corpus aggregate and authenticated SQLite catalogue | Both passed; catalogue exited 0 at 09:02:39Z. Its inherited native wording describes a projection used by actual WASM, not native execution. |
| Final preservation | Source/compiler/helper manifests rechecked, frozen worktree clean, primary-before/after hashes equal, qualified source unchanged on gomad, all 18 external planning hashes unchanged/unstaged. |

The qualifying helper hash above belongs to the frozen authenticated producer;
the worker's earlier focused helper hash `42ef0937...` remains a separate build
observation in the task 7 source artifact. Administrative receipt `3aeea7936d`
changes only historical Flow tracking, with no qualified-source difference from
the frozen candidate. Later tracking/receipt updates do not requalify source.

The prepared `lxd3oV` root was launched incompletely: its setsid launcher detached,
later process checks found no driver/stage, and its stage log is empty. Its
`DISPOSITION.md` preserves this history without a pass. The accepted `1hg1RG`
replacement uses `setsid --wait`, a fresh evidence root and native scratch.

Source fixes and red-to-green proof are in
[task 7 acceptance](fn176-task7-acceptance-2026-10-15.md); both verbatim quality
reports and conductor dispositions are in
[the cross-task audit](fn176-task7-cross-task-quality-2026-10-15.md).
Only actual native-backend execution remains deferred under fn128/fn149.
No push, PR or CI action is authorized or claimed.

## Historical stopped candidate (not acceptance)

The source-500e4d qualification below was stopped for a production recovery
validation fix and remains unaccepted. Its then-current task states and required
restart are historical observations, superseded for current acceptance by the
fresh run above.

## Source and driver

- Branch: `gomad`; implementation merge `cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d`.
- Frozen revision: `500e4d921d6e68fc623fdde3da45e9a347803855`.
- Frozen worktree: `/Users/stephan/Workspace/skunkworks/gomad-fn176-frozen-20261015`.
- Stopped evidence: `/Users/stephan/Workspace/skunkworks/gomad-fn176-qualification-native-tmp-20261015.A4gFnR`.
- Source manifest digest: `sha256:df3f7532f151ab06b106d8aba20b649f2a717b446bb13df9f6ffbda2d44d8633`.
- Driver SHA256: `e677ee71961b6a31b88e9196a2b1796ec85f6f83681452e772264b8ad3ec683e`.
- Host: Linux/aarch64; bounded CPU evidence and compiler hashes are in the evidence root.
- Test temporary files use native Linux `/tmp/gomad-fn176-qualification-scratch.XEZeZU`.

Exact invocation (the driver is one-shot; a new attempt needs a new evidence root):

```bash
env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/gomad-fn176-qualification-native-tmp-20261015.A4gFnR/run-qualification.sh 500e4d921d6e68fc623fdde3da45e9a347803855 a1f0be7e96e2746cb9c51598766a5ae96d8552c2
```

Validation, all-platform architecture, portable model/evidence/host/target suites,
genuine compile-only, retained native source/pin/preservation checks, lint-router
tests/vet, workflow lint, root/module lint, Rust formatting, both release test and
Clippy configurations, terminal controls, release-helper production and ordinary
WASM passed. Retained durable restart started but was interrupted. Terminal exit, runtime
32×25/32 exact replays, retained replays, installed operator, frozen committed
crash/reopen, network, independent clusters, ten primary workload cells and
aggregate, four SQLite selected states and catalogue remain unverified in this
attempt. The driver checks source/compiler/helper preservation around stages.

## Review finding and required restart

Task .3 added malformed-image conformance inputs at `79362d2ece`: model mismatch,
persisted lock path, rehashed truncation and digest mismatch are refused, without
mutating storage state; valid recovery afterward succeeds. Focused Go/Rust gates
and lint pass. The conformance-only change preserved the default release helper
SHA256 `b28096ccc5cda4170cdce90de74e5cc01523c2208454db0baf6b8205d031afc1`.

The fresh `gpt-6.1-sol/high` contracts review then found a production defect: a
correctly rehashed image can assign two persisted paths to one inode within a
volume, creating independent recovered nodes with inconsistent inode-based writes.
The failing reproduction was committed at `e6b912b441`; `6476da1cd1` rejects repeated
path inode references before rebuilding. Rehashed file and directory aliases now
refuse, and six Go/Rust cross-directory rename cases preserve valid recovery with
no parent sync or either parent synced. The final 54 Rust volume tests, full
conformance corpus, focused Go storage suite, clippy, formatting and lint pass.
The same-receipt `gpt-6.1-sol/high` re-review returned SHIP and task .3 is verified
Flow done. Its handover is `.flow/tmp/handovers/fn-176.3-summary.md`.

The conductor terminated the verified qualification process group after ordinary
`test-wasm` passed and retained-durable-restart started (session exit 143). The
production fix invalidates the test-only preservation argument for the final
candidate. A new frozen candidate must run the required execution gates; neither
this partial run nor the earlier helper-equality check qualifies that new source.
The corrected default helper SHA256 is
`ccee6cd032ac079851a9f5612589de4b01040e46467c0926845e3e934873075c`.

## Failed preparation attempts

1. `/Users/stephan/Workspace/skunkworks/gomad-fn176-qualification-20261015.3YG7JZ`
   ended exit 1 at architecture. The conductor-created `.toolchain` directory
   symlink was rejected as uninventoried source. It was replaced by an ordinary
   directory; no tracked source changed. A separate audit also corrected the
   driver's compile-only command to `go test -exec /bin/true`, avoiding the
   deferred native CLI TestMain.
2. `/Users/stephan/Workspace/skunkworks/gomad-fn176-qualification-retry-20261015.08HJXq`
   passed validation and all-platform architecture, then ended exit 1 at portable:
   `TestPruneTargetPoolRemovesOnlyEntriesNothingShares` retained an unshared pool
   entry. The exact selector failed 5/5 with TMPDIR on the workspace virtiofs
   mount and passed 5/5 with TMPDIR=/tmp, using unchanged source/compiler/cache.
   The current full attempt uses the native Linux temporary filesystem and its
   complete portable suite passes. No pruning assertion or product code changed.

Original scripts, logs and dispositions are preserved. Neither failed attempt
is a qualification pass. Native execution remains deferred to fn-128/fn-149;
SQLite selected-state and catalogue controls execute WASM and remain mandatory.
