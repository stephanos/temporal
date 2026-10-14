---
satisfies: [R1, R3, R4]
---
# fn-120-adopt-what-quint-does-well-named.1 Add named-choice declarations, inert IR names and Quint export

Touches: [model/umpire/**, model/lifter/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, model/gate/**, tools/umpire/model/**, tools/umpire/export/**, model/ir/**, model/SEMANTICS.md]

## Description
Start only after fn-112.11 completes Query.total schema, binding and linked API jar regeneration. Its same-spec prerequisites are fn-112.3, .4 and .5; the conductor checks fn-112.11 completion as the cross-spec entry gate. Finish before fn-112.6 rewrites branches. Settle Part A syntax and semantics against accept/stay and captured val names. The construct labels existing alternatives without adding or reordering results. Keep unnamed multi-result lists readable during migration.

**Size:** M
**Files:** model/umpire step declarations, model/lifter step lifting, IR schema and generation, Go reader/goldens, Quint export.

### Approach
- Use the fn-112.1 original-baseline harness: permit only inert choice names on existing result alternatives; prove exact tables, Definition IDs, fingerprints, Query answers, exploration identities and Case bytes.
- The four-field step record and `stepList` stay as they are. The name rides on a new field of an existing message, unset in the baseline as fn-112.1's harness requires; `SEMANTICS.md` says where it lives and that evaluation ignores it.
- Add choice metadata to the descriptor that already contains Query.total, then regenerate both IR bindings and the linked API jar. Extend historical descriptor/wire coverage in tools/umpire/model/schema_test.go for both current fields without replacing captured historical bytes.
- Specify that named alternatives do not multiply Query.total; test alongside fn-112.11's static formula.
- Export the same ordered result list to Quint with each alternative's inert name on its record. Keep nondeterministic result-index selection in the checker action, after the pure step function returns; preserve all alternatives, order and agreement rows.

### Investigation targets
- model/umpire step declarations, model/lifter/Declarations.scala, tools/umpire/model/schema_test.go, tools/umpire/export/quint.go, model/gate/Gate.scala.

## Acceptance
- [ ] R1 names existing alternatives once and refuses duplicate or singleton choices at the Scala line, while unnamed branching still lifts during rollout.
- [ ] fn-112.11 is complete before this task changes the schema; the regenerated descriptor and linked API jar contain both Query.total and choice-name metadata, with historical wire compatibility retained.
- [ ] R3 original-baseline comparisons permit only choice-name metadata; result count/order, behavior, IDs, tables, fingerprints, answers, exploration identity and Case bytes stay exact.
- [ ] R4 Quint's pure step function returns the full ordered named-result list; the checker action selects a result nondeterministically by index, and agreement rows stay exact. An unexpressible name is reported at its source.
- [ ] A choice fixture proves branch count is excluded from Query.total; both current descriptor fields have compatibility coverage and the linked API jar is regenerated.

## Done summary
Added named choices: `val committed = choice` declares a name once, and `choose(committed -> accept(...), redelivered -> stay(s).because(...))` names the alternatives of a step that can go more than one way. Each name rides on the step record's construct (`Construct.choice`, IR field 4) and is inert: tables, rows and result order, Definition IDs, fingerprints, Query answers and totals, exploration and Model identities, and Case bytes are the same with or without names. The Quint export writes each name (`f_choice`) on its record of the ordered result list. No production Model was converted; model/ir, model/cases and lifts/expected/*.json are byte-identical.

**What changed**
- **Core DSL** (`model/umpire/Machine.scala`): `Choice`, `choice` and `choose(first, second, rest*)`. `choose` is core, not sugar, and no sugar spelling was added.
- **Lifter** (`model/lifter/Expressions.scala`, beside `because`): each alternative must lift to one step-record construct, which gets the token val's simple name.
- **Refusals:**
  - Eight lifter refusals: a name used twice (by one token, or by two vals with the same name); an alternative that is a helper call, `disabled`, two steps or an `if`; a token no val declares; an alternative kept in a val.
  - A one-alternative choose does not compile (`crossed/OneChoice.scala:18:62`).
- **Fixtures:** `lifts/Choices.scala` pairs each named function with an unnamed twin, and a test proves equal IR apart from `choice`. The expected inventory is closed, so this is an assertion test, not a new expected JSON.
- **Go reader:**
  - `Value.Choice` and `Result.Choice` (`json:"-"`) are carried as `Because` is.
  - Two Model errors: a choice on a non-step construct; two results of one row with one name.
  - A channel's redelivery is unnamed.
  - `WithoutChoiceNames` strips names from the explore candidate digest and the conformance Model identity, together with `WithoutTotals`.
- **Quint:** `f_choice` on step records, `""` for composed records. Agreement compares names. A name with `"`, `\` or non-printable-ASCII is refused at its construct.
- **Schema and baseline:**
  - `Construct.choice` is in schema_test.go's closed `schemaAddedFields` beside `Query.total`, and set in `schemaAddedSupplement`.
  - original.json's `inert_fields` lists both fields.
- **Docs:** SEMANTICS "Named choices", README, export README. The spec's API contract records the final spelling.

**Decisions (autonomous unless noted)**
- **Names are tokens, not enum cases.** An enum used only for names would become an IR type the Model does not otherwise need, and the names of different step functions are not one closed set.
- **Choose takes at least two alternatives.** So a singleton is a compile error at its line.
- **The name sits on the step construct, not on `ListOf`.** It is "on each result", and travels with the value through lets, calls and branches.
- **A redelivered result has no name.** Otherwise any named receiver on a duplicating channel would fail the duplicate-name rule.
- **Conformance Model identity ignores names.** It hashes content without positions, so it is a semantic identity.
- **Quint names are restricted to printable ASCII without `"` or `\`.** Quint 0.33.0's parser keeps string literals raw, and `"` cannot be written at all.
- **Ordering vs fn-112.11 (conductor).** The conductor ran this task in parallel with fn-112.11. Review round 1 escalated that as NEEDS_HUMAN, and the conductor chose to merge fn-112.11 first (option a). After the merge, `TestAlternativesAreNoClassesOfTheirOwn` asserts that every admission Query's `QueryTotal` is the same with one alternative or two named ones, and equals its declared total.
- **Files outside Touches:** `tools/umpire/explore/explore.go`, `tools/umpire/conformance/{conformance.go,identity_test.go}`, `tools/umpire/internal/golden/{original.json,original_test.go}`, `tools/umpire/lower/choices_test.go`, `tools/umpire/explore/choices_test.go`.
- **Parallel work:** three opus subagents built the Scala DSL and lifter, the Go reader and inertness tests, and the Quint export. I did the schema, bindings, core Go plumbing, SEMANTICS, the merge and the integration.

**Review:** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus).
- Round 1 (base e7503e9504): NEEDS_HUMAN. The P1 was that Query.total was absent; the conductor resolved it by merging fn-112.11. A P3 double space in the choice error was fixed in 5e99f91028.
- Round 2 (base ae508ad646, same receipt): SHIP.

**Deferred P3/FYI**
- Quint text literals (e.g. `because`) are written with Go escapes that Quint cannot read. No checked-in IR hits this; it predates this task.
- A choice token declared as a local val gets the generic "outside the liftable subset" refusal.
- A written exploration proposal embeds names, so its own digest moves. The candidate digest and Case identity do not.
- The worktree's copy of fn-112.11's task json still says todo. That is conductor paperwork.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 61fe05dc42, b0a5a386b0, 5e99f91028, ecb6f3619a, 4eb041b932, b267a8ca04, 267d6ab7d4
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; model/ir, model/cases, lifts/expected byte-identical), scala-cli test model/lifter (exit 0), scala-cli test model/project.scala model/umpire (exit 0), make lint-model (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 0 failures), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), local quint 0.33.0 run on a named Model agrees with Go (quint-local-quint-run.log)
- PRs: