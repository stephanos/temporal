use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};

fn take<'a>(input: &mut &'a [u8], count: usize) -> &'a [u8] {
    let (value, rest) = input.split_at(count);
    *input = rest;
    value
}
fn word(input: &mut &[u8]) -> u32 {
    u32::from_le_bytes(take(input, 4).try_into().unwrap())
}
fn number(input: &mut &[u8]) -> u64 {
    u64::from_le_bytes(take(input, 8).try_into().unwrap())
}
fn handle(input: &mut &[u8]) -> Value {
    let length = word(input) as usize;
    assert!((1..=1024).contains(&length));
    let node = std::str::from_utf8(take(input, length)).unwrap();
    json!({"node":node,"incarnation":number(input)})
}
fn witness(input: &mut &[u8]) -> Value {
    let raw = take(input, 128);
    let n = |at| u64::from_be_bytes(raw[at..at + 8].try_into().unwrap());
    let w = |at| u32::from_be_bytes(raw[at..at + 4].try_into().unwrap());
    let counters: Vec<u64> = (0..7).map(|index| n(40 + index * 8)).collect();
    let draws = counters
        .iter()
        .fold(0u64, |sum, value| sum.wrapping_add(*value));
    assert!(n(0) >= n(104));
    assert_ne!(n(112), 0);
    json!({"runtime_state":STANDARD.encode(raw),"allocations":n(16),"allocated_bytes":n(96),
      "gc_cycles":w(24),"gc_phase":raw[28],"goroutines":n(112),"run_queue":w(32),
      "global_run_queue":n(120),"seeded_draws":draws,"seeded_draw_counters":counters,
      "choices":n(104),"observations":n(0)-n(104)})
}
pub fn boundaries(evidence: &Value) -> Vec<Value> {
    assert_eq!(
        evidence["boundary_encoding"],
        "gomad.inspection-boundaries-binary/v1"
    );
    let bytes = STANDARD
        .decode(evidence["boundary_frames"].as_str().unwrap())
        .unwrap();
    let mut input = bytes.as_slice();
    assert_eq!(take(&mut input, 8), b"GMWIB\x01\0\0");
    let count = word(&mut input);
    assert!(count <= 65536);
    let mut boundaries = Vec::new();
    for _ in 0..count {
        let length = word(&mut input) as usize;
        let mut frame = take(&mut input, length);
        let owner = handle(&mut frame);
        let mut boundary = json!({"handle":owner,"decision":number(&mut frame),"trace":number(&mut frame),
          "order":number(&mut frame),"time_nanos":number(&mut frame),"world_bytes":number(&mut frame),
          "lifecycle_bytes":number(&mut frame)});
        let cut_count = word(&mut frame);
        assert!((1..=65536).contains(&cut_count));
        let mut cuts = Vec::new();
        for _ in 0..cut_count {
            let mut cut = json!({"handle":handle(&mut frame),"state_encoding":evidence["state_encoding"],
              "choice_records":number(&mut frame),"choice_bytes":number(&mut frame),"wasi_bytes":number(&mut frame),
              "network_bytes":number(&mut frame),"storage_bytes":number(&mut frame),"stdout_bytes":number(&mut frame),
              "stderr_bytes":number(&mut frame)});
            let clock = word(&mut frame) as usize;
            assert!(clock <= 1024);
            cut["clock_run"] = json!(STANDARD.encode(take(&mut frame, clock)));
            for key in ["network_state", "storage_state", "recovery_state"] {
                cut[key] = json!(STANDARD.encode(take(&mut frame, 32)));
            }
            cut["witness"] = witness(&mut frame);
            cuts.push(cut);
        }
        assert_eq!(frame, [] as [u8; 0]);
        boundary["cuts"] = json!(cuts);
        boundaries.push(boundary);
    }
    assert_eq!(input, [] as [u8; 0]);
    boundaries
}
