use std::collections::BTreeSet;

use crate::activity::activity::{
    Action, Client, Deadline, ExpiredDeadline, Model, Phase, Remaining, Ticks, Worker, World,
};
use umpire_dsl::{Machine, Outcome, Rejection};

#[derive(Debug, PartialEq, Eq)]
struct Graph {
    starts: BTreeSet<String>,
    states: BTreeSet<String>,
    edges: BTreeSet<String>,
}

fn state_key(phase: Phase) -> &'static str {
    match phase {
        Phase::Scheduled => "Scheduled",
        Phase::Started => "Started",
        Phase::Completed => "Completed",
        Phase::Failed => "Failed",
        Phase::Canceled => "Canceled",
        Phase::TimedOut => "TimedOut",
    }
}

fn outcome_key(outcome: Outcome) -> &'static str {
    match outcome {
        Outcome::Accepted => "Accepted",
        Outcome::Rejected(Rejection::NotFound) => "Rejected(NotFound)",
        other @ Outcome::Rejected(_) => panic!("unexpected checkpoint outcome: {other:?}"),
    }
}

fn phase(state: &str) -> String {
    state
        .strip_prefix("Model { phase: ")
        .and_then(|s| s.strip_suffix(" }"))
        .unwrap_or_else(|| panic!("invalid original state: {state}"))
        .into()
}

fn reference(rows: &str) -> Graph {
    let states: Vec<_> = include_str!("original/states.snap.txt")
        .lines()
        .map(|line| {
            let (_, state) =
                line.split_once(": ").unwrap_or_else(|| panic!("invalid original state row"));
            phase(state)
        })
        .collect();
    let edges = rows
        .lines()
        .map(|line| {
            let (source, row) =
                line.split_once(": ").unwrap_or_else(|| panic!("invalid original row"));
            let (action, row) = row
                .split_once(" choice=0 outcome=")
                .unwrap_or_else(|| panic!("invalid original choice"));
            let (result, destination) =
                row.split_once(" -> ").unwrap_or_else(|| panic!("invalid original destination"));
            let source: usize =
                source.parse().unwrap_or_else(|_| panic!("invalid original source"));
            format!("{}: {action} {result} -> {}", states[source], phase(destination))
        })
        .collect();
    Graph {
        starts: BTreeSet::from([states[0].clone()]),
        states: states.into_iter().collect(),
        edges,
    }
}

fn projected_state(state: &Model) -> &'static str {
    assert_eq!(state.attempt, 1);
    assert!(!state.cancel_requested);
    assert_eq!(
        state.schedule_to_start_remaining,
        (state.phase == Phase::Scheduled).then_some(Remaining { ticks: 1 })
    );
    assert_eq!(
        state.start_to_close_remaining,
        (state.phase == Phase::Started).then_some(Remaining { ticks: 1 })
    );
    assert_eq!(state.schedule_to_close_remaining, None);
    assert_eq!(state.heartbeat_remaining, None);
    state_key(state.phase)
}

fn projected() -> Result<Graph, Box<dyn std::error::Error>> {
    let world = World::new(1, Some(Ticks(1)), Some(Ticks(1)), None, None, false);
    let result = umpire_testing::Checker::explore(&world, 64)?;
    let mut graph = Graph {
        starts: world.starts().iter().map(|state| projected_state(state).to_owned()).collect(),
        states: BTreeSet::new(),
        edges: BTreeSet::new(),
    };
    for (id, state) in result.states() {
        let source = projected_state(state);
        graph.states.insert(source.to_owned());
        for row in result.rows(id)? {
            assert_eq!(row.choice, 0);
            let (outcome, facts, destination) = &row.emitted;
            let destination_key = projected_state(destination);
            let action = match &row.action {
                Action::Deadline(Deadline::Tick) => {
                    assert_eq!(*outcome, Outcome::Accepted);
                    assert_eq!(destination.phase, Phase::TimedOut);
                    match (state.phase, facts.as_slice()) {
                        (Phase::Scheduled, [ExpiredDeadline::ScheduleToStart]) => {
                            "Deadline::ScheduleToStart"
                        }
                        (Phase::Started, [ExpiredDeadline::StartToClose]) => {
                            "Deadline::StartToClose"
                        }
                        other => panic!("unexpected compatibility Tick: {other:?}"),
                    }
                }
                Action::Worker(Worker::Heartbeat) => {
                    assert_eq!(state.phase, Phase::Started);
                    assert_eq!(*outcome, Outcome::Accepted);
                    assert_eq!(facts.as_slice(), []);
                    assert_eq!(destination, state);
                    continue;
                }
                action => {
                    assert_eq!(facts.as_slice(), []);
                    match action {
                        Action::Client(Client::Cancel) => "Client::Cancel",
                        Action::Worker(Worker::Poll(1)) => "Worker::Poll",
                        Action::Worker(Worker::RespondCompleted) => "Worker::RespondCompleted",
                        Action::Worker(Worker::RespondFailed { .. }) => "Worker::RespondFailed",
                        other => panic!("unexpected compatibility edge: {other:?}"),
                    }
                }
            };
            graph.edges.insert(format!(
                "{source}: {action} {} facts=[] -> {destination_key}",
                outcome_key(*outcome)
            ));
        }
    }
    Ok(graph)
}

#[test]
fn retained_original_exact_lifecycle_projection() -> umpire_testing::TestResult {
    let old = reference(include_str!("original/rows.snap.txt"));
    assert_eq!((old.starts.len(), old.states.len(), old.edges.len()), (1, 6, 11));
    assert_eq!(projected()?, old);
    Ok(())
}

#[test]
fn changed_original_lifecycle_edge_fails_the_same_comparison() -> umpire_testing::TestResult {
    let original = include_str!("original/rows.snap.txt");
    let changed = original.replacen(
        "Deadline::ScheduleToStart choice=0 outcome=Accepted facts=[] -> Model { phase: TimedOut }",
        "Deadline::ScheduleToStart choice=0 outcome=Accepted facts=[] -> Model { phase: Failed }",
        1,
    );
    assert_ne!(changed, original);
    assert_ne!(projected()?, reference(&changed));
    Ok(())
}
