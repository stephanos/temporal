use super::*;
use crate::evidence::store::Opened;
use std::collections::{BTreeMap, BTreeSet};

fn require(condition: bool, message: &str) -> Result<(), String> {
    if condition {
        Ok(())
    } else {
        Err(message.into())
    }
}
fn contained(root: &Path, path: &Path) -> Result<(), String> {
    require(
        path.is_absolute()
            && path.starts_with(root)
            && !path.components().any(|part| {
                matches!(
                    part,
                    std::path::Component::ParentDir | std::path::Component::CurDir
                )
            }),
        "stress evidence escapes retained parent",
    )
}
pub(super) fn cases() -> Vec<(&'static str, u64)> {
    (0..32)
        .map(|index| ("runtime", index * 1_000_003))
        .chain([7, 19, 42].map(|seed| ("two-cluster", seed)))
        .chain(
            ["workflow", "activity"]
                .into_iter()
                .flat_map(|name| [0, 1_000_003, 2_000_006].map(|seed| (name, seed))),
        )
        .collect()
}
fn text<'a>(row: &'a Value, key: &str) -> Result<&'a str, String> {
    row[key]
        .as_str()
        .ok_or_else(|| format!("stress {key} must be a string"))
}
fn array<'a>(row: &'a Value, key: &str) -> Result<&'a Vec<Value>, String> {
    row[key]
        .as_array()
        .ok_or_else(|| format!("stress {key} must be an array"))
}
fn child_request(
    receipt: &Receipt,
    run: &Value,
    name: &str,
    seed: u64,
    axis: &str,
    index: usize,
) -> Result<(), String> {
    let command = array(run, "executed_command")?;
    require(
        command.len() == 4 && command[1] == "stress" && command[2] == "--cell",
        "wrong stress child command",
    )?;
    let executable = Path::new(
        command[0]
            .as_str()
            .ok_or("stress child executable absent")?,
    );
    contained(&receipt.root, executable)?;
    require(
        digest(executable)? == receipt.executable_sha256,
        "stress child binary changed",
    )?;
    let request_path = Path::new(command[3].as_str().ok_or("stress child request absent")?);
    contained(&receipt.root, request_path)?;
    let child = request_path.parent().ok_or("stress child owner absent")?;
    require(
        value(&child.join("observation.json"))? == *run,
        "stress report/child observation differ",
    )?;
    let request = value(request_path)?;
    require(
        request["root"] == child.to_string_lossy().as_ref()
            && request["requested_root"] == receipt.requested_root.to_string_lossy().as_ref()
            && request["source_sha256"] == receipt.source_sha256
            && request["workload"] == name
            && request["seed"] == seed
            && request["axis"] == axis,
        "stress child request input drift",
    )?;
    let prepared_root = receipt.root.join("matrix").join(format!("prepared-{name}"));
    let spec = value(&receipt.spec)?;
    let workload = array(&spec, "workloads")?
        .iter()
        .find(|item| item["name"] == name)
        .ok_or("stress child workload recipe absent")?;
    let mut expected_options = crate::stress::admit_workload(workload)?;
    expected_options.target_kind = "exec".into();
    expected_options.source = prepared_root
        .join("module.wasm")
        .to_string_lossy()
        .into_owned();
    expected_options.provenance = prepared_root
        .join("provenance.json")
        .to_string_lossy()
        .into_owned();
    require(
        request["options"]
            == serde_json::to_value(expected_options).map_err(|error| error.to_string())?
            && request["target_pool"]
                == receipt
                    .root
                    .join("matrix/.targets")
                    .to_string_lossy()
                    .as_ref(),
        "stress child options/preparation changed",
    )?;
    for stream in ["stdout.log", "stderr.log"] {
        bytes(&child.join(stream), 64 << 20)?;
    }
    host_conditions(receipt, run, executable, &request, axis, index)
}

fn host_conditions(
    receipt: &Receipt,
    run: &Value,
    executable: &Path,
    request: &Value,
    axis: &str,
    index: usize,
) -> Result<(), String> {
    let fanout = if axis == "concurrent" {
        receipt.controls.parallel
    } else {
        1
    };
    let supervision = json!({"cpu_workers":if axis=="cpu"{receipt.controls.cpu_workers}else{0},"cohort_size":fanout,"repetition":index/fanout,"member":index%fanout});
    require(
        request["supervision"] == supervision && run["supervision"] == supervision,
        "stress original CPU/cohort/repetition/member changed",
    )?;
    let cwd = executable
        .parent()
        .ok_or("stress executable layout owner absent")?;
    require(
        run["host_conditions"]["executable"] == executable.to_string_lossy().as_ref()
            && run["host_conditions"]["cwd"] == cwd.to_string_lossy().as_ref(),
        "stress actual child executable/cwd changed",
    )?;
    let layout = axis == "layout";
    let directory = if layout {
        "p".repeat(1 + (index % 4) * 40)
    } else {
        "process".into()
    };
    require(
        cwd.file_name()
            .is_some_and(|name| name == directory.as_str())
            && run["host_conditions"]["padding_bytes"]
                == if layout { 1 + (index % 16) * 4096 } else { 0 },
        "stress actual layout perturbation changed",
    )?;
    let handoff = axis == "handoff";
    require(
        run["host_conditions"]["jitter"] == handoff
            && (!handoff
                || run["host_conditions"]["jitter_delay_calls"]
                    .as_u64()
                    .is_some_and(|value| value > 0)),
        "stress handoff lacks actual jitter delays",
    )?;
    Ok(())
}

