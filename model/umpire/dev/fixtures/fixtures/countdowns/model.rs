//! Shared countdown functions, expiry priority and exact saturating descent.

use umpire::dsl::machine;

#[machine]
pub mod clock {
    use umpire::dsl::countdown::{Remaining, Ticks};
    use umpire::dsl::{Action, State, World, check, countdown};

    #[derive(World)]
    pub struct World {
        pub first: Option<Ticks<3>>,
        pub second: Option<Ticks<3>>,
    }

    #[derive(State)]
    pub struct Model {
        pub first: Option<Remaining<3>>,
        pub second: Option<Remaining<3>>,
        pub probe: Option<Remaining<3>>,
        pub first_won: bool,
        pub second_won: bool,
    }

    #[derive(Action, Copy)]
    pub enum Clock {
        Reset,
        Tick,
    }

    impl Model {
        #[starts]
        fn starts(world: &World) -> impl IntoIterator<Item = Self> {
            vec![Self {
                first: countdown::arm(world.first),
                second: countdown::arm(world.second),
                probe: countdown::arm(world.first),
                first_won: false,
                second_won: false,
            }]
        }

        #[on(Clock)]
        fn clock(&mut self, action: Clock, world: &World) {
            match action {
                Clock::Reset => {
                    self.first = countdown::arm(world.first);
                    self.second = countdown::arm(world.second);
                    self.probe = countdown::arm(world.first);
                    self.first_won = false;
                    self.second_won = false;
                }
                Clock::Tick => {
                    self.probe = countdown::tick(self.probe);
                    self.advance();
                }
            }
        }

        #[effect]
        fn advance(&mut self) {
            self.first = countdown::tick(self.first);
            self.second = countdown::tick(self.second);
            if countdown::expired(self.first) {
                self.first_won = true;
                self.first = None;
                self.second = None;
            } else if countdown::expired(self.second) {
                self.second_won = true;
                self.first = None;
                self.second = None;
            }
        }

        #[invariant]
        fn within(&self, world: &World) {
            countdown::within(self.first, world.first);
            countdown::within(self.second, world.second);
            check!(
                world.first.is_none() || Ticks::<3>::contains(world.first.unwrap_or(Ticks(1)).0)
            );
            check!(
                self.probe.is_none()
                    || Remaining::<3>::contains(self.probe.unwrap_or(Remaining { ticks: 0 }).ticks)
            );
        }

        #[invariant]
        fn priority(&self) {
            check!(!self.first_won || !self.second_won);
            check!(
                !self.first_won && !self.second_won
                    || (!countdown::armed(self.first) && !countdown::armed(self.second))
            );
        }

        #[postcondition(after = Clock::Tick)]
        fn descends(&self, before: &Self) {
            countdown::descends(self.probe, before.probe);
        }
    }
}
