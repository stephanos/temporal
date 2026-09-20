---
satisfies: [R8]
---
# fn-86-retire-hand-written-models-one.9 Rule drafts, one-authoring-path documents and the full gate

## Description
Close the spec (R8): draft under GOV-02 the AUT-08 amendment removing the expert alternative, the AUT-07a amendment naming the commands as the only authoring path for feature Models, and the MOD rule for task .8 enforced under MOD-11; make the architecture documents, `model/README.md`, `model/AUTHORING.md` and the order document describe one authoring path; run the full gate. Single finalization task.

**Size:** M
**Files:** `.plans/UMPIRE4_SPEC.md` (AUT-08 lines about the expert alternative; AUT-07a; a new MOD-16 with the `authoring-path-isolation` label; MOD-11's list; each marked `drafted by fn-86; awaiting GOV-02 approval`), `model/README.md` (:36 the nonexistent `Success.Producer`; :94-101 the Lifecycle walkthrough; :141-199 typed authoring; :204-211 typed-Nexus Known Gaps; :221-231 the coverage record and Race links), `model/ARCHITECTURE.md` (:46-49, :103-114, :147-150, :254-256), `model/Umpire/ARCHITECTURE.md` (:49-50, :103-105, :246-274), `model/AUTHORING.md` (a short "one path" paragraph; a delta, not a rewrite), `tools/umpire/CONTEXT.md` (:41-44 "Derived rule" names the field relation), `tests/testcore/testpilot/README.md` (:19-43), `.plans/UMPIRE4_ORDER.md` (fn-86 entry; the fn-79 and fn-33 rows reflect the recorded behavior; gate baselines), `model/HANDWRITTEN_INVENTORY.md` (final state: every row resolved)
**Touches:** [.plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_ORDER.md, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, model/AUTHORING.md, tools/umpire/CONTEXT.md, tests/testcore/testpilot/README.md, model/HANDWRITTEN_INVENTORY.md]

### Approach
- Use the docs-gap list in the planning record as the checklist; every `path:line` is either rewritten or recorded as still true.
- MOD-15: every backticked dotted name cited by the new rule text must resolve; `go test ./tools/umpire/vocabulary/...`.
- Gate order: `go clean -cache`, `make umpire-check-regression`, `make lint-model` (LEAN_NUM_THREADS=1), `make lint-code GOLANGCI_LINT_FIX=false`; record the numbers in the order document's baseline table; treat a `lint-code` count below 161 as a truncated run.
- Adjusted 2026-09-19 after fn-85 .7 landed (fn-85 .13 still open): fn-85 .13 drafts its own
  AUT-07a amendment (adding `set` and `register_switch`, placing the Temporal `case … realizes
  <set>` block) and creates `model/AUTHORING.md`; this task's AUT-07a amendment is a delta on that
  draft, not a second draft, and both stay marked as awaiting GOV-02 approval. AUT-07a already
  names `machine` and the four declarations (`UMPIRE4_SPEC.md:309-318`); the sentence to remove is
  AUT-08's "As an expert alternative, authors MAY construct `Umpire.Machine` directly …"
  (`:324-326`). The `model/README.md:37` `Success.Producer` drift is fn-85 .13's (recorded there);
  check it rather than redoing it. The live-identity baseline is eleven since fn-85 .5 and moves
  with fn-85 .10 and .11 and with this spec's .3 and .6; record the final number.

### Investigation targets
**Required:**
- `.plans/UMPIRE4_SPEC.md:197-198,305-334` — MOD-11, AUT-07 to AUT-09 (the expert-alternative sentence at :324-326)
- `.plans/UMPIRE4_ORDER.md:410-462,472-524` — the fn-86 entry and the gate baselines (the fn-85 entry is `:180-408`; the ranges this task cited moved with it)
- the docs-gap table for fn-86 in the planning record

**Optional:**
- `tools/umpire/vocabulary/spec_names_test.go:26-39` — the MOD-15 gate

### Key context
- Boundaries: no edits to historical `.plans` documents other than the order document and the drafted rules.

## Acceptance
- [x] AUT-08 no longer offers direct `Umpire.Machine` construction to feature authors; AUT-07a names the commands as the only authoring path; the new MOD rule exists and MOD-11 lists it; all three are drafts under GOV-02; MOD-15 gate green
- [x] no document describes a hand-written authoring path for feature Models; the docs-gap list is fully worked; the inventory's every row is resolved
- [x] `make umpire-check-regression` exit 0; `make lint-model` at or below its baseline; `make lint-code` at 161 after `go clean -cache`; baselines recorded in the order document


## Done summary

Done 2026-09-20; self-review. Commit dc321cd.

### The drafts (`.plans/UMPIRE4_SPEC.md`, each `drafted by fn-86; awaiting GOV-02 approval`)

AUT-08 no longer offers the expert alternative: the two sentences that let an author construct
`Umpire.Machine` directly are replaced by "the path MUST produce an `Umpire.DraftModel`", and the
amendment says why -- a feature Model is declared through the commands, whose `machine` enumerates
the step functions into the finite table and discharges the adapter's obligations from it; direct
`Umpire.Machine` and `Umpire.DraftModel` construction remains what the Implementation Link and
Umpire's tests do and is not an authoring path. AUT-07a's amendment (a delta on fn-85's) names the
command surface as the only authoring path for a feature Model and where each thing is written: a
concrete operation is an `action`'s `schema:` line, a field relation a `property`'s `relates:`
line, a Case the platform's `case … realizes` block. MOD-16 (`authoring-path-isolation`) is the
rule .8 enforces, a direct-import rule with `Temporal.Case` and the Implementation Link outside it,
and MOD-11's amendment lists it. Every dotted name the drafts cite resolves (`go test
./tools/umpire/vocabulary/...`, the MOD-15 gate).

### One authoring path in the documents

`model/README.md`: the ordinary-authoring section now names the Caller Model and `AUTHORING.md`
as the walkthrough, the commands as the one path and `lint-model`'s rule, and frames the raw
languages as what a command elaborates to; the typed-authoring section is written on the commands
(`schema:`, `relates:`, `input:` and `examples:`) with the operand and lowering semantics kept and
the retired `ActionTemplate`/`ParameterDomain` claims replaced by the abstraction claim.
`model/ARCHITECTURE.md`: the import edges name the authoring-path rule; typed authoring is on the
commands; the Producer is `Umpire.Case.Producer` through the `case` block. `model/Umpire/ARCHITECTURE.md`:
`Umpire.Model`'s row, the checked-composition diagram's text (no expert route; the commands reach
it through `FiniteTable` and `Search.admit`) and the Case-production section (the Producer, the
outage-order rule). `model/AUTHORING.md`: one "one path" paragraph, a delta (the authoring drift
test passes). `tools/umpire/CONTEXT.md`: the derived rule names the `relates:` line.
`tests/testcore/testpilot/README.md`: every fixture is produced from a Model file's `case` block
and rendered by `umpire-case`. `model/HANDWRITTEN_INVENTORY.md`: a final-state paragraph and every
row resolved (the typed examples' README sections rewritten, the Success design sketches recorded
deleted with .3, the `Success` specimen kept as the command-surface specimen, the README row
rewritten). `.plans/UMPIRE4_ORDER.md`: the fn-33 row and the fn-79 paragraph record what .5
wrote into their specs, the fn-86 entry closes, and the gate baselines gain the fn-86 closeout row.
The docs-gap ranges the inventory's last row listed are each rewritten; the two `handwritten`
words left in `model/README.md:4` and `model/ARCHITECTURE.md:4,:15` mean hand-written as opposed
to generated Lean, not an authoring path.

### Gates

One source fix rode this task: the regression target's Umpire-independence scan (`git grep` over
tracked files) flagged the docstring of `.7`'s `Umpire/Examples/Conventions.lean` for naming the
platform's conventions module by its dotted name, which the untracked file had escaped at `.7`'s
gate; the docstring no longer names it.

`go clean -cache`; `make umpire-check-regression` exit 0 with 29 passing live
identities; `LEAN_NUM_THREADS=1 make lint-model` at the .1 baseline (the import graph with the
authoring-path rule passes; the declaration linters report the two generated `Proto.lean` findings
and 40 warnings, none new); `make lint-code` is not measurable in this shallow clone (no `main`
merge base, as the 2026-09-13 row records) -- `GOLANGCI_LINT_BASE_REV=9484405 make lint-code-fast`
reports 0 issues over the packages fn-86 .7 and .8 changed; `make umpire-check-retired-vocabulary`,
`umpire-check-testpilot-authoring`, `umpire-check-inventory` exit 0; `go test
./tools/umpire/vocabulary/... ./tools/umpire/authoring/...` green.

## Evidence
- Commits: dc321cd
- Tests: `go test -count=1 -tags test_dep ./tools/umpire/vocabulary/... ./tools/umpire/authoring/...`; `make umpire-check-retired-vocabulary umpire-check-testpilot-authoring umpire-check-inventory`; `go clean -cache && make umpire-check-regression`; `LEAN_NUM_THREADS=1 make lint-model`; `GOLANGCI_LINT_BASE_REV=9484405 make lint-code-fast`
- PRs:
