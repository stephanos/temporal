# fn-85 plan review — round 1

Spec: fn-85-model-side-effects-as-typed-actions-and — Model side effects as typed actions and run
query sets
Plan: 13 tasks, broken out 2026-09-12; every requirement R1 to R13 has a task and every task declares
its `satisfies`.
Reviewed against: the spec markdown, the thirteen task records, and the tree at
`claude/umpire-order-spec-0ylk31`.

**Verdict: REVISE.** One blocker and one unachievable acceptance criterion; the rest of the plan is
sound, correctly ordered, and traceable. The blocker is not a design problem — it is a gate the plan
does not mention that every protocol task will hit.

Conducted in-session: `flowctl` is not installed in this cloud session, so there is no plan-review
receipt in the spec record and no recorded review backend. Record the verdict from this document when
a session with the CLI is available.

## Blocker

**B1. The protocol-migration oracle blocks fn-85's protocol and fixture changes, and no task owns
it.** `common/testing/testpilot/internal/protocolmigration` pairs each frozen pre-fn-87 baseline
fixture with its regenerated counterpart one to one. It fails on any difference no `Declared` step
explains, on any regenerated fixture absent from `Added`, and on a deleted baseline fixture with
`baseline fixture … has no regenerated counterpart` — there is no removal list
(`equivalence.go:578-623`). CI runs `go test -count=1 -tags test_dep ./common/testing/testpilot/...`
(`.github/workflows/umpire.yml:35`), so this is a CI gate, not an optional local check.

Four tasks walk into it:

| Task | Change | Oracle effect |
| --- | --- | --- |
| .4 | a repeated named-value field on the correlated transition; `correlated.json` regenerated | needs a declared step |
| .8 | three instruction arms; the Nexus realization re-bound onto them | needs a declared step |
| .9 | observation declarations in `program.proto`; rules reference them by name in `contract.proto` and `correlated.proto`; every fixture rewritten | needs several declared steps |
| .10, .11 | `async-nexus-case.json` deleted; seven `<set>-<query>-case.json` fixtures added | needs `Added` entries **and** a removal mechanism the package does not have |

No task lists `protocolmigration` in its files or acceptance. Decide it once, before .4, in .1 or in a
new first task:

- **Retire the oracle** — its declared subject is fn-87 ("fn-87 changes the Testpilot protocol without
  changing any Verdict. This package proves it"), and fn-87's completion review discharges that. Delete
  the package and the frozen baseline tree, and drop the `.gitignore` negation and the
  `protocolMigrationBaseline` exemption in `tools/umpire/internal/retiredvocabulary/check.go:800`.
  Smaller, and honest about what the oracle was for.
- **Or extend it** — every protocol and fixture task appends declared steps, `.10` adds a `Removed`
  list beside `Added`, and the package README's "Final mapping" section grows an fn-85 table. This
  keeps a pre-fn-87 baseline meaningful across two more specs' worth of shape changes, which is a
  liability the spec has not argued for.

Either way the choice belongs in the plan, because `.10`'s fixture deletion cannot be landed without
it.

## Findings

**F1 (P2). `.1`'s acceptance criterion "a grep for `nexus` in `Umpire/` is empty" cannot pass.** Three
files already match: `model/Umpire/Case/Tests/Producer.lean:15-30` uses `async-nexus` as
fixture-identity test data, and `model/Umpire/Exploration/Engine.lean:95` and
`model/Umpire/ARCHITECTURE.md:152` mention Nexus in prose. Restate it as what it means: no
Nexus-specific branch in `model/Umpire/Case/Producer.lean`, and no RPC, instruction or event-kind name
in non-test Lean under `model/Umpire`.

**F2 (P2). `.3` is over-sized for one task.** It carries a `Finite` class with a deriving handler for
structures of finite fields; the `machine` command with `for:`, `ends:`, `state:`, `setup:`, `timers:`,
`steps:` and `evidence:` and its located diagnostics; the step-function enumerator through the
`Meta.evalExpr` bridge; `count` fields as saturating `Fin (bound+1)`; reserved non-drivable actions for
system and timer rows; stuck-state witness diagnostics; Limits accounting in `Umpire.Search` and
`Search/Admission`; the migration of 18 `model` declarations and 33 `#guard_msgs` specimens; and, in
its last acceptance bullet, the separate decision that `property` bodies become predicates. Its three
planned commits are three tasks: the finite enumerator with the fingerprint-equality prototype (whose
stop condition — fall back to rows — needs to be actionable on its own); the command with its
diagnostics and specimen migration; and property predicates. As written a re-anchored worker is
likely to exhaust its context mid-task, and the prototype's fallback cannot be taken cleanly if the
same task must also land the command.

**F3 (P3). The property-predicate change has no requirement row.** R3 reads "The machine command and
its rows" in the coverage table and the spec's R3 text is about the machine command; the predicate form
of `property` appears only as a Planning decision and a `.3` acceptance bullet. Nothing in the coverage
table fails if it is dropped. Amend R3's text to name it, or give it its own R-ID.

**F4 (P3). `.13` closes fn-83's six blocked tasks "through `flowctl`".** The CLI is not installed in
this environment. Name the fallback — edit the six task records in `.flow/tasks/` to the stored shape —
or that acceptance bullet cannot be met here.

**F5 (P3). `.5`'s acceptance rests on `tests/testpilot_async_nexus_case_test.go`, which `.10` deletes
or re-points.** That is the right order, but say in `.5` that the async-Nexus live test is the
temporary carrier for per-switch-value runs, so a later reader does not read its removal as a lost
gate.

## What the plan gets right

- The early proof point is a real stop condition with a measurable outcome (Query 2 equal to the
  checked-in fixture with identities masked) and it is measured on the assembly, before any syntax
  exists. Records-before-syntax is the right order for that.
- Per-action-class binding is justified from the tree: today's `complete` is a handler-party action
  realized as a controller instruction, so a per-party binding would be wrong by construction.
- The step-function decision (over the row grammar) carries its own prototype and fallback, with
  fingerprint equality as the pin and elaboration time measured against the Race baselines.
- Ordering is right where it matters: `.9`'s observation declarations before `.10`'s Model, `.5`'s
  switch plumbing before the per-Query live tests, and the template and `case`-command deletion last
  in `.11`, once every Case comes from a set.
- Every task's acceptance names a gate or a pinned specimen rather than a description, and the
  fixture-level claims ("the diff listed in the receipt") are checkable.

## Revisions applied, same session

Round 1's findings were applied to the plan rather than left for a later session:

- **B1** is now the first thing `.1` does, in its own commit: the oracle and its frozen baseline are
  deleted, with the `.gitignore` negation and the vocabulary gate's exemption for that tree removed.
  `.4`, `.8`, `.9`, `.10` and `.11` each carry a line saying no declared mapping step is needed and
  that the conformance `expected.json` pins are the Verdict net. The retire-or-extend choice is written
  out in `.1`'s Approach with the reasoning, so a worker does not have to re-derive it.

  This does undo a one-line `.gitignore` change fn-87's closeout made in the same session. That is not
  churn: the tree is tracked today and ignored today, which is wrong however long it lives, and `.1`
  removes the tree and its negation together.

- **F1** `.1`'s acceptance now reads "no Nexus-specific branch exists in it, and non-test Lean under
  `model/Umpire` names no RPC, instruction or event kind", and names the three pre-existing matches so
  nobody re-litigates them.

- **F2** the former `.3` is three tasks: `.3` is the `Finite` class, the step-function enumerator and
  the fingerprint-equality prototype that carries the stop condition (S); `.14` is the `machine`
  command, its witness diagnostics, the Limits accounting and the migration of the 18 `model`
  declarations and 33 specimens, with `model` retired (M, two commits); `.15` is `property` bodies as
  predicates (S). `.14` depends on `.3` and `.15` on `.14`, recorded in their task records.

- **F3** the spec's R3 now names the predicate form of `property` with its own error surface, and the
  coverage table maps R3 to `.3`, `.14`, `.15` and R4 to `.14`, `.4`.

- **F4** `.13` names the direct-edit fallback for closing fn-83's six tasks where `flowctl` is absent.

- **F5** `.5` says the async-Nexus live test is the temporary carrier that `.10` replaces.

Round 2 should be an independent read: the same session both reviewed and revised this plan, which is
exactly the cross-model independence flow-next's review step exists to provide.
