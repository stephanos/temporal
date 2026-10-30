---
satisfies: [R4]
---
# fn-254-fast-validation-gates-and-a-real-inner.3 Parallel test execution across binaries (cargo-nextest, owner-approved)


## Description
Run tests across all binaries at once instead of one binary after another (spec R4). Owner approved cargo-nextest, pinned via mise (2026-10-25).

**Size:** M
**Files:** real framework/Temporal test-runner leaves extracted by .10; framework/Temporal nextest configurations and mise pins. Tests of runner policy live in the owned leaves; existing command and stage registration already calls those working leaves.
**Touches:** [model/umpire/.config/nextest.toml, model/umpire/mise.toml, model/umpire/mise.lock, model/.config/nextest.toml, mise.toml, mise.lock, model/umpire/dev/xtask/src/test_run.rs, model/dev/xtask/src/test_run.rs, model/dev/xtask/src/gate.rs, model/umpire/author/dsl/macros/tests/ui/action_domain_unbound.stderr, model/umpire/author/dsl/macros/tests/ui/action_finite_missing.stderr, model/umpire/author/dsl/macros/tests/ui/optional_facts_direct_reject.stderr]

### Parallel ownership
- Consume committed .4 identity mapping and .10's working test-runner leaves. Own only the listed test-runner source/config/pin files, including focused unit controls inside the runner leaves. Their existing gate callers remain unchanged; no dispatcher, Make, manifest, aggregate-test or budget edit is needed to change runner policy. Task .8 owns consolidated tier/tool documentation.
- This is genuinely disjoint from .6's quick CLI/cache source. If the realized .10 interface makes another source edit necessary, amend actual Touches and recheck admission before editing; do not bypass overlap checks. Record exact inventories against the lane's candidate. Later added controls or converted cases are explicit producer inventories and are reconciled at .8, never silently omitted.

### Approach
- Pin cargo-nextest through mise. Replace the `tests` stage with `cargo nextest run --workspace --locked`, plus `cargo test --doc --workspace` (nextest does not run doctests).
- Test groups: TLC/Apalache/JVM-backed tests (filter by binary or name, e.g. `real_tlc_*`, `actual_tlc_*`, fixtures with `UMPIRE_BACKENDS=require`) get a bounded group so JVM heaps fit; everything else uses all cores.
- Keep fail-fast semantics for the gate. Keep `--quiet`-equivalent output, and emit the slow-test report so the timings stay visible.
- **Equivalence pin:** collect the pre-change Cargo package/target/full-name/ignored inventory and compare it to nextest under matching features/backend modes, using .4's checked migration mapping where applicable; include doctests separately. This correctness inventory is still required despite the owner's timing-baseline waiver. Reject missing, duplicated or unmatched identities. Bound the combined runner/test concurrency and demonstrate peak JVM count, including nested runner parallelism.

### Investigation targets
**Required:**
- `Makefile:1290-1298` — tests, wasm, Go stages
- `model/design/gate-speed.md` — baseline per-binary times
- `mise.toml` — current tool pins; .9/.13 establish the framework-local successor

## Owner validation cadence (2026-10-25)

Keep exact Cargo/nextest/ignored/doctest inventory parity, representative affected execution and combined JVM-limit controls now; unrelated full workspace execution waits for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] The test list (with ignored) is identical before and after; doctests still run.
- [ ] The `tests` stage wall time is recorded against the baseline; JVM tests never exceed their group limit.

## Done summary
Pinned nextest 0.9.149 via mise for both real test-runner leaves, followed by exactly one Cargo doctest phase. Exact migration inventory preserves 962 logical tests/four ignored and five doctests/one ignored; bounded JVM group/process controls and representative affected execution pass. Three inherited workspace-path goldens were normalized mechanically; 560 other snapshot/UI files and 119 registry tuples remain unchanged. Temporal runner consumer now covers both phases and all 28 timed/untimed failure scenarios. Same-session re-review SHIP confirms both integration findings fixed. Required standalone full-copy proof remains task .11; unrelated broad gates and final stage measurements remain .8 under owner cadence.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a21f91b116748a720ff12c910b7d50b733f9ffa0, 934fc8559d5f179b365929716b2f005f484efffa, 76fee4be039d28ae8e3c3261aace095d6b416902, 71685512286d343ad1a59e3349e23688ea82663f, adf475da992685bd601254581e1d397136bd9463, ba473b52be7800703ec60223ab40b206066ab56b, d49485e91f41cd4a1a808585a25ec13392945080
- Tests: python3 .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task3/proof/inventory.py (both fmt/xtask clippy/inline tests, Cargo+nextest exact lists, Cargo doctests: PASS), python3 .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task3/proof/runner-process.py (15 process controls: PASS), python3 .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task3/proof/tool-admission.py (public root/framework and independent framework pins: PASS), python3 .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task3/proof/representative.py (50 framework tests: PASS; Java peak2/owner1), python3 .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task3/proof/representative.py temporal (6 Temporal tests: PASS; Java peak2/owner1), python3 .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task3/proof/doctest-inventory.py (5 exact identities,1 ignored: PASS), python3 .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task3/proof/account.py (962 identities,563 snapshot/UI hashes,119 registry tuples,exact195+41 group membership: PASS), cargo nextest run -p umpire-guest-driver --locked --test-threads 2 -E test(guest_pipeline::) (8 parser/prerequisite controls: PASS), git diff --check; all551 indexed proof files tracked and SHA256 verified; git status clean, Original Cargo+nextest targeted macro fixtures/refusals compare-mode GREEN after exactly5 admitted prefix replacements across3 goldens; updates disabled (revision1/proof/trybuild-comparison.json)., python3 .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task3/revision1/proof/verify.py: 560 other snapshot/UI bytes unchanged;119 registry tuples;all runner/Rust/manifests unchanged;corrected bytes equal retained actual update output., All revision1 proof/source hashes verified; source checkout clean. Full Temporal run not launched per root cadence correction; task11 owns joint full framework proof., Revision2 exact gate::tests::actual_temporal_gate_wiring_and_named_failures RED101 ->GREEN; all28 timed/untimed success/failure child scenarios pass; nextest/doc both named tests., Revision2 cargo fmt --all --check and cargo clippy -p temporal-xtask --all-targets --locked -- -D warnings PASS., Revision2 verify.py confirms production gate bytes unchanged,prior11 source hashes+563 snapshot/UI hashes exact,and only existing test body changed., Revision2 all12 source/17 proof hashes verified and tracked;parent same-primary-session formal re-review remains pending alongside task11 fix., Same-primary-session gpt-6.1-sol high re-review SHIP; current 12 source/revision2 proof hashes verified against joined checkout; no unresolved task3 findings.
- PRs:
## Workspace-move snapshot correction

Own the three exact listed stderr files only for mechanical `$WORKSPACE/umpire/author/dsl/src/` to `$WORKSPACE/author/dsl/src/` normalization. Preserve all other diagnostic bytes and record old/new hashes; reconcile as 560 unchanged snapshots plus three path-only corrections. This repairs the inherited independent-workspace move mismatch reproduced under both runners.

## Runner consumer correction

Own the Temporal gate.rs wiring control to expect nextest then Cargo doctests and preserve each phase failure/status/stage-name assertions. Framework wiring belongs to .11. This exact consumer file is required by first-round review; no other gate composition change belongs here.
