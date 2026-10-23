//! Invariant: the first storage crash refusal never depends on selection or
//! context insertion order, and equals Go `ValidateStorageCrashSelection` for
//! the shared fn-227 corpus (the Go test reads the same file).
use super::*;
use serde_json::Value;

const CORPUS: &str =
    include_str!("../../../../testdata/parity/storage-crash-validation-order.json");

fn object(entries: &[Value]) -> String {
    let fields: Vec<String> = entries
        .iter()
        .map(|entry| format!("{}:{}", entry[0], entry[1]))
        .collect();
    format!("{{{}}}", fields.join(","))
}

#[test]
fn first_refusal_is_insertion_order_independent_and_matches_go() {
    let corpus: Value = serde_json::from_str(CORPUS).unwrap();
    assert_eq!(corpus["schema"], "gomad.storage-crash-validation-order/v1");
    let volumes: Vec<Value> = corpus["volumes"]
        .as_array()
        .unwrap()
        .iter()
        .map(|id| {
            serde_json::json!({"CapacityBytes": 512, "ID": id, "Path": format!("/{}", id.as_str().unwrap())})
        })
        .collect();
    let model: StorageModel = serde_json::from_value(serde_json::json!({
        "limits": {"PendingOperations": corpus["pending_operations"], "Transitions": 100},
        "volumes": volumes,
    }))
    .unwrap();
    let cases = corpus["cases"].as_array().unwrap();
    assert_ne!(cases.len(), 0);
    for case in cases {
        for reversed in [false, true] {
            let ordered = |key: &str| {
                case[key].as_array().map(|entries| {
                    let mut entries = entries.clone();
                    if reversed {
                        entries.reverse();
                    }
                    object(&entries)
                })
            };
            let selection = ordered("selection");
            let context = ordered("context");
            let declared = context.as_ref().map_or(Value::Null, |bytes| {
                serde_json::json!({
                    "bytes": bytes.len(),
                    "chunks": (bytes.len() as u64).div_ceil(CHUNK),
                    "sha256": format!("{:x}", Sha256::digest(bytes.as_bytes())),
                })
            });
            let text = format!(
                r#"{{"model":{},"model_identity":"{}","crash_selection":{},"crash_context":{},"return_image":false}}"#,
                serde_json::to_string(&model).unwrap(),
                model.identity(),
                selection.as_deref().unwrap_or("null"),
                declared,
            );
            let mut descriptor: StorageDescriptor = serde_json::from_str(&text).unwrap();
            let refusal = descriptor
                .validate()
                .and_then(|()| {
                    descriptor.attach_context_bytes(context.as_deref().map(str::as_bytes), 1 << 30)
                })
                .expect_err(case["name"].as_str().unwrap());
            assert_eq!(
                refusal.to_string(),
                case["want"].as_str().unwrap(),
                "{} reversed={reversed}",
                case["name"]
            );
        }
    }
}
