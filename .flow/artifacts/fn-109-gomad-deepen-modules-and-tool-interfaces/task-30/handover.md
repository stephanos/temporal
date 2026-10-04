# Task 30 public artifact copy source progress

Public artifact copies now observe payload cleanup errors while preserving the existing operation error when cleanup succeeds. The six CopyPayload and five admitted opened-test cleanup findings are resolved. Actual unfiltered artifact lint falls from 21 to 10 findings with zero introduced diagnostics. This handover supplies source progress for root's independent review and commit; the task remains in_progress.

BASE is `08096389f252e35ff2cd898ca5e381f9b878f46c` on authorized branch `gomad`. The worker wrote only `tools/gomad3/artifact/open.go`, `opened_test.go` and task-30 artifacts. Root owns Git/index/commit, Flow lifecycle/parent, MILESTONES and independent source review. Worker commits and lifecycle writes are zero. Requested implementation route was `gpt-6.1-sol` at high. Root retained `Tier: session (jev-unavailable(no_key))`; execution metadata does not identify the actual model. Routing and bridge calls were not repeated.

stage: impl-review - skipped(policy: conductor-deferred source progress - root owns review; no formal SHIP)

## Cleanup and preservation

CopyPayload names its existing error result without changing its public Go function type. The source defer is installed after successful OpenPayload acquisition; the destination defer follows exclusive destination acquisition. LIFO release attempts destination Close before source Close, once each. The five explicit destination branch closes are removed, and success returns nil so the destination defer owns the existing checked success Close. Neither defer closes the Opened root.

Each defer leaves the result untouched when Close returns nil. With a nil result and a nonnil Close error, it adopts the exact cleanup error. This preserves the direct destination-Close error on an otherwise successful operation instead of adding a gratuitous join wrapper. With an existing error, it joins primary first and cleanup second. Simultaneous operation, destination and source failure therefore traverses primary, destination, source, with a nested join when both cleanups fail. Formerly ignored genuine source-close and branch destination-close errors become visible and dual failures add cleanup text. Universal error equivalence on those failure combinations is not claimed.

The five original opened-test defers observe Close at the same test/subtest lifetime, and the new test owns its handle similarly. All original test assertions retain their strength. The evidence script reconstructs the exact original opened_test.go bytes by removing admitted additions and reversing only the five cleanup replacements. It also compares every open.go byte outside CopyPayload with BASE and recovers the unchanged operation body after normalizing only the result name and cleanup.

| Baseline location | Checked cleanup location |
| --- | --- |
| open.go:270 | open.go:271 source defer |
| open.go:276,283,288,293,297 | open.go:284 destination defer |
| opened_test.go:41,61,145,256,323 | opened_test.go:43,67,218,333,414 |

Source validation remains before destination creation. Existing nil/closed, listing, bound/mode/size/SHA/EOF/rewind and target-only hard-link checks remain in their owners. O_WRONLY/O_CREATE/O_EXCL, requested mode and Chmod, LimitedReader/MultiWriter/hash/count, extra-read synthetic-error precedence, Sync and partial destination policy remain unchanged. Pinned root, private manifest, cloning, callers, directory sync, shared verification, publication/pool, reflection switches and invariant panics retain their existing bytes and ownership. No helper framework, public API, Close seam, descriptor theft or race was added.

## Executed evidence

[evidence.json](evidence.json) retains every parsed before/after/resolved/residual/introduced diagnostic and points to the raw logs and receipts. The 10 residual path/message/linter identities match baseline. They comprise six errcheck, two exhaustive, one forbidigo and one staticcheck finding. The two unchanged reflection switches shift from lines 181/216 to 257/292. No config, suppression, rule, dependency, generator input, golden, pin or runtime source changed. Historical whole-scope 419 remains historical.

Baseline whole ordinary artifact package and errortype pass. Actual baseline lint exits 1 with the eleven admitted source-policy RED findings. Real-file controls were added and pass before production edits, with 14 test/subtest RUN entries. They characterize current behavior rather than reproducing a runtime first-Close fault.

The new destination test directly asserts raw *os.PathError with Op open, destination Path and os.ErrExist/NotExist classification. Sentinel bytes/mode remain unchanged, missing parent stays absent, and the same Opened reads stdout and later copies literal stdout at mode 0600 while preserving its manifest. The existing source-damage matrix now exercises CopyPayload for unlisted, mode, size, hash, symbolic-link and escaping-directory cases, checking source errors before collision and untouched/absent destinations. Its caller-selected maximum case remains ReadPayload/OpenPayload only. The pinned-directory replacement test copies original stdout at 0600; the replacement stdout differs, so this exercises the pinned source instead of an indistinguishable target.

Two bounded mutation checks demonstrate that these tests catch meaningful defects. A source defer that joins a primary error despite nil Close causes both raw destination-error assertions to fail on *errors.joinError. Retaining the old checked destination Close alongside its defer causes the existing target copy, subsequent stdout copies and pinned stdout copy to fail with file already closed. Both commands exit 1 for those intended reasons. Restoring the exact frozen source makes the same controls pass. These mutation receipts prove test sensitivity to wrapping and double-close mistakes; they supply no first-Close OS-fault evidence.

Final whole artifact package passes with 72 test/subtest RUN entries. Focused opened/copy/clone/pinned/source-validation/Store/publication/pool/mode controls and supplemental retained-byte/private-copy/shared-target controls pass. Root architecture/public alias/public-signature fixtures and both external Runner consumer compilation tests pass. The historical TestRecordAndArtifactHaveSeparateOwners declaration is absent; current TestPackageArchitecture enforces distinct record/artifact owners. Final errortype and scoped diff/gofmt checks pass. Pinned unfiltered lint exits 1 with the 10 residual findings above.

Makefile VERSION_INPUTS, BOUNDARY_INPUTS, COMPATIBILITY_INPUTS and validate recipes were inspected. These two edited files are outside generator inputs. `make validate` passes generated version/protocol/boundary, compiler fixtures, patch/script ownership, compatibility packs and qualification-manifest checks without regeneration. Each receipt binds exact command, environment, cwd, start/end/elapsed, child exit, timeout status, raw log SHA256, pinned tool/config hashes and stable complete 23-file artifact-source hashes. No test selection is empty. The 1,042 protected tracked files retain aggregate `03624cfa72199f66795820c3fe5e6b92734457e3fc88db05b97e1098446ca6a4`, matching root's source-admission.json; only the two admitted source files are excluded. Admission-protected original documents retain their hashes.

## Qualification limits

Cached stock Go1.27.1 linux/arm64 runs use `GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS=''`, cleared GOMADSEED/GOMAD3_CHILD_SEED and test `-count=1 -tags test_dep`. The cached File.Close and errors.Join sources explain repeated-close errors and single-error join wrapping. No genuine first-Close, simultaneous operation/destination/source-close, post-validation growth/hash, partial-write, Chmod or Sync fault execution is supplied. Their conditional cleanup identity and ordering are source-inspected. Portable legitimate seams remain absent; no failure was manufactured with injection or descriptor interference.

Original R13/R18/R19, task 12/predecessors/task 21, matched first-baseline fixed identities, full/formal qualification, both patched-native platforms and affected consumer/integration/qualification acceptance remain open wherever unproved. Unchanged broad rootfast/full/native environment failures were not rerun. Independent source review and commit remain pending root; this handover claims no formal SHIP.

Final open.go SHA256 is `69cd4af287008c5985050b6d9a7a03f3b5c96ad695bc1b005eb9c3fe2431b0de`. Final opened_test.go SHA256 is `199a4f734197686f71b00333ec5699b444444ea06cf9407df821e89d4e9f1363`. All owned commands are terminal. Spawned delegates, live handles and pending commands are zero.
