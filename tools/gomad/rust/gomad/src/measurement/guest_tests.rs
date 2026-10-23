use super::*;
use serde_json::json;

fn sampled_profile() -> Value {
    let mut profile: Value =
        serde_json::from_slice(include_bytes!("testdata/guest-profile.json")).unwrap();
    profile["meta"]["gomadGuestProfile"] =
        json!({"intervalMicros":1000,"maxSamples":10000,"samples":5,"truncated":false});
    profile
}

#[test]
fn sampled_metadata_must_match_the_request_and_decoded_rows() {
    for (field, mutation) in [
        ("intervalMicros", json!(1001)),
        ("intervalMicros", Value::Null),
        ("maxSamples", json!(10001)),
        ("maxSamples", json!(-1)),
        ("samples", json!(4)),
        ("samples", json!(6)),
        ("samples", json!(10001)),
        ("samples", Value::Null),
        ("truncated", json!("false")),
        ("truncated", Value::Null),
    ] {
        let mut profile = sampled_profile();
        profile["meta"]["gomadGuestProfile"][field] = mutation;
        let raw = serde_json::to_vec(&profile).unwrap();
        assert_eq!(
            decode(&raw).unwrap().sample_rows,
            5,
            "invalid test tables: {field}"
        );
        assert!(
            decode_sampled(&raw, 1000, 10000).is_err(),
            "accepted {field}: {profile}"
        );
    }
    let mut over_limit = sampled_profile();
    over_limit["meta"]["gomadGuestProfile"]["maxSamples"] = json!(4);
    assert!(decode_sampled(&serde_json::to_vec(&over_limit).unwrap(), 1000, 4).is_err());
}

#[test]
fn sampled_metadata_retains_typed_truncation_and_weighted_reconciliation() {
    for truncated in [false, true] {
        let mut profile = sampled_profile();
        profile["meta"]["gomadGuestProfile"]["truncated"] = json!(truncated);
        let sampled = decode_sampled(&serde_json::to_vec(&profile).unwrap(), 1000, 10000).unwrap();
        assert_eq!(sampled.attribution.sample_rows, 5);
        assert_eq!(sampled.attribution.guest_sample_weight, 11);
        assert_eq!(sampled.attribution.empty_stack_weight, 7);
        assert_eq!(
            sampled.attribution.self_sample_weight.values().sum::<u64>(),
            11
        );
        let retained = serde_json::to_value(&sampled).unwrap();
        assert_eq!(
            retained["sampling"],
            json!({"intervalMicros":1000,"maxSamples":10000,"samples":5,"truncated":truncated})
        );
        assert_eq!(retained["sample_rows"], 5);
    }
    let mut at_bound = sampled_profile();
    at_bound["meta"]["gomadGuestProfile"]["maxSamples"] = json!(5);
    at_bound["meta"]["gomadGuestProfile"]["truncated"] = json!(true);
    assert!(decode_sampled(&serde_json::to_vec(&at_bound).unwrap(), 1000, 5).is_ok());
}
