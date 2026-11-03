use umpire_runtime::{Evidence, EvidenceMatch, protocol::Refusal};
use umpire_temporal_api::temporal::api::workflowservice::v1::*;
#[path = "../registry/signature_model.rs"]
mod fixture;
use fixture::{Binding, specimen};
specimen::action_metadata!(umpire_temporal::registry; specimen; {
    #[registry(inputs = ())]
    impl Binding {
        fn fill(&self, _: &specimen::Client, _: &specimen::World, _: &(), _: &()) -> Result<DescribeActivityExecutionRequest, Refusal> {
            Ok(DescribeActivityExecutionRequest::default())
        }
        fn evidence(&self, _: RequestCancelActivityExecutionRequest, _: Result<RequestCancelActivityExecutionResponse, i32>, _: &[&Evidence<specimen::World>]) -> Result<EvidenceMatch, Refusal> {
            Ok(EvidenceMatch::Reject)
        }
    }
});
fn main() {}