fn observations(
    receipt: &Receipt,
    run: &Value,
    name: &str,
    seed: u64,
    axis: &str,
    index: usize,
    pids: &mut BTreeSet<u64>,
) -> Result<(), String> {
    require(
        run["fresh"] == true
            && run["workload"] == name
            && run["seed"] == seed
            && run["axis"] == axis
            && run["oracle_error"].is_null()
            && run["perturbation_error"].is_null()
            && run["event_order_error"].is_null(),
        "forced/failed/foreign native stress observation",
    )?;
    let pid = run["process_id"]
        .as_u64()
        .filter(|value| *value > 0)
        .ok_or("missing stress process identity")?;
    require(pids.insert(pid), "stress process identity reused")?;
    require(
        run["identities"]["source"] == receipt.source_sha256
            && run["identities"]["helper"] == receipt.identity.linked_implementation_sha256
            && run["identities"]["host"] == receipt.identity.host_implementation_sha256
            && run["identities"]["engine"] == receipt.identity.configuration,
        "stress source/helper/host/role identity changed",
    )?;
    child_request(receipt, run, name, seed, axis, index)?;
    let path = Path::new(text(run, "artifact")?);
    contained(&receipt.root, path)?;
    let opened = Opened::open(path)?;
    opened.authenticate_all()?;
    let record = &opened.manifest().record;
    require(
        record.seed == seed
            && serde_json::to_value(&record.identity).map_err(|error| error.to_string())?
                == run["identities"]["execution"]
            && opened.manifest().record_sha256 == text(run, "record_sha256")?,
        "stress canonical seed/identity changed",
    )?;
    let replay_file = if name == "two-cluster" {
        "evidence/cluster-replay.json"
    } else {
        "evidence/replay.json"
    };
    let replay: Value = crate::qualify::spec::strict(&opened.payload(replay_file, 16 << 20)?)?;
    crate::stress::admit_retained_configuration(name, replay["configuration"].clone())?;
    let binding: Value =
        crate::qualify::spec::strict(&opened.payload("evidence/stress-inputs.json", 16 << 10)?)?;
    require(
        binding
            == json!({"schema":"gomad.stress-inputs/v1","fresh":true,"seed":seed,"workload":name,"source_sha256":receipt.source_sha256}),
        "stress retained input binding changed",
    )?;
    let prepared = receipt.root.join("matrix").join(format!("prepared-{name}"));
    let provenance = value(&prepared.join("provenance.json"))?;
    require(
        digest(&prepared.join("provenance.json"))? == text(&run["identities"], "provenance")?
            && digest(&prepared.join("module.wasm"))? == record.identity.target
            && provenance["build_key"] == record.identity.toolchain
            && provenance["module_sha256"] == record.identity.target
            && canonical::digest(&canonical::encode(&provenance["inputs"])?)
                == record.identity.toolchain
            && provenance["inputs"]["host"] == receipt.identity.host_implementation_sha256
            && provenance["inputs"]["helper"] == receipt.identity.linked_implementation_sha256,
        "stress prepared module/provenance changed",
    )?;
    crate::campaign::prepare::typed_provenance(&provenance)?;
    oracles(&opened, name, &provenance)?;
    require(
        crate::stress::compare_retained(run, run)?["status"] == "identical",
        "stress reference lacks authenticated diagnostic evidence",
    )
}
fn oracles(opened: &Opened, name: &str, provenance: &Value) -> Result<(), String> {
    let record = &opened.manifest().record;
    let stdout = if name == "two-cluster" {
        Vec::new()
    } else {
        opened.payload("output/stdout", 256 << 20)?
    };
    let canary = name == "runtime"
        && record.outcome.class == "target_failure"
        && record.outcome.termination == "exit"
        && record.outcome.exit_code == Some(17)
        && !opened
            .manifest()
            .files
            .contains_key("evidence/soft-assertions.json")
        && stdout.windows(9).any(|raw| raw == b"canary 2\n");
    require(
        canary
            || record.outcome.class == "success"
                && record.outcome.termination == "exit"
                && record.outcome.exit_code == Some(0),
        "stress canonical outcome failed",
    )?;
    if name == "two-cluster" {
        return cluster_oracles(opened);
    }
    let text = String::from_utf8_lossy(&stdout);
    if name != "runtime" {
        let witness = if name == "workflow" {
            "--- PASS: TestWorkflowTypeEncodingSuite"
        } else {
            "--- PASS: TestActivityTestSuite"
        };
        return require(
            text.contains(witness) && !text.contains("--- SKIP:"),
            "stress original selection skipped or absent",
        );
    }
    for witness in [
        "entropy ",
        "map ",
        "runq ",
        "timer callbacks ",
        "timer parking",
        "note negative park wake",
        "collector automatic finalizer cleanup 1 1",
        "canary ",
    ] {
        require(text.contains(witness), "runtime stress witness absent")?;
    }
    let diagnostics = opened.payload("evidence/diagnostic", 32 << 20)?;
    let choices = opened.payload("evidence/choice", 32 << 20)?;
    let identity: linked::ChoiceIdentity =
        serde_json::from_value(provenance["choice_identity"].clone())
            .map_err(|error| error.to_string())?;
    let inspected = linked::inspect_choices(&choices, &identity)?;
    require(
        inspected.runnable > 0
            && inspected.select_poll > 0
            && inspected.select_result > 0
            && diagnostics.len() >= 96
            && diagnostics.len() == choices.len()
            && diagnostics.len() % 96 == 0,
        "runtime hook/diagnostic witness absent",
    )?;
    for (ordinal, row) in diagnostics.as_chunks::<96>().0.iter().enumerate() {
        require(
            u64::from_be_bytes(row[..8].try_into().unwrap()) == ordinal as u64
                && row[28] <= 2
                && row[29..32] == [0; 3]
                && row[36..40] == [0; 4],
            "runtime diagnostic record invalid",
        )?;
    }
    let last = &diagnostics[diagnostics.len() - 96..];
    require(
        u32::from_be_bytes(last[24..28].try_into().unwrap()) > 0
            && [16, 40, 56, 64, 80]
                .into_iter()
                .all(|at| u64::from_be_bytes(last[at..at + 8].try_into().unwrap()) > 0),
        "runtime collector/seeded draw witness absent",
    )
}
fn cluster_oracles(opened: &Opened) -> Result<(), String> {
    let summary: Value =
        crate::qualify::spec::strict(&opened.payload("evidence/cluster_summary", 128 << 20)?)?;
    let rows = array(&summary, "incarnations")?;
    require(
        rows.len() == 4,
        "stress cluster must retain four incarnations",
    )?;
    let mut observed = BTreeMap::new();
    for row in rows {
        let node = text(&row["handle"], "node")?;
        let incarnation = row["handle"]["incarnation"]
            .as_u64()
            .ok_or("cluster incarnation absent")?;
        require(
            ["active", "standby"].contains(&node)
                && [1, 2].contains(&incarnation)
                && !observed.contains_key(&(node.to_owned(), incarnation)),
            "unknown/duplicate stress cluster owner",
        )?;
        let prefix = format!("evidence/cluster/{node}/{incarnation}/");
        let raw = opened.payload(&(prefix.clone() + "stdout"), 8 << 20)?;
        let output_text = std::str::from_utf8(&raw).map_err(|error| error.to_string())?;
        let values = output_text
            .lines()
            .filter_map(|line| line.strip_prefix("GOMAD_TEMPORAL_OBSERVATION "))
            .collect::<Vec<_>>();
        require(values.len() == 1, "stress cluster observation cardinality")?;
        let value: Value = crate::qualify::spec::strict(values[0].as_bytes())?;
        let ids = array(&value, "event_ids")?;
        let types = array(&value, "types")?;
        require(
            !text(&value, "cluster_id")?.is_empty()
                && crate::qualify::cells::base64(text(&value, "history")?)
                    .is_some_and(|raw| !raw.is_empty())
                && ids.len() == types.len()
                && ids
                    .iter()
                    .enumerate()
                    .all(|(index, id)| id.as_u64() == Some(index as u64 + 1)),
            "stress history lost/duplicate events or identity",
        )?;
        if incarnation == 1 {
            require(
                row["termination"] == "crash" && row["streams"].get("storage_image").is_some(),
                "stress committed incarnation lacks crashed storage image",
            )?;
            opened.payload(&(prefix + "storage_image"), 128 << 20)?;
        } else {
            require(
                row["termination"] == "exit" && row["exit_code"] == 0,
                "stress reopened incarnation failed",
            )?;
        }
        observed.insert((node.to_owned(), incarnation), value);
    }
    for node in ["active", "standby"] {
        let initial = &observed[&(node.into(), 1)];
        let reopened = &observed[&(node.into(), 2)];
        require(
            ["history", "cluster_id", "event_ids", "types"]
                .into_iter()
                .all(|key| initial[key] == reopened[key]),
            "stress history/identity changed after reopen",
        )?;
        crate::stress::cluster::history_oracle(initial, reopened)?;
    }
    require(
        observed[&("active".into(), 1)]["cluster_id"]
            != observed[&("standby".into(), 1)]["cluster_id"]
            && observed[&("active".into(), 1)]["history"]
                == observed[&("standby".into(), 1)]["history"],
        "stress cluster identities coincide or history diverges",
    )
}

fn workload_recipes(receipt: &Receipt, spec: &Value) -> Result<(), String> {
    let workloads = array(spec, "workloads")?;
    require(workloads.len() == 4, "stress workload catalogue incomplete")?;
    let mut names = BTreeSet::new();
    for workload in workloads {
        let name = text(workload, "name")?;
        require(
            NAMES.contains(&name) && names.insert(name),
            "stress duplicate/unknown workload recipe",
        )?;
        if !receipt.controls.only.is_empty() && receipt.controls.only != name {
            continue;
        }
        let options = crate::stress::admit_workload(workload)?;
        let installation_path = Path::new(text(workload, "installation")?);
        contained(&receipt.root, installation_path)?;
        let installed = crate::install::manifest::resolve(Some(installation_path))?.manifest;
        let original = configuration(&receipt.repo, name)?;
        require(
            installed.profile == PROFILE
                && installed.fuel
                    == if name == "runtime" {
                        4_000_000_000
                    } else {
                        1 << 50
                    }
                && installed.memory_bytes
                    == if name == "runtime" {
                        256 << 20
                    } else {
                        4 << 30
                    }
                && installed
                    .admission
                    .as_ref()
                    .is_some_and(|admission| admission.guest_configuration == original),
            "stress original installation/bounds changed",
        )?;
        require(
            options.execution_timeout
                == Duration::from_secs(if name == "runtime" { 60 } else { 900 })
                && options.output_limit
                    == if name == "runtime" {
                        1 << 20
                    } else if name == "two-cluster" {
                        8 << 20
                    } else {
                        256 << 20
                    }
                && options.choice_bytes == if name == "runtime" { 1 << 20 } else { 32 << 20 },
            "stress original execution/choice/output bounds changed",
        )?;
        if name == "two-cluster" {
            let scenario = Path::new(text(workload, "scenario")?);
            contained(&receipt.root, scenario)?;
            require(
                bytes(scenario, 256 << 10)? == cluster_driver::scenario_bytes()?,
                "stress original four-incarnation scenario changed",
            )?;
        }
    }
    Ok(())
}

fn cell_runs(
    receipt: &Receipt,
    cell: &Value,
    name: &str,
    seed: u64,
    axis: &str,
    references: &BTreeMap<(String, u64), &Value>,
    pids: &mut BTreeSet<u64>,
) -> Result<bool, String> {
    let runs = array(cell, "runs")?;
    let comparisons = array(cell, "comparisons")?;
    let count = receipt.controls.repetitions
        * if axis == "concurrent" {
            receipt.controls.parallel
        } else {
            1
        };
    require(
        cell["requested"] == count,
        "stress repetition/cohort count changed",
    )?;
    let excluded = !receipt.controls.selected(name, seed)
        || !receipt.controls.axis.is_empty() && receipt.controls.axis != axis;
    if excluded {
        require(
            cell["status"] == "skipped"
                && cell["completed"] == 0
                && runs.is_empty()
                && comparisons.is_empty()
                && (cell["reason"] == "excluded by --stress-only"
                    || cell["reason"] == "excluded by --stress-axis"),
            "excluded stress cell claims execution",
        )?;
        return Ok(false);
    }
    if axis == "handoff" && name != "two-cluster" {
        require(
            cell["status"] == "skipped"
                && cell["completed"] == 0
                && runs.is_empty()
                && comparisons.is_empty()
                && cell["reasons"]
                    == json!([
                        "inapplicable: no cluster baton or remote peer replies in a standalone guest"
                    ]),
            "standalone handoff applicability changed",
        )?;
        return Ok(true);
    }
    let reference = references
        .get(&(name.into(), seed))
        .ok_or("selected stress reference absent")?;
    require(
        cell["rounds"] == receipt.controls.repetitions
            && cell["fanout"]
                == if axis == "concurrent" {
                    receipt.controls.parallel
                } else {
                    1
                }
            && cell["completed"] == count
            && cell["status"] == "pass"
            && runs.len() == count
            && comparisons.len() == count
            && cell["reasons"] == json!([]),
        "short/divergent stress cell",
    )?;
    for (index, run) in runs.iter().enumerate() {
        observations(receipt, run, name, seed, axis, index, pids)?;
        let recomputed = crate::stress::compare_retained(reference, run)?;
        require(
            recomputed == comparisons[index]
                && recomputed
                    == json!({"schema":"gomad.wasm-first-divergence/v1","status":"identical"}),
            "native stress raw comparison differs",
        )?;
    }
    Ok(true)
}

fn matrix(receipt: &Receipt, report: &Value) -> Result<bool, String> {
    let mut references = BTreeMap::new();
    let mut pids = BTreeSet::new();
    for row in array(report, "references")? {
        let name = text(row, "workload")?;
        let seed = row["seed"].as_u64().ok_or("stress seed absent")?;
        require(
            receipt.controls.selected(name, seed)
                && cases().contains(&(name, seed))
                && references.insert((name.to_owned(), seed), row).is_none(),
            "extra/duplicate stress reference",
        )?;
        observations(receipt, row, name, seed, "reference", 0, &mut pids)?;
    }
    let cells = array(report, "cells")?;
    require(
        cells.len() == 164,
        "native stress lost original 41-by-four matrix",
    )?;
    let mut coordinates = BTreeSet::new();
    let mut complete = true;
    for cell in cells {
        let name = text(cell, "workload")?;
        let seed = cell["seed"].as_u64().ok_or("cell seed absent")?;
        let axis = text(cell, "axis")?;
        require(
            cases().contains(&(name, seed))
                && ["cpu", "concurrent", "layout", "handoff"].contains(&axis)
                && coordinates.insert((name, seed, axis)),
            "duplicate/foreign stress cell",
        )?;
        complete &= cell_runs(receipt, cell, name, seed, axis, &references, &mut pids)?;
    }
    Ok(complete)
}

fn producer_observation(
    receipt: &Receipt,
    process: &build::CommandObservation,
    report: &Value,
) -> Result<(), String> {
    let expected = vec![
        receipt.executable.to_string_lossy().into_owned(),
        "stress".into(),
        "--spec".into(),
        receipt.spec.to_string_lossy().into_owned(),
    ];
    require(
        process.command == expected
            && process.directory == receipt.repo
            && process.pid.is_some_and(|pid| pid > 0)
            && process.signal.is_none()
            && process.cleanup_errors.is_empty()
            && process.reaped_monotonic_nanos.is_some()
            && process.duration_nanos > 0,
        "stress producer lacks actual clean process observation",
    )?;
    for (capture, name) in [
        (&process.stdout, "stress-producer.stdout"),
        (&process.stderr, "stress-producer.stderr"),
    ] {
        require(
            capture.path == receipt.root.join(name)
                && !capture.truncated
                && capture.error.is_none()
                && capture.sha256.as_deref() == Some(digest(&capture.path)?.as_str()),
            "stress producer raw capture changed",
        )?;
    }
    let stdout = value(&process.stdout.path)?;
    require(
        stdout["report"] == receipt.report.to_string_lossy().as_ref()
            && stdout["passed"] == report["passed"]
            && report["schema"] == "gomad.rust-stress/v1"
            && report["source_sha256"] == receipt.source_sha256
            && report["binary_sha256"] == receipt.executable_sha256
            && report["command"] == json!(expected)
            && array(report, "rejected")?.is_empty(),
        "stress report/producer/candidate mismatch",
    )?;
    Ok(())
}

pub(super) fn check(
    receipt: &Receipt,
    process: &build::CommandObservation,
) -> Result<bool, String> {
    receipt.controls.validate()?;
    require(
        receipt.schema == RESULT_SCHEMA,
        "unknown native stress result",
    )?;
    for path in [
        &receipt.spec,
        &receipt.report,
        &receipt.source_manifest,
        &receipt.producer,
    ] {
        contained(&receipt.root, path)?;
    }
    require(
        digest(&receipt.spec)? == receipt.spec_sha256
            && digest(&receipt.report)? == receipt.report_sha256
            && digest(&receipt.source_manifest)? == receipt.source_sha256
            && digest(&receipt.executable)? == receipt.executable_sha256,
        "native stress receipt input changed",
    )?;
    crate::qualify::result::validate_source_manifest(
        &bytes(&receipt.source_manifest, 16 << 20)?,
        true,
    )?;
    let spec = value(&receipt.spec)?;
    let report = value(&receipt.report)?;
    require(
        spec["root"] == receipt.root.join("matrix").to_string_lossy().as_ref()
            && spec["requested_output_root"] == receipt.requested_root.to_string_lossy().as_ref()
            && spec["source_sha256"] == receipt.source_sha256
            && spec["source_manifest"] == receipt.source_manifest.to_string_lossy().as_ref()
            && spec["repetitions"] == receipt.controls.repetitions
            && spec["parallel"] == receipt.controls.parallel
            && spec["cpu_workers"] == receipt.controls.cpu_workers
            && spec["only"] == receipt.controls.only
            && spec["seed"] == receipt.controls.seed
            && spec["axis"] == receipt.controls.axis,
        "stress spec differs from original catalogue controls",
    )?;
    workload_recipes(receipt, &spec)?;
    producer_observation(receipt, process, &report)?;
    let complete = matrix(receipt, &report)?;
    require(
        report["passed"] == complete
            && process.exit_code == Some(i32::from(!complete))
            && process.termination == if complete { "exit" } else { "exit_failure" }
            && process.error.as_deref()
                == if complete {
                    None
                } else {
                    Some("native process exit=Some(1) signal=None")
                },
        "native stress completion/process status mismatch",
    )?;
    Ok(complete)
}

/// # Errors
/// Reopens actual native stress evidence without preparing or executing another guest.
pub fn read(path: &Path, proof: &CheckoutProof) -> Result<Value, String> {
    let raw = bytes(path, 16 << 20)?;
    let header: Value = crate::qualify::spec::strict(&raw)?;
    if header["schema"] == "gomad.native-stress-selection/v1" {
        return read_selection(path, proof, &header);
    }
    let receipt: Receipt = crate::qualify::spec::strict(&raw)?;
    require(
        receipt.repo == proof.repo() && path == receipt.root.join("result.json"),
        "native stress result belongs to foreign checkout/root",
    )?;
    let lease = ChildLease::for_path(&receipt.root)?
        .ok_or("native stress result lacks authenticated parent retention")?;
    require(
        lease.requested_root()
            == fs::canonicalize(&receipt.requested_root).map_err(|error| error.to_string())?,
        "native stress result requested root changed",
    )?;
    build::validate_identity(
        &receipt.identity,
        &build::Role::Stress,
        &receipt.executable_sha256,
    )?;
    let process: build::CommandObservation =
        crate::qualify::spec::strict(&bytes(&receipt.producer, 16 << 20)?)?;
    require(
        check(&receipt, &process)? == receipt.complete,
        "native stress result completion changed",
    )?;
    Ok(
        json!({"schema":RESULT_SCHEMA,"complete":receipt.complete,"report":receipt.report,"source_sha256":receipt.source_sha256,"requested_root":receipt.requested_root}),
    )
}

fn read_selection(path: &Path, proof: &CheckoutProof, report: &Value) -> Result<Value, String> {
    require(
        report["repo"] == proof.repo().to_string_lossy().as_ref()
            && report["executed"] == false
            && report["complete"] == false
            && report["references"] == json!([]),
        "excluded stress selection claims execution",
    )?;
    let root = Path::new(text(report, "root")?);
    require(
        path == root.join("selection.json"),
        "excluded stress selection outside its original stage",
    )?;
    let lease =
        ChildLease::for_path(root)?.ok_or("excluded stress selection lacks parent stage")?;
    require(
        fs::canonicalize(text(report, "requested_root")?).map_err(|error| error.to_string())?
            == lease.requested_root(),
        "excluded stress requested root changed",
    )?;
    let controls: Controls =
        serde_json::from_value(report["controls"].clone()).map_err(|error| error.to_string())?;
    controls.validate()?;
    require(
        !cases()
            .iter()
            .any(|(name, seed)| controls.selected(name, *seed)),
        "executed stress cases cannot become an excluded selection",
    )?;
    let manifest = Path::new(text(report, "source_manifest")?);
    contained(root, manifest)?;
    require(
        digest(manifest)? == text(report, "source_sha256")?,
        "excluded stress source identity changed",
    )?;
    crate::qualify::result::validate_source_manifest(&bytes(manifest, 16 << 20)?, true)?;
    let expected = cases().into_iter().flat_map(|(name, seed)| ["cpu", "concurrent", "layout", "handoff"].map(|axis|
        json!({"workload":name,"seed":seed,"axis":axis,"status":"skipped","reason":"excluded by --stress-only","requested":controls.repetitions*if axis=="concurrent"{controls.parallel}else{1},"completed":0,"runs":[],"comparisons":[]}))).collect::<Vec<_>>();
    require(
        report["cells"] == json!(expected),
        "excluded stress matrix changed or claims a positive cell",
    )?;
    Ok(report.clone())
}
