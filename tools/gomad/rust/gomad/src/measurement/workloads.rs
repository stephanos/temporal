use super::{
    build::{self, Role},
    evidence::{self, ComparisonMask, NativeRun, VariantRole},
    guest, host,
    inputs::{CheckoutProof, Invocation},
    module::{self, Selection},
    statistics,
};
use crate::evidence::canonical;
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    collections::BTreeMap,
    fs,
    io::{Read, Write},
    path::{Path, PathBuf},
    time::Duration,
};
const WORKFLOW: &str = "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$";
const ACTIVITY: &str = "^TestActivityTestSuite$";
const SEEDS: [u64; 3] = [0, 1000003, 2000006];
#[derive(Clone, Debug, Serialize)]
pub struct RunSpec {
    pub name: String,
    pub selection: String,
    pub seed: u64,
    pub role: String,
    pub prepare: bool,
    pub warm: bool,
    pub repetition: i32,
    pub variant: String,
    pub gc: String,
    pub sampling: bool,
    pub compare_to: Option<String>,
    pub replay_from: Option<String>,
}
impl RunSpec {
    pub(super) fn new(name: impl Into<String>, selection: &str, seed: u64, role: &str) -> Self {
        Self {
            name: name.into(),
            selection: selection.into(),
            seed,
            role: role.into(),
            prepare: false,
            warm: true,
            repetition: 0,
            variant: "baseline".into(),
            gc: String::new(),
            sampling: false,
            compare_to: None,
            replay_from: None,
        }
    }
}
fn value<'a>(call: &'a Invocation, key: &str) -> &'a str {
    call.inputs.get(key).map_or("", String::as_str)
}
fn repetitions(call: &Invocation, key: &str, default: usize) -> Result<usize, String> {
    let raw = value(call, key);
    let n = if raw.is_empty() {
        default
    } else {
        raw.parse().map_err(|_| "repetitions must be 1..20")?
    };
    if !(1..=20).contains(&n) {
        return Err("repetitions must be 1..20".into());
    }
    Ok(n)
}
fn prepare_row(name: &str, selection: &str, role: &str) -> RunSpec {
    let mut row = RunSpec::new(name, selection, 0, role);
    row.prepare = true;
    row.warm = false;
    row
}
fn selections() -> [(&'static str, &'static str); 2] {
    [("workflow", WORKFLOW), ("activity", ACTIVITY)]
}
fn pair(
    rows: &mut Vec<RunSpec>,
    label: &str,
    selection: &str,
    seed: u64,
    reference: &str,
    candidate: &str,
) {
    let before = format!("{label}-{reference}");
    rows.push(RunSpec::new(&before, selection, seed, reference));
    let mut after = RunSpec::new(format!("{label}-{candidate}"), selection, seed, candidate);
    after.compare_to = Some(before);
    rows.push(after);
}
/// The execution order, including preparations and failure-tail labels, is producer-specific.
pub fn plan(call: &Invocation) -> Result<Vec<RunSpec>, String> {
    let mut rows = Vec::new();
    match call.name.as_str() {
        "runtime-cost-profile" => {
            let mut first = RunSpec::new("reference", "runtime", 7, "ordinary");
            first.warm = false;
            rows.push(first);
            let mut second = RunSpec::new("candidate", "runtime", 7, "ordinary");
            second.warm = false;
            second.compare_to = Some("reference".into());
            rows.push(second);
        }
        "host-call-profile" => {
            rows.push(prepare_row("prepare", ACTIVITY, "diagnostic"));
            for seed in SEEDS {
                for repetition in
                    0..repetitions(call, "GOMAD_WASM_HOST_CALL_PROFILE_REPETITIONS", 3)?
                {
                    let name = format!("activity-seed{seed}-rep{repetition}");
                    let mut row = RunSpec::new(name, ACTIVITY, seed, "diagnostic");
                    row.repetition = repetition as i32;
                    if repetition > 0 {
                        row.compare_to = Some(format!("activity-seed{seed}-rep0"));
                    }
                    rows.push(row);
                }
            }
        }
        "host-owner-profile" => {
            let selection = match value(call, "GOMAD_WASM_HOST_OWNER_SELECTION") {
                "" | "activity" => ACTIVITY,
                "workflow" => WORKFLOW,
                other => {
                    return Err(format!(
                        "selection must be activity or workflow, got {other:?}"
                    ));
                }
            };
            let label = if selection == ACTIVITY {
                "activity"
            } else {
                "workflow"
            };
            for role in ["reference", "diagnostic"] {
                rows.push(prepare_row(&format!("{role}-prepare"), selection, role));
            }
            let seeds = if value(call, "GOMAD_WASM_HOST_OWNER_SEEDS").is_empty() {
                SEEDS.to_vec()
            } else {
                value(call, "GOMAD_WASM_HOST_OWNER_SEEDS")
                    .split(',')
                    .map(|v| v.trim().parse::<u64>().map_err(|e| e.to_string()))
                    .collect::<Result<Vec<_>, _>>()?
            };
            for seed in seeds {
                let reference = format!("{label}-seed{seed}-reference");
                rows.push(RunSpec::new(&reference, selection, seed, "reference"));
                for repetition in
                    -1..repetitions(call, "GOMAD_WASM_HOST_OWNER_REPETITIONS", 3)? as i32
                {
                    let mut row = RunSpec::new(
                        if repetition < 0 {
                            format!("{label}-seed{seed}-warmup")
                        } else {
                            format!("{label}-seed{seed}-rep{repetition}")
                        },
                        selection,
                        seed,
                        "diagnostic",
                    );
                    row.repetition = repetition;
                    row.compare_to = Some(reference.clone());
                    rows.push(row);
                }
            }
        }
        "guest-profile" => {
            rows.push(prepare_row("guest-profile-prepare", WORKFLOW, "guest"));
            for (selection, run) in selections() {
                for seed in SEEDS {
                    let label = format!("{selection}-seed{seed}");
                    let before = format!("{label}-unsampled");
                    rows.push(RunSpec::new(&before, run, seed, "guest"));
                    let mut after = RunSpec::new(format!("{label}-sampled"), run, seed, "guest");
                    after.sampling = true;
                    after.compare_to = Some(before);
                    rows.push(after);
                }
            }
        }
        "guest-variants" => {
            for (selection, run) in selections() {
                for variant in ["baseline", "gc200", "gc400", "gc-off", "gowasm", "fuel-off"] {
                    let mut row = prepare_row(
                        &format!("{selection}-{variant}-prepare"),
                        run,
                        if variant == "fuel-off" {
                            "fuel-off"
                        } else {
                            "diagnostic"
                        },
                    );
                    row.variant = variant.into();
                    row.gc = gc(variant).into();
                    rows.push(row);
                }
                for seed in SEEDS {
                    for repetition in 0..5 {
                        for slot in 0..6 {
                            let index = if slot == 0 {
                                0
                            } else {
                                1 + (slot - 1 + repetition) % 5
                            };
                            let variant =
                                ["baseline", "gc200", "gc400", "gc-off", "gowasm", "fuel-off"]
                                    [index];
                            let mut row = RunSpec::new(
                                format!("{selection}-{variant}-seed{seed}-rep{repetition}"),
                                run,
                                seed,
                                if variant == "fuel-off" {
                                    "fuel-off"
                                } else {
                                    "diagnostic"
                                },
                            );
                            row.variant = variant.into();
                            row.gc = gc(variant).into();
                            row.repetition = repetition as i32;
                            row.compare_to = Some(format!("{selection}-baseline-seed{seed}-rep0"));
                            if index == 0 && repetition == 0 {
                                row.compare_to = None;
                            }
                            rows.push(row);
                        }
                    }
                }
            }
        }
        "failing-suite-profile" => {
            for role in ["reference", "diagnostic"] {
                rows.push(prepare_row(&format!("{role}-prepare"), WORKFLOW, role));
            }
            for (name, run) in [
                ("workflow", WORKFLOW),
                ("child", "^TestChildWorkflowSuite$"),
                ("timers", "^TestUserTimersTestSuite$"),
            ] {
                for seed in [0, 1000003] {
                    pair(
                        &mut rows,
                        &format!("{name}-seed{seed}"),
                        run,
                        seed,
                        "reference",
                        "diagnostic",
                    );
                }
                let mut replay =
                    RunSpec::new(format!("{name}-diagnostic-replay"), run, 0, "diagnostic");
                replay.replay_from = Some(format!("{name}-seed0-reference"));
                rows.push(replay);
            }
        }
        "legacy-workflow-profile" => {
            for seed in [0, 1000003] {
                rows.push(RunSpec::new(
                    format!("seed{seed}"),
                    WORKFLOW,
                    seed,
                    "ordinary",
                ));
            }
            let mut retained = RunSpec::new("seed0-retained", WORKFLOW, 0, "ordinary");
            retained.compare_to = Some("seed0".into());
            rows.push(retained);
        }
        "scratch-smoke-profile" => {
            let helpers = value(call, "GOMAD_SCRATCH_HELPERS");
            if helpers.is_empty() {
                return Err("scratch helpers must name explicit native role executables".into());
            }
            for helper in helpers.split(',') {
                let (name, path) = helper
                    .split_once('=')
                    .ok_or("scratch helper requires name=absolute executable")?;
                if name.is_empty() || name.contains(['/', '\\']) || !Path::new(path).is_absolute() {
                    return Err("scratch helper requires safe name and absolute executable".into());
                }
                if value(call, "GOMAD_SCRATCH_SKIP_PREPARE").is_empty() {
                    rows.push(prepare_row(&format!("{name}-prepare"), WORKFLOW, name));
                }
                for (selection, run) in selections() {
                    let only = value(call, "GOMAD_SCRATCH_ONLY");
                    if !only.is_empty() && only != selection {
                        continue;
                    }
                    for seed in [0, 1000003] {
                        rows.push(RunSpec::new(
                            format!("{name}-{selection}-seed{seed}"),
                            run,
                            seed,
                            name,
                        ));
                    }
                }
            }
        }
        "matched-suite" => {
            let selected = value(call, "GOMAD_WASM_MATCHED_SUITE");
            for (name, run) in [
                ("workflow", WORKFLOW),
                ("activity", ACTIVITY),
                ("child", "^TestChildWorkflowSuite$"),
                ("update", "^TestWorkflowUpdateSuite$"),
                ("timers", "^TestUserTimersTestSuite$"),
                ("frontend", "^TestFrontendSystemInfo$"),
            ] {
                if (selected.is_empty() && name == "frontend")
                    || (!selected.is_empty()
                        && !selected.split(',').any(|value| value.trim() == name))
                {
                    continue;
                }
                for role in ["baseline", "candidate"] {
                    rows.push(prepare_row(&format!("{name}-prepare-{role}"), run, role));
                }
                for (sample, seed) in [0, 1000003, 2000006, 3000009].into_iter().enumerate() {
                    let roles = if sample % 2 == 0 {
                        ["baseline", "candidate"]
                    } else {
                        ["candidate", "baseline"]
                    };
                    for role in roles {
                        let mut row =
                            RunSpec::new(format!("{name}-seed{seed}-{role}"), run, seed, role);
                        if role == "candidate" {
                            row.compare_to = Some(format!("{name}-seed{seed}-baseline"));
                        }
                        rows.push(row);
                    }
                }
                let mut replay =
                    RunSpec::new(format!("{name}-candidate-replay"), run, 0, "candidate");
                replay.replay_from = Some(format!("{name}-seed0-baseline"));
                rows.push(replay);
            }
        }
        "migration" => {}
        _ => return Err("unknown single-guest measurement recipe".into()),
    }
    if rows.is_empty() && call.name != "migration" {
        return Err("measurement selection admitted no samples".into());
    }
    Ok(rows)
}
fn gc(variant: &str) -> &str {
    match variant {
        "gc200" => "200",
        "gc400" => "400",
        "gc-off" => "off",
        _ => "",
    }
}

fn read(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    let mut bytes = Vec::new();
    fs::File::open(path)
        .map_err(|error| error.to_string())?
        .take(limit + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| error.to_string())?;
    if bytes.len() as u64 > limit {
        return Err("measurement input exceeds bound".into());
    }
    Ok(bytes)
}
fn write(path: &Path, value: &impl Serialize) -> Result<(), String> {
    use std::os::unix::fs::OpenOptionsExt;
    let mut file = fs::OpenOptions::new()
        .create_new(true)
        .write(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(&serde_json::to_vec(value).map_err(|error| error.to_string())?)
        .map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())
}
pub(super) struct Executable {
    pub(super) path: PathBuf,
    pub(super) sha256: String,
    pub(super) role: Role,
    pub(super) receipt: Option<PathBuf>,
}
impl Executable {
    pub(super) fn verify(&self, proof: &CheckoutProof) -> Result<(), String> {
        if canonical::digest(&read(&self.path, 512 << 20)?) != self.sha256 {
            return Err("admitted native role executable changed".into());
        }
        if let Some(path) = &self.receipt {
            let receipt = build::validate_admission(proof, path, &self.role)?;
            if receipt.executable != self.path || receipt.executable_sha256 != self.sha256 {
                return Err("native role admission changed executable binding".into());
            }
        }
        Ok(())
    }
}
fn role_kind(role: &Role) -> Result<VariantRole, String> {
    Ok(match role {
        Role::Stress => return Err("stress-jitter helper cannot serve profile comparisons".into()),
        Role::Release => VariantRole::Ordinary,
        Role::Warm => VariantRole::DiagnosticProfile,
        Role::Diagnostic(features) if features.iter().any(|value| value == "guest-profile") => {
            VariantRole::GuestProfile
        }
        Role::Diagnostic(features)
            if features.iter().any(|value| value == "diagnostic-fuel-off") =>
        {
            VariantRole::DiagnosticFuelOff
        }
        Role::Diagnostic(_) => VariantRole::DiagnosticProfile,
    })
}
fn role_input(call: &Invocation, name: &str) -> Result<(PathBuf, Role, String), String> {
    let (helper, role, receipt) = match name {
        "reference" => (
            if call.name == "host-owner-profile" {
                value(call, "GOMAD_WASM_HOST_OWNER_REFERENCE_HELPER")
            } else {
                value(call, "GOMAD_WASM_PROFILE_REFERENCE_HELPER")
            },
            Role::Release,
            "",
        ),
        "baseline" => (
            value(call, "GOMAD_WASM_MATCHED_BASELINE_HELPER"),
            Role::Release,
            "",
        ),
        "diagnostic" => (
            value(call, "GOMAD_WASM_TEST_HELPER"),
            Role::Warm,
            value(call, "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION"),
        ),
        "guest" => (
            value(call, "GOMAD_WASM_TEST_HELPER"),
            Role::Diagnostic(vec!["diagnostic-profile".into(), "guest-profile".into()]),
            value(call, "GOMAD_WASM_GUEST_HELPER_ADMISSION"),
        ),
        "fuel-off" => (
            value(call, "GOMAD_WASM_FUEL_OFF_HELPER"),
            Role::Diagnostic(vec![
                "diagnostic-fuel-off".into(),
                "diagnostic-profile".into(),
            ]),
            value(call, "GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION"),
        ),
        "ordinary" | "candidate" => (value(call, "GOMAD_WASM_TEST_HELPER"), Role::Release, ""),
        other => {
            let (_, path) = value(call, "GOMAD_SCRATCH_HELPERS")
                .split(',')
                .filter_map(|value| value.split_once('='))
                .find(|(name, _)| *name == other)
                .ok_or("native scratch helper role absent")?;
            (
                path,
                if other.starts_with("diag") {
                    Role::Warm
                } else {
                    Role::Release
                },
                "",
            )
        }
    };
    if !Path::new(helper).is_absolute() {
        return Err(format!(
            "{name} requires explicit absolute native executable"
        ));
    }
    Ok((helper.into(), role, receipt.into()))
}
pub(super) fn admit(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    name: &str,
) -> Result<Executable, String> {
    let (path, role, receipt) = role_input(call, name)?;
    admit_executable(
        call,
        proof,
        root,
        name,
        path,
        role,
        if receipt.is_empty() {
            None
        } else {
            Some(PathBuf::from(receipt))
        },
    )
}
pub(super) fn admit_executable(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    name: &str,
    path: PathBuf,
    role: Role,
    receipt: Option<PathBuf>,
) -> Result<Executable, String> {
    if !path.is_absolute() {
        return Err("absolute native role executable required".into());
    }
    let directory = root.join("roles");
    fs::create_dir_all(&directory).map_err(|error| error.to_string())?;
    let environment = call
        .environ
        .iter()
        .filter_map(|entry| entry.split_once('='))
        .filter(|(key, _)| !key.starts_with("GOMAD_"))
        .map(|(key, value)| (key.into(), value.into()))
        .collect();
    let digest = canonical::digest(&read(&path, 512 << 20)?);
    let bytes = build::command_bounded(
        &directory,
        &format!("{name}-identity"),
        &directory,
        &environment,
        &[path.to_string_lossy().into_owned(), "--identity".into()],
        Duration::from_secs(10),
        1 << 20,
    )?;
    let identity: build::NativeIdentity =
        serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    build::validate_identity(&identity, &role, &digest)?;
    if matches!(name, "guest" | "fuel-off") && receipt.is_none() {
        return Err(format!(
            "{name} requires retained native helper admission receipt"
        ));
    }
    let admitted = Executable {
        path,
        sha256: digest,
        role,
        receipt,
    };
    admitted.verify(proof)?;
    write(
        &directory.join(format!("{name}-admission.json")),
        &json!({"path":admitted.path,"sha256":admitted.sha256,"identity":identity,"receipt":admitted.receipt}),
    )?;
    Ok(admitted)
}
fn selection(run: &str) -> Selection {
    if run == "runtime" {
        Selection::Runtime
    } else if run == "^TestFrontendSystemInfo$" {
        Selection::Target {
            source: "./tests/gomadfunctional".into(),
            cwd: "/workspace/tests/gomadfunctional".into(),
            run: run.into(),
        }
    } else {
        Selection::Selected(run.into())
    }
}
fn selection_key(run: &str, variant: &str) -> String {
    let key = canonical::digest(run.as_bytes());
    format!(
        "{}-{}",
        &key[..16],
        if variant == "gowasm" {
            "satconv-signext"
        } else {
            "baseline"
        }
    )
}
fn apply(row: &RunSpec, mut input: Value, cache: &Path) -> Result<Value, String> {
    input["compile_cache"] = json!(cache);
    if row.prepare {
        let args = input["guest"]["model"]["arguments"]
            .as_array_mut()
            .ok_or("native preparation argument vector absent")?;
        for argument in args {
            if argument
                .as_str()
                .is_some_and(|value| value.starts_with("-test.run="))
            {
                *argument = json!("-test.run=^$");
            }
        }
    }
    if !row.gc.is_empty() {
        let limit = input["guest"]["limits"]["memory_bytes"]
            .as_u64()
            .ok_or("GC variant memory bound absent")?;
        let env = input["guest"]["model"]["environment"]
            .as_array_mut()
            .ok_or("GC variant modeled environment absent")?;
        if env.iter().any(|entry| {
            entry
                .as_str()
                .is_some_and(|entry| entry.starts_with("GOGC=") || entry.starts_with("GOMEMLIMIT="))
        }) {
            return Err("GC variant requires unmodified baseline GC environment".into());
        }
        env.push(json!(format!("GOGC={}", row.gc)));
        if row.gc == "off" {
            env.push(json!(format!("GOMEMLIMIT={}B", limit / 4 * 3)));
        }
    }
    if row.sampling {
        input["guest"]["guest_profile"] = json!({"interval_micros":1000,"max_samples":10000});
    }
    Ok(input)
}
pub(super) fn cache_directory(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::DirBuilderExt;
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(path)
        .map_err(|error| error.to_string())?;
    crate::evidence::store::open::pin(path).map(|_| ())
}
fn cache(call: &Invocation, root: &Path, role: &str) -> Result<PathBuf, String> {
    let key = match role {
        "baseline" => "GOMAD_WASM_MATCHED_BASELINE_CACHE",
        "reference" => "GOMAD_WASM_PROFILE_REFERENCE_CACHE",
        _ => "GOMAD_WASM_MATCHED_CACHE",
    };
    let given = value(call, key);
    let path = if given.is_empty() {
        root.join(format!("compile-cache-{role}"))
    } else {
        PathBuf::from(given)
    };
    if !path.is_absolute() {
        return Err("compiled cache must be absolute".into());
    }
    cache_directory(&path)?;
    Ok(path)
}
pub(super) fn execute(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    row: &RunSpec,
    role: &Executable,
    input: Value,
) -> Result<NativeRun, String> {
    role.verify(proof)?;
    let evidence_root = root.join("evidence");
    fs::create_dir_all(&evidence_root).map_err(|error| error.to_string())?;
    let input_path = evidence_root.join(format!("{}.input.json", row.name));
    write(&input_path, &input)?;
    let output = evidence_root.join(&row.name);
    let environment = call
        .environ
        .iter()
        .filter_map(|entry| entry.split_once('='))
        .filter(|(key, _)| !key.starts_with("GOMAD_"))
        .map(|(key, value)| (key.into(), value.into()))
        .collect();
    let command = build::command_bounded(
        &evidence_root,
        &format!("{}-driver", row.name),
        proof.repo(),
        &environment,
        &[
            role.path.to_string_lossy().into_owned(),
            "--linked-run".into(),
            input_path.to_string_lossy().into_owned(),
            output.to_string_lossy().into_owned(),
        ],
        Duration::from_secs(if row.selection == "runtime" {
            190
        } else {
            1810
        }),
        64 << 20,
    );
    role.verify(proof)?;
    let result = evidence::read_profile_run(&output, evidence::ReadLimits::default());
    match (command, result) {
        (Ok(_), Ok(result)) => Ok(result),
        (Err(error), _) | (_, Err(error)) => Err(format!(
            "{}: {error}; actual failed tail retained in {}",
            row.name,
            output.display()
        )),
    }
}
pub(super) fn success(run: &NativeRun, row: &RunSpec) -> Result<(), String> {
    if run.root.outcome["termination"] != "exit" || run.root.outcome["exit_code"] != 0 {
        return Err(format!(
            "{}: expected successful actual selected guest; outcome {}",
            row.name, run.root.outcome
        ));
    }
    if row.warm
        && (run.root.outcome["metrics"]["cache_hit"] != true
            || run.root.outcome["metrics"]["compile_nanos"] != 0)
    {
        return Err(format!(
            "{}: actual warm serialized-code cache contract not met",
            row.name
        ));
    }
    if !row.prepare
        && run.report["authorization_to_terminal_nanos"]
            .as_u64()
            .unwrap_or(0)
            == 0
    {
        return Err("validated native authorization-to-terminal boundary unavailable".into());
    }
    if !row.prepare && row.selection != "runtime" {
        let test = row
            .selection
            .split('/')
            .next()
            .unwrap_or("")
            .trim_matches(['^', '$']);
        let witness = format!("=== RUN   {test}");
        let stdout = run
            .root
            .streams
            .get("stdout")
            .ok_or("native guest stdout absent")?;
        if !stdout
            .windows(witness.len())
            .any(|part| part == witness.as_bytes())
        {
            return Err(format!(
                "{}: actual selected suite did not execute: {witness}",
                row.name
            ));
        }
    }
    Ok(())
}
pub(super) fn replay_input(
    input: &mut Value,
    recorded: &NativeRun,
    identity: &linked::ChoiceIdentity,
) -> Result<(), String> {
    let trace = recorded
        .root
        .streams
        .get("choice")
        .ok_or("native replay choice trace absent")?;
    let plan = linked::project_choices(trace, identity).map_err(str::to_owned)?;
    let replay = json!({"schema":"gomad.replay-trace/v1","execution_identity":identity,"trace_sha256":canonical::digest(trace),"tape_sha256":canonical::digest(&plan.bytes),"decisions":plan.decisions.len(),"trace_bytes":trace.len(),"trace_frames":trace.len().div_ceil(65536),"prefix_control":null});
    input["guest"]["model"]["configuration"]["mode"] = json!(2);
    input["guest"]["model"]["configuration"]["replay"] = replay;
    input["guest"]["model"]["replay_trace"] = json!(trace);
    Ok(())
}

