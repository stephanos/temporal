# Compatibility Pack Review: modernc-libc-xsys-v047

Review SHA-256: `sha256:61722d0b61a7833d20069f13f15a39ad9c3a830677ab0514f94ae1d8dd4e9f0e`

Owner: `temporal-server`

Reviewed at: `2026-08-15T00:00:00Z`

Justification: Preserves the exact registered modernc libc adapter boundary used by the core SQLite qualification workload.

Target: `go-test ./thirdparty/persistence`

Target module: `gomadv3.core.corpus`

Test arguments: `-test.run ^TestSQLiteCommitAndRollbackPreserveState$`

Build tags: `test_dep`

Platform: `darwin/arm64`

Workload: `modernc-libc-boundary`

Workload: `sqlite-transaction`

## Activation

- `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`
- `modernc.org/libc@v1.72.3` (`h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`), replacement `adapter`
  - profile `gomadv3-deterministic/v1` / `sha256:034755da63de6446baa5c7fefaaecaeb03c1e18c753ed18fcedbf17a76813610`
  - adapter `modernc.org/libc@v1.72.3` / `h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`
  - source inventories `sha256:6a2ed9798fa07019c328f0247548082ef51b21aad8829c5600168aac4f683429` → `sha256:8579228404e49a9df26f1a5f735cd530e17f6264ed1c231bf15051d20b2cc76c`
  - prepared source set `sha256:8e1663c90aa178a706929ae94f248051781e4278ca83991d9a5fc6fe05321833`

## Reviewed packages

### `github.com/mattn/go-isatty`

Module: `github.com/mattn/go-isatty@v0.0.20` (`h1:xfD0iDuEKnDkl03q4limB+vH+GxLEtL/jb4xVJSWWEY=`), replacement `none`

Source set: `sha256:a5f81d700b1f9b93da0a2fe3637cac475f972d057b015ea7e6dc76fef7d4b309`

Go sources:

- `doc.go`: `sha256:06182cb1a7113cae6fdef9be492893298610bfc63cf565a23f86203c3074a861`
- `isatty_bsd.go`: `sha256:b3df65aaddc2e985cc4b41be48e7a714eea17414cb8d09a04abcd2d35bf3f9e8`

Requested facts:

- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**

### `github.com/remyoudompheng/bigfft`

Module: `github.com/remyoudompheng/bigfft@v0.0.0-20230129092748-24d4a6f8daec` (`h1:W09IVJc94icq4NjY3clb7Lk8O1qJ8BdBEF8z0ibU0rE=`), replacement `none`

Source set: `sha256:2cc15ec5f7b5999948c5666e836b72903a01c6943e132b45a5775e0b0f86c0fd`

Go sources:

- `arith_decl.go`: `sha256:652c090c62611633839e469d5ffb454e8a0afca61cef680f349778758921ca87`
- `fermat.go`: `sha256:25b1256c862303f46e796f1d8a9911da106b28e7bab29f75a59c2a96f964db13`
- `fft.go`: `sha256:bc123dfafd49301821768f73acd2d46a0fac28b582902bc982352752344be59a`
- `scan.go`: `sha256:b079e8e278ca3a14d0da9e3d719ab0fbd843047bc6020bfcce8626f8c27e3715`

Requested facts:

- `linkname:arith_decl.go`: **allow** — **security-sensitive**
  - source `sha256:652c090c62611633839e469d5ffb454e8a0afca61cef680f349778758921ca87`
  - directive `addVV math/big.addVV`
  - directive `subVV math/big.subVV`
  - directive `addVW math/big.addVW`
  - directive `subVW math/big.subVW`
  - directive `shlVU math/big.shlVU`
  - directive `mulAddVWW math/big.mulAddVWW`
  - directive `addMulVVW math/big.addMulVVW`

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

### `modernc.org/libc`

Module: `modernc.org/libc@v1.72.3` (`h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`), replacement `adapter`

Source set: `sha256:8e1663c90aa178a706929ae94f248051781e4278ca83991d9a5fc6fe05321833`

Go sources:

- `builtin_all.go`: `sha256:5b2c1479edaaba777d817dbccad1fb41ee26fed87c576f70a427cb3c505c8fea`
- `capi_darwin_arm64.go`: `sha256:bff4f0b98aef95bd0e53eb7e77ff5e5ecbc4ffddb1647db741e28b1cf8ecc48b`
- `ccgo.go`: `sha256:c49dacf7c917bbab19692914b4efa950b6f461117014f7364c818fb39069b92a`
- `etc.go`: `sha256:b48935972f453553e9a623cde5ccd6810e9036df2561c6f70edc3369703244cf`
- `fsync.go`: `sha256:5760bc90f3df02bbbe2a24dd08365a403b8bbe9a01f6ee9ee345249f7f698b62`
- `gomad_darwin.go`: `sha256:751f42d790ea150f57977ae75189909eeb8ad0b55f3aee7bd5ede3e0f92f10cd`
- `int128.go`: `sha256:fa4821cd943874028ba6a953e0ee48c480759f98ea1ade6e09d1cd64a8dd7cb4`
- `ioutil_darwin.go`: `sha256:a496b5553d12820e4a9a7adc62970a6b6a5481933b9a30679fd3d95aefd1e06c`
- `libc.go`: `sha256:e37b0c81c65de523307acd877a02c0ce372fbbec0aee47126cb169ff27d18224`
- `libc64.go`: `sha256:8890988534e74862883cd891f5755203837b10b3b6f59210f06f644f12f390a0`
- `libc_all.go`: `sha256:dcd4d1f818059ab5a5f9b430b3cacde4290a15907467ed4e396169d64b778c6c`
- `libc_arm64.go`: `sha256:e176bc579a8e7d8a7188c1ee6142fc1c17af50a777d693273155133dd384d327`
- `libc_darwin.go`: `sha256:44c0ccbbb5c890dc4b10a63a199a5a9b0185724a54b4f45f0e7b0aafd4f50376`
- `libc_darwin_arm64.go`: `sha256:dc9ff2436a24e90368bd0dae18a6f1bce7f031883ac4029b9436b721888f1ac7`
- `libc_unix.go`: `sha256:f603427bda270dd60ac1a15b41326b50648b15d48dbc43d72e4588320eeb7301`
- `libc_unix1.go`: `sha256:70ab4175ef2801300f3b2922a42b98bd8c7fd22be4f7b4d3d72c84b0acf5eb5d`
- `libc_unix3.go`: `sha256:d1c2fcf135c7664801573c5ea2723e0c742ba93d6457a91cf0d4a8bc25a14a3e`
- `mem.go`: `sha256:b6569a198852a035e300c345172db81098eb78ab55784c41e12decda45661746`
- `musl_darwin_arm64.go`: `sha256:00d8825713eb0d5b01c2d85ea5bb9b87d69f39ddae289bc495ce9fa506fc5785`
- `nodmesg.go`: `sha256:18232e1a56ef0cc8f9d588d7392b3890e37820047f352e8f24889ebd78a16bde`
- `printf.go`: `sha256:981767759c5f68e9ce2a24486583574f223d2300e78e57789a1f4f5cae92e8fa`
- `probes.go`: `sha256:7712b62c336e3e39d145c2824afe7271d989170873d552da9774c544fec1dc43`
- `pthread.go`: `sha256:48782a1eef6c465f28089f93d1b816d530d336720304f26a5c2c10c45fc8ad1a`
- `pthread_all.go`: `sha256:505569f6ba4684e9f9599bea2d2ec5b70c857a296ca3efda32bc867055b27006`
- `scanf.go`: `sha256:f93c2de58f1a23ade56bfecd1d19886cdfb79a651675eaa1f2452435962906c8`
- `stdatomic.go`: `sha256:d90c1342fc268f89020ebdb4f3886ee3572837d5cb30c02992c5153654614d6b`
- `straceoff.go`: `sha256:9b1d5deaa41eb28d23a89c0ac4fc06d827d3a46a57016559342e31bcd16a4b7f`
- `sync.go`: `sha256:a65152029198a7807eb16721dffe0ab9f7e2f8a3b746e8517347e8e6c7eeaa2a`
- `watch.go`: `sha256:d4ba707bdd7dddc533659a1ab47c80caf0a87d5df2c6844ac8fadbc60d7c4a25`

Requested facts:

- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**
- `import:os/exec`: **allow** — **security-sensitive**
- `import:os/signal`: **allow**
- `import:syscall`: **allow** — **security-sensitive**
- `linkname:gomad_darwin.go`: **allow** — **security-sensitive**
  - source `sha256:751f42d790ea150f57977ae75189909eeb8ad0b55f3aee7bd5ede3e0f92f10cd`
  - directive `gomadLibcEnabled internal/gomadio.Enabled`
  - directive `gomadLibcOpen internal/gomadio.LibcOpen`
  - directive `gomadLibcClose internal/gomadio.LibcClose`
  - directive `gomadLibcRead internal/gomadio.LibcRead`
  - directive `gomadLibcWrite internal/gomadio.LibcWrite`
  - directive `gomadLibcSeek internal/gomadio.LibcSeek`
  - directive `gomadLibcTruncate internal/gomadio.LibcTruncate`
  - directive `gomadLibcSync internal/gomadio.LibcSync`
  - directive `gomadLibcMmap internal/gomadio.LibcMmap`
  - directive `gomadLibcMunmap internal/gomadio.LibcMunmap`
  - directive `gomadLibcRemove internal/gomadio.LibcRemove`
  - directive `gomadLibcRename internal/gomadio.LibcRename`
  - directive `gomadLibcMkdir internal/gomadio.LibcMkdir`
  - directive `gomadLibcAccess internal/gomadio.LibcAccess`
  - directive `gomadLibcStat internal/gomadio.LibcStat`
  - directive `gomadLibcIsDescriptor internal/gomadio.LibcIsDescriptor`
  - directive `gomadLibcNow internal/gomadio.LibcNow`

