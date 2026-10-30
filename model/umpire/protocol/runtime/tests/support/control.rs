use super::support::{Action, Outcome, Realization, State, World, any, payload};
use umpire_runtime::{AnswerPlan, Projection, Realize, Runtime, prost_types::Any, protocol};

pub(crate) struct ControlRealization {
    pub(crate) bootstrap: bool,
}
impl umpire_runtime::capability::Identity for ControlRealization {
    type Id = ();

    fn identify(&self, _: &Any) -> Result<(), protocol::Refusal> {
        Ok(())
    }
}

impl umpire_runtime::capability::Fill<<World as umpire_dsl::Machine>::Action>
    for ControlRealization
{
    type World = World;
    type Inputs = ();
    type Request = umpire_runtime::capability::NoRequest;
}

impl umpire_runtime::capability::Snapshot<World> for ControlRealization {
    type Target = umpire_runtime::NoTarget;

    fn project(
        &self,
        world: &World,
        value: &Any,
        state: &State,
    ) -> Result<Projection<Action, Outcome, umpire_runtime::NoTarget>, protocol::Refusal> {
        if payload(value)? == 20 {
            return Ok(Projection::Current);
        }
        Realize::observe(&Realization, world, value, state)
    }
}

impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for ControlRealization {
    type World = World;
    type View = umpire_runtime::capability::NoEvidence;
}

impl umpire_runtime::capability::ReplyPolicy<World> for ControlRealization {
    fn needs_bootstrap(&self) -> bool {
        self.bootstrap
    }

    fn bootstrap(
        &self,
        _world: &World,
        _: &State,
        _: u64,
    ) -> Result<umpire_runtime::Proposal<Action>, protocol::Refusal> {
        Ok(umpire_runtime::Proposal::Bootstrap { request: any(0) })
    }

    fn propose(
        &self,
        world: &World,
        state: &State,
        seed: u64,
    ) -> Result<umpire_runtime::Proposal<Action>, protocol::Refusal> {
        Realize::propose(&Realization, world, state, seed)
    }

    fn answer(
        &self,
        world: &World,
        value: &Any,
        state: &State,
    ) -> Result<AnswerPlan<Action>, protocol::Refusal> {
        Realize::answer(&Realization, world, value, state)
    }

    fn matches(&self, request: &Any, value: &Any) -> Result<(), protocol::Refusal> {
        let requested = payload(request)?;
        let observed = payload(value)?;
        if requested == observed
            || (requested != 0 && observed == 20)
            || (requested == 1 && observed == 2)
        {
            Ok(())
        } else {
            Err(protocol::Refusal::new(protocol::RefusalCode::State, "crossed observation"))
        }
    }

    fn continue_polling(
        &self,
        value: &Any,
        _: &State,
        _: Option<&Action>,
    ) -> Result<bool, protocol::Refusal> {
        Ok(payload(value)? == 20)
    }
}

impl Realize<World> for ControlRealization {}

pub(crate) struct Control {
    runtime: Runtime<World, ControlRealization>,
    frontier: Option<Runtime<super::support::frontier::World, super::support::frontier::Binding>>,
}
pub(crate) enum ControlTransaction {
    Model(umpire_runtime::Transaction<World>),
    Frontier(umpire_runtime::Transaction<super::support::frontier::World>),
    ResetFrontier(Box<Runtime<super::support::frontier::World, super::support::frontier::Binding>>),
    Reset(Box<Runtime<World, ControlRealization>>),
}
impl umpire_runtime::Dispatch for Control {
    type Transaction = ControlTransaction;
    fn prepare(
        &self,
        method: u32,
        bytes: &[u8],
    ) -> (protocol::Response, Option<Self::Transaction>) {
        use prost::Message;
        if method == 3 {
            let configured =
                protocol::ResetRequest::decode(bytes).map(|request| request.configuration);
            match configured {
                Ok(Some(configuration))
                    if configuration == any(super::support::frontier::RENEW)
                        || configuration == any(54)
                        || configuration == any(55) =>
                {
                    let fault = if configuration == any(54) {
                        super::support::frontier::Fault::ExternalProposal
                    } else if configuration == any(55) {
                        super::support::frontier::Fault::QuiescentAction
                    } else {
                        super::support::frontier::Fault::None
                    };
                    return (
                        protocol::Response {
                            outcome: Some(protocol::response::Outcome::Reset(
                                protocol::ResetResponse::current(None),
                            )),
                        },
                        Some(ControlTransaction::ResetFrontier(Box::new(Runtime::new(
                            super::support::frontier::World::default(),
                            super::support::frontier::Binding {
                                fault,
                                ..super::support::frontier::Binding::default()
                            },
                        )))),
                    );
                }
                Ok(Some(configuration)) if configuration == any(0) || configuration == any(42) => {
                    return (
                        protocol::Response {
                            outcome: Some(protocol::response::Outcome::Reset(
                                protocol::ResetResponse::current(
                                    (configuration == any(42)).then(|| any(42)),
                                ),
                            )),
                        },
                        Some(ControlTransaction::Reset(Box::new(Runtime::new(
                            World { starts: 1 },
                            ControlRealization { bootstrap: true },
                        )))),
                    );
                }
                Ok(Some(_)) => {
                    return (
                        protocol::Response {
                            outcome: Some(protocol::response::Outcome::Refusal(
                                protocol::Refusal::unsupported("unknown control configuration"),
                            )),
                        },
                        None,
                    );
                }
                _ => {}
            }
        }
        if let Some(frontier) = &self.frontier {
            let (response, transaction) = frontier.prepare(method, bytes);
            return (response, transaction.map(ControlTransaction::Frontier));
        }
        let (response, transaction) = self.runtime.prepare(method, bytes);
        (response, transaction.map(ControlTransaction::Model))
    }
    fn commit(&mut self, transaction: Self::Transaction) {
        match transaction {
            ControlTransaction::Model(transaction) => self.runtime.commit(transaction),
            ControlTransaction::Frontier(transaction) => {
                self.frontier
                    .as_mut()
                    .unwrap_or_else(|| panic!("missing frontier runtime"))
                    .commit(transaction);
            }
            ControlTransaction::ResetFrontier(runtime) => self.frontier = Some(*runtime),
            ControlTransaction::Reset(runtime) => {
                self.frontier = None;
                self.runtime = *runtime;
            }
        }
    }
}
pub(crate) fn control() -> Control {
    Control {
        runtime: Runtime::new(World { starts: 1 }, ControlRealization { bootstrap: false }),
        frontier: None,
    }
}
