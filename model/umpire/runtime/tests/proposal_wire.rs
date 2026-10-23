//! Additive proposal wire contract, preserving the existing vectors.
use prost::Message;
use umpire_runtime::{
    google::protobuf::Any,
    protocol::{self, response::Outcome},
};
#[test]
fn proposal_method_kind_and_poll_wire_contract() {
    assert_eq!(protocol::Method::Propose as i32, 6);
    assert_eq!(protocol::Method::Annotate as i32, 7);
    assert_eq!(protocol::ProposalKind::Unspecified as i32, 0);
    assert_eq!(protocol::RefusalCode::Quiescent as i32, 9);
    let opaque = Any { type_url: "x".into(), value: vec![1, 2] };
    for line in include_str!("../../proto/tests/proposal-wire.txt").lines() {
        let (name, wire) = line.split_once(' ').unwrap_or_else(|| panic!("vector"));
        let value = match name {
            "propose_request" => protocol::ProposeRequest { seed: 150 }.encode_to_vec(),
            "reset_configuration" => {
                protocol::ResetRequest { configuration: Some(opaque.clone()) }.encode_to_vec()
            }
            _ => {
                let outcome = match name {
                    "propose_action" | "propose_bootstrap" => {
                        Outcome::Propose(protocol::ProposeResponse {
                            request: Some(opaque.clone()),
                            action: if name == "propose_action" {
                                "Act".into()
                            } else {
                                String::new()
                            },
                            kind: if name == "propose_action" {
                                protocol::ProposalKind::Action as i32
                            } else {
                                protocol::ProposalKind::Bootstrap as i32
                            },
                        })
                    }
                    "judge_poll" | "judge_poll_go" => Outcome::Judge(protocol::JudgeResponse {
                        judgment: Some(protocol::judge_response::Judgment::Explained(
                            protocol::Explanation::default(),
                        )),
                        continue_polling: true,
                    }),
                    "refusal_quiescent" => Outcome::Refusal(protocol::Refusal::new(
                        protocol::RefusalCode::Quiescent,
                        "x",
                    )),
                    _ => panic!("vector"),
                };
                protocol::Response { outcome: Some(outcome) }.encode_to_vec()
            }
        };
        let mut actual = String::new();
        for byte in value {
            use std::fmt::Write;
            write!(actual, "{byte:02x}").unwrap_or_else(|error| panic!("{error}"));
        }
        if name == "judge_poll_go" {
            let bytes = [0x2a, 4, 0x18, 1, 0x0a, 0];
            let response = protocol::Response::decode(bytes.as_slice())
                .unwrap_or_else(|error| panic!("{error}"));
            assert_eq!(response.encode_to_vec(), [0x2a, 4, 0x0a, 0, 0x18, 1]);
            assert_eq!(wire, "2a0418010a00");
        } else {
            assert_eq!(actual, wire, "{name}");
        }
    }
}
