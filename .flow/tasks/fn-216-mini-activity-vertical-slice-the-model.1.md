---
satisfies: [R1]
---
# fn-216-mini-activity-vertical-slice-the-model.1 Mini activity Model as a shared fixture

## Description
`model/temporal/features/activity_mini/model.rs` in the attribute surface (after fn-210.19/.20), in a new `model/temporal` tests crate using `umpire-testing`'s runner (`model/temporal/features/` is its fixtures root); never under `model/umpire` (owner): `Phase` enum, actors `Client`, `Worker`, `Deadline` (internal), transitions per the archived `Dispatching`/`Completion` facets (start: Scheduled stays, poll: Scheduled→Started, respondCompleted: Started→Completed, respondFailed: Started→Failed, cancel: Scheduled|Started→Canceled, deadlines: →TimedOut), postcondition `terminal_remains`; goldens. Record in the task's done summary every DSL gap met (`end`, facts, inputs) for R4. Also `temporal.rs` beside it implementing the generic `Realize` trait over the three envelopes: `observe` maps `DescribeActivityExecutionResponse.status` to actions, `answer` returns a partial `RespondActivityTask*Request` (the trait itself, generic over message types, is added to `umpire-dsl` or a small `umpire-realize` crate in this task, with no Temporal names). **Touches:** `model/temporal/**` (new), `model/Cargo.toml` (member), `model/umpire/dsl/**` or `model/umpire/realize/**` (the generic trait only). **Feedback loop (R30):** paste `make model-time`.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
