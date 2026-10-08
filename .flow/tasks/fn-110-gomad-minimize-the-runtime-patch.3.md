---
satisfies: [R3, R5, R6]
---
# fn-110-gomad-minimize-the-runtime-patch.3 Relocate crypto initialization and syscall declarations to overlays

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Representation/overlay implementation, byte equivalence, pinned checks, preservation, review, Darwin gates and unmet R8 size reduction. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Relocate the crypto random-reader initialization and the three syscall linkname declarations into additive overlay files, restoring pristine `crypto/rand/rand.go`. Because this removes a patched path and adds two overlay paths, the descriptor's exact source sets and its generated consumers change in the same task. The patch stays at three context lines; its result is the final `-U3` candidate that task 4 compares against.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/go1.27.1.patch`, new `tools/gomad3/toolchain/runtime/overlay/src/crypto/rand/gomad.go`, new `tools/gomad3/toolchain/runtime/overlay/src/syscall/gomad_unix.go`, `tools/gomad3/toolchain/version/version.json`, generated consumers, `docs/research/gomad/GOMAD_PATCH_SIZE.md`
**Touches:** [tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/crypto/rand/**, tools/gomad3/toolchain/runtime/overlay/src/syscall/**, tools/gomad3/toolchain/version/version.json, tools/gomad3/toolchain/version/generated.go, tools/gomad3/version_generated.mk, tools/gomad3/deterministicio/boundary/**, docs/research/gomad/GOMAD_PATCH_SIZE.md]

### Approach
Baseline patch line numbers; re-find by file header if shifted.
- **Crypto** — the whole `src/crypto/rand/rand.go` section, patch lines 53-80. Move the `init` function and its comment verbatim into the new overlay file (package `rand`, importing `crypto/internal/rand` and `internal/gomadio`), keeping both assignments: `Reader = gomadio.RandomReader()` and `rand.SetTestingReader(Reader)`. Restore the candidate's `rand.go` to the archive bytes so the section disappears from the patch.
- **Syscall declarations** — remove from the patch: the `_ "unsafe"` import and `gomadDeterministicEnabled` declaration in `env_unix.go` (patch 868-881), and the `gomadCapabilityGuard`/`gomadWrite` declarations in `syscall_unix.go` (patch 946-958). Declare all three in the new overlay file with its own `unsafe` import, identical names, linkname targets (`runtime.gomadDeterministicEnabled`, `runtime.gomadCapabilityGuard`, `runtime.gomadSyscallWrite`), and signatures.
- **Stay in the patch:** the lazy `envs = []string{"TZ=UTC"}` reset inside `copyenv` (patch 882-890), both `Write` body hooks (patch 924-945), and the `rlimit.go` section (891-919).
- **Build constraints — resolve before writing the file.** Upstream `env_unix.go` builds under `unix || (js && wasm) || plan9 || wasip1`; `syscall_unix.go` builds under `unix`. No configuration may lose a declaration it has today, so a single `//go:build unix` file would break `env_unix.go` on js/wasm, plan9, and wasip1. Either give each declaration group a file with its origin's constraint or use one file whose constraint covers `env_unix.go`'s set; record the choice and why. Only the two supported hosts are qualified, but source selection must stay correct.
- **Descriptor** — in `version.json` remove `src/crypto/rand/rand.go` from `patch_allowlist` and add the new overlay paths to `overlay_allowlist` in sorted position (`src/crypto/rand/…` sorts after `src/cmd/link/internal/ld/gomadcap.go`; `src/syscall/…` sorts after `src/runtime/gomad_iowire_generated.go`). Run `make -C tools/gomad3 generate` and keep whatever consumers it rewrites. Do not relax `validateOverlay`, `prohibitedPath`, or the collision check.
- **Initialization timing** — the archive has no other non-test `init` in `crypto/rand`; `Reader` is still initialized before the relocated `init` runs. Confirm rather than assume, since file order affects same-package init order.

Behavior pin: enabled entropy and transcript coverage through `tools/gomad3/internal/gomadtool/conformance/testdata/io_entropy/main.go` (`rand.Read`, `rand.Text`, ECDSA key generation through the FIPS override) and `TestProfileEntropyIsIndependentOfScheduleSeed` in `tools/gomad3/runner/internal/execution/io_entropy_toolchain_test.go`; seeded environment fixture `testdata/gotest` (seed 17 observes exactly `TZ=UTC`) and its disabled-mode counterpart; unmodeled-write denial through the existing `io_failure`/`io_fd5` fixtures; upstream `crypto/rand` and `syscall` tests in disabled mode. Compare outputs against the baseline toolchain for the same seeds.

Record the new `-U3` bytes/lines and the overlay bytes/lines added (investigation estimate: 1,371 bytes and 54 lines off the patch; moved code is still reported as overlay cost). Save a copy of this final `-U3` patch at `tools/gomad3/.toolchain/fn-110/final-U3.patch` for task 4.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:53-80,864-958`
- `docs/research/gomad/GOMAD_PATCH_SIZE.md` — "Crypto initialization", "Syscall declarations"
- `tools/gomad3/toolchain/version/descriptor.go:83-160,270-282` — exact-set validation and generated artifacts
- `tools/gomad3/toolchain/patch.go:191-260`, `tools/gomad3/toolchain/build.go:373` — overlay validation, collision check
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:174,983,1220` — linkname targets

### Quick commands
```bash
make -C tools/gomad3 generate validate
df -h . && make -C tools/gomad3 toolchain
make -C tools/gomad3 test-toolchain test-builder test-live-capability test-runtime test-upstream
(cd tools/gomad3 && GOWORK=off .toolchain/bin/go test -tags test_dep -count=1 crypto/rand syscall)
grep -c 'crypto/rand' tools/gomad3/toolchain/runtime/go1.27.1.patch   # expect 0
```

### Key context

**Working constraints (apply to every fn-110 task):**
- Commit verified progress in bounded batches, as requested by the user. The conductor owns staging and commits; preserve unrelated changes and leave active shared-source edits unstaged until their verification boundary. Do not push, stash, create worktrees or rewrite history without separate authorization.
- Recheck the actual host before gates. This development session is `linux/arm64`; neither qualified native `darwin/arm64` nor native `linux/amd64` execution is available here. Keep source-owned Darwin gates incomplete until source-bound native evidence exists; Linux execution belongs to fn-128.1/.4/.7; cross-compilation, emulation and developmental stock-host checks qualify neither.
- Disk: about 19 GB was free at planning time and each toolchain build directory under `tools/gomad3/.toolchain/builds/<key>` is 2–6 GB. Run `df -h .` before every rebuild and stop if less than 8 GB is free. Only delete build directories that this spec's own intermediate candidates created, once superseded and not referenced by retained evidence. Never delete the baseline key recorded by task 1 or the active key in `.toolchain/build-key`. Pre-existing directories and `make clean-qualifications` need the user's confirmation.
- Patch and overlay bytes feed the build key (`tools/gomad3/toolchain/buildkey.go:48-58`), so every patch or overlay edit yields a new toolchain identity. Never relabel old artifacts.
- fn-128.2 owns the transferred D12 Linux replay fix; fn-105 D14 retains Darwin `TestSignalWorkflowTestSuiteChasm`. Keep their existing dispositions until their own qualification passes. Do not edit qualification expectations to get a passing gate.
- Existing comments move with their code, unchanged. Add no allocations, host reads, dependencies, CLI flags, or capability grants.
- Always pass `-tags test_dep`. In testify code use `require`, not `assert`; plain `testing` files keep their existing style.

**Patch editing workflow (governed, no hand-edited hunks):**
1. Extract the verified archive `tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz` (SHA-256 in `toolchain/version/version.json`) into a scratch directory under the gitignored `tools/gomad3/.toolchain/fn-110/`.
2. `go -C tools/gomad3 run ./cmd/gomadtool patch-materialize --root="$PWD/tools/gomad3" --source-root=<scratch>/go`
3. Edit upstream files in `<scratch>/go/src`. Do not copy overlay files into the candidate: `changedFiles` in `toolchain/patch_regenerate.go` rejects added source paths.
4. `go -C tools/gomad3 run ./cmd/gomadtool patch-regenerate --root="$PWD/tools/gomad3" --candidate-root=<scratch>/go`
5. `make -C tools/gomad3 generate validate`, then `make -C tools/gomad3 toolchain` (the build runs the archive-based overlay collision check at `toolchain/build.go:176`).

The descriptor requires `patch_allowlist` and `overlay_allowlist` to equal the checked trees exactly (`toolchain/version/descriptor.go:114-160`), so a task that adds an overlay file or empties a patched file updates `version.json` and regenerates in the same task.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] The patch has no `crypto/rand` section and materializes `rand.go` byte-identical to the archive; the overlay `init` keeps both reader assignments and its comment verbatim
- [ ] The three syscall declarations exist only in the overlay with unchanged symbols and signatures; the `copyenv` reset and both `Write` hooks remain at their patched positions; no build configuration lost a declaration, and the constraint choice is recorded
- [ ] Enabled entropy/transcript tests cover `rand.Read`, `rand.Text`, and key generation; seeded environment, output, disabled-mode, and unmodeled-write denial results match the baseline
- [ ] `version.json` allowlists equal the patch and overlay trees; `make -C tools/gomad3 generate validate` is clean and the toolchain build's archive collision check passes with unrelaxed policy
- [ ] Final `-U3` patch and overlay sizes are recorded separately and the `-U3` patch copy is saved for task 4
- [ ] Linux execution is recorded as incomplete


## Done summary
Blocked:
Blocked: implementation, local developmental verification, and review (SHIP) are complete; only native darwin/arm64 and linux/amd64 gates remain.

Done (commit 91ce1111ed on gomad-fn110):
- crypto/rand init and its comment moved verbatim to overlay src/crypto/rand/gomad.go, with both Reader assignments kept. The patch has no crypto/rand section, and the materialized rand.go is byte-identical to the archive.
- Four syscall linkname declarations moved to overlays. The fourth, gomadIOProfileEnabled, landed after planning and moved with the others. gomad_env_unix.go uses env_unix.go's constraint; gomad_unix.go uses unix. Splitting by origin keeps source selection identical: go list selects each file only where its origin file builds. The copyenv reset, both Write hooks, and rlimit.go stay in the patch.
- version.json allowlists updated. make generate rewrote only the choice-wire ImplementationSourceSHA256, because the patch is an input. make generate validate is clean without the shim.
- Patch -U3 went from 39,837 B / 1,169 lines to 38,362 B / 1,112 lines (-1,475 B / -57 lines; sha256 86def26a7f4d0b5c494a6a031c87bec284f7e76c91dcf437bc23fcc4c276ea5c). New overlay files add 1,520 B / 55 lines. A copy of the -U3 patch is saved at tools/gomad3/.toolchain/fn-110/final-U3.patch.

Local evidence (linux/arm64 development host only; built with an uncommitted descriptor shim that adds linux/arm64; never committed):
- Candidate toolchain key 6879442c… built in 225 s with exit 0; the archive overlay collision check passed. Baseline (HEAD 331b75bb6 plus the shim) key 464561b5… built in 359 s.
- Baseline and candidate gave byte-identical results for: enabled-profile io_entropy (rand.Read, rand.Text, and ECDSA through the FIPS override) with seeds 1, 11, 17, and 999; profile environment; seeded env (exactly TZ=UTC) and Write(9) GOMAD_CAPABILITY_DENIED with seeds 1 and 17; disabled-mode env and EBADF; stdout/stderr writes; gotest seed 17 and TestDisabledCompatibility; and disabled-mode upstream crypto/rand and syscall dispositions (378 cases). TestPrlimitFileLimit fails on both trees: it is Linux-only, and the failure comes from the existing rlimit hunk.
- Candidate: TestProfileEntropyIsIndependentOfScheduleSeed, TestToolchainLeavesFD5ForProcessesWithoutIOProfile, and TestIOProfileFailureArtifactReplaysExactly pass. livecap TestPinnedToolchain* skip because they are darwin-only.
- `go test ./toolchain` passes against the committed (no-shim) descriptor and the candidate build source (99 s). make test-builder passes with the no-shim descriptor. With the shim, test-toolchain and test-builder fail only on linux/arm64 inventory entries, and the baseline fails identically.
- test-host (350 s, exit 2): its 31 failures match baseline+shim (30 shared developmental host/shim failures). The exception is TestRunIOTerminalAfterTermination/watchdog_checksum ("supervisor could not be reaped after deadline" under load), which passed on 2 reruns.
- The evidence file is .flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task3-relocation-evidence.md.

Remaining native gates: toolchain build plus test-toolchain, test-builder, test-host, test-live-capability (env and Write guard), test-runtime, and test-upstream (crypto/rand, syscall) on darwin/arm64 and linux/amd64, compared with the fn-110.1 baseline dispositions. Linux execution is incomplete.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Representation/overlay implementation, byte equivalence, pinned checks, preservation, review, Darwin gates and unmet R8 size reduction. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
R8 remains independently unmet: baseline U3 patch 32,652 bytes, final U3 38,362 bytes (5,710-byte increase); U1 context reduction does not satisfy the extraction-size reduction.
