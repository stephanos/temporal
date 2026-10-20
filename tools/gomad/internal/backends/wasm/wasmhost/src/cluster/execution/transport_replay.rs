use super::*;
use serde_json::{Value, json};

#[test]
fn tagged_replay_transactions_preserve_attribution_and_poison_broken_framing() {
    let fixture: Value = serde_json::from_str(include_str!(
        "../../../tests/testdata/replay_plan/empty.json"
    ))
    .unwrap();
    let request = serde_json::from_value(json!({"schema":"gomad.replay-trace/v1",
        "execution_identity":fixture["identity"],
        "trace_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
        "tape_sha256":fixture["expected"]["sha256"],"decisions":0,
        "trace_bytes":0,"trace_frames":0}))
    .unwrap();
    for mutation in ["none", "foreign", "stale", "foreign-end", "truncated"] {
        let mut bytes = Vec::new();
        for kind in ["replay_trace", "replay_trace_end"] {
            if mutation == "truncated" && kind == "replay_trace_end" {
                break;
            }
            let node = if mutation == "foreign"
                || mutation == "foreign-end" && kind == "replay_trace_end"
            {
                "b"
            } else {
                "a"
            };
            let incarnation = if mutation == "stale" { 6 } else { 7 };
            writeln!(
                bytes,
                "{}",
                json!({"type":kind,"instance":node,"incarnation":incarnation,
                "schema":"gomad.replay-trace/v1","trace_bytes":0,"trace_frames":0})
            )
            .unwrap();
        }
        bytes.extend(b"{\"instance\":\"a\",\"incarnation\":7,\"type\":\"next\"}\n");
        let shared: SharedInput = Arc::new(Mutex::new(Box::new(std::io::Cursor::new(bytes))));
        let broken = Broken::default();
        let mut input = TaggedInput::incarnation(shared, "a", 7, broken.clone());
        let result = input.replay(&request, 2, 4096);
        if mutation == "none" {
            assert!(result.is_ok());
            assert!(broken.get().is_none());
            assert_eq!(
                crate::protocol::read_frame(&mut input).unwrap(),
                b"{\"type\":\"next\"}\n"
            );
        } else {
            let error = result.err().unwrap();
            assert_eq!(
                error
                    .downcast_ref::<crate::model::replay::Preflight>()
                    .unwrap()
                    .check,
                "framing"
            );
            assert_eq!(broken.get().as_deref(), Some(error.to_string().as_str()));
        }
    }
}
