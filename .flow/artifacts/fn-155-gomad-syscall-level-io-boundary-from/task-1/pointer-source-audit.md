# fn-155.1 pointer fixture source audit

One Important source finding remains in command cancellation. No Critical or Minor finding was established. The pointer instrumentation and relocation predicates align with the accepted design within this source-only assessment. This report supplies no formal implementation-review verdict, SHIP decision, native pass, or task completion.

## Scope and frozen inputs

Primary is `/Users/stephan/Workspace/skunkworks/gomad/temporal`, dispatched at `b24085ab55572630dd2ffd029c0df13663cde2d7`. Candidate `C` is primary's `.worktrees/fn-155-gomad-syscall-level-io-boundary-from`; inspected HEAD remained `c8b811d5344fb85e347b6db296998dd0feedc4ab`. Assessment covers the six actual pointer additions and the POINTER Makefile addition relative to that HEAD. Other active fixture/classification work is outside this report.

Fixture directory `F` is `C/tools/gomad3/toolchain/runtime/testdata/vfdpointer/`. Full files were read; all six SHA-256 values matched the dispatch both before and after inspection.

| Input under F | Consumed SHA-256 |
| --- | --- |
| `main.go` | `bd475d762971909680deace6844b639465f9c10a611676216e735da4a5ec444f` |
| `main_test.go` | `f9754b5972d53a2b30405dabd5dee54021cb469b5ffeadfc4eb1f7d6ef622197` |
| `growth.go.txt` | `73ef41aed3064571b8a99810ca893f604cea83641f6897c63cee43a1c5e8118b` |
| `fixture_test.go.txt` | `c0f259e57d1bb5a04896d7f8c423aab2a5b214bb8928811a97c735c6000d326b` |
| `linux_export_test.go.txt` | `0f1cd244467a7b688c1c18eea3b461ecfca6f1b17bc740b9c2106b4caaf26fb8` |
| `darwin_export_test.go.txt` | `5a33c453fbc2d6794a1ee3154779ae2461bc9bbcba0e58cd247d01cf7adb43bc` |

`C/tools/gomad3/Makefile` remained `eb98c324c946754a048aad7ee34968f5d572b742d8e30c4f9838cbebd17cc27c`. Its four POINTER recipe lines 140-143 independently hash to `a0be75c101501f0e06e46ce5a6d71ae1b64566379d714908580dca285686d1e8`; later disjoint native additions do not invalidate that section binding.

Required context was read completely. Consumed primary design/evidence/integration hashes are respectively `ba19976cd9c764d089af73b84ee0f6c2e2fe6d6ace7f30020fdee0a481c500f9`, `bf2a545e6dcb53894668375bfccfa1f73deb71bc14df6eb3175dc3bdf98d9eb6`, and `b631209c9670c3f9e3d923047716815aa32b19ddcd8a085798283eeb869e8bb1` for [pointer-fixture-design.md](pointer-fixture-design.md), [pointer-source-evidence.md](pointer-source-evidence.md), and [root-integration.md](root-integration.md).

## Important finding - terminate compilation descendants on cancellation

`F/main.go:195-202` constructs `exec.CommandContext` and waits only for that command. The ten-minute context at line 250 also governs the `go test -c` invocation at line 320. When the deadline expires during compilation, the default cancellation kills the Go driver alone. Its compiler/linker descendants can continue using caches, temporary files, and inherited log descriptors after `runCommand` returns and the caller considers the shared lane released. The receipt can therefore precede the end of the owned command tree.

This follows directly from the consumed pinned Go 1.27.1 source. `src/os/exec/exec.go:492-502` sets `Cancel` to `cmd.Process.Kill` with `WaitDelay` unset; `src/os/exec.go:334-336` explicitly documents that Kill affects only the process itself, excluding processes it started. Their SHA-256 values are `3ec96eeef97a375aa5e39663961f8adf3177db9449dacd3c407fd258bc9a2498` and `811e5ad1e525fa0b4ab9109ee003f493df79e30e2b17ea29d643686ac9c6db2b`, under `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/`. No internet lookup or behavioral execution supplied this conclusion.

Give each command an isolated supported-host process group or equivalent scoped cancellation, terminate its descendants on timeout, and retain a behavioral regression with an observed descendant. Preserve argv boundaries, timeout/error identity, and checked cleanup. A process-wide kill target would exceed this fixture's scope. The current command-failure host regression checks status and output only; it cannot cover descendant lifetime.

## Pointer source assessment

`main.go:22-56` inserts exactly four calls immediately after actual helper signatures and reverses them byte-for-byte. `prepareOverlay:87-148` requires selected/repository helper equality, refuses synthetic path collisions, and writes private replacements without altering source files. The retained instrumented helper confirms growth precedes writev slice/Base reads, paired-output backend/copy operations, and getsockopt's first length read. The production patch, descriptor algorithm, and callback surface receive no edit.

`growth.go.txt:26-57` passes only `p *Iovec` across recursion. No independent typed Base pointer crosses growth. The deepest frame reads the two relocated Base fields; saved observations are scalar uintptr values and are never dereferenced. Pair/option recursion keeps both typed roots together. No recursion stores caller pointers in global/container state or invokes a callback. Each 64-level path retains a checked 1 KiB local frame.

`fixture_test.go.txt:120-132,150-168,188-228,264-279` requires one observed call, helper-entry/caller-before agreement, each nonzero object address changing, deepest/current address agreement, expected output, and intact sentinels. Incidental movement before the entry record fails the success predicate; a heap-only tested object fails the address-change predicate. The no-growth case at 154-157 requires unchanged entry/deepest/current observations and rejects their relocation predicate. This is a passing negative-control fixture, not a native movement result.

Writev covers full and cross-vector partial bytes; address cases cover accept, full/truncated local/peer names and both outputs; getsockopt covers SO_TYPE, distinctive SO_ERROR and unchanged short-length failure. Null/mismatched accept cases make no relocation claim. Linux exports traverse all four generic variants and both six-argument option routes; Darwin exports include raw and X variants through the actual libc function identifiers. Exact x/sys execution remains a separate open gate. Existing uninstrumented read/write, bounds and refusal tests remain in the preceding syscall package gate.

The fresh child registers its backend, enables the switch, checks descriptor close status before disabling, preserves the installed Ready callback, and resets scalar instrumentation state. Cleanup registration is ordered correctly, including successful accepted descriptors before later assertions. Child disposal supplies backend isolation; no prior-backend restoration is claimed. Fixtures run serially in fresh goroutines/processes. Native admission checks host/selected target, pinned version, compiler guard support and source equality; build settings remove seed, child-seed and Go tuning inputs and select nogreenteagc. Commands use argv slices; selected-test output verification rejects empty selections and skips. The finding above limits command-tree wall isolation.

## Retained evidence and limits

| Consumed C/.flow/tmp input | SHA-256 |
| --- | --- |
| `fn155-pointer-overlay-1/receipt.json` | `9cb3eeaf167895fcf9b462df6ceeeb77bcda7ce52479121e63bbe1ccc1030b9a` |
| `fn155-pointer-instrumented-linux-1.log` | `bc767cad91de69e6c854b39df4743cf0ca533a983ea00bf46fd11e0ae5e639b1` |
| `fn155-pointer-instrumented-darwin-1.log` | `d90115df9b09dfda66f3d8f71ebe0696ba1cd3fca2f4218a5457a7d8467703af` |
| `fn155-pointer-runner-final.log` | `50b613a49bca1d56a47b0f2c494f0a648999636b5199a7c5702ff6fb537745d1` |
| `fn155-pointer-selection-red.log` | `ae1d7d699403715430087459ac2103db8a59d1bd1596bbdbdd108209c3ed5c06` |
| `fn155-pointer-native-refusal.log` | `0567e26a951afa1e98de4656d8203db27c343c176f7eac87ead487cdd5f45a89` |
| `fn155-pointer-direct-lint-final.log` | `e9a933ef482eac9be9f779f07670db7a1484f2b55e4e74405aa951ebe9d4b004` |

Selective diagnostics show helper/shim roots live at growth calls and recursion entry, recursion local frames of at least 0x448 bytes, caller stack storage for both payloads and guarded outputs, and the nested-vector stack object on both source sets. These new instrumented compiles used stock compiler `3f69d2da22ae2662716a74e50e64072f077d72af0f3dfa53a3e19de2729e6ed0`. Receipt original/instrumented hashes are `c18bba832662583aeb2e2d5c0dd2c53e86f0a2e744b20633064d2c0525398d1b` / `3f839670b68fe32b69ca7b53ebdfed2178b8e051082d203e720b88b3780711b8`. Original-source maps remain the separate evidence report's stock compile observations. Neither set establishes native relocation or rebuilt Gomad compiler behavior.

Retained host runner evidence has five passing tests. The selection regression retains its initial failure and final pass. Linux ARM64 refusal exits 1 before native execution. Direct lint reports zero after nine configured exclusions and does not establish canonical/template lint; its classification repair remains independently owned. The source runner receipt explicitly says source-only.

Required ordinary/instrumented native execution with the rebuilt pinned compiler, exact x/sys forwarding, disabled-path preservation, full Quick, zero host sockets, virtual deadlines/quiescence, race/fork and actual readiness/reuse proof remain open. No native owner, CI, PR, push or Flow state was activated by this audit.

Requested reviewer route was gpt-6.1-sol/high, in the same GPT family as the writer. Dispatch records `tier=session`, `jev-unavailable(no_key)`, and no actual-model telemetry. This fresh-context reviewer ran no Go/compiler/lint/build/generator/native command, bridge or agent. Only this primary report was written; product files, Git index/HEAD and Flow lifecycle were read-only. Prose follows `/home/agent/.codex/docs/flow-next/prose.md`.
