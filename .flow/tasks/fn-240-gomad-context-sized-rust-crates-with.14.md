# fn-240-gomad-context-sized-rust-crates-with.14 Adopt umpire rustfmt.toml: one reformat commit before the .1 baseline

## Description
Adopt umpire's `rustfmt.toml` (umpire/temporal/model/rustfmt.toml): `edition = "2024"`, `style_edition = "2024"`, `max_width = 100`, `use_small_heuristics = "Max"`, `reorder_imports = true`, `newline_style = "Unix"`, stable options only.

Measured 2026-10-25: every setting except `use_small_heuristics = "Max"` matches today's defaults, and `cargo fmt --check` is clean today. "Max" reformats 465 of 541 `.rs` files (about −34.3k / +9.7k lines, net −24.6k, ≈11% of 224.5k lines). It packs short calls, attributes and struct literals onto one line. That is wrapping, not code, so files get easier to read and cheaper to load into an agent's context.

Why this runs before .1: R10 (about 17k production lines per crate against the .1 baseline) and R12 (the `too_many_lines` allow-list) both count lines. Reformatting after the baseline would mix two formatting styles in those comparisons, and some offenders would leave the allow-list just from reformatting. Doing it first gives the whole spec one formatting style.

Do: add `tools/gomad/rust/rustfmt.toml`, run `cargo fmt --all`, and commit only the config plus the reformat, nothing else. Add that commit hash to `.git-blame-ignore-revs` (create the file if needed) in a follow-up commit. `make -C tools/gomad fmt-rust` and the driver `rust-fmt` stage pick up the config with no recipe change.

Ordering: after fn-242 closes and its worktrees (/home/agent/wt/fn242-b*) are merged or abandoned, because nearly every file changes and any open branch would conflict. Spec-level fn-242 → fn-240 ordering already exists. Before announcing the commit, check `git worktree list` and the Flow assignees for open Rust work.

## Acceptance
- `tools/gomad/rust/rustfmt.toml` matches the settings above; `make -C tools/gomad fmt-rust` passes.
- The reformat commit contains only the config and formatting changes; `cargo build --workspace` and `make -C tools/gomad test-harness test-rust-engine` are green on it. The reformat must not change behaviour; if the checks above disagree, stop and investigate.
- `.git-blame-ignore-revs` lists the reformat commit.
- No open fn-242 or other Rust worktree had unmerged commits when it landed (record `git worktree list` in the evidence).


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
