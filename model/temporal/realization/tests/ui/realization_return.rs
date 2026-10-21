use umpire_temporal::realization;

struct Binding;

#[realization(activity)]
impl Binding {
    #[observe(WorkflowService::DescribeActivityExecution)]
    fn first(response: Response) -> Result<usize, Refusal> { todo!() }
}

fn main() {}
