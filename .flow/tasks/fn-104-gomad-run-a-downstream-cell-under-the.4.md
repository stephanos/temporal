---
satisfies: [R2]
---
# fn-104-gomad-run-a-downstream-cell-under-the.4 Add the exact adapter for the membership address library

## Description
The membership layer's address library imports os/exec (remain_unsupported) and stays live in linked mode. Add an exact, digest-anchored adapter in the shape of the fx/SDK/otel adapters: version pinned in toolchain/version/version.json, per-file source and replacement digests, original/replacement inventories, darwin/arm64 prepared source-set pin, and a negative test that fails the build on an upstream edit. The rewrite refuses the subprocess path deterministically.

## Acceptance
- the adapter activates for the pinned version and fails closed for any other
- an upstream-edit mutation fails the build
- make validate and the adapter tests pass


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
