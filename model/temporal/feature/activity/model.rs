//! Bounded activity retries, optional deadlines and cooperative cancellation.
//! Successful Start establishes Scheduled at attempt one outside this graph.

use umpire_dsl::machine;

/// One activity's finite lifecycle, independent of any SDK or server observation.
#[machine]
pub mod activity {
    use umpire_dsl::{
        Action, Finite, Outcome, Phased, Rejection, State, World, check, closable, path, reject,
        stay,
    };

    /// Positive configured timeout in abstract ticks.
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Ticks(#[inputs(1..=3)] pub u8);

    /// A live countdown, with zero representable for exhaustion checks.
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Remaining {
        /// Number of ticks remaining.
        #[inputs(0..=3)]
        pub ticks: u8,
    }

    /// Fixed retry, timeout and cancellation policy.
    #[derive(World)]
    pub struct World {
        /// Maximum number of attempts, selected from one through three.
        pub maximum_attempts: u8,
        /// Waiting budget, rearmed for each retry.
        pub schedule_to_start: Option<Ticks>,
        /// Running budget, armed on delivery.
        pub start_to_close: Option<Ticks>,
        /// Global budget spanning every attempt.
        pub schedule_to_close: Option<Ticks>,
        /// Running budget reset by heartbeats.
        pub heartbeat_timeout: Option<Ticks>,
        /// Whether cancellation of a running attempt is cooperative.
        pub asynchronous_cancel: bool,
    }

    fn attempts(world: &World) -> impl Iterator<Item = u8> {
        1..=world.maximum_attempts
    }

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
        /// Cancellation won the terminal race.
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
        /// A fresh distinct cancellation intent; request-ID replay is unspecified.
        Cancel,
    }

    /// Actions controlled by the activity worker.
    #[derive(Action, Copy)]
    pub enum Worker {
        /// Deliver the scheduled activity at the selected attempt.
        Poll(#[domain(attempts)] u8),
        /// Complete a started activity.
        RespondCompleted,
        /// Fail a started activity, allowing retry when policy permits.
        RespondFailed {
            /// Whether this failure permits another attempt.
            retryable: bool,
        },
        /// Reset the heartbeat budget and report the existing pending bit.
        Heartbeat,
        /// Acknowledge pending cancellation for this attempt.
        RespondCanceled(#[domain(attempts)] u8),
    }

    /// Internal abstract clock; disabled without an armed budget.
    #[derive(Action, Copy)]
    #[internal]
    pub enum Deadline {
        /// Decrement every live budget once and select the highest-priority expiry.
        Tick,
    }

    /// The selected expiry trigger, independently of the terminal phase.
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
    pub enum ExpiredDeadline {
        /// Waiting for delivery expired.
        ScheduleToStart,
        /// The running attempt expired.
        StartToClose,
        /// The spanning execution deadline expired.
        ScheduleToClose,
        /// The running heartbeat deadline expired.
        Heartbeat,
    }

    /// Model's dynamic state. Terminal transitions clear every budget and pending bit.
    #[derive(State)]
    pub struct Model {
        /// Current lifecycle phase.
        pub phase: Phase,
        /// Current attempt, starting at one.
        pub attempt: u8,
        /// A cooperative cancellation request awaiting an attempt boundary.
        pub cancel_requested: bool,
        /// Current waiting budget.
        pub schedule_to_start_remaining: Option<Remaining>,
        /// Current attempt budget.
        pub start_to_close_remaining: Option<Remaining>,
        /// Global budget, never replenished by retries or worker activity.
        pub schedule_to_close_remaining: Option<Remaining>,
        /// Current heartbeat budget.
        pub heartbeat_remaining: Option<Remaining>,
    }

    impl Phased for Model {
        type Phase = Phase;

        fn phase(&self) -> Self::Phase {
            self.phase
        }
    }

    impl Model {
        #[starts]
        fn initial(world: &World) -> impl IntoIterator<Item = Self> {
            [Self {
                phase: Phase::Scheduled,
                attempt: 1,
                cancel_requested: false,
                schedule_to_start_remaining: if world.schedule_to_start.is_some() {
                    Some(Remaining { ticks: world.schedule_to_start.unwrap_or(Ticks(1)).0 })
                } else {
                    None
                },
                start_to_close_remaining: None,
                schedule_to_close_remaining: if world.schedule_to_close.is_some() {
                    Some(Remaining { ticks: world.schedule_to_close.unwrap_or(Ticks(1)).0 })
                } else {
                    None
                },
                heartbeat_remaining: None,
            }]
        }

        #[on(Client)]
        fn client(&mut self, action: Client, world: &World) {
            match action {
                Client::Cancel if self.closed() => reject!(Rejection::NotFound),
                Client::Cancel if self.cancel_requested => reject!(Rejection::FailedPrecondition),
                Client::Cancel if world.asynchronous_cancel && self.phase == Phase::Started => {
                    self.cancel_requested = true;
                }
                Client::Cancel => {
                    self.phase = Phase::Canceled;
                    self.clear_terminal();
                }
            }
        }

        #[on(Worker)]
        fn worker(&mut self, action: Worker, world: &World) {
            match action {
                Worker::Poll(expected) if !self.closed() && expected != self.attempt => {
                    reject!(Rejection::FailedPrecondition);
                }
                Worker::Poll(_) if self.phase == Phase::Scheduled => {
                    self.phase = Phase::Started;
                    self.schedule_to_start_remaining = None;
                    if world.start_to_close.is_some() {
                        self.start_to_close_remaining =
                            Some(Remaining { ticks: world.start_to_close.unwrap_or(Ticks(1)).0 });
                    }
                    if world.heartbeat_timeout.is_some() {
                        self.heartbeat_remaining = Some(Remaining {
                            ticks: world.heartbeat_timeout.unwrap_or(Ticks(1)).0,
                        });
                    }
                }
                Worker::RespondCompleted if self.phase == Phase::Started => {
                    self.phase = Phase::Completed;
                    self.clear_terminal();
                }
                Worker::RespondFailed { retryable } if self.phase == Phase::Started => {
                    if retryable && self.cancel_requested {
                        self.phase = Phase::Canceled;
                        self.clear_terminal();
                    } else if retryable && self.attempt < world.maximum_attempts {
                        self.retry(world);
                    } else {
                        self.phase = Phase::Failed;
                        self.clear_terminal();
                    }
                }
                Worker::Heartbeat if self.phase == Phase::Started => {
                    stay!();
                    if world.heartbeat_timeout.is_some() {
                        self.heartbeat_remaining = Some(Remaining {
                            ticks: world.heartbeat_timeout.unwrap_or(Ticks(1)).0,
                        });
                    }
                }
                Worker::RespondCanceled(expected) if !self.closed() && expected != self.attempt => {
                    reject!(Rejection::NotFound);
                }
                Worker::RespondCanceled(_)
                    if self.phase == Phase::Started && self.cancel_requested =>
                {
                    self.phase = Phase::Canceled;
                    self.clear_terminal();
                }
                _ => {}
            }
        }

        #[on(Deadline)]
        fn deadline(&mut self, action: Deadline, world: &World) -> Option<ExpiredDeadline> {
            match action {
                Deadline::Tick
                    if !self.closed()
                        && (self.schedule_to_start_remaining.is_some()
                            || self.start_to_close_remaining.is_some()
                            || self.schedule_to_close_remaining.is_some()
                            || self.heartbeat_remaining.is_some()) =>
                {
                    self.decrement();
                    if self.schedule_to_close_remaining.is_some()
                        && self.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                            == 0
                    {
                        self.phase = Phase::TimedOut;
                        self.clear_terminal();
                        Some(ExpiredDeadline::ScheduleToClose)
                    } else if (self.start_to_close_remaining.is_some()
                        && self.start_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                            == 0)
                        || (self.heartbeat_remaining.is_some()
                            && self.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                                == 0)
                    {
                        let start_to_close_expired = self.start_to_close_remaining.is_some()
                            && self
                                .start_to_close_remaining
                                .unwrap_or(Remaining { ticks: 0 })
                                .ticks
                                == 0;
                        self.expire_attempt(world);
                        if start_to_close_expired {
                            Some(ExpiredDeadline::StartToClose)
                        } else {
                            Some(ExpiredDeadline::Heartbeat)
                        }
                    } else if self.schedule_to_start_remaining.is_some()
                        && self.schedule_to_start_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                            == 0
                    {
                        self.phase = Phase::TimedOut;
                        self.clear_terminal();
                        Some(ExpiredDeadline::ScheduleToStart)
                    } else {
                        None
                    }
                }
                Deadline::Tick => None,
            }
        }

        #[effect]
        fn retry(&mut self, world: &World) {
            self.attempt += 1;
            self.phase = Phase::Scheduled;
            self.start_to_close_remaining = None;
            self.heartbeat_remaining = None;
            if world.schedule_to_start.is_some() {
                self.schedule_to_start_remaining =
                    Some(Remaining { ticks: world.schedule_to_start.unwrap_or(Ticks(1)).0 });
            }
        }

        #[effect]
        fn expire_attempt(&mut self, world: &World) {
            if self.cancel_requested {
                self.phase = Phase::Canceled;
                self.clear_terminal();
            } else if self.attempt < world.maximum_attempts {
                self.retry(world);
            } else {
                self.phase = Phase::TimedOut;
                self.clear_terminal();
            }
        }

        #[effect]
        fn clear_terminal(&mut self) {
            self.cancel_requested = false;
            self.schedule_to_start_remaining = None;
            self.start_to_close_remaining = None;
            self.schedule_to_close_remaining = None;
            self.heartbeat_remaining = None;
        }

        #[effect]
        fn decrement(&mut self) {
            if self.schedule_to_start_remaining.is_some()
                && self.schedule_to_start_remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0
            {
                self.schedule_to_start_remaining = Some(Remaining {
                    ticks: self.schedule_to_start_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        - 1,
                });
            }
            if self.start_to_close_remaining.is_some()
                && self.start_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0
            {
                self.start_to_close_remaining = Some(Remaining {
                    ticks: self.start_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        - 1,
                });
            }
            if self.schedule_to_close_remaining.is_some()
                && self.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0
            {
                self.schedule_to_close_remaining = Some(Remaining {
                    ticks: self.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        - 1,
                });
            }
            if self.heartbeat_remaining.is_some()
                && self.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0
            {
                self.heartbeat_remaining = Some(Remaining {
                    ticks: self.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks - 1,
                });
            }
        }

        #[invariant]
        fn state_bounds(&self, world: &World) {
            check!(world.maximum_attempts >= 1 && world.maximum_attempts <= 3);
            check!(self.attempt >= 1 && self.attempt <= world.maximum_attempts);
            check!(
                world.schedule_to_start.is_none()
                    || Ticks::contains(world.schedule_to_start.unwrap_or(Ticks(1)).0)
            );
            check!(
                self.schedule_to_start_remaining.is_none()
                    || (Remaining::contains(
                        self.schedule_to_start_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                    ) && self
                        .schedule_to_start_remaining
                        .unwrap_or(Remaining { ticks: 0 })
                        .ticks
                        > 0
                        && self
                            .schedule_to_start_remaining
                            .unwrap_or(Remaining { ticks: 0 })
                            .ticks
                            <= world.schedule_to_start.unwrap_or(Ticks(1)).0)
            );
            check!(
                world.start_to_close.is_none()
                    || Ticks::contains(world.start_to_close.unwrap_or(Ticks(1)).0)
            );
            check!(
                self.start_to_close_remaining.is_none()
                    || (Remaining::contains(
                        self.start_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                    ) && self.start_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        > 0
                        && self.start_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                            <= world.start_to_close.unwrap_or(Ticks(1)).0)
            );
            check!(
                world.schedule_to_close.is_none()
                    || Ticks::contains(world.schedule_to_close.unwrap_or(Ticks(1)).0)
            );
            check!(
                self.schedule_to_close_remaining.is_none()
                    || (Remaining::contains(
                        self.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                    ) && self
                        .schedule_to_close_remaining
                        .unwrap_or(Remaining { ticks: 0 })
                        .ticks
                        > 0
                        && self
                            .schedule_to_close_remaining
                            .unwrap_or(Remaining { ticks: 0 })
                            .ticks
                            <= world.schedule_to_close.unwrap_or(Ticks(1)).0)
            );
            check!(
                world.heartbeat_timeout.is_none()
                    || Ticks::contains(world.heartbeat_timeout.unwrap_or(Ticks(1)).0)
            );
            check!(
                self.heartbeat_remaining.is_none()
                    || (Remaining::contains(
                        self.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                    ) && self.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0
                        && self.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                            <= world.heartbeat_timeout.unwrap_or(Ticks(1)).0)
            );
        }

        #[invariant]
        fn deadline_arming(&self, world: &World) {
            check!(
                self.schedule_to_start_remaining.is_some()
                    == (self.phase == Phase::Scheduled && world.schedule_to_start.is_some())
            );
            check!(
                self.start_to_close_remaining.is_some()
                    == (self.phase == Phase::Started && world.start_to_close.is_some())
            );
            check!(
                self.heartbeat_remaining.is_some()
                    == (self.phase == Phase::Started && world.heartbeat_timeout.is_some())
            );
            check!(
                self.schedule_to_close_remaining.is_some()
                    == (!self.closed() && world.schedule_to_close.is_some())
            );
        }

        #[invariant]
        fn pending_is_cooperative(&self, world: &World) {
            check!(
                !self.cancel_requested
                    || (world.asynchronous_cancel && self.phase == Phase::Started)
            );
        }

        #[postcondition]
        fn terminal_remains(&self, before: &Self) {
            closable::terminal_remains(self, before);
        }

        #[postcondition]
        fn closed_rejects(&self, before: &Self, outcome: &Outcome) {
            closable::closed_rejects(self, before, outcome, Rejection::NotFound);
        }

        #[postcondition]
        fn global_never_replenished(&self, before: &Self, world: &World) {
            check!(
                before.closed()
                    || self.closed()
                    || world.schedule_to_close.is_none()
                    || self.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        <= before
                            .schedule_to_close_remaining
                            .unwrap_or(Remaining { ticks: 0 })
                            .ticks
            );
        }

        #[postcondition(after = Deadline::Tick)]
        fn tick_descends_and_expires(&self, before: &Self, world: &World) {
            check!(!before.closed());
            check!(
                world.schedule_to_close.is_none()
                    || self.closed()
                    || before.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        == self.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                            + 1
            );
            check!(
                world.schedule_to_close.is_none()
                    || before.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        != 1
                    || self.phase == Phase::TimedOut
            );
        }

        #[postcondition(after = Deadline::Tick)]
        fn countdowns_decrement(&self, before: &Self, world: &World) {
            if before.phase == Phase::Started && self.phase == Phase::Scheduled {
                check!(!before.cancel_requested && before.attempt < world.maximum_attempts);
                check!(self.phase == Phase::Scheduled && self.attempt == before.attempt + 1);
                check!(!self.cancel_requested);
                check!(
                    self.schedule_to_start_remaining
                        == if world.schedule_to_start.is_some() {
                            Some(Remaining { ticks: world.schedule_to_start.unwrap_or(Ticks(1)).0 })
                        } else {
                            None
                        }
                );
                check!(
                    self.start_to_close_remaining.is_none() && self.heartbeat_remaining.is_none()
                );
            } else {
                check!(
                    self.closed()
                        || before.schedule_to_start_remaining.is_none()
                        || before
                            .schedule_to_start_remaining
                            .unwrap_or(Remaining { ticks: 0 })
                            .ticks
                            == self
                                .schedule_to_start_remaining
                                .unwrap_or(Remaining { ticks: 0 })
                                .ticks
                                + 1
                );
                check!(
                    self.closed()
                        || before.start_to_close_remaining.is_none()
                        || before.start_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                            == self
                                .start_to_close_remaining
                                .unwrap_or(Remaining { ticks: 0 })
                                .ticks
                                + 1
                );
                check!(
                    self.closed()
                        || before.heartbeat_remaining.is_none()
                        || before.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                            == self.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks + 1
                );
            }
            check!(
                self.closed()
                    || before.schedule_to_close_remaining.is_none()
                    || before.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        == self.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                            + 1
            );
        }

        #[postcondition(after = Worker::Heartbeat)]
        fn heartbeat_preserves_attempt_and_global(&self, before: &Self, world: &World) {
            check!(before.phase == Phase::Started && self.phase == Phase::Started);
            check!(
                self.attempt == before.attempt && self.cancel_requested == before.cancel_requested
            );
            check!(self.schedule_to_close_remaining == before.schedule_to_close_remaining);
            check!(self.start_to_close_remaining == before.start_to_close_remaining);
            check!(self.schedule_to_start_remaining == before.schedule_to_start_remaining);
            check!(self.heartbeat_remaining.is_some() == world.heartbeat_timeout.is_some());
            check!(
                world.heartbeat_timeout.is_none()
                    || self.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        == world.heartbeat_timeout.unwrap_or(Ticks(1)).0
            );
        }

        #[postcondition]
        fn attempt_changes_only_on_retry(&self, before: &Self, world: &World) {
            check!(self.attempt <= world.maximum_attempts);
            check!(
                (before.phase == Phase::Started
                    && self.phase == Phase::Scheduled
                    && !before.cancel_requested
                    && self.attempt == before.attempt + 1)
                    || self.attempt == before.attempt
            );
            check!(
                before.phase != Phase::Started
                    || self.phase != Phase::Scheduled
                    || (!before.cancel_requested && self.attempt == before.attempt + 1)
            );
        }

        #[postcondition(after = Deadline::Tick)]
        fn tick_priority(&self, before: &Self, world: &World) {
            check!(
                !(before.schedule_to_close_remaining.is_some()
                    && before.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        <= 1)
                    || self.phase == Phase::TimedOut
            );
            check!(
                (before.schedule_to_close_remaining.is_some()
                    && before.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        <= 1)
                    || !((before.start_to_close_remaining.is_some()
                        && before
                            .start_to_close_remaining
                            .unwrap_or(Remaining { ticks: 0 })
                            .ticks
                            <= 1)
                        || (before.heartbeat_remaining.is_some()
                            && before.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                                <= 1))
                    || (before.cancel_requested && self.phase == Phase::Canceled)
                    || (!before.cancel_requested
                        && before.attempt < world.maximum_attempts
                        && self.phase == Phase::Scheduled)
                    || (!before.cancel_requested
                        && before.attempt == world.maximum_attempts
                        && self.phase == Phase::TimedOut)
            );
            check!(
                (before.schedule_to_close_remaining.is_some()
                    && before.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        <= 1)
                    || ((before.start_to_close_remaining.is_some()
                        && before
                            .start_to_close_remaining
                            .unwrap_or(Remaining { ticks: 0 })
                            .ticks
                            <= 1)
                        || (before.heartbeat_remaining.is_some()
                            && before.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                                <= 1))
                    || !(before.schedule_to_start_remaining.is_some()
                        && before
                            .schedule_to_start_remaining
                            .unwrap_or(Remaining { ticks: 0 })
                            .ticks
                            <= 1)
                    || self.phase == Phase::TimedOut
            );
            check!(
                (before.schedule_to_close_remaining.is_some()
                    && before.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                        <= 1)
                    || ((before.start_to_close_remaining.is_some()
                        && before
                            .start_to_close_remaining
                            .unwrap_or(Remaining { ticks: 0 })
                            .ticks
                            <= 1)
                        || (before.heartbeat_remaining.is_some()
                            && before.heartbeat_remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                                <= 1))
                    || (before.schedule_to_start_remaining.is_some()
                        && before
                            .schedule_to_start_remaining
                            .unwrap_or(Remaining { ticks: 0 })
                            .ticks
                            <= 1)
                    || self.phase == before.phase
            );
        }

        #[assumption]
        fn fair_worker(world: &World) -> Assumption {
            if world.maximum_attempts == 1 {
                fair([Worker::Poll(1), Worker::RespondCompleted])
            } else if world.maximum_attempts == 2 {
                fair([Worker::Poll(1), Worker::Poll(2), Worker::RespondCompleted])
            } else {
                fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::RespondCompleted])
            }
        }

        #[progress]
        fn eventually_closed(&self, world: &World) -> LeadsTo {
            path![!self.closed(), .., self.closed()].assuming(Self::fair_worker(world))
        }

        #[progress]
        fn pending_retryable_failure_cancels(&self, world: &World) -> LeadsTo {
            path![
                world.asynchronous_cancel && self.phase == Phase::Started && self.cancel_requested,
                ..=1,
                self.phase == Phase::Canceled
            ]
            .assuming(only([Worker::RespondFailed { retryable: true }]))
        }

        #[progress]
        fn pending_nonretryable_failure_fails(&self, world: &World) -> LeadsTo {
            path![
                world.asynchronous_cancel && self.phase == Phase::Started && self.cancel_requested,
                ..=1,
                self.phase == Phase::Failed
            ]
            .assuming(only([Worker::RespondFailed { retryable: false }]))
        }

        #[progress]
        fn ordinary_retry_rearms(&self, world: &World) -> LeadsTo {
            path![
                self.phase == Phase::Started
                    && !self.cancel_requested
                    && self.attempt < world.maximum_attempts,
                ..=1,
                self.phase == Phase::Scheduled
            ]
            .assuming(only([Worker::RespondFailed { retryable: true }]))
        }

        #[progress]
        fn ordinary_retry_at_cap_fails(&self, world: &World) -> LeadsTo {
            path![
                self.phase == Phase::Started
                    && !self.cancel_requested
                    && self.attempt == world.maximum_attempts,
                ..=1,
                self.phase == Phase::Failed
            ]
            .assuming(only([Worker::RespondFailed { retryable: true }]))
        }

        #[progress]
        fn ordinary_nonretryable_failure_fails(&self) -> LeadsTo {
            path![
                self.phase == Phase::Started && !self.cancel_requested,
                ..=1,
                self.phase == Phase::Failed
            ]
            .assuming(only([Worker::RespondFailed { retryable: false }]))
        }
    }
}
