---
satisfies: [R1, R2, R3]
---
# fn-139-actor-grouped-rules-per-rpc-actions.2 Framework: from(declarer) with leading import, when case forms, multi-action on, overlap across blocks, beside the old forms

## Description
Framework half of the new rule forms, added beside the old ones so every Model still compiles: `from(declarer) { import declarer.*; on(…) { … } }`, the `when` case forms, multi-action `on(a, b, …)`, and the overlap check keyed by (class, phase) across blocks instead of one block per target. The lifter half is task .3. Rule-case `in` stays until task .7 converts the Models and removes it.

**Size:** M
**Files:** model/umpire/Syntax.scala (`Rules.from`, `when` overloads, `on` arities 2 to 4, `block` and `bind` changes, comment above `Rules`), model/umpire/Machine.scala (overlap message, only if rule indexing needs it), model/check/SyntaxRule.scala (sugar name `when`), model/umpire/Rules.test.scala
**Touches:** [model/umpire/Syntax.scala, model/umpire/Machine.scala, model/umpire/Rules.test.scala, model/check/SyntaxRule.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `from(d: AnyRef)(body: => Unit)` is a plain member of `Rules`, not `inline`. It sets a `from` marker the way `block` sets `open` (Syntax.scala:279-293), with a `finally` reset. It refuses a `from` in a `from` and a `from` inside an `on`. The author writes `import d.*` as the block's first statement (spec, Architecture: how `from` scopes bare names; the probe confirmed it on 3.9.0).
- Declarer check: `ActionDecl` does not record the object that holds it (worker actions are declared `action(process)` in `object worker`, and timers are `Actor.system`). So `from` collects the `Action` values among `d`'s public vals once, by Java reflection on the object, and `block` refuses an action whose decl is not among them, naming the action and `d`. The lifter makes the same check by symbol owner (task .3).
- `when(first, rest*)`, `when(set)` and `.where(g)` mirror `in` (Syntax.scala:322-330) and build the same `Case` guard. Give the heading text `when(…)`. If fn-136.4 has landed `in[R]`, add `when[R]` beside it with the same evidence. If fn-136.4 already spells it `when[R]`, there is nothing to add.
- Multi-action `on`: fixed-arity `inline` overloads for 2, 3 and 4 actions or classes, each `codeOf` per argument. `inline` varargs render `codeOf` unreadably (probe). Each named target runs the cases once with its own `Firing`, so case thunks must stay pure. Refuse a target named twice in one `on`, and a `disabled` target (the existing `never` check, applied per target).
- Drop the "written twice: an action, or a class of it, has one block" refusal. The overlap check in Machine.scala:215 already compares every earlier rule of the same decl over its classes, so an overlap across blocks is refused there. Add a test that a whole-action block and a class block of the same action that overlap are refused.
- Preserve every existing comment, and add `Core form:` comments for the new forms (SyntaxRule requires them).

### Investigation targets
**Required:**
- model/umpire/Syntax.scala:240-370: `Rules`, `block`, `bind`, `on`, `in`, `disabled`
- model/umpire/Machine.scala:205-245: `overlap`, `writtenAction`
- model/umpire/Action.scala:1-80: `ActionDecl`, `Actor`
- model/umpire/Rules.test.scala:150-170: the overlap test style

**Optional:**
- model/temporal/features/activity/standalone/Standalone.scala:76-120: the declarers `client`, `worker`, `timers`, `deadline`

## Acceptance
- [ ] Rules.test.scala: rules written with `from` + import, `when(…)`, `when(set)`, `when(…).where(g)` and `on(a, b)` build the same rule table as the equivalent `on` + `in` rules.
- [ ] Refusals, each tested with its message: an `on` in an `on`, a `from` in a `from`, an action the `from`'s declarer does not declare (named with the declarer), a target named twice in one `on`, a disabled target in a multi-action `on`, and an overlap between two blocks of one class (naming the class, both rules and a witness state).
- [ ] An action or class may appear in several `on` blocks when the cases do not overlap.
- [ ] All existing Models still compile unchanged. `scala-cli test model/umpire` and `make lint-model` pass.


## Done summary
Added the framework half of the new rule forms beside the old ones: `from(declarer) { import declarer.*; on(...) { ... } }`, the `when(...)`, `when(set)` and `when(...).where(g)` cases, and `on(a, b)`, `on(a, b, c)` and `on(a, b, c, d)` over actions or classes. The "written twice" one-block refusal is gone, and the existing overlap check (Machine.scala `overlap`, unchanged) now guards repeats across blocks. All existing Models compile unchanged.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: claude-opus-5-5 at high (lane D)

### What changed (model/umpire/Syntax.scala, `Rules`)
- `def from(declarer: AnyRef)(body: => Unit)` is plain, with no `inline`. It sets a `within` marker (declarer name plus declared `ActionDecl`s) with a `finally` reset. The declared set comes from Java reflection over the object's zero-argument public methods that return an `Action`.
- `block` refuses an action the enclosing `from`'s declarer does not declare. The `blocks` field and the one-block refusal are removed.
- Multi-target `on` takes fixed arities of `inline a: Action[?] | Class`, with cases `Firing[S, O, F, EmptyTuple] ?=> Unit`, so effects read the state alone. `each(...)` refuses duplicates before running any block, then runs the cases once per target through `block`, which applies the disabled check to each target.
- `when[Q](first, rest*)(using PhasesOf[P, Q])` and `inline when(inline set)` mirror `in` exactly, with heading text `when(...)`. `phases(set, code, word = "in")` gained the heading word. The `PhasesOf` message now says "in and when". No `when[R]` form is added: fn-136.4 can add `when[R](using TypeTest[P, R])` as a third overload, as its probe showed for `in`.
- model/check/SyntaxRule.scala: sugar name `when` (`from` was already listed).
- Rules.test.scala: the grouped-vs-plain table equivalence, one action in several blocks, and each refusal with its message.

### Messages (task .3's lifter must match these exactly)
- `on(<action>) sits in on(<open>): a block holds cases alone` (existing)
- `from(<d>) sits in on(<open>): a block holds cases alone`
- `from(<d>) sits in from(<outer>): a from holds on blocks alone`
- `on(<action>) sits in from(<d>), and <d> declares no <action>: a from holds the blocks of the actions its declarer declares`
- `<action> is named twice in one on: name each action, or class of it, once`
- `<action> is disabled and fired by a rule` (existing, applied to each target)
At runtime each message is prefixed `requirement failed: ` because these are `require` checks.
- Overlap across blocks: the existing message, e.g. `acrossBlocks fires turn-down by two rules in Lamp(off,0): rule 1, when(off), and rule 3, when(...).where: ...`.

### Expected IR delta (batch regeneration)
None: rule headings are not written to model/ir, and no Model uses the new forms yet.

### For later tasks
- `<d>` in messages is `Actor.name` for an Actor, else the object's name with its first letter lowered (`objectName`).
- `import d.*` inside `from(hand) { ... }` also imports Actor members such as `name`; `-Wunused:imports` is satisfied once any action is used.
- A `when(set)` heading shows `codeOf(set)`, which for a def renders as an eta-expanded lambda, as `in(set)` always did.
- `rejects(...)` (fn-139.1) works inside multi-target `on`, since its Firing is `Firing[S, Outcome, ?, ?]`.

### Follow-ups (not built)
- None.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a2af47e109
- Tests: /tmp/laneD-build.sh: make model/build/model-scala.jar (every Model compiles against the new framework), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire (25 passed), make lint-model-models lint-model-check lint-model-syntax (each rc=0); scala-cli fmt --check (rc=0), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check: INCONCLUSIVE, GateSuite 'a check lints the settled IR before the Go checks' timed out at its 30s munit limit under host load ~130-300 (it runs the gate four times with stand-in tools; this task touches only the SyntaxRule sugar list); every other test passed; rerun on the lane tree after fn-139.3 (which carries this SyntaxRule change unchanged): model/check rc=0, GateSuite passed, model/irgen fixture suite: rc=0 on the lane tree after fn-139.3 (the lifter is unchanged in .2), GATE_SKIPPED:umpire-check-model:batch - DSL batch rule
- PRs: