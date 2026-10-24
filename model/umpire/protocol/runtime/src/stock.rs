use crate::{Realize, Runtime, google::protobuf::Any, protocol};

#[path = "../../../backends/native/tests/fixtures/sample_kettle_trait/kettle.rs"]
#[allow(missing_docs, unreachable_pub)]
mod kettle;

/// The stock Kettle checker dispatcher.
pub type Model = Runtime<kettle::KettleWorld, Unsupported>;

/// The stock exploration realization has no observation or callback registry.
pub struct Unsupported;

impl Realize<kettle::KettleWorld> for Unsupported {
    fn enabled(&self) -> bool {
        false
    }

    fn observe(
        &self,
        _: &Any,
        _: &kettle::Kettle,
    ) -> Result<crate::Projection<kettle::Act, ()>, protocol::Refusal> {
        Err(unsupported())
    }

    fn answer(
        &self,
        _: &Any,
        _: &kettle::Kettle,
    ) -> Result<crate::AnswerPlan<kettle::Act>, protocol::Refusal> {
        Err(unsupported())
    }
}

fn unsupported() -> protocol::Refusal {
    protocol::Refusal {
        code: protocol::RefusalCode::Unsupported as i32,
        message: "stateful operation unsupported".into(),
    }
}

impl Default for Model {
    fn default() -> Self {
        Self::new(kettle::KettleWorld, Unsupported)
    }
}
