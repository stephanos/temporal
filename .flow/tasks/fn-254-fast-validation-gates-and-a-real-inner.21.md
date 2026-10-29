---
satisfies: [R4]
---
# fn-254-fast-validation-gates-and-a-real-inner.21 Close safe residual framework integration-binary consolidation gaps

## Description
Reconcile remaining multi-target framework crates with .4's completed consolidation and consolidate demonstrably safe residual targets. Multiple binaries can be required by UI, fixture or process-global isolation contracts; the goal is justified link savings with complete coverage.

**Size:** M
**Files:** residual target manifests/harnesses under `model/umpire/{catalog,dev/testing,protocol/runtime,backends/native,reference/kettle,author/dsl,author/dsl/macros,author/suite}`, affected selectors/docs and durable inventory evidence.
**Touches:** [model/umpire/catalog/**/Cargo.toml, model/umpire/catalog/tests/**, model/umpire/dev/testing/Cargo.toml, model/umpire/dev/testing/tests/**, model/umpire/protocol/runtime/Cargo.toml, model/umpire/protocol/runtime/tests/**, model/umpire/backends/native/Cargo.toml, model/umpire/backends/native/tests/**, model/umpire/reference/kettle/Cargo.toml, model/umpire/reference/kettle/tests/**, model/umpire/author/dsl/Cargo.toml, model/umpire/author/dsl/tests/**, model/umpire/author/dsl/macros/Cargo.toml, model/umpire/author/dsl/macros/tests/**, model/umpire/author/suite/Cargo.toml, model/umpire/author/suite/tests/**, model/umpire/dev/xtask/src/gate/**, model/umpire/.config/nextest.toml, model/README.md, model/umpire/README.md, model/umpire/reference/kettle/README.md, model/design/gate-speed.md]

### Approach
- Consume .4's committed `task4/consolidation-identities.json`, exceptions and .5's UI/fixture case mapping. Derive a fresh Cargo target inventory rather than treating its historical 36 targets or the observed 38 workspace integration targets as a bug count. Current residual multi-target packages are catalog/testing (7 each), runtime/native/Kettle/DSL (3 each), DSL macros/suite (2 each); recheck current source.
- Classify every residual target as safe to merge or required separate, naming the actual process-global state, trybuild/compiler admission, fixture discovery, dedicated selector or feature-mode contract. Follow consolidated `tests/main.rs` module patterns, or explicit autotests=false targets in xtask/guest-driver where appropriate. Avoid a second migration of already-consolidated cases.
- Merge only safe residuals in the listed packages; preserve all UI globs/diagnostic/pass cases, generated fixture harnesses, root-relative source paths and process isolation. Capture package/target/full-name/ignored identities under matching feature/backend modes before/after and a checked one-to-one mapping for changed identities; separately account for doctests and snapshot/UI bytes.
- Update actual gate/nextest and documented selectors if changed. Run affected target execution and focused fmt/clippy; retain exact no-selection/missing-case failures. Compare binary/link workload and representative forced-build measurements, distinguishing compile/link from test runtime. No-safe-merge is valid only with explicit per-target contracts and evidence.
- Do not touch Temporal consolidation or release .7: it remains blocked until fn-246's committed whole-spec closure. Keep .12 paused; provide exact changed work counts for its owner and .8 rather than installing a competing budget mechanism.

### Investigation targets
**Required:**
- `.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task4/consolidation-identities.json` — prior identity/isolation pin.
- `model/umpire/catalog/tests/main.rs` and `tests/{ui,admission}.rs` — consolidated versus dedicated compiler harnesses.
- `model/umpire/dev/testing/tests/main.rs` — explicit lack of process-global fixtures.
- `model/umpire/author/dsl/macros/tests/fixtures.rs` and `model/umpire/reference/kettle/tests/exhaustiveness.rs` — compiler/pass contracts.
- `model/umpire/dev/xtask/Cargo.toml` — explicit target pattern.

### Key context
The existing .4 closure proves a 925-identity mapping and 563 unchanged snapshot/UI hashes, not that every remaining binary is redundant. Preserve complete gate selection (memory: `bug/integration/full-integration-gates-must-select-the-2026-09-04`).

## Acceptance
- [ ] Fresh inventory reconciles every residual target in the named framework packages with prior .4/.5 mappings and gives a concrete isolation/coverage reason for every retained separate target.
- [ ] Safe residual targets are consolidated; or explicit evidence establishes no safe useful merges. Full package/target/name/ignored bijection, UI/fixture/diagnostic contracts, snapshots and affected execution are checked rather than inferred from counts.
- [ ] Changed gate/nextest/documented selectors select the complete intended cases, with exact changed work counts and compile/link measurements or clearly explained unavailable attribution recorded. Temporal .7 and paused .12 remain untouched; broad final validation stays .8.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