fn comparison(
    before: &NativeRun,
    after: &NativeRun,
    row: &RunSpec,
    before_role: &Role,
    after_role: &Role,
    proof: Option<&module::VariantProof>,
    observational: bool,
) -> Result<(), String> {
    if row.sampling {
        return evidence::compare_sampling(before, after, 1000, 10000);
    }
    if row.variant == "gowasm" {
        return evidence::compare_module_variant(
            before,
            after,
            proof.ok_or("authenticated GOWASM source proof absent")?,
        );
    }
    if !row.gc.is_empty() {
        return evidence::compare(
            before,
            after,
            ComparisonMask::Variant {
                baseline: role_kind(before_role)?,
                candidate: role_kind(after_role)?,
                gc: Some(evidence::GcMask {
                    baseline: "",
                    candidate: &row.gc,
                }),
            },
        );
    }
    if observational || role_kind(before_role)? != role_kind(after_role)? {
        evidence::compare(
            before,
            after,
            ComparisonMask::Variant {
                baseline: role_kind(before_role)?,
                candidate: role_kind(after_role)?,
                gc: None,
            },
        )
    } else {
        evidence::compare(before, after, ComparisonMask::Exact)
    }
}
fn check_supplied_module(call: &Invocation, prepared: &module::Prepared) -> Result<(), String> {
    let key = match call.name.as_str() {
        "host-owner-profile" => "GOMAD_WASM_HOST_OWNER_MODULE",
        "host-call-profile" => "GOMAD_WASM_HOST_CALL_PROFILE_MODULE",
        "matched-suite" | "legacy-workflow-profile" | "failing-suite-profile" => {
            "GOMAD_WASM_MATCHED_MODULE"
        }
        _ => return Ok(()),
    };
    let supplied = value(call, key);
    if supplied.is_empty() {
        if matches!(
            call.name.as_str(),
            "legacy-workflow-profile" | "failing-suite-profile" | "matched-suite"
        ) {
            return Err("retained matched native module required".into());
        }
        return Ok(());
    }
    if !Path::new(supplied).is_absolute()
        || read(Path::new(supplied), 512 << 20)? != prepared.guest.module
    {
        return Err(
            "supplied immutable module differs from actual source-bound native preparation".into(),
        );
    }
    let declared = value(call, "GOMAD_WASM_MATCHED_TOOLCHAIN_ID");
    if !declared.is_empty() && declared != prepared.guest.toolchain {
        return Err(
            "declared native matched toolchain identity differs from actual prepared inputs".into(),
        );
    }
    Ok(())
}
fn require_external_evidence(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
) -> Result<(), String> {
    if matches!(call.name.as_str(), "guest-profile" | "guest-variants")
        && fs::canonicalize(root)
            .map_err(|error| error.to_string())?
            .starts_with(fs::canonicalize(proof.repo()).map_err(|error| error.to_string())?)
    {
        return Err(
            "raw guest profile/variant evidence must be retained outside the repository".into(),
        );
    }
    Ok(())
}
fn retained_report(
    root: &Path,
    call: &Invocation,
    rows: &[Value],
    failure: Option<&str>,
) -> Result<PathBuf, String> {
    let path = root.join(match call.name.as_str() {
        "guest-profile" => "guest-attribution-report.json",
        "guest-variants" => "guest-variants-report.json",
        "host-call-profile" => "host-call-profile.json",
        "host-owner-profile" => "host-owner-profile.json",
        "matched-suite" => "helper-matched-suite-report.json",
        "migration" => "helper-migration-report.json",
        _ => "measurement.json",
    });
    write(
        &path,
        &json!({"schema":"gomad.native-named-measurement/v1","producer":call.name,"diagnostic_only":true,"accepted":failure.is_none(),"failure":failure,"runs":rows,"qualification":"diagnostic-only; final frozen candidate qualification remains separate","scope":"actual role-specific linked native execution; complete raw requests, modules, canonical streams, terminal and profile receipts retained; timing scopes remain separate"}),
    )?;
    Ok(path)
}

