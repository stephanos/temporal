use super::{Value, bytes, fields, integer, json, require, sha, step, steps, stores};

pub(super) const FINISH: &str = "i32.const 512 i32.const 16 call $config drop
    i32.const 995 i32.const 1 i32.store8
    i32.const 992 i32.const 16 call $finish drop";

pub(super) fn module(body: &str) -> Vec<u8> {
    include_str!("held_failed.wat")
        .replace("unreachable))", &format!("{body}))"))
        .into_bytes()
}

#[expect(
    clippy::too_many_lines,
    reason = "The fixed capacity observation retains its complete request, namespace, model and store predicates together"
)]
pub(super) fn validate(variant: &str, o: &Value) -> Result<(), String> {
    fields(
        o,
        &[
            "module_bytes",
            "module_sha256",
            "request",
            "terminal",
            "exit_code",
            "message",
            "fuel_remaining",
            "offered_fuel",
            "memory_bytes",
            "peak_memory_bytes",
            "output_bytes",
            "local_model",
            "choice_bytes",
            "transcript_bytes",
            "executed_steps",
            "store_results",
            "activity",
        ],
    )?;
    let configured = variant == "memory-capacity-after-bootstrap";
    let body = if configured {
        "i32.const 512 i32.const 16 call $config drop i32.const 1 memory.grow drop"
    } else {
        "i32.const 1 memory.grow drop"
    };
    let raw = bytes(&o["module_bytes"])?;
    require(
        raw == module(body) && o["module_sha256"] == sha(&raw),
        "capacity actual module",
    )?;
    let request = &o["request"];
    fields(
        request,
        &[
            "limits",
            "model_configuration",
            "arguments",
            "environment",
            "replay_trace",
            "namespace",
            "storage",
        ],
    )?;
    require(
        request["limits"]
            == json!({"memory_bytes":65_536,"fuel":1_000_000,"output_bytes":1024,"evidence_bytes":1<<20,"wall_timeout_millis":5000})
            && request["arguments"] == json!([])
            && request["environment"] == json!(["PWD=/workspace"])
            && request["replay_trace"].is_null()
            && request["storage"].is_null(),
        "capacity request budget",
    )?;
    let model = json!({"entropy_key":vec![0u8;32],"working_directory_sha256":sha(b"/workspace"),
        "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
        "seed":7,"mode":1,"limit":1_048_576,"diagnostics":false,"replay":null,"epoch_nanos":946_684_800_000_000_000u64,
        "read_step_nanos":0,"now":1,"limits":{"calls":1_048_576,"transcript_bytes":1_048_576,"pending_events":1024},
        "digest_only":false,"split":false,"inspection":null});
    let namespace = json!({"entries":[{"path":b"/workspace","directory":true,"readonly":false,"data":[]}],
        "writable_directories":[b"/workspace"],"preopens":[b"/workspace"],"working_directory":b"/workspace","stdin":[],
        "epoch_nanos":946_684_800_000_000_000u64,"limits":{"files":100,"descriptors":16,"filesystem_bytes":4096,"output_bytes":1024}});
    require(
        request["model_configuration"] == model && request["namespace"] == namespace,
        "capacity request ownership",
    )?;
    require(
        o["terminal"] == "capacity"
            && o["exit_code"].is_null()
            && o["message"]
                .as_str()
                .is_some_and(|s| s.contains("guest linear memory capacity")),
        "capacity terminal class",
    )?;
    require(
        o["offered_fuel"] == 1_000_000
            && o["memory_bytes"] == 65_536
            && o["peak_memory_bytes"] == 65_536
            && o["output_bytes"] == 4
            && (1..=1_000_000).contains(&integer(&o["fuel_remaining"])?),
        "capacity actual resources",
    )?;
    steps(
        &o["executed_steps"],
        &json!([step("start", "a"), step("wait", "a")]),
    )?;
    let actual = stores(&o["store_results"], &o["activity"], Some(&["a/1"]), true)?;
    require(
        actual[0]["terminal"] == o["terminal"]
            && actual[0]["exit_code"].is_null()
            && actual[0]["message"] == o["message"]
            && actual[0]["stdout"] == json!(b"mark"),
        "capacity store owner",
    )?;
    let model = &o["local_model"];
    fields(
        model,
        &[
            "now",
            "advances",
            "calls",
            "records",
            "trace_sha256",
            "branching_records",
            "tape_records",
            "tape_sha256",
            "next_expected",
            "cursor",
            "configured",
            "finished",
            "inspection_sealed",
            "peak",
            "elapsed",
            "terminal_base64",
            "transcript_bytes",
            "transcript_sha256",
        ],
    )?;
    require(
        model["configured"] == configured
            && model["finished"] == false
            && model["inspection_sealed"] == false
            && integer(&model["calls"])? == 1 + u64::from(configured),
        "interrupted model lifetime/calls",
    )?;
    integer(&model["now"])?;
    for key in [
        "advances",
        "records",
        "branching_records",
        "tape_records",
        "cursor",
        "peak",
        "elapsed",
    ] {
        require(
            integer(&model[key])? == 0,
            "interrupted model invented progress",
        )?;
    }
    require(
        model["next_expected"].is_null()
            && model["tape_sha256"] == ""
            && model["terminal_base64"] == "",
        "interrupted model invented replay",
    )?;
    let choice = bytes(&o["choice_bytes"])?;
    let transcript = bytes(&o["transcript_bytes"])?;
    let mut expected = b"GMWT\x01\0\0\0".to_vec();
    expected.extend_from_slice(&[8, 0, 0, 0]);
    expected.extend_from_slice(&1u64.to_le_bytes());
    for n in [16u32, 1, 4] {
        expected.extend_from_slice(&n.to_le_bytes());
    }
    expected.extend_from_slice(b"mark");
    expected.extend_from_slice(&4u32.to_le_bytes());
    require(
        choice.is_empty() && transcript == expected,
        "interrupted completed fd_write prefix",
    )?;
    require(
        integer(&model["transcript_bytes"])? == transcript.len() as u64
            && model["transcript_sha256"] == sha(&transcript)
            && model["trace_sha256"] == sha(&choice),
        "interrupted evidence commitments",
    )
}
