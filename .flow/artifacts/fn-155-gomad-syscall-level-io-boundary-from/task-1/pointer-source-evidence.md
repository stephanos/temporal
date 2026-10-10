# fn-155.1 compiler pointer evidence

Both supported target source sets compile with explicit pointer maps for the typed handoff, nested iovec array and paired outputs. This evidence supports the source design. No target binary ran, no rebuilt Gomad compiler was used, and first-platform acceptance remains open. No new safety defect was established by this bounded check.

## Candidate and tool binding

Primary is `/Users/stephan/Workspace/skunkworks/gomad/temporal`, dispatched at `cd909a2d165a7aa23c13cd02a0a5b3420a36b01b`. Candidate `C` is its `.worktrees/fn-155-gomad-syscall-level-io-boundary-from`, HEAD `c8b811d5344fb85e347b6db296998dd0feedc4ab`, source commit `a28481d2eeffe7561b5a277770c88117d0ab24b4`. Candidate tracked status was clean before and after inspection. Root's [integration checkpoint](root-integration.md) binds all 1,075 tracked Gomad sources to primary through listing digest `a985e9a318fcbeb1825a91c35c0255630a00464b1ee2bbaaa14294e7d9dfc249`; this investigation reuses that binding.

Actual host is linux/arm64. `GO` is `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`, SHA-256 `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`. Both its stock compiler and `C/.flow/tmp/patched-goroot/pkg/tool/linux_arm64/compile` hash to `3f69d2da22ae2662716a74e50e64072f077d72af0f3dfa53a3e19de2729e6ed0`. The materialized GOROOT contains patched sources but uses that stock compiler. `GOEXPERIMENT` was empty, so these builds also do not certify the Runner's `nogreenteagc` profile.

Rechecked patch SHA-256 is `2be96ca5c7d0b108e5ecc9278fa6ad681c83cc71265f20af8e5cf46dab6813dc`. Materialized `syscall/gomad_vfd_unix.go` and `gomad_vfd_stack_test.go` match the tracked sources, hashes `c18bba832662583aeb2e2d5c0dd2c53e86f0a2e744b20633064d2c0525398d1b` and `b6f9bd43414394bb9c0506cc368846631420595c4b94a8fa2f86df940f760934`. Platform helper/test hashes match the [final syscall audit](syscall-source-audit.md). Materialized generic Linux/Darwin files hash to `5d18ca51d69af7fcc2dfda8c6098ca10a56ad09bd3c41acb8cf6280016e79987` / `1bce43b35cc1077b3c9238f210dae80887af66c80db5d5705b42dba5540b5104`; generated wrapper files hash to `857e97b4dd0e9067396fc560b30b519903c5c3631fda6ec05fbfda328de8582f` / `706ca52b6c8ca8ee73dd652215302a4fd6189611aa9e135120d846ae5e956cfa`.

## Exact compiler evidence

`-live` and emitted `FUNCDATA` agree on both targets. The map format is read from the same materialized `runtime/symtab.go:1323` and `runtime/stack.go:1372`; the first two little-endian words give bitmap count and width.

| Seam | Emitted evidence | Supported conclusion |
| --- | --- | --- |
| Generic prefix | Linux public Syscall/RawSyscall variants, Darwin internal syscall/syscall6/X/raw variants, classifiers and `gomadGenericOperation` emit `STEXT nosplit`; decoder sizes are 1370 bytes Linux and 1664 Darwin | These compiled prefixes have no entry stack-growth check before typed helper entry. Darwin public trap assembly remains a separate route. |
| Accept and name outputs | `length raw` live at entry and at Accept/Remote/Local calls; argument map `02 00 00 00 03 00 00 00 06 00` | Both pointer argument words 1 and 2 are marked together. Accept also retains both at Linux creation-flags call. |
| Getsockopt outputs | `length p` live at entry and Remote call; argument map `02 00 00 00 05 00 00 00 18 00` | Pointer words 3 and 4 are marked together. |
| Writev entry | `p` live at entry; argument map `04 00 00 00 02 00 00 00 02 00 00 00` | The array pointer has an entry pointer bit. |
| Writev allocation/copy | Local map `04 00 00 00 02 00 00 00 00 00 02 03`; `PCDATA $1,$2` at makeslice, then index 3 at memmove | The typed vectors backing pointer survives allocation; vectors and owned-buffer pointers survive copying. Linux uses `.autotmp_56`/`_58`, Darwin `_57`/`_59`. |
| Caller nested array | `stack object vectors [2]syscall.Iovec`; stack-object record size 32, pointer bytes 24, `runtime.gcbits.0500000000000000` | Caller stack metadata marks both nested Base fields at words 0 and 2. This is necessary evidence beyond the helper's single array-pointer bit. |

Linux assembly stores the typed vectors pointer at `SP+72` before makeslice and reloads it for copying. The caller constructs vectors at `SP+96` with Bases pointing to local `a` at `SP+56` and `b` at `SP+52`. Both target diagnostics report the helper's `p`/`vectors` arguments do not escape. The read/write moving-stack fixture also retains local data in the caller frame; Linux emits `data+40(SP)` and passes `data+41(SP)`, rather than allocating the 64-byte buffer on the heap. Backend Read/Write and recursive growth diagnostics retain `p` live. These facts establish the compiled fixture's stack eligibility, without observing relocation.

The dispatcher itself has scalar argument maps. Safety depends on the jointly typed callee maps and the caller's live typed objects; the scalar uintptr slots are not relocation roots. The inspected emitted calls agree with the source handoff, which never dereferences those original scalar addresses after entering the helper.

