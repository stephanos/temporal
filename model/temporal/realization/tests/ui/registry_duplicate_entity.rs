umpire_temporal::registry! {
    model { specimen }
    actors {}
    body {
        #[registry(inputs = ())]
        #[entity(rpc = WorkflowService::StartActivityExecution)]
        #[entity(rpc = WorkflowService::StartActivityExecution)]
        impl Binding {}
    }
}
fn main() {}
