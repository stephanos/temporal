//! A job runs once per attempt and closes when finished or dropped. Phase access, rejecting
//! outcomes and the stock closure laws are lifted together. Three terminal transitions are
//! deliberately broken so that each law has an actual counterexample: `Reopen` reopens a
//! dropped job, `Touch` accepts a step from closure, and `Recheck` rejects with the wrong status.

use umpire::dsl::machine;

#[machine]
pub mod job {
    use umpire::dsl::{
        Action, Outcome, Phased, Rejection, State, World, check, closable, path, reject, stay,
    };

    #[derive(World)]
    pub struct World {
        attempts: u8,
    }

    fn attempts(world: &World) -> impl Iterator<Item = u8> {
        1..=world.attempts
    }

    #[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
    pub enum Phase {
        #[default]
        Queued,
        Running,
        Done,
        Dropped,
    }

    impl umpire::dsl::Phase for Phase {
        fn closed(&self) -> bool {
            matches!(self, Self::Done | Self::Dropped)
        }
    }

    #[derive(Action, Copy)]
    pub enum Worker {
        /// Claim the job for an attempt; only the current attempt's claim runs it.
        Claim(#[domain(attempts)] u8),
        Finish,
        /// Broken: requeue a dropped job for its next attempt.
        Reopen,
    }

    #[derive(Action, Copy)]
    pub enum Client {
        Drop,
        /// Broken: accept a step from closure.
        Touch,
        /// Broken: reject a step from closure with the wrong status.
        Recheck,
    }

    #[derive(State)]
    pub struct Model {
        pub phase: Phase,
        #[starts(1..=1)]
        pub attempt: u8,
    }

    impl Phased for Model {
        type Phase = Phase;

        fn phase(&self) -> Self::Phase {
            self.phase
        }
    }

    impl Model {
        #[on(Worker)]
        fn worker(&mut self, action: Worker, selected: &World) {
            match action {
                Worker::Claim(_) if self.closed() => reject!(Rejection::NotFound),
                Worker::Claim(attempt) if attempt != self.attempt => {
                    reject!(Rejection::FailedPrecondition)
                }
                Worker::Claim(_) if self.phase == Phase::Queued => self.phase = Phase::Running,
                Worker::Finish if self.phase == Phase::Running => self.phase = Phase::Done,
                Worker::Reopen if self.phase == Phase::Dropped && self.attempt < selected.attempts => {
                    self.phase = Phase::Queued;
                    self.attempt += 1;
                }
                _ => {}
            }
        }

        #[on(Client)]
        fn client(&mut self, action: Client) {
            match action {
                Client::Drop => {
                    self.require_open();
                    // The helper's rejection is the first one; this one only returns.
                    if self.closed() {
                        reject!(Rejection::FailedPrecondition);
                    }
                    self.phase = Phase::Dropped;
                }
                Client::Touch if self.closed() => stay!(),
                Client::Recheck if self.closed() => reject!(Rejection::Unavailable),
                _ => {}
            }
        }

        #[effect]
        fn require_open(&mut self) {
            if self.closed() {
                reject!(Rejection::NotFound);
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

        #[postcondition(after = Client::Drop)]
        fn drop_rejects(&self, before: &Self, outcome: &Outcome) {
            closable::closed_rejects(self, before, outcome, Rejection::NotFound);
        }

        #[postcondition(after = Worker::Reopen)]
        fn reopen_remains(&self, before: &Self) {
            closable::terminal_remains(self, before);
        }

        #[postcondition(after = Client::Touch)]
        fn touch_remains(&self, before: &Self) {
            closable::terminal_remains(self, before);
        }

        #[postcondition(after = Client::Touch)]
        fn touch_rejects(&self, before: &Self, outcome: &Outcome) {
            closable::closed_rejects(self, before, outcome, Rejection::NotFound);
        }

        #[postcondition(after = Client::Recheck)]
        fn recheck_rejects(&self, before: &Self, outcome: &Outcome) {
            closable::closed_rejects(self, before, outcome, Rejection::NotFound);
        }

        /// Broken: a step that leaves the job open is accepted. Only a stale claim rejects
        /// there, so the counterexample replays a rejected payload edge.
        #[postcondition]
        fn steps_accept(&self, _before: &Self, outcome: &Outcome) {
            check!(*outcome == Outcome::Accepted || self.closed());
        }

        #[assumption]
        fn current_claims() -> Assumption {
            fair([Worker::Claim(1), Worker::Claim(2)])
        }

        /// Each continuously enabled current claim is eventually taken; a stale claim's
        /// rejecting self-loop discharges only its own payload's fairness.
        #[progress]
        fn queued_job_runs(&self) -> LeadsTo {
            path![self.phase == Phase::Queued, .., self.phase != Phase::Queued]
                .assuming(Self::current_claims())
        }
    }
}
