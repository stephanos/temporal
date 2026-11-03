use umpire_runtime::{AnyMessage as _, capability::Identity, prost_types::Any, protocol::Refusal};
use umpire_temporal_api::temporal::api::workflowservice::v1::*;
type RespondActivityTaskCompletedResponse = RequestCancelActivityExecutionResponse;
#[path = "../registry/fixture.rs"]
mod fixture;
fn main() {}
