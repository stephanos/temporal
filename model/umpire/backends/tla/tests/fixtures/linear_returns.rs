//! Compilation beyond the former admitted-tree and rustc memory limits.
#![expect(clippy::too_many_lines, reason = "unrolled statements reproduce admitted-data growth")]
use umpire_dsl::machine;
#[machine]
pub(crate) mod linear {
    use umpire_dsl::{Action, Rejection, State, reject};
    #[derive(State)]
    pub(crate) struct Model {
        pub count: u8,
        pub done: bool,
    }
    #[derive(Action, Copy)]
    pub(crate) enum Client {
        Returns,
        Updates,
    }
    impl Model {
        #[on(Client)]
        fn client(&mut self, action: Client) {
            match action {
                Client::Returns => {
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
                Client::Updates => {
                    if self.count == 3 {
                        reject!(Rejection::NotFound);
                    }
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.count += 1;
                    self.done = true;
                }
            }
        }
    }
}
