# Gomad patch size investigation

Assessment date: 2026-09-30.

The strongest proven reduction is to use one context line. It cuts bytes by 24.3% while producing identical source. To reduce the source edits that must be maintained against upstream Go, move Gomad-only implementation into the existing overlay. A measured combined candidate cuts bytes by 13.7% with ordinary context, or 36.9% with one context line. Its behavioral qualification remains incomplete.

## Measured reduction options

The patch can be reduced by changing its representation, moving Gomad implementation into the existing source overlay, or combining both. Preserve existing comments in the source and move them with extracted code.

| Variant | Bytes | Lines | Byte reduction | Evidence |
| --- | ---: | ---: | ---: | --- |
| Checked-in patch | 32,275 | 998 | 0% | Baseline |
| Two context lines (`-U2`) | 28,637 | 858 | 11.3% | Identical materialized source |
| One context line (`-U1`) | 24,423 | 710 | 24.3% | Identical materialized source; governed validator accepts it |
| Zero context lines (`-U0`) | 20,716 | 555 | 35.8% | Identical materialized source; fewer anchors for review and application |
| Scheduler helpers moved to overlay, `-U3` | 29,216 | 903 | 9.5% | Scratch build and limited smoke checks |
| Scheduler helpers plus crypto initialization and syscall declarations, `-U3` | 27,845 | 849 | 13.7% | Measured composition of independently checked prototypes |
| Same combined source candidate, `-U1` | 20,352 | 581 | 36.9% | Measured diff; requires integrated qualification |

All four Git diff algorithms tested (`myers`, `minimal`, `patience`, `histogram`) produced the same bytes and line counts at each context size. Switching to `--minimal` alone saves nothing.

The context-only variants applied to the cached, checksum-verified Go 1.27.1 source with the builder's `patch --dry-run --batch -V none -p1 -F 0` and application commands. Each resulting file matched the current patch's output byte-for-byte across all 20 patched files. This proves source equivalence for the pinned archive on the local macOS patch implementation. Linux application was not tested.

Use `-U1` if stored bytes are the main concern. Keep three context lines if the main concern is reviewing and porting the source changes. Context reduction changes no source comments and removes no functionality. Update the canonical regeneration command in [patch_regenerate.go](../../../tools/gomad3/toolchain/patch_regenerate.go), which currently invokes `git diff` with its default context. Otherwise regeneration restores the larger representation.

## Scheduler extraction

`runtime/proc.go` contributes 15,145 bytes and 467 patch lines, or 46.9% of the total bytes. It contains 205 added and 20 deleted lines. The complete patch contains 341 added and 54 deleted lines. These counts make `proc.go` the first place to reduce the amount of implementation carried in upstream files.

The scratch candidate moved three implementations into the existing [runtime overlay](../../../tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go).

| Extraction | Interface retained in `proc.go` | Constraint |
| --- | --- | --- |
| `gomadSimulationTimeQuiescenceChanged` | Existing call | Move the complete function verbatim. This alone saves 1,059 bytes and 35 patch lines. |
| Quiescence and time-advance selection in `checkdead` | `wakeTimer, waiting := gomadCheckDeadTime()` followed by the existing return and timer-wake path | Enter with `sched.lock` held; preserve every unlock/relock, transport response, arrival check, deadline comparison, and fatal path. Normal returns retain the lock. |
| Seeded idle-P syscall resumption in `exitsyscallNoP` | `gomadResumeSyscall(gp, pp)` inside the existing activation guard | Preserve queue admission before scheduling, `lockedm` handling, P acquisition, and calls that never return. |

Together these extractions save 3,059 bytes and 95 patch lines with three context lines. They put simulation arbitration behind a small interface without copying the entire upstream scheduler into the overlay. The scratch candidate retains the existing stock M/P timer-wake machinery in `proc.go`.

The candidate compiles and links on `darwin/arm64` and cross-compiles and links for `linux/amd64`. For each scheduler prototype, a small timer/goroutine program matched the unchanged toolchain in 20 local comparisons: four repetitions for each of seeds 0, 1, 7, 42 and disabled mode. Those checks do not exercise process simulation, syscall-arrival races, GC-heavy targets, or exact choice replay. They establish feasibility, not full behavioral equivalence.

Two smaller hook consolidations are also possible. A compiler overlay in package `gc` can call `gomadintercept.Apply` and then `gomadguard.Apply`, leaving one call in `gc/main.go`; the measured source sketch saves 356 bytes and 10 patch lines. A linker setup helper in the existing `ld/gomadcap.go` can set `runtime.gomadExternal` and then emit the capability manifest, leaving one call in `ld/lib.go`; this sketch saves 217 bytes and six patch lines. Neither sketch was built. The external-link marker must be set before the manifest helper's optional `-gomadcap` early return.

## Non-scheduler reductions

Gomad can remove the `crypto/rand/rand.go` patch and move three syscall linkname declarations into overlays while retaining their existing implementations. These changes remove 1,371 bytes and 54 lines from the three-context-line patch. An embedded goroutine-state type can remove another 573 bytes and 11 lines while retaining the measured darwin/arm64 field offsets. These figures measure the patch alone. New overlay files retain the moved code.

The investigation used the pinned Go archive identified by [version.json](../../../tools/gomad3/toolchain/version/version.json), SHA-256 `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`. The archive's `go/src/...` members provide the upstream source below. Measurements use `git diff --no-index --unified=3 --abbrev=7` with canonical patch paths. Production patch and overlay files remain unchanged.

| Candidate | Existing section | Candidate section | Saving | Evidence |
| --- | --- | --- | --- | --- |
| Move crypto/rand initialization into an overlay | 747 bytes, 28 lines | 0 | 747 bytes, 28 lines | `crypto/rand.TestRead` passed with the relocated initialization |
| Move the environment linkname declaration into an overlay | 656 bytes, 27 lines | 440 bytes, 14 lines | 216 bytes, 13 lines | Seeded environment and disabled-mode fixtures passed |
| Move the two syscall.Write linkname declarations into an overlay | 1,155 bytes, 39 lines | 747 bytes, 26 lines | 408 bytes, 13 lines | Same package compiled and linked through the fixtures |
| Embed gomadGState at the current field position | 1,352 bytes, 29 lines | 779 bytes, 18 lines | 573 bytes, 11 lines | darwin/arm64 `runtime.TestSizeof` and measured offsets matched |

### Crypto initialization

Add `src/crypto/rand/gomad.go` to the source overlay. Move the existing initialization function and its comments verbatim. Import `crypto/internal/rand` and `internal/gomadio` in that file, then restore pristine upstream `rand.go`. The pinned archive contains no other non-test initialization function in `crypto/rand`. The package still initializes `Reader` before its initialization function runs, and dependent packages still wait for that initialization to finish. The relocated function therefore installs the same reader at the same package lifecycle stage. Sources are the archive's `src/crypto/rand/rand.go` and the current [patch](../../../tools/gomad3/toolchain/runtime/go1.27.1.patch).

Keep both assignments. Upstream `crypto/rand.Read` uses the public reader, while internal key generation uses `crypto/internal/rand.Reader` and the FIPS DRBG. `crypto/internal/rand.SetTestingReader` updates the DRBG override. Replacing only the public reader would lose deterministic key generation. The [entropy fixture](../../../tools/gomad3/internal/gomadtool/conformance/testdata/io_entropy/main.go) explicitly exercises `rand.Read`, `rand.Text`, and ECDSA key generation. The local experiment exercised disabled-mode `TestRead`; enabled I/O transcript and key-generation qualification remains required.

### Syscall declarations

Add `src/syscall/gomad_unix.go` with the three declarations for `gomadDeterministicEnabled`, `gomadCapabilityGuard`, and `gomadWrite`, plus its own unsafe import and the matching build constraint. Their names, runtime symbols, and signatures remain unchanged. Remove the unsafe import added solely for the declaration in `env_unix.go`. Imports apply to individual Go source files; `syscall_unix.go` already imports unsafe independently. The runtime's existing [linkname definitions](../../../tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go) remain unchanged.

Retain the environment reset inside `copyenv` and retain both syscall.Write body hooks for this low-risk relocation. The [seeded environment fixture](../../../tools/gomad3/internal/gomadtool/conformance/testdata/gotest/gotest_test.go) passed with seed 17 and still observed exactly `TZ=UTC`. Its disabled-mode compatibility fixture also passed. These runs prove the relocation compiles and preserves those exercised behaviors on darwin/arm64. They do not prove the syscall.Write guard's negative cases or Linux output path.

### Goroutine state

Define `gomadGState` in the runtime overlay with the current four fields in the same order, then embed it at their current position after `goid`. Go's promoted field selectors preserve accesses such as `gp.gomadIdentity`. The embedded type's trailing alignment padding replaces the padding before `schedlink`. `gofmt` still realigns three fields after the anonymous embedding, so this proposal saves 11 patch lines rather than reducing the hunk to one added line. Sources are the archive's `src/runtime/runtime2.go` and [Gomad's goroutine-state consumers](../../../tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go).

