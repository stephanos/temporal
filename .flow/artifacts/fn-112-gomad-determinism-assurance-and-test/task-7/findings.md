# Actual model conformance findings

Host: darwin/arm64. Patched toolchain identity: 6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee. Source hashes: finding-source-hashes.json. Linux not executed.

The strict comparison run in initial-standard-library.log failed for seeds 0, 1, 7, 42 and 89. Ascending fresh-process reruns checked every shorter prefix; prefix lengths are operation counts, not arbitrary line truncations. The generator always constructs all 64 operations before executing the requested prefix.

1. Filesystem, shortest prefix 15: a second os.File.Close returns syscall.EBADF; stock Go returns an error matching os.ErrClosed. Model generator argv `0 15` reproduces. gomadfs.Handle.errorLocked fs.go:1373 returns EBADF when handle.closed; os.gomadFileClose os/gomad.go:916 forwards it to File.wrapErr. Wrong-access-mode EBADF must remain distinguishable from a closed handle.
2. TCP, shortest prefix 60: reading a closed TCP connection yields a net.OpError with text "use of closed network connection", but errors.Is(err, net.ErrClosed) is false. Stock Go returns true. Model generator argv `0 60` reproduces. gomadio.ErrClosed network.go:21 is a distinct errors.New instance; net/gomad.go:87 wraps it directly. Stock net.ErrClosed aliases internal/poll.ErrNetClosing.

Neither finding was added to the declared-difference normalization table. Their public Go error identities are semantic differences. Runtime edits require conductor approval and a new overlay build identity.

3. Full-log inspection also exposed self-rename data loss. Seed 0 operation 30, Rename(workspace/f0, workspace/f0), reports success but deletes f0; directory listing operation 31 and stat operation 45 expose the missing file. Seed 1 repeats the pattern at operation 49, followed by a failed rename and incorrect payload. The source assignment to nodes[newPath] followed by delete(nodes, oldPath) deletes the shared path. Baseline evidence: defect-full-logs.log; focused backend and volume durability regressions go red on the preserved baseline (red-backend.log and red-directory-volume.log).

## Correction disposition

The conductor approved minimal overlay corrections after evidence was captured. No defect became a declared difference. Closed backend handles use a distinct marker wrapping EBADF; only the os adapter translates it to os.ErrClosed after recording the raw result. Process-handle closed guards share the marker, while access-mode EBADF remains unchanged. The network model aliases internal/poll.ErrNetClosing. Valid self-renames return before modifying namespace/data/metadata/durability state, after source, parent, readonly and mount checks.

The first rebuilt corrected identity is d15ad896e542b9f64381c2f184567355bac8ffd62ae2347501fe8fcdd96f6eba. The baseline build remains present. Source bindings are frozen-source-hashes.json; the task-only patch is measured from actual before copies, including the prior dirty version descriptor. Two test-only descriptor allowlist entries were authorized and generated consumers were checked (unchanged). No runtime patch hunks or capability boundary were changed.

Focused corrected model comparisons pass three repetitions of two fixtures, five seeds and 64 operations, plus representative shorter-prefix reruns. Seeded public os and backend/process tests pass. Complete gate results belong to commands.jsonl and the handover evidence; Linux remains unverified.

## Final libc consumer correction

The first corrected build is intermediate evidence, not final qualification. An actual captured libc descriptor holds an `*os.File` while another operation closes the descriptor. Reading that captured file now returns public `os.ErrClosed`; the former `libcErrno` extractor had no matching errno and returned `EIO`. The preserved baseline returns `EBADF` for the same deterministic interleaving. Evidence: libc-captured-errno-probe.log (d15 red), libc-captured-errno-baseline.log (6b green), libc-converter-final-red.log and libc-converter-prebuild-green.log.

After the conductor authorized the narrow consumer correction, libcErrno maps os.ErrClosed to EBADF and otherwise preserves existing extraction. The existing anonymous_test.go retains a regression using the real descriptor capture, close, public read and errno converter; removed descriptors also remain EBADF. No semantic mismatch was normalized.

The final corrected identity is d49ef0309636e2301c46cfb36cda8a9b21323d87601c93c429260b3eefd4e37d. d15-frozen-source-hashes.json and d15-task-only.patch preserve the first build snapshot. Final source bindings are frozen-source-hashes.json and final-source-hashes.json. Final2 focused public os/backend/libc/model gates pass, including all 30 complete model/native comparisons (1,920 operations per side). Final2 host passes; final runtime and overlay outcomes are recorded in the handover when they finish. Parent validation independently confirms the final comparisons, protected earlier sources and task-only patch reconstruction.

## Closed-directory native contract correction

The final2 d49 runtime tier passed, but the unseeded overlay gate failed the new directory-read expectations. The expectation that all closed directory operations match os.ErrClosed was wrong. Stock Go1.27.1 actual probes show Chdir returns PathError(Op=chdir) wrapping os.ErrClosed, while ReadDir, Readdir and Readdirnames return PathError with an empty operation on Darwin, the original file name, and poll.ErrFileClosing. Neither os.ErrClosed nor EBADF matches those directory-read errors. Pinned Linux source uses operation readdirent; native Linux remains unexecuted.

Seeded d49 returned bare os.ErrClosed for these directory reads, so correcting only the expectation would leave an actual model mismatch. Evidence: parent-runtime-overlay-final2.log retains runtime success plus overlay failure; directory-native-probe.log and directory-seeded-probe.log retain exact wrapper/class/operation differences. The initial native test-overlay attempt was rejected because files beneath GOMODCACHE cannot be overlaid; this is an invocation limitation, not a native semantic failure. The successful stock test-overlay check sets GOMODCACHE to an empty scratch path without mutating the stock toolchain or cache.

The approved correction changes only the gomadFileReaddir closed-marker branch after raw transcript recording: wrap poll.ErrFileClosing in the native platform's PathError, preserving the original file name. Nonclosed errors and Chdir remain unchanged. The regression verifies exact public/internal sentinel identities and wrapper/path/operation. Corrected scratch tests pass stock, unseeded and seeded execution (directory-proposed-stock.log, directory-proposed-unseeded.log and directory-proposed-seeded.log); the formatted final-source seeded scratch run also passes. One failed final-source invocation due a missing generated overlay JSON remains labeled separately. None of these errors became declared differences.

D49 source and patch snapshots are immutable d49-frozen-source-hashes.json and d49-task-only.patch. Final3 corrected identity and expanded qualification outcomes are bound by the final handover evidence after the third rebuild and refreshed gates finish.
