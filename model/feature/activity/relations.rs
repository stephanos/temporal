//! The selected Activity run identity relations, bound to actual invocation and receipt types.

use super::{ProposalContext, catalog};
use crate::foundation::dc::Limits;
use std::{cell::OnceCell, rc::Rc, thread::LocalKey};
use umpire_catalog::{
    BindingBuilder, BoundRpc, DeclarationId, EchoPolicies, Observed, Raw, ReceiptSchema,
    RelationAdmission, RelationRoot, RelationSemanticKind, RpcContext, ScalarReceiptSlot, Source,
    Target, ValueWitness,
};
use umpire_runtime::protocol::Refusal;
use umpire_temporal_api::temporal::api::workflowservice::v1::{
    DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
    RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse,
    StartActivityExecutionRequest, StartActivityExecutionResponse,
};

/// One admitted RPC binding with the typed identities of its run-ID declarations.
pub(super) struct Declared<Q, S> {
    pub(super) rpc: BoundRpc<Q, S, ProposalContext, ()>,
    #[cfg_attr(
        not(test),
        expect(
            dead_code,
            reason = "witness controls select declarations through these identities"
        )
    )]
    pub(super) run_ids: Vec<DeclarationId<String>>,
}

type Start = Declared<StartActivityExecutionRequest, StartActivityExecutionResponse>;
type Describe = Declared<DescribeActivityExecutionRequest, DescribeActivityExecutionResponse>;
type Cancel =
    Declared<RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse>;

thread_local! {
    static START: OnceCell<Rc<Start>> = const { OnceCell::new() };
    static DESCRIBE: OnceCell<Rc<Describe>> = const { OnceCell::new() };
    static CANCEL: OnceCell<Rc<Cancel>> = const { OnceCell::new() };
}

/// Admission decodes the linked descriptor set once per guest thread, not per observation.
fn cached<T: 'static>(
    key: &'static LocalKey<OnceCell<Rc<T>>>,
    build: impl FnOnce(&Limits) -> Result<T, Refusal>,
) -> Result<Rc<T>, Refusal> {
    key.with(|cell| {
        if let Some(value) = cell.get() {
            return Ok(Rc::clone(value));
        }
        let value = Rc::new(build(&Limits::default())?);
        Ok(Rc::clone(cell.get_or_init(|| value)))
    })
}

pub(super) fn start() -> Result<Rc<Start>, Refusal> {
    cached(&START, start_binding)
}

pub(super) fn describe() -> Result<Rc<Describe>, Refusal> {
    cached(&DESCRIBE, describe_binding)
}

pub(super) fn cancel() -> Result<Rc<Cancel>, Refusal> {
    cached(&CANCEL, cancel_binding)
}

type RunIdSlot = ScalarReceiptSlot<ProposalContext, String>;

fn receipt() -> Result<(ReceiptSchema<ProposalContext>, RunIdSlot), RelationAdmission> {
    let mut schema = ReceiptSchema::record("temporal.activity.ProposalContext");
    let run_id = schema.scalar(
        "run_id",
        RelationSemanticKind::Uuid,
        ValueWitness::scalar(),
        |receipt: &ProposalContext| Observed::present(receipt.run_id.clone()),
    )?;
    Ok((schema, run_id))
}

fn admission(error: impl std::fmt::Display) -> Refusal {
    Refusal::unsupported(format!("Activity relation binding admission: {error}"))
}

macro_rules! builder {
    ($limits:expr, $method:literal, $request:ident, $response:ident, $q:ty, $s:ty) => {{
        let (receipt, slot) = receipt().map_err(admission)?;
        let builder = BindingBuilder::<$q, $s, ProposalContext, ()>::for_rpc(
            &crate::activity_api::relation_descriptors().map_err(admission)?,
            concat!("temporal.api.workflowservice.v1.WorkflowService/", $method),
            catalog::$request($limits).map_err(admission)?,
            catalog::$response($limits).map_err(admission)?,
            Some(receipt),
        )
        .map_err(admission)?;
        (builder, slot)
    }};
}

