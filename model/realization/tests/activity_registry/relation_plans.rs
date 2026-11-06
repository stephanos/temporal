//! Go relation plans and Rust/Go novelty vectors regenerated from the actual Activity bindings.
//!
//! The committed Go package `hosts/go/activityrelations` must equal what the bindings emit now;
//! `UMPIRE_BLESS=1` rewrites it. The vectors record what the Rust check and projection decide for
//! fixed invocations, so the Go evaluator and projector are compared against the same inputs.
use super::super::relations::{Declared, cancel_binding, describe_binding, start_binding};
use super::ProposalContext;
use crate::foundation::dc::Limits;
use prost::Message as _;
use std::fmt::Write as _;
use std::path::{Path, PathBuf};
use umpire_catalog::{NoveltyError, NoveltyProjector, RpcContext};
use umpire_runtime::{AnyMessage as _, realization::observation::Result as ResultValue};
use umpire_temporal_api::temporal::api::{
    common::v1::{Link, link},
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse,
        StartActivityExecutionRequest, StartActivityExecutionResponse,
    },
};

mod common {
    include!(concat!(env!("CARGO_MANIFEST_DIR"), "/tests/common/mod.rs"));
}

const PACKAGE: &str = "activityrelations";
const OTHER: &str = "33333333-3333-4333-8333-333333333333";

/// Model refusals are not `std::error::Error`; keep their value-free debug text.
fn fail(error: impl std::fmt::Debug) -> Box<dyn std::error::Error> {
    format!("{error:?}").into()
}

fn directory() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("../hosts/go").join(PACKAGE)
}

fn plan<Q: 'static, S: 'static>(
    declared: &Declared<Q, S>,
    variable: &str,
) -> Result<String, Box<dyn std::error::Error>> {
    Ok(umpire_catalog_build::relations::generate_go_registered(
        declared.rpc.schema(),
        declared.rpc.catalogs(),
        declared.rpc.adapters(),
        PACKAGE,
        variable,
        None,
    )?)
}

struct Vector {
    name: &'static str,
    method: String,
    request: Vec<u8>,
    response: Vec<u8>,
    receipt_run_id: String,
    outcome: String,
}

fn hex(bytes: &[u8]) -> String {
    bytes.iter().fold(String::new(), |mut out, b| {
        let _ = write!(out, "{b:02x}");
        out
    })
}

/// Check and project one invocation through the Rust route the Go evaluator must reproduce.
fn vector<Q: prost::Message + 'static, S: prost::Message + 'static>(
    name: &'static str,
    declared: &Declared<Q, S>,
    receipt: &ProposalContext,
    request: &Q,
    response: &S,
    received: Vec<u8>,
) -> Result<Vector, Box<dyn std::error::Error>> {
    let projector = NoveltyProjector::new(&declared.rpc)?;
    let context = RpcContext { request, response, receipt: Some(receipt), policy: None };
    let outcome = match projector.check_and_project_encoded(&declared.rpc, &context, &received) {
        Ok((_, key)) => format!("key:{}", key.hex()),
        Err(NoveltyError::Correlation(error)) => format!(
            "correlation:{:?}:{:?}:{}",
            error.class,
            error.reason,
            error.declaration.as_deref().unwrap_or_default()
        ),
        Err(NoveltyError::Projection(refusal)) => format!("projection:{}", refusal.reason.as_str()),
    };
    Ok(Vector {
        name,
        method: declared.rpc.schema().method().method.clone(),
        request: request.encode_to_vec(),
        response: received,
        receipt_run_id: receipt.run_id.clone(),
        outcome,
    })
}

fn rerun(response: &mut StartActivityExecutionResponse, run: &str) {
    response.run_id = run.into();
    response.link = Some(Link {
        variant: Some(link::Variant::Activity(link::Activity {
            namespace: "model-slice".into(),
            activity_id: "activity".into(),
            run_id: run.into(),
        })),
    });
}

type Vectors = Result<Vec<Vector>, Box<dyn std::error::Error>>;

fn vectors() -> Vectors {
    let observed = common::start(3);
    let start_request: StartActivityExecutionRequest =
        observed.request.as_ref().ok_or("Start request")?.unpack().map_err(fail)?;
    let Some(ResultValue::Response(raw)) = &observed.result else { return Err("Start".into()) };
    let start_response: StartActivityExecutionResponse = raw.unpack().map_err(fail)?;
    let receipt = |run: &str| ProposalContext { start: start_request.clone(), run_id: run.into() };
    let (run, other) = (receipt(common::RUN), receipt(OTHER));
    let mut out = start_vectors(&run, &other, &start_request, &start_response)?;
    out.extend(describe_vectors(&run, &other)?);
    out.extend(cancel_vectors(&run, &other, &start_request)?);
    Ok(out)
}

fn start_vectors(
    run: &ProposalContext,
    other: &ProposalContext,
    start_request: &StartActivityExecutionRequest,
    start_response: &StartActivityExecutionResponse,
) -> Vectors {
    let (start_request, start_response) = (start_request.clone(), start_response.clone());
    let start = start_binding(&Limits::default()).map_err(fail)?;
    let mut out = Vec::new();
    let encoded = start_response.encode_to_vec();
    out.push(vector("start", &start, run, &start_request, &start_response, encoded)?);
    let mut root_only = start_response.clone();
    root_only.run_id = OTHER.into();
    let encoded = root_only.encode_to_vec();
    out.push(vector("start-other-run-root", &start, other, &start_request, &root_only, encoded)?);
    let mut rerun_response = start_response.clone();
    rerun(&mut rerun_response, OTHER);
    let encoded = rerun_response.encode_to_vec();
    out.push(vector(
        "start-other-run-link",
        &start,
        other,
        &start_request,
        &rerun_response,
        encoded,
    )?);
    let encoded = rerun_response.encode_to_vec();
    out.push(vector("start-crossed", &start, run, &start_request, &rerun_response, encoded)?);
    let mut unknown = start_response.encode_to_vec();
    unknown.extend([0xb8, 0x3e, 0x01]);
    out.push(vector("start-unknown-field", &start, run, &start_request, &start_response, unknown)?);
    let mut not_started = start_response.clone();
    not_started.started = false;
    let encoded = not_started.encode_to_vec();
    out.push(vector("start-not-started", &start, run, &start_request, &not_started, encoded)?);
    Ok(out)
}

fn describe_vectors(run: &ProposalContext, other: &ProposalContext) -> Vectors {
    let describe = describe_binding(&Limits::default()).map_err(fail)?;
    let mut out = Vec::new();
    let describe_case = |attempt, status| -> Result<_, Box<dyn std::error::Error>> {
        let observed = common::describe(attempt, status, 1);
        let request: DescribeActivityExecutionRequest =
            observed.request.as_ref().ok_or("Describe request")?.unpack().map_err(fail)?;
        let Some(ResultValue::Response(raw)) = &observed.result else {
            return Err("Describe".into());
        };
        let response: DescribeActivityExecutionResponse = raw.unpack().map_err(fail)?;
        Ok((request, response))
    };
    let (request, response) = describe_case(1, 1)?;
    let encoded = response.encode_to_vec();
    out.push(vector("describe", &describe, run, &request, &response, encoded)?);
    let (mut other_request, mut other_response) = (request.clone(), response.clone());
    other_request.run_id = OTHER.into();
    other_response.run_id = OTHER.into();
    let encoded = other_response.encode_to_vec();
    out.push(vector(
        "describe-other-run-root",
        &describe,
        other,
        &other_request,
        &other_response,
        encoded,
    )?);
    if let Some(info) = other_response.info.as_mut() {
        info.run_id = OTHER.into();
    }
    let encoded = other_response.encode_to_vec();
    out.push(vector(
        "describe-other-run-everywhere",
        &describe,
        other,
        &other_request,
        &other_response,
        encoded,
    )?);
    let encoded = response.encode_to_vec();
    out.push(vector(
        "describe-crossed-request",
        &describe,
        run,
        &other_request,
        &response,
        encoded,
    )?);
    let (request2, response2) = describe_case(2, 1)?;
    let encoded = response2.encode_to_vec();
    out.push(vector("describe-attempt-2", &describe, run, &request2, &response2, encoded)?);
    let (request3, response3) = describe_case(1, 2)?;
    let encoded = response3.encode_to_vec();
    out.push(vector("describe-completed", &describe, run, &request3, &response3, encoded)?);
    Ok(out)
}

fn cancel_vectors(
    run: &ProposalContext,
    other: &ProposalContext,
    start_request: &StartActivityExecutionRequest,
) -> Vectors {
    let cancel = cancel_binding(&Limits::default()).map_err(fail)?;
    let start_request = start_request.clone();
    let mut out = Vec::new();
    let cancel_request = RequestCancelActivityExecutionRequest {
        namespace: start_request.namespace.clone(),
        activity_id: start_request.activity_id.clone(),
        run_id: common::RUN.into(),
        ..Default::default()
    };
    let cancel_response = RequestCancelActivityExecutionResponse::default();
    let encoded = cancel_response.encode_to_vec();
    out.push(vector("cancel", &cancel, run, &cancel_request, &cancel_response, encoded)?);
    let encoded = cancel_response.encode_to_vec();
    out.push(vector("cancel-crossed", &cancel, other, &cancel_request, &cancel_response, encoded)?);
    Ok(out)
}

fn render(vectors: &[Vector]) -> String {
    let mut out = String::from(
        "{\n  \"format\": \"umpire.temporal.activity-relations.vectors.v1\",\n  \"vectors\": [\n",
    );
    for (index, v) in vectors.iter().enumerate() {
        let _ = write!(
            out,
            "    {{\"name\": \"{}\", \"method\": \"{}\", \"receipt_run_id\": \"{}\", \"outcome\": \"{}\",\n     \"request_hex\": \"{}\",\n     \"response_hex\": \"{}\"}}{}\n",
            v.name,
            v.method,
            v.receipt_run_id,
            v.outcome,
            hex(&v.request),
            hex(&v.response),
            if index + 1 == vectors.len() { "" } else { "," }
        );
    }
    out.push_str("  ]\n}\n");
    out
}

fn expected() -> Result<Vec<(&'static str, Vec<u8>)>, Box<dyn std::error::Error>> {
    let limits = Limits::default();
    Ok(vec![
        ("start_plan.go", plan(&start_binding(&limits).map_err(fail)?, "StartPlan")?.into_bytes()),
        (
            "describe_plan.go",
            plan(&describe_binding(&limits).map_err(fail)?, "DescribePlan")?.into_bytes(),
        ),
        (
            "cancel_plan.go",
            plan(&cancel_binding(&limits).map_err(fail)?, "CancelPlan")?.into_bytes(),
        ),
        ("relation-api.binpb", umpire_temporal_api::RELATION_DESCRIPTOR_BYTES.to_vec()),
        ("testdata/vectors.json", render(&vectors()?).into_bytes()),
    ])
}

#[test]
fn go_relation_plans_and_vectors_are_fresh() -> Result<(), Box<dyn std::error::Error>> {
    let bless = std::env::var_os("UMPIRE_BLESS").is_some_and(|v| v == "1");
    let mut stale = Vec::new();
    for (name, bytes) in expected()? {
        let path = directory().join(name);
        if bless {
            std::fs::create_dir_all(path.parent().ok_or("parent")?)?;
            std::fs::write(&path, &bytes)?;
        } else if std::fs::read(&path).ok().as_deref() != Some(bytes.as_slice()) {
            stale.push(name);
        }
    }
    assert!(stale.is_empty(), "stale {PACKAGE} files {stale:?}; regenerate with UMPIRE_BLESS=1");
    Ok(())
}

#[test]
fn rust_vectors_collapse_only_echoed_run_identities() -> Result<(), Box<dyn std::error::Error>> {
    let vectors = vectors()?;
    let outcome = |name: &str| {
        vectors.iter().find(|v| v.name == name).map(|v| v.outcome.as_str()).unwrap_or_default()
    };
    assert!(outcome("start").starts_with("key:"));
    assert_eq!(outcome("start"), outcome("start-other-run-root"));
    // The Link's run_id has no declaration yet, so it stays a distinguishing value.
    assert_ne!(outcome("start"), outcome("start-other-run-link"));
    assert_ne!(outcome("start"), outcome("start-not-started"));
    assert!(outcome("start-crossed").starts_with("correlation:State:EchoMismatch:"));
    assert_eq!(outcome("start-unknown-field"), "projection:NoveltyUnknownField");
    assert_eq!(outcome("describe"), outcome("describe-other-run-root"));
    // Info's run_id has no declaration yet, so it stays a distinguishing value.
    assert_ne!(outcome("describe"), outcome("describe-other-run-everywhere"));
    assert!(outcome("describe-crossed-request").starts_with("correlation:State:EchoMismatch:"));
    assert_ne!(outcome("describe"), outcome("describe-attempt-2"));
    assert_ne!(outcome("describe"), outcome("describe-completed"));
    assert!(outcome("cancel").starts_with("key:"));
    assert!(outcome("cancel-crossed").starts_with("correlation:State:EchoMismatch:"));
    Ok(())
}
