use super::*;
use std::{
    fs,
    path::{Path, PathBuf},
    process::{Command, Stdio},
    time::{Duration, Instant},
};

const RUNTIME_EVIDENCE: &str =
    "rust:tests/inventory/runtime.rs:wasip1_runtime_draw_and_clock_references_are_reviewed";

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct SourceInventory {
    schema: String,
    platform: String,
    go_version: String,
    overlay_identity: String,
    implementation_sha256: String,
    sources: BTreeMap<PathBuf, String>,
    overlay_inputs: BTreeMap<String, PathBuf>,
    sites: BTreeMap<String, BTreeMap<String, u64>>,
}

struct Scratch(PathBuf, bool);
impl Drop for Scratch {
    fn drop(&mut self) {
        if !self.1 {
            fs::remove_dir_all(&self.0).unwrap();
        }
    }
}

fn derived() -> Result<(SourceInventory, Scratch), String> {
    let module = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../..")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let go = std::env::var_os("GOMAD_STOCK_GO").unwrap_or_else(|| "go".into());
    let command = || {
        let mut command = Command::new(&go);
        command
            .current_dir(&module)
            .env_remove("GOROOT")
            .env("GOTOOLCHAIN", "local")
            .env("GOENV", "off")
            .env("GOWORK", "off")
            .env("GOFLAGS", "")
            .env("GOMAXPROCS", "1");
        command
    };
    let root = command()
        .args(["env", "GOROOT"])
        .output()
        .map_err(|e| e.to_string())?;
    if !root.status.success() {
        return Err("stock compiler root query failed".into());
    }
    let root = PathBuf::from(
        std::str::from_utf8(&root.stdout)
            .map_err(|e| e.to_string())?
            .trim(),
    );
    let version = command()
        .arg("version")
        .output()
        .map_err(|e| e.to_string())?;
    let compiler_version = std::str::from_utf8(&version.stdout)
        .map_err(|e| e.to_string())?
        .trim()
        .to_string();
    let reviewed = parse(INVENTORY)?.reviewed;
    if !version.status.success()
        || compiler_version.split_whitespace().nth(2) != Some(reviewed.go_toolchain.as_str())
    {
        return Err("pinned compiler executable version mismatch".into());
    }
    let compiler = root
        .join("bin/go")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let compiler_sha256 = format!(
        "{:x}",
        Sha256::digest(fs::read(&compiler).map_err(|e| e.to_string())?)
    );
    let retained = std::env::var_os("GOMAD_INVENTORY_EVIDENCE_ROOT");
    let scratch = Scratch(
        retained.clone().map(PathBuf::from).unwrap_or_else(|| {
            std::env::temp_dir().join(format!("gomad-runtime-inventory-{}", std::process::id()))
        }),
        retained.is_some(),
    );
    fs::create_dir(&scratch.0).map_err(|e| e.to_string())?;
    let output = scratch.0.join("sites.json");
    let stderr = scratch.0.join("stderr.log");
    let mut child = command()
        .args([
            "run",
            "-mod=readonly",
            "-p=1",
            "./internal/backends/wasm/toolchain/cmd/runtimeinventory",
        ])
        .arg(&root)
        .arg(module.join("internal/model/runtimesource"))
        .arg(scratch.0.join("overlay"))
        .stdout(Stdio::from(
            fs::File::create(&output).map_err(|e| e.to_string())?,
        ))
        .stderr(Stdio::from(
            fs::File::create(&stderr).map_err(|e| e.to_string())?,
        ))
        .spawn()
        .map_err(|e| e.to_string())?;
    let deadline = Instant::now() + Duration::from_secs(120);
    loop {
        if let Some(status) = child.try_wait().map_err(|e| e.to_string())? {
            fs::write(scratch.0.join("exit.json"), serde_json::to_vec(&serde_json::json!({"exit": status.code(), "go": go, "stock":root,"compiler":compiler,"compiler_version":compiler_version,"compiler_sha256":compiler_sha256,"argv":[go.clone(),"run".into(),"-mod=readonly".into(),"-p=1".into(),"./internal/backends/wasm/toolchain/cmd/runtimeinventory".into(),root.as_os_str().to_owned(),module.join("internal/model/runtimesource").into_os_string(),scratch.0.join("overlay").into_os_string()],"source_sha256":format!("{:x}",Sha256::digest(include_bytes!("runtime.rs")))})).map_err(|e|e.to_string())?).map_err(|e|e.to_string())?;
            if !status.success() {
                return Err(format!(
                    "runtime source derivation failed: {}",
                    fs::read_to_string(&stderr).map_err(|e| e.to_string())?
                ));
            }
            break;
        }
        if Instant::now() >= deadline {
            child.kill().map_err(|e| e.to_string())?;
            child.wait().map_err(|e| e.to_string())?;
            return Err("runtime source derivation deadline".into());
        }
        std::thread::sleep(Duration::from_millis(10));
    }
    if fs::metadata(&output).map_err(|e| e.to_string())?.len() > 16 << 20 {
        return Err("runtime source inventory capacity".into());
    }
    let raw = fs::read(&output).map_err(|e| e.to_string())?;
    let inventory: SourceInventory = serde_json::from_slice(&raw).map_err(|e| e.to_string())?;
    authenticate(&inventory, &root, &module)?;
    Ok((inventory, scratch))
}

fn authenticate(source: &SourceInventory, stock: &Path, module: &Path) -> Result<(), String> {
    let reviewed = parse(INVENTORY)?.reviewed;
    let digest = |text: &str| {
        text.len() == 64
            && text
                .bytes()
                .all(|byte| byte.is_ascii_hexdigit() && !byte.is_ascii_uppercase())
    };
    if source.schema != "gomad.runtime-source-sites/v1"
        || source.platform != "wasip1/wasm"
        || source.go_version != reviewed.go_toolchain
        || !digest(&source.overlay_identity)
        || !source
            .implementation_sha256
            .strip_prefix("sha256:")
            .is_some_and(digest)
        || source.sites.keys().map(String::as_str).collect::<Vec<_>>()
            != ["runtime-clock", "runtime-draw", "runtime-other"]
    {
        return Err("runtime source/toolchain identity mismatch".into());
    }
    collector_patch_policy(&source.overlay_inputs)?;
    for required in [
        stock.join("VERSION"),
        module.join("internal/backends/wasm/toolchain/overlay.go"),
        module.join("internal/backends/wasm/toolchain/runtime_wasm.go.txt"),
        module.join("internal/backends/wasm/toolchain/cmd/runtimeinventory/main.go"),
    ] {
        if !source.sources.contains_key(&required) {
            return Err(format!(
                "runtime source identity missing {}",
                required.display()
            ));
        }
    }
    for (path, expected) in &source.sources {
        if !path.is_absolute()
            || !digest(expected)
            || format!(
                "{:x}",
                Sha256::digest(fs::read(path).map_err(|e| e.to_string())?)
            ) != *expected
        {
            return Err(format!(
                "runtime source identity changed {}",
                path.display()
            ));
        }
    }
    let mut implementation = Sha256::new();
    implementation.update(b"gomad-wasm-runtime-implementation/v1\0");
    for name in [
        "overlay.go",
        "runtime_wasm.go.txt",
        "runtime_inspect_wasm.go.txt",
        "net_stream_wasm.go.txt",
        "net_stream_status_generated.go.txt",
    ] {
        let raw = fs::read(module.join("internal/backends/wasm/toolchain").join(name))
            .map_err(|e| e.to_string())?;
        implementation.update((raw.len() as u64).to_be_bytes());
        implementation.update(raw);
    }
    if source.implementation_sha256 != format!("sha256:{:x}", implementation.finalize()) {
        return Err("runtime overlay implementation identity changed".into());
    }
    let mut overlay = Sha256::new();
    overlay.update(b"gomad-wasm-runtime-overlay/v1\0");
    for (name, path) in &source.overlay_inputs {
        if !source.sources.contains_key(path) {
            return Err(format!("overlay input {name} lacks source identity"));
        }
        let raw = fs::read(path).map_err(|e| e.to_string())?;
        for part in [name.as_bytes(), raw.as_slice()] {
            overlay.update((part.len() as u64).to_be_bytes());
            overlay.update(part);
        }
    }
    if source.overlay_identity != format!("{:x}", overlay.finalize()) {
        return Err("runtime overlay input identity changed".into());
    }
    for (kind, sites) in &source.sites {
        if sites.is_empty()
            || sites
                .iter()
                .any(|(key, count)| !key.starts_with("wasip1/wasm ") || *count == 0)
        {
            return Err(format!("runtime derived {kind} has invalid keys/counts"));
        }
    }
    Ok(())
}

fn collector_patch_policy(inputs: &BTreeMap<String, PathBuf>) -> Result<(), String> {
    for name in inputs.keys() {
        if let Some(path) = name
            .strip_prefix("output/")
            .or_else(|| name.strip_prefix("stock/"))
        {
            let file = path.rsplit('/').next().unwrap();
            if file.starts_with("mgc") && file.ends_with(".go") {
                return Err(format!("collector-file patch policy forbids {name}"));
            }
        }
    }
    Ok(())
}

#[test]
fn source_inventory_preserves_the_collector_file_patch_prohibition() {
    for file in [
        "mgc.go",
        "mgcpacer.go",
        "mgcsweep.go",
        "mgcmark.go",
        "runtime/mgcscavenge.go",
    ] {
        for kind in ["output", "stock"] {
            let name = format!("{kind}/{file}");
            assert!(
                collector_patch_policy(&BTreeMap::from([(name.clone(), PathBuf::from("unused"))]))
                    .unwrap_err()
                    .contains(&name)
            );
        }
    }
    collector_patch_policy(&BTreeMap::from([(
        "output/gomad_wasm.go".into(),
        PathBuf::from("unused"),
    )]))
    .unwrap();
}

fn reviewed(kind: &str, table: &str) -> BTreeMap<String, u64> {
    let mut sites = BTreeMap::new();
    for line in table.lines() {
        let fields: Vec<&str> = line.split('|').collect();
        let (count_at, key_end) = match kind {
            "runtime-draw" => (5, 5),
            "runtime-clock" => (3, 3),
            "runtime-other" => (4, 4),
            _ => unreachable!(),
        };
        assert_eq!(
            fields.len(),
            if kind == "runtime-draw" { 10 } else { 6 },
            "{line}"
        );
        assert_eq!(fields[0], "wasip1/wasm");
        let count = fields[count_at].parse().unwrap();
        assert!(count > 0);
        if kind == "runtime-draw" {
            assert!(["target-ordered", "host-timed", "reference"].contains(&fields[6]));
            assert!(["process-purpose", "M-local", "wrapper", "none"].contains(&fields[7]));
            assert!(["active", "disabled", "contract-excluded", "blocked"].contains(&fields[8]));
            assert!(!fields[9].is_empty());
            assert!(fields[6] != "host-timed" || fields[8] != "active" || fields[7] == "M-local");
        } else if kind == "runtime-clock" {
            assert!(
                [
                    "implementation",
                    "virtual",
                    "inactive",
                    "host-by-design",
                    "escape",
                    "unrelated"
                ]
                .contains(&fields[4])
            );
            assert!(
                !["escape", "inactive", "host-by-design"].contains(&fields[4])
                    || !fields[5].is_empty()
            );
        } else {
            assert_eq!(fields[3], "patch");
            assert!(!fields[5].is_empty());
        }
        assert!(
            sites.insert(fields[..key_end].join(" "), count).is_none(),
            "duplicate {line}"
        );
    }
    sites
}

fn coverage(
    entries: &[Entry],
    derived: &BTreeMap<String, BTreeMap<String, u64>>,
    history: &str,
) -> Result<(), String> {
    for (kind, active) in derived {
        let prefix = format!("{kind}:");
        let rows: BTreeMap<&str, &Entry> = entries
            .iter()
            .filter_map(|entry| entry.site.strip_prefix(&prefix).map(|key| (key, entry)))
            .collect();
        for site in active.keys() {
            let row = rows
                .get(site.as_str())
                .ok_or_else(|| format!("runtime site {site:?} has no inventory row"))?;
            if row.disposition != "controlled"
                || row.rule.trim().is_empty()
                || !row
                    .evidence
                    .iter()
                    .any(|reference| reference == RUNTIME_EVIDENCE)
            {
                return Err(format!(
                    "runtime site {site} lacks reviewed disposition/rule/runtime evidence"
                ));
            }
        }
        for (site, entry) in rows {
            if !active.contains_key(site) && !retired_identity(entry, history) {
                return Err(format!(
                    "inventory runtime site {site} no longer has a source"
                ));
            }
        }
    }
    Ok(())
}

#[test]
fn wasip1_runtime_draw_and_clock_references_are_reviewed() {
    let (source, _scratch) = derived().unwrap();
    for (kind, table) in [
        ("runtime-draw", include_str!("runtime-draw.txt")),
        ("runtime-clock", include_str!("runtime-clock.txt")),
        ("runtime-other", include_str!("runtime-other.txt")),
    ] {
        assert_eq!(source.sites[kind], reviewed(kind, table), "{kind}");
    }
    let inventory = parse(INVENTORY).unwrap();
    coverage(&inventory.entries, &source.sites, IDENTITY_HISTORY).unwrap();
    for (kind, sites) in &source.sites {
        for site in sites.keys() {
            let key = format!("{kind}:{site}");
            let mut entries = inventory.entries.clone();
            entries.retain(|entry| entry.site != key);
            assert!(
                coverage(&entries, &source.sites, IDENTITY_HISTORY)
                    .unwrap_err()
                    .contains(site)
            );
            for field in ["disposition", "rule", "evidence"] {
                let mut entries = inventory.entries.clone();
                let row = entries.iter_mut().find(|row| row.site == key).unwrap();
                match field {
                    "disposition" => row.disposition = "recorded".into(),
                    "rule" => row.rule.clear(),
                    "evidence" => row.evidence.clear(),
                    _ => unreachable!(),
                }
                assert!(
                    coverage(&entries, &source.sites, IDENTITY_HISTORY)
                        .unwrap_err()
                        .contains(site)
                );
            }
            let mut removed = source.sites.clone();
            removed.get_mut(kind).unwrap().remove(site);
            assert!(
                coverage(&inventory.entries, &removed, IDENTITY_HISTORY)
                    .unwrap_err()
                    .contains("no longer has")
            );
        }
    }
}

fn reviewed_sets() -> BTreeMap<String, BTreeMap<String, u64>> {
    [
        ("runtime-draw", include_str!("runtime-draw.txt")),
        ("runtime-clock", include_str!("runtime-clock.txt")),
        ("runtime-other", include_str!("runtime-other.txt")),
    ]
    .into_iter()
    .map(|(kind, table)| (kind.to_string(), reviewed(kind, table)))
    .collect()
}

#[test]
fn runtime_historical_retirement_keeps_appended_sites_active() {
    let inventory = parse(INVENTORY).unwrap();
    let derived = reviewed_sets();
    for kind in derived.keys() {
        let mut entries = inventory.entries.clone();
        let mut changed = derived.clone();
        for entry in &mut entries {
            if let Some(site) = entry.site.strip_prefix(&format!("{kind}:")) {
                let mut candidate = entry.clone();
                candidate.disposition = "rejected".into();
                if retired_identity(&candidate, IDENTITY_HISTORY) {
                    entry.disposition = "rejected".into();
                    changed.get_mut(kind).unwrap().remove(site);
                }
            }
        }
        coverage(&entries, &changed, IDENTITY_HISTORY).unwrap();
        let mut appended = entries[0].clone();
        appended.id = "appended.active.runtime-site".into();
        appended.site = "runtime-other:wasip1/wasm runtime/new.go execute patch".into();
        appended.disposition = "controlled".into();
        appended.evidence = vec![RUNTIME_EVIDENCE.into()];
        let mut history: serde_json::Value = serde_json::from_str(IDENTITY_HISTORY).unwrap();
        history
            .as_array_mut()
            .unwrap()
            .push(serde_json::json!({"id":appended.id,"site":appended.site}));
        changed
            .get_mut("runtime-other")
            .unwrap()
            .insert("wasip1/wasm runtime/new.go execute patch".into(), 1);
        entries.push(appended);
        check_identity_history(&history.to_string(), &entries).unwrap();
        coverage(&entries, &changed, &history.to_string()).unwrap();
        entries.last_mut().unwrap().disposition = "rejected".into();
        changed
            .get_mut("runtime-other")
            .unwrap()
            .remove("wasip1/wasm runtime/new.go execute patch");
        assert!(
            coverage(&entries, &changed, &history.to_string())
                .unwrap_err()
                .contains("runtime/new.go")
        );
    }
}

#[test]
fn wasip1_seed_finding_and_seeded_draw_decision_premises() {
    let inventory = parse(INVENTORY).unwrap();
    let row = inventory
        .entries
        .iter()
        .find(|row| row.site == "runtime-decision:gomadDiagnosticCheckSeededDraw")
        .unwrap();
    assert_eq!(row.disposition, "controlled");
    let glue = include_str!("../../../../internal/backends/wasm/toolchain/runtime_wasm.go.txt");
    assert!(glue.contains("//go:nosplit\nfunc gomadDiagnosticCheckSeededDraw(*m) {}\n"));
    for forbidden in ["gomadHostTimed", "gomadHostDraw"] {
        assert!(!glue.contains(forbidden), "{forbidden}");
    }
    assert_eq!(inventory.reviewed.go_toolchain, "go1.27.1");
    let go = std::env::var_os("GOMAD_STOCK_GO").unwrap_or_else(|| "go".into());
    let output = Command::new(go)
        .args(["env", "GOROOT"])
        .env("GOTOOLCHAIN", "local")
        .env("GOENV", "off")
        .env("GOWORK", "off")
        .output()
        .unwrap();
    assert!(output.status.success());
    let root = PathBuf::from(std::str::from_utf8(&output.stdout).unwrap().trim());
    assert!(
        fs::read_to_string(root.join("VERSION"))
            .unwrap()
            .starts_with("go1.27.1\n")
    );
    let proc = fs::read_to_string(root.join("src/runtime/proc.go")).unwrap();
    assert!(proc.contains("const haveSysmon = GOARCH != \"wasm\""));
    let os = fs::read_to_string(root.join("src/runtime/os_wasm.go")).unwrap();
    assert!(os.contains("func newosproc(mp *m) {\n\tthrow(\"newosproc: not implemented\")\n}"));
    assert!(os.contains("func cputicks() int64 {\n\t// runtime·nanotime() is a poor approximation of CPU ticks that is enough for the profiler.\n\treturn nanotime()\n}"));
    let pacer = fs::read_to_string(root.join("src/runtime/mgcpacer.go")).unwrap();
    let body = pacer
        .split_once("func (c *gcControllerState) enlistWorker() {")
        .unwrap()
        .1
        .split_once("\n}\n")
        .unwrap()
        .0;
    let guard = body.find("if gomaxprocs <= 1 {\n\t\treturn\n\t}").unwrap();
    let draw = body.find("cheaprandn(").unwrap();
    assert!(guard < draw);
    let overlay = include_str!("../../../../internal/backends/wasm/toolchain/overlay.go");
    assert!(overlay.contains(r"\tif gomadEnabled {\n\t\tprocs = 1\n"));
}
