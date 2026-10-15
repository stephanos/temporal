# fn-176 frozen qualification checkpoint

Qualification was stopped for a production recovery validation fix, not accepted. Tasks .1 and .2 are verified Flow
done; .3–.7 and whole-spec completion remain open.

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
Correctness and integration draws were SHIP, but this contracts finding prevents
task completion. The worker is reproducing and fixing it under the same receipt.

The conductor terminated the verified qualification process group after ordinary
`test-wasm` passed and retained-durable-restart started (session exit 143). The
production fix invalidates the test-only preservation argument for the final
candidate. A new frozen candidate must run the required execution gates; neither
this partial run nor the earlier helper-equality check qualifies that new source.

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
