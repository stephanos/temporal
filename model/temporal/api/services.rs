//! Service generation configuration for the API slice.
use umpire_temporal_annotate_syntax::{descriptor::Coverage, services::Classification};
pub(crate) fn generate(
    bytes: &[u8],
    coverage: &Coverage,
    classifications: &[Classification],
) -> Result<String, String> {
    umpire_catalog_build::services::generate(
        bytes,
        coverage,
        classifications,
        &umpire_catalog_build::services::Config {
            service: "temporal.api.workflowservice.v1.WorkflowService",
            rust_type_root: "::umpire_temporal_api",
            macro_type_root: "$crate",
            method_enum: "WorkflowServiceMethod",
            method_info: "WorkflowServiceMethodInfo",
            methods_const: "WORKFLOW_SERVICE_METHODS",
            dispatch_macro: "dispatch_workflow_annotations",
            bind_macro: "bind_workflow_action",
        },
    )
}
