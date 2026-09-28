# Compatibility Pack Review: modernc-libc-xsys-v047-linux-amd64

Review SHA-256: `sha256:81f19cd41f5aeb27b07435def9cf2322315b05f89fd76faeafe4f71def782181`

Owner: `temporal-server`

Reviewed at: `2026-09-27T00:00:00Z`

Justification: Preserves the exact registered modernc libc and memory adapter boundaries used by the core SQLite qualification workload on linux/amd64, where the musl translation reaches the kernel only through the hooked syscall trampolines.

Target: `go-test ./thirdparty/persistence`

Target module: `gomad3.core.corpus`

Test arguments: `-test.run ^TestSQLiteCommitAndRollbackPreserveState$`

Build tags: `test_dep`

Platform: `linux/amd64`

Workload: `modernc-libc-boundary`

Workload: `sqlite-transaction`

## Activation

- `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`
- `modernc.org/libc@v1.72.3` (`h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`), replacement `adapter`
  - profile `gomad3-deterministic/v1` / `sha256:fb9e07ea6a2d35e9b1283d02428e35a20bf0120b72e8b7e04178c9718ece4834`
  - adapter `modernc.org/libc@v1.72.3` / `h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`
  - source inventories `sha256:7455942bdfcf64ff4d46cd874f1f6e80a79e4ebe6f4d98a9c2d1ae6aaabb59da` → `sha256:325d1051e2fb18b4acc64a646144a2a2051d31e155e89328b6b618824652fe48`
  - prepared source set `sha256:2fd5cdb4987b3319011b56c1244100a8479a51a65ad561357aaea93db4c9aa9b`
- `modernc.org/memory@v1.11.0` (`h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI=`), replacement `adapter`
  - profile `gomad3-deterministic/v1` / `sha256:fb9e07ea6a2d35e9b1283d02428e35a20bf0120b72e8b7e04178c9718ece4834`
  - adapter `modernc.org/memory@v1.11.0` / `h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI=`
  - source inventories `sha256:f6d838c731cb22881e472d086c21895e431123e0a08fd190b41ff34be59f0264` → `sha256:b947d0e7fbcc3f18b995e5710e5d26d8073bf921ba1f907ef2b3972f4257a535`
  - prepared source set `sha256:40ac8382ecbdbb2b46f418da2ce63a2cc7a188971a21c31116ed832ddca5f849`

## Reviewed packages

### `github.com/remyoudompheng/bigfft`

Module: `github.com/remyoudompheng/bigfft@v0.0.0-20230129092748-24d4a6f8daec` (`h1:W09IVJc94icq4NjY3clb7Lk8O1qJ8BdBEF8z0ibU0rE=`), replacement `none`

Source set: `sha256:60700dfdb108b1900cde07a08b8af03f9a7562a0633267d75946a2129b8ea85c`

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

Source set: `sha256:b35f4fad1ca77dd4ad562563eb5455aa9d68ecacae8f1d4961cf11978890f75c`

Go sources:

