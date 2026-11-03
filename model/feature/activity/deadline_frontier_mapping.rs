//! Native activity pin for the independent runtime frontier counterexample.
//! This maps two compatible histories; it does not establish runtime or SDK refinement.

use super::configured;
use crate::activity::activity::{
    Action, Client, Deadline, ExpiredDeadline, Model, Phase, Remaining, Worker, World,
};
use umpire_dsl::{Machine, Outcome};

fn step(world: &World, before: &Model, action: Action, facts: &[ExpiredDeadline]) -> Model {
    let steps = world.checked_step(before, action).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(steps.len(), 1);
    assert_eq!(steps[0].0, Outcome::Accepted);
    assert_eq!(steps[0].1, facts);
    steps[0].2.clone()
}

fn assert_pending(state: &Model, global: u8, running: u8, heartbeat: u8) {
    assert_eq!(state.phase, Phase::Started);
    assert_eq!(state.attempt, 1);
    assert!(state.cancel_requested);
    assert_eq!(state.schedule_to_start_remaining, None);
    assert_eq!(state.start_to_close_remaining, Some(Remaining { ticks: running }));
    assert_eq!(state.schedule_to_close_remaining, Some(Remaining { ticks: global }));
    assert_eq!(state.heartbeat_remaining, Some(Remaining { ticks: heartbeat }));
}

#[test]
fn compatible_heartbeat_histories_require_distinct_global_and_local_expiry_facts() {
    let world = configured(3, true, [None, Some(3), Some(3), Some(2)]);
    let initial = world.starts().remove(0);
    assert_eq!(
        initial,
        Model {
            phase: Phase::Scheduled,
            attempt: 1,
            cancel_requested: false,
            schedule_to_start_remaining: None,
            start_to_close_remaining: None,
            schedule_to_close_remaining: Some(Remaining { ticks: 3 }),
            heartbeat_remaining: None,
        }
    );
    let started = step(&world, &initial, Worker::Poll(1).into(), &[]);
    let mut expected_started = initial.clone();
    expected_started.phase = Phase::Started;
    expected_started.start_to_close_remaining = Some(Remaining { ticks: 3 });
    expected_started.heartbeat_remaining = Some(Remaining { ticks: 2 });
    assert_eq!(started, expected_started);
    let pending = step(&world, &started, Client::Cancel.into(), &[]);
    assert_pending(&pending, 3, 3, 2);

    // Runtime Renew maps to Heartbeat. Its observation hides these countdowns.
    let renewed = step(&world, &pending, Worker::Heartbeat.into(), &[]);
    assert_pending(&renewed, 3, 3, 2);
    let ticked = step(&world, &pending, Deadline::Tick.into(), &[]);
    assert_pending(&ticked, 2, 2, 1);
    let ticked_renewed = step(&world, &ticked, Worker::Heartbeat.into(), &[]);
    assert_pending(&ticked_renewed, 2, 2, 2);
    assert_eq!(
        (renewed.phase, renewed.attempt, renewed.cancel_requested),
        (ticked_renewed.phase, ticked_renewed.attempt, ticked_renewed.cancel_requested)
    );
    assert_ne!(renewed, ticked_renewed);

    let local_tick = step(&world, &renewed, Deadline::Tick.into(), &[]);
    assert_pending(&local_tick, 2, 2, 1);
    let global_tick = step(&world, &ticked_renewed, Deadline::Tick.into(), &[]);
    assert_pending(&global_tick, 1, 1, 1);

    // Generic PendingClosed/FactLocal and GlobalClosed/FactGlobal share Temporal TimedOut.
    let local_closed =
        step(&world, &local_tick, Deadline::Tick.into(), &[ExpiredDeadline::Heartbeat]);
    let global_closed =
        step(&world, &global_tick, Deadline::Tick.into(), &[ExpiredDeadline::ScheduleToClose]);
    let terminal = Model {
        phase: Phase::TimedOut,
        attempt: 1,
        cancel_requested: false,
        schedule_to_start_remaining: None,
        start_to_close_remaining: None,
        schedule_to_close_remaining: None,
        heartbeat_remaining: None,
    };
    assert_eq!(local_closed, terminal);
    assert_eq!(global_closed, terminal);
}
