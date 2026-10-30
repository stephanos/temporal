Review of the R4 follow-up range a4e97c8..c258547 for task fn-258.3, correctness axis, checked against the candidate worktree at 3f5f613 (branch ref confirmed at that commit).

**What I verified**

- The claimed post-3658582 delta is accurate. The prior-candidate patch has the sibling preflight only on `model-quick` and the `58a6989` checkpoint; the current patch has the preflight on all four targets, `05473210`, the shipping missing-sibling control block, canonical README/module paths and the one-sentence CLI comment. The Rust test edits (`-- gate` filter, `GOWORK=off`, `UMPIRE_TOOL_ROOT`) were already in 3658582, so the "no Rust changed" claim holds.
- `framework-root.sh` resolves `model/scripts/../../..` to the parent of the Temporal root and appends `umpire`; the actual missing-sibling logs show exactly that path and make exit 2 for check/quick/fmt/update, with no tool invocation. The red log shows the pre-fix bare `Error 1` without guidance, so the control is a real red/green.
- The shipping `test-model-wiring.sh` missing block copies the real `framework-root.sh`/`xtask.sh` into `missing/temporal`, so it exercises the shipped guard, not the fake `exit 0` stub used by the wiring controls. The fake-stub path and the real-guard path are correctly separated.
- Path arithmetic for every relocated reference checks out: `time-gate.sh` `$root/model/../../umpire/...`, `proto-go.sh` after `cd model`, `test-model-validate.sh` `control_parent/umpire`, `guest-roots.sh` and `proto-go` test scratch layouts, `make_paths.rs` symlink at `f.root/umpire` (inside the fixture, no TMPDIR collision), `snapshot_ownership.rs` two-checkout split, README relative links.
- Current pair 3f5f613/05473210: clean before and after, all-target/all-feature check green, 16 unit + 7 integration pass, make-guard controls green, `go env GOWORK` and `go list -m -json` show the three intended module Dirs under the pair parent. Receipt sha256 in `readiness.json` matches `manifest.json`. Bundle sha256 matches between `current-source.json` and `bundle-recovery-3f5f613.json`; recovery from the `ac60` prerequisite alone with fsck 0.
- All six dispositions are backed by the named evidence. Guest freshness is explicitly expired to task 4 (Makefile is a declared API build input), not claimed.

## Findings

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.worktrees/fn2583-consumers/common/testing/testpilot/README.md:5` (also `tools/umpire/README.md:35`)
- **R-IDs**: []
- **Problem**: Disposition 6 replaced the retired `model/umpire/hosts/go/README.md` pointer with Markdown links to `../../../../umpire/hosts/go/README.md`. Those resolve on disk only under the paired layout and render as broken links on GitHub or in any single-repository view. The CLI comment fix used prose ("hosts/go/README.md in the sibling Umpire checkout"), which is the better pattern.
- **Suggestion**: Use the same prose wording in both READMEs instead of a filesystem-relative link that escapes the repository. Non-blocking; a task 5 docs pass may own it.

FYI (not findings):
- `actual_make_freshness_recipe_propagates_stale_failure` extracts only the cargo `-- gate` line, so the Make preflight is covered solely by the shell control. That is adequate; just noting the split.
- `UMPIRE_INNER_SOURCE` in `time-gate.sh` is now resolved relative to `model/` with a `../../umpire/...` default, so an operator's old framework-relative override silently changes meaning. Env-override only.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R4 | partial | Sibling Cargo paths and parent Go workspace proven on the exact 3f5f613/054 pair (`current-pair-3f5-054/receipt.json`, `go-workspace.json`); missing checkout fails with actionable guidance on all four model targets; workspace preserved (sha unchanged). CI activation and current-input guest freshness are explicitly deferred to tasks 5 and 4 by the task description and `proof-rebind-3f5f613.json`. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[],"suppressed_count":{}}
```

<verdict>SHIP</verdict>