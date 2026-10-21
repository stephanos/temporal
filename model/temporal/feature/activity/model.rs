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
