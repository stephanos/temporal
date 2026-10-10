# fn-155.1 native fixture source audit

The eight native fixture cases add real syscall-to-model-to-runtime coverage in source. The initial review found one Important UDP reporting assertion gap. The targeted source recheck below resolves that gap. No remaining Critical, Important or Minor defect was established within this bounded review.

This independent source assessment supplies no formal implementation-review, SHIP, Done, merge, native execution, qualification, or determinism verdict. Primary HEAD supplied by root is `685d70a4204b5d587a4eebf425f2323392a0a1a8`. Candidate HEAD is `c8b811d5344fb85e347b6db296998dd0feedc4ab`, with uncommitted additions in `.worktrees/fn-155-gomad-syscall-level-io-boundary-from`.

## Strengths

`fixture_test.go.txt:22-38` checks the direct-seed initial clock, inactive legacy network and default-off descriptor switch. Normal `net` linkage initializes the existing standalone backend. The fixture keeps the production Ready callback and backend installed. The earlier [syscall audit](syscall-source-audit.md) and [readiness audit](readiness-source-audit.md) retain production algorithm assessment; this review inspected their narrow consumers rather than reopening those algorithms.

The runtime probe at `runtime_probe.go.txt:19-39` locks `pollcache.lock` and returns scalar registration, token, physical pollDesc address, fdseq, semaphore and actual waiting-G state. `nativeParked` requires a committed reader/writer and unchanged host waiter count. Deadline cases invoke production `internal/poll.FD` operations, block the controller on the result channel and require `os.ErrDeadlineExceeded` plus exactly seven seconds of virtual advancement. The probe does not synthesize parking or deadline completion.

Cleanup callbacks check close errors and run before the earlier switch-disable callback. Explicit poll FD close sets Sysfd to -1 through production `internal/poll/fd_unix.go:77-85`, so those callbacks avoid double close. These templates do not swap callbacks and therefore require no callback restoration of their own.

## Issues

### Critical

None established within the reviewed fixture axis.

### Historical Important I1. UDP reporting has no assertion

Location is `tools/gomad3/toolchain/runtime/testdata/vfdnative/fixture_test.go.txt:148-155`. Task .1 requires UDP to be refused and reported. This case calls public `syscall.Socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP)` and checks EOPNOTSUPP, but a regression that removes the reporting call still passes.

The governing production chain is overlay `syscall/gomad_vfd_unix.go:90-91` through `gomadVirtualRefuse` at `:73-74`, leaf `internal/gomadvfd/descriptor.go:393-400`, and actual backend `internal/gomadio/descriptor_backend.go:492-497`. The backend records `net.syscall.refused` with operation `socket.type-or-protocol`. Add a private-overlay observer at that actual reporting seam and assert the operation/count while retaining the installed backend and Ready callback. Check and restore any temporary observer state. A replacement stub backend would lose this consumer proof.

Direct-seeded `internal/gomadio/transcript.go:19-22` returns before recording because the I/O profile is inactive. Such an observer proves the reporting call only. Persistent UDP transcript evidence remains a separate open acceptance requirement with its existing scope admission rules.

### Minor

None established within the reviewed fixture axis.

## Exact executable coverage in source

All fixture line references below refer to `fixture_test.go.txt`. None of these cases was executed natively during this review.

| Case | Source observation |
| --- | --- |
| TCP, :75 | Normal net ListenTCP/DialTCP/AcceptTCP, live virtual endpoint/registry witnesses, public syscall.Write through the manual guard, partial 65536-byte cross-vector writev with owned-copy check, reverse bytes, both half-close EOFs, immediate refused connect and UDP errno. |
| ReadyWake, :329 | Real accept/read/backpressured-write Gs commit before the controller causes production connect/write/dequeue transitions. Duplicate runtime hints exercise retry/repark; dequeue checks all 65 bytes in FIFO order and zero elapsed virtual time. |
| AcceptDeadline, :285 | Empty listener accept commits, times out and preserves host waiter count. |
| ReadDeadline, :304 | Empty connected read commits and times out. |
| WriteDeadline, :403 | A full 64-chunk receiver queue returns EAGAIN; internal/poll Write commits and times out. |
| SimultaneousDeadlines, :417 | Read and backpressured write commit on the same descriptor and both return the deadline error at the same logical instant. |
| CloseParkedRead, :433 | Wrapper FD.Close evicts a real reader, waits for destruction, returns ErrNetClosing to the read and clears registration without logical-time advancement. |
| PollDescReuse, :455 | A real closed pollDesc is observed at the same address with a new token and fdseq. Old-FD/old-token and new-FD/old-token hints leave the new committed reader unchanged until its deadline. Numeric virtual FDs never reuse. |

The Linux helper calls generic Syscall(SYS_WRITEV); the Darwin helper calls pinned same-package writev. Their vector input is nonempty in this fixture. This test checks data/partial progress and copying, while the separate pointer fixtures own moving-stack proof.

The narrowly consumed runner `main.go:70-90` overlays exactly these four templates, without host-entry instrumentation. Its :115-128 environment strips Gomad controls; :352-359 launches each exact case in a fresh direct-seeded child with the logical test timeout disabled. Its :131-150 rejects missing/duplicate selection and a skipped reuse case. These observations exclude runner cancellation correctness, which another reviewer owns.

## Remaining evidence and coverage bounds

Endpoint ownership and unchanged host waiter counts prove only the observed descriptor/model paths. There is no positive-controlled host socket-entry observer in this overlay, and no native OS audit. Whole-process zero host sockets remains open as the fixture and runner explicitly report. No new production observer hook is implied by this review.

The eight cases do not exercise queued-connect completion/refusal before poll registration or after a committed WaitWrite. TCP uses immediate successful and immediate refused connect; nativeFD initializes poll before Connect. Earlier portable backend/mailbox regressions and the retained readiness audit remain their source evidence. Exact pdWait/commit-window, raw-close waiter, reset, close/deadline race, no-future-event deadlock and genuine off-mode host-pipe waiter coverage are also absent from these cases. This inventory adds no synthetic requirement or new production defect.

Supported-native execution remains necessary for every observed park, wake, deadline, cleanup and reuse outcome. Task .1 also retains its prescribed toolchain build/validation, disabled tiers, Linux target compile, canonical lint, both-source-set and generated gates unless root separately binds their evidence. Linux ARM64 supplies no full Quick or native pass. Existing fn-128/fn-149 deferrals receive no automatic transfer or revival. Selection/profile identity stays with .2; compiler/closure capability admission stays with .8. Guard-off success would cover the manual Write guard only.

The parent-provided stock-source diagnostic logs contain the fixture compilation symbols. They are compilation evidence only, with no independently verified exit receipt here. Their SHA-256 values are `4858fd25ff705c1fc1b9cdb1d8a4ba78b5e491feee7d62553aa0d734f55ed7ed` for `.flow/tmp/fn155-native-linux-1.log` and `6ecbe392b6cc5620e689813a169fba35f37169df0868cdb3260cca6c465a6f91` for `.flow/tmp/fn155-native-darwin-1.log`.

## Frozen identity and assessment

Opening and closing SHA-256 checks matched all supplied template and Makefile identities. Paths in this table start at candidate `tools/gomad3/`.

| Input | SHA-256 |
| --- | --- |
| toolchain/runtime/testdata/vfdnative/fixture_test.go.txt | `f30b1c0cdb13d366fcaba4a62bfca425aa0ff6286c0981aaf303f2b339f57992` |
| toolchain/runtime/testdata/vfdnative/runtime_probe.go.txt | `d0da8381b4a9b900f6d89ee855437369cfe2b3bfe50ef16a5be8736e8b157fcd` |
| toolchain/runtime/testdata/vfdnative/linux_probe.go.txt | `67bf6fff8014b7ac7afc545f748440b032e71038bc69408905441dcc400c2cbc` |
| toolchain/runtime/testdata/vfdnative/darwin_probe.go.txt | `ecc374239e808d6d23379366b1a110a1e66f906c8fa58fa008a1787726e84957` |
| Makefile | `f4953730e0aec8e86dd9f872533cfaeef50cfd0677fbcb32899479180e6fb445` |
| Makefile lines 144-147, including newlines | `2e075c9350e3fb1ace06fe49fad16bbf9f976cc4d308865745e9db9ab0cbaefd` |

Supporting runner SHA-256 is `ab9a5b905d8f96b80b3b360ccd1410a3f53766f1333f347d227494b202a45b5b`. Review used the requesting-code-review template and `/home/agent/.codex/docs/flow-next/prose.md`; required project guides, task, [gap inventory](native-fixture-gaps.md), [safety design](safety-design.md) and retained audits were read. Requested reviewer is gpt-6.1-sol/high, with session fallback `jev-unavailable(no_key)` and unavailable actual-model telemetry; writer and reviewer belong to the GPT family.

The initial source assessment required the targeted I1 fix and recheck. Root owns remaining acceptance and lifecycle decisions. This reviewer ran only read-only source/search/hash commands and wrote this report; no Go, compiler, build, test, lint, generator, bridge, agent dispatch, Git/Flow mutation, native owner revival, CI, push or PR action ran.

## Targeted I1 source recheck

I1 is addressed in the subsequent frozen source. Candidate HEAD remains `c8b811d5344fb85e347b6db296998dd0feedc4ab`. The initial source descriptions, findings, line references and hashes above remain historical evidence. This recheck covers only the admitted reporting observer and overlay preparation changes; runner cancellation and unchanged eight-case mechanisms retain their separate reviews.

Current `main.go:26-36` requires one exact production record-call anchor, rejects existing instrumentation, inserts one `gomadNativeObserveRefuse(operation)` immediately before that call and checks byte-identical reversal. Inspection of actual `internal/gomadio/descriptor_backend.go:492-497` confirms this anchor belongs to standaloneDescriptorBackend.Refuse. The unchanged syscall-to-leaf-to-actual-backend chain above still supplies the call. `main.go:69-100` compares the selected backend source with candidate source before producing the private replacement and original snapshot; :101-121 adds the fifth synthetic source without modifying toolchain files. Original candidate and materialized backend hashes both remain `30d5b525a6df65b94e21c7495346d2efe68d618f8527a1505dcaf4bb171c3dac`.

Current `fixture_test.go.txt:152-178` rejects zero reports, wrong operation and duplicate reports, installs the observer, invokes the public UDP socket call and requires one `socket.type-or-protocol` report. The actual backend and Ready callback remain installed. `gomadio_probe.go.txt:17-30` rejects nil or preexisting observers, and the fixture's cleanup checks restoration to inactive state. The observer receives a scalar operation string synchronously; its count/string closure remains live only through this serial exact child and cleanup. No concurrent fixture or outstanding model operation accesses that observer in the admitted TCP case. This scoped seam requires no general concurrent observer contract and retains no caller I/O buffer.

The host regression consumes the actual backend source and rejects altered or duplicate record anchors, with private replacement/source agreement checks at `main_test.go:177-249`. Parent-retained RED failed because the observer was absent; GREEN ran seven host tests successfully in 0.670s. These checks verify overlay construction. Parent-reported full syscall stock-source diagnostic compiles returned 0 for Linux and Darwin; inspected logs include the new reporting predicate. None of these results executes the native UDP reporting chain. Native reporting-call execution, persistent UDP transcript, whole-process zero sockets and other acceptance gates above remain open.

All seven consumed source files and four logs retained identical opening/closing hashes in this recheck. The runtime/Linux/Darwin probes retain their initial hashes above. Changed files below start at `tools/gomad3/toolchain/runtime/testdata/vfdnative/`; logs start at candidate `.flow/tmp/`.

| Recheck input | SHA-256 |
| --- | --- |
| main.go | `6ea2e0518a3d15bdc7b17b07f3c6c704ba2b73ebe28a552a45d2b6d48d91d5cf` |
| main_test.go | `1aa2fd18e744d35e825652fa065ba54e639a6d25b05c5728f55697b3aac55d1b` |
| fixture_test.go.txt | `9dd8ff823aee1777bf4ebabb5030196ff63ae0094d3e325e1a7e14dbe1e96393` |
| gomadio_probe.go.txt | `5c3dd3503e74b1cd324f7fdc1af3c6f07a0c503c2d5fbb3d755c812cfc335174` |
| fn155-native-reporting-red.log | `363a0536c31b4e8098558f4eed448379cdc7f508cefe35d7f1e9c09f2b2fb625` |
| fn155-native-reporting-green-2.log | `f2c4bffe7071a9df8b5486c9620212712828a697d1a07efbe3cba0ec3bc75b36` |
| fn155-native-linux-reporting-1.log | `89a9aa7ba4edcb8562f5b74939dc4fe120791802b506feb8971fdab217f4f191` |
| fn155-native-darwin-reporting-1.log | `6df88b3d6f68a68a4bbd8d53050dab8cd04bf3e03f81b3e07487e80e541395dd` |

The targeted source assessment found no remaining Critical, Important or Minor defect. It closes the I1 source gap only and changes no acceptance, native qualification, model-routing, completion or mutation boundary recorded above.
