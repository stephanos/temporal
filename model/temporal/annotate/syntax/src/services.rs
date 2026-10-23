//! System service classification parameters.

pub use umpire_catalog_syntax::services::{
    ActionBinding, Classification, Declarations, ServiceTable,
};

const SERVICE: &str = "temporal.api.workflowservice.v1.WorkflowService";
const METHOD_ENUM: &str = "WorkflowServiceMethod";

/// Read and resolve reviewed workflow classifications using the shared declaration grammar.
pub fn classifications(source: &str) -> syn::Result<Vec<Classification>> {
    umpire_catalog_syntax::services::classifications(source, SERVICE, METHOD_ENUM)
}

/// Read the workflow service table and feature claims of one source, unresolved.
pub fn declarations(source: &str) -> syn::Result<Declarations> {
    umpire_catalog_syntax::services::declarations(source, SERVICE, METHOD_ENUM)
}

/// Resolve workflow declarations collected from several sources.
pub fn resolve(declarations: Declarations) -> Result<Vec<Classification>, String> {
    declarations.resolve(SERVICE)
}
