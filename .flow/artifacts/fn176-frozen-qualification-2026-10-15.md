# fn-176 frozen qualification checkpoint

Qualification is in progress, not accepted. Tasks .1 and .2 are verified Flow
done; .3–.7 and whole-spec completion remain open.

## Source and driver

- Branch: `gomad`; implementation merge `cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d`.
- Frozen revision: `500e4d921d6e68fc623fdde3da45e9a347803855`.
- Frozen worktree: `/Users/stephan/Workspace/skunkworks/gomad-fn176-frozen-20261015`.
- Current evidence: `/Users/stephan/Workspace/skunkworks/gomad-fn176-qualification-native-tmp-20261015.A4gFnR`.
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
tests/vet, workflow lint, root/module lint and Rust formatting passed. Rust release
tests are running. Helper production, ordinary WASM, terminal exit, runtime
32×25/32 exact replays, retained replays, installed operator, frozen committed
crash/reopen, network, independent clusters, ten primary workload cells and
aggregate, four SQLite selected states and catalogue remain unverified in this
attempt. The driver checks source/compiler/helper preservation around stages.

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
