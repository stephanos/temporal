//! Pinned Go protobuf generators and framework generation leaves.
use crate::{Error, Result, support};
use std::{
    collections::BTreeMap,
    env,
    ffi::OsString,
    fs,
    io::{self, Write},
    os::unix::{fs::PermissionsExt, process::ExitStatusExt},
    path::{Path, PathBuf},
    process::{Command, Output},
};
use umpire_guest_driver::context::{ToolRoot, cargo_home};

#[path = "proto_controls.rs"]
pub mod controls;

const GO: &str = "protoc-gen-go-v1.36.6";
const GRPC: &str = "protoc-gen-go-grpc-v1.3.0";

/// Paths are relative to the execution root unless absolute.
/// Only the selected destination subtree participates in a freshness check.
pub struct GoSpec {
    pub destination: PathBuf,
    pub includes: Vec<PathBuf>,
    pub inputs: Vec<PathBuf>,
    pub grpc: bool,
    pub check_subtree: Option<PathBuf>,
    pub plugin_root: PathBuf,
}

struct Context {
    root: PathBuf,
    home: PathBuf,
    target: PathBuf,
}
impl Context {
    fn new(root: &Path) -> Result<Self> {
        let caller = env::current_dir()?;
        let root = fs::canonicalize(root)?;
        let target = env::var_os("CARGO_TARGET_DIR")
            .map_or_else(|| root.join("target"), |p| absolute(&caller, Path::new(&p)));
        Ok(Self { root, home: cargo_home(&caller)?, target })
    }
    fn command(&self, program: impl AsRef<std::ffi::OsStr>) -> Command {
        let mut command = Command::new(program);
        command
            .current_dir(&self.root)
            .env("CARGO_HOME", &self.home)
            .env("CARGO_TARGET_DIR", &self.target)
            .env("MISE_AUTO_INSTALL", "false")
            .env("MISE_EXEC_AUTO_INSTALL", "false");
        command
    }
    fn mise(&self) -> Command {
        let mut command = self.command("mise");
        command.args(["exec", "--"]);
        command
    }
}
fn absolute(root: &Path, path: &Path) -> PathBuf {
    if path.is_absolute() { path.to_owned() } else { root.join(path) }
}
fn code(output: &Output) -> i32 {
    output.status.code().unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0))
}
fn display(output: &Output) -> Result<i32> {
    io::stdout().write_all(&output.stdout)?;
    io::stderr().write_all(&output.stderr)?;
    Ok(code(output))
}
fn execute(command: &mut Command) -> Result<Output> {
    Ok(crate::gate::scheduler::execute(command, false, false)?)
}

fn regular_executable(path: &Path) -> Result<()> {
    let metadata = fs::symlink_metadata(path).map_err(|error| {
        Error::Invalid(format!(
            "missing pinned generator {}; run cargo xtask proto-tools: {error}",
            path.display()
        ))
    })?;
    if !metadata.is_file() || metadata.permissions().mode() & 0o111 == 0 {
        return Err(Error::Invalid(format!(
            "generator must be a regular executable, not a symlink: {}",
            path.display()
        )));
    }
    Ok(())
}
fn version(
    context: &Context,
    path: &Path,
    grpc: bool,
    run: &mut impl FnMut(&mut Command) -> Result<Output>,
) -> Result<i32> {
    regular_executable(path)?;
    let output = run(context.command(path).arg("--version"))?;
    if !output.status.success() {
        return display(&output);
    }
    let actual = String::from_utf8_lossy(&output.stdout);
    let actual = actual.trim_end_matches('\n');
    let valid = if grpc {
        actual == "protoc-gen-go-grpc 1.3.0"
    } else {
        matches!(actual, "protoc-gen-go v1.36.6" | "protoc-gen-go-v1.36.6 v1.36.6")
    };
    if !valid {
        return Err(Error::Invalid(format!("wrong pinned generator version: {}", path.display())));
    }
    Ok(0)
}
fn plugin_directory(root: &Path) -> Result<PathBuf> {
    let directory = root.join(".bin");
    if let Ok(metadata) = fs::symlink_metadata(&directory) {
        if !metadata.is_dir() || metadata.file_type().is_symlink() {
            return Err(Error::Invalid("generator directory must not be a symlink".into()));
        }
    }
    fs::create_dir_all(&directory)?;
    if fs::canonicalize(&directory)? != directory {
        return Err(Error::Invalid("generator directory escapes framework root".into()));
    }
    Ok(directory)
}

