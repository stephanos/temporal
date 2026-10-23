use super::*;
use std::io;

#[derive(Clone, Copy)]
enum Failure {
    Zero,
    Error,
}

#[derive(Clone, Copy, Default)]
struct Counts {
    written: usize,
    calls: usize,
    failures: usize,
}

struct State {
    limit: Option<usize>,
    failure: Failure,
    counts: Counts,
}

fn observe<T>(
    limit: Option<usize>,
    failure: Failure,
    action: impl FnOnce(&mut ByteWriter<'_>) -> T,
) -> (T, Counts) {
    let mut state = State {
        limit,
        failure,
        counts: Counts::default(),
    };
    let result = action(&mut |file, bytes| {
        Writer {
            file,
            state: &mut state,
        }
        .write_all(bytes)
    });
    (result, state.counts)
}

struct Writer<'a> {
    file: &'a mut File,
    state: &'a mut State,
}

impl Write for Writer<'_> {
    fn write(&mut self, bytes: &[u8]) -> io::Result<usize> {
        self.state.counts.calls += 1;
        let remaining = self
            .state
            .limit
            .map_or(usize::MAX, |limit| limit - self.state.counts.written);
        if remaining == 0 {
            self.state.counts.failures += 1;
            return match self.state.failure {
                Failure::Zero => Ok(0),
                Failure::Error => Err(io::Error::other("injected retained writer error")),
            };
        }
        let size = bytes.len().min(3).min(remaining);
        let written = self.file.write(&bytes[..size])?;
        self.state.counts.written += written;
        Ok(written)
    }

    fn flush(&mut self) -> io::Result<()> {
        self.file.flush()
    }
}

fn publication() -> Publication {
    let mut publication = crate::fixture::publication_record(7).unwrap();
    publication.record.arguments = vec!["quoted=\"tail\"".into(), "Unicode=☃".into()];
    publication.record.environment = BTreeMap::from([("EMPTY".into(), String::new())]);
    publication.record.virtual_time_nanos = u64::MAX - 7;
    let binary: Vec<_> = (0..=255).cycle().take(257).collect();
    publication.record.streams.insert(
        "evidence/binary-writer-tail".into(),
        crate::evidence::record::Stream {
            bytes: binary.len() as u64,
            sha256: canonical::digest(&binary),
            records: 1,
        },
    );
    publication
        .payloads
        .insert("evidence/binary-writer-tail".into(), binary);
    publication
        .payloads
        .insert("record.json".into(), publication.record.bytes().unwrap());
    publication
}

fn copy(publication: &Publication) -> Publication {
    Publication {
        record: publication.record.clone(),
        payloads: publication.payloads.clone(),
    }
}

fn store(root: PathBuf) -> Store {
    Store {
        root,
        maximum_bytes: 64 << 20,
        target_pool: None,
        fault: None,
    }
}

fn authenticate(published: &super::super::Published, expected: &Publication) {
    let opened = Opened::open(&published.path).unwrap();
    opened.authenticate_all().unwrap();
    assert_eq!(opened.manifest().record, expected.record);
    for (name, bytes) in &expected.payloads {
        assert_eq!(opened.payload(name, bytes.len() as u64).unwrap(), *bytes);
    }
}

#[test]
fn publication_checks_partial_zero_and_error_writes_through_the_final_manifest_byte() {
    let root = std::env::temp_dir().join(format!(
        "gomad-retained-writer-{}",
        crate::campaign::origin::identity().unwrap()
    ));
    let publication = publication();
    let baseline = store(root.join("baseline"));
    let (published, counts) = observe(None, Failure::Error, |writer| {
        baseline.publish_with_writer(copy(&publication), writer)
    });
    let published = published.unwrap();
    authenticate(&published, &publication);
    let total = usize::try_from(published.stored_bytes).unwrap() + open::LAYOUT.len();
    let record_tail = open::LAYOUT.len()
        + publication
            .payloads
            .iter()
            .take_while(|(name, _)| name.as_str() != "record.json")
            .map(|(_, bytes)| bytes.len())
            .sum::<usize>()
        + publication.payloads["record.json"].len()
        - 1;
    let budgets = [0, 1, 19, 100, 1024, record_tail, total - 2, total - 1];
    assert_eq!(budgets.into_iter().collect::<BTreeSet<_>>().len(), 8);
    let mut variants = 0;
    let mut observations = Vec::new();
    for failure in [Failure::Zero, Failure::Error] {
        for budget in budgets {
            let store = store(root.join(format!("case-{variants}")));
            let (result, observed) = observe(Some(budget), failure, |writer| {
                store.publish_with_writer(copy(&publication), writer)
            });
            let error = result
                .err()
                .expect("actual write failure must refuse publication");
            assert!(
                error.contains(match failure {
                    Failure::Zero => "failed to write whole buffer",
                    Failure::Error => "injected retained writer error",
                }),
                "{error}"
            );
            assert_eq!(observed.written, budget);
            assert_eq!(
                observed.failures, 1,
                "publication must stop at the first I/O failure"
            );
            let committed_before_retry = std::fs::read_dir(&store.root)
                .unwrap()
                .filter(|entry| {
                    !entry
                        .as_ref()
                        .unwrap()
                        .file_name()
                        .as_encoded_bytes()
                        .starts_with(b".")
                })
                .count();
            assert_eq!(committed_before_retry, 0);
            let retried = store.publish(copy(&publication)).unwrap();
            authenticate(&retried, &publication);
            observations.push(serde_json::json!({
                "mode": match failure { Failure::Zero => "zero", Failure::Error => "error" },
                "budget": budget, "written": observed.written,
                "write_calls": observed.calls, "failures": observed.failures,
                "error": error, "committed_before_retry": committed_before_retry,
                "retry_record_sha256": retried.record_sha256,
                "retry_record_bytes": std::fs::read(retried.path.join("record.json")).unwrap(),
                "retry_manifest_bytes": std::fs::read(retried.path.join("manifest.json")).unwrap(),
            }));
            variants += 1;
        }
    }
    assert_eq!(variants, 16);
    assert_eq!(counts.written, total);
    assert!(counts.calls > 32);
    assert_eq!(counts.failures, 0);
    println!(
        "{}",
        serde_json::json!({
            "schema": "gomad.native-retained-writer-observation/v1",
            "encoding": "gomad.artifact-manifest/v2; raw record tail replaces retired Go base64-close; manifest tails are additional native boundaries",
            "record_sha256": published.record_sha256,
            "payloads": publication.payloads,
            "manifest_bytes": std::fs::read(published.path.join("manifest.json")).unwrap(),
            "layout_bytes": open::LAYOUT,
            "envelope_bytes": lineage::ENVELOPE,
            "total_written": counts.written, "record_tail": record_tail, "write_calls": counts.calls,
            "cases": observations,
        })
    );
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn retained_native_record_and_raw_payload_matrix_preserves_all_declared_fields() {
    let root = std::env::temp_dir().join(format!(
        "gomad-retained-fields-{}",
        crate::campaign::origin::identity().unwrap()
    ));
    let sizes = [
        0, 1, 2, 3, 1022, 1023, 1024, 1025, 2047, 2048, 2049, 8191, 8192, 8193,
    ];
    let mut cases = Vec::new();
    for size in sizes {
        let mut input = publication();
        input.record.arguments = vec!["<&\u{2028}".into(), "\u{2029}&".into(), String::new()];
        input.record.environment = BTreeMap::from([
            ("<&\u{2028}".into(), "\u{2029}".into()),
            ("EMPTY".into(), String::new()),
        ]);
        input.record.seed = u64::MAX;
        input.record.virtual_time_nanos = u64::MAX;
        input.record.outcome.reason = "literal <&\u{2028}\u{2029}".into();
        let binary = vec![0xff; size];
        let name = "evidence/binary-writer-tail";
        let stream = input.record.streams.get_mut(name).unwrap();
        stream.bytes = size as u64;
        stream.sha256 = canonical::digest(&binary);
        input.payloads.insert(name.into(), binary.clone());
        input
            .payloads
            .insert("record.json".into(), input.record.bytes().unwrap());
        let expected: serde_json::Value =
            serde_json::from_slice(&input.payloads["record.json"]).unwrap();
        let fields: Vec<_> = expected
            .as_object()
            .unwrap()
            .keys()
            .map(String::as_str)
            .collect();
        assert_eq!(
            fields,
            [
                "arguments",
                "environment",
                "execution_identity",
                "identity",
                "limits",
                "outcome",
                "profile",
                "schema",
                "seed",
                "streams",
                "virtual_time_nanos"
            ]
        );
        let published = store(root.join(size.to_string()))
            .publish(copy(&input))
            .unwrap();
        authenticate(&published, &input);
        let opened = Opened::open(&published.path).unwrap();
        let actual_record = opened.payload("record.json", 16 << 20).unwrap();
        assert_eq!(actual_record, input.payloads["record.json"]);
        assert_eq!(
            crate::evidence::record::Record::decode(&actual_record).unwrap(),
            input.record
        );
        let mut detached = opened.payload(name, 16 << 20).unwrap();
        assert_eq!(detached, binary);
        detached.fill(0);
        assert_eq!(opened.payload(name, 16 << 20).unwrap(), binary);
        cases.push(serde_json::json!({"size":size,"record_bytes":actual_record,
            "payload_bytes":opened.payload(name, 16 << 20).unwrap(),
            "manifest_bytes":std::fs::read(published.path.join("manifest.json")).unwrap()}));
    }
    println!(
        "{}",
        serde_json::json!({"schema":"gomad.native-retained-fields-observation/v1",
        "encoding":"gomad.execution-record/v2; separate raw binary payloads", "cases":cases})
    );
    std::fs::remove_dir_all(root).unwrap();
}
