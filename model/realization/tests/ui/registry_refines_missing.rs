use umpire_runtime::{Evidence, EvidenceMatch, protocol::Refusal};
use umpire_temporal_api::temporal::api::workflowservice::v1::*;
#[path = "../registry/signature_model.rs"]
mod fixture;
use fixture::{Binding, specimen};
specimen::action_metadata!(umpire_temporal::registry; specimen; {
    #[registry(inputs = ())]
    #[entity(rpc = WorkflowService::StartActivityExecution,
        request = StartActivityExecutionRequest, response = StartActivityExecutionResponse,
        receipt = (), admit = admit, make_receipt = receipt, correlate = correlate)]
    #[read(rpc = WorkflowService::DescribeActivityExecution,
        request = DescribeActivityExecutionRequest, response = DescribeActivityExecutionResponse,
        target = umpire_runtime::Exact<specimen::Model>, fill = read_fill, project = project,
        finality = finality, continue_polling = polling, correlate = read_correlate)]
    impl Binding {
        fn fill(&self, _: &specimen::Client, _: &specimen::World, _: &(), _: &()) -> Result<RequestCancelActivityExecutionRequest, Refusal> { Ok(RequestCancelActivityExecutionRequest::default()) }
        fn evidence(&self, _: RequestCancelActivityExecutionRequest, _: Result<RequestCancelActivityExecutionResponse, i32>, _: &[&Evidence<specimen::World>]) -> Result<EvidenceMatch, Refusal> { Ok(EvidenceMatch::Reject) }
        fn admit(&self, _: &StartActivityExecutionRequest, _: &StartActivityExecutionResponse) -> Result<(), Refusal> { Ok(()) }
        fn receipt(&self, _: &(), _: &StartActivityExecutionRequest, _: &StartActivityExecutionResponse) -> Result<(), Refusal> { Ok(()) }
        fn correlate(&self, _: &(), _: &StartActivityExecutionRequest, _: &StartActivityExecutionResponse) -> Result<(), Refusal> { Ok(()) }
        fn read_correlate(&self, _: &(), _: &DescribeActivityExecutionRequest, _: &DescribeActivityExecutionResponse) -> Result<(), Refusal> { Ok(()) }
        fn read_fill(&self, _: &specimen::World, _: &()) -> Result<DescribeActivityExecutionRequest, Refusal> { Ok(DescribeActivityExecutionRequest::default()) }
        fn project(&self, _: &DescribeActivityExecutionResponse) -> Result<umpire_runtime::Exact<specimen::Model>, Refusal> { Err(Refusal::unsupported("unobserved")) }
        fn finality(&self, _: &umpire_runtime::Exact<specimen::Model>) -> bool { false }
        fn polling(&self, _: &umpire_runtime::Exact<specimen::Model>) -> bool { false }
    }
});
fn refined<B: umpire_temporal::registry::Refined>() {}
fn main() {
    refined::<Binding>();
}
