---
satisfies: [R1, R3, R4]
---
# fn-127-simplify-the-dsls-words.1 Rename the DSL words that collide with Temporal's

## Description
Implements R1 and the word half of R3. This is a mechanical rename with zero change to the IR, Cases or Contracts. Words: `accept` → `enter`, `Accepted` → `Ok`, the realization `poll` → `readUntil`, `.setting` → `.withFields`, `always` → `everyCase`, and `umpire.realize.Outcome` → `PropertyOutcome`.

**Cross-spec entry gate:**
- fn-114, fn-118 and fn-122 are closed (fn-118.5 has rewritten the waits in every `Realization.scala`).
- Not concurrently with fn-124.8.
- fn-124.3 also edits realization surfaces; whichever lands second rebases.
- fn-126 does not start until this spec closes.

**Size:** M
**Files:**
- `model/umpire/Syntax.scala`, `model/umpire/realize/{Scripts,Realize}.scala`;
- `model/temporal/realize/*`;
- every Model and `Realization.scala` under `model/temporal`;
- `model/irgen/{Syntax,Realizations}.scala` and the lifter fixtures;
- `model/check/SyntaxRule.scala`;
- `model/README.md`, `model/SEMANTICS.md`, `.plans/DSL_OPERATORS.md`.

**Touches:** [model/umpire/**, model/temporal/**, model/irgen/**, model/check/**, model/README.md, model/SEMANTICS.md, .plans/DSL_OPERATORS.md]

### Approach
- Rename each definition in place, keeping its `Core form:` doc (update the doc's spelling). `Ok[O]` replaces `Accepted[O]` as the given that `enter` and `stay` read.
- Rewrite the call sites mechanically (sed, or a scalafix rule if one is cheaper), then format. Rename `umpire.realize.Outcome` everywhere it is imported, and drop the `Outcome as RunOutcome` alias in `standaloneactivity/admission/Queries.scala:6`.
- Update the lifter wherever it matches these names by spelling (`model/irgen/Syntax.scala` for the sugar, `Realizations.scala` for `poll`, `setting` and `always`), and its fixtures. Update `SyntaxRule.sugarNames`.
- Leave each Model's `enum Outcome` and its case `accepted` alone. R1 says why: the case is in IR type catalogs, fingerprints and Case bytes.
- Docs: replace the words in `model/README.md` and `model/SEMANTICS.md`. In `.plans/DSL_OPERATORS.md`, add one entry per rename with its reason (the Temporal word or the reserved operator it collided with).

### Investigation targets
**Required:**
- `model/umpire/Syntax.scala:1-40`
- `model/umpire/realize/Scripts.scala:40-110`, `model/umpire/realize/Realize.scala:470-490,595-605`
- `model/irgen/Syntax.scala`, `model/irgen/Realizations.scala` (grep `"poll"`, `"setting"`, `"always"`, `"accept"`)
- `model/check/SyntaxRule.scala:30-60`
**Optional:**
- `.plans/DSL_SIMPLIFICATION.md` section 4a

### Quick commands
```bash
grep -rn "accept(\|Accepted\[\|\.setting {\|always(\|\bpoll(" model --include=*.scala | grep -v /testdata/migration
make umpire-check-model && make lint-model
git diff --stat model/ir model/cases    # empty after --update
```

### Execution constraints
- Zero IR, Case, Contract or lint-finding change. If regeneration shows any diff in `model/ir/**` or `model/cases/**`, stop and find the name match that changed meaning.

## Acceptance
- [ ] `enter`, `Ok`, `readUntil`, `.withFields`, `everyCase` and `PropertyOutcome` replace the old words in the framework, kit, lifter, Models, fixtures and tests. The old definitions are gone, and each sugar keeps its `Core form:` doc.
- [ ] Each Model's `enum Outcome` keeps its case `accepted`.
- [ ] After `make umpire-check-model --update` and `make umpire-gen-cases umpire-gen-fixtures canary-gen-case`, `git diff model/ir model/cases tests/testcore/testpilot/testdata` is empty.
- [ ] The grep of R3 over `model/` and the live docs finds no old word. `.plans/DSL_OPERATORS.md` records each rename with its reason.
- [ ] The model gate, `make lint-model`, the Umpire Go tests, `make umpire-check-cases`, `make umpire-check-fixtures`, `make canary-check-case` and `make lint-code-fast` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
