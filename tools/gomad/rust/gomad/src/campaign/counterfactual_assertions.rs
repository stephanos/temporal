use crate::evidence::store::Opened;
use gomad_wasmhost::linked_inspection;
use serde_json::Value;
use std::path::Path;

pub(super) fn inspected_prefix(inspection: &Value, report: &Value) {
    assert_eq!(inspection["status"], "inspected_prefix", "{inspection}");
    assert_eq!(inspection["prefix_verified"], true, "{inspection}");
    assert_eq!(inspection["witness_match"], true, "{inspection}");
    assert_eq!(
        inspection["resolved"]["choice"],
        report["reproducer"]["stop"]["stop"]["value"]
    );
}

pub(super) fn original_predicates(case: &str, report: &Value, pair: &[Value]) {
    let open = |row: &Value| {
        let opened = Opened::open(Path::new(row["artifact"].as_str().unwrap())).unwrap();
        opened.authenticate_all().unwrap();
        opened
    };
    let a = open(&pair[0]);
    let b = open(&pair[1]);
    if case == "draw" {
        let before: Value =
            serde_json::from_str(report["first"]["expected"].as_str().unwrap()).unwrap();
        let after: Value =
            serde_json::from_str(report["first"]["actual"].as_str().unwrap()).unwrap();
        assert_eq!(
            before["value"]["runq_draws"].as_u64().unwrap() + 1,
            after["value"]["runq_draws"]
        );
    }
    if matches!(case, "draw" | "clock") {
        let metadata = a
            .payload(
                "evidence/inspection",
                linked_inspection::MAX_METADATA as u64,
            )
            .unwrap();
        let evidence = linked_inspection::Evidence::decode(&metadata).unwrap();
        assert!(evidence.boundaries.len() >= 9);
        let left = a.payload("evidence/diagnostic", 1 << 20).unwrap();
        let right = b.payload("evidence/diagnostic", 1 << 20).unwrap();
        let (left, right) = left
            .as_chunks::<96>()
            .0
            .iter()
            .zip(right.as_chunks::<96>().0.iter())
            .find(|(left, right)| left != right)
            .expect("original diagnostic difference");
        let field = if case == "draw" { 40..48 } else { 8..16 };
        assert_eq!(&left[..field.start], &right[..field.start]);
        assert_eq!(&left[field.end..], &right[field.end..]);
        let before = u64::from_be_bytes(left[field.clone()].try_into().unwrap());
        let after = u64::from_be_bytes(right[field].try_into().unwrap());
        assert_eq!(before.checked_add(1), Some(after));
    }
    if case == "output-clock" {
        let raw = b.payload("evidence/event-order", 32 << 20).unwrap();
        let frames = std::str::from_utf8(&raw)
            .unwrap()
            .lines()
            .map(|line| {
                serde_json::from_str::<Value>(
                    line.strip_prefix("gomad-wasm-event-order-v1 ").unwrap(),
                )
                .unwrap()
            })
            .collect::<Vec<_>>();
        let output = report["first"]["coordinate"]["Shared"].as_u64().unwrap();
        let boundary = frames
            .iter()
            .filter(|frame| frame["type"] == "boundary")
            .filter_map(|frame| frame["order"].as_u64())
            .filter(|order| *order >= output)
            .min()
            .unwrap();
        let clock = frames
            .iter()
            .filter(|frame| frame["type"] == "event" && frame["kind"] == "clock")
            .filter_map(|frame| frame["order"].as_u64())
            .find(|order| *order > output)
            .unwrap();
        assert!(clock > output && clock <= boundary);
    }
}
