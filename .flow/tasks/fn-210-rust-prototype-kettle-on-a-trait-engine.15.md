---
satisfies: [R41]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.15 Archive the Scala model under model/archived and rename model2 to model

## Description
Runs alone, after .5 completes, before .6 (R41). No other session may run while this does; every session re-reads its task after it.

1. `git mv model model2/archived`, then move any untracked content left in `model/` (build caches, about 4 GB) into `model2/archived/` with plain `mv`, so nothing is lost and the nested `.gitignore` files keep ignoring it.
2. `git mv model2 model`.
3. Makefile: `MODEL_ROOT := model/archived` (Scala targets stay, unmaintained); the Rust section's `cd model2`, script paths and target names become `cd model`, `model/scripts/...`, and `model-check`, `model-fmt`, `model-update`, `model-time` (keep the `MODEL2_STEP`-style macro, renamed).
4. `AGENTS.md`: Core Mandates and Project Structure describe `model/` as the Rust workspace (Umpire in Rust, gate `make model-check`) and `model/archived/` as the frozen Scala framework, reference only.
5. Workspace references: `model/README.md`, `model/DESIGN.md`, `model/Cargo.toml` comments, `model/scripts/*`, `model/hosts/go`, `model/.cargo/config.toml`; links to `model/framework/...` (Scala) become `model/archived/framework/...`.
6. Flow and docs: open specs and tasks of fn-210, fn-211, fn-212 (`model2/` to `model/`, Scala `model/` paths to `model/archived/`), MILESTONES; the deferred Scala specs are left as they are (their paths now resolve under `model/archived/`, noted in MILESTONES).
7. Run `make model-check` and `make model-time`; grep for leftover `model2` outside `.flow` history and `archived/`.

**Files:** model/ (moved), model2/ (renamed), Makefile, AGENTS.md, MILESTONES.md, .flow/specs/fn-21[012]-*, .flow/tasks/fn-21[012]-*.

**Quick commands:** `make model-check`; `make model-time`; `git status --short | head`; `grep -rn model2 --exclude-dir=.flow --exclude-dir=archived .`

## Acceptance
- `model/` is the Rust workspace and `model/archived/` the Scala tree with its history (`git log --follow` works on a moved file).
- `make model-check` passes and `make model-time` is within R30.
- `AGENTS.md` describes the new layout; no `model2` path remains outside history and `archived/`.
- Nothing from the old `model/` is lost (untracked caches moved, not deleted).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
