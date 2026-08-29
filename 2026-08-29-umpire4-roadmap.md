# Handoff: Umpire 4 Prototype Roadmap

## Objective

Work through `.plans/UMPIRE4_ORDER.md` using `$flow-next-work`, with one dedicated conductor subagent per spec. Serialize all task writers in the shared checkout, automatically follow recommended fix/intervention paths, and close completed specs when they block the next ordered spec.

## Current state

- Repository: `/Users/stephan/Workspace/skunkworks/umpire/temporal`
- Branch: `umpire`
- HEAD: `30027010d` (`chore(flow): record fn20.1 plan-sync outcome`)
- Active spec: `.flow/specs/fn-20-local-execution-semantic-conformance.md`
- Flow state: fn20 is open/ready; task `.1` is done; task `.2` is `in_progress`; tasks `.3` through `.7` are todo.
- No worker, test, build, or review process is active. The fn20 conductor and task `.2` worker were explicitly interrupted for the computer restart.
- No fn20 implementation edits are staged or unstaged. The only dirty paths are the inherited protected false-symlink stats:
  - `config/development.yaml`
  - `schema/elasticsearch/visibility/index_template_v7.json`
  Never edit, normalize, stage, or discard them.

## Completed roadmap work

- fn18 Versioned Umpire Artifact boundary: complete, completion-review SHIP, final correctness/standards audits clean, closed.
- fn19 Bounded local Temporal execution: complete, completion-review SHIP, repeated final correctness/standards audits clean, closed.
- Earlier prerequisites fn38, fn41, fn39, fn42 and their blockers were completed; fn39 was closed when it blocked fn19.
- Fn20 task `.1` is complete with SHIP:
  - implementation `12a1acd2d`
  - review fix `e9a28ac0b`
  - receipt `71a8d4d35`
  - plan-sync `30027010d`
  - focused/aggregate Observation builds, full regression (234-job UmpireTests), and model lint passed.

## Fn20 task .2 resume point

Task: `fn-20-local-execution-semantic-conformance.2`.

The task was claimed and re-anchored, but no task `.2` source edit or concrete RED was created before interruption. Its baseline regression was still the workflow gate.

The agreed architecture is:

- Protocol owns strict bounded canonical v2 bytes and closure.
- RunEvaluation alone composes source evidence → checked Observation → System trace → checked Implementation Link → Feature trace → the task `.1` Property evaluator.
- Preserve separate operational, Observation Evaluation, Implementation Link, and Property stages.
- Keep the plan-sensitive evaluation outcome checksum Lean-owned.
- Do not reimplement semantics in Go or bypass checked declarations.
- Pretty JSON with two-space indentation plus exactly one terminal LF remains the only canonical Artifact representation.

Task `.1` introduced the generic `checkRunEvaluation` seam. Review fixed logical-time preflight and exact Query-target-to-Link-destination ID/fingerprint binding. Reuse that seam rather than duplicating it.

## Guardrails

- Read `/home/agent/.codex/skills/flow-next-work/SKILL.md` and required references before resuming.
- Reuse the dedicated `/root/order_fn20` conductor with `followup_task`; do not create a second fn20 conductor.
- Preserve all existing comments.
- Never use Git worktrees.
- Use canonical lowercase paths only; uppercase `Tools/...`/`Temporal/...` listings may be same-inode aliases and must never enter commits.
- Go tests use `require`, and prefer `Equal`, `EqualValues`, or `ProtoEqual`.
- No push or PR.
- `/tmp` has repeatedly filled from unrelated caches. Route task temp/cache host-backed when practical. Delete only an exact, validated, inactive disposable cache; never interrupt an active unrelated process.
- Leave completed specs open until the next ordered spec is blocked by them, then close under the user's standing approval and commit only the Flow state.

## Remaining roadmap order

Finish fn20, then fn21, fn27, fn28, run the prototype verification gate, then fn5, fn17, fn40, fn33, and fn22. Use one dedicated `$flow-next-work` conductor per spec and perform each spec's completion review, full gates, paired audits, validation, and tracker check.

## Next action

Resume `/root/order_fn20` with `followup_task`, tell it to reconstruct task `.2` from committed HEAD `30027010d`, rerun the required baseline, establish the smallest Protocol + RunEvaluation TDD RED, and continue fn20 serially through completion review and paired audits.
