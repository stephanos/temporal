mod counter {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub struct Counter {
        #[starts(0..= 1)]
        pub value: u8,
    }
    #[derive(Action)]
    pub enum User {
        Increment,
        Reset,
        MutateRestore,
    }
    #[derive(Action)]
    #[internal]
    pub enum Clock {
        Tick,
    }
    impl Counter {
        fn tick(&mut self) {
            self.value += 1;
        }
        fn reset(&mut self) {
            self.value = 0;
        }
        pub fn increment(&mut self) {
            if self.value < 2 {
                self.value += 1;
            }
        }
        fn mutate_restore(&mut self) {
            if self.value < 2 {
                self.value += 1;
                self.value -= 1;
            }
        }
        fn helper(&self) -> bool {
            self.value <= 2
        }
    }
    impl Counter {
        fn bounded(&self) -> bool {
            self.helper()
        }
    }
    impl Counter {
        fn reset_clears(&self) -> bool {
            self.value == 0
        }
        fn grows_at_most_one(&self, before: &Self) -> bool {
            self.value <= before.value + 1
        }
    }
    impl Counter {
        fn never_reset() -> Scope {
            without([User::Reset])
        }
    }
    impl Counter {
        fn reaches_two(&self) -> Progress {
            whenever(self.value == 1).eventually(self.value == 2)
        }
    }
    impl Counter {
        pub fn unregistered(&self) -> bool {
            self.helper()
        }
    }
    struct World;
    #[derive(Clone, PartialEq, Eq, Hash)]
    enum Action {
        User(User),
        Clock(Clock),
    }
    impl ::core::convert::From<User> for Action {
        fn from(value: User) -> Self {
            Self::User(value)
        }
    }
    impl ::core::convert::From<Clock> for Action {
        fn from(value: Clock) -> Self {
            Self::Clock(value)
        }
    }
    impl ::umpire_dsl::Spell for Action {
        fn spell(&self, out: &mut ::std::string::String) {
            match self {
                Self::User(value) => ::umpire_dsl::Spell::spell(value, out),
                Self::Clock(value) => ::umpire_dsl::Spell::spell(value, out),
            }
        }
    }
    type Scope = ::umpire_dsl::Scope<Action>;
    fn without<A: ::core::convert::Into<Action>, const N: usize>(
        actions: [A; N],
    ) -> Scope {
        Scope {
            excluded: actions.into_iter().map(::core::convert::Into::into).collect(),
        }
    }
    struct Progress {
        trigger: bool,
        target: bool,
    }
    fn whenever(trigger: bool) -> Progress {
        Progress { trigger, target: true }
    }
    impl Progress {
        fn eventually(mut self, target: bool) -> Self {
            self.target = target;
            self
        }
    }
    ///Builds the module's registered state machine.
    pub fn world() -> impl ::umpire_dsl::Machine<
        State = Counter,
        Outcome = (),
        Fact = (),
        Action: ::core::convert::From<User> + ::core::convert::From<Clock>,
    > {
        World
    }
    impl ::umpire_dsl::Machine for World {
        type State = Counter;
        type Action = Action;
        type Outcome = ();
        type Fact = ();
        fn starts(&self) -> ::std::vec::Vec<Self::State> {
            Counter::starts()
        }
        fn actions(&self) -> ::std::vec::Vec<Self::Action> {
            let mut actions = ::std::vec::Vec::new();
            actions.extend(User::actions().into_iter().map(Action::User));
            actions.extend(Clock::actions().into_iter().map(Action::Clock));
            actions
        }
        fn internal(&self, action: &Self::Action) -> bool {
            match action {
                Action::User(value) => value.internal(),
                Action::Clock(value) => value.internal(),
            }
        }
        fn step(
            &self,
            state: &Self::State,
            action: Self::Action,
        ) -> ::umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact> {
            match action {
                Action::User(value) => {
                    match value {
                        User::Reset => {
                            let mut next = state.clone();
                            next.reset();
                            let mut steps = ::umpire_dsl::Steps::new();
                            if next != *state {
                                steps.push(((), ::std::vec::Vec::new(), next));
                            }
                            steps
                        }
                        User::Increment => {
                            let mut next = state.clone();
                            next.increment();
                            let mut steps = ::umpire_dsl::Steps::new();
                            if next != *state {
                                steps.push(((), ::std::vec::Vec::new(), next));
                            }
                            steps
                        }
                        User::MutateRestore => {
                            let mut next = state.clone();
                            next.mutate_restore();
                            let mut steps = ::umpire_dsl::Steps::new();
                            if next != *state {
                                steps.push(((), ::std::vec::Vec::new(), next));
                            }
                            steps
                        }
                    }
                }
                Action::Clock(value) => {
                    match value {
                        Clock::Tick => {
                            let mut next = state.clone();
                            next.tick();
                            let mut steps = ::umpire_dsl::Steps::new();
                            if next != *state {
                                steps.push(((), ::std::vec::Vec::new(), next));
                            }
                            steps
                        }
                    }
                }
            }
        }
        fn invariants(&self) -> ::std::vec::Vec<::umpire_dsl::Claim<Self>> {
            ::std::vec![
                ::umpire_dsl::Claim:: < Self > ::new("bounded", | _, state | state
                .bounded().into())
            ]
        }
        fn postconditions(&self) -> ::std::vec::Vec<::umpire_dsl::StepClaim<Self>> {
            ::std::vec![
                ::umpire_dsl::StepClaim:: < Self > ::after("reset_clears", User::Reset
                .into(), | _, _before, after, _, _, _ | after.reset_clears().into()),
                ::umpire_dsl::StepClaim:: < Self > ::new("grows_at_most_one", | _,
                _before, after, _, _, _ | after.grows_at_most_one(_before).into())
            ]
        }
        fn progress(&self) -> ::std::vec::Vec<::umpire_dsl::Progress<Self>> {
            ::std::vec![
                ::umpire_dsl::Progress:: < Self > ::new("reaches_two").whenever(| _,
                state | state.reaches_two().trigger.into()).eventually(| _, state | state
                .reaches_two().target.into()).within(Counter::never_reset())
            ]
        }
        fn scopes(
            &self,
        ) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Scope<Self::Action>)> {
            ::std::vec![("never_reset", Counter::never_reset())]
        }
        fn claim_order(&self) -> ::std::vec::Vec<&'static str> {
            ::std::vec!["bounded", "reset_clears", "grows_at_most_one", "reaches_two"]
        }
    }
}
