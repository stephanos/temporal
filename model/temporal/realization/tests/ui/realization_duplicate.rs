use umpire_temporal::realization;

struct Binding;

#[realization(activity)]
impl Binding {
    #[observe(WorkflowService::DescribeActivityExecution)]
    fn first(response: Response) -> Result<Projection<Action>, Refusal> { todo!() }
    #[observe(temporal::api::workflowservice::v1::WorkflowService::DescribeActivityExecution)]
    fn second(response: Response) -> Result<Projection<Action>, Refusal> { todo!() }
}

fn main() {}
