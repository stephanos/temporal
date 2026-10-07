---
satisfies: [R8, R9]
---
# fn-139-actor-grouped-rules-per-rpc-actions.1 Shared Outcome/Rejection and rejects(r).because(text) in framework and lifter; parameterized outcome admitted by the Go reader (proof point)

## Description
Adds the shared `Outcome` and `Rejection` types and the `rejects(r)` / `.because(text)` rule effect, in the framework and the lifter, beside today's per-machine outcomes. No Model adopts them yet (tasks .5 and .6 do). This is the spec's early proof point: it shows a parameterized outcome survives the lifter and the Go IR reader before any Model depends on it.

**Size:** M
**Files:** model/umpire/Syntax.scala (the shared types inside a namespace object, e.g. `outcomes`, holding `enum Rejection`, `enum Outcome { accepted; rejected(why: Rejection) }` and the accepted-outcome given; `rejects` with a `Core form:` comment; `.because` as a member of the value `rejects` returns), model/check/SyntaxRule.scala (sugar name `rejects` only), model/irgen/Syntax.scala (lower `rejects(r)` and `.because(t)`; `outcomeOf` at :378-396 accepts the framework's given), model/irgen/testdata/lifts/Rejections.scala + expected/rejections.json (new), model/irgen/testdata/lifts/Rejects.scala + expected/rejects.txt (refusals), model/umpire/Rules.test.scala, tools/umpire/ir (reader test, only if admission needs a change)
**Touches:** [model/umpire/Syntax.scala, model/umpire/Rules.test.scala, model/check/SyntaxRule.scala, model/irgen/**, tools/umpire/ir/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Put the shared types in the framework, inside a namespace object that `import umpire.*` does not open (spec, Architecture: where the shared types live). A top-level `umpire.Outcome` breaks about 55 files: every same-file `enum Outcome` (Worker.scala:28, Nexus.scala:20, Standalone.scala:55, about 50 irgen fixtures) becomes ambiguous, and files that read `Outcome` from their package in another file (activity product, TrustingCaller.scala:26) silently switch types. Adopters write an explicit import. Nothing that does not adopt the types changes.
- Place the accepted-outcome given where SyntaxRule allows it. `Ok` is sugar, and a core model/umpire file may not name it (SyntaxRule.scala:19-23, 276-279), so define the namespace in Syntax.scala with `Core form:` comments. Make the given reachable without a feature-level `given Ok` (for example the `Outcome` companion, which is in implicit scope). Extend the lifter's `outcomeOf` (irgen Syntax.scala:378-396), which today resolves only givens in lifted sources, to recognize the framework's given.
- `.because` is a member of the value `rejects(r)` returns, never top-level sugar. A core top-level `because` already exists (Machine.scala:13-14), and registering `because` in `sugarNames` would flag it.
- `rejects` must name `Outcome.rejected`, and the lifter recognizes sugar by owner. The cases loosely follow gRPC: `notFound`, `alreadyExists`, `failedPrecondition`, `invalidArgument` (spec, Architecture). The Temporal-side mapping to RPC codes is task .8.
- `rejects(r)` is an effect value used on the right of `~>`: the state is unchanged, the outcome is `Outcome.rejected(r)` and no facts are recorded. `.because(text)` sets the step's `because`. The IR row already carries `because`, and `reject(outcome, state)` writes it empty today (model/irgen/Syntax.scala:49-50). `rejects` compiles only where the machine's `O` is the shared `Outcome`. Keep `reject(outcome, s)` working unchanged.
- Lifter: lower `rejects(r)` and `rejects(r).because(t)` to exactly the `Step` that `reject(Outcome.rejected(r), s)` lowers to, with `because` set to `t` when given.
- Proof (the early proof point): a fixture machine whose `O` is the shared `Outcome` lifts, and the IR it produces passes the Go reader and admission (`tools/umpire/ir`), including refinement's outcome-name match between a product and a System that both use `Outcome`. Today every outcome enum is flat. If a parameterized outcome case is refused anywhere (outcome catalogs, `Refines.VisibleOutcomes`, lint), stop and report to the owner which layer refuses and what change it would need. Do not flatten the case names without a decision (spec, Early proof point).

### Investigation targets
**Required:**
- model/umpire/Syntax.scala:14-30: `enter`, `stay`, `reject`, `Ok`
- model/irgen/Syntax.scala:378-396: `outcomeOf`
- model/check/SyntaxRule.scala:19-23, 276-326: where sugar may be defined and named
- model/temporal/capabilities/Capabilities.scala:12 and model/irgen/Claims.scala:532-545: the capability `rejected` argument
- model/irgen/Syntax.scala:20-60: the `reject` lowering
- model/temporal/features/nexus/Nexus.scala:20 and features/activity/standalone/Standalone.scala:55: today's outcome enums
- tools/umpire/check/grouping_spike_test.go:35,84-118: how outcomes are compared by name across refinement
- tools/umpire/ir/validate.go:680-700: refinement outcome fields

**Optional:**
- model/irgen/testdata/lifts/Rejects.scala and expected/rejects.txt: refusal fixture style
## Acceptance
- [ ] Fixture: rules in today's case forms using `always ~> rejects(Rejection.notFound)` and `… ~> rejects(Rejection.failedPrecondition).because("…")` lift to JSON identical to the same rules written with `reject(Outcome.rejected(…), s)`, except that `because` holds the text where it is given.
- [ ] A fixture machine on the shared `Outcome` uses `enter` and `stay`, and passes a capability argument `rejected = Outcome.rejected(Rejection.notFound)`, and lifts.
- [ ] Every existing Model and irgen fixture compiles and lifts unchanged with the shared types present (no ambiguous `Outcome`).
- [ ] The fixture's IR (a product and a System both on the shared `Outcome`) passes the Go reader and admission tests in `tools/umpire/ir`, or the task stops and reports the refusing layer per the Approach.
- [ ] Refusal fixtures, one message each: `rejects` in a machine whose outcome is not the shared `Outcome` (compile error recorded in the test), and `.because` applied twice. `make lint-model` passes (SyntaxRule accepts where the types and the given are defined).
- [ ] Rules.test.scala: a `rejects` row keeps the state and answers `rejected(r)`. `scala-cli test model/umpire` and the irgen fixture suite pass.
## Done summary
Added the shared outcomes to the framework (`umpire.outcomes.{Outcome, Rejection}`, `Outcome = accepted | rejected(why: Rejection)`), the accepted-outcome given in `Ok`'s companion, and the rule effect `rejects(why)` / `rejects(why).because(reason)`, with the lifter lowering it to exactly the step `reject(Outcome.rejected(why), s)` gives. The early proof point holds: a product and a refining System both on the shared Outcome lift, pass `tools/umpire/ir` admission, interpret with outcomes keyed `accepted`, `rejected-notFound`, ..., verify the refinement and a capability Property comparing `after.outcome == rejected`, and umpire-lint reads them. No layer refused the parameterized case.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: claude-opus-5-5 at high (lane D)

### What changed
- model/umpire/Syntax.scala: `object outcomes` (enums `Rejection` notFound/alreadyExists/failedPrecondition/invalidArgument, `Outcome`), `object Ok { given Ok[outcomes.Outcome] }`, `def rejects[S](why)(using Firing[S, outcomes.Outcome, ?, ?]): Rejects[S]`, `final class Rejects[S]` (a function `S => List[Step[S, Outcome, Nothing]]` with member `because(reason)` returning a plain function, so a second `.because` does not compile). `rejects` outside a machine on the shared Outcome is a compile error with a custom message.
- model/irgen/Syntax.scala: `outcomeOf` recognizes the framework given (owner `umpire.Ok` companion) and lifts `Outcome.accepted`; new hook `rejection(effect, state)` lowers `rejects`/`.because`. model/irgen/Declarations.scala `lowered.chain`: tries `rejection` before `effect(r)` (two-line change).
- model/check/SyntaxRule.scala: sugar name `rejects`.
- Fixtures: lifts/Rejections.scala + expected/rejections.json + rejections.waivers.json (new); crossed/Rejections.scala (two compile refusals); Fixtures.test.scala (fixture roots, `waived`, crossed positions, a test comparing DoorSystem's step functions with DoorProduct's after inlining the rejecting effects); Rules.test.scala (rejects row keeps the state, answers rejected(r), carries because); tools/umpire/ir/fixtures_test.go (admission + `TestSharedOutcomesKeyEachRejection`).

### Decisions (owner unavailable)
- The accepted given lives in `Ok`'s companion, not `Outcome`'s: effects written `enter(d.copy(...))` without a result type search `Ok[?O]`, whose implicit scope holds only `Ok`'s companion. A Model's own lexical `given Ok` still outranks it, so existing Models are unchanged.
- `.because` twice is a compile error (type-level), recorded as a crossed position rather than a lifter message.
- The capability argument uses a fixture-own kind in a capabilities section (not temporal.capabilities.Closable with the old function form), so fn-134.3/.4 removing the law catalog does not break this fixture.

### Expected IR delta (batch regeneration)
None in model/ir: no Model adopts the shared types yet. Only the new lift fixture's expected files (committed here).

### For later tasks
- Adopt with `import umpire.outcomes.{Outcome, Rejection}`. Inside a machine object, `Outcome` resolves to the machine's inherited `type Outcome` member, so a machine-local import of `Outcome` is reported unused (-Wunused under lint-model-models); import `Rejection` only there.
- `rejects` needs the block's `Firing`, so it works in rules and derivation `on` blocks only, never in an `effects` def.
- Lint: on a shared Outcome every Rejection the machine never produces is an `unproduced` umpire-lint finding (`outcome rejected-invalidArgument is produced by no reachable state`). fn-139.5/.6 will see such findings for activity/Nexus/worker and need acceptances in `<file>.lint.json` or a lint change.
- The shared types' IR positions read `umpire/Syntax.scala:<line>` (no `model/` prefix): the lifter maps prefixes only for lifted (non-framework) TASTy. Harmless for admission; the batch regeneration will show it for Models that adopt.
- The outcome type is named `umpire.outcomes.Outcome` in IR; values key as `accepted`, `rejected-<rejection>`.

### Follow-ups (not built)
- None required.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f04b73fff6
- Tests: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire (20 passed), UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (own fixture expectations only: rejections.json, rejections.waivers.json), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (rc=0), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check (rc=0), mise exec -- go test -tags test_dep -count=1 ./tools/umpire/ir/ -run 'TestLiftedModelsAreAdmitted|TestSharedOutcomesKeyEachRejection|TestRulesLowerToTheCoreTables' (ok), temporary (not committed) tools/umpire/check test: refinementOf(doorSystem) ok; Check verified doorSystem refines doorProduct and doorSystem.overIsRejected, go run ./tools/umpire/cmd/umpire-lint --tables on a copy of rejections.json: reads the shared outcomes (fixture findings only), make lint-model-models lint-model-irgen lint-model-irgen-lifts lint-model-check lint-model-syntax (each rc=0, one by one); scala-cli fmt --check (rc=0), GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: the model gate runs at the batch's single regeneration
- PRs: