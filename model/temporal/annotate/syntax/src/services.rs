//! System service classification parameters.

pub use umpire_catalog_syntax::services::{ActionBinding, Classification};

/// Read reviewed workflow classifications using the shared declaration grammar.
pub fn classifications(source: &str) -> syn::Result<Vec<Classification>> {
    umpire_catalog_syntax::services::classifications(
        source,
        "temporal.api.workflowservice.v1.WorkflowService",
        "WorkflowServiceMethod",
    )
}
