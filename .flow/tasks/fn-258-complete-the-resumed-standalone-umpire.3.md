---
satisfies: [R4]
---
# fn-258-complete-the-resumed-standalone-umpire.3 Adapt actual Temporal consumers to current standalone interfaces

## Description
**Touches:** [Makefile, go.mod, go.sum, mise.toml, .github/**, model/Cargo.*, model/scripts/**, model/dev/xtask/**, model/temporal/**, model/design/native-symbolic/**, model/README.md, model/docs/paired-checkout.md, tools/umpire/**, common/testing/testpilot/**, tools/canary/**, tests/**, .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/consumer/**]

Prepare a recoverable consumer branch by porting 91547f57ec onto current Temporal logic. Start from f7b42ce's explicit interface map; finalize and test against task 2's actual exported paths/revision. Preserve supervision, admission, work_counts and nextest. Keep the xtask crate as a normal runtime dependency of the Temporal bridge (not dev-only), changing only its sibling path. Include the native-symbolic probe and every actual Rust/Go caller. Own the Temporal quick caller exclusively. Prepare explicit repository/revision/credential paired-CI wiring and activation inventory, but keep it inactive while access/published pin is unavailable; activation belongs to task 5. Prove parent workspace and local module selection without replacing existing workspace entries.

## Acceptance
- [ ] Actual consumer candidate dependencies/imports resolve the sibling standalone; native-symbolic probe paths are included and its metadata is checked.
- [ ] Cargo metadata contains only actual sibling framework manifests, not model/umpire. go env GOWORK and go list -m -json show the intended workspace/host module.
- [ ] Current validation orchestration and the normal xtask dependency are retained; focused caller-context/shell/format/compilation checks pass.
- [ ] A recoverable reviewed branch/bundle and exact source/interface record are retained without premature root integration or CI activation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