/// Runs actual named single-guest measurement recipes, retaining every attempted native child.
pub fn run(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    let result = run_inner(call, proof, root);
    if let Err(error) = &result {
        if root.is_dir() {
            let report = write(
                &root.join("failure-report.json"),
                &json!({"schema":"gomad.native-measurement-failure/v1","producer":call.name,"error":error,"raw_attempts":root.join("evidence"),"completed":false}),
            );
            if let Err(retention) = report {
                return Err(format!("{error}; retain failure report: {retention}"));
            }
        }
    }
    result
}
fn run_inner(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    if proof.repo() != call.repo || !root.is_absolute() {
        return Err("measurement selected-checkout proof and absolute output root required".into());
    }
    for key in match call.name.as_str() {
        "legacy-workflow-profile" => vec![
            "GOMAD_WASM_MATCHED_MODULE",
            "GOMAD_WASM_MATCHED_CACHE",
            "GOMAD_WASM_MATCHED_TOOLCHAIN_ID",
        ],
        "failing-suite-profile" => vec![
            "GOMAD_WASM_MATCHED_MODULE",
            "GOMAD_WASM_MATCHED_CACHE",
            "GOMAD_WASM_PROFILE_REFERENCE_HELPER",
            "GOMAD_WASM_PROFILE_REFERENCE_CACHE",
        ],
        "matched-suite" => vec!["GOMAD_WASM_MATCHED_BASELINE_CACHE"],
        _ => vec![],
    } {
        let supplied = value(call, key);
        if supplied.is_empty()
            || (key != "GOMAD_WASM_MATCHED_TOOLCHAIN_ID" && !Path::new(supplied).is_absolute())
        {
            return Err(format!(
                "{key} must explicitly bind an absolute native input (toolchain identity must be nonempty)"
            ));
        }
    }
    fs::create_dir_all(root).map_err(|error| error.to_string())?;
    require_external_evidence(call, proof, root)?;
    let schedule = plan(call)?;
    for row in &schedule {
        role_input(call, &row.role)?;
    }
    let host = host::current(&root.join("host"), &call.environ)?;
    write(&root.join("host.json"), &host)?;
    let snapshot = host::source_snapshot(proof, &root.join("source"), &call.environ)?;
    if call.name == "host-owner-profile" {
        let tools_status = build::command_bounded(
            root,
            "tools-source-status",
            proof.repo(),
            &BTreeMap::from([("PATH".into(), std::env::var("PATH").unwrap_or_default())]),
            &[
                "git".into(),
                "-C".into(),
                proof.repo().to_string_lossy().into_owned(),
                "status".into(),
                "--porcelain".into(),
                "--".into(),
                "tools/gomad".into(),
            ],
            Duration::from_secs(10),
            1 << 20,
        )?;
        if !tools_status.is_empty() {
            return Err("host owner profile requires committed tools/gomad sources".into());
        }
    }
    write(&root.join("source.json"), &snapshot)?;
    if call.name == "migration" {
        return migration(call, proof, root);
    }
    write(&root.join("schedule.json"), &schedule)?;
    let mut roles = BTreeMap::new();
    for row in &schedule {
        if !roles.contains_key(&row.role) {
            roles.insert(row.role.clone(), admit(call, proof, root, &row.role)?);
        }
    }
    for (left, right) in [("reference", "diagnostic"), ("diagnostic", "fuel-off")] {
        if let (Some(left), Some(right)) = (roles.get(left), roles.get(right))
            && left.sha256 == right.sha256
        {
            return Err("native reference/diagnostic/fuel roles require distinct actual executable identities".into());
        }
    }
    let mut modules: BTreeMap<String, module::Prepared> = BTreeMap::new();
    for row in &schedule {
        let key = selection_key(&row.selection, &row.variant);
        if !modules.contains_key(&key) {
            let directory = root.join("modules").join(&key);
            let prepared = module::prepare_variant(
                proof,
                call,
                &directory,
                selection(&row.selection),
                if row.variant == "gowasm" {
                    "satconv,signext"
                } else {
                    ""
                },
            )?;
            check_supplied_module(call, &prepared)?;
            modules.insert(key, prepared);
        }
    }
    let mut variant_proofs = BTreeMap::new();
    if call.name == "guest-variants" {
        for (_, selected) in selections() {
            let before = modules
                .get(&selection_key(selected, "baseline"))
                .ok_or("baseline variant preparation absent")?;
            let after = modules
                .get(&selection_key(selected, "gowasm"))
                .ok_or("GOWASM variant preparation absent")?;
            variant_proofs.insert(
                selected.to_string(),
                module::prove_gowasm_variant(before, after)?,
            );
        }
    }
    let mut results: BTreeMap<String, (NativeRun, String)> = BTreeMap::new();
    let mut samples = Vec::new();
    let mut pending = Vec::<RunSpec>::new();
    let mut failures = Vec::new();
    for row in &schedule {
        let prepared = &modules[&selection_key(&row.selection, &row.variant)];
        let executable = &roles[&row.role];
        module::authenticate_prepared(prepared)?;
        executable.verify(proof)?;
        host::admit_preparation(
            &root.join("host"),
            &row.name,
            value(call, "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"),
            &call.environ,
        )?;
        let execution_cache = cache(call, root, &row.role)?;
        if call.name == "matched-suite" {
            let baseline = cache(call, root, "baseline")?;
            let candidate = cache(call, root, "candidate")?;
            if fs::canonicalize(baseline).map_err(|error| error.to_string())?
                == fs::canonicalize(candidate).map_err(|error| error.to_string())?
            {
                return Err("matched baseline and candidate caches must be separate".into());
            }
        }
        let mut input = apply(row, prepared.native_input(row.seed)?, &execution_cache)?;
        if let Some(reference) = &row.replay_from {
            let (before, _) = results
                .get(reference)
                .ok_or("native replay reference execution unavailable")?;
            replay_input(
                &mut input,
                before,
                prepared
                    .guest
                    .choice_identity
                    .as_ref()
                    .ok_or("native cooperative execution identity absent")?,
            )?;
        }
        let attempt = execute(call, proof, root, row, executable, input).and_then(|observed| {
            success(&observed, row)?;
            if row.selection == "runtime" {
                let stdout = observed
                    .root
                    .streams
                    .get("stdout")
                    .ok_or("runtime witness stdout absent")?;
                if !stdout
                    .windows(b"runtime-cost 100000".len())
                    .any(|part| part == b"runtime-cost 100000")
                {
                    return Err("runtime loop witness missing".into());
                }
                let trace = observed
                    .root
                    .streams
                    .get("choice")
                    .ok_or("runtime choice witness absent")?;
                let counts = linked::inspect_choices(
                    trace,
                    prepared
                        .guest
                        .choice_identity
                        .as_ref()
                        .ok_or("runtime choice identity absent")?,
                )
                .map_err(str::to_owned)?;
                if counts.select_poll < 100000
                    || observed.root.outcome["metrics"]["execute_nanos"]
                        .as_u64()
                        .unwrap_or(0)
                        == 0
                {
                    return Err(
                        "runtime select-poll or actual execution timing witness missing".into(),
                    );
                }
            }
            if row.sampling {
                let profile = read(
                    &root
                        .join("evidence")
                        .join(&row.name)
                        .join("guest-profile.json"),
                    64 << 20,
                )?;
                let attribution = guest::decode(&profile)?;
                if attribution.guest_sample_weight == 0 {
                    return Err("guest profile contains no weighted guest samples".into());
                }
                write(
                    &root
                        .join("evidence")
                        .join(format!("{}-attribution.json", row.name)),
                    &attribution,
                )?;
            }
            if let Some(reference) = &row.replay_from {
                evidence::compare_replay(
                    &results[reference].0,
                    &observed,
                    prepared
                        .guest
                        .choice_identity
                        .as_ref()
                        .ok_or("native replay identity absent")?,
                )?;
            }
            Ok(observed)
        });
        if let Err(error) =
            module::authenticate_prepared(prepared).and_then(|_| executable.verify(proof))
        {
            return Err(format!(
                "{error}; attempted run error: {:?}",
                attempt.as_ref().err()
            ));
        }
        match attempt {
            Ok(observed) => {
                samples.push(json!({"sample":row,"status":"executed","actual_outcome":observed.root.outcome,"profile":observed.profile,"native_root":root.join("evidence").join(&row.name)}));
                results.insert(row.name.clone(), (observed, row.role.clone()));
                if row.compare_to.is_some() {
                    pending.push(row.clone());
                }
                let mut remaining = Vec::new();
                for comparison_row in pending.drain(..) {
                    let reference = comparison_row.compare_to.as_ref().unwrap();
                    if let (Some((before, before_role)), Some((after, after_role))) =
                        (results.get(reference), results.get(&comparison_row.name))
                    {
                        if let Err(error) = comparison(
                            before,
                            after,
                            &comparison_row,
                            &roles[before_role].role,
                            &roles[after_role].role,
                            variant_proofs.get(&comparison_row.selection),
                            call.name == "guest-variants",
                        ) {
                            if let Some(sample) = samples
                                .iter_mut()
                                .find(|sample| sample["sample"]["name"] == comparison_row.name)
                            {
                                sample["comparison_failure"] = json!(&error);
                            }
                            failures.push(format!("{}: {error}", comparison_row.name));
                            if call.name != "guest-variants" {
                                retained_report(root, call, &samples, Some(&failures.join("; ")))?;
                                return Err(failures.join("; "));
                            }
                        }
                    } else {
                        remaining.push(comparison_row);
                    }
                }
                pending = remaining;
            }
            Err(error) => {
                samples.push(json!({"sample":row,"status":"failed","failure":error,"native_root":root.join("evidence").join(&row.name)}));
                failures.push(error);
                if call.name != "guest-variants" {
                    retained_report(root, call, &samples, Some(&failures.join("; ")))?;
                    return Err(failures.join("; "));
                }
            }
        }
    }
    if !pending.is_empty() {
        failures.push("required actual comparison reference unavailable".into());
    }
    if call.name == "guest-variants" {
        variant_recommendations(root, &samples)?;
    }
    if call.name == "legacy-workflow-profile" {
        let retained = &results["seed0-retained"].0;
        let transcript = retained
            .root
            .streams
            .get("transcript")
            .ok_or("retained canonical transcript absent")?;
        let counts = super::count_operations(transcript, 64 << 20)?;
        write(
            &root.join("canonical-operation-counts.json"),
            &json!({"calls":counts.calls,"local":counts.local,"remote":counts.remote,"operations":counts.operations}),
        )?;
    }
    let failure = (!failures.is_empty()).then(|| failures.join("; "));
    let report = retained_report(root, call, &samples, failure.as_deref())?;
    if call.name == "guest-variants" {
        return Ok(report);
    }
    if let Some(failure) = failure {
        Err(format!("{failure}; retained {}", report.display()))
    } else {
        Ok(report)
    }
}
fn variant_recommendations(root: &Path, samples: &[Value]) -> Result<(), String> {
    let mut rows = Vec::new();
    for (selection, run) in selections() {
        for seed in SEEDS {
            let baseline: Vec<_> = samples
                .iter()
                .filter(|row| {
                    row["sample"]["selection"] == run
                        && row["sample"]["seed"] == seed
                        && row["sample"]["variant"] == "baseline"
                        && row["sample"]["prepare"] == false
                })
                .collect();
            let before = baseline
                .iter()
                .filter_map(|row| row["profile"]["phases"]["execute"]["wall_nanos"].as_u64())
                .collect::<Vec<_>>();
            for variant in ["baseline", "gc200", "gc400", "gc-off", "gowasm", "fuel-off"] {
                let selected: Vec<_> = samples
                    .iter()
                    .filter(|row| {
                        row["sample"]["selection"] == run
                            && row["sample"]["seed"] == seed
                            && row["sample"]["variant"] == variant
                            && row["sample"]["prepare"] == false
                    })
                    .collect();
                let after = selected
                    .iter()
                    .filter_map(|row| row["profile"]["phases"]["execute"]["wall_nanos"].as_u64())
                    .collect::<Vec<_>>();
                let preparation_failed = samples.iter().any(|row| {
                    row["sample"]["selection"] == run
                        && row["sample"]["prepare"] == true
                        && (row["sample"]["variant"] == variant
                            || row["sample"]["variant"] == "baseline")
                        && row["status"] != "executed"
                });
                let failed = preparation_failed
                    || selected.iter().any(|row| {
                        row["status"] != "executed" || !row["comparison_failure"].is_null()
                    })
                    || baseline.iter().any(|row| {
                        row["status"] != "executed" || !row["comparison_failure"].is_null()
                    })
                    || after.len() != 5
                    || before.len() != 5;
                let decision = if variant == "baseline" {
                    "control"
                } else {
                    statistics::recommendation(&before, &after, failed, !failed)
                };
                rows.push(json!({"selection":selection,"seed":seed,"variant":variant,"execute_nanos":after,"baseline_execute_nanos":before,"recommendation":decision,"adoption_requires_separate_qualification":true}));
            }
        }
    }
    write(&root.join("variant-recommendations.json"), &rows)
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct MigrationBaseline {
    schema: String,
    guests: BTreeMap<String, MigrationGuest>,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct MigrationGuest {
    provenance: PathBuf,
    provenance_sha256: String,
    runs: BTreeMap<String, PathBuf>,
}
fn migration(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    let base = Path::new(value(call, "GOMAD_WASM_MIGRATION_BASELINE"));
    if !base.is_absolute() {
        return Err("absolute native migration baseline root required".into());
    }
    let baseline: MigrationBaseline = serde_json::from_slice(&read(
        &base.join("native-migration-baseline.json"),
        8 << 20,
    )?)
    .map_err(|error| {
        format!("native migration baseline required; Go-lineage reports are refused: {error}")
    })?;
    if baseline.schema != "gomad.native-migration-baseline/v1" || baseline.guests.len() != 3 {
        return Err(
            "native migration baseline requires exact choices/runtime_probes/time_probes registry"
                .into(),
        );
    }
    let executable = admit(call, proof, root, "ordinary")?;
    let mut samples = Vec::new();
    let cache = if value(call, "GOMAD_WASM_MIGRATION_CACHE").is_empty() {
        root.join("compile-cache")
    } else {
        PathBuf::from(value(call, "GOMAD_WASM_MIGRATION_CACHE"))
    };
    if !cache.is_absolute() {
        return Err("migration compiled cache must be absolute".into());
    }
    cache_directory(&cache)?;
    for name in ["choices", "runtime_probes", "time_probes"] {
        let guest = baseline
            .guests
            .get(name)
            .ok_or("missing native migration guest")?;
        if !guest.provenance.is_absolute()
            || canonical::digest(&read(&guest.provenance, 16 << 20)?) != guest.provenance_sha256
        {
            return Err("native migration module provenance changed".into());
        }
        let provenance = crate::qualify::spec::strict(&read(&guest.provenance, 16 << 20)?)?;
        crate::campaign::prepare::typed_provenance(&provenance)?;
        let expected_source = proof.repo().join("tools/gomad").join(match name {
            "choices" => "internal/backends/wasm/toolchain/testdata/choices",
            "runtime_probes" => "internal/backends/wasm/testdata/runtime_probes",
            _ => "internal/backends/wasm/testdata/time_probes",
        });
        let actual_directory = Path::new(
            provenance["inputs"]["directory"]
                .as_str()
                .ok_or("native migration build directory absent")?,
        );
        let actual_source = actual_directory.join(
            provenance["inputs"]["source"]
                .as_str()
                .ok_or("native migration source target absent")?,
        );
        if fs::canonicalize(actual_source).map_err(|error| error.to_string())?
            != fs::canonicalize(expected_source).map_err(|error| error.to_string())?
            || provenance["inputs"]["kind"] != "go-run"
        {
            return Err(
                "native migration guest label does not match actual original fixture source".into(),
            );
        }
        let identity: linked::ChoiceIdentity =
            serde_json::from_value(provenance["choice_identity"].clone())
                .map_err(|error| error.to_string())?;
        if provenance["build_key"] != identity.toolchain_build_key
            || provenance["module_sha256"] != hex(&identity.target_sha256)
        {
            return Err("native migration provenance module/choice identity mismatch".into());
        }
        for inventory in ["sources", "compiler"] {
            let files: BTreeMap<PathBuf, String> =
                serde_json::from_value(provenance["inputs"][inventory].clone())
                    .map_err(|error| error.to_string())?;
            if files.is_empty() {
                return Err("native migration provenance source/compiler closure absent".into());
            }
            module::verify_inputs(&files)?;
        }
        let mut first = None;
        for (seed_index, seed) in [0, 1000003].into_iter().enumerate() {
            let original_root = guest
                .runs
                .get(&seed.to_string())
                .filter(|path| path.is_absolute())
                .ok_or("missing native migration baseline seed root")?;
            let old = evidence::read_profile_run(original_root, evidence::ReadLimits::default())?;
            if old.request["kind"] != "standalone"
                || old.request["guest"]["module_sha256"] != hex(&identity.target_sha256)
                || old.request["guest"]["model"]["configuration"]["seed"] != seed
            {
                return Err("native migration baseline actual module/seed binding mismatch".into());
            }
            let request = &old.request;
            let configuration = &request["guest"]["model"]["configuration"];
            if request["guest"]["limits"]["memory_bytes"] != (4_u64 << 30)
                || request["guest"]["limits"]["fuel"] != (1_u64 << 50)
                || request["guest"]["limits"]["wall_millis"] != 1800000_u64
                || request["guest"]["limits"]["output_bytes"] != (1_u64 << 20)
                || configuration["limits"]["calls"] != 1000000_u64
                || configuration["limits"]["transcript_bytes"] != (64_u64 << 20)
                || configuration["read_step_nanos"] != 0
                || configuration["diagnostics"] != false
            {
                return Err(
                    "native migration fixture changed original execution budgets/policy".into(),
                );
            }
            if seed_index == 0 {
                let row = prepare_row(&format!("{name}-strict-prepare"), "migration", "ordinary");
                let mut input = old.request.clone();
                input["compile_cache"] = json!(cache);
                let warmed = execute(call, proof, root, &row, &executable, input)?;
                migration_success(&warmed, &row)?;
                samples.push(json!({"sample":row,"status":"executed","native_root":root.join("evidence").join(&row.name)}));
            }
            let row = RunSpec::new(
                format!("{name}-strict-seed{seed}"),
                "migration",
                seed,
                "ordinary",
            );
            let mut input = old.request.clone();
            input["compile_cache"] = json!(cache);
            let observed = execute(call, proof, root, &row, &executable, input)?;
            migration_success(&observed, &row)?;
            evidence::compare(&old, &observed, ComparisonMask::Exact)?;
            samples.push(json!({"sample":row,"status":"identical","baseline_root":original_root,"native_root":root.join("evidence").join(&row.name),"actual_outcome":observed.root.outcome}));
            if seed == 0 {
                first = Some(observed);
            }
        }
        let first = first.ok_or("native migration seed0 witness absent")?;
        let row = RunSpec::new(format!("{name}-strict-replay"), "migration", 0, "ordinary");
        let mut input = first.request.clone();
        replay_input(&mut input, &first, &identity)?;
        let replayed = execute(call, proof, root, &row, &executable, input)?;
        migration_success(&replayed, &row)?;
        evidence::compare_replay(&first, &replayed, &identity)?;
        samples.push(json!({"sample":row,"status":"exact-replay","native_root":root.join("evidence").join(&row.name)}));
    }
    retained_report(root, call, &samples, None)
}
fn hex(bytes: &[u8]) -> String {
    bytes.iter().map(|byte| format!("{byte:02x}")).collect()
}
fn migration_success(run: &NativeRun, row: &RunSpec) -> Result<(), String> {
    let code = run.root.outcome["exit_code"].as_u64();
    if run.root.outcome["termination"] != "exit" || !matches!(code, Some(0 | 17)) {
        return Err("native migration fixture outcome changed".into());
    }
    if row.warm
        && (run.root.outcome["metrics"]["cache_hit"] != true
            || run.root.outcome["metrics"]["compile_nanos"] != 0)
    {
        return Err("native migration existing compiled cache not used".into());
    }
    Ok(())
}

#[cfg(test)]
#[path = "workloads_tests.rs"]
mod tests;
