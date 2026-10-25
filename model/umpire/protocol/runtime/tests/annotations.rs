//! Stateless annotation routing and transactional delegation use one runtime adapter.
#[path = "support/control.rs"]
mod control;
#[allow(dead_code, reason = "shared control support includes unrelated operations")]
mod support;

use prost::{Message, Name};
use umpire_runtime::{
    AnnotationRuntime, AnyMessage as _, Dispatch,
    annotations::{AnnotationRequest, AnnotationResponse, ProbeResponse},
    prost_types::Any,
    protocol::{self, Method, RefusalCode, response::Outcome},
};

#[test]
fn typed_probes_and_annotations_share_stateless_routing() {
    let model = control::control();
    let runtime = AnnotationRuntime::with_probes(
        control::control(),
        |_: &protocol::ExploreRequest| Ok(ProbeResponse::default()),
        |request| Ok(AnnotationResponse { message: request.message.clone(), ..Default::default() }),
    );
    let reset = protocol::ResetRequest::default().encode_to_vec();
    let (expected, transaction) = model.prepare(Method::Reset as u32, &reset);
    let (actual, delegated) = runtime.prepare(Method::Reset as u32, &reset);
    assert_eq!(actual, expected);
    assert_eq!(delegated.is_some(), transaction.is_some());
    for input in [
        Any::pack(&protocol::ExploreRequest::default()),
        Any::pack(&AnnotationRequest { message: vec![1, 2, 3], ..Default::default() }),
    ] {
        let probe = input.type_url == protocol::ExploreRequest::type_url();
        let request = protocol::AnnotateRequest { input: Some(input) }.encode_to_vec();
        let (response, transaction) = runtime.prepare(Method::Annotate as u32, &request);
        assert!(transaction.is_none());
        let Some(Outcome::Annotate(response)) = response.outcome else {
            panic!("expected annotation output");
        };
        let output = response.output.unwrap_or_else(|| panic!("missing output"));
        if probe {
            assert_eq!(output.unpack::<ProbeResponse>(), Ok(ProbeResponse::default()));
        } else {
            assert_eq!(output.unpack::<AnnotationResponse>().map(|r| r.message), Ok(vec![1, 2, 3]));
        }
    }
    for (bytes, code) in [
        (vec![255], RefusalCode::MalformedRequest),
        (protocol::AnnotateRequest::default().encode_to_vec(), RefusalCode::MalformedRequest),
        (
            protocol::AnnotateRequest { input: Some(Any::pack(&ProbeResponse::default())) }
                .encode_to_vec(),
            RefusalCode::Unsupported,
        ),
        (
            protocol::AnnotateRequest {
                input: Some(Any {
                    type_url: protocol::ExploreRequest::type_url(),
                    value: vec![255],
                }),
            }
            .encode_to_vec(),
            RefusalCode::MalformedRequest,
        ),
    ] {
        let (response, transaction) = runtime.prepare(Method::Annotate as u32, &bytes);
        assert!(transaction.is_none());
        assert!(matches!(response.outcome, Some(Outcome::Refusal(r)) if r.code() == code));
    }
}
