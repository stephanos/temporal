use super::*;
use umpire_catalog::{
    RelationCheckError, RelationFailureClass, RelationLocation, RelationReason, RelationStage,
    WitnessKind,
};
use umpire_runtime::{
    AnyMessage as _, protocol::RefusalCode, realization::observation::Result as ResultValue,
};

mod common {
    include!(concat!(env!("CARGO_MANIFEST_DIR"), "/tests/common/mod.rs"));
}

const OTHER: &str = "33333333-3333-4333-8333-333333333333";
const SERVICE: &str = "temporal.api.workflowservice.v1.WorkflowService";

/// The admitted run-identity inventory; a removed or added declaration fails here first.
const DECLARATIONS: [(&str, RelationRoot, RelationRoot); 4] = [
    ("StartActivityExecution", RelationRoot::Response, RelationRoot::Receipt),
    ("DescribeActivityExecution", RelationRoot::Request, RelationRoot::Receipt),
    ("DescribeActivityExecution", RelationRoot::Response, RelationRoot::Request),
    ("RequestCancelActivityExecution", RelationRoot::Request, RelationRoot::Receipt),
];

fn root(root: RelationRoot) -> &'static str {
    match root {
        RelationRoot::Request => "request",
        RelationRoot::Response => "response",
        RelationRoot::Receipt => "receipt",
    }
}

fn identity(method: &str, target: RelationRoot, source: RelationRoot) -> String {
    format!(
        "{SERVICE}/{method}/{}:temporal.activity.ProposalContext/run_id:Echo:{}.run_id",
        root(target),
        root(source)
    )
}

struct Fixture {
    receipt: ProposalContext,
    start: (StartActivityExecutionRequest, StartActivityExecutionResponse),
    describe: (DescribeActivityExecutionRequest, DescribeActivityExecutionResponse),
    cancel: (RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse),
}

fn fixture() -> Result<Fixture, Refusal> {
    let start = common::start(3);
    let request: StartActivityExecutionRequest = required(start.request.as_ref())?.unpack()?;
    let Some(ResultValue::Response(raw)) = &start.result else { panic!("Start response") };
    let response: StartActivityExecutionResponse = raw.unpack()?;
    let receipt = ProposalContext { start: request.clone(), run_id: common::RUN.into() };
    let describe = common::describe(1, 1, 1);
    let describe_request: DescribeActivityExecutionRequest =
        required(describe.request.as_ref())?.unpack()?;
    let Some(ResultValue::Response(raw)) = &describe.result else { panic!("Describe response") };
    let describe_response: DescribeActivityExecutionResponse = raw.unpack()?;
    let cancel = RequestCancelActivityExecutionRequest {
        namespace: request.namespace.clone(),
        activity_id: request.activity_id.clone(),
        run_id: common::RUN.into(),
        ..Default::default()
    };
    Ok(Fixture {
        receipt,
        start: (request, response),
        describe: (describe_request, describe_response),
        cancel: (cancel, RequestCancelActivityExecutionResponse::default()),
    })
}

fn context<'a, Q, S>(
    receipt: &'a ProposalContext,
    (request, response): &'a (Q, S),
) -> RpcContext<'a, Q, S, ProposalContext, ()> {
    RpcContext { request, response, receipt: Some(receipt), policy: None }
}

fn required(
    value: Option<&umpire_runtime::prost_types::Any>,
) -> Result<&umpire_runtime::prost_types::Any, Refusal> {
    value.ok_or_else(|| Refusal::malformed("fixture request is missing"))
}

fn declared<Q: 'static, S: 'static>(rpc: &BoundRpc<Q, S, ProposalContext, ()>) -> Vec<String> {
    rpc.schema().relations().iter().map(|relation| relation.identity().to_owned()).collect()
}

fn refused<T>(result: Result<T, RelationCheckError>) -> RelationCheckError {
    match result {
        Ok(_) => panic!("generated relation must refuse"),
        Err(error) => error,
    }
}

fn assert_echo(
    error: &RelationCheckError,
    method: &str,
    target: RelationRoot,
    source: RelationRoot,
) {
    let rpc = format!("{SERVICE}/{method}");
    assert_eq!(error.rpc, rpc);
    assert_eq!(error.declaration.as_deref(), Some(identity(method, target, source).as_str()));
    assert_eq!(error.class, RelationFailureClass::State);
    assert_eq!(error.stage, RelationStage::Compare);
    assert_eq!(error.reason, RelationReason::EchoMismatch);
    assert_eq!(error.target, Some(RelationLocation { root: target, path: "run_id".into() }));
    assert_eq!(error.source, Some(RelationLocation { root: source, path: "run_id".into() }));
    assert_eq!(
        error.expected.as_ref().map(umpire_catalog::SafeRelationSummary::text),
        Some(format!("{:?}", common::RUN).as_str())
    );
    assert_eq!(
        error.actual.as_ref().map(umpire_catalog::SafeRelationSummary::text),
        Some(format!("{OTHER:?}").as_str())
    );
}

#[test]
fn selected_activity_bindings_keep_typed_generated_diagnostics() -> Result<(), Refusal> {
    let fixture = fixture()?;
    let receipt = &fixture.receipt;
    let mut start = fixture.start.clone();
    start.1.run_id = OTHER.into();
    let error = refused(start_binding(&Limits::default())?.rpc.check(&context(receipt, &start)));
    assert_echo(&error, "StartActivityExecution", RelationRoot::Response, RelationRoot::Receipt);
    let describe = describe_binding(&Limits::default())?;
    let mut crossed = fixture.describe.clone();
    crossed.0.run_id = OTHER.into();
    let error = refused(describe.rpc.check(&context(receipt, &crossed)));
    assert_echo(&error, "DescribeActivityExecution", RelationRoot::Request, RelationRoot::Receipt);
    let mut crossed = fixture.describe.clone();
    crossed.1.run_id = OTHER.into();
    let error = refused(describe.rpc.check(&context(receipt, &crossed)));
    assert_echo(&error, "DescribeActivityExecution", RelationRoot::Response, RelationRoot::Request);
    let mut cancel = fixture.cancel.clone();
    cancel.0.run_id = OTHER.into();
    let error = refused(cancel_binding(&Limits::default())?.rpc.check(&context(receipt, &cancel)));
    assert_echo(
        &error,
        "RequestCancelActivityExecution",
        RelationRoot::Request,
        RelationRoot::Receipt,
    );
    let refusal = crate::registry::relations::refusal(&error);
    assert_eq!(refusal.code(), RefusalCode::State);
    Ok(())
}

#[test]
#[expect(
    clippy::result_large_err,
    reason = "witness closures return the framework's located report"
)]
fn every_admitted_run_identity_declaration_mints_an_isolated_witness() -> Result<(), Refusal> {
    let fixture = fixture()?;
    let receipt = &fixture.receipt;
    let mismatch = Raw::Present(String::from(OTHER));
    let mut witnessed = Vec::new();
    let mut admitted = Vec::new();
    let mut prove = |rpc: &dyn Fn(
        &DeclarationId<String>,
    )
        -> Result<umpire_catalog::WitnessReceipt, RelationCheckError>,
                     ids: &[DeclarationId<String>],
                     declared: Vec<String>| {
        admitted.extend(declared);
        for id in ids {
            let witness = rpc(id).unwrap_or_else(|error| panic!("{error}"));
            assert_eq!(witness.declaration(), id.identity());
            assert_eq!(witness.kind(), WitnessKind::EchoMismatch);
            assert_eq!(witness.reason(), RelationReason::EchoMismatch);
            assert_eq!(witness.target().path, "run_id");
            witnessed.push((
                witness.method().to_owned(),
                witness.declaration().to_owned(),
                witness.target().root,
            ));
        }
    };
    let start = start()?;
    prove(
        &|id| start.rpc.prepare_echo_witness(id, &context(receipt, &fixture.start), &mismatch),
        &start.run_ids,
        declared(&start.rpc),
    );
    let describe = describe()?;
    prove(
        &|id| {
            describe.rpc.prepare_echo_witness(id, &context(receipt, &fixture.describe), &mismatch)
        },
        &describe.run_ids,
        declared(&describe.rpc),
    );
    let cancel = cancel()?;
    prove(
        &|id| cancel.rpc.prepare_echo_witness(id, &context(receipt, &fixture.cancel), &mismatch),
        &cancel.run_ids,
        declared(&cancel.rpc),
    );
    let expected: Vec<_> = DECLARATIONS
        .iter()
        .map(|(method, target, source)| identity(method, *target, *source))
        .collect();
    assert_eq!(admitted, expected, "every admitted declaration is selected exactly once");
    assert_eq!(
        witnessed.iter().map(|(_, declaration, _)| declaration.clone()).collect::<Vec<_>>(),
        expected,
    );
    for ((method, declaration, target), (name, root, _)) in witnessed.iter().zip(DECLARATIONS) {
        assert!(method.ends_with(name), "{method} {declaration}");
        assert_eq!(*target, root);
    }
    let error = refused(cancel.rpc.prepare_echo_witness(
        &cancel.run_ids[0],
        &context(receipt, &fixture.cancel),
        &Raw::Present(String::from(common::RUN)),
    ));
    assert_eq!(
        (error.stage, error.reason),
        (RelationStage::Prepare, RelationReason::NoIsolatingWitness)
    );
    Ok(())
}

#[test]
fn empty_admitted_run_identity_never_matches_an_empty_target() -> Result<(), Refusal> {
    let fixture = fixture()?;
    let receipt = ProposalContext { run_id: String::new(), ..fixture.receipt.clone() };
    let mut cancel = fixture.cancel.clone();
    cancel.0.run_id = String::new();
    let error = refused(cancel_binding(&Limits::default())?.rpc.check(&context(&receipt, &cancel)));
    assert_eq!(
        (error.class, error.stage, error.reason),
        (RelationFailureClass::Unsupported, RelationStage::Expect, RelationReason::EmptyIdentity)
    );
    assert_eq!(crate::registry::relations::refusal(&error).code(), RefusalCode::Unsupported);
    let mut request = RequestCancelActivityExecutionRequest::default();
    let Err(refusal) = fill_request(&*super::cancel()?, &receipt, &mut request) else {
        panic!("an empty admitted identity cannot be filled");
    };
    assert_eq!(refusal.code(), RefusalCode::Unsupported);
    assert_eq!(request.run_id, "");
    Ok(())
}

#[test]
fn fills_follow_actual_requests_and_interleaved_receipts() -> Result<(), Refusal> {
    let fixture = fixture()?;
    let first = fixture.receipt.clone();
    let second = ProposalContext { run_id: OTHER.into(), ..fixture.receipt.clone() };
    let describe = describe()?;
    for receipt in [&first, &second, &first] {
        let mut request = DescribeActivityExecutionRequest {
            run_id: String::new(),
            ..fixture.describe.0.clone()
        };
        fill_request(&*describe, receipt, &mut request)?;
        assert_eq!(request.run_id, receipt.run_id);
        let filler = super::super::Realization::default().describe_request(
            &super::super::activity::World::new(3, None, None, None, None),
            receipt,
        )?;
        assert_eq!(filler.run_id, receipt.run_id, "registry filler shares the checker expectation");
        request.include_last_failure = !filler.include_last_failure;
        assert_ne!(request, filler, "the captured request differs from the filler baseline");
        // The captured request, not the filler baseline, drives the response expectation.
        let mut response = fixture.describe.1.clone();
        let outcome = describe
            .rpc
            .fill_response(&request, Some(receipt), None, &mut response)
            .unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(response.run_id, receipt.run_id);
        assert_eq!(outcome.filled().len(), 1);
        let expected = outcome.results().expected(&describe.run_ids[1]).map_err(admission)?;
        assert_eq!(expected.value, Raw::Present(receipt.run_id.clone()));
        assert!(check(&*describe, receipt, &request, &response).is_ok());
        let crossed = if receipt.run_id == second.run_id { &first } else { &second };
        let error = refused(describe.rpc.check(&RpcContext {
            request: &request,
            response: &response,
            receipt: Some(crossed),
            policy: None,
        }));
        assert_eq!(error.reason, RelationReason::EchoMismatch);
        assert_eq!(
            error.target,
            Some(RelationLocation { root: RelationRoot::Request, path: "run_id".into() })
        );
    }
    Ok(())
}