pub(super) fn start_binding(limits: &Limits) -> Result<Start, Refusal> {
    let (mut binding, receipt) = builder!(
        limits,
        "StartActivityExecution",
        start_activity_execution_request,
        start_activity_execution_response,
        StartActivityExecutionRequest,
        StartActivityExecutionResponse
    );
    let slot = binding.slot(RelationRoot::Response, "run_id").map_err(admission)?;
    let run_id = binding
        .echo_checked(
            Source::receipt_slot(&slot, &receipt).map_err(admission)?,
            Target::response(
                &slot,
                ValueWitness::scalar(),
                |response| Observed::present(response.run_id.clone()),
                |response, value| {
                    response.run_id = present(value)?;
                    Ok(())
                },
            )
            .map_err(admission)?,
            EchoPolicies::default(),
        )
        .map_err(admission)?;
    Ok(Declared { rpc: binding.build().map_err(admission)?, run_ids: vec![run_id] })
}

pub(super) fn describe_binding(limits: &Limits) -> Result<Describe, Refusal> {
    let (mut binding, receipt) = builder!(
        limits,
        "DescribeActivityExecution",
        describe_activity_execution_request,
        describe_activity_execution_response,
        DescribeActivityExecutionRequest,
        DescribeActivityExecutionResponse
    );
    let request_slot = binding.slot(RelationRoot::Request, "run_id").map_err(admission)?;
    let request_run_id = binding
        .echo_checked(
            Source::receipt_slot(&request_slot, &receipt).map_err(admission)?,
            Target::request(
                &request_slot,
                ValueWitness::scalar(),
                |request| Observed::present(request.run_id.clone()),
                |request, value| {
                    request.run_id = present(value)?;
                    Ok(())
                },
            )
            .map_err(admission)?,
            EchoPolicies::default(),
        )
        .map_err(admission)?;
    let response_slot = binding.slot(RelationRoot::Response, "run_id").map_err(admission)?;
    let response_run_id = binding
        .echo_checked(
            Source::request(&response_slot, ValueWitness::scalar(), |request| {
                Observed::present(request.run_id.clone())
            })
            .map_err(admission)?,
            Target::response(
                &response_slot,
                ValueWitness::scalar(),
                |response| Observed::present(response.run_id.clone()),
                |response, value| {
                    response.run_id = present(value)?;
                    Ok(())
                },
            )
            .map_err(admission)?,
            EchoPolicies::default(),
        )
        .map_err(admission)?;
    Ok(Declared {
        rpc: binding.build().map_err(admission)?,
        run_ids: vec![request_run_id, response_run_id],
    })
}

pub(super) fn cancel_binding(limits: &Limits) -> Result<Cancel, Refusal> {
    let (mut binding, receipt) = builder!(
        limits,
        "RequestCancelActivityExecution",
        request_cancel_activity_execution_request,
        request_cancel_activity_execution_response,
        RequestCancelActivityExecutionRequest,
        RequestCancelActivityExecutionResponse
    );
    let slot = binding.slot(RelationRoot::Request, "run_id").map_err(admission)?;
    let run_id = binding
        .echo_checked(
            Source::receipt_slot(&slot, &receipt).map_err(admission)?,
            Target::request(
                &slot,
                ValueWitness::scalar(),
                |request| Observed::present(request.run_id.clone()),
                |request, value| {
                    request.run_id = present(value)?;
                    Ok(())
                },
            )
            .map_err(admission)?,
            EchoPolicies::default(),
        )
        .map_err(admission)?;
    Ok(Declared { rpc: binding.build().map_err(admission)?, run_ids: vec![run_id] })
}

fn present(value: Raw<String>) -> Result<String, RelationAdmission> {
    match value {
        Raw::Present(value) => Ok(value),
        Raw::Absent => Err(RelationAdmission::new("missing run identity", "run_id")),
    }
}

/// Check one actual successful invocation against its admitted receipt.
pub(super) fn check<Q: 'static, S: 'static>(
    binding: &Declared<Q, S>,
    receipt: &ProposalContext,
    request: &Q,
    response: &S,
) -> Result<crate::registry::relations::CheckedRelations, Refusal> {
    crate::registry::relations::check(
        &binding.rpc,
        &RpcContext { request, response, receipt: Some(receipt), policy: None },
    )
}

/// Write the receipt-held run identity through the same expectation the checker derives.
pub(super) fn fill_request<Q: 'static, S: Default + 'static>(
    binding: &Declared<Q, S>,
    receipt: &ProposalContext,
    request: &mut Q,
) -> Result<(), Refusal> {
    binding
        .rpc
        .fill_request(Some(receipt), None, request)
        .map(|_| ())
        .map_err(|error| crate::registry::relations::refusal(&error))
}

#[cfg(test)]
#[path = "relations_tests.rs"]
mod tests;
