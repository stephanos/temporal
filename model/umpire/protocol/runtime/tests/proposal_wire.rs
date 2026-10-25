//! Current proposal and whole-prefix Reset wire contract.
#[allow(dead_code, reason = "wire vectors use only the shared hex decoder")]
mod support;

use prost::Message;
use umpire_runtime::{
    prost_types::Any,
    protocol::{self, response::Outcome},
};
#[test]
fn proposal_method_kind_and_poll_wire_contract() {
    assert_eq!(protocol::Method::Propose as i32, 6);
    assert_eq!(protocol::Method::Annotate as i32, 7);
    assert_eq!(protocol::ProposalKind::Unspecified as i32, 0);
    assert_eq!(protocol::ProposalKind::Read as i32, 3);
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
                    "propose_action" | "propose_bootstrap" | "propose_read" => {
                        Outcome::Propose(protocol::ProposeResponse {
                            request: Some(opaque.clone()),
                            action: if name == "propose_action" {
                                "Act".into()
                            } else {
                                String::new()
                            },
                            kind: if name == "propose_action" {
                                protocol::ProposalKind::Action as i32
                            } else if name == "propose_read" {
                                protocol::ProposalKind::Read as i32
                            } else {
                                protocol::ProposalKind::Bootstrap as i32
                            },
                        })
                    }
                    "reset_program" => {
                        Outcome::Reset(protocol::ResetResponse::current(Some(opaque.clone())))
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
        if name == "judge_poll_go" {
            let bytes = support::decode_hex(wire);
            let response = protocol::Response::decode(bytes.as_slice())
                .unwrap_or_else(|error| panic!("{error}"));
            assert_eq!(response.encode_to_vec(), [0x2a, 4, 0x0a, 0, 0x18, 1]);
            assert_eq!(wire, "2a0418010a00");
        } else {
            assert_eq!(value, support::decode_hex(wire), "{name}");
        }
    }
}
