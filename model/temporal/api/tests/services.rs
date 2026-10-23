//! Workflow method admission and dispatch use the real linked upstream descriptor.

use std::{
    collections::BTreeSet,
    fmt::Write,
    fs,
    path::Path,
    process::Command,
    time::{SystemTime, UNIX_EPOCH},
};

use prost::Message;
use umpire_runtime::google::protobuf::FileDescriptorSet;
use umpire_temporal_annotate_syntax::{descriptor::Coverage, services::classifications};

#[path = "../services.rs"]
mod services;

#[test]
fn generated_method_table_follows_the_actual_descriptor() -> Result<(), Box<dyn std::error::Error>>
{
    use umpire_temporal_api::{WORKFLOW_SERVICE_METHODS, WorkflowServiceMethod};
    let descriptors = FileDescriptorSet::decode(
        include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb")).as_slice(),
    )?;
    let service = descriptors
        .file
        .iter()
        .flat_map(|file| &file.service)
        .find(|service| service.name.as_deref() == Some("WorkflowService"))
        .ok_or("missing service")?;
    let names: Vec<_> =
        WORKFLOW_SERVICE_METHODS.iter().map(|method| method.method.name()).collect();
    assert_eq!(
        names,
        service.method.iter().filter_map(|method| method.name.as_deref()).collect::<Vec<_>>()
    );
    for (method, descriptor) in WORKFLOW_SERVICE_METHODS.iter().zip(&service.method) {
        assert_eq!(WorkflowServiceMethod::parse(method.method.name()), Some(method.method));
        assert_eq!(
            method.request_type,
            format!(
                "::umpire_temporal_api::{}",
                descriptor
                    .input_type
                    .as_deref()
                    .ok_or("missing request")?
                    .trim_start_matches('.')
                    .replace('.', "::")
            )
        );
        assert_eq!(
            method.response_type,
            format!(
                "::umpire_temporal_api::{}",
                descriptor
                    .output_type
                    .as_deref()
                    .ok_or("missing response")?
                    .trim_start_matches('.')
                    .replace('.', "::")
            )
        );
    }
    assert_eq!(WorkflowServiceMethod::parse("new unknown method"), None);
    Ok(())
}

#[test]
fn dispatch_selects_the_actual_annotated_type_and_direction() {
    use umpire_temporal_api::WorkflowServiceMethod;
    for (response, expected) in
        [(false, "PollActivityTaskQueueRequest"), (true, "PollActivityTaskQueueResponse")]
    {
        let name = umpire_temporal_api::dispatch_workflow_annotations!(
            WorkflowServiceMethod::PollActivityTaskQueue,
            response,
            Proto,
            { std::any::type_name::<Proto>() },
            "unannotated"
        );
        assert!(name.ends_with(expected), "dispatched {name}");
    }
    assert_eq!(
        umpire_temporal_api::dispatch_workflow_annotations!(
            WorkflowServiceMethod::GetSystemInfo,
            false,
            Proto,
            { std::any::type_name::<Proto>() },
            "unannotated"
        ),
        "unannotated"
    );
}

#[test]
fn generation_uses_annotations_for_dispatch_and_source_match_for_bindings()
-> Result<(), Box<dyn std::error::Error>> {
    let source = r"fn classify(method: WorkflowServiceMethod) -> RpcRole { match method {
        WorkflowServiceMethod::PollActivityTaskQueue => RpcRole::Action(ModelAction::Worker(Worker::Poll)),
    }}";
    let coverage = Coverage {
        messages: vec![umpire_temporal_annotate_syntax::descriptor::MessageCoverage {
            name: "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest".into(),
            fields: vec![],
            opaque_fields: vec![],
        }],
        ..Coverage::default()
    };
    let generated = services::generate(
        include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb")),
        &coverage,
        &classifications(source)?,
    )?;
    assert!(generated.contains("ModelAction :: Worker (Worker :: Poll) => Binding::Rpc(\"temporal.api.workflowservice.v1.WorkflowService.PollActivityTaskQueue\")"));
    assert!(generated.contains("($crate::WorkflowServiceMethod::PollActivityTaskQueue, false) => { type $ty = $crate::temporal::api::workflowservice::v1::PollActivityTaskQueueRequest; $body }"));
    assert!(!generated.contains("($crate::WorkflowServiceMethod::PollActivityTaskQueue, true) =>"));
    let source_bindings: BTreeSet<_> =
        classifications(source)?[0].bindings.iter().map(|binding| binding.0.clone()).collect();
    assert_eq!(source_bindings.len(), 1);
    Ok(())
}

