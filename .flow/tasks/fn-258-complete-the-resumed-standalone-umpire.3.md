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
Consumer3f5f613578f2857b738a7792fad08f12b3ea0a41 is reviewed SHIP by fresh Claude Fable5.1 high followup after initialthree-axis SHIP and six fixed findings. Exactclean3f5/054 pair passes alltargets/allfeatures compilation,16unit+7integration controls, shipping Make missing-sibling guards, and actual parentGo workspace automatic discovery/allthree moduleDirs without overwriting workspaceentries. Actual Rust/Go paths and native-symbolic metadata are covered; normal xtask runtime dependency, supervision/admission/nextest/workcounts retained. New thinbundle requiringac60 independently recovered exacttree/canonicalguest and strictfsck0. Allsix initialreview findings addressed with current evidence; finalnonblocking browser link wording is assigned task5 docs. Earlier host/Go/noRust/native-symbolic proofs bind unchanged runtime inputs. Makefile changed declaredAPI input, so task4 must renew guestfreshness and finaleditproofs; no stale currentinput claim. Candidate stays unmerged, CIinactive and source authority inTemporal per reviewed interimownership. Publication/cutover remain task5. Validation/reviewcost assessment retained.

stage: impl-review - ran [2026-10-28 final3f5 followup; SHIP] (model: claude-fable-5-1 high)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e4cf57dd013b6c86cc9d6e0e27984005608ad4ba, 1fb713f6fcdcab8f1078d52f363d948a195d65b2, c258547ed5cc7d2ddc2247fce956dc5bb8504d32
- Tests: Exactclean3f5/054: all-target/all-feature check;16unit+7integration; Makeguards; Go parent workspace/moduleDirs; clean before/after, New thinbundle actual prerequisite-only recovery; exacttree/guest; strictfsck0, Content-based unchanged host/noRust/native-symbolic runtime proof rebinding; currentinput guestfreshness explicitly expires to task4
- PRs: