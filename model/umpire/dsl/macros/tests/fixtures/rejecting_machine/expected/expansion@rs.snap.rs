#[forbid(unsafe_code)]
pub mod counter {
    use umpire_dsl::{Action, Outcome, Rejection, State, check, reject};
    #[derive(State)]
    pub struct Model {
        #[starts(0..= 0)]
        pub value: u8,
    }
    #[derive(Action, Copy)]
    pub enum User {
        Advance,
        Disabled,
        Reject,
        Helper,
    }
    fn reject_helper() {
        umpire_dsl::reject!(Rejection::Unavailable);
    }
    impl Model {
        fn user(&mut self, action: User) {
            match action {
                User::Advance if self.value == 0 => self.value = 1,
                User::Reject => reject!(Rejection::NotFound),
                User::Helper => reject_helper(),
                _ => {}
            }
        }
        #[allow(
            clippy::trivially_copy_pass_by_ref,
            reason = "postconditions borrow the emitted edge outcome"
        )]
        fn correct_outcome(&self, before: &Self, outcome: &Outcome) {
            check!(
                match outcome { Outcome::Accepted => self.value == 1 && before.value ==
                0, Outcome::Rejected(Rejection::NotFound | Rejection::Unavailable) =>
                self == before, Outcome::Rejected(_) => false, }
            );
        }
        #[allow(
            clippy::trivially_copy_pass_by_ref,
            reason = "postconditions borrow the emitted edge outcome"
        )]
        fn rejected_not_found(&self, before: &Self, outcome: &Outcome) {
            check!(
                self == before && * outcome == Outcome::Rejected(Rejection::NotFound)
            );
        }
    }
    ///The module's registered state machine.
    pub struct World;
    #[derive(Clone, Debug, PartialEq, Eq, Hash)]
    ///The registered actors' action carrier.
    pub enum Action {
        #[doc = stringify!(User)]
        User(User),
    }
    impl ::core::convert::From<User> for Action {
        fn from(value: User) -> Self {
            Self::User(value)
        }
    }
    impl ::core::fmt::Display for Action {
        fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
            match *self {
                Self::User(ref value) => ::core::fmt::Display::fmt(value, out),
            }
        }
    }
    ///Builds the module's registered state machine.
    pub fn world() -> World {
        World
    }
    impl ::umpire_dsl::Machine for World {
        type State = Model;
        type Action = Action;
        type Outcome = ::umpire_dsl::Outcome;
        type Fact = ();
        fn starts(&self) -> ::std::vec::Vec<Self::State> {
            Model::starts()
        }
        fn actions(&self) -> ::std::vec::Vec<Self::Action> {
            let mut actions = ::std::vec::Vec::new();
            actions.extend(User::actions().into_iter().map(Action::User));
            actions
        }
        fn internal(&self, action: &Self::Action) -> bool {
            match action {
                Action::User(value) => value.internal(),
            }
        }
        fn step(
            &self,
            state: &Self::State,
            action: Self::Action,
        ) -> ::umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact> {
            match <Self as ::umpire_dsl::Machine>::checked_step(self, state, action) {
                ::core::result::Result::Ok(steps) => steps,
                ::core::result::Result::Err(error) => {
                    panic!("unchecked Model step: {}", error)
                }
            }
        }
        fn checked_step(
            &self,
            state: &Self::State,
            action: Self::Action,
        ) -> ::core::result::Result<
            ::umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact>,
            ::umpire_dsl::StepError,
        > {
            match action {
                Action::User(value) => {
                    let mut next = state.clone();
                    let outcome = ::umpire_dsl::outcome::evaluate(|| next.user(value));
                    if matches!(outcome, ::umpire_dsl::Outcome::Rejected(_))
                        && next != *state
                    {
                        return ::core::result::Result::Err(
                            ::umpire_dsl::StepError::RejectedAfterMutation,
                        );
                    }
                    let mut steps = ::umpire_dsl::Steps::new();
                    if next != *state
                        || matches!(outcome, ::umpire_dsl::Outcome::Rejected(_))
                    {
                        steps.push((outcome, ::std::vec::Vec::new(), next));
                    }
                    ::core::result::Result::Ok(steps)
                }
            }
        }
        fn invariants(&self) -> ::std::vec::Vec<::umpire_dsl::Claim<Self>> {
            ::std::vec![]
        }
        fn postconditions(&self) -> ::std::vec::Vec<::umpire_dsl::StepClaim<Self>> {
            ::std::vec![
                ::umpire_dsl::StepClaim:: < Self > ::with_checks("correct_outcome", | _,
                _before, after, _, outcome, _ | after.correct_outcome(_before, outcome)),
                ::umpire_dsl::StepClaim:: < Self > ::after_checks("rejected_not_found",
                User::Reject.into(), | _, _before, after, _, outcome, _ | after
                .rejected_not_found(_before, outcome))
            ]
        }
        fn progress(&self) -> ::std::vec::Vec<::umpire_dsl::Progress<Self>> {
            ::std::vec![]
        }
        fn assumptions(
            &self,
        ) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Assumption<Self::Action>)> {
            ::std::vec![]
        }
        fn claim_order(&self) -> ::std::vec::Vec<&'static str> {
            ::std::vec!["correct_outcome", "rejected_not_found"]
        }
    }
    impl ::umpire_dsl::Registered for Model {
        type World = World;
        fn world() -> World {
            World
        }
    }
}