#[test]
fn generated_matches_refuse_new_rpc_and_model_action() -> Result<(), Box<dyn std::error::Error>> {
    let bytes = include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb"));
    let mut descriptors = FileDescriptorSet::decode(bytes.as_slice())?;
    let source = "fn classify(method: WorkflowServiceMethod) -> RpcRole { match method { WorkflowServiceMethod::PollActivityTaskQueue => RpcRole::Action(ModelAction::Worker(Worker::Poll)) }}";
    let classifications = classifications(source)?;
    let mut generated = services::generate(bytes, &Coverage::default(), &classifications)?;
    let mut classify =
        String::from("fn classify(method: WorkflowServiceMethod) { match method {\n");
    let service = descriptors
        .file
        .iter_mut()
        .flat_map(|file| &mut file.service)
        .find(|service| service.name.as_deref() == Some("WorkflowService"))
        .ok_or("missing service")?;
    for method in &service.method {
        writeln!(
            classify,
            "WorkflowServiceMethod::{} => (),",
            method.name.as_deref().ok_or("unnamed method")?
        )?;
    }
    classify.push_str("}}\n");
    let model = r#"
        mod model {
        enum Worker { Poll }
        enum ModelAction { Worker(Worker), Deadline }
        enum Binding { Rpc(&'static str), Environmental(&'static str) }
        fn bind(action: &ModelAction) -> Binding {
            bind_workflow_action!(action, ModelAction::Deadline => Binding::Environmental("clock"))
        }
        }
        fn main() {}
    "#;
    let scratch = Path::new(env!("OUT_DIR")).join(format!(
        "services-controls-{}",
        SystemTime::now().duration_since(UNIX_EPOCH)?.as_nanos()
    ));
    fs::create_dir(&scratch)?;
    compile_control(&scratch, &format!("{generated}{classify}{model}"), None)?;
    compile_control(
        &scratch,
        &format!(
            "{generated}{classify}{}",
            model.replace("enum Worker { Poll }", "enum Worker { Poll, AddedAction }")
        ),
        Some("AddedAction"),
    )?;
    let mut added = service.method.first().ok_or("empty service")?.clone();
    added.name = Some("AddedWorkflowMethod".into());
    service.method.push(added);
    generated =
        services::generate(&descriptors.encode_to_vec(), &Coverage::default(), &classifications)?;
    compile_control(
        &scratch,
        &format!("{generated}{classify}{model}"),
        Some("AddedWorkflowMethod"),
    )?;
    fs::remove_dir_all(scratch)?;
    Ok(())
}

fn compile_control(
    directory: &Path,
    source: &str,
    missing: Option<&str>,
) -> Result<(), Box<dyn std::error::Error>> {
    let input = directory.join("control.rs");
    fs::write(&input, source)?;
    let output = Command::new("rustc")
        .args(["--edition=2024", "--crate-name=service_control", "--emit=metadata", "-Awarnings"])
        .arg(&input)
        .arg("--out-dir")
        .arg(directory)
        .output()?;
    let stderr = String::from_utf8(output.stderr)?;
    if let Some(missing) = missing {
        assert!(!output.status.success(), "new symbol passed exhaustive match: {missing}");
        assert!(stderr.contains("E0004") && stderr.contains(missing), "wrong refusal: {stderr}");
    } else {
        assert!(output.status.success(), "positive control failed: {stderr}");
    }
    Ok(())
}
