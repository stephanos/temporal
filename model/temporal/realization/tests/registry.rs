//! Synthetic consumers of the descriptor-admitted typed variant registry.
#![forbid(unsafe_code)]

use umpire_dsl::Machine;
use umpire_runtime::Realize;
use umpire_runtime::{
    AnyMessage as _,
    capability::{Fill, Identity, Request},
    prost_types::Any,
    protocol::Refusal,
};
use umpire_temporal::registry::{ActionKind, Direction, Registry};
use umpire_temporal_api::temporal::api::workflowservice::v1::*;

#[path = "registry/fixture.rs"]
mod fixture;
use fixture::*;

fn id() -> ActualId {
    ActualId { namespace: "ns".into(), entity: "entity".into(), run: "run".into(), token: vec![9] }
}

#[test]
fn proposals_worker_rows_and_probe_baselines_use_one_current_typed_filler() -> Result<(), Refusal> {
    let binding = Binding::default();
    let world = specimen::World::new(4);
    let inputs = Inputs { request_id: "request".into() };
    let identity = binding.identify(&Any::pack(&RequestCancelActivityExecutionRequest {
        namespace: "ns".into(),
        activity_id: "entity".into(),
        run_id: "run".into(),
        ..Default::default()
    }))?;
    let client: specimen::Action = specimen::Client::Ping.into();
    let request = binding.fill(&client, &world, &inputs, &identity)?.pack_request();
    let proposal =
        umpire_runtime::Proposal::Action { action: client.clone(), request: request.clone() };
    assert!(matches!(proposal, umpire_runtime::Proposal::Action { .. }));
    let baseline: RequestCancelActivityExecutionRequest =
        binding.fill(&client, &world, &inputs, &identity)?.pack_request().unpack()?;
    assert_eq!(baseline, request.unpack::<RequestCancelActivityExecutionRequest>()?);
    assert_eq!(
        (
            baseline.namespace.as_str(),
            baseline.activity_id.as_str(),
            baseline.run_id.as_str(),
            baseline.request_id.as_str()
        ),
        ("ns", "entity", "run", "request-4")
    );
    assert!(matches!(
        binding.row(&client).kind,
        ActionKind::Rpc {
            direction: Direction::Client,
            method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
        }
    ));
    for (world_tag, input, token, flag, expected) in [
        (4, "request", vec![9], false, "request-4-false"),
        (5, "request", vec![9], false, "request-5-false"),
        (4, "other", vec![9], false, "other-4-false"),
        (4, "request", vec![7], false, "request-4-false"),
        (4, "request", vec![9], true, "request-4-true"),
    ] {
        let action: specimen::Action = specimen::Worker::Notify { flag }.into();
        let mut identity = identity.clone();
        identity.token = token.clone();
        let row: RespondActivityTaskCompletedRequest = binding
            .fill(
                &action,
                &specimen::World::new(world_tag),
                &Inputs { request_id: input.into() },
                &identity,
            )?
            .pack_request()
            .unpack()?;
        assert_eq!(row.task_token, token);
        assert_eq!(row.identity, expected);
        assert!(matches!(
            binding.row(&action).kind,
            ActionKind::Rpc { direction: Direction::Worker, .. }
        ));
    }
    let repeated: specimen::Action = specimen::Worker::RetryNotify(true).into();
    let repeated_request: RespondActivityTaskCompletedRequest =
        binding.fill(&repeated, &world, &inputs, &identity)?.pack_request().unpack()?;
    assert_eq!(repeated_request.identity, "request-4-true");
    assert_eq!(binding.row(&repeated).variant, "RetryNotify");
    let event: specimen::Action = specimen::Environment::Refresh.into();
    assert!(matches!(
        binding.row(&event).kind,
        ActionKind::Event { reason: "explicit refresh observation" }
    ));
    assert!(binding.fill(&event, &world, &inputs, &identity).is_err());
    Ok(())
}

fn edge(
    world: &specimen::World,
    source: specimen::Model,
    action: specimen::Action,
) -> umpire_runtime::Evidence<specimen::World> {
    let (outcome, facts, target) = world
        .checked_step(&source, action.clone())
        .unwrap_or_else(|error| panic!("{error:?}"))
        .remove(0);
    umpire_runtime::Evidence { source, action, choice: 0, outcome, facts, target }
}

#[test]
fn evidence_decodes_actual_types_and_preserves_the_complete_checked_prefix() -> Result<(), Refusal>
{
    use umpire_runtime::{
        EvidenceMatch,
        realization::{Observation, observation::Result as ResultValue},
    };
    let binding = Binding::default();
    let world = specimen::World::new(4);
    let client = edge(&world, world.starts().remove(0), specimen::Client::Ping.into());
    let worker =
        edge(&world, client.target.clone(), specimen::Worker::Notify { flag: true }.into());
    let request = binding
        .fill(&worker.action, &world, &Inputs { request_id: "request".into() }, &id())?
        .pack_request();
    let observation = Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RespondActivityTaskCompleted"
            .into(),
        request: Some(request),
        result: Some(ResultValue::Response(Any::pack(&RespondActivityTaskCompletedResponse {}))),
    };
    assert!(matches!(
        binding.reach_evidence(&Any::pack(&observation), &[&client, &worker])?,
        EvidenceMatch::Match { cursor: 2 }
    ));
    assert!(matches!(
        binding.reach_evidence(&Any::pack(&observation), &[&worker])?,
        EvidenceMatch::Reject
    ));
    let before_malformed = binding.calls.get();
    for bad in [
        Any::default(),
        Any::pack(&RequestCancelActivityExecutionRequest::default()),
        Any {
            type_url: observation
                .request
                .as_ref()
                .unwrap_or_else(|| panic!("request"))
                .type_url
                .clone(),
            value: vec![255],
        },
    ] {
        let mut wrong = observation.clone();
        wrong.request = Some(bad);
        assert!(binding.reach_evidence(&Any::pack(&wrong), &[&client, &worker]).is_err());
        assert!(binding.reach_evidence(&Any::pack(&wrong), &[]).is_err());
    }
    assert_eq!(binding.calls.get(), before_malformed);
    for token in [vec![], vec![8]] {
        let mut crossed = observation.clone();
        crossed.request = Some(Any::pack(&RespondActivityTaskCompletedRequest {
            namespace: "ns".into(),
            task_token: token,
            ..Default::default()
        }));
        assert!(binding.reach_evidence(&Any::pack(&crossed), &[&client, &worker]).is_err());
    }
    let before_malformed = binding.calls.get();
    let mut absent = observation.clone();
    absent.request = None;
    assert!(binding.reach_evidence(&Any::pack(&absent), &[&client, &worker]).is_err());
    let mut wrong = observation.clone();
    wrong.result =
        Some(ResultValue::Response(Any::pack(&RequestCancelActivityExecutionResponse {})));
    assert!(binding.reach_evidence(&Any::pack(&wrong), &[&client, &worker]).is_err());
    assert_eq!(binding.calls.get(), before_malformed);
    let mut error = observation;
    error.result = Some(ResultValue::Error(umpire_runtime::realization::Error {
        code: 5,
        message: String::new(),
    }));
    assert!(matches!(
        binding.reach_evidence(&Any::pack(&error), &[&client, &worker])?,
        EvidenceMatch::Reject
    ));
    let mut cancel = binding
        .fill(&client.action, &world, &Inputs { request_id: "request".into() }, &id())?
        .pack_request()
        .unpack::<RequestCancelActivityExecutionRequest>()?;
    cancel.run_id = "crossed".into();
    let crossed = Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        request: Some(Any::pack(&cancel)),
        result: Some(ResultValue::Response(Any::pack(&RequestCancelActivityExecutionResponse {}))),
    };
    assert!(binding.reach_evidence(&Any::pack(&crossed), &[&client]).is_err());
    cancel.run_id = "run".into();
    cancel.namespace.clear();
    let missing = Observation {
        method: crossed.method,
        request: Some(Any::pack(&cancel)),
        result: crossed.result,
    };
    assert!(binding.reach_evidence(&Any::pack(&missing), &[&client]).is_err());
    Ok(())
}

#[test]
fn typed_bootstrap_and_read_hooks_keep_correlation_outside_the_action_graph() -> Result<(), Refusal>
{
    let binding = Binding::default();
    let entity = Binding::entity();
    let request = StartActivityExecutionRequest {
        namespace: "ns".into(),
        activity_id: "entity".into(),
        request_id: "request".into(),
        ..Default::default()
    };
    let response = StartActivityExecutionResponse { run_id: "run".into(), ..Default::default() };
    let bootstrap: umpire_runtime::Proposal<specimen::Action> =
        umpire_runtime::Proposal::Bootstrap { request: Any::pack(&request) };
    assert!(matches!(bootstrap, umpire_runtime::Proposal::Bootstrap { .. }));
    let actual = (entity.admit)(&binding, &request, &response)?;
    let receipt = (entity.receipt)(&binding, &actual, &response)?;
    (entity.correlate)(&binding, &receipt, &request, &response)?;
    let mut crossed = response.clone();
    crossed.run_id = "crossed".into();
    assert!((entity.correlate)(&binding, &receipt, &request, &crossed).is_err());
    let read = Binding::read();
    let world = specimen::World::new(4);
    let current = (read.fill)(&binding, &world, &receipt)?;
    assert_eq!(
        (current.namespace.as_str(), current.activity_id.as_str(), current.run_id.as_str()),
        ("ns", "entity", "run")
    );
    let target = (read.project)(&binding, &DescribeActivityExecutionResponse::default())?;
    assert!(!(read.finality)(&binding, &target));
    assert!((read.continue_polling)(&binding, &target));
    let proposal: umpire_runtime::Proposal<specimen::Action> =
        umpire_runtime::Proposal::Read { request: Any::pack(&current) };
    assert!(matches!(proposal, umpire_runtime::Proposal::Read { .. }));
    let final_target = (read.project)(
        &binding,
        &DescribeActivityExecutionResponse {
            outcome: Some(
                umpire_temporal_api::temporal::api::activity::v1::ActivityExecutionOutcome::default(
                ),
            ),
            ..Default::default()
        },
    )?;
    assert!((read.finality)(&binding, &final_target));
    assert!(!(read.continue_polling)(&binding, &final_target));
    assert!(!world.starts()[0].ping);
    Ok(())
}

#[test]
fn registry_admission_uses_actual_inventory_descriptor_edges_and_rust_types() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/registry_*.rs");
    tests.pass("tests/ui-pass/registry_*.rs");
}
