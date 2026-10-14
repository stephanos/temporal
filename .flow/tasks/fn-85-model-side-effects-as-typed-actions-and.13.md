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
- Closing fn-83's six blocked tasks goes through `flowctl` in a clone that carries the runtime state
  (it lives in the clone's `.git` common-dir, not in the repository). In a fresh clone every task reads
  `todo` from the committed snapshot and the status commands refuse, so there edit the six records in
  `.flow/tasks/` to the stored shape — `updated_at` and a done summary naming the destination of each
  concern — and say in the receipt which route was taken. The CLI itself installs from GitHub with
  `claude plugin marketplace add gmickel/flow-next` and `claude plugin install flow-next@flow-next`.
- Drift test: parse `model/AUTHORING.md` for fenced Lean blocks tagged with a marker name, read the Model file's marked regions, compare byte for byte; a missing or duplicate marker fails naming it. No existing markdown-drift test to copy; the nearest shapes are `make umpire-check-inventory`'s regenerate-and-diff and `tools/umpire/vocabulary/spec_names_test.go`.
- Destinations for fn-83's tasks (from the planning record): .4 to fn-85 .7; .5's fault grammar to this spec's actions and the outage Model to fn-86 R4 (with the outage-order rule); .6 to fn-85 .10 Query 1; .8 to this task; .16 to fn-85 .7's derived identity; .17 to the realization's binding checks. Use `flowctl` to close them and record the mapping in each summary.
- MOD-15: every new backticked dotted name the concept entries cite must resolve in `model/`; run `go test ./tools/umpire/vocabulary/...`.
- Docs-gap list from the planning record is the checklist (each `path:line`).
- Adjusted 2026-09-19 after .16, .15, .4, .6, .5 and .7 landed. **AUT-07a already reads
  `machine`** (`.plans/UMPIRE4_SPEC.md:309-318` names `machine`, `property`, `scenario`, `limits`,
  `query` and the `entity`, `enum`, `action`, `observation` declarations), and .16 retired the
  `model` spelling in the vocabulary gate; what this task adds to the draft is `set` and the
  `register_switch` registration (Umpire commands), and it says where the platform's Case-producing
  `case … realizes <set>` block (`Temporal/Case/Syntax.lean:183-260`, the shape .11 leaves) stands
  under AUT-07a, since AUT-07a says `Umpire.Command` must not name a feature and that block is
  Temporal's. **The AUT-09 amendment** covers what landed: a `structure` of finite fields as the
  state, one step function per action enumerated into the finite table (.3, .14), a `property`
  predicate enumerated into clause records by probing (.15), `refines:`/`map:` decided by the
  kernel over the two tables (.6), and a Scenario's `instances:` product (.4). **The concept
  entries** must match the landed meaning: a class is a member of an input domain and an
  abstraction claim is the presence of an `examples:` line (.2, .7); a setup parameter is bound by
  the realization to a dynamic-config key and recorded by the Profile, an unbound one an `input`
  Known Gap (.5); a switch is declared by the realization and registered, not a Model parameter
  (.5, .7); a refinement is a stuttering forward simulation (.6). **Baselines:** live identities
  are eleven since .5 and change again in .10 and .11; `lint-model` 163; `lint-code` 161.
  **Doc drift to take here rather than in fn-86 .1/.9:** `model/README.md:37` names
  `Temporal.Feature.Nexus.Success.Producer`, which does not exist. The drift markers
  (`-- authoring: <name>`) are placed by .10 in the Caller Model; the Go drift test reads them.

### Investigation targets
**Required:**
- `.plans/UMPIRE4_SPEC.md:27-84,195-198,224-231,305-334` — concept glossary, MOD-10 and MOD-11, MOD-02, AUT-07 to AUT-09
- `.plans/UMPIRE4_ORDER.md:9-41,180-408,472-524` — the fn-83 and fn-85 entries and the gate baselines
- `model/Temporal/Feature/Nexus/DESIGN.md:1-5,615-672` — the header, section 5's "Today" column and section 6's "not yet reflected" note
- `tools/umpire/vocabulary/spec_names_test.go:26-39` — the MOD-15 gate and its planned-rule escape
- `tools/umpire/internal/retiredvocabulary/check.go:53-68` — `requiredFiles`

**Optional:**
- `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.8.md` — the walkthrough plan this task supersedes

### Key context
- The Refinement entry must say a refinement is not an Implementation Link (SEM-08 reserves that name).
- `make lint-code` under-reports on low disk; baseline 161 after `go clean -cache`; `make lint-model` needs LEAN_NUM_THREADS=1.

- The AUT-09 amendment (a `structure` of finite fields and a step function enumerated into the finite table are author-provided) is drafted here beside AUT-07a and MOD-02, marked `drafted by fn-85; awaiting GOV-02 approval`.
## Acceptance
- [x] `model/AUTHORING.md` walks from an empty file to a green live test; the drift test passes and fails on a planted missing and a planted duplicate marker; the file is in the vocabulary gate's required files
- [x] `UMPIRE4_SPEC.md` has the six concept entries and the three amended ones; AUT-07a and MOD-02 amendments are drafted under GOV-02; MOD-15 gate green
- [x] fn-83 tasks .4, .5, .6, .8, .16 and .17 are closed as superseded with destinations; the order document records fn-83 and fn-85 as done with new gate baselines
- [x] `make umpire-check-regression` exit 0; `make lint-model` at or below 163; `make lint-code` at 161 after `go clean -cache`; live identity count recorded (`lint-code` is measured as the gate baselines table says a shallow clone can: `lint-code-fast` over the changed packages at the pre-change commit; see the Done summary)
## Done summary

Done 2026-09-20; self-review. Commit b7d74c6.

### The walkthrough and its drift test

`model/AUTHORING.md` walks from an empty file to a green live test in thirteen steps (0 to 12):
the file header, entities, domains, actions, the derived observation, the product machine, the
protocol machine and its refinement, the Properties, the Scenarios and limits, the Queries, the
three sets, the `case` block, and the commands that render the fixtures, run the live test and
run the gate. Every Lean block is one marked region of `Caller/Model.lean`, quoted under
`<!-- authoring: <name> -->` and a fenced `lean` block, generated from the file so the bytes agree;
a `header` marker was added at the top of the Model file so the imports, the module doc and the
namespace are a quoted region too (twelve markers from `.10`, thirteen now). `tools/umpire/authoring`
(`authoring.go`, `Regions`, `Blocks`, `Check`) reads both files: a block naming a marker the Model
lacks, a marker the Model carries twice, a block whose bytes differ from its region, a region the
walkthrough does not quote (the `end` terminator excepted), an unfenced marker and a block quoted
twice are each an error naming the block or marker; `drift_test.go` runs the check on the
checked-in files and plants each fault (a renamed marker, a repeated marker, a one-byte edit, an
extra region, an unfenced and a repeated block) to pin its message. `model/AUTHORING.md` is in
`retiredvocabulary.requiredFiles`, so the vocabulary gate scans it.

### The spec, the design and the documents

`UMPIRE4_SPEC.md`: six concept entries -- Entity (`Umpire.Command.Entity`), Party, Refinement
(`Umpire.Command.Refinement`, not an Implementation Link), Set (`Umpire.Command.SetDeclaration`,
`Umpire.Command.CoverageTarget`), Realization (`Umpire.Case.Producer.Realization`,
`Temporal.Case.Realization.asyncNexus`, the platform's block in `Temporal.Case.Syntax`) and
Abstraction Claim (`Umpire.Case.Producer.ClassClaim`, `Umpire.Provenance.AbstractionClaimRow`) --
and three amended ones: Action (the `action` command, classes as domain members), Observation
(`evidence:` lines, declared reads, `unobservable:` as a Known Gap) and Machine (the `machine`
command, `Umpire.Command.MachineDeclaration`). Three amendments are drafted, each marked
`drafted by fn-85; awaiting GOV-02 approval`: AUT-07a (the surface carries `set` and
`register_switch`; producing Cases is the platform's block, which names the feature); MOD-02 (a
realization lives in `Temporal.Case` because MOD-10 forbids `Temporal.System` importing the Feature
machines); AUT-09 (what `machine`, `property`, `scenario` and `enum` derive is author-provided:
finite-field structures, enumerated step functions, probed predicates, kernel-decided refinement,
the instances product, refused past `Umpire.Command.elaborationBound`). The MOD-15 gate
(`tools/umpire/vocabulary`) is green over every cited name. DESIGN.md's header points at the spec,
the Model and `AUTHORING.md`; section 5's table gains a "Delivered (fn-85 task)" column beside the
"Before fn-85" one; section 6's note says the revised decisions were delivered as revised. Swept:
`model/README.md` (an `AUTHORING.md` pointer, the canary and exploratory sets; the
`Temporal.Feature.Nexus.Success.Producer` name the plan cited was already gone), `model/ARCHITECTURE.md`
(a command-surface paragraph under Semantic model), `model/Umpire/ARCHITECTURE.md` (the
`Umpire.Command` row names the eleven commands; rows for `Records`, `Finite`, `Refinement`,
`Claims`, `Coverage`; the command-surface section rewritten for the landed surface and the
platform's block), `tools/umpire/CONTEXT.md` (Entity, Party, Action, Machine, Refinement, Set,
Realization, Abstraction Claim with `_Avoid_` lists naming `interface`, `statemachine`, `link`,
the `test`/`environment` bindings, `representative`), `tests/testcore/testpilot/README.md` (seven
Queries, the canary and exploratory sets).

### fn-83 and the order document

fn-83's .4, .5, .6, .8, .16 and .17 are closed as superseded in their records: each `Done summary`
names the destination (.4 → fn-85 .7; .5 → fn-85's fault actions and fn-86 R4; .6 → fn-85 .10
Query 1; .8 → fn-85 .13; .16 → fn-85 .7; .17 → fn-85 .10/.11) and each `.json` carries a
2026-09-20 `updated_at`. Route taken: the records were edited in place, because a fresh clone
carries no runtime task state and `flowctl done` and `spec close` refuse there (the plugin was
not installed in this session); the status snapshot stays `todo` for every task, as it does for
every done task of fn-85, and Flow's status follows in a clone that has the state. `.plans/UMPIRE4_ORDER.md`
records fn-83 as closed with the destinations, fn-85 as done with a `.13` receipt, and a
2026-09-20 gate-baselines table.

### Gates

`make umpire-check-regression` exit 0: 614 Lean jobs, every offline check (goldens, conformance,
inventory, retired vocabulary with `AUTHORING.md` scanned, protocol, authoring, regression views),
the Go tests under `tools/umpire` (the new `authoring` package included), `common/testing/testpilot`
and `tests/testcore/testpilot`, and **29 passing live identities**.
`LEAN_NUM_THREADS=1 make lint-model`: the `.11` baseline: two errors in generated `Temporal/API/Proto.lean` and 41 warnings (generated binders, deprecations, the two `enum` binders in `Caller/Model.lean`), none new. `make lint-code` at 161 after `go clean -cache`
is not measurable in this clone (shallow, no `main` merge base, as the 2026-09-13 baseline row
records); `GOLANGCI_LINT_BASE_REV=39a61b4 make lint-code-fast` over the changed packages: 0 issues over the changed packages (`GOLANGCI_LINT_BASE_REV=39a61b4 make lint-code-fast`); the full `make lint-code` is not measurable in a shallow clone with no `main` merge base, as the 2026-09-13 row records.
`go vet -tags test_dep ./tools/umpire/...` clean.

## Evidence
- Commits: b7d74c6
- Tests: `go test -count=1 ./tools/umpire/authoring/... ./tools/umpire/vocabulary/...`; `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-regression`; `LEAN_NUM_THREADS=1 make lint-model`; `GOLANGCI_LINT_BASE_REV=39a61b4 make lint-code-fast`; `go vet -tags test_dep ./tools/umpire/...`
- PRs:
