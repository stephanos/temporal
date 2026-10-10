use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::io::{BufRead, BufReader, Write};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};

fn run(wat: &str, change: Value, reply: impl FnMut(&Value) -> Value) -> Vec<Value> {
    run_authorized(wat, change, reply, false)
}
fn run_authorized(
    wat: &str,
    change: Value,
    mut reply: impl FnMut(&Value) -> Value,
    mismatch: bool,
) -> Vec<Value> {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    let path = std::env::temp_dir().join(format!(
        "gomad3-wasmhost-{}-{}.wat",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ));
    std::fs::write(&path, wat).unwrap();
    let mut request = json!({"schema":"gomad3.wasm-host/v1", "type":"execute", "module_path":path, "module_sha256":format!("{:x}",Sha256::digest(wat.as_bytes())), "memory_bytes":131072, "fuel":1000000, "output_bytes":1024});
    for (key, value) in change.as_object().unwrap() {
        request[key] = value.clone();
    }
    let mut child = Command::new(env!("CARGO_BIN_EXE_gomad3-wasmhost"))
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit())
        .spawn()
        .unwrap();
    let mut input = child.stdin.take().unwrap();
    writeln!(input, "{request}").unwrap();
    let mut frames = Vec::new();
    for line in BufReader::new(child.stdout.take().unwrap()).lines() {
        let frame: Value = serde_json::from_str(&line.unwrap()).unwrap();
        if frame["type"] == "started" {
            writeln!(input, "{}",json!({"type":"authorize","module_sha256":if mismatch {"bad"}else{frame["module_sha256"].as_str().unwrap()},"configuration":frame["engine"]["configuration"]})).unwrap();
        }
        if frame["type"] == "call" {
            writeln!(input, "{}", reply(&frame)).unwrap();
        }
        frames.push(frame);
    }
    assert!(child.wait().unwrap().success());
    std::fs::remove_file(path).unwrap();
    frames
}
fn deny(call: &Value) -> Value {
    json!({"type":"reply", "id":call["id"], "errno":0, "output":{}, "unsupported":true, "message":"not modeled"})
}
fn termination(frames: &[Value]) -> &str {
    frames.last().unwrap()["termination"].as_str().unwrap()
}

#[test]
fn executes_tiny_guest_in_fresh_instances() {
    let guest = "(module (memory (export \"memory\") 1) (global $g (mut i32) (i32.const 0)) (func (export \"_start\") global.get $g if unreachable end i32.const 1 global.set $g))";
    for _ in 0..2 {
        let frames = run(guest, json!({}), deny);
        assert_eq!(termination(&frames), "exit", "{frames:?}");
        assert_eq!(frames[0]["engine"]["version"], "47.0.3");
    }
}
#[test]
fn bounds_infinite_loop_with_fuel() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\") (loop $l br $l)))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.last().unwrap()["fuel_remaining"], 0);
}
#[test]
fn rejects_unknown_import_before_started() {
    let frames = run(
        "(module (import \"env\" \"clock\" (func)) (memory (export \"memory\") 1) (func (export \"_start\")))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "unsupported");
    assert_eq!(frames.len(), 1);
}
#[test]
fn rejects_changed_module_identity() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\")))",
        json!({"module_sha256":"0000000000000000000000000000000000000000000000000000000000000000"}),
        deny,
    );
    assert_eq!(termination(&frames), "invalid");
    assert_eq!(frames.len(), 1);
}
#[test]
fn bounds_initial_memory() {
    let frames = run(
        "(module (memory (export \"memory\") 3) (func (export \"_start\")))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "capacity");
}
#[test]
fn routes_guest_output_without_polluting_protocol() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"fd_write\" (func $write (param i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 32) \"hi\") (func (export \"_start\") i32.const 0 i32.const 32 i32.store i32.const 4 i32.const 2 i32.store i32.const 1 i32.const 0 i32.const 1 i32.const 8 call $write drop i32.const 8 i32.load i32.const 2 i32.ne if unreachable end))";
    let frames = run(guest, json!({}), |call| {
        assert_eq!(call["op"], "fd_write");
        assert_eq!(call["input"], json!({"fd":1,"data_base64":"aGk="}));
        json!({"type":"reply","id":call["id"],"errno":0,"output":{"written":2}})
    });
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn rejects_wrong_import_signature() {
    let frames = run(
        "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func (param i64) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\")))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "unsupported");
    assert_eq!(frames.len(), 1);
}
#[test]
fn rejects_malformed_execute_and_unknown_fields() {
    for change in [
        json!({"schema":"other"}),
        json!({"fuel":0}),
        json!({"memory_bytes":1}),
        json!({"surprise":true}),
    ] {
        let frames = run(
            "(module (memory (export \"memory\") 1) (func (export \"_start\")))",
            change,
            deny,
        );
        assert_eq!(termination(&frames), "invalid");
        assert_eq!(frames.len(), 1);
    }
}
#[test]
fn traps_growth_at_declared_limit() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\") i32.const 2 memory.grow drop))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "capacity");
}
#[test]
fn checks_abi_range_before_callback() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func $random (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const -1 i32.const 16 call $random drop))";
    let frames = run(guest, json!({}), |_| {
        panic!("invalid memory reached host callback")
    });
    assert_eq!(termination(&frames), "trap");
}
#[test]
fn writes_host_entropy_into_guest_memory() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func $random (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const 2 call $random drop i32.const 0 i32.load16_u i32.const 513 i32.ne if unreachable end))";
    let frames = run(guest, json!({}), |call| {
        assert_eq!(call["input"], json!({"length":2}));
        json!({"type":"reply","id":call["id"],"errno":0,"output":{"data_base64":"AQI="}})
    });
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn rejects_malformed_or_mismatched_callback_reply() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func $random (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const 2 call $random drop))";
    for output in [
        json!({"data_base64":"AQ=="}),
        json!({"data_base64":"%%%"}),
        json!({"data_base64":"AQI=","extra":1}),
    ] {
        let frames = run(
            guest,
            json!({}),
            |call| json!({"type":"reply","id":call["id"],"errno":0,"output":output}),
        );
        assert_eq!(termination(&frames), "invalid");
    }
    let frames = run(
        guest,
        json!({}),
        |_| json!({"type":"reply","id":99,"errno":0,"output":{"data_base64":"AQI="}}),
    );
    assert_eq!(termination(&frames), "invalid");
}
#[test]
fn does_not_write_memory_for_errno() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"random_get\" (func $random (param i32 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 0) \"zz\") (func (export \"_start\") i32.const 0 i32.const 2 call $random i32.const 8 i32.ne if unreachable end i32.const 0 i32.load16_u i32.const 31354 i32.ne if unreachable end))";
    let frames = run(
        guest,
        json!({}),
        |call| json!({"type":"reply","id":call["id"],"errno":8,"output":{}}),
    );
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn denies_unsupported_callback_explicitly() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"sock_shutdown\" (func $shutdown (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 3 i32.const 2 call $shutdown drop))";
    let frames = run(guest, json!({}), deny);
    assert_eq!(termination(&frames), "unsupported");
}
#[test]
fn enforces_output_limit_before_delivery() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"fd_write\" (func $write (param i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 32) \"hi\") (func (export \"_start\") i32.const 0 i32.const 32 i32.store i32.const 4 i32.const 2 i32.store i32.const 1 i32.const 0 i32.const 1 i32.const 8 call $write drop))";
    let frames = run(guest, json!({"output_bytes":1}), |_| {
        panic!("over-limit output reached host")
    });
    assert_eq!(termination(&frames), "capacity");
}

#[test]
fn rejects_authorization_before_guest_start_section() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"sched_yield\" (func $yield (result i32))) (memory (export \"memory\") 1) (func $init call $yield drop) (start $init) (func (export \"_start\")))";
    let frames = run_authorized(guest, json!({}), deny, true);
    assert_eq!(termination(&frames), "invalid");
    assert_eq!(
        frames.len(),
        2,
        "guest code ran before authorization: {frames:?}"
    );
}

