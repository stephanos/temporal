use super::*;

fn fixture() -> helper::Evidence {
    helper::Evidence::decode(include_bytes!(
        "../../../wasmhost/tests/testdata/inspection/source-five-decisions.json"
    ))
    .unwrap()
}

fn request() -> InspectionRequest {
    let mut options = Options::default();
    options.set("check", "healthy").unwrap();
    options.set("choice-range", "0:4").unwrap();
    options.set("search-attempts", "10").unwrap();
    options.set("search-timeout", "1s").unwrap();
    options
        .request(Path::new("/retained-source-fixture"))
        .unwrap()
        .unwrap()
}

#[test]
fn search_admission_refuses_all_original_requests_before_evaluation() {
    for variant in [
        "unknown-check",
        "invalid-range",
        "zero-attempts",
        "zero-wall",
        "unreachable-choice",
        "nonmonotonic",
    ] {
        let mut evidence = fixture();
        let mut request = request();
        let search = request.search.as_mut().unwrap();
        match variant {
            "unknown-check" => search.check = "absent".into(),
            "invalid-range" => search.first = 5,
            "zero-attempts" => search.attempt_budget = 0,
            "zero-wall" => search.wall_time = 0,
            "unreachable-choice" => search.last = 5,
            "nonmonotonic" => {
                search.mode = "bisect".into();
                evidence.incarnations[0].registry.entries[0].monotonic = false;
            }
            _ => unreachable!(),
        }
        let error = search_admission(&evidence, &request, request.search.as_ref().unwrap())
            .expect_err(variant);
        println!(
            "{}",
            json!({"schema":"gomad.native-inspection-predicate/v1",
                "original_root":"TestInspectionSearchRejectsRequestsBeforeEvaluation",
                "variant":variant,"observed":{"error":error.message,"search_entered":false,"search":request.search,"evidence":evidence,"boundaries":evidence.boundaries}
            })
        );
    }
}

#[test]
fn retained_resolve_refuses_observation_coordinate_before_creating_control() {
    let mut evidence = fixture();
    let mut request = request();
    request.search = None;
    request.stop = Stop {
        kind: "choice".into(),
        value: 2,
    };
    let valid =
        resolve(&evidence, &request, None).unwrap_or_else(|error| panic!("{}", error.message));
    assert_eq!(valid.target.incarnation, 2);
    assert_eq!(valid.expected.trace, 2);
    let valid_evidence = serde_json::to_value(&evidence).unwrap();
    let valid_boundaries = serde_json::to_value(&evidence.boundaries).unwrap();
    evidence.boundaries[2].trace = 1;
    evidence.boundaries[2].cuts[0].choice_records = 1;
    evidence.boundaries[2].cuts[0].choice_bytes = 96;
    let error = resolve(&evidence, &request, None)
        .err()
        .expect("invalid choice mapping admitted");
    assert!(error.message.contains("decision/trace mapping"));
    println!(
        "{}",
        json!({"schema":"gomad.native-inspection-predicate/v1",
            "original_root":"TestInspectionChoiceMappingUsesDecisionsRatherThanObservations",
            "variant":"production-retained-resolve-gate","observed":{"error":error.message,"evidence":valid_evidence,"boundaries":valid_boundaries,"invalid_boundaries":evidence.boundaries,"expected":valid.expected}
        })
    );
}

#[test]
#[ignore = "requires the retained complete helper scale artifact from GOMAD_INSPECTION_RAW_DIR"]
fn native_retained_scale_admission_preserves_complete_inspection_and_owner_payloads() {
    use crate::evidence::{
        canonical,
        record::Stream,
        store::{Publication, Store},
    };
    let directory = std::env::var_os("GOMAD_INSPECTION_RAW_DIR")
        .expect("the helper scale fixture must run first with the same retained directory");
    let directory = PathBuf::from(directory);
    assert!(directory.is_absolute());
    let metadata = fs::symlink_metadata(&directory).unwrap();
    assert!(metadata.is_dir() && !metadata.file_type().is_symlink());
    let raw_path = directory.join("inspection-scale-native.json");
    let raw = fs::read(&raw_path).unwrap();
    let evidence = helper::Evidence::decode(&raw).unwrap();
    assert!(evidence.complete);
    assert_eq!(evidence.boundaries.len(), 51_000);
    assert_eq!(evidence.incarnations.len(), 1);
    let owner = &evidence.incarnations[0];
    assert_eq!(owner.streams.choice.len(), 51_000 * 96);
    assert_eq!(owner.streams.wasi.len(), 8 + 376_000 * 16);
    assert!(raw.len() <= helper::MAX_METADATA);
    let source = crate::installed_lifecycle_tests::retained(false);
    let opened = Opened::open(&source.published.path).unwrap();
    let mut record = opened.manifest().record.clone();
    record.limits.evidence_bytes = 512 << 20;
    let mut payloads = BTreeMap::new();
    for (name, entry) in &opened.manifest().files {
        payloads.insert(name.clone(), opened.payload(name, entry.bytes).unwrap());
    }
    for (name, bytes) in [
        ("evidence/inspection", raw.clone()),
        ("evidence/world", evidence.world.clone()),
        ("evidence/choice", owner.streams.choice.clone()),
        ("evidence/choice-tape", owner.streams.choice.clone()),
        ("evidence/transcript", owner.streams.wasi.clone()),
    ] {
        record.streams.insert(
            name.into(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 0,
            },
        );
        payloads.insert(name.into(), bytes);
    }
    let record_bytes = record.bytes().unwrap();
    assert!(record_bytes.len() <= 16 << 20);
    payloads.insert("record.json".into(), record_bytes.clone());
    let payload_bytes: u64 = payloads.values().map(|bytes| bytes.len() as u64).sum();
    assert!(payload_bytes <= 512 << 20);
    let published = Store {
        root: directory.join("native-retained-scale"),
        maximum_bytes: 512 << 20,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
    .publish(Publication {
        record: record.clone(),
        payloads,
    })
    .unwrap();
    assert!(published.stored_bytes <= 512 << 20);
    let retained = Opened::open(&published.path).unwrap();
    retained.authenticate_all().unwrap();
    assert_eq!(retained.manifest().record, record);
    let restored = retained
        .payload("evidence/inspection", helper::MAX_METADATA as u64)
        .unwrap();
    assert_eq!(restored, raw);
    let restored = helper::Evidence::decode(&restored).unwrap();
    assert_eq!(
        serde_json::to_value(&restored.boundaries).unwrap(),
        serde_json::to_value(&evidence.boundaries).unwrap()
    );
    assert_eq!(restored.lifecycle, evidence.lifecycle);
    assert_eq!(restored.world, evidence.world);
    for (name, expected) in [
        ("choice", &owner.streams.choice),
        ("choice-tape", &owner.streams.choice),
        ("transcript", &owner.streams.wasi),
    ] {
        assert_eq!(
            retained
                .payload(&format!("evidence/{name}"), 64 << 20)
                .unwrap(),
            *expected
        );
    }
    println!(
        "{}",
        json!({"schema":"gomad.native-inspection-predicate/v1",
            "original_root":"TestInspectionCapacityRetainsTemporalScaleBoundariesAndEffects",
            "variant":"native-retained-scale-admission",
            "observed":{"encoding":"gomad.execution-record/v2; gomad.artifact-manifest/v2; separate native owner payloads",
                "raw_file":raw_path,"raw_sha256":canonical::digest(&raw),"artifact":published.path,
                "record":record,"manifest":retained.manifest(),"payload_bytes":payload_bytes,
                "stored_bytes":published.stored_bytes,"record_bytes":record_bytes.len(),"maximum_bytes":512u64<<20}}
        )
    );
}

#[test]
fn external_search_cancellation_joins_active_evaluator_store_before_return() {
    use std::sync::{Arc, Mutex, mpsc};
    struct Output {
        started: Mutex<Option<mpsc::Sender<()>>>,
        released: Mutex<mpsc::Receiver<()>>,
    }
    impl linked::OutputConsumer for Output {
        fn write(&self, _: linked::OutputStream, _: &[u8]) -> std::io::Result<()> {
            if let Some(started) = self.started.lock().unwrap().take() {
                started.send(()).unwrap();
                self.released.lock().unwrap().recv().unwrap();
            }
            Ok(())
        }
    }
    let source = crate::installed_lifecycle_tests::retained(false);
    let cancellation = linked::Cancellation::default();
    let started = Instant::now();
    let clocks = std::cell::RefCell::new(Vec::new());
    let cancellation_samples = std::cell::RefCell::new(Vec::new());
    let calls = std::cell::RefCell::new(Vec::new());
    let joined_at = std::cell::Cell::new(None);
    let search = request().search.unwrap();
    let result = search::run_with_cancellation(
        &search,
        || {
            let elapsed = started.elapsed();
            clocks.borrow_mut().push(elapsed.as_nanos().to_string());
            elapsed
        },
        |choice, remaining| {
            let module = br#"(module
                (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
                (memory (export "memory") 1)
                (data (i32.const 0) "\08\00\00\00\01\00\00\00x")
                (func (export "_start")
                    i32.const 1 i32.const 0 i32.const 1 i32.const 16 call $write drop
                    (loop $wait br $wait)))"#.to_vec();
            let mut execution = linked::prepare(linked::Request {
                module_sha256: crate::evidence::canonical::digest(&module), module,
                limits: linked::Limits {memory_bytes:64<<20,fuel:1_000_000,output_bytes:4096,evidence_bytes:4096,wall_timeout:remaining},
                model:None,namespace:None,storage:None,
            }).unwrap();
            let (entered, entered_receiver) = mpsc::channel();
            let (release, released) = mpsc::channel();
            execution.output(Arc::new(Output {started:Mutex::new(Some(entered)),released:Mutex::new(released)}));
            let prepared = preflight::Prepared {opened:Opened::open(&source.published.path).unwrap(),execution:preflight::Execution::Standalone(Box::new(execution)),exact:true,available:true};
            let worker_cancellation = cancellation.clone();
            let worker = std::thread::spawn(move || preflight::execute_inspection_with_cancellation(
                prepared,&helper::Handle::default(),linked::Conditions::default(),Some(remaining),Some(&worker_cancellation)));
            entered_receiver.recv_timeout(remaining).unwrap();
            cancellation.cancel();
            release.send(()).unwrap();
            let execution = worker.join().unwrap().unwrap();
            assert_eq!(execution.outcome.termination,"cancelled");
            joined_at.set(Some(started.elapsed()));
            calls.borrow_mut().push(json!({"choice":choice,"remaining_nanos":remaining.as_nanos().to_string(),"outcome":"pass","prefix_verified":true,"detail":"pass"}));
            Ok::<_, std::convert::Infallible>(search::Evaluation {outcome:"pass",prefix_verified:true,detail:"pass".into()})
        },
        || {let cancelled=cancellation.cancelled();cancellation_samples.borrow_mut().push(cancelled);cancelled},
    ).unwrap();
    let returned_at = started.elapsed();
    assert_eq!(result["status"], "incomplete");
    assert_eq!(result["evaluations"][0]["outcome"], "timeout");
    assert!(result.get("first_failure").is_none());
    assert!(joined_at.get().unwrap() <= returned_at);
    let mut observed = result;
    observed["raw_input"] = json!({"search":search,"clock_nanos":clocks.into_inner(),"calls":calls.into_inner(),"cancelled":cancellation_samples.into_inner()});
    observed["cleanup"] = json!({"worker_termination":"cancelled","store_joined_nanos":joined_at.get().unwrap().as_nanos().to_string(),"returned_nanos":returned_at.as_nanos().to_string()});
    println!(
        "{}",
        json!({"schema":"gomad.native-inspection-predicate/v1","original_root":"TestInspectionSearchWallBoundIncludesEvaluatorCleanup","variant":"external-cancel-joined-store","observed":observed})
    );
}
