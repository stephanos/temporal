Review complete. Verdict details follow.

**Scope read:** task .5 contract and the parallel-shape decision, every source hunk in the range (runner refactor, generated per-fixture cases, parser-refusal table, nextest profiles, three xtask gates, standalone receipt, TLC probe heap, Makefile), plus the unchanged callees they meet: `run_selected`'s callers in Temporal and the consumer tests, `discovery::Fixture`, the proc-macro entry points, `declare::derives`, `PinnedJar::admit`, `Limits::default`, `java_command`, the standalone configuration copy list, and the `update`/`validate` Make routes. Artifacts were used only as summaries (final validation, group membership under `--profile required`, decision file).

## Findings

**Finding 1**
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/testing/src/runner.rs:171`
- **R-IDs**: []
- **Problem**: The new admission loop calls `validate_text_snapshot` on every existing golden unconditionally. Before this change the only validation of a consumer-owned golden happened inside `snapshot()`, which skips it when `INSTA_UPDATE=always` so insta can overwrite the file. Now the `update` route (`make model-update` → xtask `update` → nextest with `INSTA_UPDATE=always`) refuses with `invalid golden …` for any consumer-owned `.snap` that lacks the `---\nsource:` header or has a broken header, instead of rewriting it. This applies to the 24 framework cases and also to the unchanged Temporal umbrella, since `run_at_skipping_with_external` now routes through the same `run_selected(None)` path. Scenario: a truncated or hand-edited `expected/report.txt.snap`; previously `model-update` repaired it, now it fails and the user must delete the file by hand.
- **Suggestion**: Mirror `snapshot()`: skip the admission-time `validate_text_snapshot` when `INSTA_UPDATE` is `always`, or restrict the admission validation to fixtures other than the selected one and leave the selected fixture's goldens to `snapshot()`. Either keeps the sibling-admission guarantee the new consumer tests assert while preserving update-mode repair.

**Finding 2**
- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/src/gate.rs:13`, `model/dev/xtask/src/test_run.rs:17`
- **R-IDs**: []
- **Problem**: The Temporal gate applies `configure` twice on the same command: once in `test_run::run` before calling `stage`, and again inside `stage` when `name == "tests"`. The result is idempotent, but it is duplicated policy code with two tests covering the same behaviour; the decision file also said neither test-run leaf needed changing. Overengineering / Duplicated Code.
- **Suggestion**: Keep one site. The `stage` hook is the one the Temporal control test exercises, so drop the call in `test_run::run`, or keep the leaf and drop the `stage` branch.

**FYI (non-blocking, not filed):**
- `model/temporal/realization/tests/registry.rs:273` adds a comment asserting trybuild disables batched compile-fail when a pass case shares the `TestCases`. I could not verify that claim against trybuild's source in this session. The split itself is harmless and gives nextest two independent identities, so nothing turns on it.
- Each of the 24 generated cases redoes discovery plus validation of all 191 snapshots before selecting one. That is 24× redundant admission I/O, explicitly required by the task ("complete registry/snapshot/consumer admission before selection"), so not a finding.
- Error precedence changed for an orphan golden that is also malformed: `invalid golden` now fires before `golden has no fixture`. Both still fail; the new consumer tests cover the orphan path.
- Code-smell heuristics: `run_selected` is a Long Method (self-acknowledged by the `too_many_lines` allow), and the five-tuple `(name, run, files, selected, skipped_artifacts)` pushed to `admitted` is a Data Clump that wants a small struct. Heuristic only.

**Integration checks that passed:**
- Generated case names come from `Fixture::module`, which is already required to be a valid `mod` identifier; cases sit under a `fixtures` module so identities are `fixtures::<module>` and match the existing `binary(=fixtures)` JVM-group filter.
- Parser table calls `machine::expand`, `action::expand`, `finite::expand`, `state::expand` with the production signatures and uses `umpire_front::declare::derives`, which is public. `proc-macro2` has `span-locations` at the workspace level, so authored line/column are real. Spot-checked `action_metadata_placement` (4:1, 6:12) and `finite_shape` rows against the moved stderr bytes.
- Remaining `tests/ui` directory holds exactly the 18 files listed in `refusals()`.
- `[profile.required]` inherits the default profile's JVM override; the author's `show-config test-groups --profile required` log confirms membership. All nextest invocations in the repo go through the xtask leaves, so no required-backend path bypasses the cap.
- Standalone child: `.config/nextest.toml` is in the admitted copy list, so `NEXTEST_PROFILE=required` resolves in the isolated checkout; receipt field and env agree.
- TLC probe: `java_command` adds no `-Xmx`, so the new `-Xmx512m` is the only heap flag; `limits.heap_mb` is already part of the cache fingerprint, so no cache key change.
- Two concurrent JVM owners: `PinnedJar::admit` only reads the pinned jar and uses a per-process scratch, so no shared-write race was introduced.
- `model-check-changed` / `model-compiler-inputs` have no live references; only `model/archived/**` prose mentions them.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R4 | partial | 24 independent fixture cases in the existing binary; 84 identities / 191 snapshots unchanged per `fixture-identity-equivalence.json`; JVM group bounded at 2 owners, 4 global slots for required gates. Test-debuginfo profile and binary consolidation belong to other tasks in this spec. |
| R5 | partial | Apalache not required by routine gates and the opt-in is capped to one slot; remaining trybuild cases are the 18 rustc-dependent ones with 61 parser rows in-table; `model-check-changed` removed with no live refs. "Non-cargo stages beside Rust stages" is owned by .17 per the task text. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 0 pre_existing.
Protected-path filter: 0

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>