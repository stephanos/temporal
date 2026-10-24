use umpire_dsl::{Claim, Machine, Progress, Scope, StepClaim, Steps};

const BOILING_TEMPERATURE: i32 = 100;
const COLDEST_TEMPERATURE: i32 = -40;

pub struct KettleWorld;

#[derive(Clone, PartialEq, Eq, Hash, Debug)]
pub struct Kettle {
    pub temperature: i32,
    pub heating_on: bool,
    pub beeping: bool,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug)]
pub enum Act {
    UserPressButton,
    UserDepressButton,
    HeaterHeat,
    HeaterFailover,
}

impl std::fmt::Display for Kettle {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("Kettle { temperature: ")?;
        write!(out, "{:?}", self.temperature)?;
        out.write_str(", heating_on: ")?;
        write!(out, "{:?}", self.heating_on)?;
        out.write_str(", beeping: ")?;
        write!(out, "{:?}", self.beeping)?;
        out.write_str(" }")?;
        Ok(())
    }
}

impl std::fmt::Display for Act {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str(match self {
            Self::UserPressButton => "Act::UserPressButton",
            Self::UserDepressButton => "Act::UserDepressButton",
            Self::HeaterHeat => "Act::HeaterHeat",
            Self::HeaterFailover => "Act::HeaterFailover",
        })?;
        Ok(())
    }
}

impl Machine for KettleWorld {
    type State = Kettle;
    type Action = Act;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Kettle> {
        (COLDEST_TEMPERATURE..=40)
            .map(|temperature| Kettle { temperature, heating_on: false, beeping: false })
            .collect()
    }

    fn actions(&self) -> Vec<Act> {
        vec![Act::UserPressButton, Act::UserDepressButton, Act::HeaterHeat, Act::HeaterFailover]
    }

    fn internal(&self, action: &Act) -> bool {
        matches!(action, Act::HeaterHeat | Act::HeaterFailover)
    }

    fn step(&self, state: &Kettle, action: Act) -> Steps<Kettle, (), ()> {
        let mut next = state.clone();
        match action {
            Act::UserPressButton if !state.heating_on => {
                next.heating_on = true;
                next.beeping = false;
            }
            Act::HeaterFailover if state.heating_on && state.temperature >= BOILING_TEMPERATURE => {
                next.heating_on = false;
                next.beeping = true;
            }
            Act::HeaterHeat if state.heating_on && state.temperature < BOILING_TEMPERATURE => {
                next.temperature += 1;
            }
            Act::UserDepressButton if state.heating_on => {
                next.heating_on = false;
                next.beeping = false;
            }
            _ => return Steps::new(),
        }
        let mut steps = Steps::new();
        steps.push(((), Vec::new(), next));
        steps
    }

    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::<Self>::new("beeps_only_when_off", |_, k| (!k.beeping || !k.heating_on).into())]
    }

    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        vec![
            StepClaim::<Self>::new("warms_only_when_on", |_, before, after, _, (), _| {
                (before.heating_on || after.temperature <= before.temperature).into()
            }),
            StepClaim::<Self>::after(
                "fails_over_at_boiling",
                Act::HeaterFailover,
                |_, _, after, _, (), _| {
                    (after.temperature == BOILING_TEMPERATURE && after.beeping).into()
                },
            ),
        ]
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        vec![
            Progress::<Self>::new("boils")
                .whenever(|_, k| k.heating_on.into())
                .eventually(|_, k| k.beeping.into())
                .within(Scope::without(&[Act::UserDepressButton])),
        ]
    }
}