- `affinity_linux.go`: `sha256:b6c998fdf681f97006b0dae3ba0bf5d6d05a9f86fcc94f1518e37eab9dbc990a`
- `aliases.go`: `sha256:53e7eeba0503ad62ec18cdf2ca51a1785249a8646354439c854148dc57c06fb5`
- `auxv.go`: `sha256:5e470a481610ff746d64cb22b3e7a981ffa527d6ca546e87df4296704d6c6de6`
- `bluetooth_linux.go`: `sha256:72ce84a68d3647cd34f04e9040861333ac9e9f9ac5b00008d41514c79da664d8`
- `constants.go`: `sha256:f3405abc7484992964143eac589b951132d8e3a90f8359fc1a6e9ddcc201aa8a`
- `dev_linux.go`: `sha256:eec381cf025f58965728544dae5df9394d6bebc7e68572e429814c87ee6dd284`
- `dirent.go`: `sha256:03e3b15a8428e2f1520386291052fd30e5d74ecb4d78c724bae7953d71425be1`
- `endian_little.go`: `sha256:bc06276262c57cf21e35c13ff2f5fdc96c18feb32a055c6201edfa76e95c967f`
- `env_unix.go`: `sha256:bcb73ccfc5a8dae1f59e4debba69e0f600155b947f358c939bc753443a7a8007`
- `fcntl.go`: `sha256:e8d84df0c6ed38a56014cc73d431aa973f1d298169735eaf9f7d42cbb26f7768`
- `fdset.go`: `sha256:8339dad44215930bdd253beaa1064c4d34ed2f81d54c4bd53d2be1f64fe35530`
- `ifreq_linux.go`: `sha256:a950bf9eec4d9385836bd0dd400c6218d5033585f9cb15135d110848a20af5c3`
- `ioctl_linux.go`: `sha256:b0de1db3834fe4cb2e23e36e65c43614b493cb4c29c98f63f2b9a7e7e66972a1`
- `ioctl_unsigned.go`: `sha256:991b152e32cde753f3b64d8ebc1d18e9078a3615006bc79c474b2bbbc653487a`
- `mremap.go`: `sha256:c4f762c0f38d6a9f08c9b754b207a1ba64dd29d03692245171f8a344aac6935a`
- `pagesize_unix.go`: `sha256:5fea300c32898efe55341abb39489aeac76f81ea9df513f68d1b2556be7b48a3`
- `race0.go`: `sha256:8a78192af2b20cd177f78e3761e10d66567373cef3b7671aa3c7912d65de4c90`
- `readdirent_getdents.go`: `sha256:50e29bed47a256b79cb3754f43d0f8c1a5ea762e400bf809d0eb44d16ddd4714`
- `readv_unix.go`: `sha256:2abd5ef6af8852208f2bae9de6de1c8d90ae33caba76f7fb757004d7a4040fe8`
- `sockcmsg_linux.go`: `sha256:51d948aa226aa043c2760288aa883f9dbd53b9d65cf8bc20215146fc016dde6f`
- `sockcmsg_unix.go`: `sha256:7863093bfa7ec9e564c36992fbf2b7c97244c84d406e5fdc60b72c56a23af396`
- `sockcmsg_unix_other.go`: `sha256:73f23d4f0c002e24160530515a45f46bb6fbe8728cc79b135a8c6956d5c8458c`
- `syscall.go`: `sha256:41abaa37d079eee890fb7803dde876a19ba3acdbadf41f269ae8ed0d73ca0e37`
- `syscall_linux.go`: `sha256:5a1f64326a3de29148008a364f560fa8c8227238ca1b681ba5bec2e7c266bb98`
- `syscall_linux_alarm.go`: `sha256:eede15c54c7c4dfc0691ae0e646fd31aca15bfd72f53783081e829cd150b19e0`
- `syscall_linux_amd64.go`: `sha256:dfb1ba340e05d035b27aec5624c4743b5b686607ece9ab8d39eed329ac179df4`
- `syscall_linux_amd64_gc.go`: `sha256:6c5b86f3c4aa3dcf8619c5d444491414f72e014767e57fd9ad62f57653f4d837`
- `syscall_linux_gc.go`: `sha256:41a54001d221205e9f1538066105b46c08cd329e85d4529426063961daa86235`
- `syscall_unix.go`: `sha256:d851dcf05549674486f35d58b0357bb5c1ea9378bd234b1646534a35dc4a6da5`
- `syscall_unix_gc.go`: `sha256:8b7592bca0fff629f9bb6f2c78c4ec8810f989e04b29d76ebd1ff81efd34db5b`
- `sysvshm_linux.go`: `sha256:8f932aac3521059ca3b4156c78b1ec45f1976972d1f70dee14375a1b7a507e44`
- `sysvshm_unix.go`: `sha256:e9e5031e048cf7c58692650ea14887158c3bedaaba506be63cbd8190cea2c066`
- `timestruct.go`: `sha256:d0d07c2481ce2692f4e5728d65ccab1d105ecf64f52ac6b9a923ef106088a249`
- `vgetrandom_linux.go`: `sha256:4dfa240c42977f70ef32d1cea7effdbd902b5ef6814e67f20c48da10548b4bc9`
- `zerrors_linux.go`: `sha256:28438683d95a7fcae9d95d33cdfa19e2ae3f41247cc1ae561701553c0c7dbbdf`
- `zerrors_linux_amd64.go`: `sha256:cce71ceeb02fa2e1343b602fcad5b32faf879d3f52f1888a50e58ecaf9776907`
- `zptrace_x86_linux.go`: `sha256:dc4b1c692caaaab1e50e386776a08ce44e3a4d2def73513bb4fa8d04332ac631`
- `zsyscall_linux.go`: `sha256:2903ee63248999159b31618b694ef2be85ec4bc917d91bcef019a2bf1deccf9a`
- `zsyscall_linux_amd64.go`: `sha256:a492772c3ad8ae8600283c2223289426bfaa3f75666ef62a77914b1ca140f2eb`
- `zsysnum_linux_amd64.go`: `sha256:b409916de5e1eb0f91344bba82dbb5971736718ac2812560699e23862e9f01d3`
- `ztypes_linux.go`: `sha256:a5eab77c5555033a363e4fa1dd5cb5d4a4f0bfaf6dedb77ccbcb32990e121790`
- `ztypes_linux_amd64.go`: `sha256:f057c46400ef36587cd8516928f81e934f65d81dcfed58f5b8b2afe52a658664`

Foreign sources:

- `assembly:asm_linux_amd64.s`: `sha256:14c826e5d2db337e49c32e0b5a66317b58da198874a0eb950c33aac571e9573c`

Requested facts:

- `foreign:assembly:asm_linux_amd64.s`: **allow** — **security-sensitive**
- `import:syscall`: **allow** — **security-sensitive**
- `linkname:auxv.go`: **allow** — **security-sensitive**
  - source `sha256:5e470a481610ff746d64cb22b3e7a981ffa527d6ca546e87df4296704d6c6de6`
  - directive `runtime_getAuxv runtime.getAuxv`
- `linkname:syscall_linux.go`: **allow** — **security-sensitive**
  - source `sha256:5a1f64326a3de29148008a364f560fa8c8227238ca1b681ba5bec2e7c266bb98`
  - directive `syscall_prlimit syscall.prlimit`
- `linkname:vgetrandom_linux.go`: **allow** — **security-sensitive**
  - source `sha256:4dfa240c42977f70ef32d1cea7effdbd902b5ef6814e67f20c48da10548b4bc9`
  - directive `vgetrandom runtime.vgetrandom`

### `modernc.org/libc`

Module: `modernc.org/libc@v1.72.3` (`h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=`), replacement `adapter`

Source set: `sha256:2fd5cdb4987b3319011b56c1244100a8479a51a65ad561357aaea93db4c9aa9b`

Go sources:

- `abi0_linux_amd64.go`: `sha256:da96495088e646b9342c585ab99fdb2c60ada2b48d5df9593a67c0c4c46948ef`
- `aliases.go`: `sha256:1069cf270f87de520e1386b1fea0d7a20841e2f5b4350899d91efc46a0adced3`
- `atomic.go`: `sha256:3f323e590b85f8ce07a27a65782f681f925d0edee1bfdb6acc2bfa2ab1837852`
- `atomic64.go`: `sha256:ca887733b51e4105bf63adf4fe2fe9196ecc2d1126380114514f07e2282d342f`
- `builtin.go`: `sha256:87f5e5fd6ee003638dc83066f564c0b8b6429e66f141472e36dcf4c5c8b99dde`
- `builtin64.go`: `sha256:632d7aadbe811324495a851919e11068759ca4030bb5156a59b121772ca2c4a8`
- `builtin_all.go`: `sha256:5b2c1479edaaba777d817dbccad1fb41ee26fed87c576f70a427cb3c505c8fea`
- `capi_linux_amd64.go`: `sha256:9523cd447280543d4bed2b696d94d05eaa160d421864aa1c7501902a7a269d09`
- `ccgo_linux_amd64.go`: `sha256:dfd35080dd5c6fe142b5fc7c6854b7f3e09e5976f3c0381a690a10825654c4d5`
- `etc_musl.go`: `sha256:1946682bcd3766f9a4efbf4c679a24a33402984a5a4e42dbfcbf39fdadb06227`
- `fsync.go`: `sha256:5760bc90f3df02bbbe2a24dd08365a403b8bbe9a01f6ee9ee345249f7f698b62`
- `gomad_linux.go`: `sha256:0bb1c8c9679070bd824ad53510742a6a1ee3177bd5b70162859c5d0ebfacf2c0`
- `int128.go`: `sha256:fa4821cd943874028ba6a953e0ee48c480759f98ea1ade6e09d1cd64a8dd7cb4`
- `libc_all.go`: `sha256:dcd4d1f818059ab5a5f9b430b3cacde4290a15907467ed4e396169d64b778c6c`
- `libc_linux.go`: `sha256:9d7a8c74e114d8399810961efb4b136b4c5d7071ea7cd032db66cc6d7b197c47`
- `libc_linux_statfs.go`: `sha256:776dc1ba81b32f3add9e03dc39992b0d4e432d9db5891a4b8d417c548d4bfcab`
- `libc_musl.go`: `sha256:ce9c26d451a4fb67a4cb3ec31d9b47f14e6731a6e38c89bab517f203b19f9ea8`
- `libc_musl_linux_amd64.go`: `sha256:f18c175291702686c3780ea5caf8d098d145f2ec958ba1179e0ced565f94b89a`
- `mem_musl.go`: `sha256:80f545e7bd8a4a232cadc372a43aabd63b653163d133efc1e1e62e6c5afe72c2`
- `nodmesg.go`: `sha256:18232e1a56ef0cc8f9d588d7392b3890e37820047f352e8f24889ebd78a16bde`
- `probes.go`: `sha256:7712b62c336e3e39d145c2824afe7271d989170873d552da9774c544fec1dc43`
- `pthread_musl.go`: `sha256:057f74259257283b41214787d9c9bdd3fd9fab8b23f58d4b4f6deca4a64f4f77`
- `rtl.go`: `sha256:57397f55cee2da100595028557cd1f72b894768b0da4a48f6fda0c2daad8765a`
- `stdatomic.go`: `sha256:d90c1342fc268f89020ebdb4f3886ee3572837d5cb30c02992c5153654614d6b`
- `straceoff.go`: `sha256:9b1d5deaa41eb28d23a89c0ac4fc06d827d3a46a57016559342e31bcd16a4b7f`
- `syscall_musl.go`: `sha256:277ec2c9d5343ecb894b94964d3a9f3c9b8193d5a6a5d033bf2a4b786a7ba321`
- `tls_linux_amd64.go`: `sha256:5af1b3dde3042af2830a2d3f5d98de29044e8bb4c69abffad8a0605fc5e1dadc`
- `watch.go`: `sha256:d4ba707bdd7dddc533659a1ab47c80caf0a87d5df2c6844ac8fadbc60d7c4a25`

