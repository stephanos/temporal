---
satisfies: [R2, R9, R10, R13, R15, R16]
---
# fn-141-shrink-the-ir-generator-one-description.14 Close: remaining lifter code, rules of record, docs and counts

## Description
Remove what no path reaches, say what holds now, and report the size.

**Size:** S
**Files:** `model/irgen/**`, `model/README.md`, `model/framework/SEMANTICS.md`, `.plans/{DSL_OPERATORS,SCALA}.md`, `MILESTONES.md`
**Touches:** [model/irgen/**, model/README.md, model/framework/SEMANTICS.md, .plans/**, MILESTONES.md]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Delete lifter code nothing calls after tasks 8 to 13: value following, constant folding for declarations, the root dispatch, registries the exporter replaced.
- The syntax lint's exemption list from task 4 is empty.
- Audit the unused-spelling obligations from .2/.10. The nine direct typed constructors must remain unavailable while canonical factories/types remain usable. `Bindings` was reserved only for .3/.4's core proof mechanism; verify its unused public authoring spelling and source-reader arm are gone, and that its former oracle assertions survived. Resolve any still-open retirement here before R2/spec completion; no permanent compatibility alias or fixture-only public syntax is an accepted substitute.
- Docs: the README's account of how a Model becomes IR, the two levels and where Scala is free, and the note beside fn-113's R15. MILESTONES' Direction paragraph says the lifter reads declarations; rewrite it.
- Run `make umpire-check-backends` once and compare its answers with the run before Part B (R10).
- Report: lines of `model/irgen`, the exporter and the capture points against the starting count; every Model edit made, by kind (R13).

### Investigation targets
**Required:**
- `model/irgen/*.scala` for unreferenced definitions
- `model/README.md` (Writing a Model; Counting a Query's total, which explains why TASTy is read)
- `.plans/DSL_OPERATORS.md` rule 5 and `.plans/SCALA.md`


**Size by category (owner, 2026-10-13).** Report the lifter's lines before fn-141 (capture this at fn-141.1's start, from the tree as fn-169.1 leaves it) and after, split by category:
- declaration and DSL surface re-parsing;
- sugar matching;
- realization-kit matching;
- function-body and type lifting;
- lints and refusals.
The goal is that the first three categories are essentially gone. Anything left in them is listed with the reason.
## Acceptance
- [ ] The lifter holds function-body lifting, type lifting, the sugar expansion and the span lookup, and nothing that evaluates a declaration.
- [ ] No lifter or exporter file names a sugar; the lint has no exemption.
- [ ] The docs and rules of record match the code.
- [ ] R2's final recount finds no unused public spelling or retained source matcher kept solely for old fixture syntax; the reserved `Bindings` retirement and typed-constructor boundaries have current evidence.
- [ ] `make umpire-check-backends` answers are unchanged.
- [ ] The size report and the list of Model edits are in the done summary.
- [ ] `make umpire-check-model`, `make lint-model` and the Go suite pass; `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.

- Before/after lifter line counts by category recorded; residual declaration/sugar/kit matching listed with reasons.
## Done summary
Already superseded by fn-174; status hygiene 2026-10-18

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: