# Compatibility Pack Review: xnet-socket-activity-candidate

Review SHA-256: `sha256:35c1f706f510e1fd953fcdc23d2a0987c0d17c7c70b33aaa38db79e1d312578a`

Owner: `temporal-server`

Reviewed at: `2026-08-15T00:00:00Z`

Justification: Records four exact x/net socket blockers within the Activity workload's 60-blocker closure; direct linkname and syscall containment must be proven before approval.

Target: `go-test ./tests`

Target module: `go.temporal.io/server`

Test arguments: `-test.run ^TestActivityAPIBatchCancelClientTestSuite$`

Build tags: `test_dep`

Platform: `darwin/arm64`

Workload: `activity-batch-cancel-boundary`

## Activation

- `golang.org/x/net@v0.57.0` (`h1:K5+3DljvIuDG9/Jv9rvyMywYNFCQ9RSUY6OOTTkT+tE=`), replacement `none`

## Reviewed packages

### `golang.org/x/net/internal/socket`

Module: `golang.org/x/net@v0.57.0` (`h1:K5+3DljvIuDG9/Jv9rvyMywYNFCQ9RSUY6OOTTkT+tE=`), replacement `none`

Source set: `sha256:ef626257e74862bc629107cb3fb2e179cb1e6c6ae7ab08e5ddac3c151a89bc15`

Go sources:

- `cmsghdr.go`: `sha256:93f9c4fdf14267c9c1af899e7d18f491cafd032c7e7c385e90e6c9ddcadf7728`
- `cmsghdr_bsd.go`: `sha256:367edbee584d8ffe0499acc0f1a0dfdda9a820b28517ec3245928758f6a53876`
- `cmsghdr_unix.go`: `sha256:fd56595a67890b30aad62c4eb3f9eb9dd94d92e0f8b6e563aaf7d24a4f83bdf7`
- `complete_dontwait.go`: `sha256:3bf5218fe4199f3678551e6bfccc71923796fe91b7a431322ab798bf2d051813`
- `error_unix.go`: `sha256:5bb94cc01a2a002540eabf025bad33add287e10aee9eb8bd5adc57fd9fda908a`
- `iovec_64bit.go`: `sha256:b0affa4c544d18a0f9f9485e42ddc76ad413668c320b709ca26cf8055f5a765e`
- `mmsghdr_stub.go`: `sha256:c1506a461da28135077c783cc30d659f949adf500d0872a3560beba76cd835ea`
- `msghdr_bsd.go`: `sha256:a48f4cf26ad4bfb1ccb5ea1fb27a284201f591164f7d1459bf98cecdac27a0c7`
- `msghdr_bsdvar.go`: `sha256:4517055c1593f512c4489f0cb08d975377d6deb9c23096c07879c5cf766850ba`
- `norace.go`: `sha256:7aeb97aa0c4b5678b5a9d1ebfa8105def20ea6f51e41b8537c7f389774c61311`
- `rawconn.go`: `sha256:16955777ba12d008fb71416b64c50687f5d224b2d035d0771bfe6c98c082f2f7`
- `rawconn_msg.go`: `sha256:db7de50ce15a843f400ef959a01f3170a9a4444c9fe499c2bd907d1cceca0202`
- `rawconn_nommsg.go`: `sha256:14a6fa39a3b924e41ecefce834363a8f4c82522ea7e4e006a453e284a44294b1`
- `socket.go`: `sha256:d7f3b8907414648710c0241e2c717df473568a0eb467214ccb0f901838937a8b`
- `sys_bsd.go`: `sha256:d74a01f48b2f546a37d39da843c9243c08423ee3bce8540c0a3b6ea31d6203d5`
- `sys_const_unix.go`: `sha256:0f4985aef1899553a334e66d011e3b37b64c1ca7ba6e6f1b4cb5f77fb6b1db4e`
- `sys_posix.go`: `sha256:0d3b6f84759807e6d798999372b4f126dfd8058a4981d27bc16a93c6485f4779`
- `sys_unix.go`: `sha256:facf54b3bc8b1e36552241cdf5bf3f5cd1010cf864f995cb0cf2ed3830036d6c`
- `zsys_darwin_arm64.go`: `sha256:c87480f145e5b079e4ea4dea05152132682847649b874c03e7e9206ef9cf5f4a`

Foreign sources:

- `assembly:empty.s`: `sha256:0d09f2c52fc60c2d411818b538de77927fbf43ad530214066e26315922f5bdd6`

Requested facts:

- `foreign:assembly:empty.s`: **deny** — **security-sensitive**
- `import:golang.org/x/sys/unix`: **deny** — **security-sensitive**
- `import:syscall`: **deny** — **security-sensitive**
- `linkname:sys_unix.go`: **deny** — **security-sensitive**
  - source `sha256:facf54b3bc8b1e36552241cdf5bf3f5cd1010cf864f995cb0cf2ed3830036d6c`
  - directive `syscall_getsockopt syscall.getsockopt`
  - directive `syscall_setsockopt syscall.setsockopt`

