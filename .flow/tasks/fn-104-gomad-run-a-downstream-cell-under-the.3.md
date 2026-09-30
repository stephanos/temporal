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
Added golang-x-sys-v047-darwin-arm64, reviewed through discover/review/generate/check/qualify: it activates on golang.org/x/sys v0.47.0 alone and admits the x/sys/unix and x/sys/cpu darwin/arm64 assembly, syscall imports, and runtime/libSystem linknames, plus the x/sys imports of the x/term v0.45.0 and x/crypto/sha3 v0.55.0 versions the server pins. The fixture module internal/compatibilitypack/testdata/xsys reaches x/sys only through those modules (a direct import is a main-module finding no pack may admit) and analyzes as supported with zero blockers; compatibility-pack-qualification qualifies it; existing packs unchanged apart from the profile digest that the address-library adapter later moved. On darwin the SQLite closure now also selects this pack. linux/amd64 is fn-105 D9.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4bd232127
- Tests: gomad analyze --working-dir=testdata/xsys: supported, 0 blockers, make compatibility-pack-qualification (darwin: 9 requests), make validate
- PRs: