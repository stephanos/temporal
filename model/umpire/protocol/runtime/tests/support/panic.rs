#[path = "../../../../backends/native/tests/support/mod.rs"]
#[allow(dead_code)]
pub(crate) mod support;

use umpire_runtime::{AnswerPlan, Projection, Realize, Runtime, prost_types::Any, protocol};

pub(crate) struct Unsupported;

impl umpire_runtime::capability::Identity for Unsupported {
    type Id = ();

    fn identify(&self, _: &Any) -> Result<(), protocol::Refusal> {
        Ok(())
    }
}

impl umpire_runtime::capability::Fill<<support::Tiny as umpire_dsl::Machine>::Action>
    for Unsupported
{
    type World = support::Tiny;
    type Inputs = ();
    type Request = umpire_runtime::capability::NoRequest;
}

impl umpire_runtime::capability::Snapshot<support::Tiny> for Unsupported {
    type Target = umpire_runtime::NoTarget;

    fn project(
        &self,
        _world: &support::Tiny,
        _: &Any,
        _: &support::Value,
    ) -> Result<Projection<support::Action, bool, umpire_runtime::NoTarget>, protocol::Refusal>
    {
        Err(protocol::Refusal::unsupported("control has no observation binding"))
    }
}

impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for Unsupported {
    type World = support::Tiny;
    type View = umpire_runtime::capability::NoEvidence;
}

impl umpire_runtime::capability::ReplyPolicy<support::Tiny> for Unsupported {
    fn answer(
        &self,
        _world: &support::Tiny,
        _: &Any,
        _: &support::Value,
    ) -> Result<AnswerPlan<support::Action>, protocol::Refusal> {
        Err(protocol::Refusal::unsupported("control has no callback binding"))
    }
}

impl Realize<support::Tiny> for Unsupported {}

pub(crate) type Model = Runtime<support::Tiny, Unsupported>;

pub(crate) fn model(mode: support::Mode) -> Model {
    Runtime::new(support::Tiny(mode), Unsupported)
}
