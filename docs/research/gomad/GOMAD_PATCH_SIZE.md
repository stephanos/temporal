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

### Scheduler extraction implementation

Task fn-110.2 regenerated the three-context-line patch on 2026-10-03 after
moving the three scheduler implementations into `src/runtime/gomad.go`. The
checkout already included the fn-114.13 and fn-112.5 runtime changes, so its
pre-extraction patch was larger than the fn-110.1 baseline. Those changes were
retained.

| Measurement | fn-110.1 baseline | Task 2 start | Scheduler extraction | Task 2 delta |
| --- | ---: | ---: | ---: | ---: |
| Patch bytes | 32,652 | 36,347 | 33,288 | -3,059 |
| Patch lines | 1,007 | 1,127 | 1,032 | -95 |
| `src/runtime/proc.go` added / deleted | 205 / 20 | 214 / 26 | 126 / 26 | -88 / 0 |
| `src/runtime/gomad.go` overlay bytes | not isolated | 62,978 | 65,913 | +2,935 |
| `src/runtime/gomad.go` overlay lines | not isolated | 1,782 | 1,882 | +100 |

The extraction therefore reproduces the investigation's 3,059-byte and
95-line saving against the source it actually started from. The intermediate
patch is 636 bytes larger than the earlier fn-110.1 snapshot because of the
retained runtime work merged between tasks; the extraction itself did not
reverse or hide that work. The pre-extraction patch SHA-256 was
`9b8dda3bec6e059ba2c51a3f076dc94a8899121d19f6e837bca1b53b8c97a807`;
the regenerated patch SHA-256 is
`f0b9d836c8930adfa0f75e2bfe64b0528c65cb4cff66acdccbb677b4a234a6ad`.

The governed materialize/regenerate workflow and `make -C tools/gomad3
generate validate` passed. Separate baseline and extracted source trees both
compiled with their overlays on the available `linux/arm64` development host,
and the locked-syscall fixture produced `locked syscall resumed` on both for
disabled mode and seeds 0, 1, 7, and 42. `runtime.TestSizeof` also passed in
both local trees. An audit follow-up rebuilt the fixture against both trees and
passed the named `arrival-before-timer`, `arrival-after-timer-fired`, and
`arrival-during-quiescence-round-trip` modes. The first mode also verifies that
its pending timer eventually fires. These are development checks only:
`linux/arm64` is not a
qualified Gomad platform. No native `darwin/arm64` or `linux/amd64` fixture,
replay, runtime, upstream, live-capability, or process-simulation comparison
ran, so behavioral qualification and both supported-platform acceptance
remain incomplete.

The earlier investigation candidate compiled and linked on `darwin/arm64` and cross-compiled and linked for `linux/amd64`. For each scheduler prototype, a small timer/goroutine program matched the unchanged toolchain in 20 local comparisons: four repetitions for each of seeds 0, 1, 7, 42 and disabled mode. Those checks do not exercise process simulation, syscall-arrival races, GC-heavy targets, or exact choice replay. They establish feasibility, not full behavioral equivalence and are not task 2 qualification evidence.

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

### Crypto and syscall relocation implementation

Task fn-110.3 regenerated the three-context-line patch on 2026-10-03. The
checkout already contained a fourth syscall linkname declaration,
`gomadIOProfileEnabled`, which `copyenv` reads next to
`gomadDeterministicEnabled`. It moved with the other three, so the task
relocated four declarations instead of the investigation's three.

- [`src/crypto/rand/gomad.go`](../../../tools/gomad3/toolchain/runtime/overlay/src/crypto/rand/gomad.go)
  holds the `init` function and its comment verbatim, with both
  `Reader = gomadio.RandomReader()` and `rand.SetTestingReader(Reader)`. The
  patch no longer touches `crypto/rand`, and the materialized `rand.go` is
  byte-identical to the archive. The archive has no other non-test `init` in
  the package, and `Reader` is a package-level variable, so it is initialized
  before any `init` runs. The overlay file sorting before `rand.go` therefore
  does not change when the reader is installed.
- [`src/syscall/gomad_env_unix.go`](../../../tools/gomad3/toolchain/runtime/overlay/src/syscall/gomad_env_unix.go)
  declares `gomadDeterministicEnabled` and `gomadIOProfileEnabled` under
  `env_unix.go`'s constraint, `unix || (js && wasm) || plan9 || wasip1`.
- [`src/syscall/gomad_unix.go`](../../../tools/gomad3/toolchain/runtime/overlay/src/syscall/gomad_unix.go)
  declares `gomadCapabilityGuard` and `gomadWrite` (`runtime.gomadSyscallWrite`)
  under `syscall_unix.go`'s `unix` constraint.

Each declaration group uses its own file because one `unix` file would drop
the environment declarations from js/wasm, plan9, and wasip1. One file with the
wider constraint would add the write declarations where `syscall_unix.go`
does not build. `go list` on the rebuilt toolchain selects the new files exactly
where their source files build: both on linux, darwin, freebsd, aix, and
solaris; only the environment file on js/wasm, wasip1, and plan9; and neither
on windows. Names, linkname targets, and signatures are unchanged. The
`copyenv` reset, both `Write` hooks, and the `rlimit.go` section remain in the
patch.

| Measurement | Task 3 start | Task 3 result | Delta |
| --- | ---: | ---: | ---: |
| Patch bytes (`-U3`) | 39,837 | 38,362 | -1,475 |
| Patch lines | 1,169 | 1,112 | -57 |
| Patched files / hunks | 21 / 85 | 20 / 81 | -1 / -4 |
| Patch added / deleted lines | 341 / 103 | 318 / 103 | -23 / 0 |
| New overlay files (bytes / lines) | — | 1,520 / 55 | +1,520 / +55 |

The new overlay files are `crypto/rand/gomad.go` (469 bytes, 19 lines),
`syscall/gomad_env_unix.go` (549 bytes, 18 lines), and `syscall/gomad_unix.go`
(502 bytes, 18 lines). The overlay grows by more than the patch shrinks
because each new file carries a license header, imports, and a constraint.
This relocation reduces the upstream patch, not the total amount of source.
The start patch was larger than the task 2 result because other runtime work
was merged between tasks. That work was retained. The start patch SHA-256 was
`11450b94d35ade1caddf9bbcdf564eda17d4a0573738f462a812fb014789e3ae`; the
final `-U3` patch SHA-256 is
`86def26a7f4d0b5c494a6a031c87bec284f7e76c91dcf437bc23fcc4c276ea5c`. Because
the patch is an input to the generated choice implementation digest,
`make generate` also rewrote `ImplementationSourceSHA256` in both choice wire
codecs. The descriptor's allowlists drop `src/crypto/rand/rand.go` and add the
three overlay paths.

Only the `linux/arm64` development host was available. Local runs used a
temporary uncommitted descriptor harness that adds that platform. Both the
task 3 start tree and the candidate were rebuilt with that harness, and the
archive-based overlay collision check passed in both builds. The following
checks matched between the two trees:

- Enabled-profile `io_entropy` (`rand.Read`, `rand.Text`, and ECDSA key generation through the FIPS override) gave identical output and transcript digests for seeds 1, 11, 17, and 999.
- Profile-mode `environment` output also matched for those seeds.
- In seeded runs without a profile (seeds 1 and 17), `os.Environ()` was exactly `["TZ=UTC"]` and host variables were hidden. `syscall.Write(9, …)` died with `GOMAD_CAPABILITY_DENIED`, while direct writes to descriptors 1 and 2 succeeded.
- In disabled mode, host variables remained visible and `Write(9)` returned `EBADF`.
- `gotest` passed `TestSeedReachesTestBinary` with seed 17 and `TestDisabledCompatibility` in disabled mode, with identical output.
- Disabled-mode upstream `crypto/rand` and `syscall` tests reported identical per-test results across 378 cases. On both trees, the Linux-only `TestPrlimitFileLimit` failed because the existing `rlimit.go` hunk leaves the descriptor limit unchanged. That failure predates this task.

On the candidate, `TestProfileEntropyIsIndependentOfScheduleSeed`,
`TestToolchainLeavesFD5ForProcessesWithoutIOProfile`, and
`TestIOProfileFailureArtifactReplaysExactly` passed. The darwin-only
live-capability tests skipped. None of these local runs is native
`darwin/arm64` or `linux/amd64` evidence. Those gates remain incomplete.

### Canonical one-context-line patch

Task fn-110.4 changed only the patch representation. `RegeneratePatch` now
passes `--unified=1` to `git diff` explicitly through the
`canonicalPatchContext` constant, so a `diff.context` setting no longer changes
the patch. The `patch-regenerate` command gained no flag. Canonical headers,
`validatePatch`, the `git apply --cached --check` step, and zero-fuzz
materialization are unchanged. Tests reach the three-context form through the
unexported `regeneratePatch` seam.

The checked-in patch was regenerated from a fresh extraction with the task 3
`-U3` patch applied. Two regenerations, and the canonical command writing to
the descriptor path, produced byte-identical output.

| Patch | Bytes | Lines | Hunks | SHA-256 |
| --- | ---: | ---: | ---: | --- |
| Final extracted source, `-U3` (task 3) | 38,362 | 1,112 | 81 | `86def26a7f4d0b5c494a6a031c87bec284f7e76c91dcf437bc23fcc4c276ea5c` |
| Canonical `-U1` | 29,015 | 778 | 90 | `8497f8855011f13fb46ad36a02448d165d4bd65688ef00eed6ae09822306a90b` |

The `-U1` patch is 9,347 bytes (24.4%) and 334 lines smaller than the final
`-U3` patch. Both carry the same 318 added and 103 deleted lines in 20 files.
Both patches applied with the builder's zero-fuzz commands to separate fresh
extractions of the verified archive on the `linux/arm64` development host.
`diff -r` found no difference between the two trees, and 20 files differ from
pristine source. The same comparison now runs as
`TestPinnedContextRepresentationsMaterializeIdenticalSource` in the
`toolchain` package. That test regenerates both forms from the checked
candidate, requires at most one context line around each change in the `-U1`
form, and compares every allowlisted file.
`TestRegenerateMatchesCheckedPatchForPinnedArchive` now reads the archive name
and patch path from the descriptor and fails on a checksum mismatch. It skips
only when the pinned archive is not cached, and it requires repeated
regeneration to be byte-identical. It passed here; it had previously always
skipped because it named `go1.26.4`. The unchanged negative tests still pass.
Equivalence on native `darwin/arm64` and `linux/amd64` remains incomplete.

Because the patch bytes changed, the toolchain build key and the generated
choice implementation digest changed again. Artifacts recorded with earlier
toolchains keep their original identities.

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

The smaller patch should retain the effects of the hooks below. Their justification is visible in the [current patch](../../../tools/gomad3/toolchain/runtime/go1.27.1.patch), the overlay implementation, and the [milestone constraints and open findings](../../../MILESTONES.md).

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

## Implementation baseline

Recorded 2026-10-02 (UTC) by task fn-110.1 on darwin/arm64, before any patch or overlay edit. Every later fn-110 size and equivalence claim compares against this section. The baseline patch is 32,652 bytes and 1,007 lines, which is 377 bytes and 9 lines more than the investigation measured above. Task fn-110.1 changed no file under `tools/gomad3/toolchain/`.

Three findings limit later claims. The process-simulation test fails at the baseline in 2 of its 11 subtests. Runner source files changed after the reused gates ran. No linux/amd64 gate ran. "Blocking and drifting inputs" states what each one blocks.

### Identity and inputs

| Item | Value |
| --- | --- |
| Measured revision | `6782b55f49a0317b230e827ea2a63a37d116d502` on branch `stephanos/gomad`, plus the then-uncommitted fn-108 edits |
| Baseline commit | `38957053f1ce342a8797af1803f5f8f6bb53fcad`, committed by the user at 2026-10-02T00:23 UTC while this task ran. For `tools/gomad3`, `tools/gomad3sim`, and `tools/gomad3integration` it holds exactly the tree the reused gates ran on: `git diff 6782b55f4 38957053f --diff-filter=M` over those directories has SHA-256 `47afb287…14513855`, the digest the gates recorded, and the commit adds the same seven files |
| `git status --short -- tools/gomad3/toolchain` | empty before and after that commit; `git diff 6782b55f4 38957053f -- tools/gomad3/toolchain` is empty |
| Git tree of `tools/gomad3/toolchain` | `d3d28af6492184d34cab57780d5a5742331324d3` at both revisions |
| Patch SHA-256 | `950063a87d63cb01dada2e6ef232fdeb4acc52f71a008e2158885666669f80bc` |
| Archive SHA-256 | `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`, equal to `archive.sha256` in [version.json](../../../tools/gomad3/toolchain/version/version.json) |
| Archive location | `tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz`, 35,109,201 bytes |
| Descriptor SHA-256 | `87eb02f3f8c27c42d411c5a82ae98ef5e86d0e17605f1c6cec10bae316275c2c` |
| Baseline build key | `8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc` |
| Runner build of the reused gates | `sha256:f8b0a8d42df61c41cfc007ca19a5891a76b0fe50f9c5e8061c9ab5279a93737f`, the SHA-256 of `tools/gomad3/.bin/gomad` |
| Host | macOS 26.6.2 (25G83), Apple M2, host `go1.27.1`, `git` 2.54.0, `patch` 2.0-12u11-Apple |

`make -C tools/gomad3 toolchain` printed `gomad3 toolchain is ready (darwin/arm64, key 8d28bd44…)` in two seconds without rebuilding, so the active toolchain already matches the patch and overlay bytes. `make -C tools/gomad3 validate` exited 0.

### Difference from the investigation figures

The investigation measured HEAD `29917069e`, patch SHA-256 `3ac420be…`, 32,275 bytes and 998 lines. Task fn-105.14 (D14) then added one hunk to `src/runtime/lock_spinbit.go` that replaces `gp.m.mLockProfile.start()` with `gomadLockProfileStart(&gp.m.mLockProfile)`, and added that function to the runtime overlay. The file stayed in the patch, so the count of edited upstream files is still 20. The overlay grew by 792 bytes in total: `src/runtime/gomad.go` grew by 809 bytes and 19 lines for the new function, the regenerated `src/internal/gomadchoicewire/wire_generated.go` shrank by 17 bytes, and the regenerated `src/cmd/internal/gomadcap/protocol_generated.go` changed two lines at the same size.

| Quantity | Investigation | Baseline | Change |
| --- | ---: | ---: | ---: |
| Patch bytes | 32,275 | 32,652 | +377 |
| Patch lines | 998 | 1,007 | +9 |
| Added / deleted source lines | 341 / 54 | 342 / 55 | +1 / +1 |
| `lock_spinbit.go` section bytes / lines | 511 / 13 | 888 / 22 | +377 / +9 |
| Overlay files | 57 | 57 | 0 |
| Overlay bytes | 600,724 | 601,516 | +792 |
| Overlay lines | 17,086 | 17,105 | +19 |
| `overlay/src/runtime/gomad.go` bytes / lines | 45,821 / 1,356 | 46,630 / 1,375 | +809 / +19 |
| Build key | `1803c664…` | `8d28bd44…` | new identity |

The `proc.go`, `crypto/rand`, `env_unix.go`, `syscall_unix.go`, and `runtime2.go` sections have the sizes the investigation reported, so its per-candidate savings still apply to the same hunks. Its percentages used 32,275 bytes as the denominator. Requirement R8 compares against 32,652 bytes.

### Patch and overlay measurements

[task1-measure.sh](../../../.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-measure.sh) produced every figure in this subsection and [task1-measurements.txt](../../../.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-measurements.txt) holds its output. The script reads files and writes nothing. Run it from the repository root.

| Figure | Value | Command |
| --- | ---: | --- |
| Patch bytes | 32,652 | `wc -c < tools/gomad3/toolchain/runtime/go1.27.1.patch` |
| Patch lines | 1,007 | `wc -l < tools/gomad3/toolchain/runtime/go1.27.1.patch` |
| Edited upstream files | 20 | `grep -c '^diff --git' tools/gomad3/toolchain/runtime/go1.27.1.patch` |
| Hunks | 71 | `grep -c '^@@ ' tools/gomad3/toolchain/runtime/go1.27.1.patch` |
| Added source lines | 342 | `git apply --numstat tools/gomad3/toolchain/runtime/go1.27.1.patch`, first column summed |
| Deleted source lines | 55 | same command, second column summed |
| `src/runtime/proc.go` added / deleted | 205 / 20 | same command, `proc.go` row |
| Overlay files | 57 | `find tools/gomad3/toolchain/runtime/overlay -type f \| wc -l` |
| Overlay bytes | 601,516 | `find tools/gomad3/toolchain/runtime/overlay -type f -print0 \| xargs -0 cat \| wc -c` |
| Overlay lines | 17,105 | the same pipeline ending in `wc -l` |
| `overlay/src/runtime/gomad.go` bytes / lines | 46,630 / 1,375 | `wc -c` and `wc -l` on that file |

Each patch section starts at its `diff --git` line and ends before the next one.

| Patched file | Section bytes | Section lines | Hunks | Added | Deleted |
| --- | ---: | ---: | ---: | ---: | ---: |
| `src/cmd/compile/internal/gc/main.go` | 758 | 22 | 2 | 4 | 0 |
| `src/cmd/dist/buildtool.go` | 379 | 12 | 1 | 1 | 0 |
| `src/cmd/link/internal/ld/lib.go` | 638 | 18 | 1 | 7 | 0 |
| `src/crypto/rand/rand.go` | 747 | 28 | 2 | 10 | 0 |
| `src/runtime/lock_spinbit.go` | 888 | 22 | 2 | 2 | 2 |
| `src/runtime/panic.go` | 464 | 14 | 1 | 3 | 0 |
| `src/runtime/preempt.go` | 404 | 15 | 1 | 4 | 0 |
| `src/runtime/proc.go` | 15,145 | 467 | 31 | 205 | 20 |
| `src/runtime/rand.go` | 1,499 | 49 | 4 | 16 | 1 |
| `src/runtime/runtime2.go` | 1,352 | 29 | 1 | 11 | 7 |
| `src/runtime/select.go` | 1,583 | 47 | 4 | 14 | 1 |
| `src/runtime/sizeof_test.go` | 539 | 13 | 1 | 1 | 1 |
| `src/runtime/symtab.go` | 519 | 13 | 1 | 1 | 1 |
| `src/runtime/time.go` | 2,798 | 87 | 7 | 20 | 5 |
| `src/runtime/time_nofake.go` | 334 | 14 | 1 | 3 | 0 |
| `src/runtime/traceback.go` | 639 | 22 | 1 | 9 | 1 |
| `src/syscall/env_unix.go` | 656 | 27 | 2 | 7 | 0 |
| `src/syscall/rlimit.go` | 1,144 | 29 | 1 | 2 | 16 |
| `src/syscall/syscall_unix.go` | 1,155 | 39 | 3 | 14 | 0 |
| `src/testing/testing.go` | 1,011 | 40 | 4 | 8 | 0 |
| Total | 32,652 | 1,007 | 71 | 342 | 55 |

### Reproduction from the pinned archive

The checked-in patch is exactly the three-context-line diff of its own materialization. The 20 patched members were extracted from the verified archive into a scratch directory outside the repository and copied to `a/src` and `b/src`. The builder's `patch --dry-run --batch -V none -p1 -F 0` and `patch --batch -V none -p1 -F 0` both exited 0 on `b` with no fuzz or offset message and no `.orig` or `.rej` file. `git diff --no-index --no-ext-diff --binary --no-prefix --abbrev=7 --diff-algorithm=myers -U<N> a b` then produced the rows below. No Git repository, index, or worktree was created.

| Context | Bytes | Lines | SHA-256 | Note |
| --- | ---: | ---: | --- | --- |
| `-U3` | 32,652 | 1,007 | `950063a8…669f80bc` | `cmp` reports it identical to the checked-in patch |
| `-U2` | 28,859 | 865 | `caea8024…94ef76cf6` | reference only |
| `-U1` | 24,620 | 715 | `5f021b81…4254642067` | baseline source at the final representation, 24.6% fewer bytes than `-U3` |
| `-U0` | 20,844 | 558 | `c1adc512…b8dd208c66` | reference only; excluded by the spec |

This proves the baseline source and its `-U3` form on the local macOS `patch` and Git 2.54.0. It was not run on Linux.

### Source sets

`patch_allowlist` has 20 entries and `overlay_allowlist` has 57 entries. The script compares each list with the sorted paths in the patch headers and in the overlay tree, and both comparisons are equal. `make -C tools/gomad3 validate` ran `version-generate -check`, `protocol-generate -check`, `boundary-generate -check`, `boundary-generate -check-compiler-tests`, `patch-validate`, `script-validate`, `compatibility-pack check`, `TestHostPacksBindCurrentProfile`, and `qualification-manifest-generate -check`, and exited 0.

### Darwin outcomes before the change

Two kinds of result appear below. "Reused" results come from the fn-108.7 final gates, which ran on 2026-10-01 between 22:15 and 23:33 UTC on these toolchain inputs. "Fresh" results were run by fn-110.1 between 2026-10-01T23:57 and 2026-10-02T00:16 UTC. Reuse is valid for the toolchain because the patch SHA-256, the toolchain Git tree, the empty toolchain status, and the build key recorded in `tools/gomad3/.toolchain/fn-110/baseline/identity.json` equal the values in "Identity and inputs". The three report digests and the three row listings also equal the fn-108 copies. The Runner caveat is in "Blocking and drifting inputs".

| Gate | Command | Result | Source |
| --- | --- | --- | --- |
| Governed validation | `make -C tools/gomad3 validate` | exit 0 | fresh, and reused |
| Toolchain current | `make -C tools/gomad3 toolchain` | exit 0, no rebuild, key `8d28bd44…` | fresh |
| Gomad gate, ten tiers | `env GOFLAGS=-count=1 make -C tools/gomad3 test` | exit 0 in 1,197 s; `gomad3 all black-box tiers passed` | reused |
| Runtime tier, per case | `conformance.Run` in mode `test-runtime` | 4,245 of 4,245 cases pass in 625 s | fresh |
| Upstream tier, per case | `conformance.Run` in mode `test-upstream` | 2 of 2 cases pass in 108 s | fresh |
| Upstream `crypto/rand` and `syscall` | `.toolchain/bin/go test -tags test_dep -count=1 crypto/rand syscall` | exit 0; 317 pass in `crypto/rand`; 30 pass and 3 skip in `syscall` | fresh |
| Process simulation | `.toolchain/bin/go test -count=1 -v -tags test_dep,integration -run '^TestRootProcessSimulationUsesRunnerTransport$' ./runner/internal/execution` (14 runs), and the same command with the host `go` (11 runs) | **exit 1 in 25 of 25 runs**. In 23 runs 9 subtests pass and 2 fail. In 2 runs 8 pass and 3 fail, the third on a watchdog timeout | fresh |
| `gomad3sim` host tests | `go test -count=1 -tags test_dep ./tools/gomad3sim/...` | exit 0; 55 tests pass | reused |
| Temporal integration | `make gomad3-integration-test` | exit 0; 3 tests pass | reused |
| Smoke qualification | `make gomad3-smoke-qualification` | exit 0; 4 of 4 qualified on seed 11; 4 replayed, 0 diverged | reused |
| Core qualification | `make -C tools/gomad3 core-qualification` | exit 0; 9 compatibility-pack requests qualified; 7 of 7 qualified on seed 17; 7 replayed, 0 diverged | reused |
| Temporal qualification | `make gomad3-qualification GOMAD3_QUALIFICATION_PRUNE=1` | exit 0 in 2,361 s; 28 of 28 qualified on seeds 11 and 17; 56 seed-runs replayed, 0 diverged | reused |

The reused gate log is [final-gates/results.txt](../../../.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final-gates/results.txt) and its tier output is [final-gates/test.log](../../../.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final-gates/test.log). The ten tiers are `test-harness`, `test-toolchain`, `intercept-test`, `test-host`, `overlay-test`, `world-test`, `test-builder`, `test-live-capability`, `test-runtime`, and `test-upstream`. [test-dispositions-darwin-arm64.tsv](../../../.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final-gates/test-dispositions-darwin-arm64.tsv) lists 1,163 test names for `test-harness`, `test-host`, `world-test`, `gomad3sim`, and integration: 1,142 pass and 21 skip. Nineteen skips are helper entry points that only run as a child process. `TestMemberlistSuppliedTCPConsumer` skips without `GOMAD_MEMBERLIST_TCP_CONSUMER_DIR`. `TestRegenerateMatchesCheckedPatchForPinnedArchive` skips because it looks for the `go1.26.4` archive. The `test-toolchain`, `intercept-test`, `overlay-test`, and `test-live-capability` tiers have package-level results only.

**Runtime and upstream tiers by case.** `gomadtool test` prints one success line per tier and discards the per-case report. A 69-line scratch program under the gitignored `tools/gomad3/.toolchain/fn-110/baseline/casereport/` calls the same `toolchain.ValidatePatch` and `conformance.Run` and writes one row per case. It ran with `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off go run ./.toolchain/fn-110/baseline/casereport --root="$PWD" --mode=<tier> --go="$PWD/.toolchain/bin/go" --out=<tsv>` from `tools/gomad3`. A copy of its source is [task1-casereport-main.go.txt](../../../.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-casereport-main.go.txt).

- Runtime tier: 4,245 cases in 141 name families, all `pass`. [task1-test-runtime-case-families.tsv](../../../.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-test-runtime-case-families.tsv) lists the families with counts. The full list is `tools/gomad3/.toolchain/fn-110/baseline/test-runtime-cases.tsv`, 620,557 bytes, SHA-256 `6dc6bb1e…04b061c41`.
- Upstream tier: `upstream-clock` passes (`runtime` 76.8 s, `time`, `testing/synctest`). `upstream-dist` passes (`bytes`, `context`, `encoding/json`, `io`, `cmd/compile/internal/ssa`, `cmd/go/internal/load`, `cmd/go/internal/modload`, `cmd/go/internal/work`, `cmd/link/internal/ld`, `cmd/link/internal/loader`; `cmd/compile/internal/gc` has no test files). See [task1-test-upstream-cases.tsv](../../../.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-test-upstream-cases.tsv) and [task1-test-upstream-stdout.txt](../../../.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-test-upstream-stdout.txt).

**Process simulation.** No Makefile target or CI job runs `TestRootProcessSimulationUsesRunnerTransport`. The command in the table ran from `tools/gomad3` with `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off`. [task1-process-simulation.tsv](../../../.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-process-simulation.tsv) holds 275 rows from 25 runs: 14 with `.toolchain/bin/go` as the test runner and 11 with the host `go`. One of the 14 is the build-overlay run described below the table.

| Subtest | Baseline result in 25 runs |
| --- | --- |
| `TestScenarioChoicePlanForcesRankBoundDecisionAndExactlyReplays` | pass 25 |
| `TestScenarioChoicePlanRejectsChangedDecisionBeforeSelection` | **fail 25**: `scenario decision identity is invalid` at `scenario_control_toolchain_test.go:66` |
| `TestProcessExplorationConsumesExternalPlanAndPublishesRecord` | **fail 25**: `simulation exploration candidate identity does not match` at `scenario_control_toolchain_test.go:120` |
| `TestProcessBackendResetsGlobalsDescriptorsAndGoroutines` | pass 25 |
| `TestProcessAndInProcessBackendsHaveEquivalentDetachedModels` | pass 25 |
| `TestProcessBackendRoutesTCPThroughSharedHostModel` | pass 25 |
| `TestProcessBackendSynchronizesNodeClockWithModelDelay` | pass 25 |
| `TestProcessBackendRoutesListenThroughSharedHostModel` | pass 25 |
| `TestProcessBackendPreservesHostVolumeAcrossRestart` | pass 24; 1 watchdog timeout after 30 s |
| `TestProcessBackendCrashDrainsInflightModelOperationDeterministically` | pass 25 |
| `TestProcessBackendModelDigestsIgnoreCompletionOrder` | pass 24; 1 watchdog timeout after 30 s, at a 1-minute load average of 18.6 |

The two identity failures occur in every run with either test runner. They also occur when a Go build overlay substitutes the HEAD versions of the three fn-108-modified files in the test's dependency closure (`deterministicio/domain.go`, `deterministicio/memory_adapter.go`, `target/capability.go`), so the uncommitted fn-108 edits did not cause them. Their cause was not investigated and fn-110.1 changed nothing to address them. The two watchdog timeouts hit different subtests in 2 of 25 runs, with no target output before the timeout.

**Tests that tasks 2, 3, and 5 rely on, by name.**

| Test or case | Tier | Baseline result |
| --- | --- | --- |
| `TestProfileEntropyIsIndependentOfScheduleSeed` (`runner/internal/execution`; runs the `io_entropy` fixture: `rand.Read`, `rand.Text`, ECDSA key generation) | `test-host` | pass (reused) |
| `TestProfileSQLiteUsesVirtualTimeAndEntropy` | `test-host` | pass (reused) |
| `TestBoundaryManifestSemanticCanaries`, `TestProfilePassesHostCapabilitySandbox`, `TestRunBoundsFloodedOutputWithoutBlocking`, `TestValidateRequestRejectsChoiceEnvironmentInjection` | `test-host` | pass (reused) |
| `TestRegenerateMatchesCheckedPatchForPinnedArchive` (`toolchain`) | `test-host`, `test-builder` | **skip** (reused); proves nothing for Go 1.27.1 until task 4 retargets it |
| `random-seed-<seed>-golden-random-0` (3) and `random-seed-<seed>-repeatable-<n>` (300) | `test-runtime` | pass (fresh) |
| `gotest-seed-<seed>-repeatable-<n>` (300), `gotest-stock-compatibility`, `gotest-custom-compatibility` (seeded environment fixture, `TZ=UTC`) | `test-runtime` | pass (fresh) |
| `activation-*` (18 families, including `activation-disabled`, `activation-explicit-disabled`, `activation-io-direct`, `activation-io-disabled`, and 8 invalid-seed cases) | `test-runtime` | pass (fresh) |
| `clock-*` (including `clock-deadlock`, `clock-synctest`, `clock-blocking-io`, `clock-disabled`), `scheduler-*`, `runqueue-*`, `select-*`, `preemption-enabled`, `preemption-disabled` | `test-runtime` | pass (fresh) |
| `upstream-clock` (`runtime`, including `TestSizeof`; `time`; `testing/synctest`) | `test-upstream` | pass (fresh) |
| Upstream `crypto/rand` (317 tests) and `syscall` (30 tests; `TestForeground`, `TestForegroundSignal`, `TestRlimitRestored` skip) | none; task 3 names the command | pass (fresh); [dispositions](../../../.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-upstream-crypto-rand-syscall-dispositions.tsv) |

The runtime tier has no case that runs the `io_entropy` fixture. Enabled entropy coverage at the baseline is the one `test-host` test above.

**Qualification per workload.** Every row was `qualified` against a darwin/arm64 expectation of `qualified`, with `replayed`, `choice_replay_exact`, and `replay_match` all true on every listed seed. The row listings with evidence digests are `smoke-`, `core-`, and `temporal-qualification-rows.tsv` in `tools/gomad3/.toolchain/fn-110/baseline/` and in [final-gates](../../../.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final-gates/).

