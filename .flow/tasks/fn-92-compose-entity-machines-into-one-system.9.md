---
satisfies: [R3]
---
# fn-92-compose-entity-machines-into-one-system.9 Resolve member-qualified references to synchronized actions

## Description
Fix the completion-review finding (Codex, P1, R3): member-qualified references to a synchronized action fail. The spec documents `operation.handlerReply (async)`-style references as valid in Scenario `actions:` and Property `when:`. Both resolvers build the key mechanically as `<member>_<action>-<class>` and never consult the composition's synchronization mapping; synchronization replaces the per-member key with the sync name, so the mechanical key no longer exists. Reproduced on the existing `Forward.pipeline` fixture: `actions: [job.reply (ok)]` and `when: job.reply (ok)` both fail with `unknown Model action 'job_reply-ok'` although `reply-ok` (the synchronized name) exists.

**Size:** S
**Files:** `model/Umpire/Command/Syntax.lean` (the Scenario `actions:` resolver near line 719 and the Property `when:` resolver near line 543), `model/Umpire/Command/Tests/Compose.lean`
**Touches:** [model/Umpire/Command/Syntax.lean, model/Umpire/Command/Tests/Compose.lean, model/Umpire/Command/Compose.lean]

### Approach
- In both resolvers, resolve an unambiguous member-qualified reference through `CompositionEntry.syncs` (the member action it names maps to the sync name) before falling back to the mechanical key; keep one shared helper for both sites.
- Ambiguity: a member action that appears in more than one sync group, or an unknown member/action, is a located error naming the reference and the candidates.
- Positive regression pins for the documented dotted spelling against `Forward.pipeline` (both `actions:` and `when:`), plus a negative pin for an unknown member action; existing composition tests, goldens and the differential blocks stay byte-identical.
- Gates: `cd model && mise exec -- lake build Umpire.Command.Tests.Compose UmpireTests TemporalModelTests`, `make umpire-check-goldens`, `make lint-model-builtin LINT_MODEL_MODULES="Umpire.Command.Syntax Umpire.Command.Tests.Compose"`.

## Acceptance
- [ ] A member-qualified reference to an action in a `sync:` group resolves to the synchronized action in Scenario `actions:` and Property `when:`, pinned against `Forward.pipeline`.
- [ ] Ambiguous or unknown member-qualified references are located errors; every existing test, golden and differential block is byte-identical.


## Done summary
Fixed the fn-92 completion review finding (Codex P1, R3): the Property `when:` and Scenario
`actions:` resolvers mechanically joined a dotted member-qualified reference as
`<field>_<action>[-<class>]`, ignoring a composition's `sync:` groups, so a documented reference
like `job.reply (ok)` failed with `unknown Model action 'job_reply-ok'` although `reply-ok` (the
synchronized name) existed. Added `Compose.resolveReference` (resolves a `field.action` pair
against the composition's recorded `sync:` groups before falling back to the mechanical key;
`ambiguous` when the pair is more than one group's participant) and a shared
`resolveComposedActionKey` wrapper in Syntax.lean, used at both resolver sites in place of the old
`composedActionKeyOf`. Pinned the fix with the documented dotted spelling against `Forward.pipeline`
(both `when:` and `actions:`), a negative pin for an unknown member action, and direct `#guard`
pins of `resolveReference`'s three branches including the ambiguous case (which needs no compiled
composition to exhibit). No new `query` was declared, so the R14 differential's expected block
stays byte-identical, per the task's Approach.

stage: impl-review - ran [codex fan-out b27e559f (gpt-6-astra high, 3-axis fan-out): correctness, contracts, integration all SHIP, 0 findings]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d5f1a1b69cb18df9de13681b3bd99a64562787d4
- Tests: cd model && mise exec -- lake build Umpire.Command.Tests.Compose UmpireTests TemporalModelTests (783/783, rc=0), make umpire-check-goldens (rc=0), LEAN_NUM_THREADS=1 make lint-model-builtin LINT_MODEL_MODULES="Umpire.Command.Syntax Umpire.Command.Tests.Compose" (rc=0, no linter findings)
- PRs: