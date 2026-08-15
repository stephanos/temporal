# Compatibility Pack Review: temporal-backoff-overflow

Review SHA-256: `sha256:a3db8a539e2b4d434fd753ae2a735404f2f330a6bf1e44469a3944b0ef5d2e2d`

Owner: `temporal-server`

Reviewed at: `2026-08-15T00:00:00Z`

Justification: Records the seven exact current blockers without authorizing the live gRPC TCP keepalive path through syscall.RawConn and unix.SetsockoptInt; COMPAT-6 absence proof or a deterministic adapter is required.

Target: `go-test ./common/backoff`

Target module: `go.temporal.io/server`

Test arguments: `-test.run ^TestExponentialBackoffOverflow$`

Build tags: `test_dep`

Platform: `darwin/arm64`

Workload: `temporal-backoff-overflow`

## Activation

- `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`
- `google.golang.org/grpc@v1.80.0` (`h1:Xr6m2WmWZLETvUNvIUmeD5OAagMw3FiKmMlTdViWsHM=`), replacement `none`

## Reviewed packages

### `golang.org/x/sys/unix`

Module: `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`

Source set: `sha256:3a848e68c862f9d4b55329cf0debcfc8b29c865612b4e1ce9aa5164514347753`

Go sources:

- `aliases.go`: `sha256:53e7eeba0503ad62ec18cdf2ca51a1785249a8646354439c854148dc57c06fb5`
- `auxv.go`: `sha256:5e470a481610ff746d64cb22b3e7a981ffa527d6ca546e87df4296704d6c6de6`
- `constants.go`: `sha256:f3405abc7484992964143eac589b951132d8e3a90f8359fc1a6e9ddcc201aa8a`
- `dev_darwin.go`: `sha256:9a0bc8af77b4325bb10b651e00b8f7974cc972d0e5456a370f2c46a56181ada7`
- `dirent.go`: `sha256:03e3b15a8428e2f1520386291052fd30e5d74ecb4d78c724bae7953d71425be1`
- `endian_little.go`: `sha256:bc06276262c57cf21e35c13ff2f5fdc96c18feb32a055c6201edfa76e95c967f`
- `env_unix.go`: `sha256:bcb73ccfc5a8dae1f59e4debba69e0f600155b947f358c939bc753443a7a8007`
- `fcntl_darwin.go`: `sha256:fb6aa54ed72a392548bcd7c79a10ce16e9ed70da90492ef13346e2419fa52d3f`
- `fdset.go`: `sha256:8339dad44215930bdd253beaa1064c4d34ed2f81d54c4bd53d2be1f64fe35530`
- `ioctl_unsigned.go`: `sha256:991b152e32cde753f3b64d8ebc1d18e9078a3615006bc79c474b2bbbc653487a`
- `mmap_nomremap.go`: `sha256:f6166771a7d9a6c116617b6e21a15b020ddf709b39162c8596e0880cef3ce4be`
- `pagesize_unix.go`: `sha256:5fea300c32898efe55341abb39489aeac76f81ea9df513f68d1b2556be7b48a3`
- `ptrace_darwin.go`: `sha256:61eab9ab1e3d5d24b8e3545352e97afa0f76a77a578f75fb078026617444052c`
- `race0.go`: `sha256:8a78192af2b20cd177f78e3761e10d66567373cef3b7671aa3c7912d65de4c90`
- `readdirent_getdirentries.go`: `sha256:db3a5b0e169d6f2431e3a145d2a3cc4374330e29d40378cb3fb26ede4fcd813d`
- `readv_unix.go`: `sha256:2abd5ef6af8852208f2bae9de6de1c8d90ae33caba76f7fb757004d7a4040fe8`
- `sockcmsg_unix.go`: `sha256:7863093bfa7ec9e564c36992fbf2b7c97244c84d406e5fdc60b72c56a23af396`
- `sockcmsg_unix_other.go`: `sha256:73f23d4f0c002e24160530515a45f46bb6fbe8728cc79b135a8c6956d5c8458c`
- `syscall.go`: `sha256:41abaa37d079eee890fb7803dde876a19ba3acdbadf41f269ae8ed0d73ca0e37`
- `syscall_bsd.go`: `sha256:53d6db23c6307444ad18512ee160159c175608c0d424a86ec22dec6b28eeda65`
- `syscall_darwin.go`: `sha256:8928e516d815b6d95ff8d6b7ffdbf88843e47076737d885bd81d68bfa8f825ed`
- `syscall_darwin_arm64.go`: `sha256:ca9f22a90e5e81a736c0ad500f015b139cbb05a23ffdbc39906c6feb6975244e`
- `syscall_darwin_libSystem.go`: `sha256:0c327ad9b9845e19b1e097dfb7b569bc9793b670874e24be4a87ac9bd4647557`
- `syscall_unix.go`: `sha256:d851dcf05549674486f35d58b0357bb5c1ea9378bd234b1646534a35dc4a6da5`
- `syscall_unix_gc.go`: `sha256:8b7592bca0fff629f9bb6f2c78c4ec8810f989e04b29d76ebd1ff81efd34db5b`
- `sysvshm_unix.go`: `sha256:e9e5031e048cf7c58692650ea14887158c3bedaaba506be63cbd8190cea2c066`
- `sysvshm_unix_other.go`: `sha256:b513c4e9cd077df2b1452bd645abcc3e252ec28ef44cf4e2a984d94bd5464fb3`
- `timestruct.go`: `sha256:d0d07c2481ce2692f4e5728d65ccab1d105ecf64f52ac6b9a923ef106088a249`
- `vgetrandom_unsupported.go`: `sha256:822c28801c556c34f31cd7a14ddbe4c3708e543de6fe81c4ad1fae9420268258`
- `zerrors_darwin_arm64.go`: `sha256:a4255cecfc6a5e82653d5bb96ee2ef6191a3fa664046de7a016ddeccbf2c9e5f`
- `zsyscall_darwin_arm64.go`: `sha256:630f26d7c5679ac8ac11dfe0e7e1861aec0c801e1ef7ca503030f8e8736469a3`
- `zsysnum_darwin_arm64.go`: `sha256:3153f86ca570545e32a352b93289a5e5353cb32902c8ba0e8135d2dc26a10312`
- `ztypes_darwin_arm64.go`: `sha256:b17d2e512a7bab550ef24d3d3ed9b0f3bed89de5a15ac003d7371058d6ea168e`

