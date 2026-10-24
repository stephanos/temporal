---
satisfies: [R12, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.26 Bind profile-neutral execution fixtures to their compiled role

## Description
**Touches:** tools/gomad/rust/wasmhost/src/linked_cluster/live_controller_tests.rs, tools/gomad/rust/wasmhost/src/linked_cluster/lifecycle_predicate_tests.rs, tools/gomad/rust/wasmhost/tests/linked_streaming.rs

Size M. Cycle6 C744b on d36e998c is terminal INCOMPLETE/exit1:22 bound PASS, rust-diagnostic-test FAIL,73 unstarted,zero mismatches. Cargo compilation/listing passed; allocation_receiving actual test binary exited101 with264 PASS/16 FAIL/16 ignored. Preserve raw evidence and terminal custody. Root cause: profile-neutral cluster/single-linked test preparations retain Conditions::Ordinary, while strict admission requires Conditions::compiled(). Correct only these test fixtures using existing .conditions(...) patterns and a small shared test preparation seam where appropriate. Include the three single-linked storage preparations and the seven latent linked_streaming tests, including capture-only execution. Do not change production prepare/default/validate, feature gates, negative role tests, assertions, inputs, memory/capacity/time bounds or ignored status.

Evidence: /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/cycle6-diagnostic-failure-observation.json; cycle6-diagnostic-identity-diagnosis.json/md; diagnostic-conditions-binding-impact.json/md. Two src test files participate in helper identity hashing; preserve independent host/runtime versus helper identity distinctions. Main/source may move now that controller session49034 is terminal1. Use an isolated repair worktree and an exclusively assigned unsealed target; never compile changed source into old qualification targets. Parent owns integration, reviews, Flow completion and the new full cycle; .13 retains final96/145 and all receiving acceptance.

### Quick commands

Use cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --release --locked --offline --no-default-features -p gomad-wasmhost with each feature set reference-helper; reference-helper,diagnostic-profile; reference-helper,diagnostic-fuel-off; reference-helper,guest-profile; reference-helper,guest-profile,diagnostic-fuel-off. For each run --test allocation_receiving --test linked_streaming --test diagnostic_roles -- --test-threads=1, and separately --lib linked_cluster::live_controller_tests -- --test-threads=1. Run formatting and Clippy for affected feature sets. Capture a fresh pre-fix diagnostic streaming failure in the isolated workspace; retained cycle6 supplies the original diagnostic allocation red. Parent full mandatory merge gate and reviewed changed-source cycle remain required; focused success does not qualify the candidate.

## Acceptance
- [ ] Retain exact cycle6 failure/process/test inventory and source custody; explain the Ordinary-versus-compiled role cause using actual production entry points.
- [ ] Bind profile-neutral cluster, single-linked storage and streaming test preparations to compiled conditions. Preserve production strict admission/default behavior, every original assertion/input/bound, negative role matrix, feature support and ignored status.
- [ ] Actual focused RED/GREEN and complete affected five-profile matrix pass with original executed/ignored counts distinguished; format and affected Clippy pass. No stale/frozen artifact reruns or mutations.
- [ ] Conductor integrated mandatory gate and independent implementation review pass; verify Flow done with actual evidence. Rebind helper-dependent artifacts honestly for the next complete current-source cycle, preserving old build provenance and the failed attempt. No all96/final145, R4 or receiving pass inferred from this repair.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
