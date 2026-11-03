#![forbid(unsafe_code)]

use author_kit::dsl::Machine;
use author_kit::runtime::{
    AnyMessage as _, Realize,
    capability::{Fill, Identity, NoRequest},
    prost_types::Any,
    protocol::Refusal,
};
use temporal_kit::{Step, activity, realization};

struct Binding;

impl Identity for Binding {
    type Id = ();
    fn identify(&self, _: &Any) -> Result<(), Refusal> {
        Err(Refusal::unsupported("identity unsupported"))
    }
}
impl Fill<<activity::World as Machine>::Action> for Binding {
    type World = activity::World;
    type Inputs = ();
    type Request = NoRequest;
}

#[cfg(not(feature = "invalid-enum"))]
#[realization(activity)]
impl Binding {
    #[observe(WorkflowService::RequestCancelActivityExecution)]
    fn cancel(
        _: api_kit::temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse,
    ) -> Result<Step<activity::Client>, Refusal> {
        Ok(Step { action: activity::Client::Cancel, expected: None })
    }
}

#[cfg(feature = "invalid-enum")]
#[realization(activity)]
impl Binding {
    fn status(value: api_kit::temporal::api::enums::v1::ActivityExecutionStatus) {
        match value {
            api_kit::temporal::api::enums::v1::ActivityExecutionStatus::Running => (),
            _ => (),
        }
    }
}

#[cfg(not(feature = "invalid-enum"))]
fn main() {
    let world = activity::World::new(3, None, None, None, None, false);
    let state = world.starts().remove(0);
    let observation = Any::pack(&author_kit::runtime::realization::Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        request: None,
        result: Some(author_kit::runtime::realization::observation::Result::Response(Any::pack(
            &api_kit::temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse {},
        ))),
    });
    let _ = Binding.observe(&world, &observation, &state);
}

#[cfg(feature = "invalid-enum")]
fn main() {}