fn checked_call(
    name: &str,
    signature: &str,
    body: &str,
    expected_input: Value,
    output: Value,
    check: &str,
) {
    let guest = format!(
        "(module (import \"wasi_snapshot_preview1\" \"{name}\" (func $call {signature} (result i32))) (memory (export \"memory\") 1) (data (i32.const 256) \"file\") (func (export \"_start\") {body} call $call drop {check}))"
    );
    let frames = run(&guest, json!({}), |call| {
        assert_eq!(call["op"], name);
        assert_eq!(call["input"], expected_input);
        json!({"type":"reply","id":call["id"],"errno":0,"output":output})
    });
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn marshals_clock_and_file_metadata_into_preview1_layouts() {
    checked_call(
        "clock_time_get",
        "(param i32 i64 i32)",
        "i32.const 1 i64.const 7 i32.const 64",
        json!({"clock_id":1,"precision":7}),
        json!({"timestamp":42}),
        "i32.const 64 i64.load i64.const 42 i64.ne if unreachable end",
    );
    checked_call(
        "fd_fdstat_get",
        "(param i32 i32)",
        "i32.const 4 i32.const 64",
        json!({"fd":4}),
        json!({"filetype":4,"flags":2,"rights_base":9,"rights_inheriting":7}),
        "i32.const 64 i32.load8_u i32.const 4 i32.ne if unreachable end i32.const 66 i32.load16_u i32.const 2 i32.ne if unreachable end i32.const 72 i64.load i64.const 9 i64.ne if unreachable end i32.const 80 i64.load i64.const 7 i64.ne if unreachable end",
    );
    checked_call(
        "fd_filestat_get",
        "(param i32 i32)",
        "i32.const 4 i32.const 64",
        json!({"fd":4}),
        json!({"dev":1,"ino":2,"filetype":4,"nlink":3,"size":9,"atim":11,"mtim":12,"ctim":13}),
        "i32.const 80 i32.load8_u i32.const 4 i32.ne if unreachable end i32.const 96 i64.load i64.const 9 i64.ne if unreachable end i32.const 120 i64.load i64.const 13 i64.ne if unreachable end",
    );
}
#[test]
fn decodes_path_open_rights_without_exposing_guest_pointers() {
    checked_call(
        "path_open",
        "(param i32 i32 i32 i32 i32 i64 i64 i32 i32)",
        "i32.const 3 i32.const 1 i32.const 256 i32.const 4 i32.const 9 i64.const 8192 i64.const 64 i32.const 2 i32.const 64",
        json!({"fd":3,"dirflags":1,"path_base64":"ZmlsZQ==","oflags":9,"rights_base":8192,"rights_inheriting":64,"fdflags":2}),
        json!({"fd":8}),
        "i32.const 64 i32.load i32.const 8 i32.ne if unreachable end",
    );
}
#[test]
fn scatters_partial_positional_read_into_iovecs() {
    checked_call(
        "fd_pread",
        "(param i32 i32 i32 i64 i32)",
        "i32.const 0 i32.const 64 i32.store i32.const 4 i32.const 2 i32.store i32.const 8 i32.const 80 i32.store i32.const 12 i32.const 2 i32.store i32.const 4 i32.const 0 i32.const 2 i64.const 9 i32.const 32",
        json!({"fd":4,"length":4,"offset":9}),
        json!({"data_base64":"AQID"}),
        "i32.const 32 i32.load i32.const 3 i32.ne if unreachable end i32.const 64 i32.load16_u i32.const 513 i32.ne if unreachable end i32.const 80 i32.load16_u i32.const 3 i32.ne if unreachable end",
    );
}
#[test]
fn marshals_ordered_poll_subscriptions_and_events() {
    checked_call(
        "poll_oneoff",
        "(param i32 i32 i32 i32)",
        "i32.const 0 i64.const 91 i64.store i32.const 8 i32.const 0 i32.store8 i32.const 16 i32.const 1 i32.store i32.const 24 i64.const 123 i64.store i32.const 32 i64.const 7 i64.store i32.const 40 i32.const 1 i32.store16 i32.const 0 i32.const 64 i32.const 1 i32.const 128",
        json!({"subscriptions":[{"userdata":91,"type":0,"clock_id":1,"timeout":123,"precision":7,"flags":1}]}),
        json!({"events":[{"userdata":91,"errno":0,"type":0,"nbytes":0,"flags":0}]}),
        "i32.const 64 i64.load i64.const 91 i64.ne if unreachable end i32.const 128 i32.load i32.const 1 i32.ne if unreachable end",
    );
}
#[test]
fn packs_captured_arguments_using_preceding_sizes() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"args_sizes_get\" (func $sizes (param i32 i32) (result i32))) (import \"wasi_snapshot_preview1\" \"args_get\" (func $get (param i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const 4 call $sizes drop i32.const 8 i32.const 32 call $get drop i32.const 8 i32.load i32.const 32 i32.ne if unreachable end i32.const 32 i32.load16_u i32.const 97 i32.ne if unreachable end i32.const 12 i32.load i32.const 34 i32.ne if unreachable end))";
    let frames = run(
        guest,
        json!({}),
        |call| json!({"type":"reply","id":call["id"],"errno":0,"output":if call["op"]=="args_sizes_get"{json!({"count":2,"bytes":3})}else{json!({"values":["YQ==",""]})}}),
    );
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
#[test]
fn rejects_out_of_bounds_iovec_without_callback() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"fd_read\" (func $read (param i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 1) (func (export \"_start\") i32.const 0 i32.const -1 i32.store i32.const 4 i32.const 2 i32.store i32.const 0 i32.const 0 i32.const 1 i32.const 8 call $read drop))";
    let frames = run(guest, json!({}), |_| panic!("invalid iovec reached host"));
    assert_eq!(termination(&frames), "trap");
}

#[test]
fn bounds_positional_writes_to_standard_output() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"fd_pwrite\" (func $write (param i32 i32 i32 i64 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 32) \"hi\") (func (export \"_start\") i32.const 0 i32.const 32 i32.store i32.const 4 i32.const 2 i32.store i32.const 1 i32.const 0 i32.const 1 i64.const 0 i32.const 8 call $write drop))";
    let frames = run(
        guest,
        json!({"output_bytes":1}),
        |call| json!({"type":"reply","id":call["id"],"errno":0,"output":{"written":2}}),
    );
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.len(), 2, "over-limit positional output reached host");
}
#[test]
fn classifies_initial_table_count_as_capacity() {
    let guest = format!(
        "(module {} (memory (export \"memory\") 1) (func (export \"_start\")))",
        "(table 0 funcref)".repeat(9)
    );
    let frames = run(&guest, json!({}), deny);
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.len(), 1);
}
#[test]
fn bounds_aggregate_callback_frame() {
    let guest = "(module (import \"wasi_snapshot_preview1\" \"path_rename\" (func $rename (param i32 i32 i32 i32 i32 i32) (result i32))) (memory (export \"memory\") 112) (func (export \"_start\") i32.const 3 i32.const 0 i32.const 3145728 i32.const 3 i32.const 3145728 i32.const 3145728 call $rename drop))";
    let frames = run(guest, json!({"memory_bytes":8388608}), deny);
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.len(), 2);
}
#[test]
fn classifies_disabled_valid_proposals_as_unsupported() {
    for guest in [
        "(module (memory (export \"memory\") i64 1) (func (export \"_start\")))",
        "(module (memory (export \"memory\") 1 2 shared) (func (export \"_start\")))",
        "(module (memory (export \"memory\") 1) (memory 1) (func (export \"_start\")))",
        "(module (memory (export \"memory\") 1) (func (export \"_start\") v128.const f32x4 0 0 0 0 v128.const f32x4 0 0 0 0 v128.const f32x4 0 0 0 0 f32x4.relaxed_madd drop))",
    ] {
        let frames = run(guest, json!({}), deny);
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert_eq!(frames.len(), 1);
    }
}
#[test]
fn keeps_malformed_wat_and_binary_as_invalid() {
    for guest in [
        "(module (memory",
        "(module (memory (export \"memory\") 1) (func (export \"_start\") i32.const 1))",
        "\0asm\x01\0\0\0\x05\x01\x01",
    ] {
        let frames = run(guest, json!({}), deny);
        assert_eq!(termination(&frames), "invalid");
        assert_eq!(frames.len(), 1);
    }
}
#[test]
fn classifies_missing_module_as_infrastructure() {
    let frames = run(
        "(module)",
        json!({"module_path":"/missing-gomad3-wasmhost-module"}),
        deny,
    );
    assert_eq!(termination(&frames), "infrastructure");
}
#[test]
fn bounds_started_import_inventory() {
    let guest=format!("(module {} (memory (export \"memory\") 1) (func (export \"_start\")))","(import \"wasi_snapshot_preview1\" \"fd_write\" (func (param i32 i32 i32 i32) (result i32)))".repeat(100000));
    let frames = run(&guest, json!({}), deny);
    assert_eq!(termination(&frames), "capacity");
    assert_eq!(frames.len(), 1);
}

#[test]
fn bounds_unsupported_import_diagnostics() {
    let guest = format!(
        "(module (import \"env\" \"{}\" (func)) (memory (export \"memory\") 1) (func (export \"_start\")))",
        "x".repeat(90000)
    );
    let frames = run(&guest, json!({}), deny);
    assert_eq!(termination(&frames), "unsupported");
    assert!(frames.last().unwrap()["message"].as_str().unwrap().len() <= 65536);
}

#[test]
fn classifies_disabled_atomic_instructions_as_unsupported() {
    for instruction in ["atomic.fence", "i32.const 0 i32.atomic.load drop"] {
        let guest = format!(
            "(module (memory (export \"memory\") 1) (func (export \"_start\") {instruction}))"
        );
        let frames = run(&guest, json!({}), deny);
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert_eq!(frames.len(), 1);
    }
}
#[test]
fn classifies_real_guest_unreachable_as_trap() {
    let frames = run(
        "(module (memory (export \"memory\") 1) (func (export \"_start\") unreachable))",
        json!({}),
        deny,
    );
    assert_eq!(termination(&frames), "trap");
}

fn runtime_guest(name: &str, pointer: i32, length: i32, data: &[u8], check: &str) -> String {
    let data: String = data.iter().map(|byte| format!("\\{byte:02x}")).collect();
    format!(
        "(module (import \"gomad_wasm_v1\" \"{name}\" (func $call (param i32 i32) (result i32))) (memory (export \"memory\") 1) (data (i32.const 64) \"{data}\") (func (export \"_start\") i32.const {pointer} i32.const {length} call $call {check}))"
    )
}

#[test]
fn runtime_imports_copy_bounded_buffers_back_into_guest_memory() {
    for (name, length, count) in [
        ("config", 16, 0),
        ("finish", 16, 0),
        ("idle", 16, 0),
        ("decision", 96, 2),
        ("decision", 192, 2),
        ("decision", 8224, 256),
        ("decision", 8320, 256),
        ("observation", 96, 0),
        ("observation", 192, 0),
    ] {
        let mut bytes = vec![0u8; length];
        if count != 0 {
            bytes[20..24].copy_from_slice(&(count as u32).to_be_bytes());
        }
        let guest = runtime_guest(
            name,
            64,
            length as i32,
            &bytes,
            "if unreachable end i32.const 64 i32.load8_u i32.const 42 i32.ne if unreachable end",
        );
        let frames = run(&guest, json!({}), |call| {
            assert_eq!(call["op"], format!("runtime_{name}"));
            assert_eq!(
                call["input"],
                json!({"data_base64":STANDARD.encode(&bytes)})
            );
            let mut output = bytes.clone();
            output[0] = 42;
            json!({"type":"reply","id":call["id"],"errno":0,"output":{"data_base64":STANDARD.encode(output)}})
        });
        assert_eq!(termination(&frames), "exit", "{name}/{length}: {frames:?}");
    }
}

#[test]
fn runtime_imports_require_exact_module_name_and_signature_before_started() {
    for (module, name, signature) in [
        ("gomad_wasm_v1", "other", "(param i32 i32) (result i32)"),
        (
            "gomad_wasm_v1",
            "runtime_config",
            "(param i32 i32) (result i32)",
        ),
        (
            "gomad_wasm_v1",
            "random_get",
            "(param i32 i32) (result i32)",
        ),
        (
            "wasi_snapshot_preview1",
            "config",
            "(param i32 i32) (result i32)",
        ),
        ("other", "config", "(param i32 i32) (result i32)"),
        ("gomad_wasm_v1", "config", "(param i64 i32) (result i32)"),
        ("gomad_wasm_v1", "config", "(param i32 i32)"),
    ] {
        let guest = format!(
            "(module (import \"{module}\" \"{name}\" (func {signature})) (memory (export \"memory\") 1) (func (export \"_start\")))"
        );
        let frames = run(&guest, json!({}), deny);
        assert_eq!(termination(&frames), "unsupported", "{frames:?}");
        assert_eq!(frames.len(), 1, "{frames:?}");
    }
}

#[test]
fn runtime_imports_reject_invalid_ranges_and_layouts_before_callback() {
    for (name, pointer, length, count, expected) in [
        ("config", -1, 16, 0, "trap"),
        ("config", 65528, 16, 0, "trap"),
        ("config", 64, 15, 0, "trap"),
        ("finish", 64, 17, 0, "capacity"),
        ("idle", 64, 0, 0, "trap"),
        ("observation", 64, 97, 0, "trap"),
        ("decision", 64, 96, 1, "trap"),
        ("decision", 64, 96, 257, "trap"),
        ("decision", 64, 128, 2, "trap"),
        ("decision", 64, 8321, 256, "capacity"),
        ("decision", 64, -1, 2, "capacity"),
    ] {
        let mut bytes = [0u8; 32];
        bytes[20..24].copy_from_slice(&(count as u32).to_be_bytes());
        let guest = runtime_guest(name, pointer, length, &bytes, "drop");
        let frames = run(&guest, json!({}), |_| {
            panic!("invalid {name} {pointer}/{length}/{count} reached host")
        });
        assert_eq!(
            termination(&frames),
            expected,
            "{name}/{length}: {frames:?}"
        );
        assert_eq!(frames.len(), 2, "{frames:?}");
    }
}

#[test]
fn runtime_imports_reject_malformed_and_wrong_length_replies() {
    let guest = runtime_guest("config", 64, 16, &[0u8; 16], "drop");
    for output in [
        json!({"data_base64":STANDARD.encode([0u8; 15])}),
        json!({"data_base64":STANDARD.encode([0u8; 17])}),
        json!({"data_base64":"%%%"}),
        json!({"data_base64":42}),
        json!({"data_base64":STANDARD.encode([0u8; 16]),"extra":1}),
        json!({"data_base64":"A".repeat(16384)}),
    ] {
        let frames = run(
            &guest,
            json!({}),
            |call| json!({"type":"reply","id":call["id"],"errno":0,"output":output}),
        );
        assert_eq!(termination(&frames), "invalid", "{frames:?}");
        assert_eq!(frames.len(), 3, "{frames:?}");
    }
}

#[test]
fn runtime_imports_preserve_guest_buffer_for_errno() {
    let guest = runtime_guest(
        "idle",
        64,
        16,
        &[42u8; 16],
        "i32.const 8 i32.ne if unreachable end i32.const 64 i32.load8_u i32.const 42 i32.ne if unreachable end",
    );
    let frames = run(
        &guest,
        json!({}),
        |call| json!({"type":"reply","id":call["id"],"errno":8,"output":{}}),
    );
    assert_eq!(termination(&frames), "exit", "{frames:?}");
}
