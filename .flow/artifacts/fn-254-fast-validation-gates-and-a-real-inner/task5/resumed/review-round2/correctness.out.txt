Review complete. Scope: the eight source files at `85c0a39f8a` plus the fixes summary. Task17 design and probe artifacts were excluded as instructed.

**Verified paths**

- **Thread policy** (`test_run.rs` in both workspaces): `configure` reads the command's own env entry first, so an explicit override or `env_remove` wins; otherwise it falls back to the process env. Minimum of a positive parsed value and the TLC4/Apalache1 cap. A second `configure` finds the already-capped value and recomputes the same number, so it is idempotent. Skipped mode sets nothing and passes inherited values through untouched.
- **Standalone receipt vs child**: `clean_command` calls `env_clear`, so the find in `configure` returns nothing and both the child and the receipt in `finish` derive from the same process env value. The child's inner `gate::tests()` re-applies `configure` to the already-capped value, which is stable.
- **Temporal**: `test_run::run` now only builds the cargo command; `gate::stage` applies the policy once for the `tests` name, covering both the nextest and doctest invocations. The stage test now exercises the min logic directly.
- **UI glob**: `tests/ui/*.rs` matches exactly the 18 `.rs` files present; the `.stderr` siblings are excluded by the glob. Parser rows live under `parser_refusals`, not under `tests/ui`.
- **Snapshot repair** (`runner.rs:171-179`): under `INSTA_UPDATE=always` only `.snap` files that are neither skipped-consumer nor external artifacts bypass prevalidation. Skipped artifacts were already validated in `skipped_files`, externals in `external_artifacts`, so no admission is lost. Unselected siblings are never read later, so a corrupt sibling cannot fail the selected case. Normal mode is unchanged. The red control log confirms both new tests fail on the previous unconditional validation.

**Findings**

No P0 or P1 findings. No P2 findings.

FYI, not blocking:
- `test_threads` maps nextest's relative forms such as `-1` or `num-cpus` to the fixed cap rather than a smaller machine-relative count. This matches the summary's stated policy and the cap is the JVM ceiling, so it is a settled choice.
- `test_threads` and `configure` are byte-identical in `model/dev/xtask` and `model/umpire/dev/xtask`. Duplicated Code, but the two workspaces are independent by design and this predates the change.
- Under `always`, a `.snap` with a syntactically broken YAML header still cannot be repaired because insta's own loader rejects it inside `assert_snapshot`. Pre-existing limitation, not newly exposed.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R4 | met | Bounded `required` profile preserved in both workspaces and the standalone child; lower inherited allocations now honored; policy tests in `gate.rs`, `test_run.rs`, `gate_wiring.rs`, `standalone.rs` cover default, lower, higher, malformed, zero and negative. |
| R5 | met | Remaining trybuild cases are the 18 rustc-dependent UI files via the restored glob; the other R5 bullets were accepted in the prior three SHIP draws and are unchanged here. |

Unaddressed R-IDs: []

Suppressed findings: 2
Classification counts: 0 introduced, 0 pre_existing.

```json
{"suppressed_count":{"50":2},"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>