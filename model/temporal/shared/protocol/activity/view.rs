//! Public observations exclude latent timers and cannot construct protocol state.

use super::protocol::{Failure, Lifecycle, Model, Terminal};

/// Whether an actual observation supplies a field, including a known absent value.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
pub enum Observed<T> {
    /// The response does not establish this field.
    #[default]
    Unknown,
    /// The response establishes this value.
    Known(T),
}

impl<T: PartialEq> Observed<T> {
    fn matches(&self, value: &T) -> bool {
        match self {
            Self::Unknown => true,
            Self::Known(observed) => observed == value,
        }
    }
}

/// Public execution status in the selected operation domain.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Status {
    /// Execution has no terminal result.
    Running,
    /// Worker completion won.
    Completed,
    /// Failure ended execution.
    Failed,
    /// Cancellation was acknowledged or closed a queued execution.
    Canceled,
    /// A deadline ended execution.
    TimedOut,
}

/// Public breakdown of an execution that has not closed.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum RunState {
    /// Waiting for worker assignment.
    Scheduled,
    /// Worker owns the current attempt.
    Started,
    /// Running attempt has an unacknowledged cancellation request.
    CancelRequested,
}

/// Execution outcome, independently of the last failed attempt.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum TerminalOutcome {
    /// No terminal outcome exists yet.
    None,
    /// Successful completion.
    Completed,
    /// Failure without another retry.
    Failed,
    /// Cancellation closure.
    Canceled,
    /// Timeout without another retry.
    TimedOut,
}

/// Abstract failure classification when the response supplies sufficient detail.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum RetainedFailure {
    /// No failed attempt was recorded.
    None,
    /// Worker failure whose type allows retries.
    RetryableWorker,
    /// Worker failure whose type prohibits retries.
    PermanentWorker,
    /// Queue waiting expired.
    ScheduleToStart,
    /// Attempt execution expired.
    StartToClose,
    /// Spanning execution budget expired.
    ScheduleToClose,
    /// Heartbeat interval expired.
    Heartbeat,
}

/// Complete public projection of a full protocol candidate.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct View {
    /// Execution status.
    pub status: Status,
    /// Running substate, absent after closure.
    pub run_state: Option<RunState>,
    /// Current scheduled attempt number.
    pub attempt: u8,
    /// Current execution result, independently of retry history.
    pub terminal_outcome: TerminalOutcome,
    /// Most recent failed attempt classification.
    pub retained_failure: RetainedFailure,
    /// Whether the running attempt has an unacknowledged cancellation request.
    pub cancellation_pending: bool,
}

impl From<&Model> for View {
    fn from(state: &Model) -> Self {
        let terminal_outcome = match state.terminal_outcome {
            Terminal::None => TerminalOutcome::None,
            Terminal::Completed => TerminalOutcome::Completed,
            Terminal::Failed => TerminalOutcome::Failed,
            Terminal::Canceled => TerminalOutcome::Canceled,
            Terminal::TimedOut => TerminalOutcome::TimedOut,
        };
        Self {
            status: match state.terminal_outcome {
                Terminal::None => Status::Running,
                Terminal::Completed => Status::Completed,
                Terminal::Failed => Status::Failed,
                Terminal::Canceled => Status::Canceled,
                Terminal::TimedOut => Status::TimedOut,
            },
            run_state: match state.lifecycle {
                Lifecycle::Queued => Some(RunState::Scheduled),
                Lifecycle::Executing if state.cancellation_pending => {
                    Some(RunState::CancelRequested)
                }
                Lifecycle::Executing => Some(RunState::Started),
                Lifecycle::Closed => None,
            },
            attempt: state.attempt,
            terminal_outcome,
            retained_failure: match state.retained_failure {
                Failure::None => RetainedFailure::None,
                Failure::RetryableWorker => RetainedFailure::RetryableWorker,
                Failure::PermanentWorker => RetainedFailure::PermanentWorker,
                Failure::ScheduleToStart => RetainedFailure::ScheduleToStart,
                Failure::StartToClose => RetainedFailure::StartToClose,
                Failure::ScheduleToClose => RetainedFailure::ScheduleToClose,
                Failure::Heartbeat => RetainedFailure::Heartbeat,
            },
            cancellation_pending: state.cancellation_pending,
        }
    }
}

/// Validated actual fields; omitted fields place no constraint on a candidate view.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
pub struct Snapshot {
    /// Observed execution status.
    pub status: Observed<Status>,
    /// Observed running substate, including a known absent substate.
    pub run_state: Observed<Option<RunState>>,
    /// Observed attempt number.
    pub attempt: Observed<u8>,
    /// Observed execution result when the response supplies one.
    pub terminal_outcome: Observed<TerminalOutcome>,
    /// Observed and independently classified last-attempt failure.
    pub retained_failure: Observed<RetainedFailure>,
    /// Observed pending cancellation state.
    pub cancellation_pending: Observed<bool>,
}

impl Snapshot {
    /// Compare public fields without selecting or manufacturing latent state.
    pub fn matches(&self, candidate: &View) -> bool {
        self.status.matches(&candidate.status)
            && self.run_state.matches(&candidate.run_state)
            && self.attempt.matches(&candidate.attempt)
            && self.terminal_outcome.matches(&candidate.terminal_outcome)
            && self.retained_failure.matches(&candidate.retained_failure)
            && self.cancellation_pending.matches(&candidate.cancellation_pending)
    }
}

/// Notification returned by a successful worker heartbeat.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct HeartbeatReply {
    /// Whether cancellation was requested when the heartbeat was processed.
    pub cancellation_pending: bool,
}

/// Current-attempt worker acknowledgement, supplied by correlated operation evidence.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct Acknowledgement {
    /// Attempt identified by the correlated worker acknowledgement.
    pub attempt: u8,
}
