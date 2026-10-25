use umpire_runtime::{Realize, Runtime, prost_types::Any, protocol};

#[path = "../../../../../backends/native/tests/fixtures/sample_kettle_trait/kettle.rs"]
#[allow(missing_docs, unreachable_pub)]
mod kettle;

pub(crate) type Model = Runtime<kettle::KettleWorld, Unsupported>;

pub(crate) struct Unsupported;

impl Realize<kettle::KettleWorld> for Unsupported {
    type Target = umpire_runtime::NoTarget;

    fn enabled(&self) -> bool {
        false
    }

    fn observe(
        &self,
        _: &Any,
        _: &kettle::Kettle,
    ) -> Result<
        umpire_runtime::Projection<kettle::Act, (), umpire_runtime::NoTarget>,
        protocol::Refusal,
    > {
        Err(unsupported())
    }

    fn answer(
        &self,
        _: &Any,
        _: &kettle::Kettle,
    ) -> Result<umpire_runtime::AnswerPlan<kettle::Act>, protocol::Refusal> {
        Err(unsupported())
    }
}

fn unsupported() -> protocol::Refusal {
    protocol::Refusal {
        code: protocol::RefusalCode::Unsupported as i32,
        message: "stateful operation unsupported".into(),
    }
}

pub(crate) fn model() -> Model {
    Runtime::new(kettle::KettleWorld, Unsupported)
}
