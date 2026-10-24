//! Generic operation encoding and staged dispatch contract.

use prost::Message;
use umpire_runtime::{
    Model,
    google::protobuf::Any,
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
                    "reset_response" => Outcome::Reset(protocol::ResetResponse::default()),
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
        let (pairs, remainder) = hex.as_bytes().as_chunks::<2>();
        assert_eq!(remainder, []);
        let expected: Vec<u8> = pairs
            .iter()
            .map(|pair| {
                u8::from_str_radix(std::str::from_utf8(pair).unwrap_or_else(|e| panic!("{e}")), 16)
                    .unwrap_or_else(|e| panic!("{e}"))
            })
            .collect();
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
    let mut model = Model::default();
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
