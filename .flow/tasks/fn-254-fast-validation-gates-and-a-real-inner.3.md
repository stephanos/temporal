---
satisfies: [R4]
---
# fn-254-fast-validation-gates-and-a-real-inner.3 Parallel test execution across binaries (cargo-nextest, owner-approved)


## Description
Run tests across all binaries at once instead of one binary after another (spec R4). Owner approved cargo-nextest, pinned via mise (2026-10-25).

**Size:** M
**Files:** real framework/Temporal test-runner leaves extracted by .10; framework/Temporal nextest configurations and mise pins. Tests of runner policy live in the owned leaves; existing command and stage registration already calls those working leaves.
**Touches:** [model/umpire/.config/nextest.toml, model/umpire/mise.toml, model/umpire/mise.lock, model/.config/nextest.toml, mise.toml, mise.lock, model/umpire/dev/xtask/src/test_run.rs, model/dev/xtask/src/test_run.rs, model/umpire/author/dsl/macros/tests/ui/action_domain_unbound.stderr, model/umpire/author/dsl/macros/tests/ui/action_finite_missing.stderr, model/umpire/author/dsl/macros/tests/ui/optional_facts_direct_reject.stderr]

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Workspace-move snapshot correction

Own the three exact listed stderr files only for mechanical `$WORKSPACE/umpire/author/dsl/src/` to `$WORKSPACE/author/dsl/src/` normalization. Preserve all other diagnostic bytes and record old/new hashes; reconcile as 560 unchanged snapshots plus three path-only corrections. This repairs the inherited independent-workspace move mismatch reproduced under both runners.