The darwin/arm64 experiment measured the following values in both baseline and candidate runtimes. `runtime.TestSizeof` passed in both. The embedded candidate also cross-compiled and linked the runtime test binary for linux/amd64; Linux tests were not executed.

| Quantity | Baseline and embedded candidate |
| --- | --- |
| g size / alignment | 512 / 8 |
| goid offset | 152 |
| gomadIdentity offset | 160 |
| gomadChildOrdinal offset | 192 |
| gomadSimulationDomain offset | 200 |
| gomadSimulationTransport offset | 208 |
| schedlink offset | 216 |
| gcAssistBytes offset | 496 |
| valgrindStackID offset | 504 |

The size includes the architecture's extended-register storage. Retain the existing `sizeof_test.go` patch. Embedding changes the structural type declaration, and full replay qualification still needs to establish that supported workloads retain their behavior. A side table would add allocations, lifetime management, and scheduler-sensitive lookups to these accesses. It offers no established equivalence. Appending the fields at the end passed `TestSizeof` locally and saved 766 patch bytes, but changes existing field offsets; embedding provides stronger evidence.

## Candidates requiring additional design

Moving the environment reset to a syscall overlay initialization function could remove the remaining `env_unix.go` patch. The pinned darwin/arm64 and linux/amd64 syscall initialization functions never call the environment functions, and dependent packages initialize later. A scratch version passed the same seeded and disabled fixtures. However, it creates the replacement slice earlier and changes initialization work. Keep this proposal conditional on repeatability and replay tests. It would save 656 bytes and 27 lines in total, replacing the 216-byte declaration-only saving rather than adding to it. Source members are `src/syscall/env_unix.go`, `src/syscall/rlimit.go`, and `src/syscall/syscall_darwin.go` in the pinned archive.

The existing [compiler interceptor](../../../tools/gomad3/toolchain/runtime/overlay/src/cmd/compile/internal/gomadintercept/intercept.go) can intercept a named `syscall.Write` function. A hook could call the same capability guard and runtime write helper. It must preserve the faketime branch and race, MSan, and ASan annotations currently surrounding the write. The present interception convention prepends a hook and returns immediately when handled, which would otherwise bypass those annotations. Its declaration fingerprint and any boundary evidence must also change. Retaining the small body patch avoids introducing this unqualified behavior change.

The interceptor cannot currently select the `copyenv` closure or the resource-limit initialization function directly. It matches exact compiler function names and validates a source-level named declaration. Go renames initialization functions to `init.N`, and the closure is a function literal. Supporting those targets requires new selection and fingerprint rules. Upstream source is `src/cmd/compile/internal/noder/reader.go`, particularly `Renameinit` and `pkgInitOrder`.

The resource-limit patch suppresses the host query and limit change in both enabled and disabled modes. Replacing it with an enabled-only guard would change existing disabled behavior. Any smaller guard must preserve that deliberate unconditional suppression and preserve upstream comments. An additional ordinary overlay initialization function cannot cancel the upstream initialization side effects. Source is the archive's `src/syscall/rlimit.go` and the current [patch](../../../tools/gomad3/toolchain/runtime/go1.27.1.patch).

## Changes that need to remain

The smaller patch should retain the effects of the hooks below. Their justification is visible in the [current patch](../../../tools/gomad3/toolchain/runtime/go1.27.1.patch), the overlay implementation, and the [milestone constraints and open findings](../../../.plans/GOMAD_MILESTONES.md).

- Runtime activation must happen before random initialization, and seeded scheduling must retain one P, disabled host preemption, separate host and target random streams, and recorded local/global run-queue decisions.
- Keep `snapshotAllp`, mark-worker gating, and runtime-structure greying. A Linux bisect that still diverged after reverting greying does not prove that greying serves no other workload. The collector, allocator, and platform-file prohibitions still apply.
- Keep the arrival queue, one-arrival-per-idle-window policy, quiescence checks, and transport syscall accounting. A helper may encapsulate them; replacing them with ordinary global-run-queue admission changes host-timing behavior.
- Keep the `timeSleepUntil` presence result. `maxWhen` is also a valid saturated deadline, so a timestamp alone cannot distinguish a timer from no timers. Retain timer tie-breaking, `nanotime` activation guarding, and `time_runtimeNow`'s synctest handling.
- Keep choice identity assignment, select caller-site capture, select observations, and panic/testing completion hooks. Extraction must retain the original caller PC explicitly where the code observes it. Caller-site values are text offsets, so a changed binary needs its own replay qualification.
- Keep traceback normalization and the upstream `sizeof` expectation. They cover reproducible failure output and the larger goroutine layout.

