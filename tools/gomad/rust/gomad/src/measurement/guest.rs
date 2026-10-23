use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::BTreeMap;

const BUCKETS: [&str; 8] = [
    "gc",
    "scheduler",
    "allocation",
    "test",
    "server",
    "runtime",
    "stdlib",
    "other",
];
const STDLIB: &str = "archive bufio bytes cmp compress container context crypto database debug embed encoding errors expvar flag fmt hash html image index internal io iter log maps math mime net os path plugin reflect regexp runtime slices sort strconv strings structs sync syscall testing text time unicode unique unsafe weak";

#[derive(Debug, Serialize)]
pub struct Attribution {
    pub sample_rows: u64,
    pub guest_sample_weight: u64,
    pub empty_stack_weight: u64,
    pub self_sample_weight: BTreeMap<String, u64>,
}

fn frame_bucket(name: &str) -> Option<&'static str> {
    let name = name.strip_prefix('$').unwrap_or(name);
    let lower = name.to_ascii_lowercase();
    if "runtime.gcBgMarkWorker runtime.gcDrain runtime.scanobject runtime.markroot runtime.gcAssistAlloc runtime.bgsweep runtime.sweepone runtime.wbBufFlush runtime.gcWriteBarrier runtime.bulkBarrier runtime.writebarrier runtime.greyobject runtime.findobject runtime.trydefertospanscan runtime.scanblock runtime.scanstack runtime.scanframeworker"
        .split_whitespace().any(|prefix|lower.starts_with(&prefix.to_ascii_lowercase())) {
        return Some("gc");
    }
    if "runtime.schedule runtime.findRunnable runtime.gopark runtime.park_m runtime.goready runtime.mcall runtime.gogo runtime.handleEvent runtime.beforeIdle wasm_pc_f_loop runtime.wasm_pc_f_loop".split_whitespace().any(|function| {
        name == function || name.strip_prefix(function).is_some_and(|suffix|suffix.starts_with('.') || suffix.starts_with('('))
    }) { return Some("scheduler"); }
    if "runtime.mallocgc runtime.makeslice runtime.growslice runtime.newobject runtime.newarray runtime.makemap runtime.mapassign".split_whitespace().any(|prefix|name.starts_with(prefix)) {
        return Some("allocation");
    }
    for separator in ['/', '_'] {
        if name.starts_with(&format!("go.temporal.io{separator}server{separator}tests."))
            || name.starts_with(&format!(
                "go.temporal.io{separator}server{separator}tests{separator}"
            ))
        {
            return Some("test");
        }
        if name.starts_with(&format!("go.temporal.io{separator}server{separator}")) {
            return Some("server");
        }
    }
    if [
        "runtime.",
        "runtime/",
        "internal/runtime/",
        "internal_runtime_",
    ]
    .iter()
    .any(|prefix| name.starts_with(prefix))
    {
        return Some("runtime");
    }
    let first = name.split('.').next().unwrap_or("");
    if !STDLIB.split_whitespace().any(|part| part == first) {
        let host = name.split(['/', '_']).next().unwrap_or("");
        let mut labels = host.split('.');
        let valid_label = |part: &str| {
            !part.is_empty()
                && part
                    .bytes()
                    .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'-')
        };
        if host != name && labels.clone().count() > 1 && labels.all(valid_label) {
            return None;
        }
    }
    if name.contains('.') {
        let first = if name.contains('/') {
            name.split('/').next()
        } else {
            name.split('.').next()
        }
        .unwrap_or("");
        if !first.is_empty()
            && first != "main"
            && !first.contains(['.', ' ', '<', '>', '[', ']', ':'])
        {
            return Some("stdlib");
        }
    }
    None
}

#[must_use]
pub fn stack_bucket(frames: &[&str]) -> &'static str {
    frames
        .iter()
        .find_map(|name| frame_bucket(name))
        .unwrap_or("other")
}

fn array<'a>(value: &'a Value, field: &str) -> Result<&'a [Value], String> {
    value[field]
        .as_array()
        .map(Vec::as_slice)
        .ok_or_else(|| format!("guest profile missing {field} array"))
}
fn index(value: &Value, bound: usize) -> Result<usize, String> {
    value
        .as_u64()
        .and_then(|value| usize::try_from(value).ok())
        .filter(|value| *value < bound)
        .ok_or_else(|| "guest profile invalid table index".into())
}
fn table<'a>(value: &'a Value, field: &str) -> Result<&'a [Value], String> {
    let data = array(value, field)?;
    if value["length"].as_u64() != u64::try_from(data.len()).ok() {
        return Err(format!("guest profile inconsistent {field} table length"));
    }
    Ok(data)
}
fn add(target: &mut u64, value: u64) -> Result<(), String> {
    *target = target
        .checked_add(value)
        .ok_or("guest sample weight overflow")?;
    Ok(())
}

/// # Errors
/// Refuses inconsistent tables, invalid/cyclic stacks, negative weights and overflow.
pub fn decode(raw: &[u8]) -> Result<Attribution, String> {
    let profile: Value = serde_json::from_slice(raw).map_err(|error| error.to_string())?;
    decode_profile(&profile)
}

#[derive(Debug, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Sampling {
    pub interval_micros: u64,
    pub max_samples: u64,
    pub samples: u64,
    pub truncated: bool,
}

#[derive(Debug, Serialize)]
pub struct SampledAttribution {
    #[serde(flatten)]
    pub attribution: Attribution,
    pub sampling: Sampling,
}

/// # Errors
/// Refuses invalid profile tables or sampling metadata that does not match the request and rows.
pub fn decode_sampled(
    raw: &[u8],
    interval_micros: u64,
    max_samples: u64,
) -> Result<SampledAttribution, String> {
    let profile: Value = serde_json::from_slice(raw).map_err(|error| error.to_string())?;
    let attribution = decode_profile(&profile)?;
    let sampling: Sampling = serde_json::from_value(profile["meta"]["gomadGuestProfile"].clone())
        .map_err(|error| error.to_string())?;
    if sampling.interval_micros != interval_micros
        || sampling.max_samples != max_samples
        || sampling.samples > sampling.max_samples
        || sampling.samples != attribution.sample_rows
    {
        return Err(
            "guest profile sampling metadata does not match requested options and decoded rows"
                .into(),
        );
    }
    Ok(SampledAttribution {
        attribution,
        sampling,
    })
}

fn decode_profile(profile: &Value) -> Result<Attribution, String> {
    if !profile["meta"]["interval"]
        .as_f64()
        .is_some_and(|interval| interval.is_finite() && interval > 0.0)
    {
        return Err("guest profile has no positive sampling interval".into());
    }
    let threads = array(profile, "threads")?;
    if threads.is_empty() {
        return Err("guest profile has no threads".into());
    }
    let mut result = Attribution {
        sample_rows: 0,
        guest_sample_weight: 0,
        empty_stack_weight: 0,
        self_sample_weight: BUCKETS.into_iter().map(|name| (name.into(), 0)).collect(),
    };
    for thread in threads {
        let stacks = table(&thread["samples"], "stack")?;
        let weights = match thread["samples"].get("weight") {
            None | Some(Value::Null) => None,
            Some(value) => Some(
                value
                    .as_array()
                    .filter(|weights| weights.len() == stacks.len())
                    .ok_or("guest profile inconsistent sample weights")?,
            ),
        };
        if thread["samples"]["weightType"] != "samples" {
            return Err("guest profile weight type is not samples".into());
        }
        let frames = table(&thread["stackTable"], "frame")?;
        let prefixes = array(&thread["stackTable"], "prefix")?;
        if prefixes.len() != frames.len() {
            return Err("guest profile inconsistent stack prefixes".into());
        }
        let functions = table(&thread["frameTable"], "func")?;
        let names = table(&thread["funcTable"], "name")?;
        let strings = array(thread, "stringArray")?;
        for (row, stack) in stacks.iter().enumerate() {
            let weight = if let Some(weights) = weights {
                u64::try_from(
                    weights[row]
                        .as_i64()
                        .ok_or("guest sample weight exceeds signed bound")?,
                )
                .map_err(|_| "negative guest sample weight")?
            } else {
                1
            };
            add(&mut result.sample_rows, 1)?;
            let mut stack = stack;
            let mut leaf_to_root = Vec::new();
            while !stack.is_null() {
                if leaf_to_root.len() >= frames.len() {
                    return Err("guest profile cyclic stack".into());
                }
                let slot = index(stack, frames.len())?;
                let frame = index(&frames[slot], functions.len())?;
                let function = index(&functions[frame], names.len())?;
                let name = index(&names[function], strings.len())?;
                leaf_to_root.push(
                    strings[name]
                        .as_str()
                        .ok_or("guest profile function name is not text")?,
                );
                stack = &prefixes[slot];
            }
            if leaf_to_root.is_empty() {
                add(&mut result.empty_stack_weight, weight)?;
            } else {
                add(&mut result.guest_sample_weight, weight)?;
                add(
                    result
                        .self_sample_weight
                        .get_mut(stack_bucket(&leaf_to_root))
                        .ok_or("unknown guest bucket")?,
                    weight,
                )?;
            }
        }
    }
    Ok(result)
}

