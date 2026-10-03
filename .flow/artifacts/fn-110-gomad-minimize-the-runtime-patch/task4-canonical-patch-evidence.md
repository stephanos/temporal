# fn-110.4 canonical -U1 patch: local developmental evidence

Host: linux/arm64 (development only). Builds and the shim-dependent gates used the
uncommitted linux/arm64 descriptor shim. The committed state was checked without the
shim.

| Patch | Bytes | Lines | Hunks | Added/deleted | SHA-256 |
| --- | ---: | ---: | ---: | --- | --- |
| Final -U3 (fn-110.3) | 38,362 | 1,112 | 81 | 318 / 103 (20 files) | 86def26a7f4d0b5c494a6a031c87bec284f7e76c91dcf437bc23fcc4c276ea5c |
| Canonical -U1 | 29,015 | 778 | 90 | 318 / 103 (20 files) | 8497f8855011f13fb46ad36a02448d165d4bd65688ef00eed6ae09822306a90b |

- patch-regenerate ran twice (with --output) and once canonically (to the descriptor path) from a fresh extraction with the final -U3 patch applied. All three outputs were byte-identical.
- Builder zero-fuzz materialization of -U3 and -U1 into separate fresh extractions: `diff -r` found no differences, and 20 files differ from pristine.
- `go test -run 'TestRegenerate|TestMaterialize|TestValidate|TestPinned' ./toolchain`: TestRegenerateMatchesCheckedPatchForPinnedArchive FAILED before regeneration (red) and PASSED after (green). TestPinnedContextRepresentationsMaterializeIdenticalSource, TestRegenerateEmitsOneContextLine, and TestPinnedArchiveFollowsDescriptorAndRejectsChecksumMismatch PASS. The negative tests are unmodified and PASS.
- Toolchain rebuild (shim): key 60e4051cb0383eecae727065e8f0bc7d79d8567804304d4d7dc7dbbed425ba04, 191 s, exit 0. The archive collision check passed.
- Committed state without the shim: make generate validate is clean in the worktree. `go test ./toolchain` against the built source PASS (355 s), with the pinned tests PASS rather than SKIP. make test-builder PASS.
- With the shim: test-toolchain and test-builder fail only on TestPatchedRuntime{HostClock,Draw}ReferencesAreReviewed (linux/arm64 inventory), which also fail on the baseline with the shim. test-host failures are a subset of the baseline+shim failures, with no new failures.
- Native darwin/arm64 and linux/amd64 equivalence and gates: incomplete.
