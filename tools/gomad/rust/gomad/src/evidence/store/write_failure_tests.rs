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
    let budgets = [0, 1, 2, total / 3, total / 2, total - 2, total - 1];
    assert_eq!(budgets.into_iter().collect::<BTreeSet<_>>().len(), 7);
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
            assert!(std::fs::read_dir(&store.root).unwrap().all(|entry| {
                entry
                    .unwrap()
                    .file_name()
                    .to_str()
                    .unwrap()
                    .starts_with('.')
            }));
            let retried = store.publish(copy(&publication)).unwrap();
            authenticate(&retried, &publication);
            observations.push(serde_json::json!({
                "mode": match failure { Failure::Zero => "zero", Failure::Error => "error" },
                "budget": budget, "written": observed.written,
                "write_calls": observed.calls, "failures": observed.failures,
                "error": error, "committed_before_retry": 0,
                "retry_record_sha256": retried.record_sha256,
                "retry_record_bytes": std::fs::read(retried.path.join("record.json")).unwrap(),
                "retry_manifest_bytes": std::fs::read(retried.path.join("manifest.json")).unwrap(),
            }));
            variants += 1;
        }
    }
    assert_eq!(variants, 14);
    assert_eq!(counts.written, total);
    assert!(counts.calls > 32);
    assert_eq!(counts.failures, 0);
    println!(
        "{}",
        serde_json::json!({
            "schema": "gomad.native-retained-writer-observation/v1",
            "encoding": "gomad.artifact-manifest/v2; raw payload bytes; terminal manifest byte replaces retired Go base64-close boundary",
            "record_sha256": published.record_sha256,
            "payloads": publication.payloads,
            "manifest_bytes": std::fs::read(published.path.join("manifest.json")).unwrap(),
            "layout_bytes": open::LAYOUT,
            "envelope_bytes": lineage::ENVELOPE,
            "total_written": counts.written, "write_calls": counts.calls,
            "cases": observations,
        })
    );
    std::fs::remove_dir_all(root).unwrap();
}
