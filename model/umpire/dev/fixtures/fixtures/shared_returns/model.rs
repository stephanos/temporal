use umpire::dsl::machine;

/// Sequential conditional returns whose later conditions read the updates before them, so the
/// emitted steps share subexpressions through `LET` definitions, with and without a payload. A
/// condition after a return would overflow where the step has already returned.
#[machine]
pub mod gate {
    use umpire::dsl::{Action, Outcome, Rejection, State, World, check, reject};

    #[derive(World)]
    pub struct World {
        limit: u8,
    }

    fn amounts(world: &World) -> impl Iterator<Item = u8> {
        1..=world.limit
    }

    #[derive(Action, Copy)]
    pub enum Client {
        Step,
        Add(#[domain(amounts)] u8),
        Bump,
    }

    #[derive(State)]
    pub struct Model {
        pub count: u8,
        pub done: bool,
        #[starts(250..=255)]
        pub level: u8,
    }

    impl Model {
        #[on(Client)]
        fn client(&mut self, action: Client) {
            match action {
                Client::Step if self.count <= 3 => {
                    if self.count == 3 {
                        reject!(Rejection::NotFound);
                    }
                    self.count += 1;
                    if self.count == 3 {
                        reject!(Rejection::NotFound);
                    }
                    self.count += 1;
                    if self.count == 3 {
                        reject!(Rejection::NotFound);
                    }
                    self.count += 1;
                    self.done = true;
                }
                Client::Add(amount) if self.count == 0 => {
                    if self.count == 3 {
                        reject!(Rejection::NotFound);
                    }
                    self.count += amount;
                    if self.count == 3 {
                        reject!(Rejection::NotFound);
                    }
                    self.count += amount;
                    if self.count == 3 {
                        reject!(Rejection::NotFound);
                    }
                    self.count += amount;
                    self.done = true;
                }
                Client::Bump => {
                    if self.level == 255 {
                        reject!(Rejection::Unavailable);
                    }
                    // Natively `level + 1` is never computed at 255.
                    if self.level + 1 > 252 {
                        self.level = 250;
                    } else {
                        self.level += 1;
                    }
                }
                _ => {}
            }
        }

        /// Broken: a step from three rejects, so the counterexample replays two shared steps.
        #[postcondition(after = Client::Step)]
        fn steps_accept(&self, _before: &Self, outcome: &Outcome) {
            check!(*outcome == Outcome::Accepted || self.count < 3);
        }
    }
}
