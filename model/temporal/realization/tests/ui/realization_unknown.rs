use umpire_temporal::realization;

struct Binding;

#[realization(activity)]
impl Binding {
    #[observe(WorkflowService::DescribeActivityExecution, reads(info.not_a_field))]
    fn first(response: DescribeActivityExecutionResponse) -> Result<Projection<Action>, Refusal> {
        todo!()
    }
}

fn main() {}
