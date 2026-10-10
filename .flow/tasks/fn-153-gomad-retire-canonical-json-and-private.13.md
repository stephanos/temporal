---
satisfies: [R1]
---
# fn-153-gomad-retire-canonical-json-and-private.13 Move streamed archive and validated patch publication to hostfs

## Description
Implements R1 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** toolchain/{source,patch_regenerate}.go and focused publication/fault tests (2 core publishers).
**Touches:** [tools/gomad3/toolchain/source*.go, tools/gomad3/toolchain/patch_regenerate.go, tools/gomad3/toolchain/patch_test.go, tools/gomad3/toolchain/patch_cleanup_test.go]

### Approach

- Source download fills the shared stage through existing context-aware limited copy and SHA256 checks. Reject HTTP, overflow and checksum failure before rename; keep streamed memory and body/file/cleanup errors.
- Patch regeneration resolves the same allowed output, writes/syncs/closes through StageContext, runs validatePatch and git apply --cached --check on the exact staged Path, then publishes through the shared owner.
- Keep checksum/patch pins, destination guards and primary error precedence. A private helper for rename/directory-sync would violate R1.
- Behavior pin: archive cache/malformed/oversized/checksum fixtures and patch invalid-candidate controls, injected pre/post-rename directory and cleanup failures; external sentinel content/modes remain unchanged on refused paths.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/toolchain/source.go:233`
- `tools/gomad3/toolchain/source_test.go:19`
- `tools/gomad3/toolchain/patch_regenerate.go:292`
- `tools/gomad3/toolchain/patch_test.go:196`
- `tools/gomad3/toolchain/patch_cleanup_test.go`
- `tools/gomad3/internal/hostfs/replace.go:15`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 -run '^Test(Ensure|Regenerate|PinnedArchive)' ./toolchain

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Neither archive nor patch retains private atomic replacement; streamed download bounds/checksum and exact staged patch checks remain before publication.
- [ ] HTTP/cancel/overflow/checksum/patch/apply failures preserve the prior file, destination guards and cleanup/error identity.
- [ ] Post-rename failures report actual publication state; focused controls run without native runtime construction or external source download substitution.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
