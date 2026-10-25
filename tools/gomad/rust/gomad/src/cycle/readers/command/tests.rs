use super::*;
use serde_json::json;
use std::{
    fs,
    os::unix::fs::{PermissionsExt, symlink},
    process::Command,
};

struct Fixture {
    root: PathBuf,
    config: Value,
    row: Value,
}
impl Fixture {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "b04-command-{}-{}",
            std::process::id(),
            SEQUENCE.fetch_add(1, std::sync::atomic::Ordering::Relaxed)
        ));
        fs::create_dir(&root).unwrap();
        fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
        let executable_path = Path::new("/usr/bin/printf").canonicalize().unwrap();
        let executable = executable_path.as_path();
        let argv = vec![
            executable.to_string_lossy().into_owned(),
            "actual raw output".into(),
        ];
        let start = epoch();
        let child = Command::new(executable)
            .arg(&argv[1])
            .current_dir(&root)
            .stdout(std::process::Stdio::piped())
            .spawn()
            .unwrap();
        let pid = child.id();
        let output = child.wait_with_output().unwrap();
        assert!(output.status.success());
        let row = json!({"argv":argv,"command":argv,"argv_sha256":digest(&canonical(&json!(argv)).unwrap()),"cwd":root,"pid":pid,"stage_pid":std::process::id(),"start_ns":start,"end_ns":epoch(),"exit":0,"observation":"completed","active":false,"resources":{"pid":pid,"waited_pid":pid,"exit_code":0,"reaps":1,"wait4_calls":1},"cleanup_complete":true,"failure":null,"accepted_exits":[0],"private_context":null,"stdout":root.join("probe.stdout"),"stderr":root.join("probe.stderr"),"stdout_sha256":digest(&output.stdout),"stderr_sha256":digest(&output.stderr)});
        let config = json!({"root":root,"repo":root,"sent_epoch":start/1_000_000_000,"files":{executable.to_str().unwrap():digest(&fs::read(executable).unwrap())},"source_sha256":"a".repeat(64),"binary_sha256":"b".repeat(64),"identity":{"host":"c".repeat(64),"helper":"d".repeat(64)}});
        for (name, raw) in [
            ("probe.stdout", output.stdout),
            ("probe.stderr", output.stderr),
            ("probe.command.json", canonical(&row).unwrap()),
            ("probe.process.json", canonical(&row).unwrap()),
        ] {
            private(&root.join(name), &raw);
        }
        Self { root, config, row }
    }
    fn session(&self) -> RetainedSession {
        RetainedSession::from_value(self.config.clone()).unwrap()
    }
    fn write(&self, name: &str, row: &Value) {
        private(&self.root.join(name), &canonical(row).unwrap());
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.root).unwrap();
    }
}
static SEQUENCE: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
fn epoch() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_nanos()
        .try_into()
        .unwrap()
}
fn private(path: &Path, raw: &[u8]) {
    fs::write(path, raw).unwrap();
    fs::set_permissions(path, fs::Permissions::from_mode(0o600)).unwrap();
}
fn expected_command(
    argv: Vec<String>,
    cwd: PathBuf,
    seconds: u64,
) -> crate::cycle::process::Command {
    crate::cycle::process::Command {
        argv,
        cwd,
        environment: crate::cycle::process::Environment::Exact(BTreeMap::new()),
        declared_environment: BTreeMap::new(),
        temporary_paths: BTreeMap::new(),
        accepted: vec![0],
        timeout: std::time::Duration::from_secs(seconds),
        private_context: None,
    }
}
fn proof(case: &str, input: &Value, error: &str) {
    eprintln!(
        "B04_COMMAND_PROOF {}",
        json!({"case":case,"input":input,"input_sha256":digest(&canonical(input).unwrap()),"error":error,"source_sha256":digest(include_bytes!("mod.rs")),"test_sha256":digest(include_bytes!("tests.rs"))})
    );
}
#[test]
fn command_reopens_actual_streams_and_directly_refuses_relationship_mutations() {
    let mut fixture = Fixture::new();
    fixture.config["repo"] = json!("/");
    let argv: Vec<String> = serde_json::from_value(fixture.row["argv"].clone()).unwrap();
    let expected = expected_command(argv, fixture.root.clone(), 10);
    let run =
        |session: &mut RetainedSession| session.command(&fixture.root, "probe", &expected, false);
    assert_eq!(
        run(&mut fixture.session()).unwrap().stdout,
        b"actual raw output"
    );
    for (field, value, reason) in [
        ("accepted_exits", json!([0, 7]), "declared contract"),
        ("private_context", json!("foreign"), "declared contract"),
        ("pid", json!(0), "completed"),
        ("argv", json!(["foreign"]), "argv/cwd"),
        ("cwd", json!("/"), "argv/cwd"),
        ("start_ns", json!(0), "send interval"),
        ("end_ns", json!(0), "send interval"),
        ("exit", json!(7), "completed"),
        ("observation", json!("inconclusive"), "completed"),
        ("active", json!(true), "registration"),
        ("stage_pid", json!(0), "registration"),
        ("stdout_sha256", json!("e".repeat(64)), "raw"),
        ("argv_sha256", json!("e".repeat(64)), "raw"),
    ] {
        let mut row = fixture.row.clone();
        row[field] = value;
        fixture.write("probe.command.json", &row);
        fixture.write("probe.process.json", &row);
        let error = run(&mut fixture.session()).err().unwrap();
        assert!(error.contains(reason), "{field}: {error}");
        proof(field, &row, &error);
    }
    fixture.write("probe.command.json", &fixture.row);
    fixture.write("probe.process.json", &fixture.row);
    let mut process = fixture.row.clone();
    process["pid"] = json!(99);
    fixture.write("probe.process.json", &process);
    let error = run(&mut fixture.session()).err().unwrap();
    assert!(error.contains("process/command"));
    proof("process PID", &process, &error);
    fixture.write("probe.process.json", &fixture.row);
    private(&fixture.root.join("probe.stdout"), b"changed output");
    let error = run(&mut fixture.session()).err().unwrap();
    assert!(error.contains("raw"));
    proof("reopened stdout", &json!("changed output"), &error);
}
#[test]
fn command_custody_rejects_alias_mode_link_bound_and_root_substitution() {
    let fixture = Fixture::new();
    let session = fixture.session();
    let path = fixture.root.join("probe.stdout");
    assert_eq!(session.read(&path, 17).unwrap(), b"actual raw output");
    session.write(&path, b"actual raw output").unwrap();
    let error = session.write(&path, b"actual raw outpuX").unwrap_err();
    assert_eq!(error, "retained recomputation differs");
    proof(
        "retained recomputation equality",
        &json!({"actual":"actual raw output","expected":"actual raw outpuX"}),
        &error,
    );
    assert!(session.read(&path, 16).unwrap_err().contains("bound"));
    fs::set_permissions(&path, fs::Permissions::from_mode(0o644)).unwrap();
    assert!(session.read(&path, 17).is_err());
    fs::set_permissions(&path, fs::Permissions::from_mode(0o600)).unwrap();
    let alias = fixture.root.join("alias");
    symlink(&path, &alias).unwrap();
    assert!(session.read(&alias, 17).is_err());
    fs::remove_file(&alias).unwrap();
    fs::hard_link(&path, &alias).unwrap();
    assert!(session.read(&path, 17).is_err());
    fs::remove_file(&alias).unwrap();
    assert!(session.read(Path::new("/etc/passwd"), 1000).is_err());
    let moved = fixture.root.with_extension("moved");
    fs::rename(&fixture.root, &moved).unwrap();
    fs::create_dir(&fixture.root).unwrap();
    fs::set_permissions(&fixture.root, fs::Permissions::from_mode(0o700)).unwrap();
    assert!(session.read(&path, 17).is_err());
    fs::remove_dir(&fixture.root).unwrap();
    fs::rename(&moved, &fixture.root).unwrap();
}

pub(crate) fn run_cases(owner: &crate::cycle::process::CycleOwner) -> Result<(), String> {
    use crate::cycle::process::{Environment, Runner};
    use std::{sync::atomic::AtomicBool, time::Duration};
    if std::env::var("B04_CHILD").as_deref() == Ok("inspect") {
        return crate::cli::inspect::run(
            &std::env::args().skip(2).collect::<Vec<_>>(),
            &mut std::io::stdout(),
        )
        .map_err(|e| e.message);
    }
    let root = PathBuf::from(std::env::var("B04_COMMAND_ROOT").map_err(|e| e.to_string())?);
    fs::create_dir_all(&root).map_err(|e| e.to_string())?;
    fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).map_err(|e| e.to_string())?;
    if std::env::var("B04_CHILD").as_deref() == Ok("prepare") {
        return genuine_preparation(&root);
    }
    let registry = root.join("commands");
    let mut runner = Runner::create(owner, &registry, Some("b04-actual-preparation".into()))?;
    let executable_path = Path::new("/usr/bin/printf")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let executable = executable_path.to_str().ok_or("non-UTF8 printf")?;
    let argv = vec![executable.into(), "genuine-supervised-output".into()];
    let sent = epoch() / 1_000_000_000;
    let cancelled = AtomicBool::new(false);
    let command = crate::cycle::process::Command {
        argv: argv.clone(),
        cwd: root.clone(),
        environment: Environment::Exact(BTreeMap::new()),
        declared_environment: BTreeMap::new(),
        temporary_paths: BTreeMap::new(),
        accepted: vec![0],
        timeout: Duration::from_secs(5),
        private_context: None,
    };
    let completed = runner
        .start("actual", command.clone(), &cancelled, |_| Ok(()))
        .map_err(|e| e.to_string())?
        .wait(&cancelled)?;
    require(completed.accepted(), "supervised actual control failed")?;
    let config = json!({"root":root,"repo":root,"sent_epoch":sent,"files":{executable:digest(&fs::read(executable).map_err(|e|e.to_string())?)},"source_sha256":"a".repeat(64),"binary_sha256":"b".repeat(64),"identity":{"host":"c".repeat(64),"helper":"d".repeat(64)}});
    let mut session = RetainedSession::from_value(config)?;
    let admitted = session.command(&registry, "actual", &command, false)?;
    require(
        admitted.stdout == b"genuine-supervised-output" && admitted.stderr.is_empty(),
        "actual supervisor streams changed",
    )?;
    let row = admitted.row;
    let actual = registry.join("actual.command.json");
    let process = registry.join("actual.process.json");
    for (field, value, reason) in [
        ("pid", json!(0), "completed"),
        ("start_ns", json!(0), "send interval"),
        ("active", json!(true), "registration"),
        ("argv_sha256", json!("e".repeat(64)), "raw"),
        ("cleanup_complete", json!(false), "cleanup"),
    ] {
        let mut changed = row.clone();
        changed[field] = value;
        private(&actual, &canonical(&changed)?);
        private(&process, &canonical(&changed)?);
        let error = session
            .command(&registry, "actual", &command, false)
            .err()
            .ok_or("actual mutant accepted")?;
        require(error.contains(reason), "wrong actual predicate")?;
        proof(&format!("actual {field}"), &changed, &error);
    }
    private(&actual, &canonical(&row)?);
    private(&process, &canonical(&row)?);
    session.command(&registry, "actual", &command, false)?;
    let mut deadline = command.clone();
    deadline.timeout = Duration::ZERO;
    let error = session
        .command(&registry, "actual", &deadline, false)
        .err()
        .ok_or("original timeout ignored")?;
    require(
        error.contains("bound exceeded"),
        "wrong actual timeout predicate",
    )?;
    proof("actual elapsed timeout", &row, &error);
    let error = session
        .command(&registry, "actual", &command, true)
        .err()
        .ok_or("required capture omitted")?;
    require(
        error.contains("lacks actual preparation"),
        "wrong missing capture predicate",
    )?;
    proof("actual missing required capture", &row, &error);
    let mut foreign = session.config.clone();
    foreign["files"]
        .as_object_mut()
        .ok_or("files map missing")?
        .remove(executable);
    let error = RetainedSession::from_value(foreign)?
        .command(&registry, "actual", &command, false)
        .err()
        .ok_or("unsealed executable accepted")?;
    require(error.contains("not sealed"), "wrong executable predicate")?;
    proof("actual executable not sealed", &row, &error);
    let mut changed = row.clone();
    changed["end_ns"] = json!(unsigned(&row, "end_ns")? + 1);
    private(&process, &canonical(&changed)?);
    let error = session
        .command(&registry, "actual", &command, false)
        .err()
        .ok_or("process end escaped command interval")?;
    require(
        error.contains("process/command"),
        "wrong process end predicate",
    )?;
    proof("actual process end interval", &changed, &error);
    private(&process, &canonical(&row)?);
    let owner_path = root.join("producer-root.json");
    let owner = json!({"pid":row["stage_pid"],"start_ns":row["start_ns"]});
    private(&owner_path, &canonical(&owner)?);
    session.command(&registry, "actual", &command, false)?;
    for (field, value) in [
        ("pid", json!(unsigned(&row, "stage_pid")? + 1)),
        ("start_ns", json!(unsigned(&row, "end_ns")? + 1)),
    ] {
        let mut changed = owner.clone();
        changed[field] = value;
        private(&owner_path, &canonical(&changed)?);
        let error = session
            .command(&registry, "actual", &command, false)
            .err()
            .ok_or("foreign producer owner accepted")?;
        require(
            error.contains("actual producer"),
            "wrong producer owner predicate",
        )?;
        proof(&format!("actual producer {field}"), &changed, &error);
    }
    fs::remove_file(owner_path).map_err(|e| e.to_string())?;
    println!(
        "B04_COMMAND_ACTUAL {}",
        json!({"pid":completed.pid,"supervisor":completed.supervisor_pid,"exit":completed.exit,"command":row,"control":"normal-return","source_sha256":digest(include_bytes!("mod.rs"))})
    );
    genuine_command_preparation(&root, &registry, &mut runner)?;
    genuine_identity(&root, &mut runner)?;
    Ok(())
}

fn genuine_preparation(root: &Path) -> Result<(), String> {
    use std::time::Duration;
    let source = root.join("source");
    fs::create_dir(&source).map_err(|e| e.to_string())?;
    fs::write(
        source.join("go.mod"),
        b"module b04.current.fixture\n\ngo 1.27.1\n",
    )
    .map_err(|e| e.to_string())?;
    fs::write(
        source.join("main.go"),
        b"package main\nimport (\"os\";\"time\")\nfunc main(){a,b:=make(chan int,1),make(chan int,1);a<-1;b<-2;select{case <-a:case <-b:};time.Sleep(time.Nanosecond);if len(os.Args)>1 && os.Args[1]==\"canary\" {panic(\"b04 expected canary\")};println(\"b04-current-preparation\")}\n",
    )
    .map_err(|e| e.to_string())?;
    let compiler = PathBuf::from(
        "/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go",
    )
    .canonicalize()
    .map_err(|e| e.to_string())?;
    let installation_path = root.join("installation.json");
    crate::cli::install::run(
        &[
            "--compiler".into(),
            compiler.to_string_lossy().into_owned(),
            "--cache".into(),
            root.join("cache").to_string_lossy().into_owned(),
            "--compile-cache".into(),
            root.join("engine-cache").to_string_lossy().into_owned(),
            "--go-version=go1.27.1".into(),
            "--output".into(),
            installation_path.to_string_lossy().into_owned(),
        ],
        &mut Vec::new(),
    )
    .map_err(|e| e.message)?;
    let mut options = crate::cli::explore::parse(&["go-run".into(), ".".into()], &mut false)?;
    options.installation = Some(installation_path.clone());
    options.working_dir = source.clone();
    options.artifacts = root.join("campaigns");
    options.seeds = "7".into();
    options.parallel = 1;
    options.success_bytes = 512 << 20;
    let (prepared, _) = crate::campaign::prepare::prepare_with_compiler_controls(
        &options,
        &crate::campaign::prepare::CompilerControls {
            parallelism: Some(1),
            environment: BTreeMap::from([("GOMAXPROCS".into(), "1".into())]),
        },
    )?;
    let provenance = custody::strict_json(&prepared.provenance)?;
    let inputs = &provenance["inputs"];
    let execution = crate::campaign::execution::run(
        &prepared,
        &crate::campaign::ledger::Job {
            ordinal: 0,
            seed: 7,
        },
        Duration::from_secs(60),
        &gomad_wasmhost::linked::Cancellation::default(),
    )?
    .finalized()
    .map_err(|e| e.message())?;
    require(
        execution.publication.record.outcome.exit_code == Some(0),
        "genuine prepared guest failed",
    )?;
    let plan = crate::campaign::preparation::retain_admitted(&options, prepared)?;
    let campaign = plan.campaign;
    let sidecar = campaign.join("preparation-observations");
    let raw = fs::read(sidecar.join("receipt.json")).map_err(|e| e.to_string())?;
    let receipt = custody::strict_json(&raw)?;
    let files = receipt["files"]
        .as_object()
        .ok_or("genuine receipt files missing")?
        .iter()
        .map(|(name, entry)| {
            (
                sidecar.join(name).to_string_lossy().into_owned(),
                entry["sha256"].clone(),
            )
        })
        .collect::<serde_json::Map<_, _>>();
    let executable = std::env::current_exe().map_err(|e| e.to_string())?;
    let binary = digest(&custody::read_file(&executable, 512 << 20)?);
    let mut frozen: serde_json::Map<String, Value> =
        serde_json::from_value(inputs["compiler"].clone()).map_err(|e| e.to_string())?;
    frozen.extend(
        serde_json::from_value::<serde_json::Map<String, Value>>(inputs["sources"].clone())
            .map_err(|e| e.to_string())?,
    );
    frozen.insert(
        compiler.to_string_lossy().into_owned(),
        inputs["compiler_sha256"].clone(),
    );
    frozen.insert(executable.to_string_lossy().into_owned(), json!(binary));
    let repository = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    for name in super::result::CHOICE_SOURCES {
        let path = repository.join("tools/gomad/internal/model").join(name);
        frozen.insert(
            path.to_string_lossy().into_owned(),
            json!(digest(&custody::read_file(&path, 512 << 20)?)),
        );
    }
    let config = json!({"root":root,"repo":repository,"files":frozen,"source_sha256":digest(&custody::read_file(&source.join("main.go"),1<<20)?),"binary_sha256":binary,"binary":executable,"identity":{"host":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),"helper":gomad_wasmhost::linked::IMPLEMENTATION_SHA256}});
    let seal = json!({"source_sha256":config["source_sha256"],"binary_sha256":binary,"campaigns":[{"path":campaign,"receipt_path":sidecar.join("receipt.json"),"receipt_sha256":digest(&raw),"files":files}]});
    let row = json!({"pid":std::process::id(),"argv":[executable,"--installation",installation_path,"prepare"],"private_observations":seal});
    let control = || {
        let mut session = RetainedSession::from_value(config.clone())?;
        session.capture_preparations(&row)?;
        Ok::<_, String>(session)
    };
    let mut session = control()?;
    require(
        session.prepared_observation(&campaign, &provenance)?.1 == u64::from(std::process::id()),
        "genuine preparation owner lost",
    )?;
    session.pending_preparations.push(preparation::Pending {
        campaign: campaign.clone(),
        provenance: provenance.clone(),
        owner: u64::from(std::process::id()),
        campaigns: BTreeSet::from([campaign.clone()]),
    });
    session.verify_pending_preparations()?;
    session.authenticate_prepared_artifact(&campaign, &provenance)?;
    for (field, value, reason) in [
        ("source_sha256", json!("f".repeat(64)), "source/binary"),
        ("binary_sha256", json!("f".repeat(64)), "source/binary"),
    ] {
        let mut changed = row.clone();
        changed["private_observations"][field] = value;
        let error = RetainedSession::from_value(config.clone())?
            .capture_preparations(&changed)
            .unwrap_err();
        require(error.contains(reason), "wrong outer preparation predicate")?;
        proof(field, &changed, &error);
    }
    let mut wrong_owner = row.clone();
    wrong_owner["pid"] = json!(std::process::id() + 1);
    let error = RetainedSession::from_value(config.clone())?
        .capture_preparations(&wrong_owner)
        .unwrap_err();
    require(
        error.contains("owner mismatch"),
        "wrong preparation owner predicate",
    )?;
    proof("preparation owner", &wrong_owner, &error);
    let mut missing = row.clone();
    missing["private_observations"]["campaigns"] = json!([]);
    let error = RetainedSession::from_value(config.clone())?
        .capture_preparations(&missing)
        .unwrap_err();
    require(
        error.contains("campaign set"),
        "wrong complete campaign predicate",
    )?;
    proof("preparation empty campaigns", &missing, &error);
    for (field, value, reason) in [
        (
            "prepared_record_sha256",
            json!("f".repeat(64)),
            "stable commitments",
        ),
        ("module_sha256", json!("f".repeat(64)), "stable commitments"),
        (
            "provenance_sha256",
            json!("f".repeat(64)),
            "stable commitments",
        ),
        ("build_key", json!("f".repeat(64)), "stable commitments"),
        (
            "compiler_sha256",
            json!("f".repeat(64)),
            "stable commitments",
        ),
        (
            "sources_sha256",
            json!("f".repeat(64)),
            "stable commitments",
        ),
    ] {
        let mut changed = receipt.clone();
        changed[field] = value;
        let encoded = canonical(&changed)?;
        private(&sidecar.join("receipt.json"), &encoded);
        let mut outer = row.clone();
        outer["private_observations"]["campaigns"][0]["receipt_sha256"] = json!(digest(&encoded));
        let error = RetainedSession::from_value(config.clone())?
            .capture_preparations(&outer)
            .unwrap_err();
        require(
            error.contains(reason) || error.contains("preparation observation"),
            "wrong resealed preparation commitment predicate",
        )?;
        proof(
            &format!("resealed preparation {field}"),
            &json!({"row":outer,"receipt":changed}),
            &error,
        );
        private(&sidecar.join("receipt.json"), &raw);
        control()?;
    }
    let original = provenance.clone();
    for field in [
        "host",
        "helper",
        "compiler_path",
        "compiler_sha256",
        "schema",
        "module_metadata",
    ] {
        let mut changed = original.clone();
        changed["inputs"][field] = if field == "module_metadata" {
            json!({"imports":[],"initial_memory_pages":2})
        } else {
            json!("foreign")
        };
        changed["build_key"] = json!(digest(&canonical(&changed["inputs"])?));
        let error = control()?
            .provenance(&changed)
            .err()
            .ok_or("rehashed provenance accepted")?;
        proof(&format!("provenance {field}"), &changed, &error);
    }
    let mut pending = control()?;
    pending.pending_preparations.push(preparation::Pending {
        campaign: campaign.clone(),
        provenance: provenance.clone(),
        owner: 0,
        campaigns: BTreeSet::from([campaign.clone()]),
    });
    let error = pending.verify_pending_preparations().unwrap_err();
    require(
        error.contains("campaign set"),
        "wrong pending owner predicate",
    )?;
    proof(
        "pending producer",
        &json!({"owner":0,"campaign":campaign}),
        &error,
    );
    preparation_guards(&config, &row, &receipt, &campaign)?;
    for (cluster, canary) in [(false, false), (false, true), (true, false)] {
        let case = root.join(if cluster {
            "cluster-result"
        } else if canary {
            "canary-result"
        } else {
            "standalone-result"
        });
        fs::create_dir(&case).map_err(|e| e.to_string())?;
        fs::set_permissions(&case, fs::Permissions::from_mode(0o700)).map_err(|e| e.to_string())?;
        let installed = if cluster {
            fixture_cooperative_installation(&case, root, &compiler)?
        } else {
            installation_path.clone()
        };
        genuine_result(&case, &source, &installed, &config, cluster, canary)?;
    }
    println!(
        "B04_PREPARATION_ACTUAL {}",
        json!({"control":"normal-return","guest_exit":execution.publication.record.outcome.exit_code,"row":row,"receipt":receipt,"provenance":provenance,"config":config})
    );
    Ok(())
}

