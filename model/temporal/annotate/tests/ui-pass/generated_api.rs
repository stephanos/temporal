#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
use umpire_temporal_api::temporal::api::common::v1::WorkflowExecution;
#[derive(Catalog)]
#[mirror(response = "temporal.api.common.v1.WorkflowExecution")]
struct Execution {
    workflow_id: Identifier,
    run_id: Uuid,
}
fn main() -> Result<(), Box<dyn std::error::Error>> {
    let catalog = Execution::catalog(&foundation::dc::Limits)?;
    let message = WorkflowExecution {
        workflow_id: "workflow".into(),
        run_id: "a13cd9da-521c-48d4-ab9c-6ab174876f25".into(),
    };
    assert_eq!(umpire_catalog::validate(&message, &catalog), []);
    Ok(())
}
