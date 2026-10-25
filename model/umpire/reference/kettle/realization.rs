use crate::{
    kettle::{self, Action, Heater, Model, User},
    wire::umpire::reference::kettle::v1::{
        DescribeResponse, FillResponse, HeatRequest, HeatResponse, HeatingAction,
        PollHeatingResponse, PourResponse, RespondHeatingRequest, RespondHeatingResponse, State,
    },
};
use umpire_runtime::{
    AnswerPlan, Evidence, EvidenceMatch, Exact, Observed, Projection, Realize, callback,
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

fn current(observed: State, state: &Model) -> Projection<Action, (), Exact<Model>> {
    if matches(observed, state) { Projection::Current } else { Projection::Candidates(vec![]) }
}

fn transition(observed: State) -> Projection<Action, (), Exact<Model>> {
    Projection::Reach(Exact(Model {
        temperature: observed.temperature,
        heating_on: observed.heating_on,
        beeping: observed.beeping,
    }))
}

fn heating(action: i32) -> Result<Action, Refusal> {
    match HeatingAction::try_from(action) {
        Ok(HeatingAction::Heat) => Ok(Heater::Heat.into()),
        Ok(HeatingAction::Failover) => Ok(Heater::Failover.into()),
        _ => Err(Refusal::malformed("heating action must name Heat or Failover")),
    }
}

impl umpire_runtime::capability::Identity for Realization {
    type Id = String;
    fn identify(&self, value: &Any) -> Result<String, Refusal> {
        let task: PollHeatingResponse = callback(value)?;
        if task.kettle_id.is_empty() {
            return Err(Refusal::malformed("kettle identity is missing"));
        }
        Ok(task.kettle_id)
    }
}

impl umpire_runtime::capability::Fill<Action> for Realization {
    type World = kettle::World;
    type Inputs = ();
    type Request = RespondHeatingRequest;
    fn fill(
        &self,
        action: &Action,
        _: &kettle::World,
        (): &(),
        identity: &String,
    ) -> Result<Self::Request, Refusal> {
        if identity.is_empty() {
            return Err(Refusal::malformed("kettle identity is missing"));
        }
        let action = if *action == Heater::Heat.into() {
            HeatingAction::Heat
        } else if *action == Heater::Failover.into() {
            HeatingAction::Failover
        } else {
            return Err(Refusal::unsupported("action has no heater request"));
        };
        Ok(RespondHeatingRequest { kettle_id: identity.clone(), action: action as i32 })
    }
}

impl umpire_runtime::capability::Snapshot<kettle::World> for Realization {
    type Target = Exact<Model>;

    fn project(
        &self,
        _world: &kettle::World,
        value: &Any,
        state: &Model,
    ) -> Result<Projection<Action, (), Exact<Model>>, Refusal> {
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
                transition(response)
            }
            "RespondHeating" => {
                let response = observation.response::<RespondHeatingResponse>()?;
                heating(response.action)?;
                if let Some(request) = observation.request::<RespondHeatingRequest>()?
                    && (request.kettle_id.is_empty() || request.action != response.action)
                {
                    return Err(Refusal::malformed("heating receipt does not match its request"));
                }
                transition(required(response.state)?)
            }
            _ => {
                return Err(Refusal::new(
                    RefusalCode::Unsupported,
                    "unsupported Kettle observation",
                ));
            }
        })
    }
}

impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for Realization {
    type World = kettle::World;
    type View = Observed;
    fn admit_evidence(&self, value: &Any) -> Result<Observed, Refusal> {
        Observed::new(value)
    }
    fn check_evidence(
        &self,
        observed: &Observed,
        path: &[&Evidence<kettle::World>],
    ) -> EvidenceMatch {
        if path.is_empty() {
            return EvidenceMatch::Continue { cursor: 0 };
        }
        let [edge] = path else { return EvidenceMatch::Reject };
        let expected = (|| match observed.method.as_str() {
            "/umpire.reference.kettle.v1.KettleService/Heat" => {
                let request = observed
                    .request::<HeatRequest>()?
                    .ok_or_else(|| Refusal::malformed("Heat requires its actual request"))?;
                Ok(if request.enabled {
                    User::PressButton.into()
                } else {
                    User::DepressButton.into()
                })
            }
            "/umpire.reference.kettle.v1.KettleService/RespondHeating" => {
                heating(observed.response::<RespondHeatingResponse>()?.action)
            }
            _ => Err(Refusal::unsupported("method does not authorize a Model action")),
        })();
        if expected.is_ok_and(|action| edge.action == action) && edge.choice == 0 {
            EvidenceMatch::Match { cursor: 1 }
        } else {
            EvidenceMatch::Reject
        }
    }
}

impl umpire_runtime::capability::ReplyPolicy<kettle::World> for Realization {
    fn answer(
        &self,
        world: &kettle::World,
        value: &Any,
        state: &Model,
    ) -> Result<AnswerPlan<Action>, Refusal> {
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
            reply(&umpire_runtime::capability::Fill::fill(
                self,
                &heating(task.action)?,
                world,
                &(),
                &umpire_runtime::capability::Identity::identify(self, value)?,
            )?),
        ))
    }
}

impl Realize<kettle::World> for Realization {}
