#[path = "../../../../backends/native/tests/support/mod.rs"]
#[allow(dead_code)]
pub(crate) mod support;

use umpire_runtime::{AnswerPlan, Projection, Realize, Runtime, google::protobuf::Any, protocol};

pub(crate) struct Unsupported;

impl Realize<support::Tiny> for Unsupported {
    fn observe(
        &self,
        _: &Any,
        _: &support::Value,
    ) -> Result<Projection<support::Action, bool>, protocol::Refusal> {
        Err(protocol::Refusal::unsupported("control has no observation binding"))
    }

    fn answer(
        &self,
        _: &Any,
        _: &support::Value,
    ) -> Result<AnswerPlan<support::Action>, protocol::Refusal> {
        Err(protocol::Refusal::unsupported("control has no callback binding"))
    }
}

pub(crate) type Model = Runtime<support::Tiny, Unsupported>;

pub(crate) fn model(mode: support::Mode) -> Model {
    Runtime::new(support::Tiny(mode), Unsupported)
}
