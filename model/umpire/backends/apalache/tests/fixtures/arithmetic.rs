//! Finite source controls share one selected World, never an explored-edge table.
#![allow(clippy::assign_op_pattern)]
use umpire_dsl::machine;
#[machine]
pub(crate) mod arithmetic {
    use umpire_dsl::{Action, State, World, check};
    #[derive(World)]
    pub(crate) struct World {
        pub mode: u8,
        pub start: i8,
        pub divisor: i8,
    }
    #[derive(State)]
    pub(crate) struct Model {
        pub value: i8,
        pub divisor: i8,
        pub done: bool,
    }
    #[derive(Action, Copy)]
    pub(crate) enum User {
        Calculate,
    }
    impl Model {
        #[starts]
        fn starts(w: &World) -> impl IntoIterator<Item = Self> {
            [Self {
                value: if w.mode == 4 || w.mode == 5 { -128 } else { w.start },
                divisor: if w.mode == 4 || w.mode == 5 { -1 } else { w.divisor },
                done: false,
            }]
        }
        #[on(User)]
        fn calculate(&mut self, action: User, w: &World) {
            match action {
                User::Calculate if !self.done => {
                    if w.mode == 0 {
                        self.value += 1;
                    }
                    if w.mode == 1 {
                        self.value = (self.value + 1) - 1;
                    }
                    if w.mode == 2 || w.mode == 4 {
                        self.value = self.value / self.divisor;
                    }
                    if w.mode == 3 || w.mode == 5 {
                        self.value = self.value % self.divisor;
                    }
                    if w.mode == 6 {
                        self.value = if self.done { self.value / self.divisor } else { self.value };
                    }
                    if w.mode == 7 && self.done && self.value / self.divisor > 0 {
                        self.value = 0;
                    }
                    self.done = true;
                }
                User::Calculate => {}
            }
        }
        #[invariant]
        fn claims(&self, w: &World) {
            if w.mode == 8 {
                check!(false);
                check!(1 / self.divisor == 0);
            }
            if w.mode == 10 {
                check!(self.done || self.divisor == 0 || 1 / self.divisor == 0);
            }
        }
        #[postcondition(after=User::Calculate)]
        fn post(&self, w: &World) {
            if w.mode == 9 {
                check!(self.value / self.divisor == 0);
            }
        }
    }
}
