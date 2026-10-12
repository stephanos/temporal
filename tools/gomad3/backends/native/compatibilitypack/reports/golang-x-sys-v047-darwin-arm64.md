# Compatibility Pack Review: golang-x-sys-v047-darwin-arm64

Review SHA-256: `sha256:b5953a9fc5af8f4f9091a7b2df3346f1153d43169b59db1c19b892dbb2b310b5`

Owner: `temporal-server`

Reviewed at: `2026-09-30T00:00:00Z`

Justification: Admits the golang.org/x/sys v0.47.0 unix and cpu darwin/arm64 assembly, syscall imports, and runtime and libSystem linknames for any closure that reaches x/sys, activated by x/sys alone rather than by the modernc libc adapter, plus the x/sys imports of the exact golang.org/x/term and golang.org/x/crypto/sha3 versions the server pins. Closures without SQLite never activate the libc-bound x/sys packs.

Target: `go-test .`

Target module: `gomad3.compatibility.xsys`

Test arguments: `-test.run ^TestXSysCompatibilityClosure$`

Build tags: `test_dep`

Platform: `darwin/arm64`

Workload: `golang-x-sys-fixture`

## Activation

- `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`

## Reviewed packages

### `golang.org/x/crypto/sha3`

Module: `golang.org/x/crypto@v0.55.0` (`h1:+KWHjbgOaAQ66dh/YlkZKHlz9ZUlq61AFirAR9ntP8M=`), replacement `none`

Source set: `sha256:bea60e6d5433d0955b533ecd2ea799c2de5b58fa1f8acdfa10a224aa3788939a`

Go sources:

- `hashes.go`: `sha256:ee3a3940020475dea7cd93e4d67cf9092c846ce3a141901835cc70595cb34893`
- `legacy_hash.go`: `sha256:fcb820da94155a697afeb404731291b98f028bda6dfcc1e865c3d65671acbeb7`
- `legacy_keccakf.go`: `sha256:cd0fa888d34f8e39b435fe7b19c942dd229fe72861f5b3706c9306f848215075`
- `shake.go`: `sha256:45266f3ce645f53d2540d1d4a33528f3da6988885ae85b733d1f80878ed42960`

Requested facts:

- `import:golang.org/x/sys/cpu`: **allow** — **security-sensitive**

### `golang.org/x/sys/cpu`

Module: `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`

Source set: `sha256:c2d8bbd51ab58d5bdacdd3fcbe01c335eb2d007ceea9505beaacdfb883e4c2ce`

Go sources:

- `byteorder.go`: `sha256:825146fd4557b1cbd8161fa28bb4be8820089848d695316edeecb7fd5a551f8a`
- `cpu.go`: `sha256:b56854737e8d803d582232f9790f7dda2d8ff69e5982f6cdbdd2a150be540922`
- `cpu_arm64.go`: `sha256:8497751085f331e75f297ed0eef3ceb7c31725f63e3a142a543ec2d29befae36`
- `cpu_darwin_arm64.go`: `sha256:b8d13328eed17809afb4f807512caded369fdbd2ecdd78f80d54651079bb12a3`
- `cpu_gc_arm64.go`: `sha256:58e20a70f4400b4969c85e01f017f36bd6eeb240fb1c49799cd18ccdf8bd8bc2`
- `endian_little.go`: `sha256:c6bc70c372d9e1fe86fcf295f406b17bf04bf8d1af25c2456f58520cdaef3be9`
- `parse.go`: `sha256:97b269ea4f0b6d4071a9eed8a74f05055965c307bbab9090d9002bb01f7365a9`
- `runtime_auxv.go`: `sha256:d898ace395866bed261d403c8cd0ea6eab6d6d77f52042204957e389c38938cf`
- `runtime_auxv_go121.go`: `sha256:6eee9d1a593dce53d22545dc5d5f6ed9127a43e6568ef5f5521920946510d445`
- `syscall_darwin_arm64_gc.go`: `sha256:2ec90937ba10bdeb3a646e3fcefd531bd99ce0c3653eaa6c8de2c6d86362e758`

Foreign sources:

- `assembly:asm_darwin_arm64_gc.s`: `sha256:6f6f35fbf284f205f49db2410b6554bd72aa1fc8ccf35f30a619482277e31ccd`
- `assembly:cpu_arm64.s`: `sha256:704264fabff92f961c0bc68f0cb3cc49a21f62141e44ba709caf0ee2eccb0e5d`

Requested facts:

- `foreign:assembly:asm_darwin_arm64_gc.s`: **allow** — **security-sensitive**
- `foreign:assembly:cpu_arm64.s`: **allow** — **security-sensitive**
- `import:syscall`: **allow** — **security-sensitive**
- `linkname:runtime_auxv_go121.go`: **allow** — **security-sensitive**
  - source `sha256:6eee9d1a593dce53d22545dc5d5f6ed9127a43e6568ef5f5521920946510d445`
  - directive `runtime_getAuxv runtime.getAuxv`
- `linkname:syscall_darwin_arm64_gc.go`: **allow** — **security-sensitive**
  - source `sha256:2ec90937ba10bdeb3a646e3fcefd531bd99ce0c3653eaa6c8de2c6d86362e758`
  - directive `syscall_syscall6 syscall.syscall6`

### `golang.org/x/sys/unix`

Module: `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`

Source set: `sha256:ad888b33278a421fd2ee7592a057aa350997e935dfecb68659d665a747dc4042`

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

- `foreign:assembly:asm_bsd_arm64.s`: **allow** — **security-sensitive**
- `foreign:assembly:zsyscall_darwin_arm64.s`: **allow** — **security-sensitive**
- `import:syscall`: **allow** — **security-sensitive**
- `linkname:auxv.go`: **allow** — **security-sensitive**
  - source `sha256:5e470a481610ff746d64cb22b3e7a981ffa527d6ca546e87df4296704d6c6de6`
  - directive `runtime_getAuxv runtime.getAuxv`
- `linkname:syscall_darwin_libSystem.go`: **allow** — **security-sensitive**
  - source `sha256:0c327ad9b9845e19b1e097dfb7b569bc9793b670874e24be4a87ac9bd4647557`
  - directive `syscall_syscall syscall.syscall`
  - directive `syscall_syscall6 syscall.syscall6`
  - directive `syscall_syscall6X syscall.syscall6X`
  - directive `syscall_syscall9 syscall.syscall9`
  - directive `syscall_rawSyscall syscall.rawSyscall`
  - directive `syscall_rawSyscall6 syscall.rawSyscall6`
  - directive `syscall_syscallPtr syscall.syscallPtr`

### `golang.org/x/term`

Module: `golang.org/x/term@v0.45.0` (`h1:NwWyBmoJCbfTHpxrWoZ9C6/VxOf7ic219I8xZZFdrf0=`), replacement `none`

Source set: `sha256:77d7cf8a20e9f560463747b59a5ff0551332e143df2ad1f5c5df1bf94a37b3a7`

Go sources:

- `term.go`: `sha256:9792d91be6b3e5195591a2e5bda1fd01cb19954df79fd844de4d429538c1c5d5`
- `term_unix.go`: `sha256:e6a95d1ac948b5e2b389676545101de744f2715766e77f55160375a49a765b15`
- `term_unix_bsd.go`: `sha256:582326355334ff9492a712458087a16b705c5855d44c30ed1ae45c0aeb0be7af`
- `terminal.go`: `sha256:10c7bc02c516305135c0ce4dca998d0505b8a61d1f653d0d1e92f4f77ec12d3b`

Requested facts:

- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**

