//! Request mutation roots for the live API slice.
pub(crate) fn generate(bytes: &[u8]) -> Result<String, String> {
    umpire_catalog_build::wire_schema::generate(
        bytes,
        &umpire_catalog_build::wire_schema::Config {
            service: "temporal.api.workflowservice.v1.WorkflowService",
            rust_type_root: "",
            method_enum: "WorkflowServiceMethod",
            fields_function: "workflow_request_fields",
            mutate_function: "mutate_workflow_request",
            request_roots: &[
                "StartActivityExecution",
                "DescribeActivityExecution",
                "PollActivityExecution",
                "RequestCancelActivityExecution",
                "PollActivityTaskQueue",
                "RespondActivityTaskCompleted",
                "RespondActivityTaskFailed",
                "RespondActivityTaskCanceled",
            ],
        },
    )
}
