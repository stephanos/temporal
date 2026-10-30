**Review of fn-253.5 (75b4ee5..37aa5f6), axis: integration with unchanged code**

The change ships. I traced the three integration seams and found no introduced wrong behaviour, regression or security issue.

**What I verified**

- **Fixture vs. library surface.** `model.rs` calls `arm`, `tick`, `armed`, `expired`, `within`, `descends`, `Ticks::contains` and `Remaining::contains`. The `probe` field deliberately retains armed zero so saturation and `descends(Some(0), Some(0))` are reached without violating `within` on the live counters. The lifted TLA in `Clock.tla.snap` encodes the same optional code scheme (`0` = None, `raw+1`) as `Bounded::decode`, and `Init` with `probe = 4` matches `Some(Remaining { ticks: 3 })` in `states.txt.snap`.
- **Mutants vs. compiled good graph.** Each of the four `from` anchors occurs exactly once in `model.rs` (asserted). I hand-traced all four under their chosen Worlds: no-decrement and absent-arming fail `descends` on the first Tick edge; zero-corruption reaches `probe = Some(0)` after `first` expires, then the mutant emits a `Some(1)` edge the good Model refuses as a no-change step, so compiled replay fails as asserted; the strict `within` mutant fails at the start state, where the test correctly switches from transition replay to claim-verdict disagreement. The library's own `tick` never underflows, so the spec's "representable mutation" constraint holds.
- **`bounded_debug` normalization vs. its callers.** Only `debug_value` calls it, and only `state_text`/agreement consume the result. Library descriptors are the sole producer of qualified names (`front/src/body/library.rs:94` formats `umpire_dsl::{canonical}<{bound}>`); local newtypes (`optional.rs:42`) use the bare ident and are unaffected. Type identity in `Literal::admit` (`ir/src/literal.rs:78,86`) still compares the full `bounded.name`, and the error path at `backend.rs:25` still prints the full name, so nothing but display changed. Derived Rust `Debug` for `Remaining<3>` prints `Remaining { ticks: n }`, which the normalized form now matches; red/green logs confirm the previous mismatch.
- **Evidence.** `required-final.log` shows all three fixture tests pass under `UMPIRE_BACKENDS=require` (3/3, 32.37s); `testing-required-final.log` shows the support lib 13/13 including the new regression.

**FYI (non-blocking)**

- `tests.rs:5-13` adds the tree's only `ActionValues<Interpreted<World>>` impl, with `domains` returning an empty vector instead of forwarding. Correct for payload-less `Clock`, but if copied to a fixture with `#[domain]` payloads the interpreted replay would see an empty catalog silently. Consider forwarding to the inner World's domains, or a doc comment stating the payload-less assumption.
- `dev/fixtures/Cargo.toml` now depends directly on `umpire-front` and `syn`. The module map says fixtures "use author and test support"; the task spec and the referenced `proof_checks.rs:16` establish the same direct pattern, so this is consistent with precedent, but a `umpire_testing` helper wrapping source→`Lifted` would keep the fixture crate within its stated imports.
- `backend.rs:55-56`: both `unwrap_or` fallbacks are unreachable (`split`/`rsplit` always yield at least one item). Harmless.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R2 | met (this task's slice: fixture-source mutants, named-law refutation, interpreted replay, no override hook) | `tests.rs:62-137`; `required-final.log:703`; remaining R2 clauses belong to fn-253.1–.4 |
| R1, R3, R4, R5 | deferred | out of this task's scope (task frontmatter `satisfies: [R2]`) |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>