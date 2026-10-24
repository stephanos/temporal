use super::{
    BTreeMap, Digest, FILE_BOUND, Path, Role, Sha256, Snapshot, Value, digest, feature_bindings,
    fs, json, read_file, require,
};

const COMMON: &[&str] = &[
    "build.rs",
    "Cargo.toml",
    "../Cargo.toml",
    "../Cargo.lock",
    "../rust-toolchain.toml",
];
const RUNTIME: &[&str] = &[
    "overlay.go",
    "runtime_wasm.go.txt",
    "runtime_inspect_wasm.go.txt",
    "net_stream_wasm.go.txt",
    "net_stream_status_generated.go.txt",
];
const HOST_EXTRA: &[&str] = &[
    "src/measurement/testdata/sqlite_conn.go.orig",
    "src/measurement/testdata/LICENSE",
    "src/measurement/testdata/guest-profile.json",
    "../../internal/backends/wasm/command/testdata/scenario/network.json",
    "../../internal/model/choice/schema/choicewire.json",
    "../../internal/model/choice/schema/choicewire.go.tmpl",
    "../../internal/model/choice/schema/choicewire_runtime.go.tmpl",
    "../../internal/model/choice/schema/choicewire_test.go.tmpl",
    "../../internal/model/runtimesource/gomad_semantics.go.txt",
    "../../internal/model/choice/trace.go",
    "../../internal/model/choice/tape.go",
];
#[path = "../../../../wasmhost/build_identity.rs"]
pub(crate) mod helper_identity;

fn source_names(root: &Path, directory: &Path, names: &mut Vec<String>) -> Result<(), String> {
    require(names.len() < 200_000, "build source closure exceeds bound")?;
    for entry in fs::read_dir(directory).map_err(|e| e.to_string())? {
        let entry = entry.map_err(|e| e.to_string())?;
        let path = entry.path();
        let kind = entry.file_type().map_err(|e| e.to_string())?;
        require(
            !kind.is_symlink(),
            "native source closure contains symlinks",
        )?;
        if kind.is_dir() {
            source_names(root, &path, names)?;
        } else if kind.is_file() && path.extension().is_some_and(|v| v == "rs") {
            names.push(
                path.strip_prefix(root)
                    .map_err(|e| e.to_string())?
                    .to_str()
                    .ok_or("source filename is not UTF8")?
                    .into(),
            );
        }
    }
    Ok(())
}
fn inventory(
    repo: &Path,
    snapshot: &Snapshot,
    helper: bool,
) -> Result<BTreeMap<String, Vec<u8>>, String> {
    let root = repo
        .join("tools/gomad/rust")
        .join(if helper { "wasmhost" } else { "gomad" });
    let mut names = Vec::new();
    source_names(&root, &root.join("src"), &mut names)?;
    require(!names.is_empty(), "native source closure is empty")?;
    names.extend(COMMON.iter().map(|v| (*v).into()));
    if helper {
        names.push("build_identity.rs".into());
    } else {
        names.extend(HOST_EXTRA.iter().map(|v| (*v).into()));
        names.extend(
            RUNTIME
                .iter()
                .map(|name| format!("../../internal/backends/wasm/toolchain/{name}")),
        );
    }
    let mut files = BTreeMap::new();
    let mut total = 0u64;
    for name in names {
        let path = root.join(&name).canonicalize().map_err(|e| e.to_string())?;
        require(
            path.starts_with(repo),
            "build source escaped selected checkout",
        )?;
        let raw = read_file(&path, FILE_BOUND)?;
        total = total
            .checked_add(raw.len() as u64)
            .ok_or("source closure size overflow")?;
        require(
            total <= FILE_BOUND,
            "build source closure exceeds original bound",
        )?;
        require(
            snapshot
                .files
                .get(path.to_str().ok_or("nonUTF8 source path")?)
                == Some(&digest(&raw)),
            "native compiler/source input is not frozen",
        )?;
        files.insert(name, raw);
    }
    Ok(files)
}
fn implementation(
    files: &BTreeMap<String, Vec<u8>>,
    domain: &[u8],
    features: &[&str],
    compiler: &str,
    target: &str,
) -> String {
    let mut hash = Sha256::new();
    hash.update(domain);
    for (name, value) in feature_bindings(features, target) {
        hash.update(name.as_bytes());
        hash.update([0]);
        hash.update(value.as_bytes());
        hash.update([0]);
    }
    hash.update(compiler.as_bytes());
    for (name, raw) in files {
        hash.update((name.len() as u64).to_be_bytes());
        hash.update(name.as_bytes());
        hash.update((raw.len() as u64).to_be_bytes());
        hash.update(raw);
    }
    format!("{:x}", hash.finalize())
}
pub(super) fn identities(
    repo: &Path,
    snapshot: &Snapshot,
    role: Role,
    compiler: &str,
    target: &str,
) -> Result<(Value, Value), String> {
    let helper = inventory(repo, snapshot, true)?;
    let helper = helper_identity::implementation(
        &helper,
        &feature_bindings(&role.helper_features(), target),
        compiler.as_bytes(),
    )?;
    if role.helper() {
        return Ok((json!({"host":null,"helper":helper}), Value::Null));
    }
    let host = inventory(repo, snapshot, false)?;
    let identity = implementation(
        &host,
        b"gomad.host-implementation/v1\0",
        &role.features(),
        compiler,
        target,
    );
    let mut runtime = Sha256::new();
    runtime.update(b"gomad-wasm-runtime-implementation/v1\0");
    for name in RUNTIME {
        let raw = host
            .get(&format!("../../internal/backends/wasm/toolchain/{name}"))
            .ok_or("missing runtime identity owner")?;
        runtime.update((raw.len() as u64).to_be_bytes());
        runtime.update(raw);
    }
    Ok((
        json!({"host":identity,"helper":helper}),
        json!(format!("{:x}", runtime.finalize())),
    ))
}
