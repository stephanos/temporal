impl Counter {
    #[on(Clock::Tick)]
    fn tick(&mut self) -> Option<()> {
        self.value += 1;
        None
    }
    #[postcondition(after = User::Reset)]
    fn reset_clears(&self, _before: &Self) -> bool {
        self.value == 0
    }
    #[on(User::Reset)]
    #[allow(
        clippy::unnecessary_wraps,
        reason = "transition handlers return Option even when always enabled"
    )]
    fn reset(&mut self) -> Option<()> {
        self.value = 0;
        Some(())
    }
    #[invariant]
    fn bounded(&self) -> bool {
        self.value <= 2
    }
    #[scope]
    fn never_reset() -> Scope<CounterAction> {
        Scope::without(&[CounterAction::User(User::Reset)])
    }
    #[progress(within = Self::never_reset())]
    fn reaches_two() -> Progress<CounterWorld> {
        Progress::<CounterWorld>::new("replaced_name")
            .whenever(|_, s| (s.value == 1).into())
            .eventually(|_, s| (s.value == 2).into())
    }
    #[postcondition]
    fn grows_at_most_one(&self, before: &Self) -> bool {
        self.value <= before.value + 1
    }
    #[on(User::Increment)]
    fn increment(&mut self) -> Option<()> {
        (self.value < 2)
            .then(|| {
                self.value += 1;
            })
    }
}
///The registered state machine's world.
struct CounterWorld;
///Actions in actor declaration order, spelled with their original Rust names.
#[derive(Clone, PartialEq, Eq, Hash)]
enum CounterAction {
    User(User),
    Clock(Clock),
}
impl ::umpire_dsl::Spell for CounterAction {
    fn spell(&self, out: &mut ::std::string::String) {
        match self {
            Self::User(value) => ::umpire_dsl::Spell::spell(value, out),
            Self::Clock(value) => ::umpire_dsl::Spell::spell(value, out),
        }
    }
}
impl ::umpire_dsl::Machine for CounterWorld {
    type State = Counter;
    type Action = CounterAction;
    type Outcome = ();
    type Fact = ();
    fn starts(&self) -> ::std::vec::Vec<Self::State> {
        Counter::starts()
    }
    fn actions(&self) -> ::std::vec::Vec<Self::Action> {
        let mut actions = ::std::vec::Vec::new();
        actions.extend(User::actions().into_iter().map(CounterAction::User));
        actions.extend(Clock::actions().into_iter().map(CounterAction::Clock));
        actions
    }
    fn internal(&self, action: &Self::Action) -> bool {
        match action {
            CounterAction::User(value) => value.internal(),
            CounterAction::Clock(value) => value.internal(),
        }
    }
    fn step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> ::umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact> {
        match action {
            CounterAction::User(value) => {
                match value {
                    User::Reset => {
                        let mut next = state.clone();
                        let enabled: ::core::option::Option<()> = next.reset();
                        let mut steps = ::umpire_dsl::Steps::new();
                        if enabled.is_some() {
                            steps.push(((), ::std::vec::Vec::new(), next));
                        }
                        steps
                    }
                    User::Increment => {
                        let mut next = state.clone();
                        let enabled: ::core::option::Option<()> = next.increment();
                        let mut steps = ::umpire_dsl::Steps::new();
                        if enabled.is_some() {
                            steps.push(((), ::std::vec::Vec::new(), next));
                        }
                        steps
                    }
                }
            }
            CounterAction::Clock(value) => {
                match value {
                    Clock::Tick => {
                        let mut next = state.clone();
                        let enabled: ::core::option::Option<()> = next.tick();
                        let mut steps = ::umpire_dsl::Steps::new();
                        if enabled.is_some() {
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
            ::umpire_dsl::Claim:: < Self > ::new("bounded", | _, state | state.bounded()
            .into())
        ]
    }
    fn postconditions(&self) -> ::std::vec::Vec<::umpire_dsl::StepClaim<Self>> {
        ::std::vec![
            ::umpire_dsl::StepClaim:: < Self > ::after("reset_clears",
            CounterAction::User(User::Reset), | _, before, after, _, _, _ | after
            .reset_clears(before).into()), ::umpire_dsl::StepClaim:: < Self >
            ::new("grows_at_most_one", | _, before, after, _, _, _ | after
            .grows_at_most_one(before).into())
        ]
    }
    fn progress(&self) -> ::std::vec::Vec<::umpire_dsl::Progress<Self>> {
        ::std::vec![
            { let mut claim = Counter::reaches_two(); claim.name = "reaches_two"; claim =
            claim.within(Counter::never_reset()); claim }
        ]
    }
    fn scopes(
        &self,
    ) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Scope<Self::Action>)> {
        ::std::vec![("never_reset", Counter::never_reset())]
    }
    fn claim_order(&self) -> ::std::vec::Vec<&'static str> {
        ::std::vec!["reset_clears", "bounded", "reaches_two", "grows_at_most_one"]
    }
}
