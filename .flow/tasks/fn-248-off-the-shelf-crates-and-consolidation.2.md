---
satisfies: [R2]
---
# fn-248-off-the-shelf-crates-and-consolidation.2 One copy of each repeated helper

## Description

**Size:** M
**Files:** model/umpire/backends/native/**, model/umpire/backends/apalache/src/replay.rs, model/umpire/author/suite/**, model/umpire/catalog/**, model/umpire/front/**, model/umpire/dev/testing/**, model/umpire/protocol/runtime/tests/**, model/umpire/reference/kettle/probes.rs, model/temporal/realization/src/probes/generic.rs, model/temporal/feature/activity/**, model/umpire/dev/fixtures/fixtures/world_fairness/tests.rs
**Touches:** [model/umpire/backends/native/**, model/umpire/backends/apalache/src/replay.rs, model/umpire/author/suite/**, model/umpire/catalog/**, model/umpire/front/**, model/umpire/dev/testing/**, model/umpire/protocol/runtime/tests/**, model/umpire/reference/kettle/probes.rs, model/temporal/realization/src/probes/generic.rs, model/temporal/feature/activity/**, model/umpire/dev/fixtures/fixtures/world_fairness/tests.rs]

Isolated source lane after fn247 closure; no whole-task dependency on another fn248 source lane. Coordinate shared interfaces, accept central merge conflicts and retain private source/index/build targets. The final integration task owns snapshot reacceptance, current guest publication and the one R5 full pipeline. No local lane result claims joined validation.
Collapse repeated helpers to one copy each.

- `panic_message(payload: &(dyn Any + Send)) -> &str` in umpire-checker (`model/umpire/backends/native`), replacing the downcast chains in `model/umpire/author/suite/src/{coverage.rs ~77, pattern_search.rs ~102, search.rs ~121}`, `model/umpire/backends/native/src/explore.rs` (~261), `model/umpire/backends/apalache/src/replay.rs` (~556). One fallback text; update assertions on the old texts.
- One SplitMix64: `model/umpire/catalog/src/value.rs` (~8-23) and `model/umpire/author/suite/src/pattern_search.rs` (~46-62). Use one deterministic seeded implementation without requiring the old sequence or `% len` index; place it in a crate both already depend on, or justify a new edge.
- One applicability mapping returning the prost `ProbeApplicability` in umpire-catalog, used by `model/umpire/reference/kettle/probes.rs` (~31-38, magic `0..3` today) and `model/temporal/realization/src/probes/generic.rs` (~336-357, including `matches!(.., 0 | 3)`).
- One `Errors` accumulator (`push`, `finish(value) -> syn::Result<T>`) in umpire-front, replacing `model/umpire/front/src/declare/action.rs` (~82-93), `declare/fields.rs` (~27-31), `declare/mod.rs` (~700-711).
- `replace_once`, `replace_in`, `replace_code` in umpire-testing, replacing copies in `model/temporal/feature/activity/policy_checks.rs` (~365, ~594), `proof_checks.rs` (~17), `model/umpire/dev/fixtures/fixtures/world_fairness/tests.rs` (~36).
- One hex decoder in `model/umpire/protocol/runtime/tests/support/mod.rs`, replacing `tests/wire.rs` (~24-32), `core_protocol.rs` (~61), `proposal_wire.rs` (~62).

### Placement and error controls

Place SplitMix64 in the existing lower checker dependency chain, which both suite and catalog already reach; add an explicit local dependency only if needed and record it. No cycle or new general utility crate is required. Test fixed-seed reproducibility, index/domain boundaries and semantic witness replay independently of snapshot text. Preserve panic downcast behavior for &str/String/non-string payloads, applicability variants/unknown wire values, accumulated error ordering and edit-helper missing/duplicate match failures.

## Acceptance
- [ ] Each helper exists once; no former copy remains (grep evidence in the task summary).
- [ ] A fixed seed reproduces the new outputs; generated values stay within their declared domains and emitted witnesses replay and satisfy their recorded claim. Review and regenerate pattern-search/value-generation goldens; matching old sequences is not required.
- [ ] Focused checks below pass; joined full validation belongs to the final integration task.


## Focused verification and integration handoff

Run from the repository root, with a lane-owned target; commands containing `cd model` enter the Rust workspace. These are mandatory affected-scope checks, not another full Model gate:

```bash
cd model && mise exec -- cargo test -p umpire-checker -p umpire-suite -p umpire-catalog -p umpire-front --lib --locked
cd model && mise exec -- cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --locked
```

Lane-local candidate snapshots may be generated/reviewed to exercise focused controls in the isolated copy; final canonical reacceptance still occurs once after joining. Report actual failures and candidate snapshot diffs. Add controls specified by this task and run the new controls by their actual selectors. After central conflict resolution, rerun affected focused checks; final integration owns reviewed snapshot acceptance/current artifacts and R5. No historical output-byte comparison is required.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

