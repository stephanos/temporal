mod decode;
#[cfg(test)]
mod preservation_tests;

use gomad_wasmhost::{linked, linked_cluster};
use serde_json::Value;
use sha2::Digest as _;
use std::{collections::BTreeMap, path::Path};

pub(crate) struct Configuration {
    pub original: Value,
    pub model: Value,
    pub namespace: linked::NamespaceImage,
    pub storage: Option<linked::StorageInput>,
    pub network: Option<linked_cluster::NetworkModel>,
    pub build_tags: Vec<String>,
    pub environment: BTreeMap<String, String>,
    pub arguments: Vec<String>,
}

pub(crate) fn build(
    profile: &str,
    config_path: Option<&Path>,
    temporal: Option<(&Path, &str)>,
) -> Result<Configuration, String> {
    if config_path.is_some() && temporal.is_some() {
        return Err("--config cannot be combined with Temporal setup".into());
    }
    let original = if let Some(path) = config_path {
        read_config(&crate::qualify::spec::read_bounded(path, 16 << 20)?)?
    } else if let Some((root, package)) = temporal {
        temporal_policy(root, package, TemporalPolicy::Installation)?
    } else {
        default_value(profile)
    };
    let mut configured = convert(original, profile)?;
    if temporal.is_some() {
        configured.build_tags = temporal_build_tags();
    }
    Ok(configured)
}

#[derive(serde::Deserialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
struct Guest {
    profile: String,
    #[serde(default)]
    args: Option<Vec<String>>,
    #[serde(default)]
    environment: Option<Vec<String>>,
    working_directory: String,
    #[serde(default)]
    writable_directories: Option<Vec<String>>,
    #[serde(default)]
    captured_inputs: Captured,
    #[serde(default)]
    stdin: Option<String>,
    #[serde(default)]
    entropy_key: [u8; 32],
    #[serde(default)]
    clock: Clock,
    limits: Limits,
    #[serde(default, rename = "storage")]
    storage: Option<Value>,
    #[serde(default, rename = "network_v2")]
    network: Option<linked_cluster::NetworkModel>,
}
#[derive(Default, serde::Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
struct Clock {
    epoch_nanos: u64,
    read_step_nanos: u64,
}
#[derive(serde::Deserialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
struct Limits {
    output_bytes: u64,
    transcript_bytes: u64,
    calls: u64,
    descriptors: u64,
    files: u64,
    filesystem_bytes: u64,
    pending_events: u64,
}
#[derive(Default, serde::Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
struct Captured {
    manifest: Manifest,
    descriptor: Option<String>,
    payloads: Option<BTreeMap<String, String>>,
}
#[derive(Default, serde::Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
struct Manifest {
    bytes: u64,
    entries: u64,
    file: String,
    limits: CaptureLimits,
    mappings: Option<Vec<String>>,
    not_exist: u64,
    schema: String,
    #[serde(rename = "SHA256")]
    sha256: String,
    total_bytes: u64,
}
#[derive(Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
struct CaptureLimits {
    directory_entries: u64,
    files: u64,
    path_bytes: u64,
    requests: u64,
    single_file_bytes: u64,
    total_bytes: u64,
}
#[derive(serde::Deserialize)]
#[serde(deny_unknown_fields)]
struct Descriptor {
    entries: Vec<CapturedEntry>,
    limits: DescriptorLimits,
    mappings: Vec<String>,
    #[serde(default)]
    not_exist: Vec<String>,
    requests: String,
    schema: String,
    total_bytes: String,
}
#[derive(serde::Deserialize)]
#[serde(deny_unknown_fields)]
struct DescriptorLimits {
    directory_entries: String,
    files: String,
    path_bytes: String,
    requests: String,
    single_file_bytes: String,
    total_bytes: String,
}
#[derive(serde::Deserialize)]
#[serde(deny_unknown_fields)]
struct CapturedEntry {
    children: Vec<Child>,
    kind: String,
    mode: String,
    path: String,
    #[serde(default)]
    payload: String,
    #[serde(default)]
    sha256: String,
    size: String,
}
#[derive(serde::Deserialize)]
#[serde(deny_unknown_fields)]
struct Child {
    kind: String,
    mode: String,
    name: String,
}

const STOCK: &str = "gomad.wasi-stock-exploratory/v1";
const COOPERATIVE: &str = "gomad.wasi-cooperative/v1";
const EPOCH: u64 = 946_684_800_000_000_000;
pub(crate) fn temporal_build_tags() -> Vec<String> {
    [
        "disable_grpc_modules",
        "gomad_wasm_slim",
        "sqlite3_dotlk",
        "test_dep",
    ]
    .into_iter()
    .map(str::to_owned)
    .collect()
}

fn decode(text: Option<&str>) -> Result<Vec<u8>, String> {
    crate::qualify::cells::base64(text.unwrap_or(""))
        .ok_or_else(|| "invalid configuration base64".into())
}
fn digest(bytes: &[u8]) -> String {
    crate::evidence::canonical::digest(bytes)
}
fn absolute(path: &str) -> bool {
    path.starts_with('/')
        && !path.contains('\0')
        && (path == "/" || !path.ends_with('/'))
        && !path
            .split('/')
            .skip(1)
            .any(|part| matches!(part, "" | "." | ".."))
}
fn number(text: &str) -> Result<u64, String> {
    if text.is_empty()
        || text.len() > 1 && text.starts_with('0')
        || !text.bytes().all(|b| b.is_ascii_digit())
    {
        return Err("invalid captured integer".into());
    }
    text.parse().map_err(|_| "invalid captured integer".into())
}
fn default_value(profile: &str) -> Value {
    serde_json::json!({"Profile":profile,"Args":null,"Environment":["PWD=/workspace"],"WorkingDirectory":"/workspace","WritableDirectories":["/workspace","/tmp"],"CapturedInputs":{"Manifest":{"Bytes":0,"Entries":0,"File":"","Limits":{"DirectoryEntries":0,"Files":0,"PathBytes":0,"Requests":0,"SingleFileBytes":0,"TotalBytes":0},"Mappings":null,"NotExist":0,"Schema":"","SHA256":"","TotalBytes":0},"Descriptor":null,"Payloads":null},"Stdin":null,"EntropyKey":vec![0_u8;32],"Clock":{"EpochNanos":EPOCH,"ReadStepNanos":if profile == COOPERATIVE {0} else {1000}},"Limits":{"OutputBytes":8<<20,"TranscriptBytes":64<<20,"Calls":1_000_000,"Descriptors":64,"Files":1000,"FilesystemBytes":64<<20,"PendingEvents":4096}})
}
fn ensure(
    entries: &mut BTreeMap<String, linked::NamespaceEntry>,
    path: &str,
    readonly: bool,
) -> Result<(), String> {
    let mut prefix = String::new();
    for part in path.split('/').filter(|part| !part.is_empty()) {
        prefix.push('/');
        prefix.push_str(part);
        let entry = entries
            .entry(prefix.clone())
            .or_insert_with(|| linked::NamespaceEntry {
                path: prefix.as_bytes().to_vec(),
                directory: true,
                readonly,
                data: vec![],
            });
        if !entry.directory {
            return Err("writable directory beneath file".into());
        }
    }
    Ok(())
}
#[expect(
    clippy::too_many_lines,
    reason = "Captured payload admission checks the complete typed descriptor and byte inventory before exposure"
)]
fn capture(
    captured: &Captured,
) -> Result<(BTreeMap<String, linked::NamespaceEntry>, Vec<String>), String> {
    let manifest = &captured.manifest;
    let bytes = decode(captured.descriptor.as_deref())?;
    let payloads = captured.payloads.as_ref();
    if manifest.schema.is_empty() {
        if !bytes.is_empty() || payloads.is_some_and(|p| !p.is_empty()) {
            return Err("captured manifest required".into());
        }
        return Ok((BTreeMap::new(), vec![]));
    }
    if manifest.schema != "gomad.io-read-only-mounts/v1"
        || manifest.file != "io/mounts.json"
        || manifest.bytes != bytes.len() as u64
        || manifest.sha256 != format!("sha256:{}", digest(&bytes))
    {
        return Err("captured descriptor identity mismatch".into());
    }
    let descriptor: Descriptor = decode::typed(crate::qualify::spec::strict(&bytes)?)?;
    let limits = CaptureLimits {
        directory_entries: number(&descriptor.limits.directory_entries)?,
        files: number(&descriptor.limits.files)?,
        path_bytes: number(&descriptor.limits.path_bytes)?,
        requests: number(&descriptor.limits.requests)?,
        single_file_bytes: number(&descriptor.limits.single_file_bytes)?,
        total_bytes: number(&descriptor.limits.total_bytes)?,
    };
    let requests = number(&descriptor.requests)?;
    if descriptor.schema != manifest.schema
        || descriptor.mappings != manifest.mappings.clone().unwrap_or_default()
        || limits != manifest.limits
        || descriptor.entries.len() as u64 != manifest.entries
        || descriptor.not_exist.len() as u64 != manifest.not_exist
        || number(&descriptor.total_bytes)? != manifest.total_bytes
        || requests > limits.requests
        || limits.requests == 0
        || limits.files == 0
        || limits.directory_entries == 0
        || limits.path_bytes == 0
        || limits.single_file_bytes == 0
        || limits.single_file_bytes > limits.total_bytes
        || descriptor.entries.len() as u64 > limits.files
    {
        return Err("captured descriptor does not match its manifest or limits".into());
    }
    if descriptor.mappings.is_empty()
        || descriptor
            .mappings
            .windows(2)
            .any(|pair| pair[0] >= pair[1])
        || descriptor
            .mappings
            .iter()
            .any(|root| !absolute(root) || root == "/")
    {
        return Err("invalid captured mappings".into());
    }
    let mapped = |path: &str| {
        absolute(path)
            && path.len() as u64 <= limits.path_bytes
            && descriptor
                .mappings
                .iter()
                .any(|root| path == root || path.starts_with(&format!("{root}/")))
    };
    let mut entries = BTreeMap::new();
    let mut total = 0;
    let mut children = 0_u64;
    let mut previous = "";
    for entry in &descriptor.entries {
        if !mapped(&entry.path) || entry.path.as_str() <= previous || !valid_mode(&entry.mode) {
            return Err("invalid captured entry path or mode".into());
        }
        previous = &entry.path;
        let size = number(&entry.size)?;
        let data = match entry.kind.as_str() {
            "file" => {
                if !entry.children.is_empty()
                    || entry.payload
                        != format!(
                            "io/mounts/files/{}",
                            entry.sha256.strip_prefix("sha256:").unwrap_or("")
                        )
                {
                    return Err("invalid captured file descriptor".into());
                }
                let data = decode(Some(
                    payloads
                        .and_then(|p| p.get(&entry.payload))
                        .ok_or("missing captured payload")?,
                ))?;
                if data.len() as u64 != size
                    || entry.sha256 != format!("sha256:{}", digest(&data))
                    || size > limits.single_file_bytes
                {
                    return Err("captured payload identity mismatch".into());
                }
                total += size;
                data
            }
            "directory" if size == 0 && entry.payload.is_empty() && entry.sha256.is_empty() => {
                vec![]
            }
            _ => return Err("invalid captured entry kind".into()),
        };
        children += entry.children.len() as u64;
        if total > limits.total_bytes || children > limits.directory_entries {
            return Err("captured input capacity exceeded".into());
        }
        ensure(
            &mut entries,
            Path::new(&entry.path)
                .parent()
                .and_then(Path::to_str)
                .unwrap_or("/"),
            true,
        )?;
        if entries
            .insert(
                entry.path.clone(),
                linked::NamespaceEntry {
                    path: entry.path.as_bytes().to_vec(),
                    directory: entry.kind == "directory",
                    readonly: true,
                    data,
                },
            )
            .is_some()
        {
            return Err("invalid captured tree".into());
        }
    }
    if total != manifest.total_bytes {
        return Err("captured total bytes mismatch".into());
    }
    for entry in &descriptor.entries {
        if entry.kind != "directory" {
            continue;
        }
        let actual: Vec<_> = descriptor
            .entries
            .iter()
            .filter(|child| Path::new(&child.path).parent() == Some(Path::new(&entry.path)))
            .collect();
        if actual.len() != entry.children.len()
            || entry.children.windows(2).any(|p| p[0].name >= p[1].name)
        {
            return Err("capture must contain complete directory inventory".into());
        }
        for (child, actual) in entry.children.iter().zip(actual) {
            if child.name.is_empty()
                || child.name.contains('/')
                || !valid_mode(&child.mode)
                || child.kind != actual.kind
                || child.mode != actual.mode
                || Path::new(&actual.path).file_name().and_then(|p| p.to_str())
                    != Some(child.name.as_str())
            {
                return Err("captured child inventory mismatch".into());
            }
        }
    }
    if descriptor.not_exist.windows(2).any(|p| p[0] >= p[1])
        || descriptor
            .not_exist
            .iter()
            .any(|path| !mapped(path) || entries.contains_key(path))
    {
        return Err("invalid captured missing paths".into());
    }
    Ok((entries, descriptor.mappings))
}
fn valid_mode(mode: &str) -> bool {
    mode.len() == 4 && mode.bytes().all(|b| matches!(b, b'0'..=b'7'))
}

