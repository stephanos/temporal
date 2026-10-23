use umpire_temporal::{Step, activity, realization};
use umpire_runtime::protocol::Refusal;

struct Binding;
struct RequestCancelActivityExecutionResponse;

#[realization(activity)]
impl Binding {
    #[observe(WorkflowService::RequestCancelActivityExecution)]
    fn cancel(_: RequestCancelActivityExecutionResponse) -> Result<Step<activity::Client>, Refusal> {
        Ok(Step { action: activity::Client::Cancel, expected: None })
    }
}

fn main() {}
