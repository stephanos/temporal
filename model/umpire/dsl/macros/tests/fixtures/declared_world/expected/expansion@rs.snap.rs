#[forbid(unsafe_code)]
mod correlated {
    use umpire_dsl::{Action, State, World, check};
    #[derive(World)]
    pub(super) struct World {
        first: u8,
        second: u8,
    }
    #[derive(State)]
    #[__umpire_computed_starts]
    pub(super) struct Model {
        pub(super) left: u8,
        pub(super) right: u8,
    }
    #[derive(Action, Copy)]
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
    }
    impl Model {
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
    ///The registered actors' action carrier.
    pub(super) enum Action
    where
        User: ::umpire_dsl::Actor,
    {
        #[doc = stringify!(User)]
        User(User),
    }
    impl ::core::clone::Clone for Action
    where
        User: ::umpire_dsl::Actor,
    {
        fn clone(&self) -> Self {
            match self {
                Self::User(value) => Self::User(value.clone()),
            }
        }
    }
    impl ::core::cmp::PartialEq for Action
    where
        User: ::umpire_dsl::Actor,
    {
        fn eq(&self, other: &Self) -> bool {
            #[allow(unreachable_patterns)]
            match (self, other) {
                (Self::User(value), Self::User(other)) => value == other,
                _ => false,
            }
        }
    }
    impl ::core::cmp::Eq for Action
    where
        User: ::umpire_dsl::Actor,
    {}
    impl ::core::hash::Hash for Action
    where
        User: ::umpire_dsl::Actor,
    {
        fn hash<__UmpireHasher: ::core::hash::Hasher>(
            &self,
            hasher: &mut __UmpireHasher,
        ) {
            ::core::hash::Hash::hash(&::core::mem::discriminant(self), hasher);
            match self {
                Self::User(value) => ::core::hash::Hash::hash(value, hasher),
            }
        }
    }
    impl ::core::fmt::Debug for Action
    where
        User: ::umpire_dsl::Actor,
    {
        fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
            match self {
                Self::User(value) => {
                    out.debug_tuple(stringify!(User)).field(value).finish()
                }
            }
        }
    }
    impl ::core::fmt::Display for Action
    where
        User: ::umpire_dsl::Actor,
    {
        fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
            match self {
                Self::User(value) => ::core::fmt::Display::fmt(value, out),
            }
        }
    }
    impl ::core::convert::From<User> for Action
    where
        User: ::umpire_dsl::Actor,
    {
        fn from(value: User) -> Self {
            Self::User(value)
        }
    }
    impl ::umpire_dsl::RegisteredActor<Model> for User
    where
        u8: ::core::clone::Clone + ::core::cmp::Eq + ::core::hash::Hash
            + ::core::fmt::Debug,
        u8: ::core::clone::Clone + ::core::cmp::Eq + ::core::hash::Hash
            + ::core::fmt::Debug,
        User: ::umpire_dsl::Actor,
    {}
    impl ::umpire_dsl::ActorProjection<User> for Action
    where
        User: ::umpire_dsl::Actor,
    {
        fn project(&self) -> Option<&User> {
            match self {
                Self::User(value) => Some(value),
            }
        }
    }
    impl ::umpire_dsl::Machine for World
    where
        u8: ::core::clone::Clone + ::core::cmp::Eq + ::core::hash::Hash
            + ::core::fmt::Debug,
        u8: ::core::clone::Clone + ::core::cmp::Eq + ::core::hash::Hash
            + ::core::fmt::Debug,
        User: ::umpire_dsl::Actor,
    {
        type State = Model;
        type Action = Action;
        type Outcome = ();
        type Fact = ();
        fn starts(&self) -> ::std::vec::Vec<Self::State> {
            <Model>::starts(self).into_iter().collect()
        }
        fn actions(&self) -> ::std::vec::Vec<Self::Action> {
            let mut actions = ::std::vec::Vec::new();
            actions
                .extend(
                    <User as ::umpire_dsl::Actor>::actions()
                        .into_iter()
                        .map(Action::User),
                );
            actions
        }
        fn outcome_catalog(
            &self,
        ) -> Option<::std::vec::Vec<::umpire_dsl::OutcomeClass<Self::Outcome>>> {
            Some(::std::vec![::umpire_dsl::OutcomeClass::new("()", | _ | true)])
        }
        fn fault_catalog(&self) -> Option<::std::vec::Vec<Self::Action>> {
            let mut actions = ::std::vec::Vec::new();
            actions
                .extend(
                    <User as ::umpire_dsl::Actor>::actions()
                        .into_iter()
                        .filter(::umpire_dsl::Actor::fault)
                        .map(Action::User),
                );
            Some(actions)
        }
        fn internal(&self, action: &Self::Action) -> bool {
            match action {
                Action::User(value) => ::umpire_dsl::Actor::internal(value),
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
    impl ::umpire_dsl::Associated for Model
    where
        u8: ::core::clone::Clone + ::core::cmp::Eq + ::core::hash::Hash
            + ::core::fmt::Debug,
        u8: ::core::clone::Clone + ::core::cmp::Eq + ::core::hash::Hash
            + ::core::fmt::Debug,
        User: ::umpire_dsl::Actor,
    {
        type World = World;
    }
}