| Set | Seeds | Workloads, all `qualified` with exact replay |
| --- | --- | --- |
| Smoke (`smoke.json`) | 11 | `functional-activity`, `functional-child-workflow`, `functional-update`, `user-timers-workflow` |
| Core (`qualification/core.json`) | 17 | `concurrency-state-invariant`, `filesystem-transaction`, `loopback-tcp-roundtrip`, `modernc-libc-boundary`, `mount-reads-under-collection`, `sqlite-transaction`, `sqlite-write-ahead-log` |
| Temporal (`temporal.json`) | 11 and 17 | `activity-batch-cancel-boundary`, `clock-context-timeout`, `frontend-system-info`, `functional-activity`, `functional-cancel`, `functional-child-workflow`, `functional-continue-as-new`, `functional-cron`, `functional-query`, `functional-signal-chasm`, `functional-timer`, `functional-update`, `functional-workflow`, `future-suite`, `sqlite-persistence-boundary`, `temporal-backoff-overflow`, `temporal-cache-concurrent`, `temporal-dither-pass`, `temporal-map-concurrent`, `temporal-poller-history`, `temporal-queue-key`, `temporal-sqlite-schema-rewrite`, `temporal-transition-history`, `temporal-update-abort-matrix`, `temporal-version-set-merge`, `temporal-workflow-backoff`, `timer-local-gate`, `user-timers-workflow` |

Report digests: smoke `b25ac128…dfa964433d`, core `e82a5afe…bca27f0be`, Temporal `2bbb4659…bf4cc290d7`. Every `evidence_sha256` in the reports covers the Runner build, so a rebuilt Runner changes those digests without any change in target behavior. Compare classifications and replay fields across builds.

### D12 and D14 dispositions

D12 is open and D14 is fixed in this baseline. [MILESTONES.md](../../../MILESTONES.md) states them as follows.

> **Intermittent suites.** The F5/F6 suites and `TestSignalWorkflowTestSuiteChasm` on linux (F10 D12). The darwin divergence of `TestSignalWorkflowTestSuiteChasm` is fixed (F10 D14, 2026-09-30): the generated manifest expects it `qualified` on darwin/arm64 and `intermittent` on linux/amd64. The linux `intermittent` expectation for the F5/F6 suites is stated in the representative and smoke manifests (`temporal.json`, `smoke.json`); the generated manifest still expects those suites `qualified` on linux, where the full set is not run as a gate.

> **Linux replay divergence (F10 D12).** About one tier-3 seed-run in 26 on linux/amd64 is nondeterministic or diverges on replay, on either seed and a different suite each run, at choice ordinals from 8 to ~85k. […] The F5 and F6 suites are `intermittent` on linux, and both the dispatch-only linux gate and the required smoke gate accept `nondeterministic` and `replay_divergence` for them while failing on target failures, unsupported targets, and infrastructure errors. The darwin representative set stays fully qualified.

> D12 is a required fix that needs a linux/amd64 host; the D14 lock-profile fix is an unverified candidate for it.

> The F10 items D1, D2, D13, D14, and D21–D25 are complete as well and were removed by 2026-10-01; their outcomes are in the done summaries of the fn-105 tasks under `.flow/tasks/`.

The D12 row of the F10 table reads: "Must be fixed, decided 2026-09-30. Native Linux instrumentation is an execution prerequisite. R12 in fn-105 requires a regression reproducer, repeated exact replay on both seeds under load, and restoration of strict CI expectations; diagnosis alone cannot close the task."

Manifest expectations, verbatim. `temporal.json` has SHA-256 `31d17baf…c39433aa6` and 28 suites. `smoke.json` has SHA-256 `cc676039…4437c18c1` and 4 suites. `core.json` has SHA-256 `048222eb…a94ef9a43a` and 7 suites.

The `functional-signal-chasm` suite (`TestSignalWorkflowTestSuiteChasm`) in `temporal.json`:

```json
"expectation": {"classification": "intermittent", "finding": "MILESTONES.md#f6-a-package-level-functional-slice"},
"platform_expectations": {
  "darwin/arm64": {"classification": "qualified"},
  "linux/amd64": {"classification": "intermittent", "finding": "MILESTONES.md#f7-any-functional-test-and-ci"}
}
```

The same test in the generated `tests.json`:

```json
"expectation": {
  "classification": "qualified"
},
"platform_expectations": {
  "linux/amd64": {
    "classification": "intermittent",
    "finding": "MILESTONES.md#f7-any-functional-test-and-ci"
  }
}
```

Expectation forms across the gated manifests:

| Manifest | Suites | `expectation` | `platform_expectations` |
| --- | ---: | --- | --- |
| `temporal.json` | 10 | `intermittent`, finding `#f6-a-package-level-functional-slice` | darwin/arm64 `qualified`; linux/amd64 `intermittent`, finding `#f7-any-functional-test-and-ci` |
| `temporal.json` | 2 | `intermittent`, finding `#f5-one-workflow-executing-functional-test-deterministic` | darwin/arm64 `qualified`; linux/amd64 `intermittent`, finding `#f7-any-functional-test-and-ci` |
| `temporal.json` | 1 | `qualified` | darwin/arm64 `qualified`; linux/amd64 `intermittent`, finding `#f3-qualify-the-existing-functional-probe` |
| `temporal.json` | 10 | `qualified` | linux/amd64 `unsupported_target`, capability `foreign:assembly:xxhash_amd64.s`, import path `github.com/cespare/xxhash/v2` |
| `temporal.json` | 5 | `qualified` | none |
| `smoke.json` | 3 | `intermittent`, finding `#f6-a-package-level-functional-slice` | darwin/arm64 `qualified`; linux/amd64 `intermittent`, finding `#f7-any-functional-test-and-ci` |
| `smoke.json` | 1 | `intermittent`, finding `#f5-one-workflow-executing-functional-test-deterministic` | darwin/arm64 `qualified`; linux/amd64 `intermittent`, finding `#f7-any-functional-test-and-ci` |
| `core.json` | 7 | `qualified` | none |

