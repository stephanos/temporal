use crate::activity::activity::{Client, Deadline, Model, Phase, Worker, world};
use umpire_dsl::{Machine, Outcome, Phase as _, Rejection, Verdict};

pub(crate) fn run() -> umpire_testing::RunResult {
    let result = umpire_testing::Checker::explore(&world(), 64)?;
    let mut files = umpire_testing::render(&result)?;
    files.insert("rows.txt".into(), umpire_testing::render_rows(&result)?);
    let covered = closings()?;
    files.insert("expansion/closings.txt".into(), covered.expansion().into());
    files.insert("coverage.txt".into(), covered.coverage().to_string());
    Ok(files)
}

fn closings() -> Result<umpire_suite::Covered<Model>, umpire_suite::Refusal> {
    use umpire_suite::*;

    let start = Model { phase: Phase::Scheduled };
    let covered = cover(&Space::from([start.clone()]), Cases(2 * 4), path![start, [Worker::Poll; 0..=1], .., (Client::Cancel | Worker::RespondCompleted | Worker::RespondFailed | Deadline::StartToClose)])?;
    assert_eq!(covered.len(), 8);
    for (index, run) in covered.runs().iter().enumerate() {
        assert_eq!(run.state().phase, [Phase::Canceled, Phase::Completed, Phase::Failed, Phase::TimedOut][index % 4]);
        assert_eq!(run.path().steps.len(), if index == 0 { 1 } else { 2 });
    }
    Ok(covered)
}

const PHASES: [Phase; 6] = [
    Phase::Scheduled,
    Phase::Started,
    Phase::Completed,
    Phase::Failed,
    Phase::Canceled,
    Phase::TimedOut,
];

fn terminals() -> impl Iterator<Item = Phase> {
    PHASES.into_iter().filter(umpire_dsl::Phase::closed)
}

#[test]
fn closed_cancel_emits_a_rejected_self_loop() {
    let world = world();
    for phase in terminals() {
        let before = Model { phase };
        let steps = world.step(&before, Client::Cancel.into());
        assert_eq!(steps.len(), 1, "cancel after {phase:?}");
        assert_eq!(steps[0].0, Outcome::Rejected(Rejection::NotFound));
        assert_eq!(steps[0].2, before);
        assert_eq!(steps[0].1, Vec::<()>::new());
    }
}

#[test]
fn terminal_classification_covers_every_phase() {
    for (phase, terminal) in [
        (Phase::Scheduled, false),
        (Phase::Started, false),
        (Phase::Completed, true),
        (Phase::Failed, true),
        (Phase::Canceled, true),
        (Phase::TimedOut, true),
    ] {
        assert_eq!(phase.closed(), terminal, "{phase:?}");
    }
}

#[test]
fn six_phase_transition_matrix_includes_disabled_actions() {
    use Phase::{Canceled, Completed, Failed, Scheduled, Started, TimedOut};

    let world = world();
    let actions = world.actions();
    let expected_actions = [
        Client::Cancel.into(),
        Worker::Poll.into(),
        Worker::RespondCompleted.into(),
        Worker::RespondFailed.into(),
        Deadline::ScheduleToStart.into(),
        Deadline::StartToClose.into(),
    ];
    assert_eq!(actions, expected_actions);
    let starts = world.starts();
    assert_eq!(starts, [Model { phase: Scheduled }]);
    for (action, internal) in actions.iter().zip([false, false, false, false, true, true]) {
        assert_eq!(world.internal(action), internal);
    }
    for (phase, expected) in [
        (Scheduled, [Some(Canceled), Some(Started), None, None, Some(TimedOut), None]),
        (Started, [Some(Canceled), None, Some(Completed), Some(Failed), None, Some(TimedOut)]),
        (Completed, [Some(Completed), None, None, None, None, None]),
        (Failed, [Some(Failed), None, None, None, None, None]),
        (Canceled, [Some(Canceled), None, None, None, None, None]),
        (TimedOut, [Some(TimedOut), None, None, None, None, None]),
    ] {
        for (action, next) in actions.iter().zip(expected) {
            let steps = world.step(&Model { phase }, action.clone());
            let outcome = if phase.closed() {
                Outcome::Rejected(Rejection::NotFound)
            } else {
                Outcome::Accepted
            };
            let expected: Vec<_> =
                next.into_iter().map(|phase| (outcome, Vec::new(), Model { phase })).collect();
            assert!(steps.as_slice() == expected, "{action} after {phase:?}");
        }
    }
}

#[test]
fn registered_closure_laws_check_every_phase_action_pair() {
    let world = world();
    let claims = world.postconditions();
    assert_eq!(
        claims.iter().map(|claim| claim.name).collect::<Vec<_>>(),
        ["terminal_remains", "closed_rejects",]
    );
    assert!(claims.iter().all(|claim| claim.action.is_none()));
    let mut checked = 0;
    for phase in PHASES {
        let before = Model { phase };
        for action in world.actions() {
            let steps = world.step(&before, action.clone());
            let preserved =
                (Outcome::Rejected(Rejection::NotFound), Vec::new(), before.clone());
            let emitted = steps.first().unwrap_or(&preserved);
            for claim in &claims {
                assert_eq!(claim.holds(&world, &before, &action, emitted), Verdict::Holds);
                checked += 1;
            }
        }
    }
    assert_eq!(checked, 72);
}

#[test]
fn hypothetical_reopening_and_terminal_moves_refute_registered_closure_laws() {
    let world = world();
    let claims = world.postconditions();
    for phase in terminals() {
        let before = Model { phase };
        for action in world.actions() {
            for destination in PHASES.into_iter().filter(|destination| *destination != phase) {
                let changed = (
                    Outcome::Rejected(Rejection::NotFound),
                    Vec::new(),
                    Model { phase: destination },
                );
                for claim in &claims {
                    assert_eq!(claim.holds(&world, &before, &action, &changed), Verdict::Fails);
                }
            }
        }
    }
}

#[test]
fn hypothetical_accepted_and_wrong_rejection_refute_registered_closed_rejection() {
    let world = world();
    let claims = world.postconditions();
    let claim = &claims[1];
    assert_eq!(claim.name, "closed_rejects");
    for phase in terminals() {
        let before = Model { phase };
        for action in world.actions() {
            for outcome in [Outcome::Accepted, Outcome::Rejected(Rejection::FailedPrecondition)]
            {
                let emitted = (outcome, Vec::new(), before.clone());
                assert_eq!(claim.holds(&world, &before, &action, &emitted), Verdict::Fails);
            }
        }
    }
}