/// Provision exact Go plugin pins using the installed mise Go toolchain.
/// Mise auto-install stays disabled; no toolchain downloader is implemented here.
pub fn tools(root: &Path) -> Result<i32> {
    tools_with(&Context::new(root)?, &mut execute)
}
fn tools_with(
    context: &Context,
    run: &mut impl FnMut(&mut Command) -> Result<Output>,
) -> Result<i32> {
    let directory = plugin_directory(&context.root)?;
    let lock = directory.join("proto-tools.lock");
    fs::create_dir(&lock).map_err(|error| {
        Error::Invalid(format!("generator installation is locked: {}: {error}", lock.display()))
    })?;
    let result = (|| {
        for (name, destination, package, grpc) in [
            ("protoc-gen-go", GO, "google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6", false),
            (
                "protoc-gen-go-grpc",
                GRPC,
                "google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0",
                true,
            ),
        ] {
            let destination = directory.join(destination);
            match fs::symlink_metadata(&destination) {
                Ok(metadata) if metadata.file_type().is_symlink() => {
                    return Err(Error::Invalid(format!(
                        "generator must not be a symlink: {}",
                        destination.display()
                    )));
                }
                Ok(_) => (),
                Err(error) if error.kind() == io::ErrorKind::NotFound => {
                    let status = display(&run(context
                        .mise()
                        .args(["go", "install", package])
                        .env("GOBIN", &lock))?)?;
                    if status != 0 {
                        return Ok(status);
                    }
                    let staged = lock.join(name);
                    let status = version(context, &staged, grpc, run)?;
                    if status != 0 {
                        return Ok(status);
                    }
                    plugin_directory(&context.root)?;
                    if fs::symlink_metadata(&destination).is_ok() {
                        return Err(Error::Invalid(format!(
                            "generator appeared during installation: {}",
                            destination.display()
                        )));
                    }
                    fs::rename(&staged, &destination)?;
                }
                Err(error) => return Err(error.into()),
            }
            let status = version(context, &destination, grpc, run)?;
            if status != 0 {
                return Ok(status);
            }
        }
        Ok(0)
    })();
    fs::remove_dir_all(&lock)?;
    result
}

fn snapshot(root: &Path) -> Result<BTreeMap<PathBuf, Option<String>>> {
    fn visit(
        root: &Path,
        relative: &Path,
        result: &mut BTreeMap<PathBuf, Option<String>>,
    ) -> Result<()> {
        let path = root.join(relative);
        let metadata = fs::symlink_metadata(&path)?;
        if metadata.file_type().is_symlink() || (!metadata.is_file() && !metadata.is_dir()) {
            return Err(Error::Invalid(format!("not generated: {}", path.display())));
        }
        if metadata.is_file() {
            result.insert(relative.to_owned(), Some(support::hash(&fs::read(&path)?)));
        } else {
            result.insert(relative.to_owned(), None);
            for entry in fs::read_dir(path)? {
                visit(root, &relative.join(entry?.file_name()), result)?;
            }
        }
        Ok(())
    }
    let mut result = BTreeMap::new();
    visit(root, Path::new(""), &mut result)?;
    Ok(result)
}

/// Generate or compare a consumer's Go protobuf tree without changing checked sources in check mode.
pub fn generate(root: &Path, spec: &GoSpec, check: bool) -> Result<i32> {
    generate_with(&Context::new(root)?, spec, check, &mut execute)
}
fn generate_with(
    context: &Context,
    spec: &GoSpec,
    check: bool,
    run: &mut impl FnMut(&mut Command) -> Result<Output>,
) -> Result<i32> {
    let plugins = fs::canonicalize(absolute(&context.root, &spec.plugin_root))?;
    let go = plugins.join(".bin").join(GO);
    let grpc = plugins.join(".bin").join(GRPC);
    let status = version(context, &go, false, run)?;
    if status != 0 {
        return Ok(status);
    }
    if spec.grpc {
        let status = version(context, &grpc, true, run)?;
        if status != 0 {
            return Ok(status);
        }
    }
    let destination = absolute(&context.root, &spec.destination);
    let temporary = if check {
        let base = absolute(&env::current_dir()?, &env::temp_dir());
        Some(tempfile::Builder::new().prefix("umpire-proto-").tempdir_in(base)?)
    } else {
        None
    };
    let output = temporary.as_ref().map_or(destination.as_path(), |directory| directory.path());
    let mut command = context.mise();
    command.arg("protoc").arg(option("--plugin=protoc-gen-go=", &go));
    if spec.grpc {
        command.arg(option("--plugin=protoc-gen-go-grpc=", &grpc));
    }
    for include in &spec.includes {
        command.arg(option("--proto_path=", include));
    }
    command.arg(option("--go_out=", output)).arg("--go_opt=paths=source_relative");
    if spec.grpc {
        command.arg(option("--go-grpc_out=", output)).arg("--go-grpc_opt=paths=source_relative");
    }
    command.args(&spec.inputs);
    let result = (|| {
        let status = display(&run(&mut command)?)?;
        if status != 0 || !check {
            return Ok(status);
        }
        let subtree = spec.check_subtree.as_deref().unwrap_or(Path::new(""));
        let expected = snapshot(&destination.join(subtree))?;
        let actual = snapshot(&output.join(subtree))?;
        let mut changed: Vec<_> = expected
            .keys()
            .chain(actual.keys())
            .filter(|path| expected.get(*path) != actual.get(*path))
            .cloned()
            .collect();
        changed.sort();
        changed.dedup();
        for path in &changed {
            writeln!(
                io::stderr(),
                "protobuf output differs: {}",
                destination.join(subtree).join(path).display()
            )?;
        }
        Ok(i32::from(!changed.is_empty()))
    })();
    if let Some(temporary) = temporary {
        temporary.close()?;
    }
    result
}
fn option(prefix: &str, path: &Path) -> OsString {
    let mut value = OsString::from(prefix);
    value.push(path);
    value
}

