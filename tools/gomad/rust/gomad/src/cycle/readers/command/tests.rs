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
        b"package main\nfunc main(){println(\"b04-current-preparation\")}\n",
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
    let config = json!({"root":root,"repo":root,"files":frozen,"source_sha256":digest(&custody::read_file(&source.join("main.go"),1<<20)?),"binary_sha256":binary,"binary":executable,"identity":{"host":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),"helper":gomad_wasmhost::linked::IMPLEMENTATION_SHA256}});
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
    genuine_result(root, &source, &installation_path, &config)?;
    println!(
        "B04_PREPARATION_ACTUAL {}",
        json!({"control":"normal-return","guest_exit":execution.publication.record.outcome.exit_code,"row":row,"receipt":receipt,"provenance":provenance,"config":config})
    );
    Ok(())
}

fn genuine_result(
    root: &Path,
    source: &Path,
    installation: &Path,
    config: &Value,
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
    let specification = json!({"Manifest":{"ID":"b04-receiving-control","Workload":"genuine stock Go guest","IntegratedSourceSHA256":format!("sha256:{}",digest(manifest.as_bytes())),"IntegratedSourceManifestPath":manifest_path,"Seed":7,"Repeat":2},"Campaign":{"Target":{"Backend":"wasm","Kind":"go-run","Source":".","WorkingDir":source},"Environment":["GOMAXPROCS=1"],"Artifacts":root.join("result-campaigns"),"ExecutionTimeout":60_000_000_000_u64,"OverallTimeout":600_000_000_000_u64},"ArtifactRoot":root.join("results")});
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
        retained.report.qualified,
        "genuine Result guest/replay failed",
    )?;
    let published = cells::publish(&frozen, retained)?;
    let retained = result::open(&published.path)?;
    let (receipt, terminal) = result::open_receipt(&published.path, &retained)?;
    let reported = json!({"result":retained,"receipt":receipt,"publication":terminal});
    let mut configured = config.clone();
    configured["source_manifest"] = json!(manifest_path);
    configured["source_sha256"] = json!(digest(manifest.as_bytes()));
    let provenance = &retained.provenance;
    for group in ["compiler", "sources"] {
        for (path, sha) in provenance["inputs"][group]
            .as_object()
            .ok_or("genuine provenance inventory")?
        {
            configured["files"][path] = sha.clone();
        }
    }
    let mut session = RetainedSession::from_value(configured.clone())?;
    session.result(&published.path, &reported)?;
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
        ("/report/qualified", json!(false), "deterministic final"),
        ("/report/deterministic", json!(false), "deterministic final"),
        (
            "/report/target_success",
            json!(false),
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
    let mut session = RetainedSession::from_value(config)?;
    let command = session.command(registry, "prepare", &command, true)?;
    require(
        command.row["pid"] == observed["row"]["pid"],
        "actual child PID preparation binding changed",
    )?;
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
    let provenance = custody::strict_json(&artifact.payloads["target/provenance.json"])?;
    let session = RetainedSession::from_value(config.clone())?;
    session.prepared_replay(&replay, &provenance)?;
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
    Ok(())
}
