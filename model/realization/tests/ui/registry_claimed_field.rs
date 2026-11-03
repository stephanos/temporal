umpire_temporal::registry! {
    model { specimen }
    actors { enum Client { #[metadata(temporal(rpc = WorkflowService::RequestCancelActivityExecution, direction = client, request = RequestCancelActivityExecutionRequest, response = RequestCancelActivityExecutionResponse, fill = fill, evidence = evidence, request_reads(unknown_field)))] Ping } }
    body { #[registry(inputs = ())] impl Binding {} }
}
fn main() {}
