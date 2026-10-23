include!("../src/lib.rs");
#[path = "../../allocation_probe.rs"]
mod allocation_measure;

#[test]
#[ignore = "explicit allocation receiving operator harness"]
fn allocation_receiving() {
    allocation_measure::self_check(true);
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
    use std::io::Write as _;
    use std::os::unix::fs::{DirBuilderExt as _, OpenOptionsExt as _};
    struct Cleanup(std::path::PathBuf);
    impl Drop for Cleanup {
        fn drop(&mut self) {
            std::fs::remove_dir_all(&self.0).unwrap();
        }
    }
    let root = std::env::temp_dir().join(format!(
        "gomad-bounded-allocation-{}",
        campaign::origin::identity().unwrap()
    ));
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&root)
        .unwrap();
    let _cleanup = Cleanup(root.clone());
    let bytes = vec![b'x'; 1 << 20];
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(root.join("module"))
        .unwrap();
    file.write_all(&bytes).unwrap();
    drop(file);
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

#[test]
fn terminal_publication_allocation() {
    use measurement::terminal_publication::{
        publish,
        tests::{fixtures, output_path},
    };
    allocation_measure::self_check(false);
    for bytes in [1 << 20, 4 << 20] {
        let (records, admission) = fixtures(bytes);
        let path = output_path("allocation");
        let (result, facts) = allocation_measure::capture(|| {
            publish(
                &path,
                &records,
                None,
                &admission,
                None,
                std::path::Path::new("/fixture/provenance.json"),
            )
        });
        result.unwrap();
        let length = std::fs::metadata(&path).unwrap().len();
        allocation_measure::row(
            &format!("terminal-publication-{bytes}"),
            serde_json::json!({"release_records":6,"raw_stream_bytes_each":bytes}),
            facts,
            serde_json::json!({"ledger_bytes":length}),
        );
        assert!(
            facts.allocated_bytes < 1 << 20,
            "publication allocated {} bytes",
            facts.allocated_bytes
        );
        assert!(length > (bytes * 12) as u64);
        std::fs::remove_file(path).unwrap();
    }
}

#[test]
#[ignore = "explicit retained terminal publication replay; no guest execution"]
fn terminal_publication_retained_replay() {
    use measurement::{
        build, evidence,
        terminal_collection::{Attempt, Record},
        terminal_publication::{Observation, publish},
    };
    use std::path::{Path, PathBuf};
    let root = PathBuf::from(
        std::env::var_os("GOMAD_TERMINAL_REPLAY_ROOT").expect("retained terminal root"),
    );
    let output = PathBuf::from(
        std::env::var_os("GOMAD_TERMINAL_REPLAY_OUTPUT").expect("fresh ledger output"),
    );
    let admission: build::Admission = serde_json::from_slice(
        &std::fs::read(
            std::env::var_os("GOMAD_TERMINAL_REPLAY_ADMISSION")
                .expect("original release admission"),
        )
        .unwrap(),
    )
    .unwrap();
    let provenance = PathBuf::from(
        std::env::var_os("GOMAD_TERMINAL_REPLAY_PROVENANCE").expect("original module provenance"),
    );
    let mut records = Vec::new();
    for label in [
        "prime",
        "terminal-0",
        "terminal-1",
        "terminal-2",
        "terminal-3",
        "terminal-4",
    ] {
        let native_root = root.join("evidence").join(label);
        let request =
            serde_json::from_slice(&std::fs::read(native_root.join("request.json")).unwrap())
                .unwrap();
        let run = evidence::read_profile_run(
            &native_root,
            evidence::ReadLimits::for_request(&request).unwrap(),
        )
        .unwrap();
        records.push(Record {attempt:Attempt {label:label.into(),native_root,role:"release".into()},execution_failure:None,
            observation:Some(Observation::Release {observation:Box::new(run),process_exit:serde_json::json!({"publication_replay_only":true,"original_driver":root.join("evidence").join(format!("{label}-driver.json"))})}),
            retained_paths:vec![],verification_failure:None});
    }
    // These original controls were not both publishable. Replay must remain a nonpass.
    for label in ["module-digest", "loaded-module-descriptor"] {
        records.push(Record {attempt:Attempt {label:label.into(),native_root:root.join("evidence").join(label),role:"diagnostic-profile".into()},
            execution_failure:Some("original failed collection control is not re-executed or promoted by publication replay".into()),observation:None,retained_paths:vec![],verification_failure:None});
    }
    allocation_measure::self_check(true);
    let (result, facts) = allocation_measure::capture(|| {
        publish(
            &output,
            &records,
            Some(
                "publication replay only; original collection failed and controls remain unaccepted",
            ),
            &admission,
            None,
            Path::new(&provenance),
        )
    });
    result.unwrap();
    allocation_measure::row(
        "terminal-publication-retained-replay",
        serde_json::json!({"release_records":6,"original_root":root}),
        facts,
        serde_json::json!({"ledger_bytes":std::fs::metadata(&output).unwrap().len(),"accepted":false}),
    );
    assert!(facts.allocated_bytes < 1 << 20);
}
