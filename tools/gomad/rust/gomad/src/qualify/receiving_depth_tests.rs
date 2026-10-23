use super::strict;
use serde_json::Value;

fn nested(depth: usize, value: &str) -> Vec<u8> {
    format!("{}{}{}", "[".repeat(depth), value, "]".repeat(depth)).into_bytes()
}

#[test]
fn original_depth_128_scalar_edge_is_accepted() {
    strict::<Value>(&nested(128, "0")).unwrap();
}

#[test]
fn original_depth_129_scalar_overflow_keeps_nesting_error() {
    assert_eq!(
        strict::<Value>(&nested(129, "0")).unwrap_err(),
        "qualification JSON nesting limit"
    );
}

#[test]
fn original_depth_129_malformed_overflow_precedes_child_syntax() {
    assert_eq!(
        strict::<Value>(&nested(129, "|")).unwrap_err(),
        "qualification JSON nesting limit"
    );
}

#[test]
fn original_depth_129_empty_container_edge_is_accepted() {
    strict::<Value>(&nested(129, "")).unwrap();
}

#[test]
fn original_depth_130_empty_container_overflow_is_refused() {
    assert_eq!(
        strict::<Value>(&nested(130, "")).unwrap_err(),
        "qualification JSON nesting limit"
    );
}

#[test]
fn depth_preflight_ignores_quoted_braces_and_preserves_first_syntax_and_trailing_errors() {
    strict::<Value>(br#"{"quoted":"[[[{\\\"","nested":[]}"#).unwrap();
    let mut trailing = b"{} ".to_vec();
    trailing.extend(nested(130, "|"));
    assert_eq!(
        strict::<Value>(&trailing).unwrap_err(),
        "qualification JSON has trailing data"
    );
    let mut malformed = b"[|,".to_vec();
    malformed.extend(nested(130, "0"));
    malformed.push(b']');
    assert!(
        !strict::<Value>(&malformed)
            .unwrap_err()
            .contains("nesting limit")
    );
    for invalid in [
        b"[0,]".as_slice(),
        b"{\"key\":}",
        b"{\"key\" 0}",
        b"[true false]",
    ] {
        assert!(strict::<Value>(invalid).is_err());
    }
}

#[test]
fn depth_admission_preserves_object_empty_edges_and_json_whitespace() {
    let object = |count: usize, tail: &str| {
        format!("{}{}{}", "{\"key\":".repeat(count), tail, "}".repeat(count))
    };
    strict::<Value>(object(128, "0").as_bytes()).unwrap();
    strict::<Value>(object(128, "{}").as_bytes()).unwrap();
    for tail in ["0", "|"] {
        assert_eq!(
            strict::<Value>(object(129, tail).as_bytes()).unwrap_err(),
            "qualification JSON nesting limit"
        );
    }
    for input in [
        b"{}\x0b".as_slice(),
        b"\x0c{}",
        b"[0,]",
        b"{\"a\":0,}",
        b"{\"a\":0,\"a\":1}",
    ] {
        assert!(strict::<Value>(input).is_err());
    }
}

#[test]
fn first_duplicate_precedes_later_depth_overflow() {
    let mut input = br#"{"same":0,"same":1,"later":"#.to_vec();
    input.extend(nested(130, "0"));
    input.push(b'}');
    let error = strict::<Value>(&input).unwrap_err();
    assert!(error.contains("duplicate"), "{error}");
    assert!(!error.contains("nesting limit"), "{error}");
}

#[test]
fn scalar_first_value_keeps_adjacent_container_trailing_error() {
    for input in [b"0{}".as_slice(), b"true[]", b"null\"later\""] {
        assert_eq!(
            strict::<Value>(input).unwrap_err(),
            "qualification JSON has trailing data"
        );
    }
}

#[test]
fn original_primitive_first_token_precedes_adjacent_trailing_bytes() {
    for input in [
        b"truefalse".as_slice(),
        b"nullx",
        b"0true",
        b"01",
        b"false0",
        b"-01",
        b"1e2false",
        b"1e999false",
    ] {
        assert_eq!(
            strict::<Value>(input).unwrap_err(),
            "qualification JSON has trailing data",
            "{input:?}"
        );
    }
}

#[test]
fn original_deepest_key_depth_guard_precedes_missing_colon() {
    let input = format!(
        "{}{}{}",
        "{\"k\":".repeat(128),
        "{\"k\" 0}",
        "}".repeat(128)
    );
    assert_eq!(
        strict::<Value>(input.as_bytes()).unwrap_err(),
        "qualification JSON nesting limit"
    );
}

#[test]
fn primitive_lexing_still_refuses_malformed_numbers_and_container_separators() {
    for input in [
        b"-".as_slice(),
        b"1.",
        b"1e",
        b"1e+",
        b"[truefalse]",
        b"[01]",
        b"{\"k\":nullx}",
    ] {
        let error = strict::<Value>(input).unwrap_err();
        assert!(!error.contains("has trailing data"), "{input:?}: {error}");
    }
}
