use super::{canonical, clean, digest, read, regular, require, strict_json};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    path::{Path, PathBuf},
};
const PREFIX: &str = "target/preparation-inputs";
const STATIC: &[&str] = &[
    "internal/model/choice/schema/choicewire.json",
    "internal/model/choice/schema/choicewire.go.tmpl",
    "internal/model/choice/schema/choicewire_runtime.go.tmpl",
    "internal/model/choice/schema/choicewire_test.go.tmpl",
    "internal/model/choice/trace.go",
    "internal/model/choice/tape.go",
    "internal/backends/wasm/toolchain/overlay.go",
    "internal/backends/wasm/toolchain/runtime_wasm.go.txt",
    "internal/backends/wasm/toolchain/runtime_inspect_wasm.go.txt",
    "internal/backends/wasm/toolchain/net_stream_wasm.go.txt",
    "internal/backends/wasm/toolchain/net_stream_status_generated.go.txt",
    "internal/backends/wasm/toolchain/cmd/overlay/main.go",
];
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub(crate) struct PreparedInputs {
    pub compiler: BTreeMap<String, String>,
    pub sources: BTreeMap<String, String>,
}
pub(crate) struct Inventory {
    files: BTreeMap<String, String>,
}
struct GeneratedTest {
    import_path: String,
    cache_root: PathBuf,
    sources: BTreeMap<String, String>,
}
impl GeneratedTest {
    fn from_observations(
        observed: &Observed,
        import_path: &str,
        sources: &BTreeMap<String, String>,
    ) -> Result<Self, String> {
        require(
            !import_path.is_empty() && !import_path.chars().any(char::is_whitespace),
            "native generated test package identity changed",
        )?;
        let environment = &observed
            .commands
            .get("selected-source-closure")
            .ok_or("native selected closure missing")?
            .environment;
        let base = if let Some(cache) = environment
            .get("XDG_CACHE_HOME")
            .filter(|cache| !cache.is_empty())
        {
            PathBuf::from(cache)
        } else {
            let home = environment
                .get("HOME")
                .filter(|home| !home.is_empty())
                .ok_or("native generated test cache home missing")?;
            if cfg!(target_os = "macos") {
                Path::new(home).join("Library/Caches")
            } else {
                Path::new(home).join(".cache")
            }
        };
        let cache_root = base.join("go-build");
        require(
            clean(&cache_root),
            "native generated test cache root is not absolute and clean",
        )?;
        Ok(Self {
            import_path: format!("{import_path}.test"),
            cache_root,
            sources: sources.clone(),
        })
    }
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Command {
    pub name: String,
    pub argv: Vec<String>,
    pub cwd: String,
    pub environment: BTreeMap<String, String>,
    pub owner_pid: u64,
    pub pid: u64,
    pub start_ns: u64,
    pub end_ns: u64,
    pub exit: i64,
    pub output_limit: u64,
    pub stdout: String,
    pub stderr: String,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Blob {
    path: String,
    bytes: u64,
    sha256: String,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Observations {
    schema: String,
    phase: String,
    commands: Vec<Command>,
    snapshots: BTreeMap<String, String>,
    blobs: BTreeMap<String, Blob>,
}
pub(crate) struct Observed {
    commands: BTreeMap<String, Command>,
    snapshots: BTreeMap<String, Vec<u8>>,
    blobs: BTreeMap<String, Vec<u8>>,
}
impl Observed {
    fn stream(&self, name: &str) -> Result<&[u8], String> {
        let command = self
            .commands
            .get(name)
            .ok_or("native preparation command missing")?;
        self.blobs
            .get(&command.stdout)
            .map(Vec::as_slice)
            .ok_or_else(|| "native preparation command stream absent".into())
    }
}
fn field<'a>(value: &'a Value, name: &str) -> Result<&'a str, String> {
    value
        .get(name)
        .and_then(Value::as_str)
        .ok_or_else(|| format!("invalid native preparation field: {name}"))
}
fn strings(value: &Value) -> Result<Vec<String>, String> {
    serde_json::from_value(value.clone()).map_err(|e| e.to_string())
}
fn table(value: &Value) -> Result<BTreeMap<String, String>, String> {
    serde_json::from_value(value.clone()).map_err(|e| e.to_string())
}
fn documents(raw: &[u8]) -> Result<Vec<Value>, String> {
    let mut values = Vec::new();
    let mut start = 0;
    while start < raw.len() {
        while raw
            .get(start)
            .is_some_and(|byte| matches!(*byte, b' ' | b'\r' | b'\n' | b'\t'))
        {
            start += 1;
        }
        if start == raw.len() {
            break;
        }
        require(raw[start] == b'{', "native package stream requires objects")?;
        let mut depth = 0usize;
        let mut quote = false;
        let mut escape = false;
        let mut end = None;
        for (offset, byte) in raw[start..].iter().copied().enumerate() {
            if quote {
                if escape {
                    escape = false;
                } else if byte == b'\\' {
                    escape = true;
                } else if byte == b'"' {
                    quote = false;
                }
                continue;
            }
            match byte {
                b'"' => quote = true,
                b'{' | b'[' => depth += 1,
                b'}' | b']' => {
                    depth = depth
                        .checked_sub(1)
                        .ok_or("invalid native package stream")?;
                    if depth == 0 {
                        end = Some(start + offset + 1);
                        break;
                    }
                }
                _ => (),
            }
        }
        let end = end.ok_or("unterminated native package stream")?;
        let value = strict_json(&raw[start..end])?;
        require(value.is_object(), "native package stream requires objects")?;
        values.push(value);
        start = end;
    }
    require(!values.is_empty(), "empty native package stream")?;
    Ok(values)
}
impl Inventory {
    pub(crate) fn new(config: &Value) -> Result<Self, String> {
        let files = table(
            config
                .get("files")
                .ok_or("native frozen inventory missing")?,
        )?;
        require(
            files.iter().all(|(path, pin)| {
                clean(Path::new(path)) && crate::evidence::canonical::valid_digest(pin)
            }),
            "invalid frozen native inventory",
        )?;
        Ok(Self { files })
    }
    pub(crate) fn pin(&self, path: &Path) -> Result<String, String> {
        require(
            clean(path),
            "native preparation inventory member is not frozen",
        )?;
        self.files
            .get(path.to_str().ok_or("native path UTF8")?)
            .cloned()
            .ok_or_else(|| {
                format!(
                    "native preparation inventory member is not frozen: {}",
                    path.display()
                )
            })
    }
    pub(crate) fn file_bytes(&self, path: &Path) -> Result<Vec<u8>, String> {
        let expected = self.pin(path)?;
        let raw = read(path, 512 << 20)?;
        require(
            digest(&raw) == expected,
            "native preparation source bytes changed",
        )?;
        Ok(raw)
    }
    pub(crate) fn observed(&self, artifact: &Path, phase: &str) -> Result<Observed, String> {
        require(
            matches!(phase, "compile" | "admission"),
            "native preparation invalid phase",
        )?;
        let path = artifact.join(PREFIX).join(phase).join("metadata.json");
        if !path.exists() {
            return Err("unexecuted: actual native preparation observations missing; fresh preparation required".into());
        }
        let raw = regular(&path, 16 << 20, None)?;
        let value = strict_json(&raw)?;
        require(
            canonical(&value)? == raw,
            "wrong/noncanonical native preparation observation",
        )?;
        let observed: Observations = serde_json::from_value(value).map_err(|e| e.to_string())?;
        require(
            observed.schema == "gomad.native-preparation-observations/v1"
                && observed.phase == phase,
            "wrong/noncanonical native preparation observation",
        )?;
        require(
            observed.blobs.len() <= 32,
            "native observation raw blob table exceeds bound",
        )?;
        let mut blobs = BTreeMap::new();
        for (identity, entry) in observed.blobs {
            require(
                entry.path == format!("{PREFIX}/{phase}/{identity}.bin")
                    && identity == entry.sha256
                    && crate::evidence::canonical::valid_digest(&identity)
                    && entry.bytes <= 128 << 20,
                "invalid native preparation blob metadata",
            )?;
            let body = regular(&artifact.join(entry.path), entry.bytes, None)?;
            require(
                body.len() as u64 == entry.bytes && digest(&body) == identity,
                "native preparation raw command stream changed",
            )?;
            blobs.insert(identity, body);
        }
        let mut commands = BTreeMap::new();
        let mut pids = BTreeSet::new();
        let mut owners = BTreeSet::new();
        let mut used = BTreeSet::new();
        for command in observed.commands {
            require(
                !commands.contains_key(&command.name)
                    && command.pid > 0
                    && command.owner_pid > 0
                    && command.pid != command.owner_pid
                    && pids.insert(command.pid)
                    && command.exit == 0
                    && command.start_ns > 0
                    && command.start_ns <= command.end_ns
                    && command.end_ns - command.start_ns <= 600_000_000_000,
                "native preparation command missing/duplicate/failed/outside original deadline",
            )?;
            require(
                command
                    .argv
                    .first()
                    .is_some_and(|name| self.files.contains_key(name)),
                "native preparation actual compiler/environment is not frozen",
            )?;
            for identity in [&command.stdout, &command.stderr] {
                require(
                    blobs.contains_key(identity),
                    "native preparation command stream absent",
                )?;
                used.insert(identity.clone());
            }
            owners.insert(command.owner_pid);
            commands.insert(command.name.clone(), command);
        }
        require(
            owners.len() == 1,
            "native preparation observations borrowed another owner",
        )?;
        let mut snapshots = BTreeMap::new();
        for (name, identity) in observed.snapshots {
            let body = blobs
                .get(&identity)
                .ok_or("native preparation snapshot absent")?;
            used.insert(identity);
            snapshots.insert(name, body.clone());
        }
        require(
            used == blobs.keys().cloned().collect(),
            "native preparation unreferenced/missing raw blobs",
        )?;
        Ok(Observed {
            commands,
            snapshots,
            blobs,
        })
    }
    pub(crate) fn compiler(
        &self,
        environment: &[u8],
        snapshot: &[u8],
    ) -> Result<(BTreeMap<String, String>, PathBuf), String> {
        let value = strict_json(environment)?;
        require(
            value.as_object().is_some_and(|v| {
                v.len() == 2 && v.contains_key("GOROOT") && v.contains_key("GOTOOLDIR")
            }),
            "native compiler env output shape changed",
        )?;
        let root = PathBuf::from(field(&value, "GOROOT")?);
        let tools = PathBuf::from(field(&value, "GOTOOLDIR")?);
        require(
            clean(&root) && clean(&tools),
            "native compiler distribution is not absolute",
        )?;
        let mut expected = BTreeSet::from([
            root.join("VERSION"),
            tools.join("asm"),
            tools.join("compile"),
            tools.join("link"),
        ]);
        let mut headers = 0;
        for entry in fs::read_dir(root.join("pkg/include")).map_err(|e| e.to_string())? {
            let path = entry.map_err(|e| e.to_string())?.path();
            if path.extension().is_some_and(|ext| ext == "h") {
                headers += 1;
                expected.insert(path);
            }
        }
        require(headers > 0, "native compiler assembler headers missing")?;
        let names: Vec<String> = expected
            .iter()
            .map(|p| p.to_str().ok_or("compiler path UTF8").map(str::to_owned))
            .collect::<Result<_, _>>()?;
        require(
            strict_json(snapshot)? == json!(names),
            "native complete compiler directory inventory changed",
        )?;
        let compiler = expected
            .into_iter()
            .map(|path| {
                self.pin(&path)
                    .map(|pin| (path.to_string_lossy().into_owned(), pin))
            })
            .collect::<Result<_, _>>()?;
        Ok((compiler, root))
    }
    pub(crate) fn sources(
        &self,
        raw: &[u8],
        root: &Path,
        replacements: &BTreeMap<String, String>,
    ) -> Result<BTreeMap<String, String>, String> {
        self.collect_sources(raw, root, replacements, None, &mut BTreeSet::new())
    }
    #[expect(
        clippy::too_many_lines,
        reason = "One source projection preserves relative source policy and authenticates generated test identities"
    )]
    fn collect_sources(
        &self,
        raw: &[u8],
        root: &Path,
        replacements: &BTreeMap<String, String>,
        generated_test: Option<&GeneratedTest>,
        authenticated_generated: &mut BTreeSet<String>,
    ) -> Result<BTreeMap<String, String>, String> {
        let mut paths = BTreeSet::new();
        let mut generated_files = BTreeMap::new();
        for package in documents(raw)? {
            let directory = Path::new(field(&package, "Dir")?);
            require(
                clean(directory),
                "native selected package directory is not absolute",
            )?;
            if let Some(module) = package.get("Module").filter(|v| !v.is_null()) {
                require(
                    module.is_object() && module.get("Replace").is_none_or(Value::is_null),
                    "native ambient module replacement unsupported",
                )?;
                if let Some(path) = module
                    .get("GoMod")
                    .and_then(Value::as_str)
                    .filter(|v| !v.is_empty())
                {
                    let path = Path::new(path);
                    self.pin(path)?;
                    paths.insert(path.to_path_buf());
                    let sum = path.with_file_name("go.sum");
                    if sum.exists() {
                        self.pin(&sum)?;
                        paths.insert(sum);
                    }
                }
            }
            for name in [
                "GoFiles",
                "EmbedFiles",
                "TestGoFiles",
                "XTestGoFiles",
                "TestEmbedFiles",
                "XTestEmbedFiles",
                "SFiles",
                "SysoFiles",
                "HFiles",
            ] {
                let values = package
                    .get(name)
                    .map(strings)
                    .transpose()?
                    .unwrap_or_default();
                let single_file = values.len() == 1;
                for filename in values {
                    let relative = Path::new(&filename);
                    if relative.is_absolute()
                        && let Some(context) = generated_test
                    {
                        require(
                            name == "GoFiles"
                                && single_file
                                && package["Name"] == "main"
                                && package["ImportPath"] == context.import_path
                                && clean(relative)
                                && relative.parent().and_then(Path::parent)
                                    == Some(context.cache_root.as_path()),
                            "unsafe native generated test cache member",
                        )?;
                        let token = relative
                            .file_name()
                            .and_then(|name| name.to_str())
                            .and_then(|name| name.strip_suffix("-d"))
                            .ok_or("unsafe native generated test cache member")?;
                        let alias =
                            crate::campaign::prepare::inventory::cache_output_name(relative, token)
                                .ok_or("unsafe native generated test cache member")?;
                        let expected = context
                            .sources
                            .get(alias.to_str().ok_or("generated source key UTF8")?)
                            .ok_or("native generated test source pin missing")?
                            .clone();
                        let body = read(relative, 512 << 20)?;
                        require(
                            digest(&body) == token && token == expected,
                            "native generated test source bytes changed",
                        )?;
                        let alias = alias
                            .to_str()
                            .ok_or("generated source key UTF8")?
                            .to_owned();
                        authenticated_generated.insert(alias.clone());
                        generated_files.insert(alias, expected);
                        continue;
                    }
                    require(
                        !relative.is_absolute()
                            && !relative
                                .components()
                                .any(|p| matches!(p, std::path::Component::ParentDir)),
                        "unsafe native package filename",
                    )?;
                    let path = directory.join(relative);
                    if let Some(replacement) =
                        replacements.get(path.to_str().ok_or("source path UTF8")?)
                    {
                        require(
                            !replacement.is_empty(),
                            "native runtime overlay removed selected input",
                        )?;
                        self.pin(Path::new(replacement))?;
                        paths.insert(PathBuf::from(replacement));
                        if !path.exists() {
                            continue;
                        }
                    }
                    self.pin(&path)?;
                    paths.insert(path);
                }
            }
            for filename in package
                .get("SFiles")
                .map(strings)
                .transpose()?
                .unwrap_or_default()
            {
                let path = directory.join(filename);
                self.assembly(
                    &path,
                    directory,
                    path.parent().ok_or("assembly parent")?,
                    &root.join("pkg/include"),
                    &mut paths,
                    &mut BTreeSet::new(),
                )?;
            }
        }
        require(
            !paths.is_empty(),
            "empty native complete selected inventory",
        )?;
        let mut selected: BTreeMap<String, String> = paths
            .into_iter()
            .map(|path| {
                self.pin(&path)
                    .map(|pin| (path.to_string_lossy().into_owned(), pin))
            })
            .collect::<Result<_, String>>()?;
        selected.extend(generated_files);
        Ok(selected)
    }
    fn assembly(
        &self,
        path: &Path,
        package: &Path,
        source: &Path,
        include: &Path,
        paths: &mut BTreeSet<PathBuf>,
        visited: &mut BTreeSet<PathBuf>,
    ) -> Result<(), String> {
        if !visited.insert(path.into()) {
            return Ok(());
        }
        let bytes = self.file_bytes(path)?;
        let text = std::str::from_utf8(&bytes).map_err(|e| e.to_string())?;
        let mut in_comment = false;
        for line in text.lines() {
            let mut code = String::new();
            let mut chars = line.chars().peekable();
            let mut quote = None;
            while let Some(ch) = chars.next() {
                if in_comment {
                    if ch == '*' && chars.peek() == Some(&'/') {
                        chars.next();
                        in_comment = false;
                    }
                    continue;
                }
                if let Some(quoted) = quote {
                    code.push(ch);
                    if ch == '\\' && quoted != '`' {
                        if let Some(escaped) = chars.next() {
                            code.push(escaped);
                        }
                    } else if ch == quoted {
                        quote = None;
                    }
                    continue;
                }
                if matches!(ch, '"' | '\'' | '`') {
                    quote = Some(ch);
                    code.push(ch);
                    continue;
                }
                if ch == '/' && chars.peek() == Some(&'*') {
                    chars.next();
                    in_comment = true;
                    code.push(' ');
                    continue;
                }
                if ch == '/' && chars.peek() == Some(&'/') {
                    break;
                }
                code.push(ch);
            }
            let Some(directive) = code.trim_start().strip_prefix('#') else {
                continue;
            };
            let Some(rest) = directive
                .trim_start()
                .strip_prefix("include")
                .filter(|rest| {
                    rest.is_empty()
                        || rest.starts_with(char::is_whitespace)
                        || rest.starts_with('"')
                })
            else {
                continue;
            };
            let name: String = serde_json::from_value(strict_json(rest.trim().as_bytes())?)
                .map_err(|e| e.to_string())?;
            let mut candidates = vec![package.join(&name), source.join(&name)];
            if name != "go_asm.h" {
                candidates.push(include.join(&name));
            }
            let selected = candidates.into_iter().find(|path| path.exists());
            let Some(selected) = selected else {
                require(name == "go_asm.h", "native assembler include missing")?;
                continue;
            };
            self.pin(&selected)?;
            paths.insert(selected.clone());
            self.assembly(&selected, package, source, include, paths, visited)?;
        }
        require(!in_comment, "unterminated native assembler comment")
    }
    #[expect(
        clippy::too_many_lines,
        reason = "Independent preparation projection binds both observation phases to the exact original compiler commands and closures"
    )]
    pub(crate) fn project(
        &self,
        artifact: &Path,
        installation: &Value,
        inputs: &Value,
        owner_pid: Option<u64>,
    ) -> Result<PreparedInputs, String> {
        let compile = self.observed(artifact, "compile")?;
        let admission = self.observed(artifact, "admission")?;
        if let Some(owner) = owner_pid {
            require(
                admission
                    .commands
                    .values()
                    .all(|row| row.owner_pid == owner),
                "native admission observations borrowed another actual producer",
            )?;
        }
        let cooperative = installation["profile"] == "gomad.wasi-cooperative/v1";
        let mut names = BTreeSet::from([
            "compiler-version",
            "compiler-environment",
            "selected-package",
            "selected-source-closure",
            "compile-module",
        ]);
        if cooperative {
            names.extend(["compiler-module-cache", "runtime-overlay"]);
        }
        require(
            compile
                .commands
                .keys()
                .map(String::as_str)
                .collect::<BTreeSet<_>>()
                == names
                && admission
                    .commands
                    .keys()
                    .map(String::as_str)
                    .collect::<BTreeSet<_>>()
                    == BTreeSet::from(["selected-source-closure", "compiler-environment"])
                && compile
                    .snapshots
                    .keys()
                    .map(String::as_str)
                    .collect::<BTreeSet<_>>()
                    == if cooperative {
                        BTreeSet::from(["compiler-base-paths", "overlay-replacements"])
                    } else {
                        BTreeSet::from(["compiler-base-paths"])
                    }
                && admission
                    .snapshots
                    .keys()
                    .map(String::as_str)
                    .collect::<BTreeSet<_>>()
                    == BTreeSet::from(["compiler-base-paths"]),
            "native preparation exact phase/command/snapshot coverage changed",
        )?;
        let (mut compiler, root) = self.compiler(
            compile.stream("compiler-environment")?,
            &compile.snapshots["compiler-base-paths"],
        )?;
        let (after_compiler, after_root) = self.compiler(
            admission.stream("compiler-environment")?,
            &admission.snapshots["compiler-base-paths"],
        )?;
        same_compiler_inventory(&compiler, &root, &after_compiler, &after_root)?;
        let executable = field(inputs, "compiler_path")?;
        require(
            installation["compiler_path"] == executable
                && installation["compiler_sha256"] == self.pin(Path::new(executable))?
                && inputs["compiler_sha256"] == installation["compiler_sha256"],
            "native actual compiler differs from admitted executable",
        )?;
        let kind = field(inputs, "kind")?;
        let version = field(installation, "go_version")?;
        require(
            matches!(kind, "go-test" | "go-run")
                && inputs["go_version"] == version
                && compile
                    .stream("compiler-version")?
                    .starts_with(format!("go version {version} ").as_bytes()),
            "native compiler version or target kind changed",
        )?;
        let mut forced: BTreeMap<String, String> = [
            ("GOENV", "off"),
            ("GOPROXY", "off"),
            ("GOSUMDB", "off"),
            ("GOPRIVATE", ""),
            ("GONOPROXY", "none"),
            ("GOVCS", "*:off"),
            ("GOFLAGS", ""),
            ("GOTOOLCHAIN", "local"),
            ("GOWORK", "off"),
            ("GOOS", "wasip1"),
            ("GOARCH", "wasm"),
            ("CGO_ENABLED", "0"),
            ("GOEXPERIMENT", "nogreenteagc"),
            ("GOCACHEPROG", ""),
            ("GOWASM", ""),
            ("GODEBUG", ""),
        ]
        .into_iter()
        .map(|(k, v)| (k.into(), v.into()))
        .collect();
        let selected = table(&inputs["environment"])?;
        require(
            selected
                .keys()
                .all(|k| forced.contains_key(k) || matches!(k.as_str(), "GOMAXPROCS" | "GOTMPDIR"))
                && selected
                    .get("GOWASM")
                    .is_some_and(|v| matches!(v.as_str(), "" | "satconv,signext"))
                && selected.get("GOMAXPROCS").is_none_or(|v| v == "1")
                && selected
                    .get("GOTMPDIR")
                    .is_none_or(|v| Path::new(v).is_absolute() && !v.contains('\0')),
            "native scoped compiler controls changed",
        )?;
        for key in ["GOMAXPROCS", "GOTMPDIR", "GOWASM"] {
            if let Some(value) = selected.get(key) {
                forced.insert(key.into(), value.clone());
            }
        }
        require(
            selected == forced,
            "native original forced compiler controls changed",
        )?;
        let actual_flags = strings(&inputs["build_flags"])?;
        let ordinary = vec!["-mod=readonly", "-trimpath", "-buildvcs=false", "-pgo=off"];
        require(
            actual_flags == ordinary || actual_flags == [ordinary.clone(), vec!["-p=1"]].concat(),
            "native compiler flags changed",
        )?;
        let mut flags = vec!["-mod=readonly".to_owned(), "-pgo=off".to_owned()];
        if actual_flags.iter().any(|f| f == "-p=1") {
            flags.push("-p=1".into());
        }
        let mut replacements = BTreeMap::new();
        let module = if cooperative {
            Some(
                installation
                    .get("overlay_module_root")
                    .and_then(Value::as_str)
                    .filter(|v| !v.is_empty())
                    .map_or_else(
                        || {
                            Path::new(installation["runtime_root"].as_str().unwrap_or(""))
                                .ancestors()
                                .nth(3)
                                .unwrap_or(Path::new(""))
                                .to_path_buf()
                        },
                        PathBuf::from,
                    ),
            )
        } else {
            None
        };
        if cooperative {
            let generated = strict_json(compile.stream("runtime-overlay")?)?;
            require(
                generated.as_object().is_some_and(|v| {
                    v.len() == 3
                        && v.contains_key("path")
                        && v.contains_key("identity")
                        && v.contains_key("implementation_sha256")
                }) && generated["implementation_sha256"]
                    == installation["runtime_implementation_sha256"]
                    && generated["identity"] == inputs["runtime"],
                "native runtime overlay output identity changed",
            )?;
            let overlay_path = Path::new(field(installation, "cache_root")?)
                .join("runtime-overlay")
                .join(field(&generated, "identity")?)
                .join("overlay.json");
            let overlay_name = overlay_path.to_str().ok_or("overlay path UTF8")?;
            require(
                generated["path"] == overlay_name
                    && installation["admission"]["compiler_inputs"][overlay_name]
                        == digest(&compile.snapshots["overlay-replacements"]),
                "native raw overlay is not frozen/admitted",
            )?;
            let overlay = strict_json(&compile.snapshots["overlay-replacements"])?;
            require(
                overlay
                    .as_object()
                    .is_some_and(|v| v.len() == 1 && v.contains_key("Replace")),
                "native runtime overlay shape changed",
            )?;
            replacements = table(&overlay["Replace"])?;
            for (original, replacement) in &replacements {
                require(
                    !replacement.is_empty(),
                    "native runtime overlay removed source",
                )?;
                for path in [original, replacement] {
                    if Path::new(path).is_file() {
                        compiler.insert(path.clone(), self.pin(Path::new(path))?);
                    }
                }
            }
            let module = module.as_ref().ok_or("runtime overlay module missing")?;
            for name in STATIC {
                let path = module.join(name);
                compiler.insert(path.to_string_lossy().into_owned(), self.pin(&path)?);
            }
            for path in [
                overlay_path.clone(),
                Path::new(field(installation, "runtime_root")?).join("gomad_semantics.go.txt"),
                Path::new(field(installation, "runtime_root")?)
                    .join("gomad_choicewire_generated.go.txt"),
            ] {
                compiler.insert(path.to_string_lossy().into_owned(), self.pin(&path)?);
            }
            flags.extend(["-overlay".into(), overlay_name.to_owned()]);
            let cache = std::str::from_utf8(compile.stream("compiler-module-cache")?)
                .map_err(|e| e.to_string())?
                .trim();
            require(
                Path::new(cache).is_absolute() && !root.starts_with(cache),
                "native cooperative compiler is inside module cache",
            )?;
        }
        let tags = strings(&inputs["tags"])?;
        require(
            tags.windows(2).all(|v| v[0] < v[1]),
            "native build tags not canonical",
        )?;
        if !tags.is_empty() {
            flags.extend(["-tags".into(), tags.join(",")]);
        }
        let source = field(inputs, "source")?;
        let directory = field(inputs, "directory")?;
        let mut selected_list = vec![executable.to_owned(), "list".into()];
        selected_list.extend(flags.clone());
        selected_list.extend(["-deps".into(), "-json".into()]);
        if kind == "go-test" {
            selected_list.push("-test".into());
        }
        selected_list.push(source.into());
        let mut selected_package = vec![executable.to_owned(), "list".into()];
        selected_package.extend(flags.clone());
        selected_package.push(source.into());
        let mut expected = BTreeMap::from([
            (
                "compiler-version",
                vec![executable.into(), "version".into()],
            ),
            (
                "compiler-environment",
                vec![
                    executable.into(),
                    "env".into(),
                    "-json".into(),
                    "GOTOOLDIR".into(),
                    "GOROOT".into(),
                ],
            ),
            ("selected-package", selected_package),
            ("selected-source-closure", selected_list),
        ]);
        if cooperative {
            expected.insert(
                "compiler-module-cache",
                vec![executable.into(), "env".into(), "GOMODCACHE".into()],
            );
            expected.insert(
                "runtime-overlay",
                vec![
                    executable.into(),
                    "run".into(),
                    "-mod=readonly".into(),
                    "./internal/backends/wasm/toolchain/cmd/overlay".into(),
                    root.to_string_lossy().into_owned(),
                    field(installation, "runtime_root")?.into(),
                    Path::new(field(installation, "cache_root")?)
                        .join("runtime-overlay")
                        .to_string_lossy()
                        .into_owned(),
                ],
            );
        }
        let caps = BTreeMap::from([
            ("compiler-version", 64 << 10),
            ("compiler-environment", 64 << 10),
            ("compiler-module-cache", 64 << 10),
            ("runtime-overlay", 4 << 20),
            ("selected-package", 1 << 20),
            ("selected-source-closure", 128 << 20),
            ("compile-module", 4 << 20),
        ]);
        for (observed, phase) in [(&compile, "compile"), (&admission, "admission")] {
            for (name, command) in &observed.commands {
                let cap = caps[name.as_str()];
                require(
                    command.output_limit == cap
                        && [&command.stdout, &command.stderr]
                            .iter()
                            .all(|id| observed.blobs[*id].len() as u64 <= cap),
                    "native original command output cap changed",
                )?;
                let cwd = if name == "runtime-overlay" {
                    module
                        .as_ref()
                        .and_then(|p| p.to_str())
                        .ok_or("module cwd missing")?
                } else {
                    directory
                };
                require(
                    command.cwd == cwd,
                    "native compiler command source directory changed",
                )?;
                if name != "compile-module" {
                    require(
                        command.argv == expected[name.as_str()],
                        "native actual preparation command argv changed",
                    )?;
                }
                let environment = &command.environment;
                require(
                    environment.keys().all(|key| {
                        forced.contains_key(key)
                            || [
                                "PATH",
                                "TMPDIR",
                                "HOME",
                                "GOPATH",
                                "GOMODCACHE",
                                "XDG_CACHE_HOME",
                                "GOMAXPROCS",
                                "GOTMPDIR",
                                "GOROOT",
                                "GOCACHE",
                            ]
                            .contains(&key.as_str())
                    }),
                    "native compiler inherited unrelated host environment",
                )?;
                if name != "compile-module" {
                    require(
                        !environment.contains_key("GOCACHE"),
                        "native compiler used an unbound inherited object cache",
                    )?;
                }
                let has_root = name == "runtime-overlay"
                    || cooperative
                        && ([
                            "selected-package",
                            "selected-source-closure",
                            "compile-module",
                        ]
                        .contains(&name.as_str())
                            || phase == "admission");
                if !has_root {
                    require(
                        !environment.contains_key("GOROOT"),
                        "native compiler inherited a foreign GOROOT",
                    )?;
                }
                if name == "runtime-overlay" {
                    require(
                        forced
                            .iter()
                            .filter(|(key, _)| {
                                !["GOOS", "GOARCH", "GOEXPERIMENT"].contains(&key.as_str())
                            })
                            .all(|(k, v)| environment.get(k) == Some(v))
                            && ["GOOS", "GOARCH", "GOEXPERIMENT"]
                                .iter()
                                .all(|k| !environment.contains_key(*k))
                            && environment
                                .get("GOROOT")
                                .is_some_and(|v| Path::new(v) == root),
                        "native overlay host compiler controls changed",
                    )?;
                } else {
                    require(
                        forced.iter().all(|(k, v)| environment.get(k) == Some(v)),
                        "native preparation compiler controls changed",
                    )?;
                    if cooperative
                        && ([
                            "selected-package",
                            "selected-source-closure",
                            "compile-module",
                        ]
                        .contains(&name.as_str())
                            || phase == "admission")
                    {
                        require(
                            environment
                                .get("GOROOT")
                                .is_some_and(|v| Path::new(v) == root),
                            "native preparation GOROOT changed",
                        )?;
                    }
                }
            }
        }
        let command = &compile.commands["compile-module"];
        let mut prefix = vec![executable.to_owned()];
        if kind == "go-test" {
            prefix.extend(["test".into(), "-c".into()]);
        } else {
            prefix.push("build".into());
        }
        prefix.extend(flags);
        prefix.extend(["-trimpath".into(), "-buildvcs=false".into(), "-o".into()]);
        require(
            command.argv.len() == prefix.len() + 2
                && command.argv[..prefix.len()] == prefix
                && command.argv.last().map(String::as_str) == Some(source),
            "native actual compile command shape changed",
        )?;
        let output = Path::new(&command.argv[prefix.len()]);
        let temporary = output.parent().ok_or("compile output directory missing")?;
        let stem = format!(".rust-preparation-{}-", command.owner_pid);
        let suffix = temporary
            .file_name()
            .and_then(|s| s.to_str())
            .and_then(|s| s.strip_prefix(&stem));
        require(
            output.file_name().is_some_and(|s| s == "module.wasm")
                && temporary.parent() == Some(Path::new(field(installation, "cache_root")?))
                && suffix.is_some_and(|s| !s.is_empty() && s.bytes().all(|b| b.is_ascii_digit())),
            "native compile output is outside actual owner private preparation",
        )?;
        let mut compile_inputs = json!({"schema":"gomad.native-compilation-inputs/v1","profile":installation["profile"]});
        for key in [
            "compiler",
            "compiler_path",
            "compiler_sha256",
            "go_version",
            "sources",
            "runtime",
            "runtime_implementation",
            "kind",
            "source",
            "directory",
            "tags",
            "build_flags",
            "environment",
        ] {
            compile_inputs[key] = inputs
                .get(key)
                .ok_or_else(|| format!("missing native compile input {key}"))?
                .clone();
        }
        let key = digest(&canonical(&compile_inputs)?);
        let objects = Path::new(field(installation, "cache_root")?)
            .join("compilation-v1")
            .join(key)
            .join("objects");
        require(
            command
                .environment
                .get("GOCACHE")
                .is_some_and(|v| Path::new(v) == objects),
            "native compiler did not use its own original per-entry object cache",
        )?;
        let expected_sources = table(&inputs["sources"])?;
        let (compile_generated, admission_generated) = if kind == "go-test" {
            let import_path = std::str::from_utf8(compile.stream("selected-package")?)
                .map_err(|e| e.to_string())?
                .trim();
            (
                Some(GeneratedTest::from_observations(
                    &compile,
                    import_path,
                    &expected_sources,
                )?),
                Some(GeneratedTest::from_observations(
                    &admission,
                    import_path,
                    &expected_sources,
                )?),
            )
        } else {
            (None, None)
        };
        let mut before_generated = BTreeSet::new();
        let mut after_generated = BTreeSet::new();
        let before = self.collect_sources(
            compile.stream("selected-source-closure")?,
            &root,
            &replacements,
            compile_generated.as_ref(),
            &mut before_generated,
        )?;
        let after = self.collect_sources(
            admission.stream("selected-source-closure")?,
            &root,
            &replacements,
            admission_generated.as_ref(),
            &mut after_generated,
        )?;
        require(
            before_generated == after_generated,
            "native generated test source membership changed",
        )?;
        require(
            before == after
                && before == expected_sources
                && compiler == table(&inputs["compiler"])?,
            "native complete source/compiler membership differs from independent raw projection",
        )?;
        for path in compiler
            .keys()
            .chain(before.keys())
            .chain(std::iter::once(&executable.to_owned()))
        {
            if !before_generated.contains(path) {
                self.file_bytes(Path::new(path))?;
            }
        }
        Ok(PreparedInputs {
            compiler,
            sources: before,
        })
    }
}

pub(crate) fn same_compiler_inventory(
    before: &BTreeMap<String, String>,
    root: &Path,
    after: &BTreeMap<String, String>,
    after_root: &Path,
) -> Result<(), String> {
    require(
        before == after && root == after_root,
        "native compiler pre/post complete inventory changed",
    )
}
