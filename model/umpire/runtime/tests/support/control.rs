use crate::support::{Action, Outcome, Realization, State, World, any, payload};
use umpire_runtime::{AnswerPlan, Projection, Realize, Runtime, google::protobuf::Any, protocol};

pub(crate) struct ControlRealization {
    pub(crate) bootstrap: bool,
}
impl Realize<World> for ControlRealization {
    fn needs_bootstrap(&self) -> bool {
        self.bootstrap
    }
    fn bootstrap(
        &self,
        _: &State,
        _: u64,
    ) -> Result<umpire_runtime::Proposal<Action>, protocol::Refusal> {
        Ok(umpire_runtime::Proposal::Bootstrap { request: any(0) })
    }
    fn propose(
        &self,
        state: &State,
        seed: u64,
    ) -> Result<umpire_runtime::Proposal<Action>, protocol::Refusal> {
        Realization.propose(state, seed)
    }
    fn observe(
        &self,
        value: &Any,
        state: &State,
    ) -> Result<Projection<Action, Outcome>, protocol::Refusal> {
        if payload(value)? == 20 {
            return Ok(Projection::Current);
        }
        Realization.observe(value, state)
    }
    fn answer(&self, value: &Any, state: &State) -> Result<AnswerPlan<Action>, protocol::Refusal> {
        Realization.answer(value, state)
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

pub(crate) struct Control {
    runtime: Runtime<World, ControlRealization>,
}
pub(crate) enum ControlTransaction {
    Model(umpire_runtime::Transaction<World>),
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
                Ok(Some(configuration)) if configuration == any(0) => {
                    return (
                        protocol::Response {
                            outcome: Some(protocol::response::Outcome::Reset(
                                protocol::ResetResponse {},
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
        let (response, transaction) = self.runtime.prepare(method, bytes);
        (response, transaction.map(ControlTransaction::Model))
    }
    fn commit(&mut self, transaction: Self::Transaction) {
        match transaction {
            ControlTransaction::Model(transaction) => self.runtime.commit(transaction),
            ControlTransaction::Reset(runtime) => {
                self.runtime = *runtime;
            }
        }
    }
}
pub(crate) fn control() -> Control {
    Control { runtime: Runtime::new(World { starts: 1 }, ControlRealization { bootstrap: false }) }
}