## Commands and retained artifacts

Every shell used `env -u BASH_ENV bash -c` with explicit `cd C`, `workdir=C`, `login:false`. Scratch wrapper `C/.flow/tmp/fn155-pointer-env.sh`, SHA-256 `5eef7c2f0eaaca5faaef9f02f3279c10df36cf4dc65489d3242524f7abd6ffa3`, sets `GOROOT=C/.flow/tmp/patched-goroot`, `SANDBOX_START_DIR=C`, `GOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/go-cache`, both temp variables to that admission directory's `tmp`, `GOMODCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc`, `GOPROXY=file://$GOMODCACHE/cache/download`, `GOSUMDB=off GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS='' TZ=UTC CGO_ENABLED=0`, and execs `GO`.

The following exact commands ran from `C`; each exited 0 unless stated otherwise. Output paths are below `C/.flow/tmp/`.

```sh
bash .flow/tmp/fn155-pointer-env.sh version -m .flow/tmp/syscall-linux-amd64-final.test
bash .flow/tmp/fn155-pointer-env.sh version -m .flow/tmp/syscall-darwin-arm64-final.test
bash .flow/tmp/fn155-pointer-env.sh tool nm .flow/tmp/syscall-linux-amd64-final.test > .flow/tmp/fn155-pointer-linux-nm.txt
GOOS=linux GOARCH=amd64 bash .flow/tmp/fn155-pointer-env.sh test -tags test_dep -c -gcflags="syscall...=-S -m=2 -live" -o .flow/tmp/fn155-pointer-linux-diagnostics.test syscall > .flow/tmp/fn155-pointer-linux-compiler.log 2>&1
GOOS=darwin GOARCH=arm64 bash .flow/tmp/fn155-pointer-env.sh test -tags test_dep -c -gcflags="syscall...=-S -m=2 -live" -o .flow/tmp/fn155-pointer-darwin-diagnostics.test syscall > .flow/tmp/fn155-pointer-darwin-compiler.log 2>&1
bash .flow/tmp/fn155-pointer-env.sh tool objdump -s "syscall.gomad(Generic|Virtual(Accept|Name|Getsockopt))" .flow/tmp/syscall-linux-amd64-final.test > .flow/tmp/fn155-pointer-linux-objdump.txt
bash .flow/tmp/fn155-pointer-env.sh tool objdump -s "syscall.gomad(Generic|Virtual(Accept|Name|Getsockopt))" .flow/tmp/syscall-darwin-arm64-final.test > .flow/tmp/fn155-pointer-darwin-objdump.txt
bash .flow/tmp/fn155-pointer-env.sh tool objdump -s "syscall.gomad(Generic|Virtual(Accept|Name|Getsockopt))" .flow/tmp/fn155-pointer-linux-diagnostics.test > .flow/tmp/fn155-pointer-linux-diagnostic-objdump.txt
bash .flow/tmp/fn155-pointer-env.sh tool objdump -s "syscall.gomad(Generic|Virtual(Accept|Name|Getsockopt))" .flow/tmp/fn155-pointer-darwin-diagnostics.test > .flow/tmp/fn155-pointer-darwin-diagnostic-objdump.txt
```

Both retained binaries identify go1.27.1, gc, test_dep and the expected target. Original Linux/Darwin binary hashes are `9f3a0896d67795e35f118178190841fa6cd9eb34faf1505e24ad85cdfffb18e2` / `fefa3752b12efd369fe9957fbab8a47c734e784ad53e4521dd6c077889b1560b`. Diagnostic binary hashes are `1626201c0a82f7ab428f4db44519dbe133a478a29012a94324264e48f1928106` / `2bf8cc7eadcf54ef83a212de827ee87da73d44484830b591d5549da1f9d94869`. Compiler-log hashes are `53279d78e776be56d3f4212c0cc8fc5c855af33cb881774b96f9f705d228878a` / `261fd02c5e6660c4050e86144adb3e36cc08b29a87ee6aaf227b995cffc8e2a3`.

`cmp .flow/tmp/fn155-pointer-linux-objdump.txt .flow/tmp/fn155-pointer-linux-diagnostic-objdump.txt` exited 1. Inspected differences include linked data-address displacements. No byte-equivalence claim follows. The maps above bind the new diagnostic compiles; original-artifact objdump establishes their retained generic/helper call structure, not recovered original stack maps. An initial `ls` probe exited 2 because materialized `bin/` is absent; subsequent tool calls used the explicit stock executable and materialized tool directory successfully.

## Remaining proof

Current writev growth occurs in backend Write after the owned copy. Correct stack eligibility and pointer bits do not witness relocation of the array and both Base targets before their first dereference/copy. A native regression still needs that precise observation, updated nested addresses and intact bytes/sentinels. Paired sockaddr/length and getsockopt value/length require native relocation observations for both objects. Exact x/sys forwarding and Darwin X variants also retain their runtime gates. Rebuilt Gomad compiler/guard behavior, native stack copying, zero host sockets, deadlines/quiescence and all original acceptance remain unverified.

No product, Git or Flow state changed. Only scratch diagnostic files and this primary report were written. Go/tool/compiler lane was released to root at 07:40 UTC on 2026-10-10; compiler sessions 16723 and 67689 ended with exit 0, and no owned command remains live. Requested research route was gpt-6-astra/high; root reported `jev-unavailable(no_key)` session fallback and no actual-model telemetry. Prose follows `/home/agent/.codex/docs/flow-next/prose.md`.