fn fixture_cooperative_installation(
    case: &Path,
    cache_root: &Path,
    compiler: &Path,
) -> Result<PathBuf, String> {
    let path = case.join("cooperative-installation.json");
    crate::cli::install::run(
        &[
            "--compiler".into(),
            compiler.to_string_lossy().into_owned(),
            "--cache".into(),
            cache_root.join("cache").to_string_lossy().into_owned(),
            "--compile-cache".into(),
            cache_root
                .join("engine-cache")
                .to_string_lossy()
                .into_owned(),
            "--go-version=go1.27.1".into(),
            "--runtime-root".into(),
            Path::new(env!("CARGO_MANIFEST_DIR"))
                .join("../../internal/model/runtimesource")
                .canonicalize()
                .map_err(|e| e.to_string())?
                .to_string_lossy()
                .into_owned(),
            "--profile=gomad.wasi-cooperative/v1".into(),
            "--output".into(),
            path.to_string_lossy().into_owned(),
        ],
        &mut Vec::new(),
    )
    .map_err(|error| error.message)?;
    let installed: crate::install::manifest::Installation =
        serde_json::from_slice(&custody::read_file(&path, 16 << 20)?).map_err(|e| e.to_string())?;
    crate::install::manifest::validate(&installed)?;
    Ok(path)
}

fn fixture_specification(
    root: &Path,
    source: &Path,
    manifest_path: &Path,
    source_sha256: &str,
    cluster: bool,
    canary: bool,
) -> Result<Value, String> {
    let mut specification = json!({"Manifest":{"ID":"b04-receiving-control","Workload":"fixture","IntegratedSourceSHA256":format!("sha256:{source_sha256}"),"IntegratedSourceManifestPath":manifest_path,"Seed":7,"Repeat":2},"Campaign":{"Target":{"Backend":"wasm","Kind":"go-run","Source":".","WorkingDir":source},"Environment":[],"Artifacts":root.join("result-campaigns"),"ExecutionTimeout":60_000_000_000_u64,"OverallTimeout":600_000_000_000_u64},"ArtifactRoot":root.join("results")});
    specification["Manifest"]["Cell"] = json!(if canary { "canary" } else { "success" });
    if canary {
        specification["Campaign"]["Target"]["Args"] = json!(["canary"]);
    }
    if cluster {
        specification["Manifest"]["Workload"] = json!("multi-instance");
        let mut scenario = custody::strict_json(include_bytes!(
            "../../../qualify/workloads/testdata/sqlite.json"
        ))?;
        scenario
            .as_object_mut()
            .ok_or("scenario map missing")?
            .remove("workload");
        scenario["spec"]["nodes"] = json!([{"id":"a","boot":"guest","address":"10.0.0.1"},{"id":"b","boot":"guest","address":"10.0.0.2"}]);
        scenario["spec"]["volumes"] = json!([]);
        scenario["spec"]["links"] = json!([]);
        scenario["boots"] = json!([{"id":"guest"}]);
        scenario["actions"] = json!([{"action":"start","handle":{"node":"a","incarnation":1}},{"action":"start","handle":{"node":"b","incarnation":1}},{"action":"wait","handle":{"node":"a","incarnation":1}},{"action":"wait","handle":{"node":"b","incarnation":1}}]);
        for field in [
            "incarnation_boots",
            "output_crashes",
            "output_stops",
            "network_faults",
        ] {
            scenario[field] = json!([]);
        }
        specification["Campaign"]["Target"]["Scenario"] =
            json!(crate::campaign::exploration::base64(&canonical(&scenario)?));
    }
    Ok(specification)
}

#[test]
fn actual_fixture_declarations_match_current_spec_manifest_and_scenario_schemas() {
    use crate::qualify::{cells, result, spec};
    let fixture = Fixture::new();
    let compiler = Path::new(
        "/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go",
    )
    .canonicalize()
    .unwrap();
    fixture_cooperative_installation(&fixture.root, &fixture.root, &compiler).unwrap();
    let root = Path::new("/private/b04-fixture-declaration");
    let digest = "a".repeat(64);
    for (cluster, canary) in [(false, false), (false, true), (true, false)] {
        let value = fixture_specification(
            root,
            &root.join("source"),
            &root.join("source.manifest"),
            &digest,
            cluster,
            canary,
        )
        .unwrap();
        let declared: spec::Spec = spec::strict(&canonical(&value).unwrap()).unwrap();
        let manifest = result::Manifest {
            schema: result::MANIFEST_SCHEMA.into(),
            provisional: false,
            integrated_source_manifest_path: Some(
                declared.manifest.integrated_source_manifest_path.clone(),
            ),
            id: declared.manifest.id.clone(),
            workload: declared.manifest.workload.clone(),
            profile: "controlled".into(),
            cell: Some(declared.manifest.cell.clone()),
            integrated_source_sha256: digest.clone(),
            workload_source_sha256: digest.clone(),
            runtime_sha256: digest.clone(),
            compiler_sha256: digest.clone(),
            helper_sha256: digest.clone(),
            host_sha256: digest.clone(),
            model_sha256: digest.clone(),
            engine_sha256: digest.clone(),
            host_config_sha256: digest.clone(),
            module_sha256: digest.clone(),
            toolchain_sha256: digest.clone(),
            scenario_sha256: cluster.then(|| digest.clone()),
            host_os: std::env::consts::OS.into(),
            host_arch: std::env::consts::ARCH.into(),
            seed: declared.manifest.seed,
            repeat: declared.manifest.repeat,
            fuel: 1,
            memory_bytes: 65536,
            unsupported: Vec::new(),
        };
        manifest.validate().unwrap();
        if let Some(encoded) = declared.campaign.target.scenario.as_ref() {
            let bytes = cells::base64(encoded.as_str().unwrap()).unwrap();
            assert_eq!(
                crate::campaign::scenario::evidence_limit(&bytes, 123).unwrap(),
                246
            );
        }
    }
}

fn genuine_result(
    root: &Path,
    source: &Path,
    installation: &Path,
    config: &Value,
    cluster: bool,
    canary: bool,
) -> Result<(), String> {
    use crate::qualify::{cells, result, spec, usage};
    let manifest_path = root.join("source.manifest");
    let source_file = source.join("main.go");
    let manifest = format!(
        "{}  {}\n",
        digest(&custody::read_file(&source_file, 1 << 20)?),
        source_file.display()
    );
    private(&manifest_path, manifest.as_bytes());
    let installed: crate::install::manifest::Installation =
        serde_json::from_slice(&custody::read_file(installation, 16 << 20)?)
            .map_err(|e| e.to_string())?;
    crate::install::manifest::admit_build(&installed)?;
    let mut configured = config.clone();
    configured["source_manifest"] = json!(manifest_path);
    configured["source_sha256"] = json!(digest(manifest.as_bytes()));
    for (path, sha) in &installed
        .admission
        .as_ref()
        .ok_or("actual installation admission absent")?
        .compiler_inputs
    {
        configured["files"][path.to_str().ok_or("actual compiler path UTF8")?] = json!(sha);
    }
    let specification = fixture_specification(
        root,
        source,
        &manifest_path,
        &digest(manifest.as_bytes()),
        cluster,
        canary,
    )?;
    let specification: spec::Spec = spec::strict(&canonical(&specification)?)?;
    usage::enable_phases();
    let frozen = cells::freeze(
        &specification,
        Some(installation),
        vec!["b04-focused-receiving-control".into()],
        usage::snapshot(),
    )?;
    let retained = cells::run(&frozen)?;
    require(
        retained.report.qualified == !canary
            && retained.report.target_success == !canary
            && retained.report.executions.iter().all(|row| {
                row.record.outcome.class == if canary { "target_failure" } else { "success" }
            }),
        "genuine Result guest/canary/replay classification failed",
    )?;
    let published = cells::publish(&frozen, retained)?;
    let retained = result::open(&published.path)?;
    let (receipt, terminal) = result::open_receipt(&published.path, &retained)?;
    RetainedSession::final_publication(&retained, &receipt, &terminal)?;
    let result_raw = custody::read_file(&published.path, 64 << 20)?;
    let result_digest = digest(&result_raw);
    for (pointer, value) in [
        ("/schema", json!("foreign")),
        ("/result_sha256", json!("f".repeat(64))),
        ("/operator_process_id", json!(0)),
        ("/linked_after", Value::Null),
        ("/linked_after/user_nanos", json!(0)),
        ("/linked_after/peak_rss_bytes", json!(0)),
        ("/operator_peak_rss_bytes", Value::Null),
        ("/operator_peak_rss_bytes", json!(0)),
        ("/operator_peak_rss_bytes", json!(3_u64 << 30)),
    ] {
        let mut changed = serde_json::to_value(&receipt).map_err(|e| e.to_string())?;
        *changed
            .pointer_mut(pointer)
            .ok_or("actual receipt pointer missing")? = value;
        let typed = serde_json::from_value(changed.clone()).map_err(|e| e.to_string())?;
        let error = result::validate_receipt(&retained, &typed, &result_digest)
            .and_then(|tail| RetainedSession::final_publication(&retained, &typed, &tail))
            .unwrap_err();
        proof(&format!("actual publication {pointer}"), &changed, &error);
    }
    let mut noncanonical = result_raw.clone();
    noncanonical.push(b' ');
    let alternate = published
        .path
        .parent()
        .ok_or("result parent missing")?
        .join(format!("result-{}.json", digest(&noncanonical)));
    private(&alternate, &noncanonical);
    let error = result::open(&alternate)
        .err()
        .ok_or("noncanonical Result accepted")?;
    require(
        error.contains("canonical"),
        "wrong Result canonical predicate",
    )?;
    proof(
        "rehashed noncanonical actual Result",
        &json!({"path":alternate,"raw_sha256":digest(&noncanonical)}),
        &error,
    );
    fs::remove_file(alternate).map_err(|e| e.to_string())?;
    let reported = json!({"result":retained,"receipt":receipt,"publication":terminal});
    let mut session = RetainedSession::from_value(configured.clone())?;
    session.result(&published.path, &reported)?;
    session.provenance(&retained.provenance)?;
    let mut foreign_compiler = configured.clone();
    let compiler_path = text(&retained.provenance["inputs"], "compiler_path")?;
    foreign_compiler["files"][compiler_path] = json!("f".repeat(64));
    let error = RetainedSession::from_value(foreign_compiler)?
        .provenance(&retained.provenance)
        .unwrap_err();
    require(
        error.contains("compiler path/hash not frozen"),
        "wrong actual compiler pin predicate",
    )?;
    proof(
        "actual provenance compiler not frozen",
        &retained.provenance,
        &error,
    );
    if cluster {
        for name in super::result::CHOICE_SOURCES {
            let path = Path::new(text(&configured, "repo")?)
                .join("tools/gomad/internal/model")
                .join(name);
            let mut missing = configured.clone();
            missing["files"]
                .as_object_mut()
                .ok_or("frozen files absent")?
                .remove(path.to_str().ok_or("choice path UTF8")?);
            let error = RetainedSession::from_value(missing)?
                .provenance(&retained.provenance)
                .unwrap_err();
            require(
                error.contains("not frozen"),
                &format!("wrong actual choice source pin predicate: {error}"),
            )?;
            proof(
                "actual cooperative choice source not frozen",
                &json!({"path":path}),
                &error,
            );
        }
        for (field, value) in [
            ("GOOS", json!("foreign")),
            ("GOARCH", json!("foreign")),
            ("ToolchainBuildKey", json!("f".repeat(64))),
            ("TargetSHA256", json!(vec![0; 32])),
            ("ImplementationSHA256", json!(vec![0; 32])),
        ] {
            let mut changed = retained.provenance.clone();
            changed["choice_identity"][field] = value;
            let error = session
                .provenance(&changed)
                .err()
                .ok_or("actual cooperative choice mutation accepted")?;
            proof(
                &format!("actual cooperative choice {field}"),
                &changed,
                &error,
            );
        }
    }

    let mut campaigns = Vec::new();
    for execution in &retained.report.executions {
        let campaign = Path::new(&execution.campaign_path);
        let sidecar = campaign.join("preparation-observations");
        let raw = custody::read_file(&sidecar.join("receipt.json"), 16 << 20)?;
        let receipt = custody::strict_json(&raw)?;
        let files = receipt["files"]
            .as_object()
            .ok_or("receipt files")?
            .iter()
            .map(|(name, entry)| {
                (
                    sidecar.join(name).to_string_lossy().into_owned(),
                    entry["sha256"].clone(),
                )
            })
            .collect::<serde_json::Map<_, _>>();
        campaigns.push(json!({"path":campaign,"receipt_path":sidecar.join("receipt.json"),"receipt_sha256":digest(&raw),"files":files}));
    }
    let row = json!({"pid":std::process::id(),"argv":[configured["binary"],"--installation",installation,"qualify"],"private_observations":{"source_sha256":configured["source_sha256"],"binary_sha256":configured["binary_sha256"],"campaigns":campaigns}});
    session.capture_preparations(&row)?;
    session.verify_pending_preparations()?;
    for field in ["result", "receipt", "publication"] {
        let mut changed = reported.clone();
        changed[field] = json!({});
        let error = RetainedSession::from_value(configured.clone())?
            .result(&published.path, &changed)
            .err()
            .ok_or("public Result mutation accepted")?;
        require(
            error.contains("public report"),
            "wrong report projection predicate",
        )?;
        proof(&format!("public {field}"), &changed, &error);
    }
    for (field, value) in [
        ("source_sha256", json!("f".repeat(64))),
        (
            "identity",
            json!({"host":"f".repeat(64),"helper":"e".repeat(64)}),
        ),
    ] {
        let mut changed = configured.clone();
        changed[field] = value;
        let error = RetainedSession::from_value(changed.clone())?
            .result(&published.path, &reported)
            .err()
            .ok_or("foreign Result source/role accepted")?;
        proof(&format!("Result frozen {field}"), &changed, &error);
    }
    for (pointer, value, reason) in [
        ("/manifest/provisional", json!(true), "foreign/final"),
        ("/manifest/seed", json!(8), "foreign/final"),
        ("/manifest/repeat", json!(3), "foreign/final"),
        (
            "/manifest/host_sha256",
            json!("f".repeat(64)),
            "foreign/final",
        ),
        (
            "/manifest/helper_sha256",
            json!("f".repeat(64)),
            "foreign/final",
        ),
        (
            "/report/qualified",
            json!(!retained.report.qualified),
            "deterministic final",
        ),
        ("/report/deterministic", json!(false), "deterministic final"),
        (
            "/report/target_success",
            json!(!retained.report.target_success),
            "deterministic final",
        ),
        (
            "/report/executions/1/iteration",
            json!(3),
            "deterministic final",
        ),
        ("/cost_failure", json!("unmeasured"), "raw accounting"),
        ("/budget/within_budget", json!(false), "raw accounting"),
    ] {
        let mut changed = serde_json::to_value(&retained).map_err(|e| e.to_string())?;
        *changed
            .pointer_mut(pointer)
            .ok_or("missing genuine Result field")? = value;
        let typed = serde_json::from_value(changed.clone()).map_err(|e| e.to_string())?;
        let error = RetainedSession::from_value(configured.clone())?
            .final_contract(&typed)
            .unwrap_err();
        require(
            error.contains(reason),
            "wrong admitted final Result predicate",
        )?;
        proof(pointer, &changed, &error);
    }
    for (pointer, value, reason) in [
        ("/schema", json!("foreign"), "result schema"),
        ("/report/schema", json!("foreign"), "qualification report"),
        (
            "/report/executions/0/replay",
            Value::Null,
            "omitted its replay",
        ),
        (
            "/report/executions/0/replay/attempted",
            json!(false),
            "replay evidence",
        ),
        (
            "/report/executions/0/replay/matched",
            json!(false),
            "replay evidence",
        ),
        (
            "/report/executions/0/replay/divergence",
            json!("receiver-mutant"),
            "replay evidence",
        ),
        (
            "/report/executions/0/replay/choice_replay",
            json!("unavailable"),
            "replay evidence",
        ),
        (
            "/manifest/host_arch",
            json!("foreign-arch"),
            "execution identity",
        ),
        (
            "/observations/samples/0/trace_bytes",
            json!(retained.observations.samples[0].trace_bytes + 1),
            "trace bytes",
        ),
        (
            "/observations/samples/0/guest_memory_limit_bytes",
            json!(retained.manifest.memory_bytes + 1),
            "store sample",
        ),
    ] {
        let mut changed = serde_json::to_value(&retained).map_err(|e| e.to_string())?;
        *changed
            .pointer_mut(pointer)
            .ok_or("actual Result mutation pointer absent")? = value;
        let typed = serde_json::from_value(changed.clone()).map_err(|e| e.to_string())?;
        let error = result::validate(&typed).unwrap_err();
        require(
            error.contains(reason),
            "wrong shared Result inner predicate",
        )?;
        proof(&format!("actual Result {pointer}"), &changed, &error);
    }
    let mut duplicate = retained.clone();
    duplicate.report.executions[1].artifact_path =
        duplicate.report.executions[0].artifact_path.clone();
    let duplicate_path = result::publish(
        published.path.parent().ok_or("Result directory absent")?,
        &duplicate,
    )?;
    let duplicate_raw = custody::read_file(&duplicate_path, 64 << 20)?;
    let mut duplicate_receipt = receipt.clone();
    duplicate_receipt.result_sha256 = digest(&duplicate_raw);
    result::publish_receipt(&duplicate_path, &duplicate_receipt)?;
    let duplicate_reported =
        json!({"result":duplicate,"receipt":duplicate_receipt,"publication":terminal});
    let error = RetainedSession::from_value(configured.clone())?
        .result(&duplicate_path, &duplicate_reported)
        .unwrap_err();
    require(
        error.contains("duplicate qualification artifact"),
        "wrong actual duplicate artifact predicate",
    )?;
    proof(
        "actual Result duplicate qualification artifact",
        &duplicate_reported,
        &error,
    );
    if !cluster && !canary {
        for case in [
            "missing",
            "multiple",
            "mode",
            "link",
            "bound",
            "canonical",
            "name",
        ] {
            let directory = root.join(format!("receipt-{case}"));
            let path = result::publish(&directory, &retained)?;
            let mut bytes = canonical(&receipt)?;
            bytes.push(b'\n');
            if case == "bound" {
                bytes.resize((8 << 10) + 1, b' ');
            }
            if case == "canonical" {
                bytes.push(b' ');
            }
            let name = format!(
                "{}.publication-{}.json",
                path.file_name()
                    .ok_or("Result name absent")?
                    .to_string_lossy(),
                if case == "name" {
                    "f".repeat(64)
                } else {
                    digest(&bytes)
                }
            );
            let receipt_path = path.with_file_name(name);
            if case != "missing" {
                private(&receipt_path, &bytes);
            }
            match case {
                "multiple" => private(
                    &path.with_file_name(format!(
                        "{}.publication-{}.json",
                        path.file_name()
                            .ok_or("Result name absent")?
                            .to_string_lossy(),
                        "e".repeat(64)
                    )),
                    &bytes,
                ),
                "mode" => fs::set_permissions(&receipt_path, fs::Permissions::from_mode(0o644))
                    .map_err(|e| e.to_string())?,
                "link" => fs::hard_link(&receipt_path, directory.join("linked-receipt"))
                    .map_err(|e| e.to_string())?,
                _ => {}
            }
            let error = result::open_receipt(&path, &retained).unwrap_err();
            let reason = match case {
                "missing" | "multiple" => "exactly one",
                "canonical" => "not canonical",
                "name" => "content identity",
                _ => "artifact",
            };
            require(
                error.contains(reason),
                &format!("wrong actual receipt {case} predicate: {error}"),
            )?;
            proof(
                &format!("actual receipt {case}"),
                &json!({"path":path,"receipt":receipt_path,"raw_sha256":digest(&bytes)}),
                &error,
            );
        }
    }
    let mut raw_mismatch = retained.clone();
    for execution in &mut raw_mismatch.report.executions {
        execution.record.arguments.push("receiver-mutant".into());
        execution.record_sha256 = execution.record.hash()?;
    }
    raw_mismatch.report.record_sha256 =
        Some(raw_mismatch.report.executions[0].record_sha256.clone());
    let error = result::validate(&raw_mismatch).unwrap_err();
    require(
        error.contains("raw artifact record differs"),
        "wrong actual record comparison predicate",
    )?;
    proof(
        "actual Result coherent raw record mismatch",
        &serde_json::to_value(&raw_mismatch).map_err(|e| e.to_string())?,
        &error,
    );
    for case in ["limits", "implementation", "inputs"] {
        if cluster && case == "inputs" {
            continue;
        }
        let mut changed = retained.clone();
        for execution in &mut changed.report.executions {
            match case {
                "limits" => execution.record.limits.evidence_bytes += 1,
                "implementation" => {
                    execution.record.identity.implementation = "f".repeat(64);
                    execution.record.execution_identity = execution.record.identity.digest()?;
                }
                _ => execution.record.arguments.push("receiver-mutant".into()),
            }
            execution.record_sha256 = execution.record.hash()?;
            let original = Path::new(
                execution
                    .artifact_path
                    .as_deref()
                    .ok_or("actual artifact absent")?,
            );
            let opened = store::Opened::open(original)?;
            let destination = root.join(format!("mutant-{case}-{}", execution.iteration));
            fs::create_dir(&destination).map_err(|e| e.to_string())?;
            fs::set_permissions(&destination, fs::Permissions::from_mode(0o700))
                .map_err(|e| e.to_string())?;
            let mut manifest = opened.manifest().clone();
            manifest.record = execution.record.clone();
            manifest.record_sha256 = execution.record_sha256.clone();
            for (name, entry) in &mut manifest.files {
                let bytes = if name == "record.json" {
                    let bytes = execution.record.bytes()?;
                    entry.bytes = bytes.len() as u64;
                    entry.sha256 = digest(&bytes);
                    bytes
                } else {
                    opened.payload(name, entry.bytes)?
                };
                let target = destination.join(name);
                if let Some(parent) = target.parent() {
                    fs::create_dir_all(parent).map_err(|e| e.to_string())?;
                    let mut current = Some(parent);
                    while let Some(directory) =
                        current.filter(|path| path.starts_with(&destination))
                    {
                        fs::set_permissions(directory, fs::Permissions::from_mode(0o700))
                            .map_err(|e| e.to_string())?;
                        current = directory.parent();
                    }
                }
                private(&target, &bytes);
            }
            private(
                &destination.join("storage.json"),
                &custody::read_file(&original.join("storage.json"), 1 << 20)?,
            );
            private(&destination.join("manifest.json"), &canonical(&manifest)?);
            execution.artifact_path = Some(destination.to_string_lossy().into_owned());
        }
        changed.report.record_sha256 = Some(changed.report.executions[0].record_sha256.clone());
        let error = result::validate(&changed).unwrap_err();
        require(
            error.contains("execution identity differs"),
            "wrong coherently resealed actual Result input predicate",
        )?;
        proof(
            &format!("actual Result resealed {case}"),
            &serde_json::to_value(&changed).map_err(|e| e.to_string())?,
            &error,
        );
    }
    if cluster {
        let record = &retained.report.executions[0].record;
        RetainedSession::stream_owners(record)?;
        for name in [
            "evidence/cluster/a",
            "evidence/cluster/a/1",
            "evidence/cluster//1/transcript",
        ] {
            let mut changed = record.clone();
            let key = changed
                .streams
                .keys()
                .find(|name| name.starts_with("evidence/cluster/"))
                .ok_or("actual cluster stream absent")?
                .clone();
            let stream = changed
                .streams
                .remove(&key)
                .ok_or("actual cluster stream absent")?;
            changed.streams.insert(name.into(), stream);
            let error = RetainedSession::stream_owners(&changed).unwrap_err();
            require(
                error.contains("malformed cluster stream owner"),
                "wrong cluster owner predicate",
            )?;
            proof(
                "actual cluster malformed stream owner",
                &serde_json::to_value(&changed).map_err(|e| e.to_string())?,
                &error,
            );
        }
    }
    if cluster {
        let mut changed = retained.clone();
        changed.observations.samples[0].incarnation = Some("foreign/1".into());
        let error = result::validate(&changed).unwrap_err();
        require(
            error.contains("record's incarnations"),
            "wrong borrowed cluster handle predicate",
        )?;
        proof(
            "actual Result borrowed cluster incarnation",
            &serde_json::to_value(&changed).map_err(|e| e.to_string())?,
            &error,
        );
        let handle = retained.observations.samples[0].incarnation.clone();
        let mut changed = retained.clone();
        changed.observations.samples.retain(|sample| {
            !(sample.phase == "execute" && sample.iteration == 1 && sample.incarnation == handle)
        });
        let error = result::validate(&changed).unwrap_err();
        require(
            error.contains("record's incarnations"),
            "wrong missing cluster handle predicate",
        )?;
        proof(
            "actual Result omitted cluster incarnation",
            &serde_json::to_value(&changed).map_err(|e| e.to_string())?,
            &error,
        );
    }
    for phase in ["execute", "replay"] {
        let mut changed = retained.clone();
        changed
            .observations
            .samples
            .retain(|sample| sample.phase != phase);
        let error = RetainedSession::from_value(configured.clone())?
            .final_contract(&changed)
            .unwrap_err();
        require(
            error.contains("sample coverage"),
            "wrong final sample coverage predicate",
        )?;
        proof(
            &format!("missing {phase} store samples"),
            &serde_json::to_value(&changed).map_err(|e| e.to_string())?,
            &error,
        );
    }
    println!(
        "B04_RESULT_ACTUAL {}",
        json!({"path":published.path,"config":configured,"row":row,"result":retained,"receipt":receipt,"publication":terminal,"control":"normal-return"})
    );
    Ok(())
}

fn genuine_identity(root: &Path, runner: &mut crate::cycle::process::Runner) -> Result<(), String> {
    let log = PathBuf::from(std::env::var("B04_IDENTITY_LOG").map_err(|e| e.to_string())?);
    let context = PathBuf::from(std::env::var("B04_IDENTITY_CONTEXT").map_err(|e| e.to_string())?);
    let raw = custody::read_file(&log, 32 << 20)?;
    let context = custody::strict_json(&custody::read_file(&context, 1 << 20)?)?;
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let snapshot = custody::Snapshot::observe(
        &repo,
        runner,
        &std::sync::atomic::AtomicBool::new(false),
        "b04-actual-preparation",
    )?;
    let config = json!({"root":root,"repo":repo,"files":snapshot.files});
    let session = RetainedSession::from_value(config)?;
    let value = session.core_identity(&raw, &context)?;
    require(value["case_count"] == 18, "genuine identity case coverage")?;
    let rows = std::str::from_utf8(&raw)
        .map_err(|e| e.to_string())?
        .lines()
        .filter_map(|line| line.split_once("GOMAD_CORE_OBSERVATION ").map(|(_, v)| v))
        .map(|v| custody::strict_json(v.as_bytes()))
        .collect::<Result<Vec<_>, _>>()?;
    for (name, index, field, value, reason) in [
        (
            "compiled digest",
            0,
            "observed",
            json!({"compiled":"f".repeat(64),"recomputed":"f".repeat(64)}),
            "compiled identity",
        ),
        (
            "unknown case",
            0,
            "case",
            json!("identity.foreign"),
            "unknown/duplicate",
        ),
        (
            "foreign schema",
            0,
            "schema",
            json!("foreign"),
            "unknown/duplicate",
        ),
    ] {
        let mut changed = rows.clone();
        changed[index][field] = value;
        let bytes = changed
            .iter()
            .map(|row| {
                format!(
                    "GOMAD_CORE_OBSERVATION {}\n",
                    String::from_utf8(canonical(row).unwrap()).unwrap()
                )
            })
            .collect::<String>();
        let error = session
            .core_identity(bytes.as_bytes(), &context)
            .unwrap_err();
        require(error.contains(reason), "wrong identity mutation predicate")?;
        proof(name, &json!(changed), &error);
    }
    for (name, index, pointer, value, reason) in [
        (
            "identity bindings",
            0,
            "/inputs/bindings/PROFILE",
            json!("debug"),
            "compiled context",
        ),
        (
            "identity compiler",
            0,
            "/inputs/compiler_utf8",
            json!("foreign"),
            "compiled context",
        ),
        (
            "identity owners",
            0,
            "/inputs/required_owners",
            json!([]),
            "compiled context",
        ),
        (
            "identity typed bytes",
            0,
            "/inputs/files/0/bytes",
            json!(true),
            "file closure",
        ),
        (
            "identity file digest",
            0,
            "/inputs/files/0/sha256",
            json!("f".repeat(64)),
            "file closure",
        ),
        (
            "identity file path",
            0,
            "/inputs/files/0/path",
            json!("/foreign"),
            "file closure",
        ),
        (
            "identity empty closure",
            1,
            "/observed/error",
            json!("accepted"),
            "empty identity closure",
        ),
        (
            "identity perturbation",
            2,
            "/inputs/mutation_append_utf8",
            json!("foreign"),
            "perturbation",
        ),
        (
            "identity before",
            2,
            "/observed/before",
            json!("f".repeat(64)),
            "identity mutation",
        ),
        (
            "identity after",
            2,
            "/observed/after",
            json!("f".repeat(64)),
            "identity mutation",
        ),
        (
            "identity missing owner error",
            2,
            "/observed/missing_error",
            json!("accepted"),
            "identity mutation",
        ),
    ] {
        let mut changed = rows.clone();
        *changed[index]
            .pointer_mut(pointer)
            .ok_or("identity fixture pointer absent")? = value;
        let bytes = changed
            .iter()
            .map(|row| {
                format!(
                    "GOMAD_CORE_OBSERVATION {}\n",
                    String::from_utf8(canonical(row).unwrap()).unwrap()
                )
            })
            .collect::<String>();
        let error = session
            .core_identity(bytes.as_bytes(), &context)
            .unwrap_err();
        require(error.contains(reason), "wrong inner identity predicate")?;
        proof(name, &json!(changed), &error);
    }
    let mut changed = rows.clone();
    changed[0]["inputs"]["unexpected"] = json!(true);
    let bytes = changed
        .iter()
        .map(|row| {
            format!(
                "GOMAD_CORE_OBSERVATION {}\n",
                String::from_utf8(canonical(row).unwrap()).unwrap()
            )
        })
        .collect::<String>();
    let error = session
        .core_identity(bytes.as_bytes(), &context)
        .unwrap_err();
    require(
        error.contains("input fields"),
        "wrong identity fields predicate",
    )?;
    proof("identity input fields", &json!(changed), &error);
    let missing = rows
        .iter()
        .skip(1)
        .map(|row| {
            format!(
                "GOMAD_CORE_OBSERVATION {}\n",
                String::from_utf8(canonical(row).unwrap()).unwrap()
            )
        })
        .collect::<String>();
    let error = session
        .core_identity(missing.as_bytes(), &context)
        .unwrap_err();
    require(
        error.contains("eighteen"),
        "wrong identity missing predicate",
    )?;
    proof("identity missing row", &json!(missing), &error);
    let mut wrong = context.clone();
    wrong["bindings"]["PROFILE"] = json!("debug");
    let error = session.core_identity(&raw, &wrong).unwrap_err();
    require(
        error.contains("compiled context"),
        "wrong identity context predicate",
    )?;
    proof("identity compiled context", &wrong, &error);
    println!(
        "B04_IDENTITY_ACTUAL {}",
        json!({"control":"normal-return","log":log,"raw_sha256":digest(&raw),"context":context,"projection":value})
    );
    Ok(())
}

#[test]
fn common_observation_and_module_metadata_refuse_unadmitted_raw_bytes() {
    assert_eq!(
        observation(b"observation {\"value\":1}\n", "observation ").unwrap(),
        json!({"value":1})
    );
    for (name, raw) in [
        ("missing", b"other {}".to_vec()),
        ("duplicate", b"observation {}\nobservation {}".to_vec()),
        ("empty", b"observation ".to_vec()),
        (
            "oversize",
            [b"observation ".as_slice(), &vec![b' '; (1 << 20) + 1]].concat(),
        ),
        (
            "duplicate-key",
            b"observation {\"value\":1,\"value\":2}".to_vec(),
        ),
        ("trailing", b"observation {}{}".to_vec()),
        ("nonfinite", b"observation {\"value\":NaN}".to_vec()),
    ] {
        let error = observation(&raw, "observation ").unwrap_err();
        proof(name, &json!({"raw":String::from_utf8_lossy(&raw)}), &error);
    }
    let module = b"\0asm\x01\0\0\0\x05\x03\x01\0\x01";
    assert_eq!(
        gomad_wasmhost::linked::module_metadata(module)
            .unwrap()
            .initial_memory_pages,
        1
    );
    let section = |tag: u8, payload: &[u8]| {
        let mut raw = b"\0asm\x01\0\0\0".to_vec();
        raw.extend([tag, u8::try_from(payload.len()).unwrap()]);
        raw.extend(payload);
        raw
    };
    let mut import = section(1, &[1, 0x60, 0, 0]);
    import.extend([2, 7, 1, 1, b'm', 1, b'f', 0, 1]);
    let mut duplicate = module.to_vec();
    duplicate.extend(&module[8..]);
    for (name, bytes) in [
        (
            "integer-overflow",
            [
                b"\0asm\x01\0\0\0\x05".as_slice(),
                &[0xff, 0xff, 0xff, 0xff, 0x1f],
            ]
            .concat(),
        ),
        (
            "integer-oversize",
            [b"\0asm\x01\0\0\0\x05".as_slice(), &[0x80; 6]].concat(),
        ),
        ("integer-truncated", b"\0asm\x01\0\0\0\x05\x80".to_vec()),
        ("import-text-truncated", section(2, &[1, 20, b'm'])),
        ("section-truncated", b"\0asm\x01\0\0\0\x05\x08\x01".to_vec()),
        ("metadata-duplicate", duplicate),
        ("type-unsupported", section(1, &[1, 0x61, 0, 0])),
        ("value-unsupported", section(1, &[1, 0x60, 1, 0x70, 0])),
        (
            "import-nonfunction",
            section(2, &[1, 1, b'm', 1, b'f', 2, 0, 1]),
        ),
        ("import-type-invalid", import),
        ("memory-count", section(5, &[2, 0, 1, 0, 1])),
        ("memory-flags", section(5, &[1, 4, 1])),
        ("memory-shared", section(5, &[1, 3, 1, 1])),
        ("memory-maximum", section(5, &[1, 1, 2, 1])),
        ("memory-zero", section(5, &[1, 0, 0])),
        ("memory-upper", section(5, &[1, 0, 0x81, 0x80, 4])),
        ("memory-missing", b"\0asm\x01\0\0\0".to_vec()),
        ("metadata-trailing", section(5, &[1, 0, 1, 0])),
        ("truncated", module[..12].to_vec()),
        ("trailing", [module.as_slice(), b"bad"].concat()),
        ("invalid-header", b"not wasm".to_vec()),
    ] {
        let error = gomad_wasmhost::linked::module_metadata(&bytes)
            .err()
            .unwrap();
        proof(name, &json!({"module":bytes}), &error.to_string());
    }
}

fn genuine_command_preparation(
    root: &Path,
    registry: &Path,
    runner: &mut crate::cycle::process::Runner,
) -> Result<(), String> {
    use crate::cycle::process::{Command, Environment};
    use std::{sync::atomic::AtomicBool, time::Duration};
    let producer = root.join("producer");
    fs::create_dir(&producer).map_err(|e| e.to_string())?;
    fs::set_permissions(&producer, fs::Permissions::from_mode(0o700)).map_err(|e| e.to_string())?;
    let executable = std::env::current_exe().map_err(|e| e.to_string())?;
    let argv = vec![
        executable.to_string_lossy().into_owned(),
        "--installation".into(),
        producer
            .join("installation.json")
            .to_string_lossy()
            .into_owned(),
        "prepare".into(),
    ];
    let mut environment = std::env::vars()
        .filter(|(key, _)| {
            !key.starts_with("GIT_") && !key.starts_with("SANDBOX_") && key != "GOROOT"
        })
        .collect::<BTreeMap<_, _>>();
    environment.insert("B04_CHILD".into(), "prepare".into());
    environment.insert(
        "B04_COMMAND_ROOT".into(),
        producer.to_string_lossy().into_owned(),
    );
    let command = Command {
        argv: argv.clone(),
        cwd: producer.clone(),
        environment: Environment::Exact(environment.clone()),
        declared_environment: environment,
        temporary_paths: BTreeMap::new(),
        accepted: vec![0],
        timeout: Duration::from_secs(600),
        private_context: Some("b04-actual-preparation".into()),
    };
    let sent = epoch() / 1_000_000_000;
    let cancelled = AtomicBool::new(false);
    let mut observed = None;
    let completed = runner
        .start("prepare", command.clone(), &cancelled, |_| Ok(()))
        .map_err(|e| e.to_string())?
        .wait_observed(&cancelled, |_, _, streams| {
            let raw = custody::read_file(&streams.stdout, 32 << 20)?;
            let row = std::str::from_utf8(&raw)
                .map_err(|e| e.to_string())?
                .lines()
                .find_map(|line| line.strip_prefix("B04_PREPARATION_ACTUAL "))
                .ok_or("actual child preparation output missing")?;
            let row = custody::strict_json(row.as_bytes())?;
            let seal = row["row"]["private_observations"].clone();
            observed = Some(row);
            Ok(Some(seal))
        })?;
    require(
        completed.accepted(),
        "actual child preparation command failed",
    )?;
    let observed = observed.ok_or("actual child preparation unobserved")?;
    let mut config = observed["config"].clone();
    config["root"] = json!(root);
    config["repo"] = json!(producer);
    config["sent_epoch"] = json!(sent);
    let mut session = RetainedSession::from_value(config.clone())?;
    let command = session.command(registry, "prepare", &command, true)?;
    require(
        command.row["pid"] == observed["row"]["pid"],
        "actual child PID preparation binding changed",
    )?;
    let campaign = PathBuf::from(text(
        &observed["row"]["private_observations"]["campaigns"][0],
        "path",
    )?);
    genuine_inspection(root, registry, runner, &config, &campaign)?;
    println!(
        "B04_PREPARATION_COMMAND_ACTUAL {}",
        json!({"control":"normal-return","pid":completed.pid,"row":command.row,"stdout_sha256":digest(&command.stdout),"stderr_sha256":digest(&command.stderr)})
    );
    Ok(())
}

fn preparation_guards(
    config: &Value,
    row: &Value,
    receipt: &Value,
    campaign: &Path,
) -> Result<(), String> {
    let empty = RetainedSession::from_value(config.clone())?;
    let error = empty
        .prepared_observation(campaign, &receipt["provenance"])
        .unwrap_err();
    require(
        error.contains("lacks actual command observation seal"),
        "wrong missing preparation seal predicate",
    )?;
    proof(
        "actual preparation missing outer observation",
        &json!({"campaign":campaign}),
        &error,
    );
    let sidecar = campaign.join("preparation-observations");
    let receipt_path = sidecar.join("receipt.json");
    let original = custody::read_file(&receipt_path, 16 << 20)?;
    for (pointer, value, reason) in [
        (
            "/private_observations/campaigns/0/receipt_path",
            json!(sidecar.join("foreign.json")),
            "another campaign",
        ),
        (
            "/private_observations/campaigns/0/receipt_sha256",
            json!("f".repeat(64)),
            "actual command seal",
        ),
        (
            "/private_observations/campaigns/0/files",
            json!({}),
            "outer-sealed",
        ),
    ] {
        let mut changed = row.clone();
        *changed
            .pointer_mut(pointer)
            .ok_or("missing genuine seal field")? = value;
        let error = RetainedSession::from_value(config.clone())?
            .capture_preparations(&changed)
            .unwrap_err();
        require(
            error.contains(reason),
            "wrong outer preparation field predicate",
        )?;
        proof(pointer, &changed, &error);
    }
    let mut duplicate = row.clone();
    let campaign_row = duplicate["private_observations"]["campaigns"][0].clone();
    duplicate["private_observations"]["campaigns"]
        .as_array_mut()
        .unwrap()
        .push(campaign_row);
    let error = RetainedSession::from_value(config.clone())?
        .capture_preparations(&duplicate)
        .unwrap_err();
    require(
        error.contains("campaign set"),
        "wrong duplicate campaign predicate",
    )?;
    proof("duplicate campaigns", &duplicate, &error);
    let mut implicit = row.clone();
    implicit["argv"] = json!([config["binary"], "prepare"]);
    let error = RetainedSession::from_value(config.clone())?
        .capture_preparations(&implicit)
        .unwrap_err();
    require(
        error.contains("argument"),
        "wrong implicit installation predicate",
    )?;
    proof("implicit installation", &implicit, &error);
    for (field, value, reason) in [
        ("owner_pid", json!(0), "owner mismatch"),
        ("host_sha256", json!("f".repeat(64)), "owner mismatch"),
        ("helper_sha256", json!("f".repeat(64)), "owner mismatch"),
        ("files", json!({}), "file table"),
    ] {
        let mut changed = receipt.clone();
        changed[field] = value;
        let bytes = canonical(&changed)?;
        private(&receipt_path, &bytes);
        let mut outer = row.clone();
        outer["private_observations"]["campaigns"][0]["receipt_sha256"] = json!(digest(&bytes));
        let error = RetainedSession::from_value(config.clone())?
            .capture_preparations(&outer)
            .unwrap_err();
        require(
            error.contains(reason),
            "wrong resealed preparation field predicate",
        )?;
        proof(field, &json!({"row":outer,"receipt":changed}), &error);
        private(&receipt_path, &original);
    }
    let first = receipt["files"]
        .as_object()
        .ok_or("actual sidecar table absent")?
        .keys()
        .next()
        .ok_or("actual sidecar file absent")?
        .clone();
    for case in ["oversized", "unsafe", "unknown"] {
        let mut changed = receipt.clone();
        match case {
            "oversized" => changed["files"][&first]["bytes"] = json!((128_u64 << 20) + 1),
            "unsafe" => {
                let entry = changed["files"]
                    .as_object_mut()
                    .ok_or("actual sidecar table absent")?
                    .remove(&first)
                    .ok_or("actual sidecar file absent")?;
                changed["files"]["target/preparation-inputs/../foreign"] = entry;
            }
            _ => changed["files"][&first]["unexpected"] = json!(true),
        }
        let bytes = canonical(&changed)?;
        private(&receipt_path, &bytes);
        let mut outer = row.clone();
        outer["private_observations"]["campaigns"][0]["receipt_sha256"] = json!(digest(&bytes));
        let refused = RetainedSession::from_value(config.clone())?
            .capture_preparations(&outer)
            .err();
        private(&receipt_path, &original);
        let error = refused.ok_or("actual sidecar table mutation accepted")?;
        require(
            error.contains(match case {
                "oversized" => "oversized",
                "unsafe" => "unsafe payload",
                _ => "unknown field",
            }),
            &format!("wrong sidecar table {case} predicate: {error}"),
        )?;
        proof(
            &format!("actual sidecar table {case}"),
            &json!({"receipt":changed,"row":outer}),
            &error,
        );
    }
    let blob = sidecar.join(&first);
    let bytes = custody::read_file(&blob, 128 << 20)?;
    for case in ["mode", "link", "hash"] {
        let alias = campaign.join("sidecar-link");
        match case {
            "mode" => fs::set_permissions(&blob, fs::Permissions::from_mode(0o644))
                .map_err(|e| e.to_string())?,
            "link" => fs::hard_link(&blob, &alias).map_err(|e| e.to_string())?,
            _ => {
                let mut changed = bytes.clone();
                if let Some(first) = changed.first_mut() {
                    *first ^= 1;
                } else {
                    changed.push(1);
                }
                private(&blob, &changed);
            }
        }
        let refused = RetainedSession::from_value(config.clone())?
            .capture_preparations(row)
            .err();
        if case == "link" {
            fs::remove_file(alias).map_err(|e| e.to_string())?;
        }
        private(&blob, &bytes);
        let error = refused.ok_or("actual sidecar file mutation accepted")?;
        require(
            error.contains(match case {
                "hash" => "sidecar file changed",
                "link" => "hard link",
                _ => "private",
            }),
            &format!("wrong sidecar file {case} predicate: {error}"),
        )?;
        proof(
            &format!("actual sidecar file {case}"),
            &json!({"path":blob,"original_sha256":digest(&bytes),"case":case}),
            &error,
        );
    }
    let extra = sidecar.join("unsealed");
    private(&extra, b"unexpected");
    let error = RetainedSession::from_value(config.clone())?
        .capture_preparations(row)
        .unwrap_err();
    proof(
        "unsealed sidecar entry",
        &json!({"path":extra,"bytes":"unexpected"}),
        &error,
    );
    fs::remove_file(extra).map_err(|e| e.to_string())?;
    let plan_path = campaign.join("plan.json");
    let plan_raw = custody::read_file(&plan_path, 16 << 20)?;
    let mut plan = custody::strict_json(&plan_raw)?;
    plan["prepared_artifact"] = json!("../foreign");
    private(&plan_path, &canonical(&plan)?);
    let error = RetainedSession::from_value(config.clone())?
        .capture_preparations(row)
        .unwrap_err();
    proof("prepared path traversal", &plan, &error);
    private(&plan_path, &plan_raw);
    let plan = custody::strict_json(&plan_raw)?;
    let prepared = campaign.join(text(&plan, "prepared_artifact")?);
    let replay_path = prepared.join("evidence/replay.json");
    let replay_raw = custody::read_file(&replay_path, 16 << 20)?;
    let replay = custody::strict_json(&replay_raw)?;
    let artifact = RetainedSession::from_value(config.clone())?.artifact(&prepared)?;
    genuine_artifact_guards(config, &prepared, &artifact.manifest)?;
    let provenance = custody::strict_json(&artifact.payloads["target/provenance.json"])?;
    let session = RetainedSession::from_value(config.clone())?;
    session.prepared_replay(&replay, &provenance)?;
    session.prepared_record(
        &artifact.manifest.record,
        &provenance["configuration"],
        text(&provenance, "module_sha256")?,
        text(&provenance, "build_key")?,
    )?;
    for (pointer, value) in [
        ("/seed", json!(1)),
        ("/virtual_time_nanos", json!(1)),
        ("/outcome/class", json!("success")),
        ("/outcome/termination", json!("exit")),
        ("/outcome/exit_code", json!(0)),
        ("/outcome/reason", json!("foreign")),
        ("/profile", json!("foreign")),
        ("/arguments", json!(["foreign"])),
        ("/environment", json!({"FOREIGN":"1"})),
        ("/limits/output_bytes", json!(1)),
        ("/identity/target", json!("f".repeat(64))),
    ] {
        let mut changed =
            serde_json::to_value(&artifact.manifest.record).map_err(|e| e.to_string())?;
        *changed
            .pointer_mut(pointer)
            .ok_or("actual prepared record field absent")? = value;
        let typed = serde_json::from_value(changed.clone()).map_err(|e| e.to_string())?;
        let error = session
            .prepared_record(
                &typed,
                &provenance["configuration"],
                text(&provenance, "module_sha256")?,
                text(&provenance, "build_key")?,
            )
            .unwrap_err();
        require(
            error.contains("prepared record role/"),
            &format!("wrong actual prepared record predicate: {error}"),
        )?;
        proof(
            &format!("actual admitted prepared record {pointer}"),
            &changed,
            &error,
        );
    }

    for field in [
        "schema",
        "helper_sha256",
        "host_sha256",
        "toolchain_sha256",
        "runtime_implementation_sha256",
        "configuration",
    ] {
        let mut changed = replay.clone();
        changed[field] = json!("receiver-owned mutation");
        let error = session.prepared_replay(&changed, &provenance).unwrap_err();
        require(
            error == "prepared replay preimage changed",
            "wrong prepared preimage predicate",
        )?;
        proof(
            &format!("admitted prepared replay {field}"),
            &json!({"replay":changed,"provenance":provenance}),
            &error,
        );
    }
    RetainedSession::from_value(config.clone())?.capture_preparations(row)?;
    let mut observed = RetainedSession::from_value(config.clone())?;
    observed.capture_preparations(row)?;
    observed.authenticate_prepared_artifact(campaign, &provenance)?;
    let mut conflicting = config.clone();
    let source = provenance["inputs"]["sources"]
        .as_object()
        .ok_or("actual source inventory absent")?
        .keys()
        .find(|path| path.ends_with("/main.go"))
        .ok_or("actual source main.go absent")?;
    conflicting["files"][source] = json!("f".repeat(64));
    let mut conflict_session = RetainedSession::from_value(conflicting)?;
    conflict_session.capture_preparations(row)?;
    let error = conflict_session
        .authenticate_prepared_artifact(campaign, &provenance)
        .unwrap_err();
    require(
        error.contains("source/compiler membership"),
        &format!("wrong frozen source conflict predicate: {error}"),
    )?;
    proof(
        "actual preparation conflicting frozen source pin",
        &json!({"source":source,"original":provenance["inputs"]["sources"][source],"conflicting":"f".repeat(64)}),
        &error,
    );

    let mut wrong = provenance.clone();
    wrong["build_key"] = json!("f".repeat(64));
    let error = observed.prepared_observation(campaign, &wrong).unwrap_err();
    require(
        error.contains("different prepared provenance"),
        "wrong observed provenance predicate",
    )?;
    proof("actual borrowed prepared provenance", &wrong, &error);
    let installation_arg = argument(
        &serde_json::from_value::<Vec<String>>(row["argv"].clone()).map_err(|e| e.to_string())?,
        "--installation",
    )?
    .to_owned();
    let mut runtime = row.clone();
    runtime["argv"] = json!([
        config["binary"],
        "produce",
        installation_arg,
        "fixture",
        "0",
        campaign.parent().ok_or("campaign parent missing")?
    ]);
    let error = RetainedSession::from_value(config.clone())?
        .capture_preparations(&runtime)
        .unwrap_err();
    require(
        error.contains("not frozen release harness"),
        "wrong runtime harness substitution predicate",
    )?;
    proof("actual runtime foreign harness", &runtime, &error);
    let mut runtime_config = config.clone();
    runtime_config["stages"] = json!({"runtime":{"harness":config["binary"]}});
    let error = RetainedSession::from_value(runtime_config)?
        .capture_preparations(&runtime)
        .unwrap_err();
    require(
        error.contains("runtime preparation root changed"),
        "wrong runtime campaign path predicate",
    )?;
    proof("actual runtime campaign path", &runtime, &error);
    observed
        .preparations
        .get_mut(campaign)
        .ok_or("observed campaign missing")?
        .owner += 1;
    let error = observed.capture_preparations(row).unwrap_err();
    require(
        error.contains("conflicting producer"),
        "wrong conflicting observation predicate",
    )?;
    proof("actual conflicting preparation cache", row, &error);
    let installation_path = Path::new(argument(
        &serde_json::from_value::<Vec<String>>(row["argv"].clone()).map_err(|e| e.to_string())?,
        "--installation",
    )?)
    .to_owned();
    let installation = custody::strict_json(&custody::read_file(&installation_path, 16 << 20)?)?;
    let mut wrong_config = config.clone();
    let pin = installation["admission"]["compiler_inputs"]
        .as_object()
        .ok_or("compiler inventory missing")?
        .keys()
        .find(|name| name.as_str() != text(&provenance["inputs"], "compiler_path").unwrap())
        .ok_or("compiler auxiliary pin missing")?;
    wrong_config["files"][pin] = json!("f".repeat(64));
    let mut session = RetainedSession::from_value(wrong_config.clone())?;
    session.capture_preparations(row)?;
    let error = session
        .authenticate_prepared_artifact(campaign, &provenance)
        .unwrap_err();
    require(
        error.contains("complete compiler inventory differs"),
        "wrong installation inventory predicate",
    )?;
    proof("actual installation compiler pin", &wrong_config, &error);
    let original = custody::read_file(&prepared.join("target/provenance.json"), 16 << 20)?;
    let session = RetainedSession::from_value(config.clone())?;
    let changed = session.with_artifact(&prepared, |opened| {
        let bytes = opened.payload("target/provenance.json", 16 << 20)?;
        private(
            &prepared.join("target/provenance.json"),
            b"receiver changed after callback read",
        );
        Ok(bytes)
    });
    private(&prepared.join("target/provenance.json"), &original);
    let error = changed.unwrap_err();
    require(
        error.contains("payload"),
        "wrong callback payload reauthentication predicate",
    )?;
    proof(
        "artifact callback payload replacement",
        &json!({"path":prepared,"before_sha256":digest(&original)}),
        &error,
    );
    let moved = prepared.with_extension("callback-moved");
    let changed = session.with_artifact(&prepared, |_| {
        fs::rename(&prepared, &moved).map_err(|e| e.to_string())?;
        fs::create_dir(&prepared).map_err(|e| e.to_string())?;
        fs::set_permissions(&prepared, fs::Permissions::from_mode(0o700))
            .map_err(|e| e.to_string())?;
        Ok(())
    });
    fs::remove_dir(&prepared).map_err(|e| e.to_string())?;
    fs::rename(&moved, &prepared).map_err(|e| e.to_string())?;
    let error = changed.unwrap_err();
    require(
        error.contains("directory identity changed"),
        "wrong callback directory identity predicate",
    )?;
    proof(
        "artifact callback named directory replacement",
        &json!({"path":prepared}),
        &error,
    );
    session.artifact(&prepared)?;
    Ok(())
}

fn genuine_artifact_guards(
    config: &Value,
    prepared: &Path,
    manifest: &store::Manifest,
) -> Result<(), String> {
    let session = RetainedSession::from_value(config.clone())?;
    for bound in [None, Some((512_u64 << 20) + 1)] {
        let path = prepared
            .parent()
            .ok_or("prepared parent missing")?
            .join(if bound.is_some() {
                "oversized-module"
            } else {
                "missing-module"
            });
        fs::create_dir(&path).map_err(|e| e.to_string())?;
        fs::set_permissions(&path, fs::Permissions::from_mode(0o700)).map_err(|e| e.to_string())?;
        let mut changed = manifest.clone();
        changed.record.streams.clear();
        changed.record_sha256 = changed.record.hash()?;
        changed.files.clear();
        if let Some(bytes) = bound {
            let mut module = manifest.files["target/module.wasm"].clone();
            module.bytes = bytes;
            changed.files.insert("target/module.wasm".into(), module);
            let target = path.join("target");
            fs::create_dir(&target).map_err(|e| e.to_string())?;
            fs::set_permissions(&target, fs::Permissions::from_mode(0o700))
                .map_err(|e| e.to_string())?;
            let module_path = target.join("module.wasm");
            private(&module_path, b"");
            fs::OpenOptions::new()
                .write(true)
                .open(&module_path)
                .map_err(|e| e.to_string())?
                .set_len(bytes)
                .map_err(|e| e.to_string())?;
        }
        private(&path.join("manifest.json"), &canonical(&changed)?);
        private(
            &path.join("storage.json"),
            &custody::read_file(&prepared.join("storage.json"), 1 << 20)?,
        );
        let error = session
            .artifact(&path)
            .err()
            .ok_or("module cardinality/bound ignored")?;
        require(
            error.contains(if bound.is_some() {
                "global bound"
            } else {
                "exactly one"
            }),
            &format!("wrong actual module cardinality/bound predicate: {error}"),
        )?;
        proof(
            if bound.is_some() {
                "actual module pre-read512MiB bound"
            } else {
                "actual module cardinality"
            },
            &json!({"path":path,"manifest":changed,"sparse_zero_bytes":bound}),
            &error,
        );
        fs::remove_dir_all(&path).map_err(|e| e.to_string())?;
    }
    session.artifact(prepared)?;
    Ok(())
}

fn genuine_inspection(
    root: &Path,
    registry: &Path,
    runner: &mut crate::cycle::process::Runner,
    config: &Value,
    campaign: &Path,
) -> Result<(), String> {
    use crate::cycle::process::{Command, Environment};
    let mut session = RetainedSession::from_value(config.clone())?;
    let plan = session.json(&campaign.join("plan.json"), 16 << 20)?;
    let artifact = campaign.join(text(&plan, "prepared_artifact")?);
    let mut environment = std::env::vars()
        .filter(|(k, _)| !k.starts_with("GIT_") && !k.starts_with("SANDBOX_") && k != "GOROOT")
        .collect::<BTreeMap<_, _>>();
    environment.insert("B04_CHILD".into(), "inspect".into());
    let expected = Command {
        argv: vec![
            text(config, "binary")?.into(),
            "inspect".into(),
            "--json".into(),
            artifact.to_string_lossy().into_owned(),
        ],
        cwd: root.to_owned(),
        environment: Environment::Exact(environment.clone()),
        declared_environment: environment,
        temporary_paths: BTreeMap::new(),
        accepted: vec![0],
        timeout: std::time::Duration::from_secs(60),
        private_context: None,
    };
    let cancelled = std::sync::atomic::AtomicBool::new(false);
    let completed = runner
        .start("inspect-actual", expected.clone(), &cancelled, |_| Ok(()))
        .map_err(|e| e.to_string())?
        .wait(&cancelled)?;
    require(completed.accepted(), "actual public inspect failed")?;
    let admitted = session.command(registry, "inspect-actual", &expected, false)?;
    let projection = custody::strict_json(&admitted.stdout)?;
    for field in ["record", "files"] {
        let mut changed = projection.clone();
        changed["artifact"][field] = json!({});
        let bytes = canonical(&changed)?;
        let mut row = admitted.row.clone();
        row["stdout_sha256"] = json!(digest(&bytes));
        private(&registry.join("inspect-actual.stdout"), &bytes);
        for suffix in ["command.json", "process.json"] {
            private(
                &registry.join(format!("inspect-actual.{suffix}")),
                &canonical(&row)?,
            );
        }
        let error = session
            .command(registry, "inspect-actual", &expected, false)
            .err()
            .ok_or("public inspect mutant accepted")?;
        require(
            error.contains("public inspect"),
            "wrong resealed public inspect predicate",
        )?;
        proof(
            &format!("actual public inspect {field}"),
            &json!({"row":row,"reported":changed}),
            &error,
        );
    }
    private(&registry.join("inspect-actual.stdout"), &admitted.stdout);
    for suffix in ["command.json", "process.json"] {
        private(
            &registry.join(format!("inspect-actual.{suffix}")),
            &canonical(&admitted.row)?,
        );
    }
    session.command(registry, "inspect-actual", &expected, false)?;
    println!(
        "B04_INSPECT_ACTUAL {}",
        json!({"control":"normal-return","pid":completed.pid,"artifact":artifact,"stdout_sha256":digest(&admitted.stdout)})
    );
    Ok(())
}
