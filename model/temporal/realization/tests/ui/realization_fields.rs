use umpire_temporal::realization;

struct Empty;

#[realization(activity)]
impl Empty {
    #[observe(WorkflowService::DescribeActivityExecution, ignores(info.status = " "))]
    fn describe(response: DescribeActivityExecutionResponse) -> Result<Projection<Action>, Refusal> {
        todo!()
    }
}

struct Overlap;

#[realization(activity)]
impl Overlap {
    #[observe(WorkflowService::DescribeActivityExecution, reads(info), ignores(info.status = "overlap"))]
    fn describe(response: DescribeActivityExecutionResponse) -> Result<Projection<Action>, Refusal> {
        todo!()
    }
}

struct Opaque;

#[realization(activity)]
impl Opaque {
    #[observe(WorkflowService::DescribeActivityExecution, reads(info.priority.name))]
    fn describe(response: DescribeActivityExecutionResponse) -> Result<Projection<Action>, Refusal> {
        todo!()
    }
}

fn main() {}
