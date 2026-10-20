use umpire_temporal_api::temporal::api::common::v1::WorkflowExecution;

umpire_temporal_annotate::annotate! {
    catalog controls;
    message response execution: WorkflowExecution = "temporal.api.common.v1.WorkflowExecution" {
        workflow_id: Identifier;
        run_id: Uuid;
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let catalog = controls::execution()?;
    let message = WorkflowExecution {
        workflow_id: "workflow".into(),
        run_id: "a13cd9da-521c-48d4-ab9c-6ab174876f25".into(),
    };
    assert_eq!(umpire_annotations::validate(&message, &catalog), []);
    Ok(())
}
