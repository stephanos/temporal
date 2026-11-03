//! Standalone activity protocol, authored from the pinned API and declared discrete-time laws.

use umpire_dsl::machine;

/// Full protocol state, independently of feature state or public snapshots.
#[machine]
pub mod protocol {
    use umpire_dsl::{Action, Finite, Outcome, Rejection, State, World, check, reject, stay};

    /// Effective configured duration in abstract ticks.
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Ticks(#[inputs(1..=3)] pub u8);

    /// A latent remaining duration; armed zero is representable and invalid in reached states.
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Remaining {
        /// Remaining abstract ticks.
        #[inputs(0..=3)]
        pub ticks: u8,
    }

    /// Exact effective parameters. Raw request defaults belong to a separate admission boundary.
    #[derive(World)]
    pub struct World {
        /// Finite selected retry cap.
        pub maximum_attempts: u8,
        /// Per-queue-visit budget.
        pub schedule_to_start: Option<Ticks>,
        /// Per-attempt execution budget.
        pub start_to_close: Option<Ticks>,
        /// Budget spanning all queue visits and attempts.
        pub schedule_to_close: Option<Ticks>,
        /// Maximum interval between successful heartbeats.
        pub heartbeat_timeout: Option<Ticks>,
        /// Cooperative cancellation; false is the declared atomic compatibility abstraction.
        pub asynchronous_cancel: bool,
    }

    fn attempts(world: &World) -> impl Iterator<Item = u8> {
        1..=world.maximum_attempts
    }

    /// Internal scheduling state, distinct from the terminal result.
    #[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
    pub enum Lifecycle {
        /// Waiting for a worker assignment.
        #[default]
        Queued,
        /// A worker owns the current attempt.
        Executing,
        /// Execution has a terminal result.
        Closed,
    }

    /// Terminal outcome, never inferred from retained last-attempt failure.
    #[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
    pub enum Terminal {
        /// Execution is still open.
        #[default]
        None,
        /// A completion response won.
        Completed,
        /// A failure exhausted or blocked retry.
        Failed,
        /// Scheduled cancellation, explicit acknowledgement, or atomic compatibility cancellation.
        Canceled,
        /// A deadline ended execution without another attempt.
        TimedOut,
    }

    /// Latest selected failure/expiry classification, retained after later success.
    /// This protocol history does not assert an undocumented protobuf `last_failure` update.
    #[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
    pub enum Failure {
        /// No failed attempt has been recorded.
        #[default]
        None,
        /// Worker failure eligible for retry if the execution permits it.
        RetryableWorker,
        /// Worker failure prohibiting retry.
        PermanentWorker,
        /// Queue waiting limit.
        ScheduleToStart,
        /// Attempt execution limit.
        StartToClose,
        /// Whole execution limit.
        ScheduleToClose,
        /// Missing heartbeat limit.
        Heartbeat,
    }

    /// Client RPC intent; request-ID replay is outside this selected action domain.
    #[derive(Action, Copy)]
    pub enum Client {
        /// Request cancellation with a fresh logical intent.
        RequestCancel,
    }

    /// Worker protocol operations; delivery and acknowledgement are separate events.
    #[derive(Action, Copy)]
    pub enum Worker {
        /// Assign the selected attempt to a worker.
        Deliver(#[domain(attempts)] u8),
        /// Commit success for the assigned attempt.
        Complete,
        /// Commit a failure and evaluate the retry policy.
        Fail {
            /// Whether this failure permits retry.
            retryable: bool,
        },
        /// Report liveness and observe the existing cancellation request.
        Heartbeat,
        /// Explicitly acknowledge cancellation of the selected attempt.
        AcknowledgeCancel(#[domain(attempts)] u8),
    }

    /// Independent discrete-time abstraction.
    #[derive(Action, Copy)]
    #[internal]
    pub enum Timer {
        /// Consume one unit from every armed budget, then select one expiry.
        Tick,
    }

    /// Exact selected timeout cause, distinct from both outcome and retained failure.
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
    pub enum Expired {
        /// Queue waiting expired.
        ScheduleToStart,
        /// Current attempt expired.
        StartToClose,
        /// Whole execution expired.
        ScheduleToClose,
        /// Heartbeat interval expired.
        Heartbeat,
    }

    /// Complete protocol state. Public snapshot conversion deliberately excludes latent budgets.
    #[derive(State)]
    pub struct Model {
        /// Queue, running, or closed state.
        pub lifecycle: Lifecycle,
        /// Current attempt; increments only when scheduling a retry.
        pub attempt: u8,
        /// A running attempt has an unacknowledged cancellation request.
        pub cancellation_pending: bool,
        /// Current execution result.
        pub terminal_outcome: Terminal,
        /// Last failed attempt, independently of the current result.
        pub retained_failure: Failure,
        /// Current queue visit's remaining budget.
        pub queued_budget: Option<Remaining>,
        /// Current attempt's remaining budget.
        pub attempt_budget: Option<Remaining>,
        /// Spanning execution countdown.
        pub execution_budget: Option<Remaining>,
        /// Current heartbeat interval countdown.
        pub heartbeat_budget: Option<Remaining>,
    }

    impl Model {
        #[starts]
        fn initial(world: &World) -> impl IntoIterator<Item = Self> {
            [Self {
                lifecycle: Lifecycle::Queued,
                attempt: 1,
                cancellation_pending: false,
                terminal_outcome: Terminal::None,
                retained_failure: Failure::None,
                queued_budget: if world.schedule_to_start.is_some() {
                    Some(Remaining { ticks: world.schedule_to_start.unwrap_or(Ticks(1)).0 })
                } else {
                    None
                },
                attempt_budget: None,
                execution_budget: if world.schedule_to_close.is_some() {
                    Some(Remaining { ticks: world.schedule_to_close.unwrap_or(Ticks(1)).0 })
                } else {
                    None
                },
                heartbeat_budget: None,
            }]
        }

        #[on(Client)]
        fn client(&mut self, action: Client, world: &World) {
            match action {
                Client::RequestCancel => {
                    if self.lifecycle == Lifecycle::Closed {
                        reject!(Rejection::NotFound);
                    } else if self.cancellation_pending {
                        reject!(Rejection::FailedPrecondition);
                    } else if self.lifecycle == Lifecycle::Executing && world.asynchronous_cancel {
                        self.cancellation_pending = true;
                    } else {
                        self.terminal_outcome = Terminal::Canceled;
                        self.finish();
                    }
                }
            }
        }

        #[on(Worker)]
        fn worker(&mut self, action: Worker, world: &World) {
            match action {
                Worker::Deliver(token)
                    if self.lifecycle != Lifecycle::Closed && token != self.attempt =>
                {
                    reject!(Rejection::FailedPrecondition);
                }
                Worker::Deliver(_) if self.lifecycle == Lifecycle::Queued => {
                    self.lifecycle = Lifecycle::Executing;
                    self.queued_budget = None;
                    self.attempt_budget = if world.start_to_close.is_some() {
                        Some(Remaining { ticks: world.start_to_close.unwrap_or(Ticks(1)).0 })
                    } else {
                        None
                    };
                    self.heartbeat_budget = if world.heartbeat_timeout.is_some() {
                        Some(Remaining { ticks: world.heartbeat_timeout.unwrap_or(Ticks(1)).0 })
                    } else {
                        None
                    };
                }
                Worker::Complete if self.lifecycle == Lifecycle::Executing => {
                    self.terminal_outcome = Terminal::Completed;
                    self.finish();
                }
                Worker::Fail { retryable } if self.lifecycle == Lifecycle::Executing => {
                    self.retained_failure =
                        if retryable { Failure::RetryableWorker } else { Failure::PermanentWorker };
                    if retryable
                        && !self.cancellation_pending
                        && self.attempt < world.maximum_attempts
                    {
                        self.schedule_retry(world);
                    } else {
                        self.terminal_outcome = Terminal::Failed;
                        self.finish();
                    }
                }
                Worker::Heartbeat if self.lifecycle == Lifecycle::Executing => {
                    stay!();
                    self.heartbeat_budget = if world.heartbeat_timeout.is_some() {
                        Some(Remaining { ticks: world.heartbeat_timeout.unwrap_or(Ticks(1)).0 })
                    } else {
                        None
                    };
                }
                Worker::AcknowledgeCancel(token)
                    if self.lifecycle != Lifecycle::Closed && token != self.attempt =>
                {
                    reject!(Rejection::NotFound);
                }
                Worker::AcknowledgeCancel(_)
                    if self.lifecycle == Lifecycle::Executing && self.cancellation_pending =>
                {
                    self.terminal_outcome = Terminal::Canceled;
                    self.finish();
                }
                _ => {}
            }
        }

        #[on(Timer)]
        fn timer(&mut self, action: Timer, world: &World) -> Option<Expired> {
            match action {
                Timer::Tick => {
                    if self.lifecycle != Lifecycle::Closed
                        && (self.queued_budget.is_some()
                            || self.attempt_budget.is_some()
                            || self.execution_budget.is_some()
                            || self.heartbeat_budget.is_some())
                    {
                        self.consume();
                        if self.execution_budget.is_some()
                            && self.execution_budget.unwrap_or(Remaining { ticks: 0 }).ticks == 0
                        {
                            self.retained_failure = Failure::ScheduleToClose;
                            self.terminal_outcome = Terminal::TimedOut;
                            self.finish();
                            Some(Expired::ScheduleToClose)
                        } else if self.attempt_budget.is_some()
                            && self.attempt_budget.unwrap_or(Remaining { ticks: 0 }).ticks == 0
                        {
                            self.retained_failure = Failure::StartToClose;
                            self.attempt_expired(world);
                            Some(Expired::StartToClose)
                        } else if self.heartbeat_budget.is_some()
                            && self.heartbeat_budget.unwrap_or(Remaining { ticks: 0 }).ticks == 0
                        {
                            self.retained_failure = Failure::Heartbeat;
                            self.attempt_expired(world);
                            Some(Expired::Heartbeat)
                        } else if self.queued_budget.is_some()
                            && self.queued_budget.unwrap_or(Remaining { ticks: 0 }).ticks == 0
                        {
                            self.retained_failure = Failure::ScheduleToStart;
                            self.terminal_outcome = Terminal::TimedOut;
                            self.finish();
                            Some(Expired::ScheduleToStart)
                        } else {
                            None
                        }
                    } else {
                        None
                    }
                }
            }
        }

        #[effect]
        fn attempt_expired(&mut self, world: &World) {
            if !self.cancellation_pending && self.attempt < world.maximum_attempts {
                self.schedule_retry(world);
            } else {
                self.terminal_outcome = Terminal::TimedOut;
                self.finish();
            }
        }

        #[effect]
        fn schedule_retry(&mut self, world: &World) {
            self.attempt += 1;
            self.lifecycle = Lifecycle::Queued;
            self.queued_budget = if world.schedule_to_start.is_some() {
                Some(Remaining { ticks: world.schedule_to_start.unwrap_or(Ticks(1)).0 })
            } else {
                None
            };
            self.attempt_budget = None;
            self.heartbeat_budget = None;
        }

        #[effect]
        fn finish(&mut self) {
            self.lifecycle = Lifecycle::Closed;
            self.cancellation_pending = false;
            self.queued_budget = None;
            self.attempt_budget = None;
            self.execution_budget = None;
            self.heartbeat_budget = None;
        }

        #[effect]
        fn consume(&mut self) {
            if self.queued_budget.is_some()
                && self.queued_budget.unwrap_or(Remaining { ticks: 0 }).ticks > 0
            {
                self.queued_budget = Some(Remaining {
                    ticks: self.queued_budget.unwrap_or(Remaining { ticks: 0 }).ticks - 1,
                });
            }
            if self.attempt_budget.is_some()
                && self.attempt_budget.unwrap_or(Remaining { ticks: 0 }).ticks > 0
            {
                self.attempt_budget = Some(Remaining {
                    ticks: self.attempt_budget.unwrap_or(Remaining { ticks: 0 }).ticks - 1,
                });
            }
            if self.execution_budget.is_some()
                && self.execution_budget.unwrap_or(Remaining { ticks: 0 }).ticks > 0
            {
                self.execution_budget = Some(Remaining {
                    ticks: self.execution_budget.unwrap_or(Remaining { ticks: 0 }).ticks - 1,
                });
            }
            if self.heartbeat_budget.is_some()
                && self.heartbeat_budget.unwrap_or(Remaining { ticks: 0 }).ticks > 0
            {
                self.heartbeat_budget = Some(Remaining {
                    ticks: self.heartbeat_budget.unwrap_or(Remaining { ticks: 0 }).ticks - 1,
                });
            }
        }

        #[invariant]
        fn selected_attempts(&self, world: &World) {
            check!(world.maximum_attempts >= 1 && world.maximum_attempts <= 3);
            check!(self.attempt >= 1 && self.attempt <= world.maximum_attempts);
        }

        #[invariant]
        fn outcome_and_pending_are_distinct(&self, world: &World) {
            check!(
                (self.lifecycle == Lifecycle::Closed) == (self.terminal_outcome != Terminal::None)
            );
            check!(
                !self.cancellation_pending
                    || (world.asynchronous_cancel && self.lifecycle == Lifecycle::Executing)
            );
        }

        #[invariant]
        fn exact_budget_arming(&self, world: &World) {
            check!(
                self.queued_budget.is_some()
                    == (self.lifecycle == Lifecycle::Queued && world.schedule_to_start.is_some())
            );
            check!(
                self.attempt_budget.is_some()
                    == (self.lifecycle == Lifecycle::Executing && world.start_to_close.is_some())
            );
            check!(
                self.execution_budget.is_some()
                    == (self.lifecycle != Lifecycle::Closed && world.schedule_to_close.is_some())
            );
            check!(
                self.heartbeat_budget.is_some()
                    == (self.lifecycle == Lifecycle::Executing
                        && world.heartbeat_timeout.is_some())
            );
        }

        #[invariant]
        fn positive_bounded_budgets(&self, world: &World) {
            check!(
                self.queued_budget.is_none()
                    || (self.queued_budget.unwrap_or(Remaining { ticks: 0 }).ticks > 0
                        && self.queued_budget.unwrap_or(Remaining { ticks: 0 }).ticks
                            <= world.schedule_to_start.unwrap_or(Ticks(1)).0)
            );
            check!(
                self.attempt_budget.is_none()
                    || (self.attempt_budget.unwrap_or(Remaining { ticks: 0 }).ticks > 0
                        && self.attempt_budget.unwrap_or(Remaining { ticks: 0 }).ticks
                            <= world.start_to_close.unwrap_or(Ticks(1)).0)
            );
            check!(
                self.execution_budget.is_none()
                    || (self.execution_budget.unwrap_or(Remaining { ticks: 0 }).ticks > 0
                        && self.execution_budget.unwrap_or(Remaining { ticks: 0 }).ticks
                            <= world.schedule_to_close.unwrap_or(Ticks(1)).0)
            );
            check!(
                self.heartbeat_budget.is_none()
                    || (self.heartbeat_budget.unwrap_or(Remaining { ticks: 0 }).ticks > 0
                        && self.heartbeat_budget.unwrap_or(Remaining { ticks: 0 }).ticks
                            <= world.heartbeat_timeout.unwrap_or(Ticks(1)).0)
            );
        }

        #[postcondition]
        fn terminal_is_final(&self, before: &Self) {
            check!(before.lifecycle != Lifecycle::Closed || self == before);
        }

        #[postcondition]
        fn rejection_preserves_full_state(&self, before: &Self, outcome: &Outcome) {
            check!(*outcome == Outcome::Accepted || self == before);
        }

        #[postcondition]
        fn attempt_changes_only_by_retry(&self, before: &Self) {
            check!(
                self.attempt == before.attempt
                    || (self.attempt == before.attempt + 1
                        && before.lifecycle == Lifecycle::Executing
                        && self.lifecycle == Lifecycle::Queued
                        && !before.cancellation_pending)
            );
        }

        #[postcondition(after = Worker::Heartbeat)]
        fn heartbeat_resets_only_its_interval(&self, before: &Self, world: &World) {
            check!(self.lifecycle == Lifecycle::Executing && self.attempt == before.attempt);
            check!(self.cancellation_pending == before.cancellation_pending);
            check!(
                self.terminal_outcome == before.terminal_outcome
                    && self.retained_failure == before.retained_failure
            );
            check!(
                self.queued_budget == before.queued_budget
                    && self.attempt_budget == before.attempt_budget
                    && self.execution_budget == before.execution_budget
            );
            check!(
                self.heartbeat_budget
                    == if world.heartbeat_timeout.is_some() {
                        Some(Remaining { ticks: world.heartbeat_timeout.unwrap_or(Ticks(1)).0 })
                    } else {
                        None
                    }
            );
        }

        #[postcondition]
        fn retry_rearms_queue_preserves_execution(&self, before: &Self, world: &World) {
            if self.attempt != before.attempt {
                check!(
                    self.queued_budget
                        == if world.schedule_to_start.is_some() {
                            Some(Remaining { ticks: world.schedule_to_start.unwrap_or(Ticks(1)).0 })
                        } else {
                            None
                        }
                );
                check!(self.attempt_budget.is_none() && self.heartbeat_budget.is_none());
                check!(self.terminal_outcome == Terminal::None && !self.cancellation_pending);
            }
        }

        #[postcondition]
        fn spanning_budget_never_increases(&self, before: &Self) {
            check!(
                self.lifecycle == Lifecycle::Closed
                    || before.execution_budget.is_none()
                    || self.execution_budget.unwrap_or(Remaining { ticks: 0 }).ticks
                        <= before.execution_budget.unwrap_or(Remaining { ticks: 0 }).ticks
            );
        }

        #[postcondition(after = Client::RequestCancel)]
        fn cancellation_request_preserves_running_timers(&self, before: &Self) {
            if self.cancellation_pending {
                check!(
                    self.queued_budget == before.queued_budget
                        && self.attempt_budget == before.attempt_budget
                        && self.execution_budget == before.execution_budget
                        && self.heartbeat_budget == before.heartbeat_budget
                );
                check!(
                    self.attempt == before.attempt
                        && self.retained_failure == before.retained_failure
                );
            }
        }

        #[postcondition(after = Timer::Tick)]
        fn every_live_budget_descends(&self, before: &Self) {
            check!(
                self.lifecycle == Lifecycle::Closed
                    || before.execution_budget.is_none()
                    || before.execution_budget.unwrap_or(Remaining { ticks: 0 }).ticks
                        == self.execution_budget.unwrap_or(Remaining { ticks: 0 }).ticks + 1
            );
            if self.lifecycle != Lifecycle::Closed && self.attempt == before.attempt {
                check!(
                    before.queued_budget.is_none()
                        || before.queued_budget.unwrap_or(Remaining { ticks: 0 }).ticks
                            == self.queued_budget.unwrap_or(Remaining { ticks: 0 }).ticks + 1
                );
                check!(
                    before.attempt_budget.is_none()
                        || before.attempt_budget.unwrap_or(Remaining { ticks: 0 }).ticks
                            == self.attempt_budget.unwrap_or(Remaining { ticks: 0 }).ticks + 1
                );
                check!(
                    before.heartbeat_budget.is_none()
                        || before.heartbeat_budget.unwrap_or(Remaining { ticks: 0 }).ticks
                            == self.heartbeat_budget.unwrap_or(Remaining { ticks: 0 }).ticks + 1
                );
            }
        }

        #[postcondition(after = Timer::Tick)]
        fn execution_expiry_dominates(&self, before: &Self) {
            check!(
                before.execution_budget != Some(Remaining { ticks: 1 })
                    || (self.terminal_outcome == Terminal::TimedOut
                        && self.retained_failure == Failure::ScheduleToClose)
            );
        }

        #[postcondition(after = Timer::Tick)]
        fn local_expiry_priority_retains_its_cause(&self, before: &Self) {
            if before.execution_budget != Some(Remaining { ticks: 1 }) {
                if before.attempt_budget == Some(Remaining { ticks: 1 }) {
                    check!(self.retained_failure == Failure::StartToClose);
                }
                if before.attempt_budget != Some(Remaining { ticks: 1 })
                    && before.heartbeat_budget == Some(Remaining { ticks: 1 })
                {
                    check!(self.retained_failure == Failure::Heartbeat);
                }
                if before.attempt_budget != Some(Remaining { ticks: 1 })
                    && before.heartbeat_budget != Some(Remaining { ticks: 1 })
                    && before.queued_budget == Some(Remaining { ticks: 1 })
                {
                    check!(self.retained_failure == Failure::ScheduleToStart);
                }
            }
        }

        #[postcondition]
        fn pending_failure_is_not_acknowledgement(&self, before: &Self) {
            check!(
                !before.cancellation_pending
                    || self.retained_failure == before.retained_failure
                    || self.terminal_outcome == Terminal::Failed
                    || self.terminal_outcome == Terminal::TimedOut
            );
        }

        #[postcondition(after = Timer::Tick)]
        fn pending_expiry_is_not_acknowledgement(&self, before: &Self) {
            check!(
                !before.cancellation_pending
                    || self.lifecycle != Lifecycle::Closed
                    || self.terminal_outcome == Terminal::TimedOut
            );
        }
    }
}
