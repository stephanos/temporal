//! Literal pre-rename wire controls, independent of generated descriptors.

use prost::Message;
use umpire_wasm::protocol::{self, response::Outcome};

#[test]
fn original_wire_bytes_and_values_are_stable() {
    assert_eq!(protocol::Method::Explore as i32, 1);
    assert_eq!(protocol::Method::Rows as i32, 2);
    for line in include_str!("../../tests/fixtures/shared/protocol-wire.txt").lines() {
        let (name, hex) = line.split_once(' ').unwrap_or_else(|| panic!("invalid vector"));
        let bytes: Vec<u8> = hex
            .as_bytes()
            .chunks_exact(2)
            .map(|pair| {
                let pair = std::str::from_utf8(pair).unwrap_or_else(|e| panic!("{e}"));
                u8::from_str_radix(pair, 16).unwrap_or_else(|e| panic!("{e}"))
            })
            .collect();
        match name {
            "explore_request" => {
                let value = protocol::ExploreRequest::decode(bytes.as_slice())
                    .unwrap_or_else(|e| panic!("{e}"));
                assert_eq!(value.budget, 150);
                assert_eq!(value.encode_to_vec(), bytes);
            }
            "rows_request" => {
                let value = protocol::RowsRequest::decode(bytes.as_slice())
                    .unwrap_or_else(|e| panic!("{e}"));
                assert_eq!(value.state_id, 150);
                assert_eq!(value.encode_to_vec(), bytes);
            }
            _ => {
                let value =
                    protocol::Response::decode(bytes.as_slice()).unwrap_or_else(|e| panic!("{e}"));
                assert_eq!(value.encode_to_vec(), bytes);
                match value.outcome {
                    Some(Outcome::Explore(v)) => {
                        assert_eq!(name, "explore_response");
                        assert_eq!(
                            v,
                            protocol::ExploreResponse {
                                report: "r".into(),
                                starts: 1,
                                reached: 2,
                                emitted: 3,
                                evaluated: 4,
                                refused: 5,
                                depth: 6,
                                cut: true
                            }
                        );
                    }
                    Some(Outcome::Rows(v)) => {
                        assert_eq!(name, "rows_response");
                        assert_eq!(v.rows, ["x", "y"]);
                    }
                    Some(Outcome::Refusal(v)) => {
                        let code = name
                            .strip_prefix("refusal_")
                            .and_then(|s| s.parse::<i32>().ok())
                            .unwrap_or_else(|| panic!("invalid refusal name"));
                        assert_eq!(v.code, code);
                        assert_eq!(v.message, "x");
                        assert!(protocol::RefusalCode::try_from(code).is_ok());
                    }
                    _ => panic!("unexpected original outcome"),
                }
            }
        }
    }
}