Foreign sources:

- `assembly:asm_bsd_arm64.s`: `sha256:f7740a9d925eccd280e54e7971a36508a7d2856d9ef996a394ad5cfd80bec8c3`
- `assembly:zsyscall_darwin_arm64.s`: `sha256:5daa70eefd10942e6ba8da79d69152b330da1981874d6726d1d09cf8d8a0d30e`

Requested facts:

- `foreign:assembly:asm_bsd_arm64.s`: **deny** — **security-sensitive**
- `foreign:assembly:zsyscall_darwin_arm64.s`: **deny** — **security-sensitive**
- `import:syscall`: **deny** — **security-sensitive**
- `linkname:auxv.go`: **deny** — **security-sensitive**
  - source `sha256:5e470a481610ff746d64cb22b3e7a981ffa527d6ca546e87df4296704d6c6de6`
  - directive `runtime_getAuxv runtime.getAuxv`
- `linkname:syscall_darwin_libSystem.go`: **deny** — **security-sensitive**
  - source `sha256:0c327ad9b9845e19b1e097dfb7b569bc9793b670874e24be4a87ac9bd4647557`
  - directive `syscall_syscall syscall.syscall`
  - directive `syscall_syscall6 syscall.syscall6`
  - directive `syscall_syscall6X syscall.syscall6X`
  - directive `syscall_syscall9 syscall.syscall9`
  - directive `syscall_rawSyscall syscall.rawSyscall`
  - directive `syscall_rawSyscall6 syscall.rawSyscall6`
  - directive `syscall_syscallPtr syscall.syscallPtr`

### `google.golang.org/grpc/internal`

Module: `google.golang.org/grpc@v1.80.0` (`h1:Xr6m2WmWZLETvUNvIUmeD5OAagMw3FiKmMlTdViWsHM=`), replacement `none`

Source set: `sha256:46e406452f4634f805707736db893a9524e66a35cb2c32c4c05f871f5afb5868`

Go sources:

- `experimental.go`: `sha256:cee5c034eb7a26d6d20fb30105ac8961b450ee4082dd9b9e7a3cd37b83029610`
- `internal.go`: `sha256:9dabad1ab1175eebe6029a8de36a16ebcea786f60c6fb0af18530a3ab66ef36f`
- `tcp_keepalive_unix.go`: `sha256:e8bfe03234b391d24006a3a274590111f0f8705fc5b25d9a78391bfdde3df32c`

Requested facts:

- `import:golang.org/x/sys/unix`: **deny** — **security-sensitive**
- `import:syscall`: **deny** — **security-sensitive**