fn open_relative(
    root: &std::fs::File,
    path: &Path,
    directory: bool,
) -> Result<std::fs::File, String> {
    use rustix::fs::{Mode, OFlags, openat};
    let mut file = root.try_clone().map_err(|e| e.to_string())?;
    let parts: Vec<_> = path
        .components()
        .filter(|part| {
            !matches!(
                part,
                std::path::Component::RootDir | std::path::Component::CurDir
            )
        })
        .collect();
    for (index, part) in parts.iter().enumerate() {
        let mut flags = OFlags::RDONLY | OFlags::CLOEXEC | OFlags::NOFOLLOW | OFlags::NONBLOCK;
        if directory || index + 1 < parts.len() {
            flags |= OFlags::DIRECTORY;
        }
        file = openat(&file, part.as_os_str(), flags, Mode::empty())
            .map(std::fs::File::from)
            .map_err(|e| {
                format!("Temporal schema capture refuses unsafe or unreadable path: {e}")
            })?;
    }
    Ok(file)
}

#[derive(Clone, Copy)]
pub enum TemporalPolicy {
    Installation,
    Qualification,
    Inspection,
}

/// # Errors
/// Refuses unsafe or oversized schema captures and packages outside the checkout.
#[expect(
    clippy::too_many_lines,
    reason = "Temporal policy capture binds its source tree and complete guest configuration in one transaction"
)]
pub fn temporal_policy(
    root: &Path,
    package: &str,
    policy: TemporalPolicy,
) -> Result<Value, String> {
    use std::os::unix::fs::PermissionsExt as _;
    if root.as_os_str().is_empty()
        || package.is_empty()
        || package.starts_with('/')
        || package.contains(['\\', '\0'])
        || package.contains("...")
    {
        return Err("temporal setup requires a checkout and one relative package directory".into());
    }
    let mut parts = Vec::new();
    for part in package.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if parts.pop().is_none() {
                    return Err("temporal package escapes checkout".into());
                }
            }
            _ => parts.push(part),
        }
    }
    let cwd = if parts.is_empty() {
        "/workspace".into()
    } else {
        format!("/workspace/{}", parts.join("/"))
    };
    let source = root.join("schema/sqlite");
    let base = std::fs::File::open(if source.is_absolute() {
        Path::new("/")
    } else {
        Path::new(".")
    })
    .map_err(|e| e.to_string())?;
    let source_root = open_relative(&base, &source, true)?;
    let mut pending = vec![(
        std::path::PathBuf::from("."),
        "/workspace/schema/sqlite".to_owned(),
    )];
    let mut entries = Vec::new();
    let mut payloads = BTreeMap::new();
    let mut total = 0_u64;
    let mut child_count = 0_u64;
    while let Some((source, target)) = pending.pop() {
        if entries.len() >= 10_000 || target.len() > 4096 {
            return Err("Temporal schema capture capacity exceeded".into());
        }
        let file = open_relative(&source_root, &source, false)?;
        let metadata = file.metadata().map_err(|e| e.to_string())?;
        let mode = format!("{:04o}", metadata.permissions().mode() & 0o777);
        if metadata.is_dir() {
            let mut directory = Vec::new();
            for child in rustix::fs::Dir::read_from(&file).map_err(|e| e.to_string())? {
                let child = child.map_err(|e| e.to_string())?;
                let name = child
                    .file_name()
                    .to_str()
                    .map_err(|_| "Temporal schema filename is not UTF-8")?;
                if matches!(name, "." | "..") {
                    continue;
                }
                child_count += 1;
                if child_count > 100_000
                    || entries.len() + pending.len() + directory.len() >= 10_000
                {
                    return Err("Temporal directory entry capacity exceeded".into());
                }
                directory.push(name.to_owned());
            }
            directory.sort();
            let mut children = Vec::new();
            for name in directory {
                let child = open_relative(&file, Path::new(&name), false)?;
                let meta = child.metadata().map_err(|e| e.to_string())?;
                if !meta.is_dir() && !meta.is_file() {
                    return Err("Temporal schema capture refuses special files".into());
                }
                children.push(serde_json::json!({"kind":if meta.is_dir(){"directory"}else{"file"},"mode":format!("{:04o}",meta.permissions().mode()&0o777),"name":name}));
                pending.push((source.join(&name), format!("{target}/{name}")));
            }
            entries.push(serde_json::json!({"children":children,"kind":"directory","mode":mode,"path":target,"size":"0"}));
        } else if metadata.is_file() {
            use std::io::Read as _;
            let mut data = Vec::new();
            file.take((16 << 20) + 1)
                .read_to_end(&mut data)
                .map_err(|e| e.to_string())?;
            if data.len() > 16 << 20 {
                return Err("Temporal schema single-file byte capacity exceeded".into());
            }
            total += data.len() as u64;
            if total > 64 << 20 {
                return Err("Temporal schema capture byte capacity exceeded".into());
            }
            let sha = digest(&data);
            let payload = format!("io/mounts/files/{sha}");
            payloads.insert(payload.clone(), crate::campaign::exploration::base64(&data));
            entries.push(serde_json::json!({"children":[],"kind":"file","mode":mode,"path":target,"size":data.len().to_string(),"payload":payload,"sha256":format!("sha256:{sha}")}));
        } else {
            return Err("Temporal schema capture refuses symlinks and special files".into());
        }
    }
    entries.sort_by(|a, b| a["path"].as_str().cmp(&b["path"].as_str()));
    let limits = serde_json::json!({"directory_entries":"100000","files":"10000","path_bytes":"4096","requests":"100000","single_file_bytes":(16<<20).to_string(),"total_bytes":(64<<20).to_string()});
    let descriptor = serde_json::json!({"entries":entries,"limits":limits,"mappings":["/workspace/schema/sqlite"],"requests":entries.len().to_string(),"schema":"gomad.io-read-only-mounts/v1","total_bytes":total.to_string()});
    let descriptor = go_bytes(&descriptor)?;
    let captured = serde_json::json!({"Manifest":{"Bytes":descriptor.len(),"Entries":entries.len(),"File":"io/mounts.json","Limits":{"DirectoryEntries":100_000,"Files":10_000,"PathBytes":4096,"Requests":100_000,"SingleFileBytes":16<<20,"TotalBytes":64<<20},"Mappings":["/workspace/schema/sqlite"],"NotExist":0,"Schema":"gomad.io-read-only-mounts/v1","SHA256":format!("sha256:{}",digest(&descriptor)),"TotalBytes":total},"Descriptor":crate::campaign::exploration::base64(&descriptor),"Payloads":payloads});
    let mut config = default_value(COOPERATIVE);
    config["WorkingDirectory"] = serde_json::json!(cwd);
    config["Args"] = serde_json::json!([
        "temporal.test",
        "-test.v",
        "-test.count=1",
        "-test.timeout=72h"
    ]);
    config["Environment"] = serde_json::json!([
        format!("PWD={cwd}"),
        "TEMPORAL_ROOT=/workspace",
        "TMPDIR=/tmp",
        "TZ=UTC",
        "TEMPORAL_TEST_LOG_LEVEL=warn"
    ]);
    config["WritableDirectories"] = serde_json::json!(["/workspace", "/tmp", cwd]);
    config["CapturedInputs"] = captured;
    config["EntropyKey"] =
        serde_json::json!(sha2::Sha256::digest(b"gomad-wasm-prototype/v1").to_vec());
    config["Limits"] = serde_json::json!({"OutputBytes":256<<20,"TranscriptBytes":1<<30,"Calls":1<<28,"Descriptors":4096,"Files":100_000,"FilesystemBytes":512<<20,"PendingEvents":4096});
    if matches!(
        policy,
        TemporalPolicy::Installation | TemporalPolicy::Inspection
    ) {
        config["network_v2"] = serde_json::json!({"profile":"gomad.simulation-network-stream/v3","limits":{"listeners":128,"connections":4096,"bytes":128<<20,"connection_bytes":4<<20,"stall_nanos":10_000_000_000_u64,"transitions":1<<20}});
    }
    match policy {
        TemporalPolicy::Installation => {}
        TemporalPolicy::Qualification => config["Args"] = Value::Null,
        TemporalPolicy::Inspection => {
            config["Args"] = Value::Null;
            config["Environment"]
                .as_array_mut()
                .ok_or("Temporal environment missing")?
                .insert(1, serde_json::json!("GOMAD_DEBUG_CANARY=1"));
            config["WritableDirectories"] = serde_json::json!(["/workspace", "/tmp"]);
            config["EntropyKey"] = serde_json::json!(vec![0_u8; 32]);
            config["Limits"]["OutputBytes"] = serde_json::json!(16 << 20);
            config["Limits"]["TranscriptBytes"] = serde_json::json!(256 << 20);
        }
    }
    Ok(config)
}

