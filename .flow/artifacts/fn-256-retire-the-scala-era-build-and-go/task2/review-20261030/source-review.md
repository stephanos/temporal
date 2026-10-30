# Review: fn-256.2 source checkpoint (work/fn2562-scala-cleanup @ c796de882e)

Baseline/merge-base: 78801d1feb. Commits: 427a45e113 (removal), 9f63c3966e (archive note), c796de882e (proof artifacts).

## Checks run (read-only)
- `git merge-tree --write-tree umpire work/fn2562-scala-cleanup` -> clean (tree ec0985e552), exit 0. `umpire` has no changes to Makefile, mise.toml, mise.lock, AGENTS.md, model/README.md, model/archived, .gitignore or .github since the merge-base, so no overlap.
- `git grep` on branch for `scala-cli|MODEL_JAR|MODEL_ROOT|scalafix|scalapb|check-framework|MODEL_*|FEATURES` in Makefile, mise.toml, .github: no hits. The only remaining entries are `model_quote` (Makefile:127, used by `SOURCE_FIND` at :128, kept as required) and the `.scala-build`/`.bsp` prune in `SOURCE_FIND`, which is harmless.
- Conditionals are balanced: the `ifneq`/`else` wrapper at the top and its `endif` are both gone; the remaining `ifeq`/`endif` pairs are at 97/103 and 145/150.
- In the worktree: `make -n model-check`, `make -n umpire-check-live-tests` and `make -n canary-build` all succeed. `make -n umpire-clean-scratch UMPIRE_SCRATCH=x` prints only the .flow/tmp loop. `make check-framework` now fails with "No rule to make target", as expected.
- The Rust section from the `##### model: Umpire in Rust` marker to EOF is byte-identical between `umpire` and the branch (sha256 d751aa37...). That matches identity-proof.json.
- `git diff --name-status base..branch -- model/archived` lists only README.md. The README names the full removal SHA 427a45e113..., which is the earlier commit, as required.
- mise.lock: only the `[[tools.scala-cli]]` table and its platform tables are removed. The file still ends with a single newline. mise.toml drops only `scala-cli`; java is kept for TLC.
- .gitignore: only the comment changed; `.bsp/` and `.scala-build/` are kept.
- No CI reference to removed targets (`.github/workflows/umpire.yml` uses `make model-check` only). The artifacts' SHA256SUMS verify. `git diff --check` is clean.

## Findings

### P0
None.

### P1
None.

### P2
None.

### P3
- .plans/UMPIRE_MODULES.md:532-534 (and :293 `make check-model-boundary`): the "Command after migration" table still lists `umpire-check-model`, `umpire-gen-model` and `lint-model`/`fmt-model`/`fix-model` as commands. Spec R3 names UMPIRE_MODULES.md as current guidance that should describe what remains. That file is outside task .2's Files list, and the table reads as a migration record, so treat it as a spec-level follow-up (for task .1/fn-254.13 or a note) rather than a defect here.
- .flow/artifacts/.../phase2-receipt.json: `"final_documentation_commit": "PENDING root commit"` was never filled in. checkpoint-status.json does carry 9f63c3966e, so the record is recoverable, but the receipt is stale.
- .flow/artifacts/.../lint-code-fast.log records a terminated (Error 143) attempt. The passing retry is in lint-code-fast-admitted-baseline.log ("No changed Go packages to lint"). That is fine, but a reader could misread the failing log as the result. The first-phase receipt explains it.
- Makefile:520 comment: the old note "Agents use this target instead of running rm directly" was dropped. This is minor guidance loss. Leftover `.bsp/`/`.scala-build/` directories now have no cleanup target, which is acceptable because they are gitignored.
- The branch also edits the task spec (.flow/tasks/...2.md) to add `mise.lock` to Files/Touches. This scope amendment is reasonable, because the lock entry is useless without the pin.
- R4 (an actual `make model-check` result) remains open. Per the owner, it is deferred to fn-254.8, and the checkpoint artifacts state that explicitly. The parse-time claim is honest: 0.096s before vs 0.099s after, so there was no measurable gain.

<verdict>SHIP</verdict>
