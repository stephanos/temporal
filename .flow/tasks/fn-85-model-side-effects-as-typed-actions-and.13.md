---
satisfies: [R13]
---
# fn-85-model-side-effects-as-typed-actions-and.13 AUTHORING.md with its drift test, concept entries and rule drafts, fn-83 closure, full gate

## Description
Close the spec (R13): write `model/AUTHORING.md` as a walk from an empty file to a green live test over the Nexus Model, with every Lean block equal to a marked region of the Model file and a Go drift test that fails on a missing or duplicate marker; add the concept entries (Entity, Party, Set, Realization, Refinement, Abstraction Claim) and amend Action, Observation and Machine in `UMPIRE4_SPEC.md`; draft the AUT-07a and MOD-02 amendments under GOV-02; point DESIGN.md at the spec and the Model; close fn-83's six blocked tasks as superseded naming each concern's destination; sweep the documents; run the full gate. Single finalization task.

**Size:** M
**Files:** `model/AUTHORING.md` (new), `tools/umpire/authoring/drift_test.go` (new; markers `-- authoring: <name>` in the Model file), `tools/umpire/internal/retiredvocabulary/check.go` (`model/AUTHORING.md` added to `requiredFiles`; `model/` root is not a scanned tree), `.plans/UMPIRE4_SPEC.md` (concept entries; AUT-07a and MOD-02 drafts marked `drafted by fn-85; awaiting GOV-02 approval`; the AUT-09 amendment fn-83 .8 would have drafted), `model/Temporal/Feature/Nexus/DESIGN.md` (header points at the spec and the Model; section 5's "Today" column and section 6's "not yet reflected" note updated), `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md` (`Umpire.Command` row and the command-surface section), `tools/umpire/CONTEXT.md` (glossary entries with `_Avoid_` lists: `interface`, `statemachine`, `link`, `test`/`environment` bindings), `tests/testcore/testpilot/README.md`, `.plans/UMPIRE4_ORDER.md` (fn-83 and fn-85 entries; gate baselines), `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.{4,5,6,8,16,17}.md` (closed as superseded through `flowctl`, each naming its destination)
**Touches:** [model/AUTHORING.md, tools/umpire/authoring/**, tools/umpire/internal/retiredvocabulary/check.go, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_ORDER.md, model/Temporal/Feature/Nexus/DESIGN.md, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, tools/umpire/CONTEXT.md, tests/testcore/testpilot/README.md, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.*.md]

### Approach
- Closing fn-83's six blocked tasks goes through `flowctl` where it is installed. Where it is not (the
  cloud sessions have no flow-next plugin install), edit the six records in `.flow/tasks/` to the
  stored shape — `status`, `updated_at`, and a done summary naming the destination of each concern —
  and say in the receipt which route was taken.
- Drift test: parse `model/AUTHORING.md` for fenced Lean blocks tagged with a marker name, read the Model file's marked regions, compare byte for byte; a missing or duplicate marker fails naming it. No existing markdown-drift test to copy; the nearest shapes are `make umpire-check-inventory`'s regenerate-and-diff and `tools/umpire/vocabulary/spec_names_test.go`.
- Destinations for fn-83's tasks (from the planning record): .4 to fn-85 .7; .5's fault grammar to this spec's actions and the outage Model to fn-86 R4 (with the outage-order rule); .6 to fn-85 .10 Query 1; .8 to this task; .16 to fn-85 .7's derived identity; .17 to the realization's binding checks. Use `flowctl` to close them and record the mapping in each summary.
- MOD-15: every new backticked dotted name the concept entries cite must resolve in `model/`; run `go test ./tools/umpire/vocabulary/...`.
- Docs-gap list from the planning record is the checklist (each `path:line`).

### Investigation targets
**Required:**
- `.plans/UMPIRE4_SPEC.md:27-84,201-249,284-301` — concept glossary, MOD-02, AUT-07a, AUT-08, AUT-09
- `.plans/UMPIRE4_ORDER.md:9-41,123-164` — the fn-83 and fn-85 entries
- `model/Temporal/Feature/Nexus/DESIGN.md:1-5,543-600`
- `tools/umpire/vocabulary/spec_names_test.go:26-39` — the MOD-15 gate and its planned-rule escape
- `tools/umpire/internal/retiredvocabulary/check.go:53-68` — `requiredFiles`

**Optional:**
- `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.8.md` — the walkthrough plan this task supersedes

### Key context
- The Refinement entry must say a refinement is not an Implementation Link (SEM-08 reserves that name).
- `make lint-code` under-reports on low disk; baseline 161 after `go clean -cache`; `make lint-model` needs LEAN_NUM_THREADS=1.

- The AUT-09 amendment (a `structure` of finite fields and a step function enumerated into the finite table are author-provided) is drafted here beside AUT-07a and MOD-02, marked `drafted by fn-85; awaiting GOV-02 approval`.
## Acceptance
- [ ] `model/AUTHORING.md` walks from an empty file to a green live test; the drift test passes and fails on a planted missing and a planted duplicate marker; the file is in the vocabulary gate's required files
- [ ] `UMPIRE4_SPEC.md` has the six concept entries and the three amended ones; AUT-07a and MOD-02 amendments are drafted under GOV-02; MOD-15 gate green
- [ ] fn-83 tasks .4, .5, .6, .8, .16 and .17 are closed as superseded with destinations; the order document records fn-83 and fn-85 as done with new gate baselines
- [ ] `make umpire-check-regression` exit 0; `make lint-model` at or below 163; `make lint-code` at 161 after `go clean -cache`; live identity count recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