### `modernc.org/memory`

Module: `modernc.org/memory@v1.11.0` (`h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI=`), replacement `none`

Source set: `sha256:55c4d42b5ce341d5b92d7a319c47f4f0fc48ead7ec8aa4ac2fdb6b09d5d3429e`

Go sources:

- `memory.go`: `sha256:3f5cee8943da57ffd3db70129f87abdf45c86e5299b81d09504ec2838dc31909`
- `memory64.go`: `sha256:1b91a327f3b95d6fc3f80f090175a9c5bc033083a2c675b7a56f009af8fa4813`
- `mmap_unix.go`: `sha256:d487e0d7f447b25397874a79e53c0e42b8568ed0503b562c959c83e8ef47f0a7`
- `nocounters.go`: `sha256:070021a593fc3c28988ad04bee02d83547ff56a9665e7703a80366392e30c18c`
- `trace_disabled.go`: `sha256:ecd151b29853826767c9fd0e2c9b48b4acd08bd211ce558016399afae693b950`

Requested facts:

- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**

### `modernc.org/sqlite`

Module: `modernc.org/sqlite@v1.51.0` (`h1:aH/MMSoayAIhozZ7uJbVTT9QO/VhzBf0J9tymmmuC/U=`), replacement `none`

Source set: `sha256:29f4683488f42fb072ac4c34dfd633fdd76e178205564c6b128e1bff038d67b1`

Go sources:

- `backup.go`: `sha256:f0f1fb41fc30e716181cde96c06c942b2a57fed7888f5356e0f8a254114a3c66`
- `conn.go`: `sha256:f768b85b92bd4b5c43440ebeaf817492a9a977b50203a085c4b8f65fd8c7a9fe`
- `convert.go`: `sha256:4a36eadd13bbbead694c7b946638e3444c10888ba36c9e46c78d3d8a8dcfb245`
- `doc.go`: `sha256:4f69916cc4abfc10063cf56725197095bcbe81bc1ac4b47462c3c391ab1221c1`
- `driver.go`: `sha256:07adacaf190fea84c8134e15230417b3de0f6c572215c9e2bbad7d7b42540474`
- `error.go`: `sha256:0a112a84c6c9e052df56673913989913180c49e43ef99655cd6e2ba66182d2d7`
- `fcntl.go`: `sha256:d3fc3e9ea4ad02a456534bb890536ef5eaa1afed48abb4f75c64f30b6b3b0bcb`
- `mutex.go`: `sha256:a35ec3aa1d40010e5031c5e861b4ffa4fd7b50162bea17640e2ccae787f1fefb`
- `nodmesg.go`: `sha256:a463819cb7b9878057b2a6373f5aad2440f8c2efefa656c1fb9ed78c71e91ed7`
- `pre_update_hook.go`: `sha256:4c062cbd09d1223b8c4d38087a60a1309e1e38a0fad7997e2feb1bd5d2a01bf9`
- `result.go`: `sha256:c5830f2e1ad4fb343e9d1952e0a1c5b9b2c456b4d23613ce3eb74ee13ccbfe6a`
- `rows.go`: `sha256:09175589263bbcf61bef21fbf9a15a37bfc6e18dda4f88c26a4e03a5cfbfa4b0`
- `rulimit.go`: `sha256:6528c8b341bfc99dfc31084fc1c21c3056f5bf29612f8ad9574b22aa2ea23566`
- `sqlite.go`: `sha256:ec9406735b0d6ea66a59334d974523f44a163fc86dbb5426b1bde80bd2cd2163`
- `stmt.go`: `sha256:ae8a35a88b725796ca41ce2c1c36bb0d49780f28cd2beee7faf587d0ce67d9de`
- `tx.go`: `sha256:f33774d1d59fd03f4170edd85441de487a8c3ca820e0eb1264e2f51150496106`
- `vtab.go`: `sha256:d4713c601a2a3a5b8f476692c681c7788bf63d75b5bf17f042d5f1ac3f70feb0`

Requested facts:

- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**

