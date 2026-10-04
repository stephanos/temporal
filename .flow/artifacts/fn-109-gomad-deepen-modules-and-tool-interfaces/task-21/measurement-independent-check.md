# Independent read-only measurement check

The measurement scout verified the current and retained baseline artifacts
without writing files or launching campaigns. Its complete-metric reconstruction
matched all 32 raw profiles, all leaf sites and all size buckets. Current 98
command receipts have terminal exit 0 and the same 14 bindings. Current source
982, original source 670, bound scratch 675 and bound artifact 202 complete
path/hash/mode inventories matched the protected manifests at the check.

All four binary build-setting records match except their paths. Every paired
gate is `0/0/2`, `N-2/N-2/2`, then `N/N/0`. The current transport counter has no
guard call sites, unlike baseline's two guarded Dup2 paths; the fixture's
injected preparer/executor establishes the transport exclusion. Private novelty
cardinality and arbitrary transient copies remain outside the measurement.

The scout independently found current logical accounting 4472 versus baseline
4120 bytes, constant at both N values. The 352-byte change belongs entirely to
four completion slots. Current direct-runLocal late live allocation is 7432
for discard-10 and 7448 for the other three cases, so exact N-invariance of
that aggregate is unsupported.

## Exact publication stacks

The following samples occur in both novel-10 and novel-100 completed profiles.
Each buffer is 65536 bytes and all have zero live bytes at completion.

| Caller/path | Baseline objects/bytes | Current objects/bytes |
| --- | --- | --- |
| Six inline payload writes | 6 / 393216 | 6 / 393216 |
| Manifest write | 1 / 65536 | 1 / 65536 |
| Source target copy | 1 / 65536 | 1 / 65536 |
| Shared-target SHA256 verification read | 0 / 0 | 1 / 65536 |

Baseline inline stack is `copyWithContext store.go:347`, `writePayload:331`,
`Store.PublishArtifact:112`; manifest uses publication line 138 and source
target copy uses `copyPayload:276`, publication line 110. Current inline stack
is `copyWithContext store.go:421`, `writePayload:405`, `placePayload:249`,
publication line 130. Manifest uses publication line 156. Target copy uses
`copyPayload:350`, `placePayload:247`, `createPoolEntry target_pool.go:192`,
`placeSharedPayload:73`, publication line 128. The extra stack is
`verifySharedPayload target_pool.go:226`, `placeSharedPayload:67`, publication
line 128; it opens through `openSharedFile` and streams into a SHA256 hasher.

Current target bytes are `fake prepared target`, length 20. Its inode equals
the one target-pool entry and link count is 2. Stdout, stderr and transcript
are each 1048576 bytes; World payloads are 516, 290 and 805 bytes. Current
manifest write is 7968 bytes versus baseline 7967. Git blame/show identifies
`bc2e970b5306aa09f594657a8d42c159cf4a1270`, subject
`gomad: keep one copy of each prepared target per artifact store`, Task
`fn-114-gomad-correct-search-path-defects-and.9`, as the shared-target verifier
owner. No `openSharedPayload` symbol exists; the actual names above are retained.

The extra bounded allocation supports a filesystem target-verification read,
not a new complete 1MiB payload heap clone. Completed profiles supply the final
per-execution denominator. Publication count is one in each novel fixture.
These claims apply to the observed fixture and source sites only. This check
supplies no review verdict or native qualification.
