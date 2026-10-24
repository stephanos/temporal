//! Exact World-selected fairness for retries, stale polls and an explicit close.

use umpire::dsl::machine;

#[machine]
pub mod retry {
    use umpire::dsl::{Action, State, World, check, path, stay};

    #[derive(World)]
    pub struct World {
        pub maximum: u8,
    }

    fn attempts(selected: &World) -> impl Iterator<Item = u8> {
        1..=selected.maximum
    }

    #[derive(State)]
    pub struct Model {
        pub attempt: u8,
        pub ready: bool,
        pub closed: bool,
    }

    #[derive(Action, Copy)]
    pub enum Worker {
        Poll(#[domain(attempts)] u8),
        Retry,
        Close,
        Heartbeat,
    }

    impl Model {
        #[starts]
        fn initial() -> impl IntoIterator<Item = Self> {
            [Self { attempt: 1, ready: false, closed: false }]
        }

        #[on(Worker)]
        fn worker(&mut self, requested: Worker, constants: &World) {
            match requested {
                Worker::Poll(attempt) if !self.ready && !self.closed && attempt == self.attempt => {
                    self.ready = true;
                }
                Worker::Poll(_) if !self.ready && !self.closed => stay!(),
                Worker::Retry if self.ready && !self.closed && self.attempt < constants.maximum => {
                    self.attempt += 1;
                    self.ready = false;
                }
                Worker::Close if self.ready && !self.closed => self.closed = true,
                Worker::Heartbeat if !self.closed => stay!(),
                _ => {}
            }
        }

        #[invariant]
        fn bounded(&self, constants: &World) {
            check!(self.attempt >= 1 && self.attempt <= constants.maximum);
            check!(!self.closed || self.ready);
        }

        #[postcondition]
        fn monotonic(&self, before: &Self) {
            check!(self.attempt >= before.attempt);
            check!(!before.closed || self.closed);
        }

        #[assumption]
        fn selected_polls(constants: &World) -> Assumption {
            if constants.maximum == 1 {
                fair([Worker::Poll(1), Worker::Close])
            } else if constants.maximum == 2 {
                fair([Worker::Poll(1), Worker::Poll(2), Worker::Close])
            } else {
                fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])
            }
        }

        #[progress]
        fn closes(&self, constants: &World) -> LeadsTo {
            path![!self.closed, .., self.closed].assuming(Self::selected_polls(constants))
        }
    }
}
