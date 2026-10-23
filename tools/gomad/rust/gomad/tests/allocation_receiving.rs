include!("../src/lib.rs");
#[path = "../../allocation_probe.rs"]
mod allocation_measure;

fn main() {
    assert_eq!(
        std::env::args_os().len(),
        1,
        "fixed allocation harness takes no arguments"
    );
    allocation_measure::self_check();
    bounded_opened_file();
    replay::cluster::allocation_probe(|opened, descriptor| {
        let (bytes, facts) =
            allocation_measure::capture(|| opened.payload("evidence/world", 8 << 20).unwrap());
        assert_eq!(bytes.len(), 8 << 20);
        assert!(bytes.iter().all(|byte| *byte == 0xa5));
        allocation_measure::row(
            "opened-payload",
            serde_json::json!({"payload_bytes":8<<20,"limit_bytes":8<<20}),
            facts,
            serde_json::json!({"length":bytes.len(),"sha256":evidence::canonical::digest(&bytes)}),
        );
        assert!(facts.allocated_bytes <= 9 << 20);
        drop(bytes);
        let ((inputs, bytes), facts) = allocation_measure::capture(|| {
            let inputs = replay::cluster::inputs(opened).unwrap();
            let bytes = inputs.retained[0].evidence["world"]
                .read(opened, 8 << 20)
                .unwrap();
            (inputs, bytes)
        });
        assert_eq!(
            inputs.retained[0].evidence["world"].sha256,
            descriptor.sha256
        );
        assert_eq!(bytes.len(), 8 << 20);
        assert!(bytes.iter().all(|byte| *byte == 0xa5));
        allocation_measure::row(
            "retained-decode",
            serde_json::json!({"payload_bytes":8<<20}),
            facts,
            serde_json::json!({"length":bytes.len(),"sha256":evidence::canonical::digest(&bytes),
                "schema":inputs.schema,"descriptor_sha256":descriptor.sha256}),
        );
        assert!(facts.allocated_bytes <= 56 << 20);
    });
}

fn bounded_opened_file() {
    let root = std::env::temp_dir().join(format!(
        "gomad-bounded-allocation-{}",
        campaign::origin::identity().unwrap()
    ));
    std::fs::create_dir(&root).unwrap();
    struct Cleanup(std::path::PathBuf);
    impl Drop for Cleanup {
        fn drop(&mut self) {
            std::fs::remove_dir_all(&self.0).unwrap();
        }
    }
    let _cleanup = Cleanup(root.clone());
    let bytes = vec![b'x'; 1 << 20];
    std::fs::write(root.join("module"), &bytes).unwrap();
    let directory = std::fs::File::open(&root).unwrap();
    let (actual, facts) = allocation_measure::capture(|| {
        evidence::store::open::read(&directory, "module", 512 << 20, false).unwrap()
    });
    assert_eq!(actual, bytes);
    allocation_measure::row(
        "bounded-file-read",
        serde_json::json!({"file_bytes":1<<20,"limit_bytes":512<<20}),
        facts,
        serde_json::json!({"length":actual.len(),"sha256":evidence::canonical::digest(&actual)}),
    );
    assert!(facts.allocated_bytes <= 2 * (1 << 20) + (64 << 10));
}
