---
satisfies: [R1, R3, R4, R5, R6, R9]
---
# fn-230-gomad-explore-site-scope-for-choice.1 Site scope flag, pattern validation and resolution against the base trace

**Touches:** tools/gomad/rust/gomad/src/cli/{explore.rs,plan.rs}, tools/gomad/rust/gomad/src/campaign/exploration/**, tools/gomad/testdata/parity/**, tools/gomad/CLI.md
**Semantic references only:** tools/gomad/internal/cli/explore.go, cli/plan.go, tools/gomad/internal/runner/options/, tools/gomad/internal/runner/internal/exploration/site_scope.go, CLI.md

## Description
Implement once on the Rust host after fn-232.8 and the existing site-spec predecessors; the Go locations below document existing semantics only. Port the behavior into the Rust ownership surface above and extend the shared golden/behavioral corpus. Do not create or extend a Go-host implementation.
Add `--site-scope` to explore and plan for `--strategy=choice-exploration` only. Validate patterns (import path grammar, optional `/...` suffix, non-empty, no whitespace) at parse time with status 2. After the base execution, symbolize every site in its trace via the symbolize package, resolve patterns to a fingerprint set, and store `{patterns, resolved_sha256, resolved_count}` in the plan payload and campaign record; a module without symbols fails with status 2 before execution; an empty resolved set stops round one with `site_scope_empty` after publishing the base artifact. Plan validation refuses a malformed scope block on open.

## Acceptance
Invalid patterns and wrong strategies exit 2 with named reasons; a stripped module exits 2 before any execution; an empty scope publishes the base artifact and stops with `site_scope_empty`; the plan and campaign record carry the scope block; without the flag the plan bytes are unchanged.

## Done summary
R1/R3/R4/R5/R6/R9 accepted. The previously pending public plan receiver is now accepted in fn232.11. Completed full installation lanes15/16 each pass316 direct public calls plus authenticated child lanes, with four independent attempts and unchanged input hashes. Actual matching/empty scoped plans explicitly execute and retain one authenticated base, persist patterns/resolved digest/count, refuse stripped symbols and malformed requests, and keep portable zero-guest seed preparation distinct. Immutable fixture63c43b4525d16dae1aa1432ddbff8424516c8595c31b3f67c37eaadcda133500/source82ba and committed golden654693efd83b2e9e49d0b420d623a5f364ca24c1ba58527be5b59f3ffaf4ff1b bind this receipt.

Earlier complete public scope runs03/04 remain retained and were re-inspected:34 command receipts each, identical observation SHA e895c690cdc2ebe767b37ae8e5ba833f22a905deaf5cb3bb119afeb4d77485c7. Task2/3 are authoritatively done in shared Flow state; this receipt closes task1's sole remaining public-plan dependency, not their reimplementation. Existing site-scope parsing/unit/controller/golden and independent source/journal reviews cover invalid patterns/strategies, unchanged unscoped behavior, retained empty frontier, persisted bounds and record/evidence separation. Scope selects proposal sites; it does not remove seed-driven decisions or change the acceptance predicate.

Source and actual receipt review has no remaining producer/plan finding. fn231's new authenticated search-context additions are a subsequent unaccepted feature batch with its own review fixes, not evidence for this closure. Final integrated-current-source runtime qualification remains fn232.13/R12. No performance, broad test support, Go retirement or final-candidate pass is claimed.

stage: impl-review - ran (existing fresh site_scope_review plus integrated install_source_review; independent source/actual artifact checks)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 97ac127b01, 38572d7e49, 734bc7c566, 9c97d59ff7, 21adc20beb, e5ea438c90, 64f90e84c3, af3a7a3740
- Tests: actual public site-scope03/04:34 commands each; identical observation SHA e895c690cdc2ebe767b37ae8e5ba833f22a905deaf5cb3bb119afeb4d77485c7 rechecked, actual integrated public plan scoped matching/empty retained base and stripped/no-symbol refusals through full15/16: exit0, full15 regeneration and16 strict:316 commands each plus authenticated children; four independent attempts agree, site_scope unit/controller and malformed plan/resume checks in accepted source; current integrated default library94passed, fresh independent source and actual plan/artifact review: no remaining findings
- PRs: