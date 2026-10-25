//! Descriptor-driven request mutations are independent of annotation mirrors.

use prost::Message;
use umpire_catalog::{GenerationError, MutationApplicability, WireKind, WireMutation, wire_cases};
use umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest;
use umpire_temporal_api::{
    WorkflowServiceMethod, mutate_workflow_request, workflow_request_fields,
};

#[path = "../wire_schema.rs"]
mod generator;

#[test]
fn ignored_fields_and_descriptor_shapes_remain_mutable() -> Result<(), Box<dyn std::error::Error>> {
    let method = WorkflowServiceMethod::StartActivityExecution;
    let fields = workflow_request_fields(method);
    let root = "temporal.api.workflowservice.v1.StartActivityExecutionRequest";
    for name in [
        "search_attributes",
        "header",
        "user_metadata",
        "priority",
        "completion_callbacks",
        "links",
        "on_conflict_options",
    ] {
        assert!(
            fields.iter().any(|field| field.path == format!("{root}.{name}")),
            "omitted ignored field: {name}"
        );
    }
    let bytes = mutate_workflow_request(
        method,
        &StartActivityExecutionRequest::default().encode_to_vec(),
        &format!("{root}.user_metadata.summary.data"),
        &WireMutation::LimitPlusOne { limit: 5 },
    )?;
    let request = StartActivityExecutionRequest::decode(bytes.as_slice())?;
    assert_eq!(
        request
            .user_metadata
            .ok_or("missing metadata")?
            .summary
            .ok_or("missing summary")?
            .data
            .len(),
        6
    );
    assert!(fields.iter().any(|field| matches!(field.kind, WireKind::Map)));
    Ok(())
}

#[test]
fn every_applicable_descriptor_case_constructs_a_typed_request()
-> Result<(), Box<dyn std::error::Error>> {
    let mut fields_seen = 0;
    let mut cases_applied = 0;
    let mut refusals = 0;
    for method in [
        WorkflowServiceMethod::StartActivityExecution,
        WorkflowServiceMethod::DescribeActivityExecution,
        WorkflowServiceMethod::PollActivityExecution,
        WorkflowServiceMethod::RequestCancelActivityExecution,
        WorkflowServiceMethod::PollActivityTaskQueue,
        WorkflowServiceMethod::RespondActivityTaskCompleted,
        WorkflowServiceMethod::RespondActivityTaskFailed,
        WorkflowServiceMethod::RespondActivityTaskCanceled,
    ] {
        let fields = workflow_request_fields(method);
        fields_seen += fields.len();
        for case in wire_cases(&fields, &[5]) {
            if matches!(case.applicability, MutationApplicability::Applicable) {
                let bytes = mutate_workflow_request(method, &[], &case.field.path, &case.mutation)
                    .map_err(|error| {
                        format!("{}:{}: {error}", case.field.path, case.mutation.name())
                    })?;
                assert!(
                    !bytes.is_empty()
                        || matches!(case.mutation, WireMutation::Absent | WireMutation::Empty)
                );
                cases_applied += 1;
            } else if matches!(case.applicability, MutationApplicability::Refused(_)) {
                refusals += 1;
            }
        }
    }
    assert!(fields_seen > 100 && fields_seen < 1000);
    assert!(cases_applied > fields_seen);
    assert!(refusals > 0, "recursive/depth boundaries must remain visible");
    Ok(())
}

#[test]
fn descriptor_changes_drive_inventory_and_unsupported_shapes_refuse()
-> Result<(), Box<dyn std::error::Error>> {
    use prost_types::{FieldDescriptorProto, FileDescriptorSet, field_descriptor_proto::Type};
    let bytes = include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb"));
    let mut descriptors = FileDescriptorSet::decode(bytes.as_slice())?;
    let root = descriptors
        .file
        .iter_mut()
        .flat_map(|file| &mut file.message_type)
        .find(|message| message.name.as_deref() == Some("StartActivityExecutionRequest"))
        .ok_or("missing root")?;
    root.field.push(FieldDescriptorProto {
        name: Some("new_unannotated_field".into()),
        number: Some(999),
        r#type: Some(Type::String as i32),
        ..FieldDescriptorProto::default()
    });
    let source = generator::generate(&descriptors.encode_to_vec())?;
    assert!(source.contains("StartActivityExecutionRequest.new_unannotated_field"));
    assert!(source.contains("node0.r#new_unannotated_field"));
    descriptors.file.reverse();
    assert_eq!(source, generator::generate(&descriptors.encode_to_vec())?);

    let rpc = descriptors
        .file
        .iter_mut()
        .flat_map(|file| &mut file.service)
        .flat_map(|service| &mut service.method)
        .find(|rpc| rpc.name.as_deref() == Some("StartActivityExecution"))
        .ok_or("missing RPC")?;
    let original = rpc
        .input_type
        .replace(".temporal.api.workflowservice.v1.DescribeActivityExecutionRequest".into());
    let changed = generator::generate(&descriptors.encode_to_vec())?;
    assert!(changed.contains("WorkflowServiceMethod::StartActivityExecution=>{let mut message=<temporal::api::workflowservice::v1::DescribeActivityExecutionRequest"));
    let rpc = descriptors
        .file
        .iter_mut()
        .flat_map(|file| &mut file.service)
        .flat_map(|service| &mut service.method)
        .find(|rpc| rpc.name.as_deref() == Some("StartActivityExecution"))
        .ok_or("missing RPC")?;
    rpc.input_type = original;
    let root = descriptors
        .file
        .iter_mut()
        .flat_map(|file| &mut file.message_type)
        .find(|message| message.name.as_deref() == Some("StartActivityExecutionRequest"))
        .ok_or("missing root")?;
    let field = root.field.last_mut().ok_or("missing added field")?;
    field.r#type = Some(Type::Message as i32);
    field.type_name = Some(".missing.Message".into());
    assert!(generator::generate(&descriptors.encode_to_vec()).is_err());
    Ok(())
}

#[test]
fn invalid_input_unknown_routes_and_unbounded_collections_refuse() {
    let method = WorkflowServiceMethod::StartActivityExecution;
    let path = "temporal.api.workflowservice.v1.StartActivityExecutionRequest.links";
    assert_eq!(
        mutate_workflow_request(method, &[0xff], path, &WireMutation::Empty),
        Err(GenerationError::InvalidEncoding)
    );
    assert_eq!(
        mutate_workflow_request(method, &[], "unknown.path", &WireMutation::Empty),
        Err(GenerationError::UnsupportedType)
    );
    assert_eq!(
        mutate_workflow_request(method, &[], path, &WireMutation::LimitPlusOne { limit: 4096 }),
        Err(GenerationError::SizeLimit)
    );
    assert_eq!(
        mutate_workflow_request(method, &[], path, &WireMutation::Negative),
        Err(GenerationError::InapplicableMutation)
    );
}

#[test]
fn nested_absence_does_not_materialize_an_ancestor() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_temporal_api::temporal::api::common::v1::{Header, Link, Payload, link};
    let root = "temporal.api.workflowservice.v1.StartActivityExecutionRequest";
    let seeded = StartActivityExecutionRequest {
        header: Some(Header { fields: [("kept".into(), Payload::default())].into() }),
        ..StartActivityExecutionRequest::default()
    }
    .encode_to_vec();
    let alternative = StartActivityExecutionRequest {
        links: vec![Link { variant: Some(link::Variant::BatchJob(link::BatchJob::default())) }],
        ..StartActivityExecutionRequest::default()
    }
    .encode_to_vec();
    for (bytes, path) in [
        (Vec::new(), format!("{root}.user_metadata.summary.data")),
        (seeded, format!("{root}.header.fields.value.data")),
        (alternative.clone(), format!("{root}.links.workflow_event")),
        (alternative, format!("{root}.links.workflow_event.namespace")),
    ] {
        assert_eq!(
            mutate_workflow_request(
                WorkflowServiceMethod::StartActivityExecution,
                &bytes,
                &path,
                &WireMutation::Absent
            )?,
            bytes
        );
    }
    Ok(())
}

#[test]
fn double_wrapper_descriptor_value_mutates_the_native_scalar()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_temporal_api::temporal::api::workflowservice::v1::PollActivityTaskQueueRequest;
    let method = WorkflowServiceMethod::PollActivityTaskQueue;
    let path = "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue_metadata.max_tasks_per_second";
    let present = mutate_workflow_request(method, &[], path, &WireMutation::Empty)?;
    let decoded = PollActivityTaskQueueRequest::decode(present.as_slice())?;
    assert_eq!(
        decoded.task_queue_metadata.ok_or("missing metadata")?.max_tasks_per_second,
        Some(0.0)
    );
    let changed = mutate_workflow_request(
        method,
        &present,
        &format!("{path}.value"),
        &WireMutation::Negative,
    )?;
    let decoded = PollActivityTaskQueueRequest::decode(changed.as_slice())?;
    assert_eq!(
        decoded.task_queue_metadata.ok_or("missing metadata")?.max_tasks_per_second,
        Some(-1.0)
    );
    let removed = mutate_workflow_request(method, &changed, path, &WireMutation::Absent)?;
    let decoded = PollActivityTaskQueueRequest::decode(removed.as_slice())?;
    assert_eq!(decoded.task_queue_metadata.ok_or("missing metadata")?.max_tasks_per_second, None);
    Ok(())
}
