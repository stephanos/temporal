# Task 40 deferred pipe-close fault reachability

No safe reproducer of a genuine first deferred pipe-close failure was identified within task 40's admitted surface. The inspected Go and kernel sources explain why ordinary pipe operations cannot supply one. This source-derived limitation does not satisfy the task's requirement for executed genuine cleanup faults, including simultaneous failures. The owner must explicitly change that evidence requirement or admit an additional fault mechanism before this gap can be closed.

This is narrow read-only research, not acceptance or an implementation review. No Go command, build, test, lint, generator, qualification, lifecycle operation, staging, commit, CI or publication ran. Only this findings file was written. Native owners fn-128 and fn-149 remain deferred.

## Candidate and evidence boundaries

The inspected HEAD was `29c80199cdf1a7444f3a3aa99388e2f408e3cf73`, checked again before writing. The worktree contains another worker's task 9 changes. The two mechanism files below were unchanged against reviewed `d2e0e035519f1385b9acf630a70152655b113f61` (`git diff` empty) and had the same hashes on both reads.

| Repository input | SHA-256 |
| --- | --- |
| `tools/gomad3/internal/hostexec/command.go` | `af5cddc9e64692bf450ca134e9be51e314faf100ce70c9eab387825573da5b33` |
| `tools/gomad3/internal/hostexec/command_unix.go` | `d45c444c47f75089de9ec3f9369a85b4c8d2c60d21f191d3bef4733aa0726a45` |
| `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.40.md` | `1b71e7d44905b02c469cccaec9e48646d010614e52cf6fce35849ecacdd6f21d` |

The local stock source root is `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64`. Its `VERSION` states `go1.27.1` and `time 2026-08-28T16:20:06Z`. Each Go source in the table below was compared byte-for-byte with the official Go repository's `go1.27.1` tag and matched. The cached `tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz` hashes to `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`, the value in `toolchain/version/version.json`.

The observed host identifies itself as Linux aarch64, kernel release `7.0.14`. Reading sources and hashing files are the only empirical observations here. No close-fault experiment ran. Linux v7.0.14 source is release-matched evidence, not verification of this host's running kernel build. Linux v6.12 provides a second pinned source comparison. The selected Apple XNU source is not bound to a current native Darwin candidate. No result here qualifies linux/amd64 or darwin/arm64.

## What must fail

`command_unix.go:43-86` creates two actual `os.Pipe` pairs and captures their concrete `*os.File` values in four defers. Their execution order is stderr writer, stderr reader, stdout writer, stdout reader. Callers receive no pipe handles and cannot supply replacement closers through `Request`.

After successful Start, line 130 explicitly closes both writers. A later deferred writer close therefore normally produces `os.ErrClosed`, which those two defers exclude. The reader defers perform the first close of their respective owned handles. Before Start, or when Start fails, the registered writer defers also perform first closes. Failure of the second `os.Pipe` would execute the first pair's defers but does not itself make either existing pipe's close fail.

The source preserves a primary error unchanged when cleanup succeeds; uses a sole cleanup error directly; and joins successive nonnil errors primary-first in defer order without replacing `Result`. That describes the code. It supplies no execution evidence for either one genuine deferred close error or multiple such errors. The [historical checked-cleanup progress](../lint-cleanup-2026-10-05/progress.md) explicitly retains the same unexecuted paths. `gocommand.New` can exercise result/error projection but cannot drive these internal close calls.

## Go 1.27.1 return path

For a nonnil File, `File.Close` calls the private file's close operation. The latter calls `poll.FD.Close`, converts its already-closing error into `os.ErrClosed`, and wraps a nonnil error in `*os.PathError` with operation `close`. Pipe creation constructs ordinary `kindPipe` Files; it supplies no close-failure control. See [file_posix.go:20-25](https://github.com/golang/go/blob/go1.27.1/src/os/file_posix.go#L20), [file_unix.go:307-325](https://github.com/golang/go/blob/go1.27.1/src/os/file_unix.go#L307), [pipe2_unix.go:13-22](https://github.com/golang/go/blob/go1.27.1/src/os/pipe2_unix.go#L13), and [pipe_unix.go:13-29](https://github.com/golang/go/blob/go1.27.1/src/os/pipe_unix.go#L13).

