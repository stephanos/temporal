Both claims check out: the front call yields the IR `Lifted` tree, and the fixture's only front/syn use is parsing mutant source for `Interpreted::from_tree`. Writing up the verdict now.

**Scope:** two source hunks in the range 1b7e420..3d10f5a, plus Flow artifacts. I read the task file, the full diff, the current fixture test, the fixture manifest, the module map row, and the `replace_once` helper.

**Verified**

- **replace_once reuse is semantically equivalent.** `umpire_testing::replace_once` at `model/umpire/dev/testing/src/source_edit.rs:2` asserts exactly one match then does `replacen(..., 1)`. The removed inline code asserted one match then called `replace`, which is identical under that assertion. It is exported from `dev/testing/src/lib.rs:17`. `name` is still used at `tests.rs:96,106,115,121`, so no unused-variable warning under `-D warnings`, matching the committed clippy log.
- **Module-map exception matches actual imports.** `dev/fixtures/Cargo.toml:18-26` lists `umpire`, `umpire-checker`, `umpire-front`, `syn`, `umpire-testing`, `umpire-tla`, `umpire-tlc`, `imbl`. The new row text "author and test support plus generic checker/backend controls" covers checker/tla/tlc; the `umpire-front`/`syn` clause is confined to source-mutant parsing. The only front/syn use in the fixture crate is `tests.rs:90`, which feeds `umpire_front::read` output (`umpire_ir::Lifted`, re-exported at `front/src/lib.rs:32-33`) into `Interpreted::from_tree` (`backends/native/src/interpret.rs:133`). So "parse candidate source into ordinary IR" is an accurate description, not a loophole.
- **"Neither imports system Models" holds.** No `umpire-temporal`/`temporal` references anywhere under `dev/fixtures` or `dev/testing`.
- **No semantic drift.** No changes to `model.rs`, goldens, the `ActionValues<Interpreted<World>>` adapter, or any backend code; `source-envelope.json` hashes of the untouched paths are consistent with that.
- **Evidence consistency.** `followup/source-hashes.json` after-hashes for both paths equal the `source-envelope.json` hashes; `evidence.json` commands match the focused required-mode test that the log shows passing 1/1.

**Findings**

None at P0–P2.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/fixtures/fixtures/countdowns/tests.rs:89`
- **R-IDs**: []
- **Problem**: The uniqueness-assertion message changed from the mutant `name` to `mutation target: {from}`. Three mutants share the `tick` anchor, so a future duplicate-anchor failure names the anchor, not which mutant row tripped. Harmless: any such failure fires on the first row regardless.
- **Suggestion**: FYI only; no change needed.

**FYI (process, non-blocking):** the task file's Done summary and Evidence sections are still `TBD`/empty; the Flow closure is owned by the root per `followup/summary.md`.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R2 | met (task .5 slice) | Four source-only mutants refute `Post_descends`/`Invariant_within` and replay in required mode (`followup/source-mutants-required.log`, 1 passed); no override hook; this follow-up changes only the helper call and the module-map contract. |
| R1, R3, R4, R5, R6 | deferred | Owned by other fn-253 tasks per task frontmatter `satisfies: [R2]`. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>