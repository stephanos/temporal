---
satisfies: [R2]
---
# fn-104-gomad-run-a-downstream-cell-under-the.3 Decouple x/sys admission from the libc adapter

## Description
The x/sys packs bind activation to the modernc libc adapter, so a closure without SQLite never activates them and the x/sys/unix and x/sys/cpu assembly, linknames, and syscall imports become blockers. Add a standalone x/sys v0.47.0 pack per qualified platform (darwin/arm64 here; linux/amd64 is D9) whose activation names only golang.org/x/sys, through discover/review/generate/check/qualify, using a server-side or fixture closure that reaches x/sys without libc.

## Acceptance
- a fixture closure that imports x/sys/unix without SQLite analyzes as supported on darwin/arm64
- compatibility-pack-qualification qualifies the new request; existing packs unchanged


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
