use umpire_temporal::{Step, activity, realization};
use umpire_runtime::protocol::Refusal;
use umpire_temporal_api::temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse;

struct Binding;

#[realization(activity)]
#[reach(target = umpire_temporal::ActivityTarget, project = project, evidence = evidence)]
impl Binding {
    #[observe(WorkflowService::RequestCancelActivityExecution)]
    fn cancel(_: RequestCancelActivityExecutionResponse) -> Result<Step<activity::Client>, Refusal> {
        Ok(Step { action: activity::Client::Cancel, expected: None })
    }
}

fn main() {}
