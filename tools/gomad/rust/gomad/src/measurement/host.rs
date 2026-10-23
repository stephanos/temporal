use super::{
    build,
    inputs::{self, CheckoutProof},
};
use crate::evidence::canonical;
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    fs,
    io::Read,
    path::{Path, PathBuf},
    time::Duration,
};
pub const CPUINFO_LIMIT: u64 = 1 << 20;
const MEMINFO_LIMIT: u64 = 8 << 10;
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct HostWitness {
    pub kernel_os: String,
    pub machine: String,
    pub kernel_version: String,
    pub cpuinfo: String,
    pub translation: String,
}
#[derive(Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct HostEvidence {
    pub schema: String,
    pub host_os: String,
    pub host_arch: String,
    pub boot_id: String,
    pub witness: HostWitness,
    pub inputs: BTreeMap<PathBuf, String>,
}
fn os(value: &str) -> &str {
    match value {
        "darwin" | "Darwin" | "macos" => "macos",
        "Linux" => "linux",
        other => other,
    }
}
#[must_use]
pub fn architecture(value: &str) -> &str {
    match value.trim() {
        "arm64" | "aarch64" => "aarch64",
        "amd64" | "x86_64" => "x86_64",
        other => other,
    }
}
#[must_use]
pub fn performance_host(host_os: &str, host_arch: &str, cpuinfo: &str) -> bool {
    if cpuinfo.is_empty() || cpuinfo.len() as u64 > CPUINFO_LIMIT {
        return false;
    }
    if let Ok(witness) = serde_json::from_str::<HostWitness>(cpuinfo) {
        return host_check(host_os, host_arch, &witness);
    }
    if os(host_os) != "linux" {
        return false;
    }
    let mut arm = false;
    let mut simd = false;
    let mut vendor = false;
    for line in cpuinfo.lines() {
        if let Some((key, value)) = line.split_once(':') {
            match key.trim() {
                "CPU architecture" => arm |= value.trim() == "8",
                "Features" => simd |= value.split_whitespace().any(|value| value == "asimd"),
                "vendor_id" => vendor |= !value.trim().is_empty(),
                _ => {}
            }
        }
    }
    match architecture(host_arch) {
        "aarch64" => arm && simd,
        "x86_64" => vendor,
        _ => false,
    }
}
#[must_use]
pub fn host_check(host_os: &str, host_arch: &str, witness: &HostWitness) -> bool {
    if witness.kernel_version.is_empty()
        || witness.cpuinfo.is_empty()
        || witness.cpuinfo.len() as u64 > CPUINFO_LIMIT
        || (!witness.translation.is_empty() && witness.translation != "0")
        || architecture(&witness.machine) != architecture(host_arch)
        || os(&witness.kernel_os) != os(host_os)
    {
        return false;
    }
    match (os(host_os), architecture(host_arch)) {
        ("linux", arch) => performance_host("linux", arch, &witness.cpuinfo),
        ("macos", "aarch64") => witness
            .cpuinfo
            .lines()
            .any(|line| line == "hw.optional.arm64: 1"),
        ("macos", "x86_64") => witness
            .cpuinfo
            .lines()
            .any(|line| line == "hw.optional.arm64: 0"),
        _ => false,
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
        return Err(format!(
            "host evidence exceeds byte bound: {}",
            path.display()
        ));
    }
    Ok(bytes)
}
#[must_use]
pub fn performance_host_evidence(path: &Path, host_os: &str, host_arch: &str) -> (String, bool) {
    let Ok(bytes) = read(path, CPUINFO_LIMIT) else {
        return (String::new(), false);
    };
    let Ok(cpuinfo) = String::from_utf8(bytes) else {
        return (String::new(), false);
    };
    let admitted = performance_host(host_os, host_arch, &cpuinfo);
    (cpuinfo, admitted)
}
fn environment(environ: &[String]) -> BTreeMap<String, String> {
    environ
        .iter()
        .filter_map(|entry| entry.split_once('='))
        .filter(|(name, _)| matches!(*name, "PATH" | "HOME" | "TMPDIR"))
        .map(|(key, value)| (key.into(), value.into()))
        .collect()
}
fn probe(
    root: &Path,
    label: &str,
    environment: &BTreeMap<String, String>,
    argv: &[&str],
) -> Result<String, String> {
    let bytes = build::command_bounded(
        root,
        label,
        root,
        environment,
        &argv
            .iter()
            .map(|value| (*value).to_owned())
            .collect::<Vec<_>>(),
        Duration::from_secs(10),
        CPUINFO_LIMIT,
    )?;
    Ok(String::from_utf8(bytes)
        .map_err(|error| error.to_string())?
        .trim()
        .into())
}
fn save(
    root: &Path,
    name: &str,
    bytes: &[u8],
    inputs: &mut BTreeMap<PathBuf, String>,
) -> Result<(), String> {
    use std::io::Write;
    use std::os::unix::fs::OpenOptionsExt;
    let path = root.join(name);
    let mut file = fs::OpenOptions::new()
        .create_new(true)
        .write(true)
        .mode(0o600)
        .open(&path)
        .map_err(|error| error.to_string())?;
    file.write_all(bytes).map_err(|error| error.to_string())?;
    file.sync_all().map_err(|error| error.to_string())?;
    inputs.insert(path, canonical::digest(bytes));
    Ok(())
}
/// # Errors
/// Returns errors collecting and retaining the current host identity and witness.
pub fn current(root: &Path, environ: &[String]) -> Result<HostEvidence, String> {
    if !root.is_absolute() {
        return Err("absolute host evidence root required".into());
    }
    fs::create_dir_all(root).map_err(|error| error.to_string())?;
    let environment = environment(environ);
    let mut witness = HostWitness {
        kernel_os: probe(root, "uname-os", &environment, &["uname", "-s"])?,
        machine: probe(root, "uname-machine", &environment, &["uname", "-m"])?,
        kernel_version: probe(root, "uname-kernel", &environment, &["uname", "-v"])?,
        cpuinfo: String::new(),
        translation: String::new(),
    };
    let mut inputs = BTreeMap::new();
    let boot_id = match os(std::env::consts::OS) {
        "linux" => {
            let boot = read(Path::new("/proc/sys/kernel/random/boot_id"), 128)?;
            let cpu = read(Path::new("/proc/cpuinfo"), CPUINFO_LIMIT)?;
            save(root, "host-boot-id.txt", &boot, &mut inputs)?;
            save(root, "host-cpuinfo.txt", &cpu, &mut inputs)?;
            witness.cpuinfo = String::from_utf8(cpu).map_err(|error| error.to_string())?;
            String::from_utf8(boot)
                .map_err(|error| error.to_string())?
                .trim()
                .into()
        }
        "macos" => {
            let boot = probe(
                root,
                "host-boot",
                &environment,
                &["sysctl", "-n", "kern.boottime"],
            )?;
            let arm = probe(
                root,
                "host-arm",
                &environment,
                &["sysctl", "-n", "hw.optional.arm64"],
            )?;
            let brand = probe(
                root,
                "host-brand",
                &environment,
                &["sysctl", "-n", "machdep.cpu.brand_string"],
            )?;
            witness.translation = probe(
                root,
                "host-translation",
                &environment,
                &["sysctl", "-in", "sysctl.proc_translated"],
            )?;
            witness.cpuinfo =
                format!("hw.optional.arm64: {arm}\nmachdep.cpu.brand_string: {brand}\n");
            boot
        }
        _ => {
            return Err(
                "native host evidence requires available procfs or Darwin sysctl witnesses".into(),
            );
        }
    };
    if boot_id.is_empty() || !host_check(std::env::consts::OS, std::env::consts::ARCH, &witness) {
        return Err("host evidence does not bind the current native engine host".into());
    }
    save(
        root,
        "host-witness.json",
        &serde_json::to_vec(&witness).map_err(|error| error.to_string())?,
        &mut inputs,
    )?;
    Ok(HostEvidence {
        schema: "gomad.native-measurement-host/v1".into(),
        host_os: os(std::env::consts::OS).into(),
        host_arch: architecture(std::env::consts::ARCH).into(),
        boot_id,
        witness,
        inputs,
    })
}
#[derive(Debug, Serialize)]
pub struct SourceSnapshot {
    pub schema: &'static str,
    pub source_commit: String,
    pub source_dirty: bool,
    pub status: String,
    pub source_inputs: BTreeMap<PathBuf, String>,
}
/// # Errors
/// Returns errors collecting the checkout, toolchain and host source bindings.
pub fn source_snapshot(
    proof: &CheckoutProof,
    root: &Path,
    environ: &[String],
) -> Result<SourceSnapshot, String> {
    fs::create_dir_all(root).map_err(|error| error.to_string())?;
    let env = environment(environ);
    let repo = proof
        .repo()
        .to_str()
        .ok_or("UTF-8 selected checkout required")?;
    let commit = probe(
        root,
        "source-commit",
        &env,
        &["git", "-C", repo, "rev-parse", "HEAD"],
    )?;
    if commit.len() != 40 || !commit.bytes().all(|byte| byte.is_ascii_hexdigit()) {
        return Err("source commit identity required".into());
    }
    let status = probe(
        root,
        "source-status",
        &env,
        &[
            "git",
            "-C",
            repo,
            "status",
            "--porcelain=v1",
            "--untracked-files=all",
        ],
    )?;
    let mut source_inputs = BTreeMap::new();
    for (base, manifest) in [
        (
            proof.repo().join("tools/gomad/rust/gomad"),
            inputs::SOURCE_INPUTS,
        ),
        (
            proof.repo().join("tools/gomad/rust/wasmhost"),
            gomad_wasmhost::linked::SOURCE_INPUTS,
        ),
    ] {
        inputs::authenticate_source_closure(&base, manifest)?;
        for line in manifest.lines() {
            let (digest, name) = line
                .split_once('\t')
                .ok_or("malformed native source manifest")?;
            source_inputs.insert(base.join(name), digest.into());
        }
    }
    Ok(SourceSnapshot {
        schema: "gomad.native-measurement-source/v1",
        source_commit: commit,
        source_dirty: !status.is_empty(),
        status,
        source_inputs,
    })
}
#[cfg(test)]
#[path = "host_tests.rs"]
mod tests;

