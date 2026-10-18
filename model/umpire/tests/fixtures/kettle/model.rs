//! Kettle authored with the attribute surface.

pub(crate) use model::run;

pub(crate) fn cut_run() -> umpire_testing::RunResult {
    model::cut::run()
}

pub(crate) fn no_heat_run() -> umpire_testing::RunResult {
    model::no_heat::run()
}

mod model {
    use umpire_dsl::{
        Action, Claim, Progress, Scope, State, invariant, machine, on, postcondition, progress, scope,
    };

    const BOILING_TEMPERATURE: i32 = 100;
    const COLDEST_TEMPERATURE: i32 = -40;

    #[derive(State)]
    struct Kettle {
        #[starts(COLDEST_TEMPERATURE..=40)]
        temperature: i32,
        heating_on: bool,
        beeping: bool,
    }

    #[derive(Action)]
    enum User {
        PressButton,
        DepressButton,
    }

    #[derive(Action)]
    enum Heater {
        #[internal]
        Heat,
        #[internal]
        Failover,
    }

    #[machine(actions = (User, Heater))]
    impl Kettle {
        #[on(User::PressButton)]
        fn press_button(&mut self) -> Option<()> {
            (!self.heating_on).then(|| {
                self.heating_on = true;
                self.beeping = false;
            })
        }

        #[on(Heater::Failover)]
        fn failover(&mut self) -> Option<()> {
            (self.heating_on && self.temperature >= BOILING_TEMPERATURE).then(|| {
                self.heating_on = false;
                self.beeping = true;
            })
        }

        #[on(Heater::Heat)]
        fn heat(&mut self) -> Option<()> {
            (self.heating_on && self.temperature < BOILING_TEMPERATURE).then(|| {
                self.temperature += 1;
            })
        }

        #[on(User::DepressButton)]
        fn depress_button(&mut self) -> Option<()> {
            self.heating_on.then(|| {
                self.heating_on = false;
                self.beeping = false;
            })
        }

        #[invariant]
        fn beeps_only_when_off(&self) -> bool {
            !self.beeping || !self.heating_on
        }

        #[postcondition]
        fn warms_only_when_on(&self, before: &Self) -> bool {
            before.heating_on || self.temperature <= before.temperature
        }

        #[postcondition(after = Heater::Failover)]
        fn fails_over_at_boiling(&self, _before: &Self) -> bool {
            self.temperature == BOILING_TEMPERATURE && self.beeping
        }

        #[scope]
        fn never_switched_off() -> Scope<KettleAction> {
            Scope::without(&[KettleAction::User(User::DepressButton)])
        }

        #[progress(within = Self::never_switched_off())]
        fn boils() -> Progress<KettleWorld> {
            Progress::<KettleWorld>::new("boils")
                .whenever(|_, k| k.heating_on.into())
                .eventually(|_, k| k.beeping.into())
        }
    }

    umpire_testing::fixture!(KettleWorld, 1 << 20, Claim::<KettleWorld>::new("never_beeps", |_, state| (!state.beeping).into()));

    pub(super) mod cut {
        use super::KettleWorld;

        umpire_testing::fixture!(KettleWorld, 82);
    }

    pub(super) mod no_heat {
        use super::{Heater, Kettle, KettleAction, KettleWorld, User};
        use umpire_dsl::{Machine, Progress, Scope, Steps};

        struct NoHeat;

        impl Machine for NoHeat {
            type State = Kettle;
            type Action = KettleAction;
            type Outcome = ();
            type Fact = ();

            fn starts(&self) -> Vec<Kettle> {
                vec![Kettle { temperature: 20, heating_on: false, beeping: false }]
            }

            fn actions(&self) -> Vec<KettleAction> {
                KettleWorld.actions()
            }

            fn internal(&self, action: &KettleAction) -> bool {
                KettleWorld.internal(action)
            }

            fn step(&self, state: &Kettle, action: KettleAction) -> Steps<Kettle, (), ()> {
                if action == KettleAction::Heater(Heater::Heat) {
                    Steps::new()
                } else {
                    KettleWorld.step(state, action)
                }
            }

            fn progress(&self) -> Vec<Progress<Self>> {
                vec![
                    Progress::<Self>::new("boils")
                        .whenever(|_, state| state.heating_on.into())
                        .eventually(|_, state| state.beeping.into())
                        .within(Scope::without(&[KettleAction::User(User::DepressButton)])),
                ]
            }
        }

        umpire_testing::fixture!(NoHeat, 16);
    }
}
