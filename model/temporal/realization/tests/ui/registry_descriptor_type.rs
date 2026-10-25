use umpire_runtime::{AnyMessage as _, capability::Identity, prost_types::Any, protocol::Refusal};
use umpire_temporal_api::temporal::api::workflowservice::v1::*;
#[derive(Clone, PartialEq, prost::Message)]
struct RequestCancelActivityExecutionRequest {
    #[prost(string, tag = "1")]
    namespace: String,
    #[prost(string, tag = "2")]
    activity_id: String,
    #[prost(string, tag = "3")]
    run_id: String,
    #[prost(string, tag = "4")]
    request_id: String,
}
impl prost::Name for RequestCancelActivityExecutionRequest {
    const NAME: &'static str = "RequestCancelActivityExecutionRequest";
    const PACKAGE: &'static str = "fake";
}
#[path = "../registry/fixture.rs"]
mod fixture;
fn main() {}