#[cfg(test)]
mod tests {
    use super::*;
    const PROFILE: &[u8] = include_bytes!("testdata/guest-profile.json");
    #[test]
    fn original_frame_classification_keeps_leaf_precedence() {
        for name in "runtime.gcBgMarkWorker runtime.gcDrainN runtime.scanobject runtime.markrootBlock runtime.gcAssistAlloc1 runtime.bgsweep runtime.sweepone runtime.wbBufFlush1 runtime.gcWriteBarrier runtime.bulkBarrierPreWrite runtime.writebarrierptr runtime.scanObject runtime.scanObjectsSmall runtime.greyobject runtime.findObject runtime.tryDeferToSpanScan runtime.scanblock runtime.scanstack runtime.scanframeworker".split_whitespace() { assert_eq!(stack_bucket(&[name]),"gc","{name}"); }
        for name in "runtime.schedule runtime.findRunnable runtime.gopark runtime.park_m runtime.goready runtime.mcall runtime.gogo runtime.handleEvent runtime.beforeIdle wasm_pc_f_loop runtime.wasm_pc_f_loop".split_whitespace() { assert_eq!(stack_bucket(&[name]),"scheduler","{name}"); }
        for name in "runtime.mallocgcSmallScanNoHeader runtime.makeslice runtime.growslice runtime.newobject runtime.newarray runtime.makemap runtime.mapassign".split_whitespace() { assert_eq!(stack_bucket(&[name]),"allocation","{name}"); }
        for name in "go.temporal.io/server/tests.(*ActivityTestSuite).TestActivity go.temporal.io/server/tests/testcore.NewTestServer go.temporal.io_server_tests.__ActivityTestSuite_.TestActivity go.temporal.io_server_tests_testcore.NewTestServer".split_whitespace() { assert_eq!(stack_bucket(&[name]),"test","{name}"); }
        for name in "go.temporal.io/server/service/history.(*Engine).Execute go.temporal.io_server_service_history.__historyEngineImpl_.StartWorkflowExecution".split_whitespace() { assert_eq!(stack_bucket(&[name]),"server","{name}"); }
        for name in "runtime.memmove runtime/atomic.Store internal/runtime/maps.(*Map).get internal_runtime_maps.memHashFallback".split_whitespace() { assert_eq!(stack_bucket(&[name]),"runtime","{name}"); }
        for name in "encoding/json.Marshal net/http.(*Client).Do testing.tRunner reflect.Value.Call sync.(*Mutex).Lock encoding_json.Marshal sync.__Mutex_.Lock sort.insertionSort_func syscall.fd_write syscall.path_open sort.pdqsort_func".split_whitespace() { assert_eq!(stack_bucket(&[name]),"stdlib","{name}"); }
        for name in "github.com/stretchr/testify/require.NoError main.main type:.eq.go.temporal.io/server.Task <wasm-function[1]> github.com_ncruces_go_sqlite3_wasm_v6.__Module_._sqlite3VdbeExec go.temporal.io_api_workflowservice_v1.__workflowServiceClient_.RespondWorkflowTaskCompleted golang.org_x_net_http2.__Framer_.ReadFrame".split_whitespace() { assert_eq!(stack_bucket(&[name]),"other","{name}"); }
        assert_eq!(
            stack_bucket(&[
                "runtime.mallocgc",
                "runtime.gcAssistAlloc",
                "go.temporal.io/server/tests.Test"
            ]),
            "allocation"
        );
        assert_eq!(stack_bucket(&["runtime.gcDrain", "runtime.schedule"]), "gc");
        assert_eq!(
            stack_bucket(&["runtime.schedule", "runtime.gcDrain"]),
            "scheduler"
        );
        assert_eq!(
            stack_bucket(&["runtime.schedule.func1", "go.temporal.io/server/tests.Test"]),
            "scheduler"
        );
        assert_eq!(
            stack_bucket(&["go.temporal.io/server/tests.Test", "runtime.gcDrain"]),
            "test"
        );
        assert_eq!(
            stack_bucket(&["runtime.memmove", "go.temporal.io/server/tests.Test"]),
            "runtime"
        );
        assert_eq!(
            stack_bucket(&["<wasm-function[1]>", "runtime.gcDrain"]),
            "gc"
        );
        assert_eq!(stack_bucket(&[]), "other");
    }
    #[test]
    fn weighted_thread_tables_preserve_rows_empty_and_guest_weight() {
        let result = decode(PROFILE).unwrap();
        assert_eq!(result.sample_rows, 5);
        assert_eq!(result.guest_sample_weight, 11);
        assert_eq!(result.empty_stack_weight, 7);
        let expected = BTreeMap::from([
            ("gc".into(), 3),
            ("scheduler".into(), 5),
            ("allocation".into(), 0),
            ("test".into(), 2),
            ("server".into(), 0),
            ("runtime".into(), 0),
            ("stdlib".into(), 1),
            ("other".into(), 0),
        ]);
        assert_eq!(result.self_sample_weight, expected);
    }
    #[test]
    fn original_malformed_weight_and_table_mutations_refuse() {
        let raw = std::str::from_utf8(PROFILE).unwrap();
        for (label, before, after) in [
            ("sample length", r#""length":4"#, r#""length":3"#),
            ("weight length", r"[3,2,5,7]", r"[3,2,5]"),
            ("negative weight", r"[3,2,5,7]", r"[-3,2,5,7]"),
            (
                "weight overflow",
                r"[3,2,5,7]",
                r"[9223372036854775807,9223372036854775807,5,7]",
            ),
            (
                "allocation weight",
                r#""weightType":"samples""#,
                r#""weightType":"bytes""#,
            ),
            ("stack length", r#""frame":[2,0,1]"#, r#""frame":[2,0]"#),
            (
                "prefix length",
                r#""prefix":[null,0,0]"#,
                r#""prefix":[null,0]"#,
            ),
            (
                "stack index",
                r#""stack":[1,0,2,null]"#,
                r#""stack":[3,0,2,null]"#,
            ),
            (
                "negative stack",
                r#""stack":[1,0,2,null]"#,
                r#""stack":[-1,0,2,null]"#,
            ),
            ("cycle", r#""prefix":[null,0,0]"#, r#""prefix":[1,0,0]"#),
            ("frame index", r#""frame":[2,0,1]"#, r#""frame":[2,3,1]"#),
            ("function index", r#""func":[1,2,0]"#, r#""func":[3,2,0]"#),
            ("name index", r#""name":[2,0,1]"#, r#""name":[3,0,1]"#),
            ("no interval", r#""interval":1"#, r#""interval":0"#),
        ] {
            let changed = raw.replacen(before, after, 1);
            assert_ne!(
                changed, raw,
                "mutation {label} did not change its real input"
            );
            assert!(decode(changed.as_bytes()).is_err(), "accepted {label}");
        }
        for raw in [
            b"{}".as_slice(),
            b"{".as_slice(),
            br#"{"meta":{"interval":1},"threads":[]}"#.as_slice(),
        ] {
            assert!(decode(raw).is_err());
        }
    }
}

#[cfg(test)]
#[path = "guest_tests.rs"]
mod sampled_tests;
