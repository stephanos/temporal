---
satisfies: [R4, R5]
---
# fn-110-gomad-minimize-the-runtime-patch.4 Emit the canonical one-context-line patch and pin regeneration to the descriptor

## Description
Change the patch representation only: make the governed regenerator emit one context line, regenerate the checked-in patch from the final extracted source, and make the pinned regeneration check follow the release descriptor. Kept apart from the extraction tasks so the `-U3`/`-U1` comparison isolates representation from source changes.

**Size:** M
**Files:** `tools/gomad3/toolchain/patch_regenerate.go`, `tools/gomad3/toolchain/patch_test.go`, `tools/gomad3/toolchain/runtime/go1.27.1.patch`, generated consumers if `make generate` rewrites any
**Touches:** [tools/gomad3/toolchain/patch_regenerate.go, tools/gomad3/toolchain/patch_test.go, tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/version/**, tools/gomad3/version_generated.mk, docs/research/gomad/GOMAD_PATCH_SIZE.md]

### Approach
- **Regenerator** — the `git diff` argv in `RegeneratePatch` (`patch_regenerate.go:76-78`) uses Git's default context. Make one context line explicit there. Keep `--no-ext-diff`, the `a/`/`b/` prefixes, plain text output, `validatePatch`, the `git apply --cached --check` step, and the zero-fuzz `patch -F 0` materialization (`patch.go:68-72`). No `patch-regenerate` flag for context and no compression (`cmd/gomadtool/main.go:202-232` stays as is).
- **Pinned check** — `TestRegenerateMatchesCheckedPatchForPinnedArchive` (`patch_test.go:335-373`) hardcodes `go1.26.4.src.tar.gz` and `toolchain/runtime/go1.26.4.patch`; neither exists, so it always skips. Read the archive name and patch path from the descriptor (`gomadversion.Load`, `toolchain/version/descriptor.go:62`). It must execute whenever the verified archive is cached (it is at `tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz`; the `test-toolchain` tier depends on `toolchain`, which fetches it) and must fail on a checksum mismatch rather than skip. The synthetic fixtures elsewhere in the file that use `go1.26.4` as made-up data are not the obsolete pin; leave them.
- **Tests to add or extend** — regenerated output has one context line per side of each hunk; repeated regeneration is byte-identical (`TestRegeneratePublishesDeterministicExactPatch` already covers the synthetic case); for the pinned archive, a three-context and a one-context patch of the same candidate both apply with zero fuzz and materialize identical trees. Reach the three-context form through an unexported seam, not a CLI option. Putting this in the `toolchain` package makes Linux CI execute it. Existing rejection tests (malformed, unlisted path, fuzz-dependent, no changes, wrong version) keep passing unmodified.
- **Regenerate** — materialize `tools/gomad3/.toolchain/fn-110/final-U3.patch` (from task 3) into a fresh extraction, regenerate with the changed command, and run `make -C tools/gomad3 generate validate`. Allowlists do not change here.
- **Equivalence on this host** — apply the saved `-U3` patch and the new `-U1` patch to two fresh extractions with the builder's commands and `diff -r` the trees (expect no difference across all 19 patched files); regenerate twice and `cmp` the outputs. Record SHA-256, bytes, and lines for both.
- Rebuild the toolchain (new key) and run the toolchain and builder tiers; this also exercises the archive-based collision check.

`git diff` output can be influenced by ambient Git configuration (`diff.context`, `diff.algorithm`); the explicit context removes one such dependency. Do not widen scope to others unless a test shows drift.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/patch_regenerate.go:17-100,273-320`
- `tools/gomad3/toolchain/patch_test.go:194-243,325-373`
- `tools/gomad3/toolchain/patch.go:37-135` — validation and materialization
- `tools/gomad3/toolchain/version/descriptor.go:62-112`
- `docs/research/gomad/GOMAD_PATCH_SIZE.md` — "Measured reduction options", "Qualification and reproducibility"

### Quick commands
```bash
(cd tools/gomad3 && GOWORK=off go test -tags test_dep -count=1 -run 'TestRegenerate|TestMaterialize|TestValidate' -v ./toolchain | grep -E '^(=== RUN|--- |ok|FAIL)')   # the pinned test must show PASS, not SKIP
make -C tools/gomad3 generate validate
df -h . && make -C tools/gomad3 toolchain && make -C tools/gomad3 test-toolchain test-builder
wc -c -l tools/gomad3/toolchain/runtime/go1.27.1.patch tools/gomad3/.toolchain/fn-110/final-U3.patch
```

### Key context

**Working constraints (apply to every fn-110 task):**
- Commit verified progress in bounded batches, as requested by the user. The conductor owns staging and commits; preserve unrelated changes and leave active shared-source edits unstaged until their verification boundary. Do not push, stash, create worktrees or rewrite history without separate authorization.
- Recheck the actual host before gates. This development session is `linux/arm64`; neither qualified native `darwin/arm64` nor native `linux/amd64` execution is available here. Keep each platform gate incomplete until source-bound native evidence exists; cross-compilation, emulation and developmental stock-host checks qualify neither.
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
- [ ] `patch-regenerate` emits one context line with no new flag; two regenerations from the same candidate are byte-identical and equal the checked-in patch
- [ ] The pinned regeneration test derives archive and patch from the descriptor, reports PASS (not SKIP) on this host, and fails on checksum or version mismatch
- [ ] On darwin/arm64 the final `-U3` and `-U1` patches apply with zero fuzz and materialize byte-identical files; the same check exists as a `toolchain` package test for Linux CI
- [ ] Malformed, unexpected-path, fuzz-only, no-change, and stale-generated-output cases are still rejected by unmodified negative tests
- [ ] Allowlists match the final inputs; `make -C tools/gomad3 generate validate`, `test-toolchain`, and `test-builder` pass on the rebuilt toolchain
- [ ] `-U1` is smaller than the final `-U3`; both sizes and digests are recorded; linux/amd64 equivalence is recorded as incomplete


## Done summary
Blocked:
Blocked: implementation, local developmental verification, and review (SHIP) are complete; only native darwin/arm64 and linux/amd64 gates remain.

Done (commit fab9378c79 on gomad-fn110):
- RegeneratePatch passes --unified=1 explicitly through canonicalPatchContext. No CLI flag was added, and the canonical headers, validatePatch, git apply --check, and zero-fuzz materialization are unchanged. The unexported regeneratePatch seam gives tests the -U3 form.
- The checked-in patch was regenerated from the final task 3 source: -U3 38,362 B / 1,112 lines (sha256 86def26a…76ea5c) to -U1 29,015 B / 778 lines (sha256 8497f885…6a90b), -24.4% bytes. Repeated regeneration is byte-identical. Allowlists are unchanged. The choice implementation digest was regenerated. make generate validate is clean without the shim.
- TestRegenerateMatchesCheckedPatchForPinnedArchive reads the archive and patch from the descriptor, fails on a checksum mismatch (TestPinnedArchiveFollowsDescriptorAndRejectsChecksumMismatch), and requires byte-identical repeats. It went red before regeneration and green after, with PASS rather than SKIP. TestPinnedContextRepresentationsMaterializeIdenticalSource proves -U3/-U1 zero-fuzz file equivalence in the toolchain package for Linux CI. TestRegenerateEmitsOneContextLine covers the synthetic hunk shape. The negative tests are unmodified and pass.

Local evidence (linux/arm64 development host; the uncommitted shim was used only for builds and shim gates):
- -U3 and -U1 applied with builder zero-fuzz commands to fresh extractions; diff -r found the trees identical.
- Toolchain rebuild key 60e4051c…, 191 s, exit 0, collision check passed.
- No-shim committed state: go test ./toolchain PASS (355 s) and make test-builder PASS.
- With the shim, test-toolchain and test-builder fail only on the linux/arm64 inventory tests, and the baseline fails identically. test-host failures are a subset of the baseline+shim failures.
- Evidence file: .flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task4-canonical-patch-evidence.md.

Non-blocking review notes (P3, not applied): diff.interHunkContext would be caught by the pinned test as drift, and the spec says not to widen scope until a test shows drift. The two pinned tests overlap in extraction cost, and one helper could be shared.

Remaining native gates: -U3/-U1 zero-fuzz equivalence plus toolchain rebuild, test-toolchain (pinned tests), and test-builder on darwin/arm64 and linux/amd64. linux/amd64 equivalence is incomplete.
## Evidence
- Commits:
- Tests:
- PRs:
