use umpire_runtime::{Realize, Runtime, prost_types::Any, protocol};

#[path = "../../../../../backends/native/tests/fixtures/sample_kettle_trait/kettle.rs"]
#[allow(missing_docs, unreachable_pub)]
mod kettle;

pub(crate) type Model = Runtime<kettle::KettleWorld, Unsupported>;

pub(crate) struct Unsupported;

impl umpire_runtime::capability::Identity for Unsupported {
    type Id = ();

    fn identify(&self, _: &Any) -> Result<(), protocol::Refusal> {
        Ok(())
    }
}

impl umpire_runtime::capability::Fill<<kettle::KettleWorld as umpire_dsl::Machine>::Action>
    for Unsupported
{
    type World = kettle::KettleWorld;
    type Inputs = ();
    type Request = umpire_runtime::capability::NoRequest;
}

impl umpire_runtime::capability::Snapshot<kettle::KettleWorld> for Unsupported {
    type Target = umpire_runtime::NoTarget;

    fn project(
        &self,
        _world: &kettle::KettleWorld,
        _: &Any,
        _: &kettle::Kettle,
    ) -> Result<
        umpire_runtime::Projection<kettle::Act, (), umpire_runtime::NoTarget>,
        protocol::Refusal,
    > {
        Err(unsupported())
    }
}

impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for Unsupported {
    type World = kettle::KettleWorld;
    type View = umpire_runtime::capability::NoEvidence;
}

impl umpire_runtime::capability::ReplyPolicy<kettle::KettleWorld> for Unsupported {
    fn enabled(&self) -> bool {
        false
    }

    fn answer(
        &self,
        _world: &kettle::KettleWorld,
        _: &Any,
        _: &kettle::Kettle,
    ) -> Result<umpire_runtime::AnswerPlan<kettle::Act>, protocol::Refusal> {
        Err(unsupported())
    }
}

impl Realize<kettle::KettleWorld> for Unsupported {}

fn unsupported() -> protocol::Refusal {
    protocol::Refusal {
        code: protocol::RefusalCode::Unsupported as i32,
        message: "stateful operation unsupported".into(),
    }
}

pub(crate) fn model() -> Model {
    Runtime::new(kettle::KettleWorld, Unsupported)
}
