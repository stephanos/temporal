use super::*;
use crate::{
    campaign::ledger::{Event as CampaignEvent, Execution, Job, Journal as Campaign},
    evidence::{
        identity::Identity,
        record::{Limits, Outcome as ExecutionOutcome, Record, Stream},
        store::{Publication, Store},
    },
    install::manifest::{Admission, Tool},
    sweep::process::Completed,
};
use std::{
    collections::VecDeque,
    os::unix::fs::PermissionsExt as _,
    sync::{
        Arc,
        atomic::{AtomicU64, Ordering},
    },
};

static SEQUENCE: AtomicU64 = AtomicU64::new(0);

struct Fixture {
    root: PathBuf,
    options: Options,
    clock: Arc<AtomicU64>,
    start: Instant,
}
impl Fixture {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-sweep-lifecycle-{}-{}",
            std::process::id(),
            SEQUENCE.fetch_add(1, Ordering::Relaxed)
        ));
        directory(&root).unwrap();
        let checkout = root.join("checkout");
        for path in [
            checkout.join("tests/testcore"),
            checkout.join("tests/example"),
            checkout.join("schema/sqlite"),
            root.join("runtime"),
        ] {
            directory(&path).unwrap();
        }
        fs::write(checkout.join("go.mod"), "module example\n").unwrap();
        fs::write(checkout.join("tests/example/example_test.go"), "package example\nimport \"testing\"\nfunc TestA(t *testing.T) {}\nfunc TestB(t *testing.T) {}\n").unwrap();
        fs::write(
            checkout.join("schema/sqlite/schema.sql"),
            "CREATE TABLE fixture (id INTEGER);\n",
        )
        .unwrap();
        for name in ["gomad", "helper", "compiler"] {
            let path = root.join(name);
            fs::write(&path, b"fixture executable identity\n").unwrap();
            fs::set_permissions(path, fs::Permissions::from_mode(0o700)).unwrap();
        }
        let options = Options {
            checkout,
            gomad: root.join("gomad"),
            helper: root.join("helper"),
            compiler: root.join("compiler"),
            runtime_root: root.join("runtime"),
            config_source: "temporal-install".into(),
            packages: vec!["./tests/example".into()],
            max_tests: 0,
            seed: 7,
            per_test_timeout: Duration::from_secs(10),
            prepare_timeout: Duration::from_secs(20),
            overall_budget: Duration::from_secs(300),
            retain_bytes: 64 << 20,
            out: root.join("out"),
            work: root.join("work"),
            artifacts: root.join("retained"),
            environ: BTreeMap::new(),
        };
        Self {
            root,
            options,
            clock: Arc::new(AtomicU64::new(0)),
            start: Instant::now(),
        }
    }
    fn context(&self, seconds: u64) -> Context {
        let mut context = Context::new(self.start + Duration::from_secs(seconds));
        let clock = self.clock.clone();
        let start = self.start;
        context.now = Arc::new(move || start + Duration::from_secs(clock.load(Ordering::Acquire)));
        context
    }
    fn rows(&self) -> Vec<TestEntry> {
        let sidecar: record::Sidecar =
            serde_json::from_slice(&fs::read(self.options.out.join("sweep.json")).unwrap())
                .unwrap();
        sidecar.rows
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.root).unwrap();
    }
}

#[derive(Clone)]
enum Response {
    Pass,
    Failure,
    Error(&'static str),
    Invalid,
    Cancel,
    SoftAssertion { report_only: bool },
    InconclusiveSoftAssertion,
    SoftAssertionWithoutProgress,
}
#[derive(Clone)]
struct Call {
    path: PathBuf,
    args: Vec<String>,
    dir: PathBuf,
    env: BTreeMap<String, String>,
}
struct Fake {
    options: Options,
    calls: Vec<Call>,
    responses: VecDeque<Response>,
    clock: Arc<AtomicU64>,
    warm_seconds: u64,
    test_seconds: u64,
    source_reads: usize,
    change_source_after: Option<usize>,
    dirty: bool,
    exclude_guest: bool,
}
impl Fake {
    fn new(fixture: &Fixture, responses: impl IntoIterator<Item = Response>) -> Self {
        Self {
            options: fixture.options.clone(),
            calls: Vec::new(),
            responses: responses.into_iter().collect(),
            clock: fixture.clock.clone(),
            warm_seconds: 0,
            test_seconds: 0,
            source_reads: 0,
            change_source_after: None,
            dirty: false,
            exclude_guest: false,
        }
    }
    fn explores(&self) -> Vec<&Call> {
        self.calls
            .iter()
            .filter(|call| call.args.iter().any(|arg| arg == "explore"))
            .collect()
    }
    fn tests(&self) -> Vec<&Call> {
        self.explores()
            .into_iter()
            .filter(|call| !call.args.iter().any(|arg| arg == "-test.run=^$"))
            .collect()
    }
    fn value<'a>(args: &'a [String], flag: &str) -> &'a str {
        &args[args.iter().position(|arg| arg == flag).unwrap() + 1]
    }
    fn install(&self, command: &Command) {
        let config = if command.args.iter().any(|arg| arg == "--config") {
            config::build(
                "gomad.wasi-cooperative/v1",
                Some(Path::new(Self::value(&command.args, "--config"))),
                None,
            )
        } else {
            let package = Self::value(&command.args, "--temporal-package");
            config::build(
                "gomad.wasi-cooperative/v1",
                None,
                Some((&self.options.checkout, package)),
            )
        }
        .unwrap();
        let executable = file_sha256(&self.options.gomad)
            .unwrap()
            .trim_start_matches("sha256:")
            .to_owned();
        let installed = Installation {
            schema: crate::install::manifest::SCHEMA.into(),
            admission: Some(Admission {
                compiler_inputs: BTreeMap::new(),
                inputs_sha256: "1".repeat(64),
                configuration_sha256: canonical::digest(
                    &canonical::encode(&config.original).unwrap(),
                ),
                guest_configuration: config.original,
                host_executable_sha256: executable.clone(),
                explicit_helper: Some(Tool {
                    path: self.options.helper.clone(),
                    sha256: executable,
                }),
                environment: config.environment,
                arguments: config.arguments,
                compile_cache_root: Some(Self::value(&command.args, "--compile-cache").into()),
            }),
            compiler_path: self.options.compiler.clone(),
            compiler_sha256: file_sha256(&self.options.compiler)
                .unwrap()
                .trim_start_matches("sha256:")
                .into(),
            go_version: "go1.27.1".into(),
            runtime_root: self.options.runtime_root.clone(),
            overlay_module_root: None,
            cache_root: Self::value(&command.args, "--cache").into(),
            profile: "gomad.wasi-cooperative/v1".into(),
            fuel: 1 << 50,
            memory_bytes: 4 << 30,
            helper_sha256: gomad_wasmhost::linked::IMPLEMENTATION_SHA256.into(),
            host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
            runtime_implementation_sha256: Some(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
            model: config.model,
            namespace: config.namespace,
            storage: config.storage,
            network: config.network,
            build_tags: config.build_tags,
        };
        fs::write(
            Self::value(&command.args, "--output"),
            serde_json::to_vec(&installed).unwrap(),
        )
        .unwrap();
    }
    fn native(&self, command: &Command, failed: bool) -> Completed {
        self.native_with_soft_assertions(command, failed, None)
    }
    #[expect(
        clippy::too_many_lines,
        reason = "The fixture publishes one complete authenticated native campaign transaction"
    )]
    fn native_with_soft_assertions(
        &self,
        command: &Command,
        failed: bool,
        report_only: Option<bool>,
    ) -> Completed {
        let campaign_path = PathBuf::from(Self::value(&command.args, "--artifacts"));
        let arguments =
            command.args[command.args.iter().position(|arg| arg == "--").unwrap() + 1..].to_vec();
        let selected = arguments
            .iter()
            .find_map(|arg| arg.strip_prefix("-test.run="))
            .unwrap();
        let name = selected.replace(['^', '$'], "");
        let stdout = if report_only.is_some() {
            format!("failed assertion: fixture assertion\n--- PASS: {name} (0s)\nPASS\n")
        } else if failed {
            format!("panic: deterministic fixture failure\n--- FAIL: {name} (0s)\n")
        } else {
            format!("--- PASS: {name} (0s)\nPASS\n")
        }
        .into_bytes();
        let limits = Limits {
            fuel: 100,
            memory_bytes: 65536,
            wall_millis: 1000,
            output_bytes: 4096,
            evidence_bytes: 1 << 20,
        };
        let configuration = json!({"profile":"gomad.wasi-cooperative/v1","arguments":arguments,"environment":{},"limits":limits});
        let module = b"\0asm\x01\0\0\0".to_vec();
        let identity = Identity {
            target: canonical::digest(&module),
            toolchain: "1".repeat(64),
            platform: crate::replay::preflight::platform(),
            implementation: crate::replay::preflight::implementation().unwrap(),
            configuration: canonical::digest(&canonical::encode(&configuration).unwrap()),
        };
        let soft_assertions = report_only.and_then(|report_only| {
            crate::campaign::soft_assertions::observe(&stdout, &[], true, report_only)
        });
        let mut record = Record {
            schema: "gomad.execution-record/v2".into(),
            execution_identity: identity.digest().unwrap(),
            identity,
            profile: "gomad.wasi-cooperative/v1".into(),
            seed: self.options.seed,
            arguments,
            environment: BTreeMap::new(),
            limits,
            virtual_time_nanos: 0,
            outcome: ExecutionOutcome {
                class: if failed { "target_failure" } else { "success" }.into(),
                termination: "exit".into(),
                exit_code: Some(if report_only.is_some() {
                    0
                } else {
                    u32::from(failed)
                }),
                reason: if report_only == Some(false) {
                    "soft_assertion"
                } else if failed {
                    "panic_or_runtime_fatal"
                } else {
                    ""
                }
                .into(),
            },
            streams: BTreeMap::from([(
                "output/stdout".into(),
                Stream {
                    bytes: stdout.len() as u64,
                    sha256: canonical::digest(&stdout),
                    records: 1,
                },
            )]),
        };
        let mut payloads = BTreeMap::from([
            ("target/module.wasm".into(), module),
            ("output/stdout".into(), stdout),
        ]);
        if let Some(report) = &soft_assertions {
            let bytes = canonical::encode(report).unwrap();
            record.streams.insert(
                "evidence/soft-assertions.json".into(),
                Stream {
                    bytes: bytes.len() as u64,
                    sha256: canonical::digest(&bytes),
                    records: 1,
                },
            );
            payloads.insert("evidence/soft-assertions.json".into(), bytes);
        }
        let plan = json!({"soft_assertion_report_only":report_only.unwrap_or(false),"schema":"gomad.seed-campaign-plan/v2","selection":self.options.seed.to_string(),"selection_count":1,"parallel":1,"on_failure":"all","failure_budget":0,"journal_bytes":1 << 20,"module_sha256":record.identity.target,"toolchain":record.identity.toolchain,"configuration":configuration});
        let mut journal = Campaign::create(&campaign_path, &plan, 1 << 20).unwrap();
        let publication = Store {
            root: campaign_path.join("evidence"),
            maximum_bytes: 0,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        }
        .publish(Publication {
            record: record.clone(),
            payloads,
        })
        .unwrap();
        let execution = Execution {
            ordinal: 0,
            seed: self.options.seed,
            record: record.clone(),
            artifact: Some(
                publication
                    .path
                    .strip_prefix(&campaign_path)
                    .unwrap()
                    .display()
                    .to_string(),
            ),
            stored_bytes: publication.stored_bytes,
            semantic_probes: vec![],
            choice_features: vec![],
            divergence: None,
            soft_assertions: soft_assertions
                .as_ref()
                .map(|report| serde_json::to_value(report).unwrap()),
            scenario_steps: None,
        };
        journal
            .append(CampaignEvent::Admit {
                jobs: vec![Job {
                    ordinal: 0,
                    seed: self.options.seed,
                }],
            })
            .unwrap();
        journal
            .append(CampaignEvent::Receipt {
                execution: execution.clone(),
            })
            .unwrap();
        journal
            .append(CampaignEvent::Outcome {
                executions: vec![execution],
                admitted: vec![],
                stop_reason: None,
            })
            .unwrap();
        journal
            .append(CampaignEvent::Complete {
                stop_reason: "seeds_exhausted".into(),
            })
            .unwrap();
        drop(journal);
        let result = json!({"type":"result","classification":if failed {"target_failure"} else {"success"},"campaign_path":campaign_path,"selection_count":1,"attempted":1,"succeeded":u64::from(!failed),"failures":u64::from(failed),"watchdogs":0,"cancelled":0,"distinct_failures":u64::from(failed),"stop_reason":"seeds_exhausted","success_artifacts":if failed {vec![]} else {vec![json!({"seed":self.options.seed,"path":publication.path,"record_hash":record.hash().unwrap()})]},"retained_evidence":1,"retained_bytes":publication.stored_bytes});
        let mut stdout = Vec::new();
        if let Some(report) = soft_assertions {
            stdout.extend(serde_json::to_vec(&json!({"schema":"gomad.explore-event/v3","type":"soft_assertion","campaign_path":campaign_path,"seed":self.options.seed,"soft_assertions":report})).unwrap());
            stdout.push(b'\n');
        }
        stdout.extend(serde_json::to_vec(&result).unwrap());
        Completed {
            stdout,
            exit: i32::from(failed),
            ..Completed::default()
        }
    }
}
impl Executor for Fake {
    fn exec(&mut self, context: &Context, command: &Command) -> Completed {
        self.calls.push(Call {
            path: command.path.clone(),
            args: command.args.clone(),
            dir: command.dir.clone(),
            env: command.env.clone(),
        });
        if command.path == Path::new("git") {
            if command.args.iter().any(|arg| arg == "status") {
                return Completed {
                    stdout: if self.dirty {
                        b" M tracked.go\n".to_vec()
                    } else {
                        vec![]
                    },
                    ..Completed::default()
                };
            }
            if command.args.iter().any(|arg| arg == "rev-parse") {
                self.source_reads += 1;
                let changed = self
                    .change_source_after
                    .is_some_and(|after| self.source_reads > after);
                return Completed {
                    stdout: if changed {
                        b"changed-head\n".to_vec()
                    } else {
                        b"frozen-head\n".to_vec()
                    },
                    ..Completed::default()
                };
            }
            return Completed {
                stdout: b"100644 frozen source.go\n".to_vec(),
                ..Completed::default()
            };
        }
        if command.path == self.options.compiler {
            if command.args[0] == "list" {
                let files = if self.exclude_guest
                    && command
                        .env
                        .get("GOOS")
                        .is_some_and(|value| value == "wasip1")
                {
                    vec![]
                } else {
                    vec!["example_test.go"]
                };
                return Completed { stdout:serde_json::to_vec(&json!({"ImportPath":"example/tests/example","Dir":self.options.checkout.join("tests/example"),"TestGoFiles":files})).unwrap(),..Completed::default() };
            }
            assert_eq!(command.args[0], "test");
            return Completed {
                stdout: b"TestA\nTestB\n".to_vec(),
                ..Completed::default()
            };
        }
        if command.args[0] == "install" {
            self.install(command);
            return Completed::default();
        }
        assert!(command.args.iter().any(|arg| arg == "explore"));
        let install = PathBuf::from(Self::value(&command.args, "--installation"));
        let dirs = install.parent().unwrap().parent().unwrap();
        fs::write(dirs.join("preparation/module"), b"prepared").unwrap();
        fs::write(dirs.join("engine/compiled"), b"compiled").unwrap();
        if command.args.iter().any(|arg| arg == "-test.run=^$") {
            self.clock.fetch_add(self.warm_seconds, Ordering::AcqRel);
            return Completed::default();
        }
        self.clock.fetch_add(self.test_seconds, Ordering::AcqRel);
        match self.responses.pop_front().expect("unexpected test execution") {
            Response::Pass => self.native(command,false), Response::Failure => self.native(command,true),
            Response::Error(classification) => Completed { stdout:serde_json::to_vec(&json!({"schema":"gomad.explore-event/v3","type":"error","classification":classification,"message":"fixture error"})).unwrap(),exit:3,..Completed::default() },
            Response::SoftAssertion { report_only } => self.native_with_soft_assertions(command, !report_only, Some(report_only)),
            Response::SoftAssertionWithoutProgress => {
                let mut output = self.native_with_soft_assertions(command, true, Some(false));
                let first_line = output.stdout.iter().position(|byte| *byte == b'\n').unwrap();
                output.stdout.drain(..=first_line);
                output
            }
            Response::InconclusiveSoftAssertion => Completed {
                stdout: serde_json::to_vec(&json!({"schema":"gomad.explore-event/v3","type":"error","classification":"capacity","message":"soft assertion observation inconclusive","soft_assertions":{"messages":[],"report_only":false,"inconclusive":true}})).unwrap(), exit: 3, ..Completed::default()
            },
            Response::Cancel => {
                context.cancelled.store(true, Ordering::Release);
                Completed::default()
            }
            Response::Invalid => Completed { stdout:b"invalid output".to_vec(),exit:3,..Completed::default() },
        }
    }
}

#[test]
fn native_success_failure_retention_cache_and_resume_use_the_real_lifecycle() {
    let fixture = Fixture::new();
    let context = fixture.context(300);
    let mut fake = Fake::new(&fixture, [Response::Pass, Response::Failure]);
    assert_eq!(
        run_with(
            &mut fake,
            &context,
            &fixture.options,
            "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        )
        .unwrap(),
        Vec::<String>::new()
    );
    let rows = fixture.rows();
    assert_eq!(rows.len(), 2);
    assert_eq!(
        (rows[0].class, rows[1].class),
        (Class::Pass, Class::GuestFailure)
    );
    assert!(!rows[0].artifact_retained);
    assert!(rows[1].artifact_retained);
    assert!(Path::new(&rows[1].artifact_path).is_dir());
    assert!(
        rows.iter()
            .all(|row| row.preparation_bytes > 0 && row.engine_cache_hit)
    );
    assert_eq!(fake.explores().len(), 3);
    let installs: Vec<_> = fake
        .explores()
        .iter()
        .map(|call| Fake::value(&call.args, "--installation").to_owned())
        .collect();
    assert!(installs.windows(2).all(|pair| pair[0] == pair[1]));
    for call in fake.tests() {
        assert_eq!(call.path, fixture.options.gomad);
        assert_eq!(
            Fake::value(&call.args, "--working-dir"),
            fixture.options.checkout.display().to_string()
        );
        assert!(call.args.contains(&"./tests/example".into()));
        assert!(call.args.contains(&"--parallel=1".into()));
        assert_eq!(
            call.args
                .iter()
                .filter(|arg| arg.starts_with("-test.run="))
                .count(),
            1
        );
        assert!(call.dir.starts_with(&fixture.options.work));
    }
    let before = fs::read(fixture.options.out.join("sweep.jsonl")).unwrap();
    let calls = fake.tests().len();
    run_with(
        &mut fake,
        &context,
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    assert_eq!(fake.tests().len(), calls);
    let after = fs::read(fixture.options.out.join("sweep.jsonl")).unwrap();
    assert!(after.starts_with(&before));
    let appended: Vec<record::Row> = after[before.len()..]
        .split(|byte| *byte == b'\n')
        .filter(|line| !line.is_empty())
        .map(|line| serde_json::from_slice(line).unwrap())
        .collect();
    assert_eq!(appended.len(), 1);
    assert!(matches!(appended[0].entry, record::RowEntry::Header { .. }));
    assert!(Path::new(&rows[1].artifact_path).is_dir());
    assert!(
        fs::read_dir(fixture.options.work.join("packages"))
            .unwrap()
            .next()
            .is_none()
    );
}

#[test]
fn infrastructure_retries_once_and_guest_failure_does_not_retry() {
    let mut fixture = Fixture::new();
    fixture.options.max_tests = 1;
    let mut fake = Fake::new(&fixture, [Response::Invalid, Response::Pass]);
    run_with(
        &mut fake,
        &fixture.context(300),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    let row = fixture.rows().remove(0);
    assert_eq!((row.class, row.attempts), (Class::Pass, 2));
    assert_ne!(row.first_attempt_reason, "");
    assert_eq!(fake.tests().len(), 2);
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, [Response::Failure, Response::Failure]);
    run_with(
        &mut fake,
        &fixture.context(300),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    let rows = fixture.rows();
    assert!(rows.iter().all(|row| row.attempts == 1));
    assert!(rows[0].artifact_retained);
    assert!(!rows[1].artifact_retained);
    assert!(rows[1].retention_note.contains("already retained"));
}

#[test]
fn virtual_budget_stops_after_warmup_and_resumes_not_run_rows() {
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, []);
    fake.warm_seconds = 80;
    run_with(
        &mut fake,
        &fixture.context(100),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    assert_eq!(fake.tests().len(), 0);
    assert!(
        fixture
            .rows()
            .iter()
            .all(|row| row.class == Class::NotRun && row.attempts == 0)
    );
    fixture.clock.store(0, Ordering::Release);
    fake.warm_seconds = 0;
    fake.responses.extend([Response::Pass, Response::Pass]);
    run_with(
        &mut fake,
        &fixture.context(100),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    assert_eq!(fake.tests().len(), 2);
    assert!(fixture.rows().iter().all(|row| row.class == Class::Pass));
}

#[test]
fn virtual_budget_during_test_cleans_attempt_and_forbids_retry() {
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, [Response::Invalid]);
    fake.test_seconds = 100;
    run_with(
        &mut fake,
        &fixture.context(100),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    assert_eq!(fake.tests().len(), 1);
    assert!(fixture.rows().iter().all(|row| row.class == Class::NotRun));
    assert!(
        fs::read_dir(fixture.options.work.join("packages"))
            .unwrap()
            .next()
            .is_none()
    );
}

#[test]
fn changed_source_and_candidate_refuse_before_additional_test_execution() {
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, []);
    fake.change_source_after = Some(1);
    assert!(
        run_with(
            &mut fake,
            &fixture.context(300),
            &fixture.options,
            "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        )
        .unwrap_err()
        .contains("source changed")
    );
    assert!(fake.tests().is_empty());
    assert!(!fixture.options.out.join("sweep.jsonl").exists());
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, [Response::Pass, Response::Pass]);
    run_with(
        &mut fake,
        &fixture.context(300),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    let before = fs::read(fixture.options.out.join("sweep.jsonl")).unwrap();
    let mut changed = fixture.options.clone();
    changed.seed += 1;
    fake.options = changed.clone();
    assert!(
        run_with(
            &mut fake,
            &fixture.context(300),
            &changed,
            "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        )
        .is_err()
    );
    assert_eq!(fake.tests().len(), 2);
    assert_eq!(
        fs::read(fixture.options.out.join("sweep.jsonl")).unwrap(),
        before
    );
}

#[test]
fn guest_excluded_and_dirty_sources_do_not_execute_tests() {
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, []);
    fake.exclude_guest = true;
    run_with(
        &mut fake,
        &fixture.context(300),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    assert!(fake.explores().is_empty());
    assert!(
        fixture
            .rows()
            .iter()
            .all(|row| row.class == Class::BuildFailure && row.reason.contains("guest-file-set"))
    );
    assert!(
        fake.calls
            .iter()
            .any(|call| call.env.get("GOOS").is_some_and(|value| value == "wasip1"))
    );
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, []);
    fake.dirty = true;
    assert!(
        run_with(
            &mut fake,
            &fixture.context(300),
            &fixture.options,
            "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        )
        .unwrap_err()
        .contains("checkout has tracked or untracked changes")
    );
    assert!(fake.explores().is_empty());
}

#[test]
fn zero_retention_keeps_classification_and_error_event_never_becomes_pass() {
    let mut fixture = Fixture::new();
    fixture.options.retain_bytes = 0;
    let mut fake = Fake::new(
        &fixture,
        [Response::Failure, Response::Error("unsupported_target")],
    );
    run_with(
        &mut fake,
        &fixture.context(300),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    let rows = fixture.rows();
    assert_eq!(
        (rows[0].class, rows[1].class),
        (Class::GuestFailure, Class::RefusedImport)
    );
    assert!(!rows[0].artifact_retained);
    assert!(rows[0].retention_note.contains("budget"));
    assert_eq!(rows[1].attempts, 1);
}

#[test]
fn authentic_native_artifact_tampering_is_rejected_before_classification() {
    let fixture = Fixture::new();
    let fake = Fake::new(&fixture, []);
    let root = fixture.root.join("attempt");
    directory(&root).unwrap();
    let pkg = Package {
        import_path: "example/tests/example".into(),
        pattern: "./tests/example".into(),
        module_directory: fixture.options.checkout.display().to_string(),
        target: "./tests/example".into(),
        tests: vec!["TestA".into()],
        top_level_tests: vec!["TestA".into()],
        units: BTreeMap::from([("TestA".into(), vec!["TestA".into()])]),
        guest_excluded: BTreeMap::new(),
        methods: vec![],
        registrations: vec![],
        list_count: 1,
        parsed_count: 1,
    };
    let dirs = Dirs::new(&fixture.options.work, &pkg.pattern);
    let command = Command {
        path: fixture.options.gomad.clone(),
        args: explore_args(
            &fixture.options,
            &pkg,
            &dirs,
            &root.join("artifacts"),
            "^TestA$",
            fixture.options.per_test_timeout,
        ),
        dir: root.clone(),
        env: BTreeMap::new(),
        timeout: Duration::from_secs(30),
    };
    let completed = fake.native(&command, false);
    let native: classify::NativeResult = serde_json::from_slice(&completed.stdout).unwrap();
    let (observation, events) =
        authenticated_observation("TestA", fixture.options.seed, &root, completed).unwrap();
    assert_eq!(
        classify::classify_native(&observation, &events).class,
        Class::Pass
    );
    let stream = Path::new(&native.success_artifacts[0].path).join("output/stdout");
    fs::set_permissions(&stream, fs::Permissions::from_mode(0o600)).unwrap();
    fs::write(&stream, b"--- PASS: TestA (tampered)\n").unwrap();
    let result = authenticated_observation(
        "TestA",
        fixture.options.seed,
        &root,
        Completed {
            stdout: observation.stdout,
            ..Completed::default()
        },
    );
    assert!(result.is_err());
}

#[test]
fn virtual_budget_before_start_and_before_retry_never_launches_extra_work() {
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, []);
    run_with(
        &mut fake,
        &fixture.context(20),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    assert!(fake.explores().is_empty());
    assert!(
        fixture
            .rows()
            .iter()
            .all(|row| row.class == Class::NotRun && row.attempts == 0)
    );
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, [Response::Invalid]);
    fake.test_seconds = 75;
    run_with(
        &mut fake,
        &fixture.context(100),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    assert_eq!(fake.tests().len(), 1);
    let rows = fixture.rows();
    assert_eq!(
        (rows[0].class, rows[0].attempts),
        (Class::Infrastructure, 1)
    );
    assert_eq!((rows[1].class, rows[1].attempts), (Class::NotRun, 0));
}

#[test]
fn cancellation_during_execution_leaves_no_verdict_and_resume_runs_only_remaining_unit() {
    let fixture = Fixture::new();
    let mut fake = Fake::new(&fixture, [Response::Pass, Response::Cancel]);
    let error = run_with(
        &mut fake,
        &fixture.context(300),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap_err();
    assert!(error.contains("cancelled"));
    assert_eq!(fake.tests().len(), 2);
    let before = fs::read(fixture.options.out.join("sweep.jsonl")).unwrap();
    let rows: Vec<record::Row> = before
        .split(|byte| *byte == b'\n')
        .filter(|line| !line.is_empty())
        .map(|line| serde_json::from_slice(line).unwrap())
        .collect();
    let verdicts: Vec<_> = rows
        .iter()
        .filter_map(|row| match &row.entry {
            record::RowEntry::Test { test } => Some(test),
            _ => None,
        })
        .collect();
    assert_eq!(verdicts.len(), 1);
    assert_eq!(verdicts[0].name, "TestA");
    assert_eq!(verdicts[0].class, Class::Pass);
    let interrupted_root = fake.tests()[1].dir.clone();
    assert!(!interrupted_root.exists());
    fake.responses.push_back(Response::Pass);
    run_with(
        &mut fake,
        &fixture.context(300),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    let after = fs::read(fixture.options.out.join("sweep.jsonl")).unwrap();
    assert!(after.starts_with(&before));
    assert_eq!(fake.tests().len(), 3);
    assert!(fake.tests()[2].args.contains(&"-test.run=^TestB$".into()));
    let rows = fixture.rows();
    assert_eq!(rows.len(), 2);
    assert!(
        rows.iter()
            .all(|row| row.class == Class::Pass && row.attempts == 1)
    );
}

#[test]
fn nested_module_execution_uses_its_module_directory_and_local_target() {
    let fixture = Fixture::new();
    let module = fixture.options.checkout.join("tests/nested");
    directory(&module).unwrap();
    let package = Package {
        import_path: "nested".into(),
        pattern: "./tests/nested".into(),
        module_directory: module.display().to_string(),
        target: "./.".into(),
        tests: vec!["TestA".into()],
        top_level_tests: vec!["TestA".into()],
        units: BTreeMap::from([("TestA".into(), vec!["TestA".into()])]),
        guest_excluded: BTreeMap::new(),
        methods: vec![],
        registrations: vec![],
        list_count: 1,
        parsed_count: 1,
    };
    let dirs = Dirs::new(&fixture.options.work, &package.pattern);
    directory(&fixture.options.work).unwrap();
    let context = fixture.context(300);
    let mut fake = Fake::new(&fixture, [Response::Pass]);
    install_package(&mut fake, &context, &fixture.options, &package, &dirs).unwrap();
    let mut retention = Retention::new(0, &BTreeMap::new());
    let row = run_test(
        &mut fake,
        &context,
        &fixture.options,
        &package,
        "TestA",
        &dirs,
        &mut retention,
    )
    .unwrap();
    assert_eq!((row.class, row.attempts), (Class::Pass, 1));
    assert_eq!(fake.tests().len(), 1);
    let call = fake.tests()[0];
    assert_eq!(
        Fake::value(&call.args, "--working-dir"),
        module.display().to_string()
    );
    let target = call.args.iter().position(|arg| arg == "go-test").unwrap();
    assert_eq!(call.args[target + 1], "./.");
    assert!(call.args.contains(&"-test.run=^TestA$".into()));
}

#[test]
fn warmup_admission_observes_the_exact_preparation_threshold() {
    for (budget, expected_warmups) in [(69, 0), (70, 1)] {
        let fixture = Fixture::new();
        let mut fake = Fake::new(&fixture, [Response::Pass, Response::Pass]);
        run_with(
            &mut fake,
            &fixture.context(budget),
            &fixture.options,
            "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        )
        .unwrap();
        let warmups = fake
            .explores()
            .iter()
            .filter(|call| call.args.contains(&"-test.run=^$".into()))
            .count();
        assert_eq!(warmups, expected_warmups, "budget {budget}");
        assert_eq!(fake.tests().len(), 2);
        assert!(fixture.rows().iter().all(|row| row.class == Class::Pass));
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum DeadlinePhase {
    Enumeration,
    Installation,
    Warmup,
    Test,
}
struct ExpireDuring {
    fake: Fake,
    phase: DeadlinePhase,
    deadline: Instant,
    expired: bool,
}
impl Executor for ExpireDuring {
    fn exec(&mut self, context: &Context, command: &Command) -> Completed {
        assert_eq!(
            context.deadline, self.deadline,
            "deadline changed between operations"
        );
        let phase = if command.path == self.fake.options.compiler && command.args[0] == "list" {
            Some(DeadlinePhase::Enumeration)
        } else if command.args[0] == "install" {
            Some(DeadlinePhase::Installation)
        } else if command.args.iter().any(|arg| arg == "explore") {
            Some(if command.args.iter().any(|arg| arg == "-test.run=^$") {
                DeadlinePhase::Warmup
            } else {
                DeadlinePhase::Test
            })
        } else {
            None
        };
        let completed = self.fake.exec(context, command);
        if phase == Some(self.phase) && !self.expired {
            self.fake.clock.store(300, Ordering::Release);
            self.expired = true;
        }
        assert_eq!(context.deadline, self.deadline);
        completed
    }
}

#[test]
fn one_absolute_deadline_covers_enumeration_installation_warmup_and_test() {
    for phase in [
        DeadlinePhase::Enumeration,
        DeadlinePhase::Installation,
        DeadlinePhase::Warmup,
        DeadlinePhase::Test,
    ] {
        let fixture = Fixture::new();
        let context = fixture.context(300);
        let mut executor = ExpireDuring {
            fake: Fake::new(&fixture, [Response::Pass]),
            phase,
            deadline: context.deadline,
            expired: false,
        };
        let result = run_with(
            &mut executor,
            &context,
            &fixture.options,
            "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        );
        assert!(executor.expired, "{phase:?} was not reached");
        assert_eq!(context.deadline, executor.deadline);
        if matches!(
            phase,
            DeadlinePhase::Enumeration | DeadlinePhase::Installation
        ) {
            assert!(result.unwrap_err().contains("budget"));
            assert!(!fixture.options.out.join("sweep.jsonl").exists());
            assert!(!fixture.options.out.join("sweep.json").exists());
            assert!(executor.fake.explores().is_empty());
        } else {
            result.unwrap();
            assert!(fixture.options.out.join("sweep.jsonl").is_file());
            let rows = fixture.rows();
            assert_eq!(rows.len(), 2);
            assert!(
                rows.iter()
                    .all(|row| row.class == Class::NotRun && row.attempts == 0)
            );
            assert_eq!(
                executor.fake.tests().len(),
                usize::from(phase == DeadlinePhase::Test)
            );
            assert!(
                fs::read_dir(fixture.options.work.join("packages"))
                    .unwrap()
                    .next()
                    .is_none()
            );
        }
    }
}

#[test]
fn two_package_candidates_share_identity_and_cap_each_sorted_inventory() {
    struct Packages(Fake);
    impl Executor for Packages {
        fn exec(&mut self, context: &Context, command: &Command) -> Completed {
            let mut completed = self.0.exec(context, command);
            if command.path == self.0.options.compiler && command.args[0] == "list" {
                let another = json!({"ImportPath":"example/tests/another","Dir":self.0.options.checkout.join("tests/another"),"TestGoFiles":["another_test.go"]});
                completed
                    .stdout
                    .extend_from_slice(&serde_json::to_vec(&another).unwrap());
            }
            completed
        }
    }
    let mut fixture = Fixture::new();
    fixture.options.packages = vec!["./tests/...".into()];
    fixture.options.max_tests = 1;
    directory(&fixture.options.checkout.join("tests/another")).unwrap();
    fs::write(fixture.options.checkout.join("tests/another/another_test.go"),"package another\nimport \"testing\"\nfunc TestA(t *testing.T) {}\nfunc TestB(t *testing.T) {}\n").unwrap();
    let mut executor = Packages(Fake::new(&fixture, [Response::Pass, Response::Pass]));
    let findings = run_with(
        &mut executor,
        &fixture.context(300),
        &fixture.options,
        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    )
    .unwrap();
    assert_eq!(findings, Vec::<String>::new());
    let calls = executor.0.tests();
    assert_eq!(calls.len(), 2);
    for call in &calls {
        assert!(call.args.iter().any(|arg| arg == "-test.run=^TestA$"));
    }
    let targets = calls
        .iter()
        .map(|call| {
            call.args[call.args.iter().position(|arg| arg == "go-test").unwrap() + 1].as_str()
        })
        .collect::<Vec<_>>();
    assert_eq!(targets, ["./tests/another", "./tests/example"]);
    let rows = fixture.rows();
    assert_eq!(rows.len(), 4);
    for row in rows {
        assert_eq!(
            (row.class, row.attempts),
            if row.name == "TestA" {
                (Class::Pass, 1)
            } else {
                (Class::NotRun, 0)
            }
        );
    }
}

#[test]
fn throwaway_installation_uses_valid_cooperative_config_and_explicit_public_flags() {
    let mut fixture = Fixture::new();
    fixture.options.config_source = "throwaway".into();
    fixture.options.max_tests = 1;
    let mut fake = Fake::new(&fixture, [Response::Pass]);
    assert_eq!(
        run_with(
            &mut fake,
            &fixture.context(300),
            &fixture.options,
            "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        )
        .unwrap(),
        Vec::<String>::new()
    );
    let install = fake
        .calls
        .iter()
        .find(|call| call.args[0] == "install")
        .unwrap();
    assert!(
        !install
            .args
            .iter()
            .any(|arg| arg == "--temporal-checkout" || arg == "--temporal-package")
    );
    assert_eq!(
        Fake::value(&install.args, "--memory-bytes"),
        (4_u64 << 30).to_string()
    );
    assert_eq!(
        Fake::value(&install.args, "--fuel"),
        (1_u64 << 50).to_string()
    );
    assert_eq!(
        Fake::value(&install.args, "--profile"),
        "gomad.wasi-cooperative/v1"
    );
    assert!(install.args.iter().any(|arg| arg == "--config"));
    assert_eq!(
        (
            fixture.rows()[0].class,
            fixture.rows()[0].config_source.as_str()
        ),
        (Class::Pass, "throwaway")
    );
}

#[test]
fn native_soft_assertion_progress_and_inconclusive_capacity_do_not_retry() {
    for (response, class) in [
        (
            Response::SoftAssertion { report_only: false },
            Class::GuestFailure,
        ),
        (Response::SoftAssertion { report_only: true }, Class::Pass),
        (Response::InconclusiveSoftAssertion, Class::CapacityExceeded),
        (Response::SoftAssertionWithoutProgress, Class::GuestFailure),
    ] {
        let mut fixture = Fixture::new();
        fixture.options.max_tests = 1;
        let mut fake = Fake::new(&fixture, [response]);
        run_with(
            &mut fake,
            &fixture.context(300),
            &fixture.options,
            "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        )
        .unwrap();
        let rows = fixture.rows();
        assert_eq!(rows.len(), 2);
        assert_eq!((rows[0].class, rows[0].attempts), (class, 1));
        assert_eq!((rows[1].class, rows[1].attempts), (Class::NotRun, 0));
        assert_eq!(fake.tests().len(), 1);
        if class == Class::GuestFailure {
            assert!(rows[0].reason.contains("soft_assertion"));
        }
    }
}

#[test]
fn soft_assertion_progress_must_match_its_authenticated_campaign_execution() {
    let fixture = Fixture::new();
    let fake = Fake::new(&fixture, []);
    let root = fixture.root.join("soft-attempt");
    directory(&root).unwrap();
    let package = Package {
        import_path: "example/tests/example".into(),
        pattern: "./tests/example".into(),
        module_directory: fixture.options.checkout.display().to_string(),
        target: "./tests/example".into(),
        tests: vec!["TestA".into()],
        top_level_tests: vec!["TestA".into()],
        units: BTreeMap::from([("TestA".into(), vec!["TestA".into()])]),
        guest_excluded: BTreeMap::new(),
        methods: vec![],
        registrations: vec![],
        list_count: 1,
        parsed_count: 1,
    };
    let dirs = Dirs::new(&fixture.options.work, &package.pattern);
    let command = Command {
        path: fixture.options.gomad.clone(),
        args: explore_args(
            &fixture.options,
            &package,
            &dirs,
            &root.join("artifacts"),
            "^TestA$",
            fixture.options.per_test_timeout,
        ),
        dir: root.clone(),
        env: BTreeMap::new(),
        timeout: Duration::from_secs(30),
    };
    let output = fake
        .native_with_soft_assertions(&command, false, Some(true))
        .stdout;
    let (progress, result) =
        output.split_at(output.iter().position(|byte| *byte == b'\n').unwrap());
    let progress: Value = serde_json::from_slice(progress).unwrap();
    let (observation, events) = authenticated_observation(
        "TestA",
        fixture.options.seed,
        &root,
        Completed {
            stdout: output.clone(),
            ..Completed::default()
        },
    )
    .unwrap();
    assert_eq!(
        classify::classify_native(&observation, &events).class,
        Class::Pass
    );
    for (key, value) in [
        ("seed", json!(8)),
        ("campaign_path", json!("/unrelated/campaign")),
        (
            "soft_assertions",
            json!({"messages":["different assertion"],"report_only":true,"inconclusive":false}),
        ),
    ] {
        let mut changed = progress.clone();
        changed[key] = value;
        let mut stdout = serde_json::to_vec(&changed).unwrap();
        stdout.extend_from_slice(result);
        assert!(
            authenticated_observation(
                "TestA",
                fixture.options.seed,
                &root,
                Completed {
                    stdout,
                    ..Completed::default()
                }
            )
            .is_err(),
            "{key}"
        );
    }
}