#[derive(serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
struct Storage {
    #[serde(default)]
    volumes: Option<Vec<Volume>>,
    limits: VolumeLimits,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    faults: Option<Vec<StorageFault>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    sqlite_lock_databases: Option<Vec<String>>,
}
#[derive(serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
struct Volume {
    #[serde(rename = "ID")]
    id: String,
    #[serde(rename = "Path")]
    path: String,
    #[serde(rename = "CapacityBytes")]
    capacity: u64,
}
#[derive(serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
struct VolumeLimits {
    #[serde(rename = "PendingOperations")]
    pending: u64,
    #[serde(rename = "Transitions")]
    transitions: u64,
}
#[derive(serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
struct StorageFault {
    name: String,
    resource: String,
    operation: String,
    occurrence: u64,
    action: String,
    #[serde(default, skip_serializing_if = "zero")]
    bytes: u64,
}
#[expect(
    clippy::trivially_copy_pass_by_ref,
    reason = "Serde skip_serializing_if callbacks take borrowed fields"
)]
fn zero(value: &u64) -> bool {
    *value == 0
}
fn go_bytes<T: serde::Serialize>(value: &T) -> Result<Vec<u8>, String> {
    let text = serde_json::to_string(value).map_err(|e| e.to_string())?;
    Ok(text
        .replace('&', "\\u0026")
        .replace('<', "\\u003c")
        .replace('>', "\\u003e")
        .replace('\u{2028}', "\\u2028")
        .replace('\u{2029}', "\\u2029")
        .into_bytes())
}
fn storage(value: Option<Value>) -> Result<Option<linked::StorageInput>, String> {
    let Some(value) = value else { return Ok(None) };
    let mut model: Storage = decode::typed(value)?;
    if model.faults.as_ref().is_some_and(Vec::is_empty) {
        model.faults = None;
    }
    if model
        .sqlite_lock_databases
        .as_ref()
        .is_some_and(Vec::is_empty)
    {
        model.sqlite_lock_databases = None;
    }
    let identity = format!("sha256:{}", digest(&go_bytes(&model)?));
    serde_json::from_value(serde_json::json!({"descriptor":{"model":model,"model_identity":identity,"return_image":true}})).map(Some).map_err(|e|e.to_string())
}
#[expect(
    clippy::too_many_lines,
    reason = "Configuration admission validates policy, namespace, storage and network together before returning a sealed model"
)]
pub(crate) fn convert(original: Value, profile: &str) -> Result<Configuration, String> {
    let guest: Guest = decode::typed(original.clone())?;
    if guest.profile != profile || !matches!(profile, STOCK | COOPERATIVE) {
        return Err("configuration profile does not match selected profile".into());
    }
    let l = &guest.limits;
    if l.output_bytes == 0
        || l.output_bytes > 1 << 30
        || l.transcript_bytes == 0
        || l.transcript_bytes > 1 << 30
        || l.calls == 0
        || l.calls
            > if profile == COOPERATIVE {
                1 << 28
            } else {
                1_000_000
            }
        || !(5..=1 << 20).contains(&l.descriptors)
        || !(1..=1 << 20).contains(&l.files)
        || !(1..=1 << 30).contains(&l.filesystem_bytes)
        || !(1..=4096).contains(&l.pending_events)
    {
        return Err("invalid configuration limits".into());
    }
    if !absolute(&guest.working_directory)
        || guest.clock.epoch_nanos > i64::MAX as u64
        || guest.clock.read_step_nanos > 1_000_000_000
        || profile == STOCK && guest.clock.read_step_nanos == 0
    {
        return Err("invalid working directory or clock policy".into());
    }
    if (guest.network.is_some() || guest.storage.is_some()) && profile != COOPERATIVE
        || guest.network.is_some() && guest.clock.read_step_nanos != 0
    {
        return Err(
            "network and storage require cooperative profile and network requires frozen clock"
                .into(),
        );
    }
    let arguments = guest.args.unwrap_or_default();
    let environment_list = guest.environment.unwrap_or_default();
    for values in [&arguments, &environment_list] {
        if values.len() > 4 << 20
            || values.iter().any(|v| v.contains('\0'))
            || values.iter().map(|v| v.len() + 1).sum::<usize>() > 4 << 20
        {
            return Err("invalid argument or environment bounds".into());
        }
    }
    let mut environment = BTreeMap::new();
    for value in environment_list {
        let (name, value) = value
            .split_once('=')
            .ok_or("environment entries require NAME=VALUE")?;
        if name.is_empty()
            || environment
                .insert(name.to_owned(), value.to_owned())
                .is_some()
        {
            return Err("invalid or duplicate environment entry".into());
        }
    }
    if environment.get("PWD") != Some(&guest.working_directory) {
        return Err("captured PWD must match working directory".into());
    }
    let (mut entries, roots) = capture(&guest.captured_inputs)?;
    let writable = guest.writable_directories.unwrap_or_default();
    let mut unique = std::collections::BTreeSet::new();
    for directory in &writable {
        if !absolute(directory)
            || directory == "/"
            || !unique.insert(directory.clone())
            || roots
                .iter()
                .any(|root| directory == root || directory.starts_with(&format!("{root}/")))
        {
            return Err("invalid writable directory or captured overlap".into());
        }
        ensure(&mut entries, directory, false)?;
        entries
            .get_mut(directory)
            .ok_or("missing writable directory")?
            .readonly = false;
    }
    if guest.working_directory != "/"
        && !entries
            .get(&guest.working_directory)
            .is_some_and(|e| e.directory)
    {
        return Err("working directory absent from captured namespace".into());
    }
    let stdin = decode(guest.stdin.as_deref())?;
    if entries.len() as u64 + 1 > l.files
        || stdin.len() as u64 + entries.values().map(|e| e.data.len() as u64).sum::<u64>()
            > l.filesystem_bytes
    {
        return Err("initial namespace capacity exceeded".into());
    }
    let mut preopens = writable.clone();
    preopens.extend(roots);
    preopens.sort();
    preopens.dedup();
    if preopens.len() as u64 + 3 > l.descriptors {
        return Err("preopen descriptor capacity exceeded".into());
    }
    let model = serde_json::json!({"entropy_key":guest.entropy_key,"working_directory_sha256":digest(guest.working_directory.as_bytes()),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"seed":0,"mode":1,"limit":8<<20,"diagnostics":false,"epoch_nanos":guest.clock.epoch_nanos,"read_step_nanos":guest.clock.read_step_nanos,"now":u8::from(profile==COOPERATIVE && guest.clock.read_step_nanos==0),"limits":{"calls":l.calls,"transcript_bytes":l.transcript_bytes,"pending_events":l.pending_events}});
    let checked: linked::ModelConfig =
        serde_json::from_value(model.clone()).map_err(|e| e.to_string())?;
    checked.validate().map_err(|e| e.to_string())?;
    let config = Configuration {
        original,
        model,
        namespace: linked::NamespaceImage {
            entries: entries.into_values().collect(),
            writable_directories: writable.into_iter().map(String::into_bytes).collect(),
            preopens: preopens.into_iter().map(String::into_bytes).collect(),
            working_directory: guest.working_directory.into_bytes(),
            stdin,
            limits: linked::NamespaceLimits {
                files: l.files,
                descriptors: l.descriptors,
                filesystem_bytes: l.filesystem_bytes,
                output_bytes: l.output_bytes,
            },
            epoch_nanos: guest.clock.epoch_nanos,
        },
        storage: storage(guest.storage)?,
        network: guest.network,
        build_tags: vec![],
        environment,
        arguments,
    };
    validate(&config)?;
    Ok(config)
}

