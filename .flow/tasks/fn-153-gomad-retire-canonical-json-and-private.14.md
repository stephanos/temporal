---
satisfies: [R1]
---
# fn-153-gomad-retire-canonical-json-and-private.14 Preserve the builder's staged stamp and launcher transaction

## Description
Implements R1 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** toolchain/build.go and existing/new stable-publication failure tests (1 core publisher).
**Touches:** [tools/gomad3/toolchain/build.go, tools/gomad3/toolchain/build_test.go, tools/gomad3/toolchain/build_*test.go]

### Approach

- Replace temporaryFile and file-publish copies with shared stages. Create and close both launcher 0755 and stamp 0644 stages before publishing either.
- Preserve stable-bin symlink/non-directory guards, then stamp publish+directory-sync, after-stamp-publish failpoint, launcher publish+directory-sync and after-launcher-publish failpoint.
- Keep immutable build-directory publication separate; reuse existing stage cleanup with primary error joins and per-stage publication status.
- Behavior pin: record exact state/file/mode/error outcomes at every existing checkpoint and new stage-write/close/dir-sync failure. Compare baseline old launcher/stamp visibility and cleanup.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/toolchain/build.go:556`
- `tools/gomad3/toolchain/build.go:623`
- `tools/gomad3/toolchain/build_test.go:36`
- `tools/gomad3/toolchain/build_test.go:83`
- `tools/gomad3/toolchain/build_test.go:175`
- `tools/gomad3/internal/hostfs/replace_test.go`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 -run '^TestBuild' ./toolchain

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Both files stage before first publication; requested modes, guard refusals and stamp/failpoint/launcher/failpoint ordering match the frozen transaction pin.
- [ ] Failures at each stage preserve the baseline observable intermediate state and old launcher behavior; close/cleanup errors remain discoverable.
- [ ] No private atomic file publisher remains in the builder; immutable directory publication and unsupported-host refusal are unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
