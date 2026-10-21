#[forbid(unsafe_code)]
mod correlated {
    use umpire_dsl::{Action, State, World, check};
    #[derive(World)]
    #[__umpire_world]
    pub(super) struct World {
        first: u8,
        second: u8,
    }
    #[derive(State)]
    #[__umpire_computed_starts]
    #[__umpire_world]
    pub(super) struct Model {
        pub(super) left: u8,
        pub(super) right: u8,
    }
    #[derive(Action, Copy)]
    #[__umpire_world]
    pub(super) enum User {
        Swap,
    }
    impl Model {
        fn starts(w: &World) -> Vec<Self> {
            if w.first == 0 {
                return Vec::new();
            }
            vec![
                Self { left : w.first, right : w.second }, Self { left : w.second, right
                : w.first }, Self { left : w.first, right : w.second },
            ]
        }
        fn user(&mut self, _: User, w: &World) {
            if self.left == w.first {
                std::mem::swap(&mut self.left, &mut self.right);
            }
        }
        fn correlated(&self, w: &World) {
            check!(
                (self.left, self.right) == (w.first, w.second) || (self.left, self.right)
                == (w.second, w.first)
            );
        }
    }
    #[derive(Clone, Debug, PartialEq, Eq, Hash)]
    ///The registered actors' action carrier.
    pub(super) enum Action {
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
    impl ::umpire_dsl::Associated for Model {
        type World = World;
    }
    impl ::umpire_dsl::Machine for World {
        type State = Model;
        type Action = Action;
        type Outcome = ();
        type Fact = ();
        fn starts(&self) -> ::std::vec::Vec<Self::State> {
            Model::starts(self).into_iter().collect()
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
            Some(::std::vec![::umpire_dsl::OutcomeClass::new("()", | _ | true)])
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
            match action {
                Action::User(value) => {
                    let mut next = state.clone();
                    next.user(value, self);
                    let mut steps = ::umpire_dsl::Steps::new();
                    if next != *state {
                        steps.push(((), ::std::vec::Vec::new(), next));
                    }
                    steps
                }
            }
        }
        fn invariants(&self) -> ::std::vec::Vec<::umpire_dsl::Claim<Self>> {
            ::std::vec![
                ::umpire_dsl::Claim:: < Self > ::with_checks("correlated", |
                __umpire_world, state | state.correlated(__umpire_world), |
                __umpire_world, state | ::umpire_dsl::claim::evaluate(|| state
                .correlated(__umpire_world)).verdict,)
            ]
        }
        fn postconditions(&self) -> ::std::vec::Vec<::umpire_dsl::StepClaim<Self>> {
            ::std::vec![]
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
            ::std::vec!["correlated"]
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