Foreign sources:

- `assembly:abi0_linux_amd64.s`: `sha256:8fbd757afb4af0e1ced5b3158c385aee77f6d45f881587a6a304c5f416b8df44`
- `assembly:tls_linux_amd64.s`: `sha256:ee7326a13cb79331c1d7fead2ed2df03c8adff7f31037b99daa0e91d6e5b0317`

Requested facts:

- `foreign:assembly:abi0_linux_amd64.s`: **allow** — **security-sensitive**
- `foreign:assembly:tls_linux_amd64.s`: **allow** — **security-sensitive**
- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**
- `import:syscall`: **allow** — **security-sensitive**
- `linkname:gomad_linux.go`: **allow** — **security-sensitive**
  - source `sha256:0bb1c8c9679070bd824ad53510742a6a1ee3177bd5b70162859c5d0ebfacf2c0`
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

Module: `modernc.org/memory@v1.11.0` (`h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI=`), replacement `adapter`

Source set: `sha256:40ac8382ecbdbb2b46f418da2ce63a2cc7a188971a21c31116ed832ddca5f849`

Go sources:

- `memory.go`: `sha256:3f5cee8943da57ffd3db70129f87abdf45c86e5299b81d09504ec2838dc31909`
- `memory64.go`: `sha256:1b91a327f3b95d6fc3f80f090175a9c5bc033083a2c675b7a56f009af8fa4813`
- `mmap_unix.go`: `sha256:c8a86dca80f526b39f0a855f59552d1085a352fb7fef47b86da827b854ab88ad`
- `nocounters.go`: `sha256:070021a593fc3c28988ad04bee02d83547ff56a9665e7703a80366392e30c18c`
- `trace_disabled.go`: `sha256:ecd151b29853826767c9fd0e2c9b48b4acd08bd211ce558016399afae693b950`

Requested facts:

- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**
- `linkname:mmap_unix.go`: **allow** — **security-sensitive**
  - source `sha256:c8a86dca80f526b39f0a855f59552d1085a352fb7fef47b86da827b854ab88ad`
  - directive `gomadMemoryEnabled internal/gomadio.Enabled`
  - directive `gomadMemoryMap internal/gomadio.AnonymousMap`
  - directive `gomadMemoryUnmap internal/gomadio.AnonymousUnmap`

### `modernc.org/sqlite`

Module: `modernc.org/sqlite@v1.51.0` (`h1:aH/MMSoayAIhozZ7uJbVTT9QO/VhzBf0J9tymmmuC/U=`), replacement `none`

Source set: `sha256:a1025b9b4c8e13e79221d7986f206afeae7571cae7b50214e8265f23b4296c44`

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