fn validate(config: &Configuration) -> Result<(), String> {
    let model = serde_json::from_value(config.model.clone())
        .map_err(|error| format!("configuration model: {error}"))?;
    let environment = config
        .environment
        .iter()
        .map(|(name, value)| format!("{name}={value}"))
        .collect();
    let request = linked::Request {
        module: vec![],
        module_sha256: digest(&[]),
        limits: linked::Limits {
            memory_bytes: 64 << 20,
            fuel: 1_000_000,
            output_bytes: config.namespace.limits.output_bytes,
            evidence_bytes: 1 << 30,
            wall_timeout: std::time::Duration::from_secs(1),
        },
        model: Some(linked::ModelInput {
            configuration: model,
            arguments: config.arguments.clone(),
            environment,
            replay_trace: None,
        }),
        namespace: Some(config.namespace.clone()),
        storage: config.storage.clone(),
    };
    let mut prepared = linked::prepare(request).map_err(|e| e.to_string())?;
    if let Some(network) = &config.network {
        prepared
            .network(network.clone())
            .map_err(|e| e.to_string())?;
    }
    Ok(())
}

fn read_config(bytes: &[u8]) -> Result<Value, String> {
    crate::qualify::spec::decode_first(bytes, |value, trailing| {
        let _: Guest = decode::typed(value.clone())?;
        if trailing {
            return Err("qualification JSON has trailing data".into());
        }
        Ok(value)
    })
}

