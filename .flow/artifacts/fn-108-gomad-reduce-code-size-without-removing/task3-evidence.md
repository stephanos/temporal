# fn-108.3 evidence: modernc memory adapter through the rewritten-module owner

Platform darwin/arm64, patched go1.27.1 toolchain (`tools/gomad3/.toolchain/bin/go`, key `8d28bd44...`), base
revision `6782b55f49a0317b230e827ea2a63a37d116d502` plus the uncommitted fn-108.2 and fn-108.4 edits. Nothing
staged or committed. linux/amd64: not run (no host).

## Change

- `tools/gomad3/deterministicio/memory_adapter.go`: `prepareModerncMemory` keeps its name and signature and
  is now one call to `prepareRewrittenModule`. The three anchor/replacement pairs moved into
  `memoryRewrites` (`[]sourceRewrite`, one entry for `mmap_unix.go`). `rewriteModerncMemory` and the private
  copy of the preparation and anchored-rewrite loop are gone. The file follows `sprig_adapter.go`.
- `tools/gomad3/deterministicio/memory_adapter_test.go`: the two tests that called `rewriteModerncMemory`
  call `rewriteAdapterSource(memoryModulePath, memoryRewrites[0], ...)` with their assertions unchanged.
  Added tests are listed under "Tests".

No other file changed. [task3.diff](task3.diff) is the working-tree diff of the two files.

## Byte equality (R4)

The probe [task3-fingerprint-probe_test.go.txt](task3-fingerprint-probe_test.go.txt) was compiled into the
package with `go test -overlay` by [task3-fingerprint.sh](task3-fingerprint.sh), so it never existed in the
checkout. It calls only `prepareModerncMemory` and helpers present before and after, and ran once on the
unmodified code and once after the rewrite.

| Item | Before | After |
| --- | --- | --- |
| Fingerprint file (sha256) | `ab05ceb1...d71cdd` | `ab05ceb1...d71cdd` (`cmp` equal) |
| Replacement `mmap_unix.go` bytes | `sha256:c8a86dca...54ab88ad` | same (`cmp` equal, equals `memoryMmapReplacementSHA256`) |
| Replacement tree, 26 entries (path, mode, size, sha256) | `sha256:76434754...808876d97` | same |
| Original inventory | `sha256:f6d838c7...59f0264` | same |
| Replacement inventory | `sha256:b947d0e7...257a535` | same |
| Profile identity (implementation / inventory) | `sha256:9cd0cff9...6ac7c0` / `sha256:0d34edfc...7358f7` | same |
| `BuildAdapter` evidence from `prepareModerncMemory` (paths scrubbed) | recorded | identical |
| Profile path: `PrepareBuildAdapters` evidence, build modfile, `target.ReviewCapabilities` | accepted | identical, accepted |

Files: [before](task3-fingerprint-before.txt), [after](task3-fingerprint-after.txt), and the replacement
source [before](task3-replacement-mmap_unix.go.before.txt) / [after](task3-replacement-mmap_unix.go.after.txt).

Textual pins: the `const` block and `memoryPreparedSourceSetSHA256` (both platform entries) are identical to
`HEAD`; the six anchor/replacement literal lines are identical to `HEAD` (script check in the task log,
`git diff` shows no changed line carrying a digest, version or sum). The replacement directory stays
`modernc-memory`, so the published cache path `adapters/modernc-memory@v1.11.0-b947d0e7fbcc3f18` and with it
the target identity are unchanged.

The rewrite has no build constraints and no platform branch; the only per-platform value is
`memoryPreparedSourceSetSHA256` through `hostPin`. linux/amd64 was compile-checked only
(`GOOS=linux GOARCH=amd64 go vet`, exit 0). The linux/amd64 prepared source-set pin is **not run (no host)**.

## Behaviour differences

All come from the shared owner. No assertion, doc, script or plan depends on the old diagnostics (searched
`tools/gomad3`, `tools/gomad3sim`, `tools/gomad3integration`, `tests/gomadfunctional`, `.plans`, `docs`,
`.github`, `Makefile` for `modernc memory` and `pinned modernc`). Outside `memory_adapter.go` the remaining
hits are the libc adapter's own `modernc libc` strings and `t.Fatalf` messages in `memory_adapter_test.go`
(one pre-existing, four in the added tests) that word the test's own failure report and match nothing.

Diagnostic strings, old -> new:

| Old | New |
| --- | --- |
| `modernc memory adapter identity mismatch` | `modernc.org/memory adapter identity mismatch` |
| `resolve pinned modernc memory module: %w` | `resolve pinned modernc.org/memory module: %w` |
| `hash pinned modernc memory source inventory: %w` | `hash pinned modernc.org/memory source inventory: %w` |
| `pinned modernc memory source inventory identity mismatch: got %s, want %s` | `pinned modernc.org/memory source inventory identity mismatch: got %s, want %s` |
| `read pinned modernc memory source: %w` | `read pinned modernc.org/memory source mmap_unix.go: %w` |
| `pinned modernc memory source identity mismatch` | `pinned modernc.org/memory source identity mismatch for mmap_unix.go` |
| `pinned modernc memory rewrite anchor mismatch` | `pinned modernc.org/memory rewrite anchor mismatch for mmap_unix.go: %q` (the anchor) |
| `modernc memory replacement identity mismatch: got %s, want %s` | `modernc.org/memory replacement identity mismatch for mmap_unix.go: got %s, want %s` |
| `copy modernc memory adapter module: %w` | `copy modernc.org/memory adapter module: %w` |
| `hash modernc memory replacement inventory: %w` | `hash modernc.org/memory replacement inventory: %w` |
| `modernc memory replacement inventory identity mismatch: got %s, want %s` | `modernc.org/memory replacement inventory identity mismatch: got %s, want %s` |