The poll close marks the descriptor closed, unblocks pending I/O, and drops its reference. If that drop destroys the descriptor, its syscall error is returned. If another operation still holds a reference, destruction happens on that operation's final release; waiting for destruction does not turn an I/O error into a close error. Thus concurrent I/O offers no extra close-error source and may prevent an eventual destroy error from being returned by Close itself. See [fd_unix.go:77-117](https://github.com/golang/go/blob/go1.27.1/src/internal/poll/fd_unix.go#L77) and [fd_mutex.go:222-227](https://github.com/golang/go/blob/go1.27.1/src/internal/poll/fd_mutex.go#L222).

Poller close and eviction return no error. Deadlines affect poll readiness and I/O. Destruction calls `CloseFunc`, whose normal value is `syscall.Close`; it does not retry EINTR because descriptor ownership after EINTR is platform-dependent. `internal/poll` is a Go implementation package, not an admitted hostexec injection API. Changing its hook would change the tested implementation. See [fd_poll_runtime.go:48-62 and 132-168](https://github.com/golang/go/blob/go1.27.1/src/internal/poll/fd_poll_runtime.go#L48), [fd_unixjs.go:18-25](https://github.com/golang/go/blob/go1.27.1/src/internal/poll/fd_unixjs.go#L18), and [hook_unix.go:11-12](https://github.com/golang/go/blob/go1.27.1/src/internal/poll/hook_unix.go#L11).

## Kernel evidence and its limits

Linux v7.0.14's close syscall returns EBADF for a missing descriptor, otherwise returns the result of `filp_flush` after releasing the file. `filp_flush` starts at zero and obtains an error only from a file's optional flush callback. Its restart-to-EINTR conversion cannot manufacture an error when flush returned zero. See [fs/open.c:1458-1522](https://github.com/gregkh/linux/blob/v7.0.14/fs/open.c#L1458).

Anonymous pipe creation selects `pipeanon_fops`. That table has no flush callback, and its `pipe_release` returns zero after reference bookkeeping and wakeups. Peer closure, full buffers and unread bytes do not introduce a failing return there. This rules out a natural pipe-close error in this inspected path when the descriptor remains valid and syscall execution is not intercepted. See [fs/pipe.c:926-965](https://github.com/gregkh/linux/blob/v7.0.14/fs/pipe.c#L926), [pipe_release:725-745](https://github.com/gregkh/linux/blob/v7.0.14/fs/pipe.c#L725), and [pipeanon_fops:1257-1266](https://github.com/gregkh/linux/blob/v7.0.14/fs/pipe.c#L1257).

The final release machinery invokes a file's release callback without using its return value as the close syscall result. See [fs/file_table.c:480-519](https://github.com/gregkh/linux/blob/v7.0.14/fs/file_table.c#L480). Linux v6.12 reaches the same conclusion through its shared `pipefifo_fops`, which likewise lacks flush, and a zero-returning release. See [v6.12 open.c:1516-1576](https://github.com/torvalds/linux/blob/v6.12/fs/open.c#L1516) and [v6.12 pipe.c:723-744 and 1232-1241](https://github.com/torvalds/linux/blob/v6.12/fs/pipe.c#L723).

Apple XNU `xnu-11215.1.10` also makes its pipe-specific close return zero after teardown. See [sys_pipe.c:1306-1319](https://github.com/apple-oss-distributions/xnu/blob/xnu-11215.1.10/bsd/kern/sys_pipe.c#L1306). The generic close path can reject a missing or guarded descriptor. For an ordinary valid, unguarded pipe it reaches `fp_close_and_unlock`, then `fg_drop`, whose final fileops close is the pipe operation. Vnode-specific errors in `fg_drop` do not apply to a pipe. See [kern_descrip.c:248-298](https://github.com/apple-oss-distributions/xnu/blob/xnu-11215.1.10/bsd/kern/kern_descrip.c#L248), [1696-1797](https://github.com/apple-oss-distributions/xnu/blob/xnu-11215.1.10/bsd/kern/kern_descrip.c#L1696), and [5371-5389](https://github.com/apple-oss-distributions/xnu/blob/xnu-11215.1.10/bsd/kern/kern_descrip.c#L5371).

Darwin's general [close(2) manual](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/close.2.html) allows EINTR and EIO. That generic list does not establish a trigger for this pipe path. Guard manipulation, foreign thread cancellation, syscall interception, kernel defects, and other kernel versions are outside this bounded source conclusion. No universal theorem that every Unix pipe Close always succeeds is claimed.

## Candidate fault mechanisms

| Candidate | Why it cannot discharge the current requirement |
| --- | --- |
| Close a peer, fill a pipe, leave bytes unread, cancel the command or expire a deadline | These change I/O, process or readiness outcomes. The inspected close return path still has no pipe flush error. |
| Exhaust descriptors so a later Pipe or Start fails | This exercises setup failure and successful cleanup of already-created valid pipes. It does not create the required cleanup failure. |
| Close the same File twice, concurrently, or through a copied File value | The second close reaches the already-closed state. It is excluded evidence even where a reader would return that error. |
| `os.NewFile(f.Fd(), ...)` followed by Close, direct `syscall.Close`, `dup2`, or close-range | Closing an independently wrapped alias invalidates a descriptor behind its owner. An ensuing EBADF is not a genuine first OS close. These are descriptor interference in substance and violate the explicit raw-descriptor prohibition. |
| FUSE, NFS, a failing device, socket shutdown or SO_LINGER | The actual sites own anonymous pipes. Replacing their resource kind or creation would change the mechanism being tested. |
| `gocommand.New` returning an infrastructure error | Useful projection coverage; it never executes hostexec's deferred OS close fault path. |
| `internal/poll.CloseFunc`, an injected closer or source overlay | A substituted error can test propagation, but requires new admission and remains synthetic fault evidence. |
| External syscall-error injection or descriptor guards | These require a separately designed and authorized harness/platform operation. They are not an existing File API trigger, and synthetic syscall returns cannot be presented as naturally occurring pipe-release faults. No such experiment was attempted. |

The source-derived inference is that no admitted ordinary Go File operation supplies the required first-close error. Simultaneous genuine cleanup failures inherit that reachability problem. More repetitions of success, cancellation or duplicate-close tests cannot resolve it.

## Smallest explicit owner decision

The smallest change is an evidence-policy amendment confined to task 40's genuine deferred-close-fault requirement. The owner could accept this bounded source-derived reachability argument together with ordinary controls and independent review, while expressly retaining that single and simultaneous genuine faults were never executed. That choice changes acceptance, not production semantics. It must be recorded as an owner decision; this report neither makes nor applies it.

If the owner requires dynamic error-propagation coverage, a separate narrow amendment could admit a private close-substitution seam solely at these four defer sites and accept synthetic errors for that requirement. A concrete prospective matrix would cover nil primary/nil cleanup preserving identity, sole cleanup returned directly, primary plus cleanup joined in order, two reader errors joined in LIFO order, all four first-close errors after failed Start, writer-only ErrClosed exclusion, and preservation of every Result field and CommandError object. Ordinary real-process controls would still cover resource cleanup. That alternative changes the admitted implementation surface and the evidence definition, needs fresh preservation review, and still proves no natural OS pipe fault.

If the owner retains the existing genuine-fault requirement and all prohibitions, leave this proof obligation open. A supported native host alone does not supply the missing trigger; no native owner should be revived merely to repeat these ordinary close cases. Any proposed native or external fault harness needs its own explicit scope and must state whether it executes a real kernel failure or substitutes a return value.

## Retained primary-source identities

Line numbers above refer to raw source files, including blank lines. Browser-rendered line numbers may differ. Remote bytes were fetched into memory and hashed without writing source copies.

All Go paths below are relative to the local source root stated above and to `https://raw.githubusercontent.com/golang/go/go1.27.1/`. Every comparison returned `upstream_equal=true`.

| Go input | SHA-256 |
| --- | --- |
| `VERSION` | `25eb74b09036e1ad894fb63b86cb0f0ee53342df8d2e2e860da4ff5079607758` |
| `src/os/file_posix.go` | `2e93ac5f812cd7184d3ac3a2eb9a51583911a6261cf9c27772fa7dfedcbe514f` |
| `src/os/file_unix.go` | `877dd004355c475e8805b5700061604558422da057037556e0cf6edeb36ab7ec` |
| `src/os/pipe2_unix.go` | `40080f961b8a6848640f344494f843b0612b9074943f719c6485ec305ab03ef1` |
| `src/os/pipe_unix.go` | `afbd7c5f5d714feb8b72739913311719e3c352ad2c21b70b9c043106da8a9bb1` |
| `src/internal/poll/fd_unix.go` | `675f74e8cbfc73170e08b43947c131706cd59e498d722555fe3b793618f5f162` |
| `src/internal/poll/fd_mutex.go` | `ee83b1177202cc7b930ccff3ac3f5661e7f2e69b2d6b4566afbd41ea4696b8dc` |
| `src/internal/poll/fd_unixjs.go` | `ed73269460d0c3dcd551a2b9caa5c58e5c818a064d2971f8d63b3b13c6fdebe0` |
| `src/internal/poll/fd_poll_runtime.go` | `88fda43019c5dcf42545effdebd1747aaea377560dd041a35df8499538ee07c5` |
| `src/internal/poll/hook_unix.go` | `4a467c97720f9d92849b4d64fd95980f8528d9ae2ac2a50cfc1309ff0759cba0` |
| `src/syscall/zsyscall_linux_amd64.go` | `9d3d6ce5403a67d06471690e1c0f7ebf36bade8d843c7e3bef31ca2b7acaf80c` |
| `src/syscall/zsyscall_darwin_arm64.go` | `941547c92d1204cf9666c40299b1685f393cb5546369b9238ec534dcd7a81201` |

Kernel paths resolve under the linked official repository and tag above.

| Repository/tag/path | SHA-256 |
| --- | --- |
| `gregkh/linux v7.0.14 fs/open.c` | `f5e0e6f8afab4a9defd31f06702ae2ce7d5e78789898d3c16be3f22555b826c2` |
| `gregkh/linux v7.0.14 fs/pipe.c` | `56251a2dd4baac01424132e2b96413b14d85d9ff13284146658be5d3c347b68c` |
| `gregkh/linux v7.0.14 fs/file_table.c` | `49ce70ad10db0e036bc439ee9c4abcee6c3ea4deacb668a35c46877f144e3190` |
| `torvalds/linux v6.12 fs/open.c` | `587769343f2fc32552fb8593138c72f47110de8b7933f921a0bf9b627282b59a` |
| `torvalds/linux v6.12 fs/pipe.c` | `a9c48d85993a1a5bc1c293f0dd8ea94e1f7cabe029d17e04a06faa60ade4ccf7` |
| `torvalds/linux v6.12 fs/file_table.c` | `3e71b50d0c53d169162ac85f3166d813b03a8ab04ebb4c6dd05e88d9778d1629` |
| `apple-oss-distributions/xnu xnu-11215.1.10 bsd/kern/kern_descrip.c` | `7385f86a8c0b98946f00d0993b4ece5a3707faf4443bfb0232bcc5e5ae5bb494` |
| `apple-oss-distributions/xnu xnu-11215.1.10 bsd/kern/sys_pipe.c` | `51dfde3a22b6b64316cbed6c4099f30b99e1e056522798e45cab6cf982c749d3` |
