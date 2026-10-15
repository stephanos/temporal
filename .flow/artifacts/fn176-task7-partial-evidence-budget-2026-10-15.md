# Partial storage evidence budget correction

Task 7's first review returned NEEDS_WORK with one contracts finding: a permitted
history can exceed the recovery-image cap, so that image-derived cap cannot also
bound refused-run operation JSON. Integration returned SHIP; the correctness
draw failed without a verdict and was not retried. The delivered contracts
verdict consumed round 1 and stays on the same review receipt. The green frozen
run at `bfb53b2c45` remains valid only for that historical source; the correction
requires a fresh candidate and frozen run before same-receipt re-review.

## Actual-owner reproduction

`TestStorageEvidenceLargeRefusedHistoryOwnerParity` performs 100,000 one-byte
`fd_pwrite` overwrites. Both owners use a 64-KiB filesystem/volume capacity,
200,000 calls, a 128-MiB transcript allowance and 200,000 pending operations and
transitions. Before the fix, Go retained the prefix and returned `capacity` /
`WASI capacity: storage-image-bytes`; helper admission returned `capacity` /
`WASI capacity: helper evidence bytes` and discarded the prefix. The failing
actual-owner checkpoint is `4d2a77bb02`; raw observation is
`.flow/tmp/fn176.7-review-history-red.log` (exit 1, 5.625 seconds).

The helper already produced correct partial-v2 evidence. No Rust semantics or
image limit changed. Go now retains the identical 11,677,888-byte storage log
and 28,589,558-byte operation JSON, with the original image-capacity terminal.
`.flow/tmp/fn176.7-review-history-green.log` retains the focused pass. The old
image-derived state allowance was 20,054,272 bytes.

The separate stream-bound reproduction is `b3707e1020`. Its first fixture used
capitalized frame tags and failed before admission; that observation is not
proof of this defect. Corrected lowercase tags produce the intended red in
`.flow/tmp/fn176.7-review-stream-red-corrected.log`: complete-v1 still refuses
one byte beyond its cap; explicit partial-v2 also refused that permitted byte.

## Finite bound derivation

Let `C` be explicit `Config.Limits.Calls`, `F` explicit `Config.Limits.Files`, and
`P=4096` (`volume.MaximumPathBytes`). NewEnvironment requires positive explicit
limits; zero does not request defaults. Storage PendingOperations and Transitions
must also be nonzero, but neither safely bounds lifetime history: sync can drain
pending operations while retaining history. The correction does not use them as
history ceilings. The storage log remains independently bounded by TranscriptBytes.

The source invariants used by the bound are:

- Rust `Namespace::new_node` increments `files`; remove and rename unlink child
  entries without decrementing that counter or dropping retained nodes. Recovery
  attachment also rejects a loaded tree above the Files limit. Thus F bounds
  every allocated namespace node in the incarnation, including unlinked nodes
  retained through descriptors. `resolve_path` admits at most P bytes and the
  volume's `normalize` also checks the absolute resource at create/rename, so
  each separator plus basename fits P bytes. A historical path has at most F
  components: at most F*P bytes. This includes ancestor-rename growth; the bound does not
  assume every retained descendant's full path stays within P.
- A guest call introduces at most two persistence operation records: create
  introduces allocation plus namespace; a growing write introduces resize plus
  write. `Storage::open` truncates only nonempty data, so create+truncate of a
  new empty file does not introduce a third record. Rename introduces one
  compound namespace record. Guest mkdir requires an existing namespace parent.
- Observed Operation contains hashes/scalars and only its first/last effect
  paths, not raw file payloads or the full namespace-effect list. Its non-path,
  non-array-element JSON is at most 360 bytes with maximum-width numbers,
  `namespace`, 71-byte hashes and null dependencies; reserve 361 including an
  array comma. A directory rename has at most 2F effect paths and two parent
  plus one inode dependency: at most 2F+3 dependency numbers. A uint64 and comma
  require at most 21 bytes. UTF-8/control/path JSON escaping needs at most six
  output bytes per input byte.
- Do not assume persisted snapshots have only the current linked paths. A call
  can introduce at most F destination paths (a rename inserts at most one per
  allocated node; removed paths introduce none). Each snapshot tree starts with
  at most F paths; over C calls it has at most F+C*F <=2*C*F paths because C>=1.
  Both persisted and volatile trees therefore fit 4*C*F entries, including
  initial/seeded paths and names retained by partial persistence. SnapshotEntry
  non-path JSON fits 194 bytes with maximum-width values, `directory` and a
  71-byte hash; reserve 195 including a comma.

If B is the existing complete-v1 allowance (including enumeration budgets), the
partial-v2 allowance is conservatively:

```
B
+ 2*C * (361 + 2*6*F*P + 21*(2*F+3))
+ 4*C*F * (195 + 6*F*P)
+ sum_volumes(18*len(ID) + 6*len(Mount) + 1024)
```

B still covers any recovered operation history admitted under the independent
input-image cap. Fresh volumes start without persistence history; initial
snapshot paths are explicitly covered above even if final image encoding fails.
The per-volume term covers repeated map/wrapper names and fixed metadata, so
volume count, ID escaping and mount names are explicit. Calls/Files apply
globally across volumes. Every addition and product saturates at MaxUint64,
including C*F and C*F*F. Large declared budgets intentionally permit a large
conservative bound; the receiver allocates only bytes actually received, never
the declared maximum. This is a finite permitted-evidence allowance, not a
measured memory-cost claim.

## Admission and regression controls

Only a recognized full 16-byte magic/version/count prefix for partial v2 selects
the larger allowance. A fragmented prefix is assembled in a fixed 16-byte buffer.
It is recognized, not authenticated: the existing final envelope digest, log
binding, typed JSON, stage, terminal and image checks still decide admission.
Unknown magic/version/count keeps the complete allowance and fails strict
decoding. Complete-v1 and recovery-image limits remain unchanged.

Tests cover every header split 0–16, short received prefixes 0–15 (including a
13-byte zero-padding regression), malformed magic/version/count, exact partial
limit and one byte over, unchanged complete and independent image limits,
maximum-width scalar/escaped-path/dependency record sizes, separate product/addition overflow and zero
terms. The small stream-bound fixture explicitly selects valid Calls=1/Files=4
to test exact/+1 allocation economically; its expected complete and partial caps
are independently calculated literals 19,169,536 and 21,141,076 bytes. Existing
45-MB enumeration and escaped-metadata assertions remain unchanged, as do strict
corrupt/truncated/trailing/terminal/stage/image refusal controls.

Final source checks, identity and new freeze are recorded by the task acceptance
artifact and handover once they have run. No second formal review or current
qualification claim is made by this derivation record.
