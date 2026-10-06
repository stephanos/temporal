---
satisfies: [R1]
---
# fn-138-retries-and-deadline-capabilities.1 Retries capability: kit Properties and lifter refusals (owner question first)

## Description
The early proof, and framework/kit work only: the Retries capability in fn-134's shape, its Properties in its companion, and its lifter refusals, proven on irgen fixtures. No Model declares it here; task 3 does that. Starts with the owner question below, because the Properties as the spec words them do not hold on the shapes fn-128.1, fn-128.3 and the Nexus workflow specify.

**Size:** M
**Files:** `model/temporal/capabilities/Retries.scala` (new), `model/temporal/capabilities/Capabilities.scala` (kind and fields, as reshaped by fn-134), `model/irgen/Capabilities.scala` (refusals), `model/irgen/testdata/lifts/Capabilities.scala` + `lifts/CapabilityRejects.scala` + `lifts/expected/*`, `model/irgen/test/Fixtures.test.scala` (`capabilityRejects` list), `tools/umpire/ir/framework_test.go` (kind name list)
**Touches:** [model/temporal/capabilities/**, model/irgen/Capabilities.scala, model/irgen/testdata/lifts/**, model/irgen/test/Fixtures.test.scala, tools/umpire/ir/framework_test.go, .flow/specs/fn-138-retries-and-deadline-capabilities.md]
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- **Ask the owner first, before writing the Properties** (record the answer in the spec's Decision Context and update R1 if it changes; never absorb a mismatch):
  1. Exhausted attempts. fn-128.3 makes the activity's retryable failure (and its retryable start-to-close timeout) fail once `states.retriesRemaining` is false; the spec says both models saturate and keep retrying. Proposed: Retries binds an optional retries-remaining predicate (default: always true, which the Nexus workflow keeps), and the Properties read "a retryable failure with retries remaining lands in `Waiting`; a non-retryable one, or one with none remaining, lands in `Failed`".
  2. The failing attempt's role. In the Nexus workflow the retryable `handler.reply(handlerError(true))` (and `network.fault`) fires from `scheduled`, a `Waiting` phase, not a `Held` one. Proposed: the Property's source is any `Live` phase, not `Held`; or Retries binds the failure step only and states no source role.
  3. Request phases. In the activity, a retryable failure from `pauseRequested` lands in `paused` (`Suspended`) and from `cancelRequested` in `canceled` (`Closed`), and fn-136 makes both phases `Held` (`System.scala:188-192`). So "lands in `Waiting`" fails whatever the source role. Proposed: the retryable Property promises only "a retryable failure with retries remaining does not land in `Failed`".
  4. Vacuous bound. "Attempts never exceed the bound" holds by type in both models (`UpTo[2]`; `Finite.upTo(attemptBound)`). Proposed: bound the count against fn-128.3's `maxAttempts` where it is finite, or drop the Property and amend R1.
- Capability kind: a case class of binding fields extending the capability marker, companion defining the Properties as defs taking the model and the fields (fn-134.2's shape; follow `Closable` after fn-137.6). Fields: attempt-count projection, attempt bound, the failure input (`ClassRef` or `Composed`, as `Pausable.pause`) with its retryable classification, plus whatever question 1 adds. No phase projection field: the phase comes from the machine's `Phased` given (fn-137.1); read roles through `using` `TypeTest` witnesses resolved at the declaration (fn-137.6's pattern), so the build shows no unchecked warning.
- Properties, as the owner's answers to questions 1–4 shape them. Name them in the companion's Scaladoc style; the old law `promises` text moves to Scaladoc, per fn-134. **Expressible forms only:** Check answers a transition Property with a `when` as `unsupported` (`model/SEMANTICS.md:407, :775`; `tools/umpire/check/checking_test.go:302`), and `Step` doesn't carry the action. So each Property is either a `when <failure>(…) holds` over the state after the step (for example "retryable with retries remaining: the phase is not `Failed`" and "non-retryable: the phase is `Failed`"), or a transition Property with no `when`. If an answer needs the state before the failure, say so to the owner, and record that it needs a framework change; never write an `unsupported` Property.
- Role ownership: every Property reads a Retries field, so fn-134's "reads no field of its own capability" refusal does not fire. Do not mark `Held` or `Waiting` as Retries-owned, and make sure reading `Held` (owned by Pollable, fn-137.7) does not make a Retries Property depend on Pollable being declared: the Nexus workflow declares no Pollable. If fn-137's ownership rule counts any read of an owned role, narrow it so that only Properties with no field of their own capability fall back to role ownership.
- Bound: the type's finite bound (`UpTo[2]`, `Finite.upTo(attemptBound)`) already guarantees count ≤ bound, so a Property over it can't fail (question 4). Under fn-128.3's `unlimited` the count still saturates at that type bound. A bound Property that isn't vacuous reads `maxAttempts`.
- Refusals in `model/irgen/Capabilities.scala`, reusing fn-137's shared "phase type has no case with role R" check: Retries on a non-`Phased` machine (names the machine); a phase with no `Waiting` case, no `Failed` case, or (if question 2 keeps it) no `Held` case, each naming the missing role.
- Kind name: add `Retries(` to `TestFrameworkNamesNoTemporal` so `model/umpire` never mentions it.

### Investigation targets
**Required:**
- `model/temporal/capabilities/Capabilities.scala:9-55`, `Close.scala`, `Pause.scala` (law-to-Property shape, as fn-134.3 reshapes it)
- `model/irgen/Capabilities.scala` (~200-420 role/field refusals, ~632-680 binding checks)
- `model/irgen/test/Fixtures.test.scala:308-329`, `model/irgen/testdata/lifts/CapabilityRejects.scala`, `lifts/expected/rejects.txt`
- `model/temporal/features/nexus/workflow/system/System.scala:157-182` (retryable split, `network.fault`)
**Optional:**
- `model/temporal/features/activity/standalone/system/System.scala:130-192` (poll raises the count; failure split)
- `tools/umpire/ir/framework_test.go:188-193`

### Key context
- Relies on: fn-134.2 (capability kind, companion Properties, `capabilities` section, bounds in `queries`, `origin`), fn-134.4 (Law/Catalog gone), fn-136.1 (role traits, role lowering, role conflicts), fn-137.1 (`Phased` given), fn-137.6/.7 (role witnesses, shared missing-role check, owned-role rule), fn-128.3 (`maxAttempts`, `retriesRemaining`, failing at exhaustion).
- The activity raises its attempt count on `worker.poll`, the Nexus workflow on the retryable failure; the bound Property must hold for both, so state it over every state rather than over the failure step.
- After fn-128.1 the activity's back-off is `dispatch = backoff` in `scheduled`; no Property may name `backingOff` or `Retrying`.
## Acceptance
- [ ] The owner's answers to questions 1–4 are recorded in the spec, and R1 matches them.
- [ ] A Retries fixture lifts: the generated Properties carrying `origin`, none of which Check would answer `unsupported`, bounded from `queries`, on a fixture machine without Pollable.
- [ ] Refusal fixtures: non-`Phased` machine, and each missing role, each message naming the machine and the role.
- [ ] `scala-cli test model/irgen` and `scala-cli test model/temporal` pass; the build shows no unchecked warning; `TestFrameworkNamesNoTemporal` lists `Retries(`.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
