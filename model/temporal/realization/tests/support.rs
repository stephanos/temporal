//! Controls for shared Temporal helpers and realization-only phase decoding.

use umpire_runtime::AnyMessage as _;
use umpire_runtime::realization::Callback;
use umpire_runtime::realization::Error;
use umpire_runtime::realization::Observation;
use umpire_runtime::realization::Reply;
use umpire_runtime::realization::observation::Result as ResultValue;

use std::collections::BTreeMap;

use umpire_dsl::Rejection;
use umpire_runtime::{
    prost_types::Any,
    protocol::{Refusal, RefusalCode},
};
use umpire_temporal::support::{
    ApplicationFailure, Observed, WorkflowService, callback, method, plain_text, rejection, reply,
};
use umpire_temporal_api::temporal::api::{
    common::v1::Payload,
    failure::v1::{Failure, failure::FailureInfo},
};

fn code<T>(result: Result<T, Refusal>) -> RefusalCode {
    result.err().unwrap_or_else(|| panic!("expected refusal")).code()
}

#[test]
fn observation_admission_preserves_envelope_and_result_precedence() {
    let malformed = RefusalCode::MalformedRequest;
    let unsupported = RefusalCode::Unsupported;
    assert_eq!(code(Observed::new(&Any::default())), malformed);
    assert_eq!(code(Observed::new(&Any::pack(&Reply::default()))), unsupported);
    assert_eq!(code(Observed::new(&Any::pack(&Observation::default()))), malformed);
    let observed = Observed::new(&Any::pack(&Observation {
        method: "unknown".into(),
        result: None,
        request: None,
    }))
    .unwrap_or_else(|e| panic!("{e:?}"));
    assert_eq!(observed.method, "unknown");
    assert_eq!(code(observed.response::<Reply>()), malformed);
    assert_eq!(code(observed.code()), malformed);
    for (result, expected) in [
        (ResultValue::Response(Any::default()), malformed),
        (ResultValue::Response(Any::pack(&Callback::default())), unsupported),
        (ResultValue::Error(Error { code: 5, message: String::new() }), malformed),
    ] {
        let observed = Observed::new(&Any::pack(&Observation {
            method: "m".into(),
            result: Some(result),
            request: None,
        }))
        .unwrap_or_else(|e| panic!("{e:?}"));
        assert_eq!(code(observed.response::<Reply>()), expected);
    }
}

#[test]
fn actual_request_is_optional_and_typed() {
    let observed = |request| {
        Observed::new(&Any::pack(&Observation {
            method: "m".into(),
            result: Some(ResultValue::Response(Any::pack(&Reply::default()))),
            request,
        }))
        .unwrap_or_else(|e| panic!("{e:?}"))
    };
    assert_eq!(observed(None).request::<Callback>().unwrap_or_else(|e| panic!("{e:?}")), None);
    let request = Callback { task: Some(Any::pack(&Reply::default())) };
    assert_eq!(
        observed(Some(Any::pack(&request)))
            .request::<Callback>()
            .unwrap_or_else(|e| panic!("{e:?}")),
        Some(request)
    );
    assert_eq!(
        code(observed(Some(Any::pack(&Reply::default()))).request::<Callback>()),
        RefusalCode::Unsupported
    );
    assert_eq!(
        code(observed(Some(Any::default())).request::<Callback>()),
        RefusalCode::MalformedRequest
    );
}

#[test]
fn response_and_grpc_codes_are_distinct() {
    for result in [
        ResultValue::Response(Any::pack(&Reply::default())),
        ResultValue::Error(Error { code: 5, message: String::new() }),
    ] {
        let is_error = matches!(result, ResultValue::Error(_));
        let observed = Observed::new(&Any::pack(&Observation {
            method: "m".into(),
            result: Some(result),
            request: None,
        }))
        .unwrap_or_else(|e| panic!("{e:?}"));
        assert_eq!(observed.code().unwrap_or_else(|e| panic!("{e:?}")), is_error.then_some(5));
        if !is_error {
            assert_eq!(
                observed.response::<Reply>().unwrap_or_else(|e| panic!("{e:?}")),
                Reply::default()
            );
        }
    }
    for (status, expected) in [
        (5, Rejection::NotFound),
        (6, Rejection::AlreadyExists),
        (9, Rejection::FailedPrecondition),
        (3, Rejection::InvalidArgument),
        (7, Rejection::PermissionDenied),
        (14, Rejection::Unavailable),
    ] {
        assert_eq!(rejection(status), Some(expected));
    }
    for status in [0, 1, -1, 999] {
        assert_eq!(rejection(status), None);
    }
}

#[test]
fn plain_payload_refusals_keep_metadata_precedence() {
    let malformed = RefusalCode::MalformedRequest;
    let unsupported = RefusalCode::Unsupported;
    for (metadata, expected, message) in [
        (vec![], malformed, "payload encoding is missing"),
        (vec![("extra", b"x".as_slice())], malformed, "payload encoding is missing"),
        (
            vec![("encoding", b"json/plain".as_slice()), ("extra", b"x")],
            malformed,
            "payload metadata must contain only encoding",
        ),
        (vec![("encoding", b"json/plain".as_slice())], unsupported, "unsupported payload encoding"),
    ] {
        let payload = Payload {
            metadata: metadata.into_iter().map(|(k, v)| (k.into(), v.to_vec())).collect(),
            ..Default::default()
        };
        let error = plain_text(&payload).err().unwrap_or_else(|| panic!("expected refusal"));
        assert_eq!(error.code(), expected);
        assert_eq!(error.message, message);
    }
    let payload = Payload {
        metadata: BTreeMap::from([("encoding".into(), b"binary/plain".to_vec())]),
        data: b"echo".to_vec(),
        ..Default::default()
    };
    assert_eq!(plain_text(&payload).unwrap_or_else(|e| panic!("{e:?}")), b"echo");
}

#[test]
fn callback_refusals_and_typed_reply_round_trip() {
    let malformed = RefusalCode::MalformedRequest;
    for (value, expected) in [
        (Any::default(), malformed),
        (Any::pack(&Reply::default()), RefusalCode::Unsupported),
        (Any::pack(&Callback::default()), malformed),
        (Any::pack(&Callback { task: Some(Any::default()) }), malformed),
        (
            Any::pack(&Callback { task: Some(Any::pack(&Observation::default())) }),
            RefusalCode::Unsupported,
        ),
    ] {
        assert_eq!(code(callback::<Reply>(&value)), expected);
    }
    let request = Reply::default();
    assert_eq!(
        callback::<Reply>(&Any::pack(&Callback { task: Some(Any::pack(&request)) }))
            .unwrap_or_else(|e| panic!("{e:?}")),
        request
    );
    assert_eq!(
        reply(&request).unpack::<Reply>().unwrap_or_else(|e| panic!("{e:?}")),
        Reply { request: Some(Any::pack(&request)) }
    );
    assert_eq!(
        method::<WorkflowService>("Describe"),
        "/temporal.api.workflowservice.v1.WorkflowService/Describe"
    );
    let failure = Failure::application("Kind", "message", true);
    assert_eq!(failure.message, "message");
    let Some(FailureInfo::ApplicationFailureInfo(info)) = failure.failure_info else {
        panic!("missing application failure")
    };
    assert_eq!(info.r#type, "Kind");
    assert!(info.non_retryable);
}

#[test]
fn phase_conversion_preserves_all_pairs_and_unknown_enum_precedence() {
    use umpire_temporal::activity::Phase;
    use umpire_temporal_api::temporal::api::activity::v1::ActivityExecutionInfo;

    for status in 0..=8 {
        for run_state in 0..=6 {
            let result =
                Phase::try_from(ActivityExecutionInfo { status, run_state, ..Default::default() });
            let expected = match (status, run_state) {
                (1, 1) => Ok(Phase::Scheduled),
                (1, 2) => Ok(Phase::Started),
                (2, 0) => Ok(Phase::Completed),
                (3, 0) => Ok(Phase::Failed),
                (4, 0) => Ok(Phase::Canceled),
                (6, 0) => Ok(Phase::TimedOut),
                (8, _) | (_, 6) => Err(RefusalCode::Unsupported),
                (0 | 2 | 3 | 4 | 6, _) | (1, 0) => Err(RefusalCode::MalformedRequest),
                _ => Err(RefusalCode::Unsupported),
            };
            assert_eq!(result.map_err(|error| error.code()), expected, "{status}/{run_state}");
        }
    }
}
