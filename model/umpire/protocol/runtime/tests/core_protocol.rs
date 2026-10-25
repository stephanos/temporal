//! Generic operation encoding and staged dispatch contract.

#[allow(dead_code, reason = "wire vectors use only the shared hex decoder")]
mod support;

#[path = "../guests/stock/src/model.rs"]
mod stock;

use prost::Message;
use umpire_runtime::AnyMessage as _;
use umpire_runtime::{
    prost_types::Any,
    protocol::{self, response::Outcome},
};

#[test]
fn explanation_outcomes_are_additive_at_tag_two() {
    for (outcomes, bytes) in
        [(vec![], vec![10, 1, b'x']), (vec!["()".into()], vec![10, 1, b'x', 18, 2, b'(', b')'])]
    {
        let explanation = protocol::Explanation { steps: vec!["x".into()], outcomes };
        assert_eq!(explanation.encode_to_vec(), bytes);
        assert_eq!(protocol::Explanation::decode(bytes.as_slice()), Ok(explanation));
    }
}

#[test]
fn new_core_messages_match_shared_literal_bytes() {
    let payload = Any { type_url: "x".into(), value: vec![1, 2] };
    for line in include_str!("../../proto/tests/protocol-core-wire.txt").lines() {
        let (name, hex) = line.split_once(' ').unwrap_or_else(|| panic!("invalid vector"));
        let actual = match name {
            "judge_request" => {
                protocol::JudgeRequest { observation: Some(payload.clone()) }.encode_to_vec()
            }
            "answer_request" => {
                protocol::AnswerRequest { callback: Some(payload.clone()) }.encode_to_vec()
            }
            _ => {
                let outcome = match name {
                    "reset_response" => Outcome::Reset(protocol::ResetResponse::current(None)),
                    "judge_explained" => Outcome::Judge(protocol::JudgeResponse {
                        continue_polling: false,
                        judgment: Some(protocol::judge_response::Judgment::Explained(
                            protocol::Explanation::default(),
                        )),
                    }),
                    "judge_unexplained" => Outcome::Judge(protocol::JudgeResponse {
                        continue_polling: false,
                        judgment: Some(protocol::judge_response::Judgment::Unexplained(
                            protocol::Unexplained { expected_actions: vec![] },
                        )),
                    }),
                    "answer_response" => {
                        Outcome::Answer(protocol::AnswerResponse { reply: Some(payload.clone()) })
                    }
                    _ => panic!("unknown vector: {name}"),
                };
                protocol::Response { outcome: Some(outcome) }.encode_to_vec()
            }
        };
        let expected = support::decode_hex(hex);
        assert_eq!(actual, expected, "{name}");
    }
}

#[test]
fn explained_zero_steps_is_present_and_distinct_from_unexplained() {
    let explained = protocol::JudgeResponse {
        continue_polling: false,
        judgment: Some(protocol::judge_response::Judgment::Explained(protocol::Explanation {
            steps: vec![],
            outcomes: vec![],
        })),
    };
    assert_eq!(explained.encode_to_vec(), [10, 0]);
    assert_eq!(
        protocol::JudgeResponse::decode(&[10, 0][..]).unwrap_or_else(|e| panic!("{e}")),
        explained
    );
    let unexplained = protocol::JudgeResponse {
        continue_polling: false,
        judgment: Some(protocol::judge_response::Judgment::Unexplained(protocol::Unexplained {
            expected_actions: vec![],
        })),
    };
    assert_eq!(unexplained.encode_to_vec(), [18, 0]);
    assert_ne!(explained, unexplained);
    assert_eq!(protocol::JudgeResponse::default().encode_to_vec(), []);
}

#[test]
fn new_requests_round_trip_opaque_payloads_and_dispatch_honest_refusals() {
    assert_eq!(protocol::Method::Reset as u32, 3);
    assert_eq!(protocol::Method::Judge as u32, 4);
    assert_eq!(protocol::Method::Answer as u32, 5);
    let payload = Any { type_url: "example.net/system.Input".into(), value: vec![255, 0, 42] };
    let judge = protocol::JudgeRequest { observation: Some(payload.clone()) };
    let answer = protocol::AnswerRequest { callback: Some(payload.clone()) };
    assert_eq!(
        protocol::JudgeRequest::decode(judge.encode_to_vec().as_slice())
            .unwrap_or_else(|e| panic!("{e}")),
        judge
    );
    assert_eq!(
        protocol::AnswerRequest::decode(answer.encode_to_vec().as_slice())
            .unwrap_or_else(|e| panic!("{e}")),
        answer
    );
    let reply = protocol::AnswerResponse { reply: Some(payload) };
    assert_eq!(
        protocol::AnswerResponse::decode(reply.encode_to_vec().as_slice())
            .unwrap_or_else(|e| panic!("{e}")),
        reply
    );
    let mut model = stock::model();
    for (method, request) in [(3, vec![]), (4, judge.encode_to_vec()), (5, answer.encode_to_vec())]
    {
        let Some(Outcome::Refusal(refusal)) = model.dispatch(method, &request).outcome else {
            panic!("expected staged unsupported refusal")
        };
        assert_eq!(refusal.code, protocol::RefusalCode::Unsupported as i32);
        assert!(refusal.message.contains("unsupported"));
    }
    for (method, request) in
        [(3, vec![255]), (4, vec![255]), (5, vec![255]), (4, vec![]), (5, vec![])]
    {
        let Some(Outcome::Refusal(refusal)) = model.dispatch(method, &request).outcome else {
            panic!("expected malformed refusal")
        };
        assert_eq!(refusal.code, protocol::RefusalCode::MalformedRequest as i32);
    }
}

#[test]
fn any_admits_only_canonical_urls_and_distinguishes_refusals() {
    use prost::Name;
    use protocol::RefusalCode::{MalformedRequest, Unsupported};
    let message = protocol::ResetRequest::default();
    let valid = Any::pack(&message);
    assert_eq!(
        valid.unpack::<protocol::ResetRequest>().unwrap_or_else(|error| panic!("{error:?}")),
        message
    );
    for (type_url, bytes, code) in [
        (String::new(), vec![], MalformedRequest),
        (protocol::RowsRequest::type_url(), vec![], Unsupported),
        (format!("other.example/{}", protocol::ResetRequest::full_name()), vec![], Unsupported),
        (protocol::ResetRequest::full_name(), vec![], Unsupported),
        (protocol::ResetRequest::type_url(), vec![0xff], MalformedRequest),
    ] {
        let error = Any { type_url, value: bytes }
            .unpack::<protocol::ResetRequest>()
            .err()
            .unwrap_or_else(|| panic!("invalid Any must refuse"));
        assert_eq!(error.code, code as i32);
    }
    assert_eq!(
        Any::required(None).err().unwrap_or_else(|| panic!("missing payload")).code,
        MalformedRequest as i32
    );
}