/// Framework selectors retain the core/kettle plugin and input closures.
pub fn go(arguments: Vec<OsString>) -> Result<i32> {
    let check = arguments.get(1).is_some_and(|value| value == "--check");
    if arguments.is_empty() || arguments.len() > 2 || (arguments.len() == 2 && !check) {
        return Err(Error::Invalid("usage: cargo xtask proto-go <core|kettle> [--check]".into()));
    }
    let kettle = arguments[0] == "kettle";
    if !kettle && arguments[0] != "core" {
        return Err(Error::Invalid("usage: cargo xtask proto-go <core|kettle> [--check]".into()));
    }
    let root = ToolRoot::local()?;
    let spec = framework_spec(root.path(), kettle);
    generate(root.path(), &spec, check)
}
fn framework_spec(root: &Path, kettle: bool) -> GoSpec {
    GoSpec {
        destination: if kettle { "reference/kettle/gen" } else { "hosts/go/gen" }.into(),
        includes: if kettle {
            vec!["reference/kettle/proto".into(), "protocol/proto".into()]
        } else {
            vec!["protocol/proto".into()]
        },
        inputs: if kettle {
            ["kettle", "probe"]
                .iter()
                .map(|name| PathBuf::from(format!("reference/kettle/proto/kettle/v1/{name}.proto")))
                .collect()
        } else {
            ["model", "realization", "annotations", "run"]
                .iter()
                .map(|name| PathBuf::from(format!("protocol/proto/umpire/{name}/v1/{name}.proto")))
                .collect()
        },
        grpc: kettle,
        check_subtree: None,
        plugin_root: root.to_owned(),
    }
}

/// Generic consumer bridge; no consumer schema or workspace names enter the framework.
pub fn go_spec(arguments: Vec<OsString>) -> Result<i32> {
    let mut root = None;
    let mut spec = GoSpec {
        destination: PathBuf::new(),
        includes: vec![],
        inputs: vec![],
        grpc: false,
        check_subtree: None,
        plugin_root: PathBuf::new(),
    };
    let mut check = false;
    let mut arguments = arguments.into_iter();
    while let Some(argument) = arguments.next() {
        if argument == "--check" {
            check = true;
            continue;
        }
        if argument == "--grpc" {
            spec.grpc = true;
            continue;
        }
        let slot = match argument.to_str() {
            Some("--root") => &mut root,
            Some("--check-subtree") => &mut spec.check_subtree,
            Some("--destination" | "--plugin-root" | "--include" | "--input") => {
                let value = PathBuf::from(
                    arguments
                        .next()
                        .ok_or_else(|| Error::Invalid("missing protobuf path argument".into()))?,
                );
                match argument.to_str() {
                    Some("--destination") => spec.destination = value,
                    Some("--plugin-root") => spec.plugin_root = value,
                    Some("--include") => spec.includes.push(value),
                    _ => spec.inputs.push(value),
                }
                continue;
            }
            _ => return Err(Error::Invalid("unknown proto-go-spec argument".into())),
        };
        *slot = Some(PathBuf::from(
            arguments
                .next()
                .ok_or_else(|| Error::Invalid("missing protobuf path argument".into()))?,
        ));
    }
    let root = root.ok_or_else(|| Error::Invalid("proto-go-spec requires --root".into()))?;
    if spec.destination.as_os_str().is_empty()
        || spec.plugin_root.as_os_str().is_empty()
        || spec.includes.is_empty()
        || spec.inputs.is_empty()
    {
        return Err(Error::Invalid(
            "proto-go-spec requires --destination, --plugin-root, --include and --input".into(),
        ));
    }
    if let Some(subtree) = &spec.check_subtree {
        if subtree.components().any(|part| {
            !matches!(part, std::path::Component::Normal(_) | std::path::Component::CurDir)
        }) {
            return Err(Error::Invalid("check subtree must remain below the destination".into()));
        }
    }
    generate(&root, &spec, check)
}
