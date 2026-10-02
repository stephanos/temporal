---
satisfies: [R9]
---
# fn-108-gomad-reduce-code-size-without-removing.8 Run the linux/amd64 gates for the code-size cleanup

## Description
fn-108's seven tasks are done and verified on darwin/arm64, and R9 stays incomplete because no linux/amd64 gate ran. Run the linux commands listed in `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md` on a native linux/amd64 host, or in GitHub Actions on the commit that carries the fn-108 changes.

## Acceptance
- `make -C tools/gomad3 validate` and `test`, the `gomad3sim` and integration tests, the smoke set, and the core set pass on linux/amd64 against the fn-108 baseline dispositions.
- Results, platform identity, and toolchain identity are retained beside final.md.
- An unexplained regression against the recorded baseline keeps R9 open; the D12 allowances stay as the baseline states them.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
