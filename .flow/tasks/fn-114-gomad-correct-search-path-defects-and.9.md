---
satisfies: [R7]
---
# fn-114-gomad-correct-search-path-defects-and.9 Keep one copy of each prepared target per artifact store

## Description
E2 (R7), store half: publication places the target once per store and every artifact in that store shares it; readers verify it before execution. Accounting, pruning, merge, and the measurement are task 10. Depends on tasks 3 and 8 because they edit the corpus and minimizer files this task also changes.

**Size:** M
**Files:** `tools/gomad3/artifact/store.go`, `publication.go`, `open.go`, `store_test.go`, `publication_test.go`, `tools/gomad3/runner/replay_operation.go` and its test, `tools/gomad3/runner/minimize_operation.go`, `tools/gomad3/runner/internal/corpus/corpus.go`, `tools/gomad3/runner/inspect.go`
**Touches:** [tools/gomad3/artifact/**, tools/gomad3/runner/replay_operation.go, tools/gomad3/runner/replay_operation_test.go, tools/gomad3/runner/minimize_operation.go, tools/gomad3/runner/internal/corpus/**, tools/gomad3/runner/inspect.go, tools/gomad3/runner/inspect_test.go, tools/gomad3/runner/runner.go, tools/gomad3/runner/choice_exploration_campaign.go, tools/gomad3/runner/simulation_exploration_campaign.go, tools/gomad3/runner/runner_test.go, tools/gomad3/record/**, .flow/specs/fn-114-gomad-correct-search-path-defects-and.md]

### Approach
- First step: choose the sharing form and record the choice in the spec's Decision Context.
  - Recommended: a content-addressed target pool, keyed by the SHA-256 and size the manifest already records, with each artifact's `target` file a hard link to the pool entry. The manifest, the artifact schema, and every reader stay unchanged, retained artifacts stay readable, and any recursive copy of an artifact directory is a self-contained export.
  - Alternative: the manifest references the pool entry and the artifact directory holds no target. This needs a new target form in the record, a schema decision for retained artifacts, store-level resolution in a reader that is confined to the artifact directory today, and an explicit export step. Take it only if hard links cannot meet R7, and say why.
- Pool ownership: the publication stores in use today are narrower than the sharing boundary. Their roots are per-kind directories and per-round staging directories (`successes` under a staged round, the campaign's successes path, the corpus cases path, the minimizer output root). A pool under each of those would not share across rounds or campaigns. The pool therefore belongs to the artifacts root the command was given (the directory that holds every campaign of that root), and the corpus and the minimizer output root each own one for their own directory. `Store` receives the pool location from its caller; it does not derive it from `Root`.
- A staged round is renamed into place on commit. Links survive the rename, and the pool is outside the staged directory, so an abandoned round leaves at most an unreferenced pool entry for pruning (task 10).
- Publication: write the pool entry by temporary file and rename, then link. On an existing entry verify SHA-256 and size before linking. Two publishers of the same target (parallel executions, shards sharing a store) must both succeed with one pool entry.
- A store on a filesystem without hard links falls back to a private copy and the result says so. It is never a failure and never a silent success of sharing.
- Readers keep verifying the target's hash and size before execution, as they do today. Add tests for a pool entry or link whose content was altered and for a truncated one; both fail before execution.
- An artifact copied out of its store with a plain recursive copy replays on its own.
- Cover the same path for the minimized-artifact store and the corpus store.
- Mode bits are shared across links. The pool entry carries the mode the manifest records for the target today, unchanged; the manifest validator accepts only the two existing modes and readers compare the mode exactly. Do not make targets read-only. Readers already copy the target before executing it, so execution never writes through a link.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/artifact/publication.go:40-80` — target payload placement
- `tools/gomad3/artifact/store.go:65-190`, `:229-300` — publication, staging, store identity, payload copy
- `tools/gomad3/artifact/open.go:19-70`, `:122-180`, `:214-258` — open, directory validation, payload copy
- `tools/gomad3/runner/minimize_operation.go:200-204` — target copy for minimization
- `tools/gomad3/runner/runner.go:923`, `:1805`, `tools/gomad3/runner/choice_exploration_campaign.go:374` — store roots that publication uses today
- `tools/gomad3/record/validation.go:655`, `tools/gomad3/artifact/open.go:311-320` — accepted file modes and the exact mode check

**Optional** (reference as needed):
- `tools/gomad3/runner/replay_operation.go:137-141`, `:477` — target copy and verification before replay
- `tools/gomad3/record/validation.go:89` — file reference validation
- `tools/gomad3/runner/internal/corpus/corpus.go:259-300` — corpus store and byte cap
- `.flow/memory/bug/integration/shard-merge-and-prepared-target-cache-2026-09-29.md` — prior identity bug in the prepared-target cache
- `tools/gomad3/ARCHITECTURE.md:355-381` — artifact layout contract

### Key context
- fn-109 task 12 separates detached Artifact references from owned handles in the same files. If it has landed, build on its handle types; if it has not, keep this change inside the existing functions so that task can still apply.
- Artifact directories are validated file by file against the manifest; an unlisted file fails validation. The pool lives outside artifact directories.
- A hard-linked file passes regular-file checks. Confirm the no-symlink open path treats it as regular on both platforms.
## Acceptance
- [ ] The sharing form is chosen and recorded with its reason
- [ ] An artifacts root with N artifacts of one target holds one copy of the target binary, shown by a test that counts distinct files and spans successes and failures, several exploration rounds, and two campaigns
- [ ] The published target's recorded mode and on-disk mode are the ones published today, and existing artifacts open unchanged
- [ ] Replay, `replay --verify-only`, resume, minimize, and inspect pass their existing tests unchanged
- [ ] Two concurrent publications of the same target into one store both succeed and leave one pool entry
- [ ] An altered or truncated shared target fails before execution; a missing one fails before execution
- [ ] An artifact copied out of its store replays on its own
- [ ] A store without hard-link support publishes private copies and reports that sharing is off
- [ ] `go -C tools/gomad3 test -tags test_dep ./artifact/... ./runner/...` and `make -C tools/gomad3 validate` pass
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
