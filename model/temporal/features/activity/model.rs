//! Six-phase restriction of the archived mini activity Product.
//!
//! Cancellation is atomic and failure is fatal-only: the archive's cancelRequested and retry
//! branches are deliberately omitted. Successful Start establishes Scheduled outside this graph.

use umpire_dsl::machine;

/// One activity's finite lifecycle, independent of any SDK or server observation.
#[machine]
pub mod activity {
    use umpire_dsl::{Action, Spell, State, check};

    /// The phase of the activity.
    #[derive(Clone, Copy, Default, PartialEq, Eq, Hash)]
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

    impl Spell for Phase {
        fn spell(&self, out: &mut String) {
            out.push_str(match self {
                Self::Scheduled => "Phase::Scheduled",
                Self::Started => "Phase::Started",
                Self::Completed => "Phase::Completed",
                Self::Failed => "Phase::Failed",
                Self::Canceled => "Phase::Canceled",
                Self::TimedOut => "Phase::TimedOut",
            });
        }
    }

    /// Actions controlled by the client.
    #[derive(Action, Copy)]
    pub enum Client {
        /// Cancel a scheduled or started activity.
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

    impl Model {
        #[on(Client)]
        fn client(&mut self, action: Client) {
            match action {
                Client::Cancel if matches!(self.phase, Phase::Scheduled | Phase::Started) => {
                    self.phase = Phase::Canceled;
                }
                Client::Cancel => {}
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
            if matches!(
                before.phase,
                Phase::Completed | Phase::Failed | Phase::Canceled | Phase::TimedOut
            ) {
                check!(self.phase == before.phase);
            }
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
    use umpire_dsl::{Machine, Spell, Verdict};

    const TERMINALS: [Phase; 4] =
        [Phase::Completed, Phase::Failed, Phase::Canceled, Phase::TimedOut];

    fn spelling(value: &impl Spell) -> String {
        let mut text = String::new();
        value.spell(&mut text);
        text
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
        assert!(actions == expected_actions);
        let starts = world.starts();
        assert!(starts == [Model { phase: Scheduled }]);
        for (action, internal) in actions.iter().zip([false, false, false, false, true, true]) {
            assert_eq!(world.internal(action), internal);
        }
        for (phase, expected) in [
            (Scheduled, [Some(Canceled), Some(Started), None, None, Some(TimedOut), None]),
            (Started, [Some(Canceled), None, Some(Completed), Some(Failed), None, Some(TimedOut)]),
            (Completed, [None; 6]),
            (Failed, [None; 6]),
            (Canceled, [None; 6]),
            (TimedOut, [None; 6]),
        ] {
            for (action, next) in actions.iter().zip(expected) {
                let steps = world.step(&Model { phase }, action.clone());
                let expected: Vec<_> =
                    next.into_iter().map(|phase| ((), Vec::new(), Model { phase })).collect();
                assert!(
                    steps.as_slice() == expected,
                    "{} after {}",
                    spelling(action),
                    spelling(&phase),
                );
            }
        }
    }

    #[test]
    fn registered_terminal_preservation_checks_every_terminal_action_pair() {
        let world = world();
        let claims = world.postconditions();
        assert_eq!(claims.len(), 1);
        let claim = &claims[0];
        assert_eq!(claim.name, "terminal_remains");
        assert!(claim.action.is_none());
        let mut checked = 0;
        for phase in TERMINALS {
            let before = Model { phase };
            for action in world.actions() {
                assert!(world.step(&before, action.clone()).is_empty());
                // Disabled actions emit no edge; invoke the registered callback on preservation
                // directly so the terminal antecedent is exercised beyond the emitted-step report.
                let preserved = ((), Vec::new(), before.clone());
                assert_eq!(claim.holds(&world, &before, &action, &preserved), Verdict::Holds);
                checked += 1;
            }
        }
        assert_eq!(checked, 24);
    }

    #[test]
    fn hypothetical_reopening_refutes_the_same_registered_preservation_callback() {
        let world = world();
        let claims = world.postconditions();
        let claim = &claims[0];
        for phase in TERMINALS {
            let before = Model { phase };
            for action in world.actions() {
                let reopened = ((), Vec::new(), Model { phase: Phase::Scheduled });
                assert_eq!(claim.holds(&world, &before, &action, &reopened), Verdict::Fails);
            }
        }
    }
}
