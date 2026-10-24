macro_rules! claim {
    ($name:ident, $($pattern:tt)*) => {
        #[umpire_dsl::machine]
        mod $name {
            use umpire_dsl::{Action, State, path};
            #[derive(State)]
            pub struct Model { pub done: bool }
            #[derive(Action)]
            pub enum User { Poll }
            impl Model {
                #[on(User)]
                fn user(&mut self, _: User) { self.done = true; }
                #[progress]
                fn completes(&self) -> LeadsTo { path![$($pattern)*] }
            }
        }
    };
}
claim!(action, true, .., User::Poll);
claim!(inputs, true, .., User::Poll { amount: _, .. });
claim!(wildcard, true, .., User::*);
claim!(outcome, true, .., User::Poll => Outcome::Accepted);
claim!(alternatives, true, .., (true | false));
claim!(repetition, true, .., [true; 2]);
claim!(permutations, true, .., unordered![true, false]);
claim!(exclusion, true, ..!(User::Poll), false);
fn main() {}
