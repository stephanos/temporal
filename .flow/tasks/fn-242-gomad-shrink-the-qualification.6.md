---
satisfies: [R4]
---
# fn-242-gomad-shrink-the-qualification.6 Remove duplicated measurement checking

**Owner amendment, 2026-10-25 (size counting is informational).** The `gomad-source-size/v1` counter crate and `tools/gomad/docs/source-size.md` are deleted. Record sizes with `tokei` on the named revisions, listing the path groups (production, tests, generated) next to the numbers. Size never gates or blocks closure. Requirements below for the original counter, counter versions, blob hashes/authentication and the three-path classifier correction are retired; earlier reports stay as history.

## Description
Using the .1 inventory, remove logic in tools/gomad/rust/gomad/src/measurement/ (about 25k lines) that re-checks what qualify, report and the Result/receipt readers already prove; keep producers (host-owner-profile, runtime-cost-profile, stress, warm, cluster, retention) and their own checks. Each removal cites its inventory row. Fixture tests stay green; no bound or assertion weakened.

**Touches:** tools/gomad/rust/gomad/src/measurement/**

## Acceptance
- [ ] Every removed check cites an inventory row marked duplicate
- [ ] Producers and their non-duplicate checks unchanged in behavior; existing fixture tests pass
- [ ] Line count of measurement/ recorded before and after

## Done summary
Removed the two inventoried duplicate measurement checks while preserving their dominating profile validators and all producer-specific, report-binding and cleanup checks. Both original inventory IDs resolve through the committed initial/final proof ledgers. The real linked descriptor-refusal fixture catches both guest-count mutations; the deliberate validator-bypass mutant failed. Review caught a feature-role mismatch in that test; the fixed fixture passes for all three admitted diagnostic variants. Removed the stale too_many_lines expectation left by extraction.

Canonical gomad-source-size/v1 measurement counts are bound to before3a0e6dc8e2 and afterf490d7b29c: 59files unchanged, all physical lines25884→25991 (+107). Embedded tests remain in the counter's production-file category; no net size-reduction claim.

Tier: session (jev-unavailable(no_key)); project implementer pin retained.
stage: impl-review - ran (codex:gpt-6.1-sol:high; same family; first three draws retained one P2, resumed review SHIP).

Actual integrated source checks:189 measurement tests passed/1 existing ignored,7 independent plan tests and4 profile tests passed; formatting and strict no-deps Clippy passed. Feature-fix evidence preserves one expected pre-fix failure and three admitted variant passes, each selected1. Original bound evidence remains byte-identical. The reviewed measurement subtree exactly matches integrated main.

No full qualification, full profile matrix, journal reservation, push or CI dispatch ran. Both full attempts remain unused. Complete all fn242 tasks, then all13 approved fn240 tasks, then fn246 actual qualification/receiving; original acceptance stays open. Tracker sync: n/a (inactive).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 23eb8097cd, a110e4209d, 3d33bcfa61, 4d24f8d061
- Tests: Integrated main: cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib --features helper-diagnostic-profile measurement:: -- --test-threads=1 (189pass/1ignored exit0), Integrated main: same command filter cycle::plan::tests:: (7pass exit0) and profile:: (4pass exit0), Integrated main: cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check (exit0), Integrated main: cargo +1.99.1 clippy --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib --no-deps -- -D warnings (exit0), Source guard validator-bypass mutation exit101 intended; genuine valid control and count mutations pass (initial proof ledger), Feature-review-fix: original guest selected1 exit101 expected; admitted plain/guest/fueloff selected1 passed1 exit0 each; final plain selected1 pass1 exit0; whitelist refusal selected1 pass1 exit0, Canonical gomad-source-size/v1 exact before/after inventories and unchanged initial proof hashes verified
- PRs: