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

impl umpire_runtime::capability::Identity for Binding {
    type Id = ();
    fn identify(&self, _: &umpire_runtime::prost_types::Any) -> Result<(), Refusal> {
        Err(Refusal::unsupported("identity unsupported"))
    }
}

impl umpire_runtime::capability::Fill<<activity::World as umpire_dsl::Machine>::Action> for Binding {
    type World = activity::World;
    type Inputs = ();
    type Request = umpire_runtime::capability::NoRequest;
}
