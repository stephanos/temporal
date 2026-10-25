use crate::{
    kettle::{self, Action, Heater, Model, User},
    wire::umpire::reference::kettle::v1::{
        DescribeResponse, FillResponse, HeatRequest, HeatResponse, HeatingAction,
        PollHeatingResponse, PourResponse, RespondHeatingRequest, RespondHeatingResponse, State,
    },
};
use umpire_dsl::Machine;
use umpire_runtime::{
    AnswerPlan, Observed, Projection, Realize, callback,
    prost_types::Any,
    protocol::{Refusal, RefusalCode},
    reply,
};

/// Strict evidence realization for one Kettle instance.
pub struct Realization;

fn matches(observed: State, state: &Model) -> bool {
    observed.temperature == state.temperature
        && observed.heating_on == state.heating_on
        && observed.beeping == state.beeping
}

fn required(state: Option<State>) -> Result<State, Refusal> {
    state.ok_or_else(|| Refusal::malformed("Kettle state is missing"))
}

fn current(observed: State, state: &Model) -> Projection<Action, ()> {
    if matches(observed, state) { Projection::Current } else { Projection::Candidates(vec![]) }
}

fn transition(observed: State, state: &Model, action: Action) -> Projection<Action, ()> {
    if kettle::world()
        .step(state, action.clone())
        .into_iter()
        .any(|((), _, next)| matches(observed, &next))
    {
        Projection::step(action)
    } else {
        Projection::Candidates(vec![])
    }
}

fn heating(action: i32) -> Result<Action, Refusal> {
    match HeatingAction::try_from(action) {
        Ok(HeatingAction::Heat) => Ok(Heater::Heat.into()),
        Ok(HeatingAction::Failover) => Ok(Heater::Failover.into()),
        _ => Err(Refusal::malformed("heating action must name Heat or Failover")),
    }
}

impl Realize<kettle::World> for Realization {
    fn observe(&self, value: &Any, state: &Model) -> Result<Projection<Action, ()>, Refusal> {
        let observation = Observed::new(value)?;
        let method = observation
            .method
            .strip_prefix("/umpire.reference.kettle.v1.KettleService/")
            .ok_or_else(|| Refusal::new(RefusalCode::Unsupported, "unknown Kettle service"))?;
        Ok(match method {
            "Fill" => current(required(observation.response::<FillResponse>()?.state)?, state),
            "Pour" => current(required(observation.response::<PourResponse>()?.state)?, state),
            "Describe" => {
                current(required(observation.response::<DescribeResponse>()?.state)?, state)
            }
            "Heat" => {
                let request = observation
                    .request::<HeatRequest>()?
                    .ok_or_else(|| Refusal::malformed("Heat requires its actual request"))?;
                if request.kettle_id.is_empty() {
                    return Err(Refusal::malformed("kettle identity is missing"));
                }
                let response = required(observation.response::<HeatResponse>()?.state)?;
                transition(
                    response,
                    state,
                    if request.enabled {
                        User::PressButton.into()
                    } else {
                        User::DepressButton.into()
                    },
                )
            }
            "RespondHeating" => {
                let response = observation.response::<RespondHeatingResponse>()?;
                let action = heating(response.action)?;
                if let Some(request) = observation.request::<RespondHeatingRequest>()?
                    && (request.kettle_id.is_empty() || request.action != response.action)
                {
                    return Err(Refusal::malformed("heating receipt does not match its request"));
                }
                transition(required(response.state)?, state, action)
            }
            _ => {
                return Err(Refusal::new(
                    RefusalCode::Unsupported,
                    "unsupported Kettle observation",
                ));
            }
        })
    }

    fn answer(&self, value: &Any, state: &Model) -> Result<AnswerPlan<Action>, Refusal> {
        let task: PollHeatingResponse = callback(value)?;
        let observed = required(task.state)?;
        if task.kettle_id.is_empty() || !matches(observed, state) || !state.heating_on {
            return Err(Refusal::new(
                RefusalCode::State,
                "heater poll does not match the current Kettle",
            ));
        }
        let expected =
            if state.temperature < 100 { HeatingAction::Heat } else { HeatingAction::Failover };
        if task.action != expected as i32 {
            return Err(Refusal::malformed("heater poll action does not match the current Kettle"));
        }
        Ok(AnswerPlan::new(
            vec![],
            heating(task.action)?,
            reply(&RespondHeatingRequest { kettle_id: task.kettle_id, action: task.action }),
        ))
    }
}
