---
satisfies: [R6, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.6 Delete the temporal/annotate facade crates

## Description
Implement **Facade crates (B4)**: delete `temporal/annotate` and `temporal/annotate/macros`; add a crate-root `CatalogContext` alias and api re-export; switch the four importers; move the view tests to `catalog/tests`.

## Acceptance
- [ ] Both crates are gone from the workspace; importers build.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
Removed the two Temporal annotation facade packages; retained the syntax crate, switched authors to shared catalog macros with realization-root defaults, moved all admission/view/descriptor coverage to its actual owners, and reviewed the path/qualified-name golden updates. Three implementation review draws returned SHIP with no findings.

Joined focused Rust tests and strict combined clippy passed. The Temporal guest was regenerated and freshness-checked; every standard guest and native panic control built; Temporal, runtime and Kettle Go host suites passed on the joined source tree. The model tree at the integrated commit is byte-identical to the verified joined tree. Parent R7 retains its final full model gate after the remaining consolidation work; this scoped task does not claim a new full-gate run.

Tier: session (jev-unavailable(no_key)). Reviewer: codex:gpt-6.1-sol:high; writer/reviewer are the same model family. Logs and review receipts: `.flow/tmp/milestones-resume-20261024/`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: aaaf2271324a4f4ce46bf8a2c1211e725cd81dc1
- Tests: Joined scoped cargo test for runtime, stock guest, catalog, Temporal syntax, Temporal and API; new-join-tests.exit=0, Combined --all-targets --all-features strict clippy; new-join-host-validation.exit=0, guest.sh umpire-temporal-guest and guest-freshness.sh; canonical build-guests.sh; all passed, CGO_ENABLED=0 mise exec -- go -C model/temporal/hosts/go test -tags test_dep -count=1 ./..., CGO_ENABLED=0 mise exec -- go -C model/umpire/hosts/go test -tags test_dep -count=1 ./... with current canonical guests, CGO_ENABLED=0 mise exec -- go -C model/umpire/reference/kettle test -tags test_dep -count=1 ./... with current Kettle guest
- PRs: