use umpire_dsl::{Machine, Steps};
use umpire_runtime::{
    AnswerPlan, Candidate, Projection, Realize, Runtime, Transition, google::protobuf::Any,
    protocol,
};

pub(crate) const TYPE: &str = "type.googleapis.com/umpire.control.Value";

#[derive(Clone, PartialEq, Eq, Hash)]
pub(crate) struct State(pub(crate) u8);

#[derive(Clone, PartialEq, Eq, Hash)]
pub(crate) enum Action {
    Deliver,
    Respond,
    Finish,
}

#[derive(Debug)]
pub(crate) struct Outcome(pub(crate) u8);

impl PartialEq for Outcome {
    fn eq(&self, other: &Self) -> bool {
        assert_ne!(self.0, 15, "final projection failed");
        self.0 == other.0
    }
}
#[derive(Debug)]
pub(crate) struct Fact;

impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "{:?}", self.0)?;
        Ok(())
    }
}

impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        out.write_str(match self {
            Self::Deliver => "Deliver",
            Self::Respond => "Respond",
            Self::Finish => "Finish",
        })?;
        Ok(())
    }
}

pub(crate) struct World {
    pub(crate) starts: usize,
}

impl Machine for World {
    type State = State;
    type Action = Action;
    type Outcome = Outcome;
    type Fact = Fact;

    fn starts(&self) -> Vec<State> {
        (0..self.starts).map(|_| State(0)).collect()
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::Deliver, Action::Respond, Action::Finish]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &State, action: Action) -> Steps<State, Outcome, Fact> {
        let next = match (state.0, action) {
            (0, Action::Deliver) => 1,
            (1, Action::Respond) => 2,
            (2, Action::Finish) => 3,
            _ => return Steps::new(),
        };
        let mut steps = Steps::new();
        steps.push((Outcome(next), vec![Fact], State(next)));
        steps
    }
}

pub(crate) struct Realization;

pub(crate) fn payload(value: &Any) -> Result<u8, protocol::Refusal> {
    if value.type_url != TYPE {
        return Err(protocol::Refusal {
            code: protocol::RefusalCode::Unsupported as i32,
            message: "unknown type".into(),
        });
    }
    match value.value.as_slice() {
        [value] => Ok(*value),
        _ => Err(protocol::Refusal {
            code: protocol::RefusalCode::MalformedRequest as i32,
            message: "expected one byte".into(),
        }),
    }
}

pub(crate) fn transition(action: Action) -> Transition<Action> {
    Transition { action, choice: 0 }
}

impl Realize<World> for Realization {
    fn propose(
        &self,
        state: &State,
        _seed: u64,
    ) -> Result<umpire_runtime::Proposal<Action>, protocol::Refusal> {
        let action = match state.0 {
            0 => Action::Deliver,
            1 => Action::Respond,
            2 => Action::Finish,
            _ => {
                return Err(protocol::Refusal::new(
                    protocol::RefusalCode::Quiescent,
                    "no external action",
                ));
            }
        };
        Ok(umpire_runtime::Proposal::Action { action, request: any(state.0 + 1) })
    }
    fn matches(&self, request: &Any, observation: &Any) -> Result<(), protocol::Refusal> {
        if payload(request)? == payload(observation)? {
            Ok(())
        } else {
            Err(protocol::Refusal::new(protocol::RefusalCode::State, "crossed observation"))
        }
    }

    fn observe(
        &self,
        observation: &Any,
        state: &State,
    ) -> Result<Projection<Action, Outcome>, protocol::Refusal> {
        let value = payload(observation)?;
        Ok(match value {
            value if value == state.0 => Projection::Current,
            1 => Projection::step(Action::Deliver),
            11 => Projection::Candidates(vec![Candidate {
                path: vec![transition(Action::Deliver)],
                outcome: Some(Outcome(2)),
            }]),
            15 => Projection::Candidates(vec![Candidate {
                path: vec![transition(Action::Deliver)],
                outcome: Some(Outcome(15)),
            }]),
            2 => Projection::step(Action::Respond),
            3 => Projection::step(Action::Finish),
            8 | 13 => Projection::steps([Vec::new()]),
            9 => Projection::steps([vec![Transition { action: Action::Deliver, choice: 1 }]]),
            10 => Projection::Candidates(vec![Candidate {
                path: vec![transition(Action::Deliver), transition(Action::Respond)],
                outcome: Some(Outcome(2)),
            }]),
            14 => panic!("realization failed"),
            _ => Projection::Candidates(Vec::new()),
        })
    }

    fn answer(&self, callback: &Any, _: &State) -> Result<AnswerPlan<Action>, protocol::Refusal> {
        let value = payload(callback)?;
        let response_action = if value == 5 { Action::Finish } else { Action::Respond };
        let mut delivery = transition(Action::Deliver);
        if value == 8 {
            delivery.choice = 1;
        }
        let reply = match value {
            6 => Any { type_url: TYPE.into(), value: vec![0; 1 << 20] },
            7 => Any { type_url: String::new(), value: vec![42] },
            _ => any(42),
        };
        Ok(AnswerPlan { delivery: vec![delivery], response_action, reply })
    }
}

pub(crate) fn any(value: u8) -> Any {
    Any { type_url: TYPE.into(), value: vec![value] }
}
pub(crate) fn runtime() -> Runtime<World, Realization> {
    Runtime::new(World { starts: 1 }, Realization)
}
