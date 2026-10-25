umpire_temporal::registry! {
    model { specimen }
    actors { enum Client { #[metadata(temporal(event = "event", rpc = WorkflowService::RequestCancelActivityExecution, evidence = event))] First } }
    body { #[registry(inputs = ())] impl Binding {} }
}
fn main() {}