fn memory_decimal(value: &str) -> Result<u64, String> {
    if value.is_empty() || value.len() > 20 || !value.bytes().all(|byte| byte.is_ascii_digit()) {
        return Err("bounded memory decimal digits required".into());
    }
    value
        .parse()
        .map_err(|error: std::num::ParseIntError| error.to_string())
}

/// # Errors
/// Refuses missing, malformed, duplicate or overflowing available-memory values.
pub fn linux_available_memory(raw: &str) -> Result<u64, String> {
    if raw.len() as u64 > MEMINFO_LIMIT {
        return Err("MemAvailable host witness exceeds byte bound".into());
    }
    let mut available = None;
    for line in raw.split('\n') {
        let fields = line.split_whitespace().collect::<Vec<_>>();
        let Some(first) = fields.first() else {
            continue;
        };
        let key = first.split_once(':').map_or(*first, |(key, _)| key);
        if !key.eq_ignore_ascii_case("MemAvailable") {
            continue;
        }
        if available.is_some()
            || *first != "MemAvailable:"
            || fields.len() != 3
            || fields[2] != "kB"
        {
            return Err("unique exact MemAvailable field in kB required".into());
        }
        available = Some(
            memory_decimal(fields[1])?
                .checked_mul(1024)
                .ok_or("MemAvailable byte overflow")?,
        );
    }
    available.ok_or_else(|| "MemAvailable host witness required".into())
}
/// # Errors
/// Refuses missing, malformed, duplicate or overflowing available-page values.
pub fn darwin_available_memory(raw: &str) -> Result<u64, String> {
    if raw.len() as u64 > CPUINFO_LIMIT {
        return Err("vm_stat host witness exceeds byte bound".into());
    }
    let (header, rest) = raw.split_once('\n').ok_or("vm_stat header required")?;
    let page_size = memory_decimal(
        header
            .strip_prefix("Mach Virtual Memory Statistics: (page size of ")
            .and_then(|header| header.strip_suffix(" bytes)"))
            .ok_or("vm_stat page size required")?,
    )?;
    if page_size == 0 {
        return Err("vm_stat positive page size required".into());
    }
    let mut values = BTreeMap::new();
    for line in rest.split('\n') {
        let (key, value) = line.split_once(':').unwrap_or((line, ""));
        if !matches!(key, "Pages free" | "Pages inactive") {
            continue;
        }
        let count = memory_decimal(value.trim().strip_suffix('.').unwrap_or(value.trim()))?;
        if values.insert(key, count).is_some() {
            return Err("unique vm_stat available page counts required".into());
        }
    }
    if values.len() != 2 {
        return Err("vm_stat free and inactive page counts required".into());
    }
    values["Pages free"]
        .checked_add(values["Pages inactive"])
        .and_then(|pages| pages.checked_mul(page_size))
        .ok_or_else(|| "vm_stat available memory overflow".into())
}
fn memory_witness(
    root: &Path,
    label: &str,
    platform: &str,
    environ: &[String],
) -> Result<String, String> {
    match os(platform) {
        "linux" => {
            let bytes = read(Path::new("/proc/meminfo"), MEMINFO_LIMIT)?;
            let mut inputs = BTreeMap::new();
            save(root, &format!("{label}-meminfo.txt"), &bytes, &mut inputs)?;
            String::from_utf8(bytes).map_err(|error| error.to_string())
        }
        "macos" => probe(
            root,
            &format!("{label}-vm-stat"),
            &environment(environ),
            &["vm_stat"],
        ),
        _ => Err("cold-memory guard requires actual procfs or Darwin vm_stat witness".into()),
    }
}