New diagnostics the owner adds: `pinned modernc.org/memory source is not a regular file: mmap_unix.go`
(an `Lstat` check before the read; a failed `Lstat` reports this text without the OS error, where the old
code returned the wrapped `ReadFile` error), `modernc.org/memory adapter has no rewrites` and
`adapter source rewrite has no anchors` (both unreachable for this adapter).

Unchanged: check order (identity, module resolution, original inventory, source read, source digest, anchors,
replacement digest, copy, replacement inventory), `%w` wrapping of the inventory and copy errors (so
`AdapterCapacityError` still matches through `errors.As`), and every failure remains a preparation failure.
`OriginalSourceInventorySHA256` in the evidence is now the pin instead of the computed value; the preceding
equality check makes them the same string.

Recorded failure text for 14 inputs, [before](task3-failure-diagnostics-before.txt) and
[after](task3-failure-diagnostics-after.txt): the same input fails at the same check in both, and the files
differ only in `modernc memory` -> `modernc.org/memory`. Any edit inside the pinned module (source drift,
edited or repeated anchor text, missing file, directory in place of the file, symbolic link) is rejected by
the original-inventory check before the source is read, before and after.

## Tests

`memory_adapter_test.go`, style of the sibling adapter tests (`t.Fatalf`, substring on the diagnostic):

| Test | Covers |
| --- | --- |
| `TestRewriteModerncMemoryModelsOnlyAnonymousAllocatorMappings` (retargeted) | assertions unchanged |
| `TestRewriteModerncMemoryRejectsSourceIdentityDrift` (retargeted) | source drift at the rewrite; assertion unchanged |
| `TestPrepareModerncMemoryRejectsChangedIdentity` (extended) | original combined case plus changed version alone and changed sum alone |
| `TestPrepareModerncMemoryRecordsExactPrivateReplacement` (extended) | adds source, replacement and both inventory digests of the evidence |
| `TestModerncMemoryRewriteRejectsAnchorDrift` (new) | missing anchor, duplicate anchor, replacement digest |
| `TestPrepareModerncMemoryRejectsModuleDrift` (new) | changed source and non-regular source through `prepareModerncMemory`; the owner's regular-file check through `readAdapterSource` |
| `TestModerncMemoryRejectsChangedReplacementInventory` (new) | owner called with the memory description and a different replacement-inventory pin |

Red check: an overlay copy of `adapter_rewrite.go` with the seven checks disabled
([task3-mutated-adapter_rewrite.go.txt](task3-mutated-adapter_rewrite.go.txt), never written into the
checkout) fails every rejection test for the intended reason: [task3-mutation-red.txt](task3-mutation-red.txt).
The changed-replacement-inventory case cannot be reached through `prepareModerncMemory` without editing a
pin, so its test passes the owner a description whose only difference is that pin.

## Size

| File | physical | code | code bytes |
| --- | --- | --- | --- |
| `memory_adapter.go` (production) | 110 -> 52 (-58) | 104 -> 47 (-57) | 4963 -> 2741 (-2222) |
| `memory_adapter_test.go` (test) | 119 -> 218 (+99) | 110 -> 206 (+96) | 4582 -> 8470 (+3888) |

`git diff HEAD --numstat`: `29 87 memory_adapter.go`, `105 6 memory_adapter_test.go`. Whole tree with
fn-108.2 and fn-108.4 present: [task3-size-compare.txt](task3-size-compare.txt), residual code -203, code
bytes -6853, `size-compare.sh` exit 0. `api-capture.sh` output against `api-baseline/` by `diff -r`: empty.

## Commands and results (darwin/arm64)

Timestamps and exit statuses: [task3-gates.txt](task3-gates.txt). Captured output of every row, which carries
the package and request counts: [task3-gate-logs.txt](task3-gate-logs.txt). The `gofmt` and `vet` rows were
run before the gates and again afterwards for the timestamped record; the baseline row has no end timestamp.

| Command | Result |
| --- | --- |
| baseline before any edit: `go test -tags test_dep ./deterministicio` | exit 0 |
| `gofmt -l deterministicio` | no output |
| `go vet -tags test_dep ./deterministicio` | exit 0 |
| `GOOS=linux GOARCH=amd64 go vet -tags test_dep ./deterministicio` | exit 0 (compile check, not a linux test run) |
| `go test -count=1 -tags test_dep ./deterministicio/...` | exit 0, 3 packages ok |
| `make -C tools/gomad3 validate validate-compatibility` | exit 0, `TestHostPacksBindCurrentProfile` ok |
| `make gomad3` | exit 0, toolchain key `8d28bd44...` |
| `tools/gomad3/.bin/gomad doctor` | exit 0, `adapter:modernc.org/memory ok modernc.org/memory@v1.11.0` |
| `make -C tools/gomad3 test-host` | exit 0, 45 packages ok |
| `make -C tools/gomad3 compatibility-pack-qualification` | exit 0, 9 requests qualified |
| `make -C tools/gomad3 core-qualification` | exit 0, `expectations-met=true supported=7 unsupported=0 failed=0 infrastructure-errors=0 completed=7/7`; the report names `modernc.org/memory` |

Not run: every linux/amd64 gate (no host), including the linux/amd64 prepared source-set pin; the full
`make -C tools/gomad3 test`, Temporal integration and smoke (owned by fn-108.7).
