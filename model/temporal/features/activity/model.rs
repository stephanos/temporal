//! Six-phase restriction of the archived mini activity Product.
//!
//! Cancellation is atomic and failure is fatal-only: the archive's cancelRequested and retry
//! branches are deliberately omitted. Successful Start establishes Scheduled outside this graph.

use umpire_dsl::machine;

/// One activity's finite lifecycle, independent of any SDK or server observation.
#[machine]
pub mod activity {
    use umpire_dsl::{Action, Outcome, Phased, Rejection, State, closable, reject};

    /// The phase of the activity.
    #[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
    pub enum Phase {
        /// Started successfully and waiting for delivery.
        #[default]
        Scheduled,
        /// Delivered to a worker.
        Started,
        /// Completed by the worker.
        Completed,
        /// Failed without retry.
        Failed,
        /// Canceled atomically by the client.
        Canceled,
        /// An applicable abstract deadline expired.
        TimedOut,
    }

    impl umpire_dsl::Phase for Phase {
        fn closed(&self) -> bool {
            matches!(self, Self::Completed | Self::Failed | Self::Canceled | Self::TimedOut)
        }
    }

    /// Actions controlled by the client.
    #[derive(Action, Copy)]
    pub enum Client {
        /// Request cancellation of the activity.
        Cancel,
    }

    /// Actions controlled by the activity worker.
    #[derive(Action, Copy)]
    pub enum Worker {
        /// Deliver the scheduled activity.
        Poll,
        /// Complete a started activity.
        RespondCompleted,
        /// Fail a started activity fatally.
        RespondFailed,
    }

    /// Internal abstract events with no clock or fairness assumption.
    #[derive(Action, Copy)]
    #[internal]
    pub enum Deadline {
        /// Expire while waiting for a worker.
        ScheduleToStart,
        /// Expire while held by a worker.
        StartToClose,
    }

    /// Model's dynamic state.
    #[derive(State)]
    pub struct Model {
        /// The activity's only dynamic field; Default supplies Scheduled at the start.
        pub phase: Phase,
    }

    impl Phased for Model {
        type Phase = Phase;

        fn phase(&self) -> Self::Phase {
            self.phase
        }
    }

    impl Model {
        #[on(Client)]
        fn client(&mut self, action: Client) {
            match action {
                Client::Cancel if self.closed() => reject!(Rejection::NotFound),
                Client::Cancel => {
                    self.phase = Phase::Canceled;
                }
            }
        }

        #[on(Worker)]
        fn worker(&mut self, action: Worker) {
            match action {
                Worker::Poll if self.phase == Phase::Scheduled => {
                    self.phase = Phase::Started;
                }
                Worker::RespondCompleted if self.phase == Phase::Started => {
                    self.phase = Phase::Completed;
                }
                Worker::RespondFailed if self.phase == Phase::Started => {
                    self.phase = Phase::Failed;
                }
                _ => {}
            }
        }

        #[on(Deadline)]
        fn deadline(&mut self, action: Deadline) {
            match action {
                Deadline::ScheduleToStart if self.phase == Phase::Scheduled => {
                    self.phase = Phase::TimedOut;
                }
                Deadline::StartToClose if self.phase == Phase::Started => {
                    self.phase = Phase::TimedOut;
                }
                _ => {}
            }
        }

        #[postcondition]
        fn terminal_remains(&self, before: &Self) {
            closable::terminal_remains(self, before);
        }

        #[postcondition]
        fn closed_rejects(&self, before: &Self, outcome: &Outcome) {
            closable::closed_rejects(self, before, outcome, Rejection::NotFound);
        }
    }
}

#[cfg(test)]
pub(crate) fn run() -> umpire_testing::RunResult {
    let result = umpire_testing::Checker::explore(&activity::world(), 64)?;
    let mut files = umpire_testing::render(&result)?;
    files.insert("rows.txt".into(), umpire_testing::render_rows(&result)?);
    Ok(files)
}

#[cfg(test)]
mod tests {
    use super::activity::{Client, Deadline, Model, Phase, Worker, world};
    use umpire_dsl::{Machine, Outcome, Phase as _, Rejection, Verdict};

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
}
