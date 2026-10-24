---
satisfies: [R2, R5]
---
# fn-248-off-the-shelf-crates-and-consolidation.2 One copy of each repeated helper

## Description
Collapse repeated helpers to one copy each.

- `panic_message(payload: &(dyn Any + Send)) -> &str` in umpire-checker (`model/umpire/backends/native`), replacing the downcast chains in `model/umpire/author/suite/src/{coverage.rs ~77, pattern_search.rs ~102, search.rs ~121}`, `model/umpire/backends/native/src/explore.rs` (~261), `model/umpire/backends/apalache/src/replay.rs` (~556). One fallback text; update assertions on the old texts.
- One SplitMix64: `model/umpire/catalog/src/value.rs` (~8-23) and `model/umpire/author/suite/src/pattern_search.rs` (~46-62). Use one deterministic seeded implementation without requiring the old sequence or `% len` index; place it in a crate both already depend on, or justify a new edge.
- One applicability mapping returning the prost `ProbeApplicability` in umpire-catalog, used by `model/umpire/reference/kettle/probes.rs` (~31-38, magic `0..3` today) and `model/temporal/realization/src/probes/generic.rs` (~336-357, including `matches!(.., 0 | 3)`).
- One `Errors` accumulator (`push`, `finish(value) -> syn::Result<T>`) in umpire-front, replacing `model/umpire/front/src/declare/action.rs` (~82-93), `declare/fields.rs` (~27-31), `declare/mod.rs` (~700-711).
- `replace_once`, `replace_in`, `replace_code` in umpire-testing, replacing copies in `model/temporal/feature/activity/policy_checks.rs` (~365, ~594), `proof_checks.rs` (~17), `model/umpire/dev/fixtures/fixtures/world_fairness/tests.rs` (~36).
- One hex decoder in `model/umpire/protocol/runtime/tests/support/mod.rs`, replacing `tests/wire.rs` (~24-32), `core_protocol.rs` (~61), `proposal_wire.rs` (~62).

## Acceptance
- [ ] Each helper exists once; no former copy remains (grep evidence in the task summary).
- [ ] A fixed seed reproduces the new outputs; generated values stay within their declared domains and emitted witnesses replay and satisfy their recorded claim. Review and regenerate pattern-search/value-generation goldens; matching old sequences is not required.
- [ ] `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