Replacing patched upstream files wholesale with overlay copies would require relaxing the current overlay collision rule and carrying those full upstream implementations. Compressing the patch on disk would require changing its plain-text validation/application path. Neither approach reduces the number of upstream hooks that must be understood during Go upgrades. See [build.go](../../../tools/gomad3/toolchain/build.go) and [patch.go](../../../tools/gomad3/toolchain/patch.go).

## Experiment boundaries

Scratch sources and Go overlay maps were created under `/var/folders/4w/5qdjw8sd6417nldg5pvhs_rr0000gn/T/gomad-patch-nonproc-a50y42fq`. The cached pinned compiler rebuilt the affected packages with `GOTOOLCHAIN=local`, `GOWORK=off`, `GOENV=off`, empty `GOFLAGS`, `CGO_ENABLED=0`, and `GOEXPERIMENT=nogreenteagc`. Runtime tests and fixture builds used `-tags=test_dep`; the crypto/rand test used its default standard-library configuration. These are development experiments, not a rebuilt and qualified release toolchain.

Before adopting semantic reductions, run the repository's builder, live-capability, runtime, and upstream tiers on both supported platforms, then the entropy and process-simulation repeatability and exact-replay gates affected by the moved code. [The Makefile](../../../tools/gomad3/Makefile) and [upstream fixture definitions](../../../tools/gomad3/internal/gomadtool/conformance/driver.go) own those gates. Update the descriptor's overlay allowlist and regenerate its derived artifacts; the builder rejects undeclared overlay additions and collisions with upstream files through [patch validation](../../../tools/gomad3/toolchain/patch.go).

## Qualification and reproducibility

The current patch and overlay were not modified by this investigation. The source-reorganization measurements are candidates. Approve equivalence only after exercising the integrated candidate through the existing gates on both supported hosts.

Use `make -C tools/gomad3 test` for the governed descriptor, archive collision checks, compiler interception, disabled behavior, runtime repeatability under host load, upstream runtime/time/synctest coverage, and boundary tests. Scheduler/quiescence changes also need the process-backend clock, transport, crash-drain, and exact-replay tests in [cluster_toolchain_test.go](../../../tools/gomad3sim/cluster_toolchain_test.go), plus the affected smoke qualification workloads. Preserve the existing treatment of known intermittent Linux findings when comparing evidence; do not claim that this refactor fixes those findings.

Removing the `crypto/rand` hunk and adding overlay files requires updating the exact source sets in [version.json](../../../tools/gomad3/toolchain/version/version.json) and regenerating its consumers. The [descriptor](../../../tools/gomad3/toolchain/version/descriptor.go) requires the allowlists to equal the actual patch and overlay paths. The current pinned regeneration test in [patch_test.go](../../../tools/gomad3/toolchain/patch_test.go) still hardcodes `go1.26.4` and skips when that archive is absent. Retarget that check to the descriptor when changing regeneration behavior; its present skip cannot prove Go 1.27.1 reproducibility.

Every patch-byte or overlay change alters the [toolchain build key](../../../tools/gomad3/toolchain/buildkey.go). Even a context-only change needs a new cached toolchain identity; existing artifacts remain bound to their original toolchain. Source equivalence does not permit relabeling those artifacts.

Measurements used repository HEAD `29917069e` on 2026-09-30. The patch SHA-256 was `3ac420beb1ea52271ff3320a196894a16110b536f9635ce19073e4d772047d0c`. The cached source archive SHA-256 was `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`, matching the release descriptor.

To reproduce the representation measurements, initialize an ordinary temporary Git repository containing pristine files from the verified archive, stage those files, apply the current patch, and run `git diff --no-ext-diff --diff-algorithm=myers -U<N>` for each context size. No commit or worktree is needed. Apply each generated patch to a separate pristine copy using the builder's zero-fuzz commands and compare every resulting file to the current materialization. The local `patch-validate` command also accepted the `-U1` candidate.

Temporary experiment outputs remain under `/var/folders/4w/5qdjw8sd6417nldg5pvhs_rr0000gn/T/gomad-patch-size-aa2bzfn6`, including measured patches and Go build-overlay JSON files. They are investigation artifacts, not governed release inputs.