#[cfg(test)]
mod receiving_config_tests {
    use super::read_config;

    #[test]
    fn first_typed_value_errors_precede_trailing_data() {
        for (input, expected) in [
            (br#"{"Profile":1} {}"#.as_slice(), "string"),
            (br#"{"UnknownOwned":true} {}"#.as_slice(), "unknown field"),
        ] {
            let error = read_config(input).unwrap_err();
            assert!(error.contains(expected), "{error}");
            assert!(!error.contains("trailing"), "{error}");
        }
    }
}

#[cfg(test)]
mod temporal_policy_tests {
    use super::*;

    #[test]
    fn inspection_policy_has_its_complete_distinct_limits_and_environment() {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
        let value = temporal_policy(&root, ".", TemporalPolicy::Inspection).unwrap();
        assert!(value["Args"].is_null());
        assert_eq!(
            value["WritableDirectories"],
            serde_json::json!(["/workspace", "/tmp"])
        );
        assert_eq!(value["EntropyKey"], serde_json::json!(vec![0; 32]));
        assert_eq!(value["Limits"]["OutputBytes"], 16 << 20);
        assert_eq!(value["Limits"]["TranscriptBytes"], 256 << 20);
        assert!(
            value["Environment"]
                .as_array()
                .unwrap()
                .contains(&serde_json::json!("GOMAD_DEBUG_CANARY=1"))
        );
        assert!(convert(value, COOPERATIVE).unwrap().network.is_some());
    }

    #[test]
    fn qualification_policy_preserves_schema_and_has_no_network_or_arguments() {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
        let value = temporal_policy(&root, "./tests", TemporalPolicy::Qualification).unwrap();
        let installed = build(COOPERATIVE, None, Some((&root, "./tests"))).unwrap();
        assert!(installed.network.is_some());
        assert_eq!(
            installed.arguments,
            [
                "temporal.test",
                "-test.v",
                "-test.count=1",
                "-test.timeout=72h"
            ]
        );
        assert!(value["network_v2"].is_null());
        assert!(value["Args"].is_null());
        assert_eq!(value["WorkingDirectory"], "/workspace/tests");
        assert_eq!(
            value["WritableDirectories"],
            serde_json::json!(["/workspace", "/tmp", "/workspace/tests"])
        );
        assert_eq!(value["Clock"]["EpochNanos"], 946_684_800_000_000_000_u64);
        assert_eq!(value["Clock"]["ReadStepNanos"], 0);
        assert_eq!(value["Limits"]["TranscriptBytes"], 1 << 30);
        assert!(
            value["CapturedInputs"]["Manifest"]["Entries"]
                .as_u64()
                .unwrap()
                > 0
        );
        let configured = convert(value, COOPERATIVE).unwrap();
        assert!(configured.network.is_none());
    }
}
