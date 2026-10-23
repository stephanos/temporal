#[forbid(unsafe_code)]
mod counter {
    use umpire_dsl::{Action, State, check, path};
    use std::sync::atomic::{AtomicUsize, Ordering};
    pub static PROGRESS_CALLS: AtomicUsize = AtomicUsize::new(0);
    #[derive(State)]
    #[__umpire_values]
    pub struct Counter {
        #[starts(0..= 1)]
        pub value: u8,
    }
    #[derive(Action, Copy)]
    #[__umpire_values]
    pub enum User {
        Increment,
        Reset,
        MutateRestore,
    }
    #[derive(Action, Copy)]
    #[internal]
    #[__umpire_values]
    pub enum Clock {
        Tick,
    }
    mod helpers {
        pub(super) struct Counter {
            pub(super) value: u8,
        }
        impl std::ops::AddAssign<u8> for Counter {
            fn add_assign(&mut self, increment: u8) {
                self.value += increment;
            }
        }
        const _: () = {
            struct __Probe<T: ?Sized>(::core::marker::PhantomData<T>);
            struct __MutableModelMethod;
            trait __Readonly {
                type __Result;
                fn __probe(self) -> Self::__Result;
            }
            impl __Readonly for &__Probe<super::Counter> {
                type __Result = __MutableModelMethod;
                fn __probe(self) -> Self::__Result {
                    __MutableModelMethod
                }
            }
            impl<T: ?Sized> __Readonly for &&__Probe<T> {
                type __Result = ();
                fn __probe(self) {}
            }
            const _: () = {
                #[diagnostic::on_unimplemented(
                    message = "add_assign takes &mut self; only #[on] and #[effect] methods change the state"
                )]
                trait __Allowed {}
                impl __Allowed for () {}
                fn __require(_: impl __Allowed) {}
                trait __Guard: std::ops::AddAssign<u8> {
                    fn __check(&self, _: ::core::marker::PhantomData<Counter>);
                }
                impl __Guard for Counter {
                    fn __check(&self, _: ::core::marker::PhantomData<Counter>) {
                        __require(
                            (&__Probe::<Self>(::core::marker::PhantomData)).__probe(),
                        );
                    }
                }
            };
        };
    }
    struct Helper<T> {
        value: T,
    }
    impl<T: std::ops::AddAssign<u8>> Helper<T> {
        fn advance(&mut self)
        where
            Self: Sized,
        {
            const _: () = {
                struct __Probe<T: ?Sized>(::core::marker::PhantomData<T>);
                struct __MutableModelMethod;
                trait __Readonly {
                    type __Result;
                    fn __probe(self) -> Self::__Result;
                }
                impl __Readonly for &__Probe<self::Counter> {
                    type __Result = __MutableModelMethod;
                    fn __probe(self) -> Self::__Result {
                        __MutableModelMethod
                    }
                }
                impl<T: ?Sized> __Readonly for &&__Probe<T> {
                    type __Result = ();
                    fn __probe(self) {}
                }
                const _: () = {
                    #[diagnostic::on_unimplemented(
                        message = "advance takes &mut self; only #[on] and #[effect] methods change the state"
                    )]
                    trait __Allowed {}
                    impl __Allowed for () {}
                    fn __require(_: impl __Allowed) {}
                    trait __Guard {
                        fn __check(&self, _: ::core::marker::PhantomData<Increment>);
                    }
                    impl __Guard for Increment {
                        fn __check(&self, _: ::core::marker::PhantomData<Increment>) {
                            __require(
                                (&__Probe::<Self>(::core::marker::PhantomData)).__probe(),
                            );
                        }
                    }
                };
            };
            const _: () = {
                struct __Probe<T: ?Sized>(::core::marker::PhantomData<T>);
                struct __MutableModelMethod;
                trait __Readonly {
                    type __Result;
                    fn __probe(self) -> Self::__Result;
                }
                impl __Readonly for &__Probe<self::Counter> {
                    type __Result = __MutableModelMethod;
                    fn __probe(self) -> Self::__Result {
                        __MutableModelMethod
                    }
                }
                impl<T: ?Sized> __Readonly for &&__Probe<T> {
                    type __Result = ();
                    fn __probe(self) {}
                }
                const _: () = {
                    #[diagnostic::on_unimplemented(
                        message = "apply takes &mut self; only #[on] and #[effect] methods change the state"
                    )]
                    trait __Allowed {}
                    impl __Allowed for () {}
                    fn __require(_: impl __Allowed) {}
                    trait __Guard: Advance
                    where
                        Self::Assoc: Copy,
                    {
                        fn __check(
                            &self,
                            _: ::core::marker::PhantomData<<Carrier as Resolve>::Target>,
                        );
                    }
                    impl __Guard for <Carrier as Resolve>::Target
                    where
                        Self::Assoc: Copy,
                    {
                        fn __check(
                            &self,
                            _: ::core::marker::PhantomData<<Carrier as Resolve>::Target>,
                        ) {
                            __require(
                                (&__Probe::<Self>(::core::marker::PhantomData)).__probe(),
                            );
                        }
                    }
                };
            };
            struct Increment(u8);
            impl Increment {
                fn advance(&mut self) {
                    self.0 += 1;
                }
            }
            struct Carrier;
            trait Resolve {
                type Target;
            }
            impl Resolve for Carrier {
                type Target = Increment;
            }
            trait Types {
                type Assoc;
            }
            impl Types for Increment {
                type Assoc = u8;
            }
            trait Advance: Types {
                fn apply(&mut self);
            }
            impl Advance for <Carrier as Resolve>::Target
            where
                Self::Assoc: Copy,
            {
                fn apply(&mut self) {
                    self.advance();
                }
            }
            let mut increment = Increment(0);
            increment.apply();
            self.value += increment.0;
        }
    }
    impl Counter {
        fn advance(&mut self, action: Clock) {
            match action {
                Clock::Tick => {
                    self.value += 1;
                }
            }
        }
        fn helper(&self) -> bool {
            self.value <= 2
        }
        pub fn dispatch(&mut self, action: User) {
            match action {
                User::Reset => {
                    self.value = 0;
                }
                User::Increment if self.value < 2 => {
                    self.increment();
                }
                User::MutateRestore if self.value < 2 => {
                    self.value += 1;
                    self.value -= 1;
                }
                _ => {}
            }
        }
        pub fn increment(&mut self) {
            let mut helper = Helper {
                value: helpers::Counter {
                    value: self.value,
                },
            };
            helper.advance();
            self.value = helper.value.value;
        }
        pub fn bounded(&self) {
            check!(self.helper());
        }
        pub fn reset_clears(&self) {
            check!(self.value == 0);
        }
        fn grows_at_most_one(&self, before: &Self) {
            check!(self.value <= before.value + 1);
        }
        fn never_reset() -> Assumption {
            without([User::Reset])
        }
        fn reaches_two(&self) -> LeadsTo {
            ::core::convert::From::from({
                PROGRESS_CALLS.fetch_add(1, Ordering::SeqCst);
                path![self.value == 1, ..= 2, self.value == 2]
                    .assuming(Self::never_reset())
            })
        }
        pub fn unregistered(&self) -> bool {
            self.helper()
        }
    }
    const _: () = {
        struct __Probe<T: ?Sized>(::core::marker::PhantomData<T>);
        struct __MutableModelMethod;
        trait __Readonly {
            type __Result;
            fn __probe(self) -> Self::__Result;
        }
        impl __Readonly for &__Probe<self::Counter> {
            type __Result = __MutableModelMethod;
            fn __probe(self) -> Self::__Result {
                __MutableModelMethod
            }
        }
        impl<T: ?Sized> __Readonly for &&__Probe<T> {
            type __Result = ();
            fn __probe(self) {}
        }
        const _: () = {
            #[diagnostic::on_unimplemented(
                message = "advance takes &mut self; only #[on] and #[effect] methods change the state"
            )]
            trait __Allowed {}
            impl __Allowed for () {}
            fn __require(_: impl __Allowed) {}
            trait __Guard<T: std::ops::AddAssign<u8>> {
                fn __check(&self, _: ::core::marker::PhantomData<Helper<T>>);
            }
            impl<T: std::ops::AddAssign<u8>> __Guard<T> for Helper<T> {
                fn __check(&self, _: ::core::marker::PhantomData<Helper<T>>) {
                    __require((&__Probe::<Self>(::core::marker::PhantomData)).__probe());
                }
            }
        };
    };
    ///The module's registered state machine.
    pub struct World;
    #[derive(Clone, Debug, PartialEq, Eq, Hash)]
    ///The registered actors' action carrier.
    pub enum Action {
        #[doc = stringify!(User)]
        User(User),
        #[doc = stringify!(Clock)]
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
    impl ::umpire_dsl::RegisteredActor<Counter> for User {}
    impl ::umpire_dsl::ActorProjection<User> for Action {
        fn project(&self) -> ::core::option::Option<&User> {
            match self {
                Self::User(value) => Some(value),
                _ => None,
            }
        }
    }
    impl ::umpire_dsl::RegisteredActor<Counter> for Clock {}
    impl ::umpire_dsl::ActorProjection<Clock> for Action {
        fn project(&self) -> ::core::option::Option<&Clock> {
            match self {
                Self::Clock(value) => Some(value),
                _ => None,
            }
        }
    }
    impl ::core::fmt::Display for Action {
        fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
            match *self {
                Self::User(ref value) => ::core::fmt::Display::fmt(value, out),
                Self::Clock(ref value) => ::core::fmt::Display::fmt(value, out),
            }
        }
    }
    use ::umpire_dsl::AssumptionExt as _;
    type Assumption = ::umpire_dsl::Assumption<Action>;
    fn without<A: ::core::convert::Into<Action>, const N: usize>(
        actions: [A; N],
    ) -> Assumption {
        Assumption::without(
            &actions
                .into_iter()
                .map(::core::convert::Into::into)
                .collect::<::std::vec::Vec<Action>>(),
        )
    }
    fn only<A: ::core::convert::Into<Action>, const N: usize>(
        actions: [A; N],
    ) -> Assumption {
        Assumption::only(
            &actions
                .into_iter()
                .map(::core::convert::Into::into)
                .collect::<::std::vec::Vec<Action>>(),
        )
    }
    fn fair<A: ::core::convert::Into<Action>, const N: usize>(
        actions: [A; N],
    ) -> Assumption {
        Assumption::fair(
            &actions
                .into_iter()
                .map(::core::convert::Into::into)
                .collect::<::std::vec::Vec<Action>>(),
        )
    }
    struct LeadsTo {
        trigger: ::umpire_dsl::Waypoint<Counter>,
        target: ::umpire_dsl::Waypoint<Counter>,
        assumption: Assumption,
        bound: Option<u32>,
    }
    #[allow(dead_code, reason = "a claim may be written as path! instead")]
    fn whenever(trigger: bool) -> LeadsTo {
        ::umpire_dsl::path![trigger, .., true].into()
    }
    #[allow(dead_code, reason = "a claim may be written as path! instead")]
    impl LeadsTo {
        fn eventually(mut self, target: bool) -> Self {
            self.target = ::umpire_dsl::Waypoint::Holds(target);
            self
        }
        fn within(mut self, bound: u32) -> Self {
            self.bound = Some(bound);
            self
        }
        fn assuming(mut self, assumption: Assumption) -> Self {
            self.assumption = assumption;
            self
        }
    }
    impl ::core::convert::From<::umpire_dsl::Path<Counter>> for LeadsTo {
        fn from(path: ::umpire_dsl::Path<Counter>) -> Self {
            let (trigger, gap, target) = path.claim();
            LeadsTo {
                trigger,
                target,
                assumption: Assumption::default(),
                bound: gap.bound(),
            }
        }
    }
    #[allow(dead_code, reason = "a path! claim may carry no assumption")]
    trait ClaimedPath {
        fn assuming(self, assumption: Assumption) -> LeadsTo;
    }
    impl ClaimedPath for ::umpire_dsl::Path<Counter> {
        fn assuming(self, assumption: Assumption) -> LeadsTo {
            LeadsTo::from(self).assuming(assumption)
        }
    }
    ///Builds the module's registered state machine.
    pub fn world() -> World {
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
            actions
                .extend(
                    Clock::actions().into_iter().filter(Clock::fault).map(Action::Clock),
                );
            Some(actions)
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
                Action::Clock(value) => {
                    let mut next = state.clone();
                    next.advance(value);
                    let mut steps = ::umpire_dsl::Steps::new();
                    if next != *state {
                        steps.push(((), ::std::vec::Vec::new(), next));
                    }
                    steps
                }
                Action::User(value) => {
                    let mut next = state.clone();
                    next.dispatch(value);
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
                ::umpire_dsl::Claim:: < Self > ::with_checks("bounded", | _, state |
                state.bounded(), | _, state | ::umpire_dsl::claim::evaluate(|| state
                .bounded()).verdict,)
            ]
        }
        fn postconditions(&self) -> ::std::vec::Vec<::umpire_dsl::StepClaim<Self>> {
            ::std::vec![
                ::umpire_dsl::StepClaim:: < Self > ::after_checks("reset_clears",
                User::Reset.into(), | _, _before, after, _, _, _ | after.reset_clears()),
                ::umpire_dsl::StepClaim:: < Self > ::with_checks("grows_at_most_one", |
                _, _before, after, _, _, _ | after.grows_at_most_one(_before))
            ]
        }
        fn progress(&self) -> ::std::vec::Vec<::umpire_dsl::Progress<Self>> {
            ::std::vec![
                ::umpire_dsl::Progress:: < Self > ::evaluated("reaches_two", | _, state |
                { let value = state.reaches_two(); ::umpire_dsl::ProgressEvaluation {
                trigger : value.trigger.judge(state).into(), target : value.target
                .judge(state).into(), assumption : value.assumption, bound : value.bound,
                } })
            ]
        }
        fn assumptions(
            &self,
        ) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Assumption<Self::Action>)> {
            ::std::vec![("never_reset", Counter::never_reset())]
        }
        fn claim_order(&self) -> ::std::vec::Vec<&'static str> {
            ::std::vec!["bounded", "reset_clears", "grows_at_most_one", "reaches_two"]
        }
    }
    impl ::umpire_dsl::Registered for Counter {
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
            Counter::__umpire_start_failure(selected)
        }
    }
    #[doc(hidden)]
    impl ::umpire_dsl::ActionValues<World> for Action
    where
        for<'__umpire> User: ::umpire_dsl::ActionValues<World>,
        for<'__umpire> Clock: ::umpire_dsl::ActionValues<World>,
    {
        fn action_value(
            &self,
        ) -> (&'static str, ::std::vec::Vec<(&'static str, ::umpire_dsl::Value)>) {
            match self {
                Self::User(value) => {
                    ::umpire_dsl::ActionValues::<World>::action_value(value)
                }
                Self::Clock(value) => {
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
            domains.extend(<Clock as ::umpire_dsl::ActionValues<World>>::domains(world));
            domains
        }
    }
}
