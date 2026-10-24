#[forbid(unsafe_code)]
#[machine]
pub mod counter {
    use umpire::dsl::{Action, Outcome, Rejection, State, check, reject, stay};
    #[derive(State)]
    #[__umpire_values]
    pub struct Model {
        #[starts(0..= 0)]
        pub value: u8,
    }
    #[derive(Action, Copy)]
    #[__umpire_values]
    pub enum User {
        Advance,
        Ping,
        Refuse,
    }
    impl Model {
        fn user(&mut self, action: User) {
            match action {
                User::Advance if self.value == 0 => {
                    stay!(__umpire_on_handler);
                    self.value = 1;
                }
                User::Ping => stay!(__umpire_on_handler),
                User::Refuse => {
                    stay!(__umpire_on_handler);
                    reject!(Rejection::FailedPrecondition);
                }
                User::Advance => {}
            }
        }
        #[allow(
            clippy::trivially_copy_pass_by_ref,
            reason = "postconditions borrow the emitted edge outcome"
        )]
        fn changed_or_stayed(&self, before: &Self, outcome: &Outcome) {
            check!(
                match outcome { Outcome::Accepted => self.value == 1 || self == before,
                Outcome::Rejected(rejection) => { * rejection ==
                Rejection::FailedPrecondition && self == before } }
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
    impl ::umpire_dsl::RegisteredActor<Model> for User {}
    impl ::umpire_dsl::ActorProjection<User> for Action {
        fn project(&self) -> ::core::option::Option<&User> {
            match self {
                Self::User(value) => Some(value),
            }
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
        fn outcome_catalog(
            &self,
        ) -> ::core::option::Option<
            ::std::vec::Vec<::umpire_dsl::OutcomeClass<Self::Outcome>>,
        > {
            Some(::umpire_dsl::Outcome::catalog())
        }
        fn fault_catalog(
            &self,
        ) -> ::core::option::Option<::std::vec::Vec<Self::Action>> {
            let mut actions = ::std::vec::Vec::new();
            actions
                .extend(
                    User::actions().into_iter().filter(User::fault).map(Action::User),
                );
            Some(actions)
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
                    let (outcome, stayed) = ::umpire_dsl::outcome::evaluate_staying(|| {
                        next.user(value)
                    });
                    let rejected = matches!(outcome, ::umpire_dsl::Outcome::Rejected(_));
                    if rejected && next != *state {
                        return ::core::result::Result::Err(
                            ::umpire_dsl::StepError::RejectedAfterMutation,
                        );
                    }
                    let mut steps = ::umpire_dsl::Steps::new();
                    if next != *state || rejected || stayed {
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
                ::umpire_dsl::StepClaim:: < Self > ::with_checks("changed_or_stayed", |
                _, _before, after, _, outcome, _ | after.changed_or_stayed(_before,
                outcome))
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
            ::std::vec!["changed_or_stayed"]
        }
    }
    impl ::umpire_dsl::Registered for Model {
        type World = World;
        fn world() -> World {
            World
        }
    }
    #[doc(hidden)]
    impl ::umpire_dsl::Values for World
    where
        for<'__umpire> u8: ::umpire_dsl::IntoValue,
    {
        fn values(&self) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Value)> {
            ::std::vec![]
        }
        #[cfg(not(target_arch = "wasm32"))]
        fn start_failure(
            &self,
            selected: &[(&str, i128)],
        ) -> ::core::option::Option<::umpire_dsl::StartFailure> {
            Model::__umpire_start_failure(selected)
        }
    }
    #[doc(hidden)]
    impl ::umpire_dsl::ActionValues<World> for Action
    where
        for<'__umpire> User: ::umpire_dsl::ActionValues<World>,
    {
        fn action_value(
            &self,
        ) -> (&'static str, ::std::vec::Vec<(&'static str, ::umpire_dsl::Value)>) {
            match self {
                Self::User(value) => {
                    ::umpire_dsl::ActionValues::<World>::action_value(value)
                }
            }
        }
        fn domains(
            world: &World,
        ) -> ::std::vec::Vec<
            (&'static str, &'static str, ::std::vec::Vec<::umpire_dsl::Value>),
        > {
            let mut domains = ::std::vec::Vec::new();
            domains.extend(<User as ::umpire_dsl::ActionValues<World>>::domains(world));
            domains
        }
    }
}
