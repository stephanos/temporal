---
satisfies: [R1]
---
# fn-153-gomad-retire-canonical-json-and-private.2 Centralize staged and streamed whole-file publication

## Description
Implements R1 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** internal/hostfs replacement/staging owner and fault tests (2-4 core files).
**Touches:** [tools/gomad3/internal/hostfs/replace.go, tools/gomad3/internal/hostfs/replace_test.go, tools/gomad3/internal/hostfs/stage*.go]

### Approach

- Consume fn152's landed hostfs seam without changing protected log/open/lock ownership. Implement one synced-and-closed staged-file handle with path accessor, PublishContext returning publication status plus error, and error-reporting cleanup.
- StageContext accepts a same-directory temp pattern, requested mode and bounded fill callback. The caller validates its staged path before publication. Existing Replace/ReplaceContext wrap this primitive.
- Keep primary and cleanup errors discoverable via errors.Is/As. Track rename success separately from directory open/sync/close errors and cancellation before publish.
- Behavior pin: replay existing replacement controls plus injected create/chmod/fill/write/sync/close/rename/directory/cleanup failures. The helper does not create new containment or locking promises.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/internal/hostfs/replace.go:15`
- `tools/gomad3/internal/hostfs/replace_test.go:11`
- `tools/gomad3/qualification/qualification.go:224`
- `tools/gomad3/toolchain/source.go:233`
- `tools/gomad3/toolchain/patch_regenerate.go:292`
- `tools/gomad3/toolchain/build.go:556`
- `tools/gomad3/internal/hostfs/open.go:23`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./internal/hostfs

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Byte replacement and streamed staging share one file-publication implementation; modes, same-directory atomic replacement and cancellation behavior pass.
- [ ] Failure before rename preserves the old destination and cleans staged state; after-rename failures report published=true without claiming durability.
- [ ] Fault controls cover primary/close/cleanup joins, cancellation, cleanup idempotence and caller-visible published status; existing guarded opens/locks retain their tests.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
