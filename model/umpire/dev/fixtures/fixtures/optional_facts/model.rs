//! Optional edge facts remain independent of enabledness and outcome.

use umpire::dsl::machine;

#[expect(clippy::match_same_arms, reason = "the later overlapping arm detects priority drift")]
#[expect(clippy::unused_self, reason = "effect helpers borrow the Model")]
#[machine]
pub mod facts {
    use umpire::dsl::{Action, Rejection, State, World, check, reject, stay};

    #[derive(World)]
    pub struct World {
        selector: bool,
    }

    #[derive(Debug)]
    pub enum Fact {
        A,
        B,
        C,
        D,
    }

    #[derive(Action, Copy)]
    pub enum Signal {
        A,
        B,
        C,
        D,
        Overlap,
        ChangedNone,
        UnchangedNone,
        UnchangedSome,
        Stay,
        HelperReject,
    }

    #[derive(Action, Copy)]
    pub enum Ordinary {
        Clear,
        Reject,
    }

    #[derive(State)]
    pub struct Model {
        pub value: u8,
    }

    impl Model {
        #[on(Signal)]
        fn signal(&mut self, action: Signal, selected: &World) -> Option<Fact> {
            match action {
                Signal::A if self.value == 0 => {
                    self.value = 1;
                    Some(Fact::A)
                }
                Signal::B if self.value == 0 => {
                    self.value = 2;
                    Some(Fact::B)
                }
                Signal::C if self.value == 0 => {
                    self.value = 3;
                    Some(Fact::C)
                }
                Signal::D if self.value == 0 => {
                    self.value = 4;
                    Some(Fact::D)
                }
                Signal::Overlap if self.value == 0 => {
                    self.value = 1;
                    if selected.selector { Some(Fact::B) } else { Some(Fact::A) }
                }
                Signal::Overlap if self.value == 0 => {
                    self.value = 4;
                    Some(Fact::D)
                }
                Signal::ChangedNone if self.value == 0 => {
                    self.value = 6;
                    None
                }
                Signal::UnchangedSome => Some(Fact::D),
                Signal::Stay => {
                    stay!();
                    Some(Fact::C)
                }
                Signal::HelperReject => {
                    self.reject_helper();
                    Some(Fact::B)
                }
                _ => None,
            }
        }

        #[on(Ordinary)]
        fn ordinary(&mut self, action: Ordinary) {
            match action {
                Ordinary::Clear if self.value > 0 && self.value < 5 => self.value = 5,
                Ordinary::Reject => reject!(Rejection::NotFound),
                Ordinary::Clear => {}
            }
        }

        #[effect]
        fn reject_helper(&mut self) {
            reject!(Rejection::Unavailable);
        }

        #[invariant]
        fn never_a(&self) {
            check!(self.value != 1);
        }
        #[invariant]
        fn never_b(&self) {
            check!(self.value != 2);
        }
        #[invariant]
        fn never_c(&self) {
            check!(self.value != 3);
        }
        #[invariant]
        fn never_d(&self) {
            check!(self.value != 4);
        }
        #[invariant]
        fn never_clear(&self) {
            check!(self.value != 5);
        }
    }
}
