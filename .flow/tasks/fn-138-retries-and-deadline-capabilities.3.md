---
satisfies: [R1, R2, R3]
---
# fn-138-retries-and-deadline-capabilities.3 Activity and Nexus workflow systems declare Retries and Deadlines; timer windows as role tests; docs

## Description
Model adoption: the activity system and the Nexus workflow system declare Retries and one Deadline per timer, their remaining hand-written timer windows become role tests, and the docs and two-entity test learn the two kinds. Split from tasks 1 and 2 because it is the only part that touches Models (the batch keeps Model tasks free of regeneration and gates).

**Size:** M
**Files:** `model/temporal/features/activity/standalone/system/System.scala`, `model/temporal/features/nexus/workflow/system/System.scala`, `model/temporal/capabilities/Catalog.test.scala` (or its fn-134 successor), `model/README.md` (capabilities section and kind count), `.flow/specs/fn-138-retries-and-deadline-capabilities.md` (declared IR delta)
**Touches:** [model/temporal/features/activity/standalone/system/System.scala, model/temporal/features/nexus/workflow/system/System.scala, model/temporal/capabilities/Catalog.test.scala, model/README.md, .flow/specs/fn-138-retries-and-deadline-capabilities.md]
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Activity system: in its `capabilities` section add `retries` (attempt count `_.attempts`, bound `states.attemptBound`, failure input `worker.respond(AttemptResult.failed(_))` or fn-139's split `Failure` action, retries remaining `states.retriesRemaining` per fn-128.3) and three Deadlines: schedule-to-close covering `Live`, schedule-to-start covering `Waiting` (set predicate includes `dispatch = now`, fn-128.1), start-to-close covering `Held` (retryable, per task 2's decision). Bound them in `queries` with the machine's existing Limits.
- Nexus workflow system: it declares no capabilities today, so add its `capabilities` section (fn-134 placement and order lint) with `retries` (count `_.attempts`, bound `attemptBound`, failure `handler.reply(Reply.handlerError(_))`; `network.fault` is a back-off with no retryable input, so it stays outside Retries unless task 1's answer says otherwise) and three Deadlines: schedule-to-close covering `Live` (today `states.running`), schedule-to-start covering `Waiting`, start-to-close covering `Held`.
- Timer windows: fn-136.5 already moved `live`/`waiting`/`held` callers to `in[R]`. Rewrite what is left: the Nexus start-to-close rule's inline `s.phase == started` becomes `in[Held]` (or `when[Held]` if fn-139 renamed the form first; only if `started` is the Nexus phase's only `Held` case; otherwise stop and ask), and any timer rule still reading a named window.
- Where a Deadline Property duplicates an existing hand-written Property (`scheduleToStartFires`, `scheduleToCloseFires`, `startToCloseFires` in the activity; the Nexus `*TimedOut` claims), keep the hand-written one: R3 forbids changing any existing Query. Record the overlap in the task summary.
- Derived objects: check whether any derived machine or composition takes the activity system's or the Nexus workflow system's `capabilities` (fn-137's derivations carry `Phased`); if one does, the new Properties appear there too and belong in the declared IR delta.
- Two-entity test: add both kinds to the `declared` list and the expected Property-name list; the activity system and Nexus workflow system are the two instantiating machines with their own state types.
- IR delta for the batch diff (R3): list in the spec the new generated Properties and Queries per machine (`<machine>.<property>`) and the rewritten window's IR (case-set membership equal to the old comparison). Nothing else may change: existing Query answers, receipts, Definition IDs and Cases stay.
- Docs: add Retries and Deadline rows and a short example to `model/README.md`'s capabilities section and fix "the six capability kinds".

### Investigation targets
**Required:**
- `model/temporal/features/activity/standalone/system/System.scala:71-87, 186-231, 278-316`
- `model/temporal/features/nexus/workflow/system/System.scala:90-110, 157-247`
- `model/temporal/capabilities/Catalog.test.scala:17-40, 96-105`
- `model/README.md:376, 1017-1060`
**Optional:**
- `model/cases/*retry*`, `model/cases/*Timeout*` (Cases that must not change)

### Key context
- Relies on: tasks 1 and 2, fn-134.3 (Models on the `capabilities` section, bounds in `queries`), fn-136.2/.3 (roles on both phase enums), fn-136.5 (windows already `in[R]`), fn-137.3 (both systems `Phased`), fn-128.1 (dispatch field), fn-128.3 (`maxAttempts`, `retriesRemaining`), fn-139 (if landed, the worker failure action's name).
- If a generated Property fails on a Model at the batch regeneration, the model is wrong or the Property is: stop and resolve with the owner, never waive silently.
## Acceptance
- [ ] Both systems declare Retries and three Deadlines; every declaration compiles against the machine's types and witnesses.
- [ ] No hand-written timer window predicate or inline phase comparison remains in either system's timer rules.
- [ ] The two-entity test passes listing the new Properties; `scala-cli test model/temporal` passes.
- [ ] The spec lists the expected IR delta (new Properties and Queries, the rewritten window) for the batch diff.
- [ ] `model/README.md` documents both kinds.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
