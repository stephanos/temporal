//! A worker polls with the attempt it expects, retries on a retryable failure and may skip.
//! Poll's attempts and Skip's slots are World-sized; a stale Poll is an accepted self-loop.

use umpire_dsl::machine;

#[machine]
pub mod inputs {
    use umpire_dsl::{Action, State, World, check, path, stay};

    #[derive(World)]
    pub struct World {
        attempts: u8,
        spares: u8,
    }

    fn attempts(world: &World) -> impl Iterator<Item = u8> {
        1..=world.attempts
    }

    fn spares(world: &World) -> impl Iterator<Item = u8> {
        0..world.spares
    }

    #[derive(State)]
    pub struct Model {
        pub attempt: u8,
        pub done: bool,
        pub failed: bool,
        pub skipped: bool,
    }

    #[derive(Action, Copy)]
    pub enum Worker {
        Poll(#[domain(attempts)] u8),
        Fail { retryable: bool },
        Skip(#[domain(spares)] u8),
    }

    impl Model {
        #[starts]
        fn initial() -> impl IntoIterator<Item = Self> {
            [Self { attempt: 1, done: false, failed: false, skipped: false }]
        }

        #[on(Worker)]
        fn worker(&mut self, action: Worker, selected: &World) {
            match action {
                Worker::Poll(attempt) if attempt == self.attempt && !self.done && !self.failed => {
                    self.done = true;
                }
                Worker::Poll(_) => stay!(),
                Worker::Fail { retryable } if !self.done && !self.failed => {
                    if retryable && self.attempt < selected.attempts {
                        self.attempt += 1;
                    } else {
                        self.failed = true;
                    }
                }
                Worker::Skip(_) if !self.skipped => self.skipped = true,
                _ => {}
            }
        }

        #[invariant]
        fn attempts_bounded(&self, selected: &World) {
            check!(self.attempt >= 1 && self.attempt <= selected.attempts);
        }

        #[postcondition]
        fn attempts_never_decrease(&self, before: &Self) {
            check!(self.attempt >= before.attempt);
            if before.done {
                check!(self.done);
            }
        }

        #[assumption]
        fn current_polls() -> Assumption {
            fair([Worker::Poll(1), Worker::Poll(2)])
        }

        #[assumption]
        fn stale_poll() -> Assumption {
            fair([Worker::Poll(2)])
        }

        #[assumption]
        fn first_excluded() -> Assumption {
            without([Worker::Poll(1)]).fair([Worker::Poll(1), Worker::Poll(2)])
        }

        /// Each continuously enabled fair current Poll is eventually taken; a third attempt's Poll
        /// is owed nothing, however often the fair stale ones are taken.
        #[progress]
        fn current_poll_settles(&self) -> LeadsTo {
            path![!self.done && !self.failed, .., self.done || self.failed]
                .assuming(Self::current_polls())
        }

        /// Fairness owed to the second attempt's Poll is discharged by its stale self-loop alone.
        #[progress]
        fn stale_poll_settles(&self) -> LeadsTo {
            path![!self.done && !self.failed, .., self.done || self.failed]
                .assuming(Self::stale_poll())
        }

        /// Fair membership does not admit the excluded first attempt.
        #[progress]
        fn excluded_poll_settles(&self) -> LeadsTo {
            path![!self.done && !self.failed, .., self.done || self.failed]
                .assuming(Self::first_excluded())
        }

        /// A bound receives no fairness benefit: stale polls may delay settling.
        #[progress]
        fn settles_within_four(&self) -> LeadsTo {
            path![!self.done && !self.failed, ..=4, self.done || self.failed]
                .assuming(Self::current_polls())
        }
    }
}
