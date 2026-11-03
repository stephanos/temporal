umpire_temporal::registry! {
    model { specimen }
    actors {}
    body {
        #[registry(inputs = ())]
        #[read(rpc = WorkflowService::DescribeActivityExecution)]
        #[read(rpc = WorkflowService::DescribeActivityExecution)]
        impl Binding {}
    }
}
fn main() {}