fn preparation_admitted<T>(
    platform: &str,
    witness: impl FnOnce() -> Result<String, String>,
    dispatch: impl FnOnce() -> Result<T, String>,
) -> Result<T, String> {
    let available = match os(platform) {
        "linux" => linux_available_memory(&witness()?)?,
        "macos" => darwin_available_memory(&witness()?)?,
        _ => {
            return Err(
                "cold-memory guard requires actual procfs or Darwin vm_stat witness".into(),
            );
        }
    };
    if available < 8_u64 << 30 {
        return Err(
            "at least 8GiB available memory required before matched preparation dispatch".into(),
        );
    }
    dispatch()
}

fn matched_preparation_admitted<T>(
    platform: &str,
    mode: &str,
    witness: impl FnOnce() -> Result<String, String>,
    dispatch: impl FnOnce() -> Result<T, String>,
) -> Result<T, String> {
    if mode.is_empty() {
        return dispatch();
    }
    if mode != "1" || platform != "linux" {
        return Err("explicit matched memory admission requires mode 1 on Linux".into());
    }
    preparation_admitted(platform, witness, dispatch)
}

/// # Errors
/// Refuses unsupported preparation modes, insufficient memory or unavailable host evidence.
pub fn admit_preparation(
    root: &Path,
    label: &str,
    mode: &str,
    environ: &[String],
) -> Result<(), String> {
    matched_preparation_admitted(
        std::env::consts::OS,
        mode,
        || memory_witness(root, label, std::env::consts::OS, environ),
        || Ok(()),
    )
}

pub(super) fn with_warm_preparation<T>(
    root: &Path,
    label: &str,
    environ: &[String],
    dispatch: impl FnOnce() -> Result<T, String>,
) -> Result<T, String> {
    preparation_admitted(
        std::env::consts::OS,
        || memory_witness(root, label, std::env::consts::OS, environ),
        dispatch,
    )
}
