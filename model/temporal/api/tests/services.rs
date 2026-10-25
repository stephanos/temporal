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
use prost_types::FileDescriptorSet;
use umpire_temporal_annotate_syntax::{
    descriptor::{self, Coverage},
    services::{classifications, declarations, resolve},
};

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
    let source = r#"fn family(method: WorkflowServiceMethod) -> &'static str { match method {
        WorkflowServiceMethod::PollActivityTaskQueue => "Activity tasks and executions",
        WorkflowServiceMethod::GetSystemInfo => "Cluster and service information",
    }}
    fn classify(method: WorkflowServiceMethod) -> RpcRole<ModelAction> { match method {
        WorkflowServiceMethod::PollActivityTaskQueue => RpcRole::Action(ModelAction::Worker(Worker::Poll)),
        _ => RpcRole::OutOfScope(service::family(method)),
    }}"#;
    let coverage = Coverage {
        messages: vec![descriptor::MessageCoverage {
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
    let generated = generated.parse::<proc_macro2::TokenStream>()?.to_string();
    for (expected, present) in [
        (
            "ModelAction::Worker(Worker::Poll) => Binding::Rpc(\"temporal.api.workflowservice.v1.WorkflowService.PollActivityTaskQueue\")",
            true,
        ),
        (
            "($crate::WorkflowServiceMethod::PollActivityTaskQueue, false) => { type $ty = $crate::temporal::api::workflowservice::v1::PollActivityTaskQueueRequest; $body }",
            true,
        ),
        ("($crate::WorkflowServiceMethod::PollActivityTaskQueue, true) =>", false),
        (
            "(\"temporal.api.workflowservice.v1.WorkflowService.GetSystemInfo\", \"OutOfScope\", \"Cluster and service information\",)",
            true,
        ),
    ] {
        let expected = expected.parse::<proc_macro2::TokenStream>()?.to_string();
        assert_eq!(generated.contains(&expected), present, "{expected}");
    }
    let source_bindings: BTreeSet<_> =
        classifications(source)?[0].bindings.iter().map(|binding| binding.0.clone()).collect();
    assert_eq!(source_bindings.len(), 2);
    Ok(())
}

#[test]
fn service_table_and_bindings_refuse_new_rpc_and_model_action()
-> Result<(), Box<dyn std::error::Error>> {
    let bytes = include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb"));
    let mut descriptors = FileDescriptorSet::decode(bytes.as_slice())?;
    let service = descriptors
        .file
        .iter_mut()
        .flat_map(|file| &mut file.service)
        .find(|service| service.name.as_deref() == Some("WorkflowService"))
        .ok_or("missing service")?;
    let mut table =
        String::from("fn family(method: WorkflowServiceMethod) -> &'static str { match method {\n");
    for method in &service.method {
        writeln!(
            table,
            "WorkflowServiceMethod::{} => \"family\",",
            method.name.as_deref().ok_or("unnamed method")?
        )?;
    }
    table.push_str("}}\n");
    let claim = "fn classify(method: WorkflowServiceMethod) -> RpcRole<ModelAction> { match method { WorkflowServiceMethod::PollActivityTaskQueue => RpcRole::Action(ModelAction::Worker(Worker::Poll)), _ => RpcRole::OutOfScope(family(method)) }}";
    let classifications = classifications(&format!("{table}{claim}"))?;
    let mut generated = services::generate(bytes, &Coverage::default(), &classifications)?;
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
    compile_control(&scratch, &format!("{generated}{table}{model}"), None)?;
    compile_control(
        &scratch,
        &format!(
            "{generated}{table}{}",
            model.replace("enum Worker { Poll }", "enum Worker { Poll, AddedAction }")
        ),
        Some("AddedAction"),
    )?;
    let mut added = service.method.first().ok_or("empty service")?.clone();
    added.name = Some("AddedWorkflowMethod".into());
    service.method.push(added);
    generated =
        services::generate(&descriptors.encode_to_vec(), &Coverage::default(), &classifications)?;
    compile_control(&scratch, &format!("{generated}{table}{model}"), Some("AddedWorkflowMethod"))?;
    fs::remove_dir_all(scratch)?;
    Ok(())
}

/// The reviewed service table and the activity feature's claims, as the API build reads them.
fn reviewed()
-> Result<umpire_temporal_annotate_syntax::services::Declarations, Box<dyn std::error::Error>> {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("..");
    let mut reviewed = declarations(&fs::read_to_string(root.join("foundation/service.rs"))?)?;
    reviewed.extend(declarations(&fs::read_to_string(root.join("feature/activity/api.rs"))?)?);
    Ok(reviewed)
}

#[test]
fn the_service_table_classifies_every_rpc_and_the_activity_feature_claims_eight()
-> Result<(), Box<dyn std::error::Error>> {
    let reviewed = reviewed()?;
    assert_eq!(reviewed.tables.len(), 1);
    assert!(reviewed.classifications.is_empty());
    let claimed: Vec<_> = reviewed
        .claims
        .iter()
        .flat_map(|claim| &claim.coverage.methods)
        .map(|method| method.name.as_str())
        .collect();
    assert_eq!(
        claimed,
        [
            "PollActivityTaskQueue",
            "RespondActivityTaskCompleted",
            "RespondActivityTaskFailed",
            "RespondActivityTaskCanceled",
            "StartActivityExecution",
            "DescribeActivityExecution",
            "PollActivityExecution",
            "RequestCancelActivityExecution",
        ]
    );
    let resolved = resolve(reviewed)?;
    let coverage = Coverage {
        services: resolved.iter().map(|classification| classification.coverage.clone()).collect(),
        ..Coverage::default()
    };
    descriptor::check(
        include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb")),
        &coverage,
    )?;
    for (rpc, role, target) in resolved.iter().flat_map(|classification| &classification.bindings) {
        assert!(
            role != "OutOfScope" || !target.contains("Model"),
            "{rpc} names a Model in its out-of-scope reason: {target}"
        );
    }
    Ok(())
}

/// Message coverage of every catalog the API build reads.
fn mirrored() -> Result<Coverage, Box<dyn std::error::Error>> {
    let mut coverage = Coverage::default();
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("..");
    for path in umpire_catalog_build::coverage::inputs(&root)?.files {
        for catalog in umpire_temporal_annotate_syntax::catalogs(&fs::read_to_string(path)?)? {
            coverage.messages.extend(catalog.coverage().messages);
        }
    }
    Ok(coverage)
}

#[test]
fn a_dropped_claim_or_an_unclaimed_annotated_rpc_fails_generation()
-> Result<(), Box<dyn std::error::Error>> {
    let bytes = include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb"));
    let mut coverage = mirrored()?;
    services::generate(bytes, &coverage, &resolve(reviewed()?)?)?;
    let mut dropped = reviewed()?;
    let claim = &mut dropped.claims[0];
    let index = claim
        .coverage
        .methods
        .iter()
        .position(|method| method.name == "DescribeActivityExecution")
        .ok_or("missing Observed claim")?;
    claim.coverage.methods.remove(index);
    claim.bindings.remove(index);
    assert_eq!(
        services::generate(bytes, &coverage, &resolve(dropped)?).err().as_deref(),
        Some(
            "annotated RPC is out of scope: temporal.api.workflowservice.v1.WorkflowService.DescribeActivityExecution has mirrored messages but no Action or Observed classification"
        )
    );
    coverage.messages.push(descriptor::MessageCoverage {
        name: "temporal.api.workflowservice.v1.RecordActivityTaskHeartbeatRequest".into(),
        fields: vec![],
        opaque_fields: vec![],
    });
    assert_eq!(
        services::generate(bytes, &coverage, &resolve(reviewed()?)?).err().as_deref(),
        Some(
            "annotated RPC is out of scope: temporal.api.workflowservice.v1.WorkflowService.RecordActivityTaskHeartbeat has mirrored messages but no Action or Observed classification"
        )
    );
    Ok(())
}

#[test]
fn duplicate_claims_and_unclassified_rpcs_are_named() -> Result<(), Box<dyn std::error::Error>> {
    let mut duplicate = reviewed()?;
    duplicate.extend(declarations(
        "fn classify(method: WorkflowServiceMethod) -> RpcRole<NexusAction> { match method {
            WorkflowServiceMethod::PollActivityTaskQueue => RpcRole::Observed(\"second feature\"),
            _ => RpcRole::OutOfScope(service::family(method)),
        }}",
    )?);
    assert_eq!(
        resolve(duplicate).err().as_deref(),
        Some(
            "duplicate feature claim: temporal.api.workflowservice.v1.WorkflowService.PollActivityTaskQueue"
        )
    );
    let mut claimed = reviewed()?;
    claimed.tables[0].families.retain(|(method, _)| method != "PollActivityTaskQueue");
    assert!(resolve(claimed).is_err_and(|error| error.starts_with(
        "unclassified RPC: temporal.api.workflowservice.v1.WorkflowService.PollActivityTaskQueue"
    )));
    let mut unclaimed = reviewed()?;
    unclaimed.tables[0]
        .families
        .retain(|(method, _)| method != "PollWorkflowExecutionTimeSkipping");
    let coverage = Coverage {
        services: resolve(unclaimed)?
            .iter()
            .map(|classification| classification.coverage.clone())
            .collect(),
        ..Coverage::default()
    };
    let error = descriptor::check(
        include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb")),
        &coverage,
    )
    .err()
    .ok_or("an RPC missing from the service table passed admission")?;
    assert_eq!(
        error,
        "uncovered protobuf symbol: temporal.api.workflowservice.v1.WorkflowService.PollWorkflowExecutionTimeSkipping"
    );
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
