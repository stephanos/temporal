---
satisfies: [R3, R5, R6]
---
# fn-110-gomad-minimize-the-runtime-patch.3 Relocate crypto initialization and syscall declarations to overlays

## Description
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
- No `git commit`, `git add`, stash, or worktrees. The user owns commits; leave changes in the working tree and report them. Earlier fn-110 tasks may therefore be uncommitted working-tree changes — do not revert them.
- Host is `darwin/arm64` only. `linux/amd64` gates cannot run here: record every Linux-dependent check as **incomplete**, never as passing. Cross-compilation is not Linux evidence.
- Disk: about 19 GB was free at planning time and each toolchain build directory under `tools/gomad3/.toolchain/builds/<key>` is 2–6 GB. Run `df -h .` before every rebuild and stop if less than 8 GB is free. Only delete build directories that this spec's own intermediate candidates created, once superseded and not referenced by retained evidence. Never delete the baseline key recorded by task 1 or the active key in `.toolchain/build-key`. Pre-existing directories and `make clean-qualifications` need the user's confirmation.
- Patch and overlay bytes feed the build key (`tools/gomad3/toolchain/buildkey.go:48-58`), so every patch or overlay edit yields a new toolchain identity. Never relabel old artifacts.
- fn-105 D12 (Linux replay divergence) and D14 (Darwin `TestSignalWorkflowTestSuiteChasm`) keep their owners and dispositions. Do not edit qualification expectations to get a passing gate.
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
- [ ] The patch has no `crypto/rand` section and materializes `rand.go` byte-identical to the archive; the overlay `init` keeps both reader assignments and its comment verbatim
- [ ] The three syscall declarations exist only in the overlay with unchanged symbols and signatures; the `copyenv` reset and both `Write` hooks remain at their patched positions; no build configuration lost a declaration, and the constraint choice is recorded
- [ ] Enabled entropy/transcript tests cover `rand.Read`, `rand.Text`, and key generation; seeded environment, output, disabled-mode, and unmodeled-write denial results match the baseline
- [ ] `version.json` allowlists equal the patch and overlay trees; `make -C tools/gomad3 generate validate` is clean and the toolchain build's archive collision check passes with unrelaxed policy
- [ ] Final `-U3` patch and overlay sizes are recorded separately and the `-U3` patch copy is saved for task 4
- [ ] Linux execution is recorded as incomplete


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
