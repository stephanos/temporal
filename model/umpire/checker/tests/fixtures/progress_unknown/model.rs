use umpire_dsl::{Assumption, Hole, Machine, Progress, Scope, Steps, Verdict};

#[derive(Clone, PartialEq, Eq, Hash)]
pub struct State(pub u8);

impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str("State { 0: ")?;
        write!(out, "{:?}", self.0)?;
        out.write_str(" }")?;
        Ok(())
    }
}

#[derive(Clone, Copy, PartialEq, Eq, Hash)]
pub enum Action {
    Advance,
    Loop,
    Absent,
}

impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str(match self {
            Self::Advance => "Action::Advance",
            Self::Loop => "Action::Loop",
            Self::Absent => "Action::Absent",
        })?;
        Ok(())
    }
}

pub enum Mode {
    Unknown,
    IndependentFailure,
    EffectPanic,
    StartsCut,
    AdmissionCut,
    QueuedCut,
    ChoicesCut,
    ExternalCycleCut,
    InternalCycleCut,
    RefutedCut,
    UnknownScope,
}

pub struct Graph(pub Mode);

impl Machine for Graph {
    type State = State;
    type Action = Action;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<State> {
        match self.0 {
            Mode::StartsCut | Mode::Unknown | Mode::IndependentFailure => vec![State(0), State(1)],
            _ => vec![State(0)],
        }
    }

    fn actions(&self) -> Vec<Action> {
        vec![Action::Advance, Action::Loop]
    }

    fn internal(&self, action: &Action) -> bool {
        matches!(self.0, Mode::InternalCycleCut) && *action == Action::Advance
    }

    fn step(&self, state: &State, action: Action) -> Steps<State, (), ()> {
        assert!(!(matches!(self.0, Mode::EffectPanic) && action == Action::Advance), "broken progress effect");
        let mut steps = Steps::new();
        match self.0 {
            Mode::ExternalCycleCut | Mode::InternalCycleCut
                if state.0 == 0 && action == Action::Advance =>
            {
                steps.push(((), vec![], state.clone()));
                steps.push(((), vec![], State(1)));
            }
            Mode::Unknown | Mode::IndependentFailure | Mode::RefutedCut if state.0 == 1 && action == Action::Loop => {
                steps.push(((), vec![], state.clone()));
            }
            Mode::ChoicesCut if state.0 == 0 && action == Action::Advance => {
                steps.push(((), vec![], State(1)));
                steps.push(((), vec![], State(2)));
            }
            Mode::QueuedCut | Mode::RefutedCut if state.0 == 0 => {
                steps.push(((), vec![], State(if action == Action::Advance { 1 } else { 2 })));
            }
            Mode::RefutedCut if state.0 == 2 && action == Action::Advance => {
                steps.push(((), vec![], State(3)));
            }
            Mode::AdmissionCut if state.0 == 0 && action == Action::Advance => {
                steps.push(((), vec![], State(1)));
            }
            _ => {}
        }
        steps
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        if matches!(self.0, Mode::Unknown | Mode::IndependentFailure) {
            return vec![
                Progress::<Self>::new("trigger_hole")
                    .whenever(|_, state| {
                        if state.0 == 0 {
                            Verdict::Unknown(Hole { reason: "trigger unavailable" })
                        } else {
                            Verdict::Fails
                        }
                    })
                    .eventually(|_, _| Verdict::Fails),
                Progress::<Self>::new("target_hole")
                    .whenever(|_, state| (state.0 == 0).into())
                    .eventually(|_, state| {
                        if state.0 == 0 {
                            Verdict::Unknown(Hole { reason: "target unavailable" })
                        } else {
                            Verdict::Fails
                        }
                    }),
                Progress::<Self>::new("trigger_panic")
                    .whenever(|_, state| {
                        assert!(state.0 != 0, "broken progress trigger");
                        Verdict::Fails
                    })
                    .eventually(|_, _| Verdict::Fails),
                Progress::<Self>::new("target_panic")
                    .whenever(|_, state| (state.0 == 0).into())
                    .eventually(|_, state| {
                        assert!(state.0 != 0, "broken progress target");
                        Verdict::Fails
                    }),
                Progress::<Self>::new("independent_failure")
                    .whenever(|world, _| matches!(world.0, Mode::IndependentFailure).into())
                    .eventually(|_, state| {
                        if state.0 == 0 {
                            Verdict::Unknown(Hole { reason: "independent target unavailable" })
                        } else {
                            Verdict::Fails
                        }
                    }),
            ];
        }
        let claim = Progress::<Self>::new("reaches_target").eventually(|_, _| Verdict::Fails);
        vec![if matches!(self.0, Mode::UnknownScope) {
            claim.within(Scope::without(&[Action::Absent]))
        } else if matches!(self.0, Mode::InternalCycleCut) {
            claim.within(Assumption::fair(&[Action::Advance]))
        } else {
            claim
        }]
    }
}

umpire_testing::fixture!(Graph(Mode::Unknown), 8);
