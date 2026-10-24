//! A bounded optional countdown, independent of handler facts and activity policy.

use umpire::dsl::machine;

#[machine]
pub mod countdown {
    use umpire::dsl::{Action, Finite, State, World, check, path};

    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Ticks(#[inputs(1..=3)] pub u8);

    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Remaining {
        #[inputs(0..=3)]
        pub ticks: u8,
    }

    const REMAINING: Option<Remaining> = Some(Remaining { ticks: 0 });

    #[derive(World)]
    pub struct World {
        pub timeout: Option<Ticks>,
        pub remaining: Option<Remaining>,
        pub empty: bool,
        pub duplicates: bool,
        pub invalid_update: bool,
    }

    #[derive(State)]
    pub struct Model {
        pub remaining: Option<Remaining>,
        pub started: bool,
    }

    #[derive(Action, Copy)]
    pub enum Clock {
        Start,
        Tick,
    }

    impl Model {
        #[starts]
        fn starts(selected: &World) -> impl IntoIterator<Item = Self> {
            if selected.empty {
                Vec::new()
            } else if selected.duplicates {
                vec![
                    Self { remaining: selected.remaining, started: false },
                    Self { remaining: None, started: true },
                    Self { remaining: selected.remaining, started: false },
                ]
            } else {
                vec![Self { remaining: selected.remaining, started: false }]
            }
        }

        #[on(Clock)]
        fn clock(&mut self, requested: Clock, selected: &World) {
            match requested {
                Clock::Start if !self.started => {
                    self.started = true;
                    if selected.timeout.is_some() {
                        self.remaining =
                            Some(Remaining { ticks: selected.timeout.unwrap_or(Ticks(1)).0 });
                    } else {
                        self.remaining = None;
                    }
                }
                Clock::Tick
                    if self.started
                        && self.remaining.is_some()
                        && self.remaining.unwrap_or(Remaining { ticks: 0 }).ticks > 0 =>
                {
                    if selected.invalid_update {
                        self.remaining = Some(Remaining { ticks: 4 });
                    } else {
                        self.remaining = Some(Remaining {
                            ticks: self.remaining.unwrap_or(Remaining { ticks: 0 }).ticks - 1,
                        });
                    }
                }
                _ => {}
            }
        }

        #[invariant]
        fn bounded(&self, selected: &World) {
            check!(
                self.remaining.is_none()
                    || Remaining::contains(self.remaining.unwrap_or(Remaining { ticks: 0 }).ticks)
            );
            check!(
                selected.timeout.is_none()
                    || Ticks::contains(selected.timeout.unwrap_or(Ticks(1)).0)
            );
        }

        #[invariant]
        fn never_exhausted(&self) {
            check!(!self.started || self.remaining != REMAINING);
        }

        #[postcondition(after = Clock::Tick)]
        fn decrements(&self, previous: &Self, selected: &World) {
            check!(
                previous.remaining.unwrap_or(Remaining { ticks: 0 }).ticks
                    == self.remaining.unwrap_or(Remaining { ticks: 0 }).ticks + 1
            );
            check!(selected.timeout.is_some());
        }

        #[postcondition(after = Clock::Start)]
        fn starts_selected(&self, previous: &Self, selected: &World) {
            check!(!previous.started);
            check!(self.started);
            check!(self.remaining.is_some() == selected.timeout.is_some());
        }

        #[progress]
        fn finishes(&self) -> LeadsTo {
            path![self.started, ..=3, self.remaining.is_none() || self.remaining == REMAINING]
        }
    }
}
