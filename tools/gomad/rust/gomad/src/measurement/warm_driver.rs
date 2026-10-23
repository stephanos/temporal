//! Actual fresh-process warm and terminal measurements over admitted native roles.
use super::{
    build, evidence, host,
    inputs::{CheckoutProof, Invocation},
    module::{self, Selection},
    warm,
    workloads::{self, RunSpec},
};
use crate::evidence::canonical;
use serde_json::{Value, json};
use std::{
    fs,
    io::{Read, Write},
    path::{Path, PathBuf},
};
const WORKFLOW: &str = "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$";
fn value<'a>(call: &'a Invocation, key: &str) -> &'a str {
    call.inputs.get(key).map_or("", String::as_str)
}
fn read(path: &Path, maximum: u64) -> Result<Vec<u8>, String> {
    let mut raw = Vec::new();
    fs::File::open(path)
        .map_err(|e| e.to_string())?
        .take(maximum + 1)
        .read_to_end(&mut raw)
        .map_err(|e| e.to_string())?;
    if raw.len() as u64 > maximum {
        return Err("warm input exceeds original bound".into());
    }
    Ok(raw)
}
fn write(path: &Path, value: &impl serde::Serialize) -> Result<(), String> {
    use std::os::unix::fs::OpenOptionsExt;
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| e.to_string())?;
    file.write_all(&canonical::encode(value)?)
        .and_then(|()| file.sync_all())
        .map_err(|e| e.to_string())
}
fn crc32_update(mut crc: u32, bytes: &[u8]) -> u32 {
    for byte in bytes {
        crc ^= u32::from(*byte);
        for _ in 0..8 {
            crc = (crc >> 1) ^ if crc & 1 == 1 { 0xedb8_8320 } else { 0 };
        }
    }
    crc
}
fn cache_sizes(directory: &Path, module: &str, helper: &str) -> Result<(u64, u64, Value), String> {
    use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
    let mut found = None;
    for entry in fs::read_dir(directory).map_err(|e| e.to_string())? {
        let path = entry.map_err(|e| e.to_string())?.path();
        if path.extension().and_then(|s| s.to_str()) != Some("gomad-cache-v3") {
            continue;
        }
        let metadata = fs::symlink_metadata(&path).map_err(|e| e.to_string())?;
        if !metadata.is_file() {
            return Err("warm cache entry must be a regular private file".into());
        }
        let mut file = fs::OpenOptions::new()
            .read(true)
            .custom_flags(
                i32::try_from(rustix::fs::OFlags::NOFOLLOW.bits()).map_err(|e| e.to_string())?,
            )
            .open(&path)
            .map_err(|e| e.to_string())?;
        let opened = file.metadata().map_err(|e| e.to_string())?;
        if opened.dev() != metadata.dev()
            || opened.ino() != metadata.ino()
            || opened.nlink() != 1
            || opened.mode() & 0o7777 != 0o600
            || opened.uid() != rustix::process::geteuid().as_raw()
        {
            return Err("warm artifact must be unchanged private owned cache data".into());
        }
        let mut header = [0u8; 24];
        file.read_exact(&mut header).map_err(|e| e.to_string())?;
        if &header[..8] != b"GOMADC03" {
            return Err("warm cache envelope version mismatch".into());
        }
        let manifest_bytes = u64::from(u32::from_le_bytes(header[8..12].try_into().unwrap()));
        let artifact_bytes = u64::from_le_bytes(header[12..20].try_into().unwrap());
        if manifest_bytes > 16 * 1024
            || artifact_bytes == 0
            || artifact_bytes > 1 << 30
            || 24 + manifest_bytes + artifact_bytes != metadata.len()
        {
            return Err("warm cache envelope size mismatch".into());
        }
        let mut raw = vec![0; usize::try_from(manifest_bytes).map_err(|error| error.to_string())?];
        file.read_exact(&mut raw).map_err(|e| e.to_string())?;
        let manifest: Value = crate::qualify::spec::strict(&raw)?;
        if path.file_name().and_then(|name| name.to_str())
            != Some(&format!("{}.gomad-cache-v3", canonical::digest(&raw)))
        {
            return Err("warm cache manifest address mismatch".into());
        }
        let mut checksum = u32::MAX;
        let mut remaining = artifact_bytes;
        let mut buffer = vec![0u8; 65536];
        while remaining > 0 {
            let count =
                usize::try_from(remaining.min(buffer.len() as u64)).map_err(|e| e.to_string())?;
            file.read_exact(&mut buffer[..count])
                .map_err(|e| e.to_string())?;
            checksum = crc32_update(checksum, &buffer[..count]);
            remaining -= count as u64;
        }
        if !checksum != u32::from_le_bytes(header[20..24].try_into().unwrap()) {
            return Err("warm artifact checksum mismatch".into());
        }
        let after = file.metadata().map_err(|e| e.to_string())?;
        if after.len() != opened.len()
            || after.mtime() != opened.mtime()
            || after.mtime_nsec() != opened.mtime_nsec()
            || after.ctime() != opened.ctime()
            || after.ctime_nsec() != opened.ctime_nsec()
        {
            return Err("warm cache artifact changed during inventory".into());
        }
        if manifest["module_sha256"] != module || manifest["helper_sha256"] != helper {
            continue;
        }
        if manifest["engine"] != "wasmtime-47.0.3" || manifest["version"] != 3 {
            return Err("warm cache manifest mismatch".into());
        }
        if found.is_some() {
            return Err("ambiguous warm cache artifact identity".into());
        }
        found = Some((
            artifact_bytes,
            metadata.len(),
            json!({"path":path,"manifest":manifest,"envelope_bytes":metadata.len(),"artifact_bytes":artifact_bytes}),
        ));
    }
    found.ok_or_else(|| "actual warm serialized artifact not retained".into())
}

#[expect(
    clippy::too_many_lines,
    reason = "Keep the full warm and terminal measurement sequence together."
)]
/// # Errors
/// Returns errors admitting inputs, executing the warm measurement sequence or retaining its evidence.
pub fn run(call: &Invocation, proof: &CheckoutProof, root: &Path) -> Result<PathBuf, String> {
    let terminal = call.name == "terminal-exit";
    for key in if terminal {
        vec![
            "GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT",
            "GOMAD_WASM_TERMINAL_EXIT_MODULE",
            "GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT",
        ]
    } else {
        vec![
            "GOMAD_WASM_WARM_HELPER_RECEIPT",
            "GOMAD_WASM_WARM_MODULE",
            "GOMAD_WASM_WARM_MODULE_RECEIPT",
            "GOMAD_WASM_WARM_CACHE",
        ]
    } {
        if !Path::new(value(call, key)).is_absolute() {
            return Err(format!("{key} must be an explicit absolute bound input"));
        }
    }
    let helper_receipt = value(
        call,
        if terminal {
            "GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT"
        } else {
            "GOMAD_WASM_WARM_HELPER_RECEIPT"
        },
    );
    if helper_receipt.is_empty() {
        return Err("native helper admission receipt required".into());
    }
    let expected = if terminal {
        build::Role::Release
    } else {
        build::Role::Warm
    };
    let admission = build::validate_admission(proof, Path::new(helper_receipt), &expected)?;
    if admission.executable != Path::new(value(call, "GOMAD_WASM_TEST_HELPER")) {
        return Err("warm helper input differs from admitted executable".into());
    }
    let executable = workloads::Executable {
        path: admission.executable.clone(),
        sha256: admission.executable_sha256.clone(),
        role: expected,
        receipt: Some(helper_receipt.into()),
    };
    let prepared = module::prepare(proof, call, &root.join("module"), &Selection::Workflow)?;
    module::authenticate_prepared(&prepared)?;
    let (module_key, receipt_key) = if terminal {
        (
            "GOMAD_WASM_TERMINAL_EXIT_MODULE",
            "GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT",
        )
    } else {
        ("GOMAD_WASM_WARM_MODULE", "GOMAD_WASM_WARM_MODULE_RECEIPT")
    };
    // Supplied artifacts must be the actual current producer's module and full provenance.
    for (key, current, maximum) in [
        (module_key, &prepared.module_path, 512 << 20),
        (receipt_key, &prepared.provenance_path, 16 << 20),
    ] {
        let supplied = value(call, key);
        if !supplied.is_empty() && read(Path::new(supplied), maximum)? != read(current, maximum)? {
            return Err(format!(
                "{key} differs from the actual current module producer"
            ));
        }
    }
    let snapshot = host::source_snapshot(proof, &root.join("source"), &call.environ)?;
    let host = host::current(&root.join("host"), &call.environ)?;
    write(&root.join("source.json"), &snapshot)?;
    write(&root.join("host.json"), &host)?;
    let mut input = prepared.native_input(0)?;
    let cache_given = value(call, "GOMAD_WASM_WARM_CACHE");
    let cache = if cache_given.is_empty() {
        root.join("compile-cache")
    } else {
        PathBuf::from(cache_given)
    };
    if !terminal {
        if !cache.is_absolute() {
            return Err("absolute warm cache required".into());
        }
        input["compile_cache"] = json!(cache);
    }
    let module_sha = canonical::digest(&prepared.guest.module);
    let mut row = RunSpec::new("prime", WORKFLOW, 0, "diagnostic");
    row.warm = false;
    let prime = workloads::execute(call, proof, root, &row, &executable, &input)?;
    if prime.root.outcome["termination"] != "exit" || prime.root.outcome["exit_code"] != 0 {
        return Err("native priming workload did not exit successfully".into());
    }
    if terminal {
        let mut observations = vec![prime];
        for index in 0..5 {
            row.name = format!("terminal-{index}");
            observations.push(workloads::execute(
                call,
                proof,
                root,
                &row,
                &executable,
                &input,
            )?);
        }
        let mut process_exit_observations = Vec::new();
        for (index, run) in observations.iter().enumerate() {
            if run.root.outcome["termination"] != "exit"
                || run.root.outcome["exit_code"] != 0
                || run.report["terminal_to_return_nanos"]
                    .as_u64()
                    .is_none_or(|v| v > 1_000_000_000)
            {
                return Err(
                    "native terminal-to-return bound exceeds one second or observation absent"
                        .into(),
                );
            }
            evidence::compare(&observations[0], run, evidence::ComparisonMask::Exact)?;
            let label = if index == 0 {
                "prime".into()
            } else {
                format!("terminal-{}", index - 1)
            };
            process_exit_observations.push(process_exit_interval(root, &label, &run.report)?);
        }
        let receipt = value(call, "GOMAD_WASM_TERMINAL_EXIT_DIAGNOSTIC_HELPER_RECEIPT");
        let diagnostic_path = value(call, "GOMAD_WASM_TERMINAL_EXIT_DIAGNOSTIC_HELPER");
        if !Path::new(receipt).is_absolute() || !Path::new(diagnostic_path).is_absolute() {
            return Err(
                "terminal refusal controls require explicit admitted diagnostic helper and receipt"
                    .into(),
            );
        }
        let diagnostic_role = build::Role::Diagnostic(vec!["diagnostic-profile".into()]);
        let diagnostic_admission =
            build::validate_admission(proof, Path::new(receipt), &diagnostic_role)?;
        if diagnostic_admission.executable != Path::new(diagnostic_path) {
            return Err("terminal diagnostic helper differs from admitted executable".into());
        }
        let diagnostic = workloads::Executable {
            path: diagnostic_admission.executable.clone(),
            sha256: diagnostic_admission.executable_sha256.clone(),
            role: diagnostic_role,
            receipt: Some(receipt.into()),
        };
        let refusals = admission_refusals(
            call,
            proof,
            root,
            &diagnostic,
            &input,
            &prepared.module_metadata,
        )?;
        let path = root.join("terminal-exit.json");
        write(
            &path,
            &json!({"schema":"gomad.native-terminal-exit/v2","admission_refusals":refusals,"process_exit_observations":process_exit_observations,"diagnostic_control_admission":diagnostic_admission,"bound_nanos":1_000_000_000u64,"boundary":"actual classified native Outcome through completed linked Store/engine/watchdog cleanup; child supervisor wait retained separately","observations":observations,"helper_admission":admission,"module_provenance":prepared.provenance_path}),
        )?;
        return Ok(path);
    }
    let (artifact_bytes, envelope_bytes, artifact) =
        cache_sizes(&cache, &module_sha, &executable.sha256)?;
    write(&root.join("cache-artifact.json"), &artifact)?;
    if call.name == "warm-preparation-prime" {
        let path = root.join("warm-prime.json");
        write(
            &path,
            &json!({"schema":"gomad.native-warm-prime/v1","acceptance_sample":false,"artifact":artifact,"run":prime,"helper_admission":helper_receipt,"module_provenance":prepared.provenance_path}),
        )?;
        return Ok(path);
    }
    let binding = warm::Binding {
        source_commit: snapshot.source_commit,
        source_root: proof.repo().to_string_lossy().into_owned(),
        source_inputs: snapshot
            .source_inputs
            .into_iter()
            .map(|(p, h)| (p.to_string_lossy().into_owned(), h))
            .collect(),
        driver_sha256: canonical::digest(&read(
            &proof
                .repo()
                .join("tools/gomad/rust/gomad/src/measurement/warm_driver.rs"),
            1 << 20,
        )?),
        host_sha256: executable.sha256.clone(),
        module_sha256: module_sha,
        config_sha256: warm::retained_configuration_sha256(&input)?,
        model_sha256: canonical::digest(&canonical::encode(
            &gomad_wasmhost::linked::SOURCE_INPUTS
                .lines()
                .filter_map(|line| line.split_once('\t'))
                .filter(|(_, name)| {
                    name.starts_with("src/model") || name.starts_with("src/namespace")
                })
                .map(|(digest, name)| (name, digest))
                .collect::<std::collections::BTreeMap<_, _>>(),
        )?),
        runtime_identity: prepared
            .guest
            .choice_identity
            .as_ref()
            .ok_or("actual cooperative choice identity absent")?
            .toolchain_build_key
            .clone(),
        host_boot_id: host.boot_id,
        host_cpu: host.witness.cpuinfo,
        host_os: host.host_os,
        host_arch: host.host_arch,
        engine_version: "47.0.3".into(),
        engine_configuration: admission.identity.configuration,
        module_producer_sha256: canonical::digest(&prepared.guest.provenance),
        seed: 0,
        artifact_bytes,
        envelope_bytes,
    };
    let mut primed = Some(prime);
    let series = warm::collect_retained(binding, warm::Role::Candidate, 5, |index| {
        if index == 0 {
            return primed.take().ok_or_else(|| "warmup consumed twice".into());
        }
        row.name = format!("warm-{index}");
        workloads::execute(call, proof, root, &row, &executable, &input)
    })?;
    let path = root.join(if call.name == "warm-preparation-prime" {
        "warm-prime.json"
    } else {
        "warm-preparation.json"
    });
    let baseline_path = value(call, "GOMAD_WASM_WARM_BASELINE_RECEIPT");
    let comparison = (|| -> Result<Value, String> {
        if !Path::new(baseline_path).is_absolute() {
            return Err("baseline-material-gap: explicit authenticated native historical baseline receipt required".into());
        }
        let descriptor: Value =
            crate::qualify::spec::strict(&read(Path::new(baseline_path), 8 << 20)?)?;
        if descriptor["schema"] != "gomad.native-warm-baseline/v1" {
            return Err("native warm historical baseline schema mismatch".into());
        }
        let binding: warm::Binding =
            serde_json::from_value(descriptor["binding"].clone()).map_err(|e| e.to_string())?;
        let receipt = Path::new(
            descriptor["helper_admission"]
                .as_str()
                .ok_or("native baseline helper receipt absent")?,
        );
        let historical = build::admit_historical(
            Path::new(&binding.source_root),
            receipt,
            &build::Role::Warm,
            &root.join("historical-admission"),
            &call.environ,
        )?;
        if historical.admission.executable_sha256 != binding.host_sha256 {
            return Err("native baseline helper differs from immutable source admission".into());
        }
        for (path, digest) in &binding.source_inputs {
            if canonical::digest(&read(Path::new(path), 128 << 20)?) != *digest {
                return Err("native baseline source inventory changed".into());
            }
        }
        let runs = descriptor["runs"]
            .as_array()
            .filter(|runs| runs.len() == 6)
            .ok_or("native baseline needs warmup and five raw native run roots")?;
        let before = warm::collect_retained(binding, warm::Role::Baseline, 5, |index| {
            let path = Path::new(
                runs[index]
                    .as_str()
                    .ok_or("native baseline raw root absent")?,
            );
            evidence::read_profile_run(path, evidence::ReadLimits::default())
        })?;
        historical.revalidate()?;
        warm::compare(&before, &series)?;
        Ok(
            json!({"receipt":baseline_path,"receipt_sha256":canonical::digest(&read(Path::new(baseline_path),8<<20)?),"series":before}),
        )
    })();
    write(
        &path,
        &json!({"schema":"gomad.native-warm-measurement/v1","accepted":comparison.is_ok(),"baseline":comparison.as_ref().ok(),"failure":comparison.as_ref().err(),"series":series,"artifact":artifact,"helper_admission":helper_receipt,"module_provenance":prepared.provenance_path,"qualification":"source producer only; frozen native execution qualification remains separate"}),
    )?;
    comparison?;
    Ok(path)
}

#[expect(
    clippy::too_many_lines,
    reason = "Keep both actual admission controls and their raw phase/timing checks in one producer transaction."
)]
fn admission_refusals(
    call: &Invocation,
    proof: &CheckoutProof,
    root: &Path,
    executable: &workloads::Executable,
    input: &Value,
    metadata: &Value,
) -> Result<Vec<Value>, String> {
    if metadata["initial_memory_pages"]
        .as_u64()
        .is_none_or(|pages| pages <= 1)
    {
        return Err("loaded-module descriptor refusal requires the actual module to declare more than one initial memory page".into());
    }
    let mut observations = Vec::new();
    for stage in ["module-digest", "loaded-module-descriptor"] {
        let mut controlled = input.clone();
        controlled
            .as_object_mut()
            .ok_or("native request object absent")?
            .remove("compile_cache");
        if stage == "module-digest" {
            let expected = controlled["guest"]["module_sha256"]
                .as_str()
                .ok_or("module digest absent")?;
            controlled["guest"]["module_sha256"] = json!(if expected == "0".repeat(64) {
                "1".repeat(64)
            } else {
                "0".repeat(64)
            });
        } else {
            controlled["guest"]["limits"]["memory_bytes"] = json!(65536);
        }
        let mut row = RunSpec::new(stage, WORKFLOW, 0, "diagnostic");
        row.warm = false;
        let _attempt = workloads::execute(call, proof, root, &row, executable, &controlled);
        executable.verify(proof)?;
        let native_root = root.join("evidence").join(stage);
        let raw_request = read(&native_root.join("request.json"), 8 << 20)?;
        if crate::qualify::spec::strict::<Value>(&raw_request)? != controlled {
            return Err("admission control changed its retained actual request".into());
        }
        let report: Value = crate::qualify::spec::strict::<Value>(&read(
            &native_root.join("report.json"),
            8 << 20,
        )?)?;
        let profile = crate::qualify::spec::strict::<Value>(&read(
            &native_root.join("profile.json"),
            48 << 10,
        )?)?;
        evidence::validate_admission_profile(&profile)?;
        if report["schema"] != "gomad.linked-profile-run/v1"
            || report["host_sha256"] != executable.sha256
            || report["request_sha256"] != canonical::digest(&raw_request)
            || report["features"] != json!(executable.role.features()?)
            || report["diagnostic_only"] != true
        {
            return Err("admission control lost its actual helper/request/profile binding".into());
        }
        let refusal = &report["admission_refusal"];
        if refusal["schema"] != "gomad.native-admission-refusal/v1"
            || refusal["clock"] != "monotonic"
            || refusal["terminal_to_return_nanos"]
                .as_u64()
                .is_none_or(|nanos| nanos == 0 || nanos > 1_000_000_000)
            || ["instantiate", "execute"]
                .into_iter()
                .any(|phase| profile["phases"][phase]["count"] != 0)
        {
            return Err(
                "actual admission refusal exceeded cleanup bound or entered guest execution".into(),
            );
        }
        if stage == "module-digest" {
            if refusal["stage"] != "request-admission"
                || refusal["error"] != "module SHA-256 mismatch"
                || report["execution"]["Err"] != "module SHA-256 mismatch"
                || ["engine_init", "compile", "deserialize"]
                    .into_iter()
                    .any(|phase| profile["phases"][phase]["count"] != 0)
                || native_root.join("outcome.json").exists()
            {
                return Err("actual digest control was not refused before engine creation".into());
            }
        } else {
            let outcome = &refusal["outcome"];
            if refusal["stage"] != "loaded-module-descriptor"
                || outcome["termination"] != "capacity"
                || outcome["message"] != "initial guest memory exceeds limit"
                || outcome["metrics"]["prepare_nanos"]
                    .as_u64()
                    .is_none_or(|nanos| nanos == 0)
                || outcome["metrics"]["compile_nanos"]
                    .as_u64()
                    .is_none_or(|nanos| nanos == 0)
                || outcome["metrics"]["instantiate_nanos"] != 0
                || outcome["metrics"]["execute_nanos"] != 0
                || profile["phases"]["compile"]["count"] != 1
                || crate::qualify::spec::strict::<Value>(&read(
                    &native_root.join("outcome.json"),
                    8 << 20,
                )?)? != *outcome
            {
                return Err(
                    "actual loaded module descriptor control was not refused before instantiation"
                        .into(),
                );
            }
        }
        let process_exit = process_exit_interval(root, stage, &report)?;
        observations.push(json!({"stage":stage,"native_root":native_root,
            "request_sha256":report["request_sha256"],"refusal":refusal,"profile":profile,"process_exit":process_exit,
            "terminal_to_return_nanos":refusal["terminal_to_return_nanos"]}));
    }
    Ok(observations)
}

pub(super) fn process_exit_interval(
    root: &Path,
    label: &str,
    report: &Value,
) -> Result<Value, String> {
    let path = root.join("evidence").join(format!("{label}-driver.json"));
    let command: build::CommandObservation = crate::qualify::spec::strict(&read(&path, 8 << 20)?)?;
    let terminal = report["terminal_monotonic_nanos"]
        .as_u64()
        .ok_or("native terminal kernel timestamp absent")?;
    let reaped = command
        .reaped_monotonic_nanos
        .ok_or("actual child reap kernel timestamp absent")?;
    let elapsed = reaped.checked_sub(terminal).filter(|nanos| *nanos > 0 && *nanos <= 1_000_000_000)
        .ok_or("actual terminal-to-process-exit interval exceeds original one-second bound or clocks reversed")?;
    if report["terminal_clock"] != "CLOCK_MONOTONIC"
        || command.pid.map(u64::from) != report["fresh_store_id"].as_u64()
        || !command.cleanup_errors.is_empty()
        || !matches!(command.termination.as_str(), "exit" | "exit_failure")
        || command.signal.is_some()
    {
        return Err("native terminal-to-reap clock/process/cleanup binding changed".into());
    }
    Ok(
        json!({"schema":"gomad.native-terminal-process-exit/v1","sample":label,
        "clock":"CLOCK_MONOTONIC","pid":command.pid,"driver_observation":path,
        "terminal_monotonic_nanos":terminal,"reaped_monotonic_nanos":reaped,
        "terminal_to_reap_nanos":elapsed,"bound_nanos":1_000_000_000u64,
        "boundary":"actual child terminal classification through observed child exit/reap, including raw report/stream/profile publication"}),
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn artifact_checksum_matches_native_cache_crc32() {
        assert_eq!(!crc32_update(u32::MAX, b"123456789"), 0xcbf4_3926);
        let partial = crc32_update(u32::MAX, b"1234");
        assert_eq!(!crc32_update(partial, b"56789"), 0xcbf4_3926);
        assert_ne!(!crc32_update(u32::MAX, b"123456788"), 0xcbf4_3926);
    }
}
