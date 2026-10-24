use super::{canonical, canonical_path, digest, read, regular, require, strict_json};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    os::unix::fs::MetadataExt as _,
    path::{Path, PathBuf},
};
const SCHEMA: &str = "gomad.gnu-identity-toolchain/v1";
const BINUTILS: [&str; 8] = [
    "as", "ld", "ar", "ranlib", "nm", "objcopy", "objdump", "strip",
];
const DELEGATES: [&str; 12] = [
    "cc1",
    "cc1plus",
    "collect2",
    "lto-wrapper",
    "as",
    "ld",
    "ar",
    "ranlib",
    "nm",
    "objcopy",
    "objdump",
    "strip",
];
const PREFIXES: [&str; 8] = [
    "/usr/libexec",
    "/usr/lib64",
    "/usr/lib",
    "/usr/local/lib",
    "/usr/include",
    "/usr/local/include",
    "/lib",
    "/lib64",
];
const SOURCES: &[&str] = &[
    "tools/gomad/rust/gomad/src/cycle/preparation/inputs/gnu.rs",
    "tools/gomad/rust/gomad/src/cycle/preparation/inputs/mod.rs",
    "tools/gomad/rust/gomad/src/cycle/process/mod.rs",
    "tools/gomad/rust/gomad/src/cycle/process/environment.rs",
    "tools/gomad/rust/gomad/src/cycle/process/registry.rs",
    "tools/gomad/rust/gomad/src/cycle/process/sys.rs",
];
#[derive(Debug, PartialEq, Eq)]
pub(crate) struct Elf {
    pub machine: u16,
    pub interpreter: Vec<String>,
    pub needed: Vec<String>,
    pub search: Vec<String>,
}
fn word<const N: usize>(raw: &[u8], at: usize) -> Result<[u8; N], String> {
    raw.get(at..at.checked_add(N).ok_or("malformed compiler ELF bounds")?)
        .and_then(|v| v.try_into().ok())
        .ok_or_else(|| "malformed compiler ELF bounds".into())
}
#[expect(
    clippy::too_many_lines,
    reason = "One bounded ELF parser validates offsets and strings before extraction"
)]
pub(crate) fn elf(raw: &[u8]) -> Result<Option<Elf>, String> {
    if !raw.starts_with(b"\x7fELF") {
        return Ok(None);
    }
    require(
        raw.len() >= 64 && raw[4..6] == [2, 1],
        "GNU envelope supports ELF64 little-endian only",
    )?;
    let machine = u16::from_le_bytes(word(raw, 18)?);
    require(
        [62, 183].contains(&machine),
        "GNU envelope supports native x86_64/aarch64 only",
    )?;
    let offset = u64::from_le_bytes(word(raw, 32)?);
    let size = u16::from_le_bytes(word(raw, 54)?);
    let count = u16::from_le_bytes(word(raw, 56)?);
    require(
        (size == 56 || count == 0 && size == 0)
            && count <= 1024
            && offset
                .checked_add(u64::from(size) * u64::from(count))
                .is_some_and(|end| end <= raw.len() as u64),
        "malformed compiler ELF program headers",
    )?;
    let mut headers = Vec::new();
    for n in 0..count {
        let pos =
            usize::try_from(offset + u64::from(n) * u64::from(size)).map_err(|e| e.to_string())?;
        headers.push((
            u32::from_le_bytes(word(raw, pos)?),
            u64::from_le_bytes(word(raw, pos + 8)?),
            u64::from_le_bytes(word(raw, pos + 16)?),
            u64::from_le_bytes(word(raw, pos + 40)?),
        ));
    }
    let segment = |begin: u64, length: u64| -> Result<&[u8], String> {
        let end = begin
            .checked_add(length)
            .ok_or("malformed compiler ELF segment")?;
        require(end <= raw.len() as u64, "malformed compiler ELF segment")?;
        Ok(&raw[usize::try_from(begin).map_err(|e| e.to_string())?
            ..usize::try_from(end).map_err(|e| e.to_string())?])
    };
    let mut interpreter = Vec::new();
    let mut dynamic = Vec::new();
    for &(kind, begin, _, length) in &headers {
        if kind == 3 {
            let value = segment(begin, length)?;
            let end = value.iter().rposition(|v| *v != 0).map_or(0, |n| n + 1);
            interpreter.push(
                std::str::from_utf8(&value[..end])
                    .map_err(|e| e.to_string())?
                    .into(),
            );
        }
        if kind == 2 {
            dynamic.push(segment(begin, length)?);
        }
    }
    require(interpreter.len() <= 1, "multiple compiler ELF interpreters")?;
    require(dynamic.len() <= 1, "multiple compiler ELF dynamic tables")?;
    let mut result = Elf {
        machine,
        interpreter,
        needed: vec![],
        search: vec![],
    };
    let Some(table) = dynamic.first() else {
        return Ok(Some(result));
    };
    require(
        table.len() % 16 == 0,
        "malformed compiler ELF dynamic entries",
    )?;
    let mut entries = Vec::new();
    for pos in (0..table.len()).step_by(16) {
        let tag = i64::from_le_bytes(word(table, pos)?);
        if tag == 0 {
            break;
        }
        entries.push((tag, u64::from_le_bytes(word(table, pos + 8)?)));
    }
    let strings: Vec<_> = entries
        .iter()
        .filter(|(tag, _)| *tag == 5)
        .map(|(_, v)| *v)
        .collect();
    let lengths: Vec<_> = entries
        .iter()
        .filter(|(tag, _)| *tag == 10)
        .map(|(_, v)| *v)
        .collect();
    require(
        strings.len() == 1 && lengths.len() == 1,
        "compiler ELF string table missing",
    )?;
    let mut locations = Vec::new();
    for &(kind, begin, address, length) in &headers {
        if kind == 1
            && address <= strings[0]
            && strings[0]
                .checked_add(lengths[0])
                .zip(address.checked_add(length))
                .is_some_and(|(a, b)| a <= b)
        {
            locations.push(
                begin
                    .checked_add(strings[0] - address)
                    .ok_or("malformed compiler ELF string location")?,
            );
        }
    }
    require(
        locations.len() == 1 && lengths[0] <= 64 << 20,
        "compiler ELF string table outside load segments",
    )?;
    let strings = segment(locations[0], lengths[0])?;
    let text = |index: u64| -> Result<String, String> {
        let index = usize::try_from(index).map_err(|e| e.to_string())?;
        let tail = strings
            .get(index..)
            .ok_or("unterminated compiler ELF string")?;
        let end = tail
            .iter()
            .position(|v| *v == 0)
            .ok_or("unterminated compiler ELF string")?;
        Ok(std::str::from_utf8(&tail[..end])
            .map_err(|e| e.to_string())?
            .into())
    };
    for (tag, value) in entries {
        if tag == 1 {
            result.needed.push(text(value)?);
        }
        if tag == 15 || tag == 29 {
            result
                .search
                .extend(text(value)?.split(':').map(str::to_owned));
        }
    }
    Ok(Some(result))
}
fn normalize(path: &Path) -> PathBuf {
    let mut out = PathBuf::new();
    for component in path.components() {
        match component {
            std::path::Component::ParentDir => {
                out.pop();
            }
            std::path::Component::CurDir => {}
            other => out.push(other.as_os_str()),
        }
    }
    out
}
pub(crate) fn aliases(
    path: &Path,
    table: &mut BTreeMap<String, String>,
) -> Result<PathBuf, String> {
    require(path.is_absolute(), "GNU input must be absolute")?;
    let mut path = path.to_path_buf();
    for _ in 0..40 {
        let mut current = PathBuf::from("/");
        let mut replacement = None;
        for part in path.components().skip(1) {
            current.push(part.as_os_str());
            if fs::symlink_metadata(&current).is_ok_and(|m| m.file_type().is_symlink()) {
                let target = fs::read_link(&current).map_err(|e| e.to_string())?;
                table.insert(
                    current.to_str().ok_or("non-UTF8 GNU alias")?.into(),
                    target.to_str().ok_or("non-UTF8 GNU alias target")?.into(),
                );
                let tail = path.strip_prefix(&current).map_err(|e| e.to_string())?;
                let resolved = if target.is_absolute() {
                    target
                } else {
                    current
                        .parent()
                        .ok_or("GNU alias parent missing")?
                        .join(target)
                };
                replacement = Some(normalize(&resolved.join(tail)));
                break;
            }
        }
        match replacement {
            Some(next) => path = next,
            None => return Ok(normalize(&path)),
        }
    }
    Err("GNU input alias cycle".into())
}
pub(crate) fn probes(linker: &str, loader: &str) -> BTreeMap<String, Vec<String>> {
    let mut out = BTreeMap::new();
    for (name, flag) in [
        ("version", "--version"),
        ("machine", "-dumpmachine"),
        ("gcc-version", "-dumpfullversion"),
        ("search", "-print-search-dirs"),
        ("specs", "-dumpspecs"),
        ("sysroot", "-print-sysroot"),
    ] {
        out.insert(name.into(), vec![linker.into(), flag.into()]);
    }
    for name in DELEGATES {
        out.insert(
            name.into(),
            vec![linker.into(), format!("-print-prog-name={name}")],
        );
    }
    for (name, lang) in [("include-c", "c"), ("include-cxx", "c++")] {
        out.insert(
            name.into(),
            [linker, "-E", "-x", lang, "/dev/null", "-v"]
                .map(str::to_string)
                .to_vec(),
        );
    }
    out.insert("loader-cache".into(), vec![loader.into(), "-p".into()]);
    out
}
pub(crate) fn probe_environment(root: &Path) -> BTreeMap<String, String> {
    [
        ("PATH", "/usr/bin:/bin".into()),
        ("HOME", root.join("home").to_string_lossy().into_owned()),
        ("TMPDIR", root.join("tmp").to_string_lossy().into_owned()),
        ("LC_ALL", "C".into()),
        ("TZ", "UTC".into()),
    ]
    .into_iter()
    .map(|(k, v)| (k.into(), v))
    .collect()
}
fn host_machine(os: &str, architecture: &str) -> Result<u16, String> {
    require(
        os.starts_with("linux") && matches!(architecture, "aarch64" | "x86_64"),
        "GNU identity source-build envelope requires native Linux aarch64/x86_64",
    )?;
    Ok(if architecture == "aarch64" { 183 } else { 62 })
}
pub(crate) fn native_machine() -> Result<u16, String> {
    host_machine(std::env::consts::OS, std::env::consts::ARCH)
}
fn linker_path(path: &Path) -> Result<(), String> {
    let name = path.file_name().and_then(|v| v.to_str()).unwrap_or("");
    let suffix = name
        .strip_prefix("aarch64-linux-gnu-gcc-")
        .or_else(|| name.strip_prefix("x86_64-linux-gnu-gcc-"));
    require(
        canonical_path(path).is_ok()
            && path.parent() == Some(Path::new("/usr/bin"))
            && suffix.is_some_and(|v| !v.is_empty() && v.bytes().all(|v| v.is_ascii_digit())),
        "GNU envelope requires its supported canonical system GCC driver",
    )
}
fn native_elf(raw: &[u8], expected: u16, message: &str) -> Result<(), String> {
    require(
        elf(raw)?.is_some_and(|info| info.machine == expected),
        message,
    )
}
pub(crate) fn validate_linker(linker: &str) -> Result<(), String> {
    let path = Path::new(linker);
    linker_path(path)?;
    native_elf(
        &read(path, 512 << 20)?,
        native_machine()?,
        "GNU driver is not a native supported ELF executable",
    )
}
fn producer_identity(owner: &Value, sources: &BTreeMap<String, String>) -> Result<(), String> {
    require(
        owner.as_object().is_some_and(|v| {
            v.len() == 4
                && [
                    "pid",
                    "start_monotonic_nanos",
                    "end_monotonic_nanos",
                    "sources",
                ]
                .iter()
                .all(|k| v.contains_key(*k))
        }) && owner["pid"].as_u64().is_some_and(|p| p > 0)
            && owner["start_monotonic_nanos"]
                .as_u64()
                .zip(owner["end_monotonic_nanos"].as_u64())
                .is_some_and(|(a, b)| a > 0 && a < b)
            && owner["sources"] == json!(sources),
        "GNU prefreeze producer identity changed",
    )
}
fn static_plan(tools: &Value) -> Result<(), String> {
    require(
        tools["schema"] == "gomad.driver-static-plan/v1",
        "GNU envelope requires the actual static tool plan",
    )
}
fn gcc_version(version: &str) -> Result<(), String> {
    require(
        version
            .lines()
            .next()
            .is_some_and(|line| line.to_ascii_lowercase().contains("gcc")),
        "GNU envelope requires actual GNU GCC",
    )
}
fn compiler_identity(
    machine: &str,
    linker: &str,
    version: &str,
    gcc: &str,
    specs: &str,
    sysroot: &str,
) -> Result<(), String> {
    require(
        matches!(machine, "aarch64-linux-gnu" | "x86_64-linux-gnu")
            && linker.contains(machine)
            && gcc_version(version).is_ok()
            && !gcc.is_empty()
            && gcc
                .split('.')
                .all(|part| !part.is_empty() && part.bytes().all(|b| b.is_ascii_digit()))
            && !specs.is_empty()
            && matches!(sysroot, "" | "/"),
        "unsupported GNU compiler family/machine/specs/sysroot",
    )
}
fn host_matches(machine: u16, expected: u16) -> Result<(), String> {
    require(
        machine == expected,
        "GNU toolchain does not match this actual native source-build host",
    )
}
fn delegate(path: &Path, raw: &[u8], machine: &str, expected: u16) -> Result<(), String> {
    require(
        [
            "/usr/bin",
            "/usr/lib/gcc",
            "/usr/libexec/gcc",
            &format!("/usr/{machine}/bin"),
        ]
        .iter()
        .any(|prefix| path.starts_with(prefix))
            && elf(raw)?.is_some_and(|e| e.machine == expected),
        "unsupported GNU delegate placement or architecture",
    )
}
fn compiler_root(path: &Path) -> Result<(), String> {
    require(
        path.starts_with("/usr/lib/gcc") || path.starts_with("/usr/libexec/gcc"),
        "GNU internal compiler root missing",
    )
}
fn search_directory(path: &Path, header: bool) -> Result<(), String> {
    require(
        path.is_dir(),
        if header {
            "GNU header search input missing"
        } else {
            "GNU library search root is not a directory"
        },
    )
}
fn header_roots(raw: &str) -> Result<Vec<&str>, String> {
    let selected = raw
        .split_once("#include <...> search starts here:\n")
        .and_then(|(_, tail)| tail.split_once("End of search list.").map(|(body, _)| body))
        .ok_or("GNU actual header search probe missing")?;
    Ok(selected.lines().map(str::trim).collect())
}
fn no_preload(path: &Path) -> Result<(), String> {
    require(
        !path.exists(),
        "GNU envelope refuses a global loader preload",
    )
}
fn closure_machine(machine: u16, expected: u16) -> Result<(), String> {
    require(
        machine == expected,
        "GNU closure contains a foreign dynamic ELF machine",
    )
}
fn needed_name(needed: &str) -> Result<(), String> {
    require(
        !needed.contains('/'),
        "GNU closure has an unsupported absolute DT_NEEDED entry",
    )
}
fn loader_search(value: &str, parent: &str) -> Result<PathBuf, String> {
    require(
        !value.is_empty()
            && !value
                .replace("${ORIGIN}", "")
                .replace("$ORIGIN", "")
                .contains('$'),
        "unsupported GNU dynamic-loader expansion",
    )?;
    let base = value
        .replace("${ORIGIN}", parent)
        .replace("$ORIGIN", parent);
    require(
        Path::new(&base).is_absolute(),
        "relative GNU ELF loader search",
    )?;
    Ok(PathBuf::from(base))
}
fn dependency_found(found: bool, needed: &str) -> Result<(), String> {
    require(
        found,
        &format!("GNU compiler ELF dependency is not captured: {needed}"),
    )
}
fn envelope_bounds(count: usize, bytes: u64) -> Result<(), String> {
    require(
        count <= 100_000 && bytes <= 16 << 30,
        "GNU compiler envelope exceeds bounded native toolchain inputs",
    )
}
fn directory_attributes(is_dir: bool, mode: u32, uid: u32, message: &str) -> Result<(), String> {
    require(
        is_dir && mode & 0o7777 == 0o700 && uid == rustix::process::geteuid().as_raw(),
        message,
    )
}
fn lookup_directory(bin: &Path, links: &BTreeMap<String, String>) -> Result<(), String> {
    let message = "GNU executable lookup contains foreign or missing entries";
    require(canonical_path(bin).is_ok(), message)?;
    let info = fs::symlink_metadata(bin).map_err(|e| e.to_string())?;
    directory_attributes(info.is_dir(), info.mode(), info.uid(), message)?;
    let actual: BTreeSet<String> = fs::read_dir(bin)
        .map_err(|e| e.to_string())?
        .map(|entry| {
            entry.map_err(|e| e.to_string()).and_then(|entry| {
                entry
                    .file_name()
                    .into_string()
                    .map_err(|_| "GNU lookup UTF8".into())
            })
        })
        .collect::<Result<_, String>>()?;
    require(actual == links.keys().cloned().collect(), message)
}
fn lookup_link(path: &Path, target: &str) -> Result<(), String> {
    require(
        fs::symlink_metadata(path).is_ok_and(|m| m.file_type().is_symlink())
            && fs::read_link(path).map_err(|e| e.to_string())?.as_os_str()
                == std::ffi::OsStr::new(target),
        "GNU closed executable lookup changed",
    )
}
fn fresh_root(root: &Path) -> Result<(), String> {
    require(
        super::clean(root)
            && !root.exists()
            && root
                .parent()
                .is_some_and(|p| fs::canonicalize(p).is_ok_and(|v| v == p)),
        "GNU prefreeze root must be fresh canonical absolute",
    )
}
fn null_device(mode: u32, device: u64) -> Result<(), String> {
    require(
        mode & 0o170_000 == 0o020_000 && device == rustix::fs::makedev(1, 3),
        "GNU header probes require actual Linux /dev/null",
    )
}
fn harness_binding(binding: &Value) -> Result<(), String> {
    require(
        binding
            .as_object()
            .is_some_and(|v| v.len() == 1 && v.contains_key("toolchain_plan")),
        "identity harness requires its actual prepared GNU toolchain plan",
    )
}
fn declared_roots(declared: &Value, tools: &Path, path: &Path, root: &Path) -> Result<(), String> {
    require(
        declared["schema"] == SCHEMA
            && declared["tools_plan"] == json!(tools)
            && path == root.join("gnu-toolchain.json"),
        "GNU toolchain plan has foreign roots/static tools",
    )
}
fn closure_equal(declared: &Value, actual: &Value) -> Result<(), String> {
    require(
        declared == actual,
        "GNU toolchain probes/delegates/search/loader closure changed",
    )
}
fn validate_probe(
    row: &Value,
    process: &Value,
    argv: &[String],
    repo: &str,
    owner: &Value,
    environment: &BTreeMap<String, String>,
    pids: &mut BTreeSet<u64>,
) -> Result<(), String> {
    let expected_argv = serde_json::to_value(argv).map_err(|e| e.to_string())?;
    for item in [row, process] {
        require(
            item.is_object()
                && item["argv"] == expected_argv
                && item["pid"].as_u64().is_some_and(|v| v > 0),
            "wrong live command/PID",
        )?;
        require(
            item["exit"].as_i64() == Some(0) && item["observation"] == "completed",
            "nonpassing/inconclusive live producer",
        )?;
        require(
            item["start_ns"]
                .as_u64()
                .zip(item["end_ns"].as_u64())
                .is_some_and(|(a, b)| a > 0 && a < b),
            "unexecuted live producer interval",
        )?;
    }
    require(
        row["cwd"] == repo
            && row["pid"] == process["pid"]
            && row["start_ns"] == process["start_ns"]
            && process["active"] == false
            && (process["cleanup_signal"].is_null() || process["cleanup_signal"] == false),
        "live process was not cleanly reaped",
    )?;
    let env = serde_json::to_value(environment).map_err(|e| e.to_string())?;
    require(
        row["environment_mode"] == "exact"
            && row["effective_environment"] == env
            && row["environment"] == env,
        "GNU probe substituted its environment",
    )?;
    let stage = owner["pid"]
        .as_u64()
        .filter(|v| *v > 0)
        .ok_or("invalid GNU producer PID")?;
    let start = owner["start_monotonic_nanos"]
        .as_u64()
        .ok_or("invalid GNU producer start")?;
    let end = owner["end_monotonic_nanos"]
        .as_u64()
        .ok_or("invalid GNU producer end")?;
    let pid = row["pid"].as_u64().ok_or("invalid GNU probe PID")?;
    let began = row["started_monotonic_nanos"]
        .as_u64()
        .ok_or("invalid GNU probe start")?;
    let reaped = row["reaped_monotonic_nanos"]
        .as_u64()
        .ok_or("invalid GNU probe reap")?;
    require(
        row["stage_pid"].as_u64() == Some(stage)
            && process["stage_pid"].as_u64() == Some(stage)
            && row["parent_pid"].as_u64().is_some_and(|v| v > 0)
            && row["parent_pid"] == row["supervisor_pid"]
            && row["supervisor_parent_pid"].as_u64() == Some(stage)
            && row["clock"] == "CLOCK_MONOTONIC"
            && process["clock"] == "CLOCK_MONOTONIC"
            && !pids.contains(&pid)
            && start <= began
            && began < reaped
            && reaped <= end,
        "GNU probe borrowed another producer/lifecycle",
    )?;
    for field in [
        "parent_pid",
        "supervisor_pid",
        "supervisor_parent_pid",
        "started_monotonic_nanos",
        "reaped_monotonic_nanos",
    ] {
        require(
            row[field] == process[field],
            "GNU probe process/command lifecycle differs",
        )?;
    }
    require(
        process["cleanup_complete"] == true && process["failure"].is_null(),
        "GNU probe process cleanup incomplete",
    )?;
    pids.insert(pid);
    Ok(())
}
#[cfg(test)]
#[path = "gnu_tests.rs"]
mod tests;
#[derive(Default)]
struct Closure {
    aliases: BTreeMap<String, String>,
    files: BTreeMap<String, String>,
    missing: BTreeSet<String>,
    walked: BTreeSet<PathBuf>,
    bytes: u64,
}
impl Closure {
    fn resolve(&mut self, path: &Path) -> Result<PathBuf, String> {
        aliases(path, &mut self.aliases)
    }
    fn add(&mut self, path: &Path) -> Result<PathBuf, String> {
        let path = self.resolve(path)?;
        let name = path.to_str().ok_or("GNU path UTF8")?.to_owned();
        if !self.files.contains_key(&name) {
            envelope_bounds(self.files.len() + 1, self.bytes)?;
            let bytes = read(&path, 512 << 20)?;
            self.bytes = self
                .bytes
                .checked_add(bytes.len() as u64)
                .ok_or("GNU compiler input size overflow")?;
            envelope_bounds(self.files.len() + 1, self.bytes)?;
            self.files.insert(name, digest(&bytes));
        }
        Ok(path)
    }
    fn within(&mut self, path: &Path, prefixes: &[String]) -> Result<bool, String> {
        for prefix in prefixes {
            if path.starts_with(self.resolve(Path::new(prefix))?) {
                return Ok(true);
            }
        }
        Ok(false)
    }
    fn supported(&mut self, path: &Path, machine: &str) -> Result<PathBuf, String> {
        let resolved = self.resolve(path)?;
        let prefixes: Vec<String> = PREFIXES
            .iter()
            .map(|s| (*s).to_owned())
            .chain([
                format!("/usr/{machine}/lib"),
                format!("/usr/{machine}/include"),
            ])
            .collect();
        require(
            path.is_absolute() && self.within(&resolved, &prefixes)?,
            "GNU search path escapes the supported system envelope",
        )?;
        Ok(resolved)
    }
    fn program_root(&mut self, path: &Path, machine: &str) -> Result<PathBuf, String> {
        let path = self.resolve(path)?;
        let prefixes = [
            "/usr/lib".into(),
            "/usr/libexec".into(),
            "/usr/bin".into(),
            "/bin".into(),
            format!("/usr/{machine}/bin"),
        ];
        require(
            path.is_absolute() && self.within(&path, &prefixes)?,
            "GNU executable search root escapes its supported prefix",
        )?;
        Ok(path)
    }
    fn walk(&mut self, path: &Path) -> Result<(), String> {
        let root = self.resolve(path)?;
        if !self.walked.insert(root.clone()) {
            return Ok(());
        }
        let mut pending = vec![root.clone()];
        while let Some(directory) = pending.pop() {
            for entry in fs::read_dir(&directory).map_err(|e| e.to_string())? {
                let path = entry.map_err(|e| e.to_string())?.path();
                let info = fs::symlink_metadata(&path).map_err(|e| e.to_string())?;
                if info.file_type().is_symlink() {
                    let target = self.resolve(&path)?;
                    if target.is_dir() {
                        require(
                            target.starts_with(&root)
                                || self.within(&target, &PREFIXES.map(str::to_owned))?,
                            "GNU directory alias escapes supported compiler/include roots",
                        )?;
                        if !target.starts_with(&root) {
                            self.walk(&target)?;
                        }
                    } else {
                        self.add(&path)?;
                    }
                } else if info.is_dir() {
                    pending.push(path);
                } else {
                    self.add(&path)?;
                }
            }
        }
        Ok(())
    }
}
fn document(path: &Path) -> Result<Value, String> {
    strict_json(&regular(path, 64 << 20, None)?)
}
fn output(root: &Path, name: &str) -> Result<String, String> {
    String::from_utf8(regular(
        &root.join(format!("{name}.stdout")),
        128 << 20,
        None,
    )?)
    .map(|s| s.trim().to_owned())
    .map_err(|e| e.to_string())
}
fn which(name: &str) -> Result<PathBuf, String> {
    if Path::new(name).is_absolute() {
        return Ok(PathBuf::from(name));
    }
    for base in ["/usr/bin", "/bin"] {
        let path = Path::new(base).join(name);
        if fs::metadata(&path).is_ok_and(|m| m.is_file() && m.mode() & 0o111 != 0) {
            return Ok(path);
        }
    }
    Err(format!("missing GNU delegate {name}"))
}
fn source_pins(repo: &Path) -> Result<BTreeMap<String, String>, String> {
    let mut pins = BTreeMap::new();
    for source in SOURCES {
        let path = repo.join(source);
        pins.insert(
            path.to_str().ok_or("GNU producer source UTF8")?.to_owned(),
            digest(&read(&path, 512 << 20)?),
        );
    }
    let executable = std::env::current_exe().map_err(|e| e.to_string())?;
    canonical_path(&executable)?;
    pins.insert(
        executable.to_string_lossy().into_owned(),
        digest(&read(&executable, 512 << 20)?),
    );
    Ok(pins)
}
fn private_directory(path: &Path) -> Result<(), String> {
    let message = "GNU prefreeze root is not its private actual owner directory";
    require(canonical_path(path).is_ok(), message)?;
    let info = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    directory_attributes(info.is_dir(), info.mode(), info.uid(), message)
}
fn search_roots(text: &str, kind: &str) -> Result<Vec<String>, String> {
    let prefix = format!("{kind}: =");
    let values: Vec<_> = text
        .lines()
        .filter_map(|line| line.strip_prefix(&prefix))
        .collect();
    require(
        values.len() == 1,
        if kind == "programs" {
            "GNU executable search probe missing"
        } else {
            "GNU library search probe missing"
        },
    )?;
    Ok(values[0].split(':').map(str::to_owned).collect())
}
#[expect(
    clippy::too_many_lines,
    reason = "Reopening reconstructs the complete GNU compiler envelope from actual probe bytes"
)]
pub(crate) fn derive(root: &Path, tools_path: &Path, linker: &str) -> Result<Value, String> {
    private_directory(root)?;
    let tools = document(tools_path)?;
    static_plan(&tools)?;
    let repo = tools["repo"].as_str().ok_or("GNU static repo missing")?;
    validate_linker(linker)?;
    let mut closure = Closure::default();
    let loader = closure.resolve(Path::new("/usr/sbin/ldconfig"))?;
    let owner = document(&root.join("producer.json"))?;
    producer_identity(&owner, &source_pins(Path::new(repo))?)?;
    let mut commands = BTreeMap::new();
    let mut pids = BTreeSet::new();
    for (name, argv) in probes(linker, loader.to_str().ok_or("GNU loader UTF8")?) {
        let row = document(&root.join(format!("{name}.command.json")))?;
        let process = document(&root.join(format!("{name}.process.json")))?;
        validate_probe(
            &row,
            &process,
            &argv,
            repo,
            &owner,
            &probe_environment(root),
            &mut pids,
        )?;
        let mut table = BTreeMap::new();
        for suffix in ["command.json", "process.json", "stdout", "stderr"] {
            let path = closure.add(&root.join(format!("{name}.{suffix}")))?;
            table.insert(
                suffix.to_owned(),
                closure.files[path.to_str().ok_or("GNU command path UTF8")?].clone(),
            );
        }
        commands.insert(name, table);
    }
    let machine = output(root, "machine")?;
    let version = output(root, "gcc-version")?;
    compiler_identity(
        &machine,
        linker,
        &output(root, "version")?,
        &version,
        &output(root, "specs")?,
        &output(root, "sysroot")?,
    )?;
    let expected_machine = if machine.starts_with("aarch64") {
        183
    } else {
        62
    };
    host_matches(native_machine()?, expected_machine)?;
    native_elf(
        &read(Path::new(linker), 512 << 20)?,
        expected_machine,
        "GNU compiler is not native",
    )?;
    let mut programs = BTreeMap::new();
    for name in DELEGATES {
        let declared = output(root, name)?;
        let resolved = closure.add(&which(&declared)?)?;
        delegate(
            &resolved,
            &read(&resolved, 512 << 20)?,
            &machine,
            expected_machine,
        )?;
        programs.insert(name.to_owned(), resolved.to_string_lossy().into_owned());
    }
    let search_output = output(root, "search")?;
    let mut program_search = BTreeSet::new();
    for name in search_roots(&search_output, "programs")? {
        let path = closure.program_root(Path::new(&name), &machine)?;
        program_search.insert(path.to_string_lossy().into_owned());
        for delegate in DELEGATES {
            let candidate = path.join(delegate);
            if candidate.is_file() {
                closure.add(&candidate)?;
            } else {
                closure
                    .missing
                    .insert(candidate.to_string_lossy().into_owned());
            }
        }
    }
    let version_root = Path::new(&programs["cc1"])
        .parent()
        .ok_or("GNU cc1 parent missing")?;
    compiler_root(version_root)?;
    closure.walk(version_root)?;
    let mut search = BTreeSet::new();
    for name in search_roots(&search_output, "libraries")? {
        let path = closure.supported(Path::new(&name), &machine)?;
        if !path.exists() {
            closure.missing.insert(path.to_string_lossy().into_owned());
            continue;
        }
        search_directory(&path, false)?;
        search.insert(path.to_string_lossy().into_owned());
        for entry in fs::read_dir(&path).map_err(|e| e.to_string())? {
            let path = entry.map_err(|e| e.to_string())?.path();
            if path.is_file() {
                closure.add(&path)?;
            }
        }
        for child in ["ldscripts", "bfd-plugins"] {
            let child = path.join(child);
            if child.is_dir() {
                closure.walk(&child)?;
            }
        }
    }
    let mut includes = BTreeSet::new();
    for name in ["include-c", "include-cxx"] {
        let raw = String::from_utf8(regular(
            &root.join(format!("{name}.stderr")),
            128 << 20,
            None,
        )?)
        .map_err(|e| e.to_string())?;
        for line in header_roots(&raw)? {
            let path = closure.supported(Path::new(line.trim()), &machine)?;
            search_directory(&path, true)?;
            includes.insert(path.to_string_lossy().into_owned());
            closure.walk(&path)?;
        }
    }
    no_preload(Path::new("/etc/ld.so.preload"))?;
    closure.missing.insert("/etc/ld.so.preload".into());
    for name in ["/etc/ld.so.cache", "/etc/ld.so.conf"] {
        closure.add(Path::new(name))?;
    }
    if Path::new("/etc/ld.so.conf.d").is_dir() {
        closure.walk(Path::new("/etc/ld.so.conf.d"))?;
    }
    for path in [
        PathBuf::from(linker),
        loader,
        tools_path.to_path_buf(),
        root.join("producer.json"),
    ] {
        closure.add(&path)?;
    }
    for source in source_pins(Path::new(repo))?.keys() {
        closure.add(Path::new(source))?;
    }
    let rustc = tools["tools"]["rustc"]
        .as_str()
        .ok_or("GNU static Rust compiler missing")?;
    closure.walk(
        Path::new(rustc)
            .parent()
            .and_then(Path::parent)
            .ok_or("GNU Rust distribution root missing")?,
    )?;
    let python = closure.add(Path::new(
        tools["tools"]["python"]
            .as_str()
            .ok_or("GNU static python input missing")?,
    ))?;
    let git = closure.add(&which("git")?)?;
    let mut cache: BTreeMap<String, Vec<String>> = BTreeMap::new();
    for line in output(root, "loader-cache")?.lines() {
        if let Some((left, right)) = line.split_once("=>") {
            let mut words = left.split_whitespace();
            if let (Some(name), Some(kind)) = (words.next(), words.next())
                && kind.starts_with('(')
                && right.split_whitespace().count() == 1
            {
                cache
                    .entry(name.into())
                    .or_default()
                    .push(right.trim().into());
            }
        }
    }
    let mut pending: Vec<_> = closure.files.keys().cloned().collect();
    let mut visited = BTreeSet::new();
    while let Some(name) = pending.pop() {
        if !visited.insert(name.clone()) {
            continue;
        }
        let path = Path::new(&name);
        let Some(info) = elf(&read(path, 512 << 20)?)? else {
            continue;
        };
        closure_machine(info.machine, expected_machine)?;
        for interpreter in info.interpreter {
            let candidate = closure.add(Path::new(&interpreter))?;
            if !visited.contains(candidate.to_str().ok_or("GNU interpreter UTF8")?) {
                pending.push(candidate.to_string_lossy().into_owned());
            }
        }
        for needed in info.needed {
            needed_name(&needed)?;
            let mut candidates = cache.get(&needed).cloned().unwrap_or_default();
            for value in &info.search {
                let parent = path
                    .parent()
                    .and_then(Path::to_str)
                    .ok_or("GNU ELF parent UTF8")?;
                let candidate = loader_search(value, parent)?.join(&needed);
                if candidate.is_file() {
                    candidates.push(candidate.to_string_lossy().into_owned());
                }
            }
            let mut found = false;
            for candidate in candidates {
                let resolved = closure.resolve(Path::new(&candidate))?;
                if !resolved.is_file() {
                    continue;
                }
                if elf(&read(&resolved, 512 << 20)?)?
                    .is_some_and(|other| other.machine == expected_machine)
                {
                    found = true;
                    let resolved = closure.add(Path::new(&candidate))?;
                    let name = resolved.to_string_lossy().into_owned();
                    if !visited.contains(&name) {
                        pending.push(name);
                    }
                }
            }
            dependency_found(found, &needed)?;
        }
    }
    let bin = root.join("bin");
    let mut links = BTreeMap::from([
        ("cc".into(), linker.to_owned()),
        ("gcc".into(), linker.to_owned()),
        ("c++".into(), linker.to_owned()),
        ("g++".into(), linker.to_owned()),
        ("git".into(), git.to_string_lossy().into_owned()),
    ]);
    for name in BINUTILS {
        links.insert(name.to_owned(), programs[name].clone());
    }
    lookup_directory(&bin, &links)?;
    for (name, target) in links {
        let path = bin.join(name);
        lookup_link(&path, &target)?;
        closure.resolve(&path)?;
    }
    Ok(
        json!({"schema":SCHEMA,"root":root,"tools_plan":tools_path,"linker":linker,"machine":machine,"programs":programs,"python":python,"git":git,"program_search":program_search,"search":search,"includes":includes,"missing":closure.missing,"aliases":closure.aliases,"files":closure.files,"commands":commands}),
    )
}
fn now() -> Result<u64, String> {
    let time = rustix::time::clock_gettime(rustix::time::ClockId::Monotonic);
    u64::try_from(time.tv_sec)
        .map_err(|e| e.to_string())?
        .checked_mul(1_000_000_000)
        .and_then(|n| n.checked_add(u64::try_from(time.tv_nsec).ok()?))
        .ok_or_else(|| "GNU monotonic clock overflow".into())
}
fn write_fresh(path: &Path, value: &Value) -> Result<(), String> {
    use std::io::Write as _;
    use std::os::unix::fs::OpenOptionsExt as _;
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| e.to_string())?;
    let mut bytes = canonical(value)?;
    bytes.push(b'\n');
    file.write_all(&bytes)
        .and_then(|()| file.sync_all())
        .map_err(|e| e.to_string())
}
pub(crate) fn prepare(
    tools_path: &Path,
    linker: &str,
    root: &Path,
    owner: &crate::cycle::process::CycleOwner,
    cancelled: &std::sync::atomic::AtomicBool,
) -> Result<Value, String> {
    use crate::cycle::process::{Command, Environment, Runner};
    use std::os::unix::fs::DirBuilderExt as _;
    let tools = document(tools_path)?;
    validate_linker(linker)?;
    fresh_root(root)?;
    let mut runner = Runner::create(owner, root, None)?;
    for name in ["home", "tmp", "bin"] {
        fs::DirBuilder::new()
            .mode(0o700)
            .create(root.join(name))
            .map_err(|e| e.to_string())?;
    }
    native_machine()?;
    let null = fs::metadata("/dev/null").map_err(|e| e.to_string())?;
    null_device(null.mode(), null.rdev())?;
    let start = now()?;
    let loader = fs::canonicalize("/usr/sbin/ldconfig").map_err(|e| e.to_string())?;
    let environment = probe_environment(root);
    let probes = probes(linker, loader.to_str().ok_or("GNU loader UTF8")?);
    let order = std::iter::once("version".to_owned()).chain(
        probes
            .keys()
            .filter(|name| name.as_str() != "version")
            .cloned(),
    );
    let repo = Path::new(tools["repo"].as_str().ok_or("GNU static repo missing")?);
    for name in order {
        let command = Command {
            argv: probes[&name].clone(),
            cwd: repo.into(),
            environment: Environment::Exact(environment.clone()),
            declared_environment: environment.clone(),
            temporary_paths: BTreeMap::new(),
            accepted: vec![0],
            timeout: std::time::Duration::from_secs(60),
            private_context: None,
        };
        let completed = runner
            .start(&name, command, cancelled, |_| Ok(()))
            .map_err(|e| e.to_string())?
            .wait(cancelled)?;
        require(completed.accepted(), "GNU preparation probe failed")?;
        if name == "version" {
            gcc_version(&output(root, "version")?)?;
        }
    }
    write_fresh(
        &root.join("producer.json"),
        &json!({"pid":std::process::id(),"start_monotonic_nanos":start,"end_monotonic_nanos":now()?,"sources":source_pins(repo)?}),
    )?;
    let mut links = BTreeMap::from([
        ("cc".into(), linker.to_owned()),
        ("gcc".into(), linker.to_owned()),
        ("c++".into(), linker.to_owned()),
        ("g++".into(), linker.to_owned()),
        (
            "git".into(),
            fs::canonicalize(which("git")?)
                .map_err(|e| e.to_string())?
                .to_string_lossy()
                .into_owned(),
        ),
    ]);
    for name in BINUTILS {
        links.insert(
            name.to_owned(),
            fs::canonicalize(which(&output(root, name)?)?)
                .map_err(|e| e.to_string())?
                .to_string_lossy()
                .into_owned(),
        );
    }
    for (name, target) in links {
        std::os::unix::fs::symlink(target, root.join("bin").join(name))
            .map_err(|e| e.to_string())?;
    }
    let result = derive(root, tools_path, linker)?;
    write_fresh(&root.join("gnu-toolchain.json"), &result)?;
    Ok(result)
}
pub(crate) fn inspect(config: &Value, tools_path: &Path) -> Result<(PathBuf, Value), String> {
    let binding = &config["identity_harness"];
    harness_binding(binding)?;
    let path = PathBuf::from(
        binding["toolchain_plan"]
            .as_str()
            .ok_or("GNU toolchain plan missing")?,
    );
    let declared = document(&path)?;
    let root = Path::new(
        declared["root"]
            .as_str()
            .ok_or("GNU declared root missing")?,
    );
    declared_roots(&declared, tools_path, &path, root)?;
    let actual = derive(
        root,
        tools_path,
        declared["linker"]
            .as_str()
            .ok_or("GNU declared linker missing")?,
    )?;
    closure_equal(&declared, &actual)?;
    Ok((path, actual))
}
#[cfg(test)]
pub(crate) fn run_cases(owner: &crate::cycle::process::CycleOwner) -> Result<(), String> {
    let base = PathBuf::from(format!(
        "/home/agent/wt/evidence/fn242-b01-custody-inputs/prep/gnu-{}",
        std::process::id()
    ));
    fs::create_dir_all(&base).map_err(|e| e.to_string())?;
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let architecture = if native_machine()? == 183 {
        "aarch64"
    } else {
        "x86_64"
    };
    let linker = fs::read_dir("/usr/bin")
        .map_err(|e| e.to_string())?
        .filter_map(Result::ok)
        .map(|entry| entry.path())
        .filter(|path| {
            path.file_name()
                .and_then(|n| n.to_str())
                .is_some_and(|name| {
                    name.starts_with(&format!("{architecture}-linux-gnu-gcc-"))
                        && name.rsplit_once('-').is_some_and(|(_, n)| {
                            !n.is_empty() && n.bytes().all(|b| b.is_ascii_digit())
                        })
                })
        })
        .find(|path| fs::canonicalize(path).is_ok_and(|canonical| canonical == *path))
        .ok_or("genuine GNU fixture canonical GCC missing")?;
    let rustc = PathBuf::from(format!(
        "/home/agent/.rustup/toolchains/1.99.1-{architecture}-unknown-linux-gnu/bin/rustc"
    ));
    let python = fs::canonicalize("/usr/bin/python3").map_err(|e| e.to_string())?;
    let tools = base.join("tools.json");
    write_fresh(
        &tools,
        &json!({"schema":"gomad.driver-static-plan/v1","repo":repo,"tools":{"rustc":rustc,"python":python}}),
    )?;
    let root = base.join("prepared");
    let linker = linker.to_str().ok_or("GNU fixture linker UTF8")?;
    let valid = prepare(
        &tools,
        linker,
        &root,
        owner,
        &std::sync::atomic::AtomicBool::new(false),
    )?;
    let config = json!({"identity_harness":{"toolchain_plan":root.join("gnu-toolchain.json")}});
    inspect(&config, &tools)?;
    let plan = root.join("gnu-toolchain.json");
    let raw = regular(&plan, 64 << 20, None)?;
    let mut observed = Vec::new();
    for (field, value) in [("machine", json!("foreign"))] {
        let mut changed = valid.clone();
        changed[field] = value;
        fs::write(&plan, canonical(&changed)?).map_err(|e| e.to_string())?;
        let refused = inspect(&config, &tools).unwrap_err();
        require(
            refused.contains("closure changed"),
            &format!("GNU resealed {field} wrong predicate: {refused}"),
        )?;
        observed.push(json!({"case":field,"input_sha256":digest(&canonical(&changed)?),"input":changed,"refusal":refused,"intended_predicate":"GNU toolchain probes/delegates/search/loader closure changed"}));
        fs::write(&plan, &raw).map_err(|e| e.to_string())?;
    }
    observed.extend(tests::original_cases(&base, &root, &tools, linker, &valid)?);
    write_fresh(
        &base.join("proof.json"),
        &json!({"schema":"gomad.gnu-inputs-proof/v1","control":"normal-return","input_sha256":digest(&canonical(&valid)?),"input":valid,"observations":observed}),
    )?;
    println!(
        "GNU_INPUTS_ACTUAL {} control=normal-return probes=21 mutations={}",
        base.display(),
        observed.len()
    );
    Ok(())
}
