use serde_json::{Map, Value, json};

const PROFILE_PREFIX: &str = "gomad-wasm-profile-v1 ";

/// Host clocks and resource counters (`metrics`, cluster `scheduler`) are
/// informational: durations keep only whether they are zero, process
/// counters only that they were measured.
fn project_metrics(metrics: &Value) -> Value {
    let Some(fields) = metrics.as_object() else {
        return metrics.clone();
    };
    let mut out = Map::new();
    for (key, value) in fields {
        let projected = match (key.as_str(), value.as_u64()) {
            ("process_user_nanos" | "process_system_nanos" | "peak_rss_bytes", Some(_)) => {
                json!("measured")
            }
            (name, Some(number)) if name.ends_with("_nanos") => {
                if number == 0 {
                    json!(0)
                } else {
                    json!("nonzero")
                }
            }
            _ => project(value),
        };
        out.insert(key.clone(), projected);
    }
    Value::Object(out)
}

pub(super) fn project(value: &Value) -> Value {
    match value {
        Value::Object(fields) => {
            let mut out = Map::new();
            for (key, item) in fields {
                let projected = match key.as_str() {
                    // Host telemetry outside semantic evidence.
                    "metrics" | "scheduler" => project_metrics(item),
                    "engine" if item.is_object() => {
                        let mut engine = project(item);
                        for host in ["host_os", "host_arch"] {
                            if engine[host].is_string() {
                                engine[host] = json!(format!("${host}"));
                            }
                        }
                        engine
                    }
                    _ => project(item),
                };
                out.insert(key.clone(), projected);
            }
            Value::Object(out)
        }
        Value::Array(items) => Value::Array(items.iter().map(project).collect()),
        other => other.clone(),
    }
}

/// Diagnostic builds report one `gomad-wasm-profile-v1` stderr line of host
/// timing; it is kept as its projected document, the rest as text.
pub(super) fn split_profile(stderr: &str) -> Result<(String, Value), String> {
    let mut rest = String::new();
    let mut profile = Value::Null;
    for line in stderr.split_inclusive('\n') {
        match line.strip_prefix(PROFILE_PREFIX) {
            Some(document) if profile.is_null() => {
                let document: Value = serde_json::from_str(document)
                    .map_err(|error| format!("diagnostic profile: {error}"))?;
                profile = project_profile(&document);
            }
            _ => rest.push_str(line),
        }
    }
    Ok((rest, profile))
}

/// `gomad.wasm-profile/v2`: event counts stay exact; every duration, CPU,
/// RSS and residual figure keeps only that it was measured (nested spans can
/// read zero nanoseconds, so not even zero/nonzero is stable).
fn project_profile(value: &Value) -> Value {
    match value {
        Value::Object(fields) => {
            let mut out = Map::new();
            for (key, item) in fields {
                let projected = match (key.as_str(), item.as_u64()) {
                    ("residual", _) | ("peak_rss_bytes_high_water", Some(_)) => json!("measured"),
                    ("host_os" | "host_arch", _) => json!(format!("${key}")),
                    (name, Some(_)) if name.ends_with("_nanos") => json!("measured"),
                    _ => project_profile(item),
                };
                out.insert(key.clone(), projected);
            }
            Value::Object(out)
        }
        Value::Array(items) => Value::Array(items.iter().map(project_profile).collect()),
        other => other.clone(),
    }
}