Later fn-110 tasks keep these expectations unchanged. A darwin result weaker than `qualified` for any of the 28 Temporal, 4 smoke, or 7 core workloads is a regression against this baseline.

### Blocking and drifting inputs

| Input | State | Effect on later claims |
| --- | --- | --- |
| Pinned archive | present; SHA-256 equals the descriptor | none |
| Patch | SHA-256 recorded; reproduces byte-for-byte from the archive at `-U3` | none; a different patch SHA-256 before task 2 starts invalidates this section |
| `tools/gomad3/toolchain` tree | clean; identical at `6782b55f4` and `38957053f` | none |
| Active build | key `8d28bd44…` matches the inputs | none |
| Process-simulation test | **fails at the baseline**: 2 of 11 subtests fail in every run, and 2 of 25 runs also hit a 30 s watchdog timeout in one process-backend subtest | **Blocks any claim that this test passes.** Tasks 2 and 5 compare per subtest against the table above. The two identity failures must stay the same two failures with the same messages. One watchdog timeout in a single run does not show a regression and its absence does not show equivalence; repeat the run at least ten times on a quiet host before judging a process-backend subtest. Exploration-plan consumption over the Runner transport has no passing coverage at the baseline. |
| Runner source | **drifted after the reused gates, and still changing.** The reused gates ran on the source now committed as `38957053f`. Another session then left uncommitted edits in `tools/gomad3/runner/coordinator.go` (modified 2026-10-02T00:06 UTC), `tools/gomad3/target/capability.go` and `capability_test.go` (00:34 UTC), and new untracked tests under `tools/gomad3/runner/`. `tools/gomad3/.bin/gomad` is still the `f8b0a8d4…` build, which no longer matches the working tree. | **Blocks a before/after comparison across different Runner sources.** `make -C tools/gomad3 test` compiles the Runner, and the integration, smoke, core, and Temporal qualification targets rebuild it, so an "after" run on a changed Runner differs from this baseline in two inputs. Task 5 has two valid routes. Either the Runner source of the "after" run equals `38957053f` for `tools/gomad3`, `tools/gomad3sim`, and `tools/gomad3integration` outside `tools/gomad3/toolchain`, or task 5 reruns the "before" gates on its own Runner source with the retained baseline toolchain `8d28bd44…` and compares against that rerun. The size figures and the toolchain identity do not depend on the Runner. The fresh results in this section ran before 00:17 UTC, when their dependency closure equaled `38957053f`: package `runner` is outside the closure of `runner/internal/execution`, `internal/gomadtool/conformance`, and `toolchain`, and `target/capability.go` changed after the last fresh run. |
| Host load | another session ran test suites during both the reused gates (1-minute load average 4 to 32) and the fresh runs (3 to 19) | no gate other than the two process-simulation runs reported a watchdog timeout |
| linux/amd64 | **not run** | blocks every both-platform claim; see below |

### Linux baseline

No linux/amd64 gate ran for this baseline. No Linux host is available locally, and cross-compilation is not Linux evidence. The Linux `-U3` application of the patch is also unverified. The jobs that supply a Linux baseline run in GitHub Actions after a push: `host-tools-linux` and `core-linux` in [gomad3.yml](../../../.github/workflows/gomad3.yml), and `functional-smoke-linux` in [gomad3-smoke.yml](../../../.github/workflows/gomad3-smoke.yml). Until a run of those jobs on this baseline revision exists, every Linux comparison in fn-110 is incomplete.

### Disk and build directories

`df -h .` reported 28 GiB free of 228 GiB at 2026-10-02T00:16 UTC. `tools/gomad3/.toolchain/builds/` holds one directory, the baseline key `8d28bd44…`, at 6.4 GiB. The whole `tools/gomad3/.toolchain` directory is 7.1 GiB. Each candidate build adds a directory of about that size, so three more builds would leave about 8.8 GiB, just above the spec's 8 GiB floor. Remove a superseded fn-110 candidate before a fourth. Keep the baseline directory, because task 2 runs fixtures on it for comparison. `make -C tools/gomad3 prune-cache` deletes build directories other than the active key, and downloads, once they are older than its age limit, so it can remove the baseline build after another key becomes active.

### Evidence

Tracked, under `.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/`: `task1-measure.sh`, `task1-measurements.txt`, `task1-process-simulation.tsv`, `task1-test-runtime-case-families.tsv`, `task1-test-upstream-cases.tsv`, `task1-test-upstream-stdout.txt`, `task1-upstream-crypto-rand-syscall-dispositions.tsv`, `task1-casereport-main.go.txt`, and `task1-baseline.json`, which indexes the gitignored files with their digests.

Gitignored, under `tools/gomad3/.toolchain/fn-110/baseline/`: `identity.json` and the three set reports and row listings from the fn-108 gates, `test-runtime-cases.tsv`, `test-upstream-cases.tsv`, `upstream-case-output/`, `upstream-crypto-rand-syscall-dispositions.tsv`, `casereport/main.go`, and `logs/` with the toolchain, validate, case-report, `crypto/rand`, and 25 process-simulation logs. `make -C tools/gomad3 clean-qualifications` and `make -C tools/gomad3 clean` leave this directory in place.
