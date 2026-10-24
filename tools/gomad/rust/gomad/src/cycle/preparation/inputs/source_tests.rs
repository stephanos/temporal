use super::*;
use serde_json::json;
use std::{
    os::unix::fs::{PermissionsExt, symlink},
    sync::atomic::{AtomicU64, Ordering},
};
static NEXT: AtomicU64 = AtomicU64::new(0);
struct Fixture {
    base: PathBuf,
    root: PathBuf,
    repo: PathBuf,
    manifest: PathBuf,
    pins: BTreeMap<String, String>,
}
impl Fixture {
    fn new() -> Self {
        let base = std::env::temp_dir().join(format!(
            "gomad-source-proof-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        let root = base.join("producer");
        let repo = base.join("repo");
        fs::create_dir_all(&root).unwrap();
        fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
        let mut v = Self {
            manifest: base.join("sources.manifest"),
            base,
            root,
            repo,
            pins: BTreeMap::new(),
        };
        for (name, body) in [
            ("go.mod", "module example.test/fixture\n"),
            ("go.sum", "sum\n"),
            (
                "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go",
                "package original\n",
            ),
            (
                "tools/gomad/internal/backends/wasm/testdata/network_streams/temporal_retry_test.go",
                "package retry\n",
            ),
        ] {
            let path = v.repo.join(name);
            fs::create_dir_all(path.parent().unwrap()).unwrap();
            fs::write(&path, body).unwrap();
            v.pins
                .insert(path.to_str().unwrap().into(), digest(body.as_bytes()));
        }
        v.reseal();
        v
    }
    fn reseal(&mut self) {
        self.pins.remove(self.manifest.to_str().unwrap());
        let text = self
            .pins
            .iter()
            .map(|(name, pin)| format!("{pin}  {name}\n"))
            .collect::<String>();
        self.manifest_bytes(text.as_bytes());
    }
    fn manifest_bytes(&mut self, bytes: &[u8]) {
        fs::write(&self.manifest, bytes).unwrap();
        self.pins
            .insert(self.manifest.to_str().unwrap().into(), digest(bytes));
    }
    fn recipe(&self) -> Result<(SourceRecord, Copied), String> {
        recipe(
            &self.root,
            &self.repo,
            &self.pins,
            "recovery",
            &self.manifest,
        )
    }
    fn stage(&self) -> SourceRecord {
        stage(
            &self.root,
            &self.repo,
            &self.pins,
            "recovery",
            &self.manifest,
        )
        .unwrap()
    }
    fn verify(&self, r: &SourceRecord) -> Result<BTreeMap<String, String>, String> {
        verify(
            &self.root,
            &self.repo,
            &self.pins,
            "recovery",
            &r.source,
            &r.working_dir,
            Some(&self.manifest),
        )
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.base).unwrap();
    }
}
fn stat(info: &fs::Metadata) -> serde_json::Value {
    json!({"dev":info.dev(),"ino":info.ino(),"mode":info.mode(),"uid":info.uid(),"gid":info.gid(),"nlink":info.nlink(),"bytes":info.len(),"mtime":[info.mtime(),info.mtime_nsec()],"ctime":[info.ctime(),info.ctime_nsec()]})
}
fn tree(path: &Path) -> serde_json::Value {
    let meta = fs::symlink_metadata(path).unwrap();
    if meta.file_type().is_symlink() {
        return json!({"stat":stat(&meta),"symlink":fs::read_link(path).unwrap()});
    }
    if meta.is_dir() {
        let children = fs::read_dir(path)
            .unwrap()
            .map(|v| v.unwrap().path())
            .map(|p| {
                (
                    p.file_name().unwrap().to_str().unwrap().to_owned(),
                    tree(&p),
                )
            })
            .collect::<BTreeMap<_, _>>();
        return json!({"stat":stat(&meta),"children":children});
    }
    json!({"stat":stat(&meta),"bytes":fs::read(path).unwrap()})
}
impl Fixture {
    fn facts(&self) -> serde_json::Value {
        json!({"root":self.root,"repo":self.repo,"manifest":fs::read(&self.manifest).unwrap(),"pins":self.pins,"originals":tree(&self.repo),"derived":tree(&self.root)})
    }
}
fn proof(lines: &[usize], input: &str, error: &str, facts: &serde_json::Value) {
    for line in lines {
        eprintln!(
            "SOURCE_PROOF {}",
            json!({"id":format!("workload-callback:source_custody:{}:{line}:{}",if *line==25{"raise"}else{"require"},if matches!(*line,25|40|41|49|72|75|78|87|89|91|112|168|184){8}else if matches!(*line,146|148|149|150|154|156){if *line==146{16}else{20}}else{4}),"input_mutation":input,"observed_error":error,"control":"returned","inputs":facts,"input_sha256":digest(&canonical::encode(facts).unwrap()),"test_source_sha256":digest(include_bytes!("source_tests.rs")),"source_sha256":digest(include_bytes!("source.rs")),"shared_source_sha256":digest(include_bytes!("mod.rs"))})
        );
    }
}
#[test]
fn manifest_recipe_direct_catching_controls() {
    for case in [
        "unsealed",
        "malformed",
        "foreign",
        "duplicate",
        "empty",
        "original-pin",
        "missing-module",
        "replace",
        "unknown-workload",
        "missing-wrapper",
        "missing-key",
        "overlap",
    ] {
        let mut f = Fixture::new();
        f.recipe().unwrap();
        let control = f.facts();
        let mut workload = "recovery";
        let (lines, message) = match case {
            "unsealed" => {
                f.pins.remove(f.manifest.to_str().unwrap());
                (vec![25, 68], "unsealed source manifest")
            }
            "malformed" => {
                f.manifest_bytes(b"not a source manifest\n");
                (vec![72], "invalid frozen source manifest")
            }
            "foreign" => {
                f.manifest_bytes(format!("{}  /foreign/source.go\n", digest(b"source")).as_bytes());
                (vec![75], "foreign or unclean frozen source")
            }
            "duplicate" => {
                let mut bytes = fs::read(&f.manifest).unwrap();
                bytes.extend(bytes.clone());
                f.manifest_bytes(&bytes);
                (vec![78], "duplicate frozen source")
            }
            "empty" => {
                f.manifest_bytes(b"");
                (vec![80], "empty frozen source manifest")
            }
            "original-pin" => {
                fs::write(f.repo.join("go.sum"), b"changed").unwrap();
                (vec![91], "original source changed or unsealed")
            }
            "missing-module" => {
                f.pins.remove(f.repo.join("go.sum").to_str().unwrap());
                f.reseal();
                (vec![93], "original module files missing")
            }
            "replace" => {
                let body = b"module fixture\n replace other => /ambient\n";
                let p = f.repo.join("go.mod");
                fs::write(&p, body).unwrap();
                f.pins.insert(p.to_str().unwrap().into(), digest(body));
                f.reseal();
                (vec![94], "ambient module replacement refused")
            }
            "unknown-workload" => {
                workload = "foreign";
                (vec![100], "unknown or empty wrapper workload")
            }
            "missing-wrapper" => {
                f.pins.remove(
                    f.repo
                        .join(BASE)
                        .join("temporal_storage/recovery_test.go")
                        .to_str()
                        .unwrap(),
                );
                f.reseal();
                (vec![104], "wrapper missing from frozen module")
            }
            "missing-key" => {
                workload = "retry";
                let key = f
                    .repo
                    .join(BASE)
                    .join("network_streams/temporal_retry_test.go");
                let body = fs::read(&key).unwrap();
                let other = key.with_file_name("other_test.go");
                fs::write(&other, &body).unwrap();
                f.pins.remove(key.to_str().unwrap());
                f.pins.insert(other.to_str().unwrap().into(), digest(&body));
                f.reseal();
                (vec![106], "wrapper key missing")
            }
            "overlap" => {
                let nested = f.repo.join("producer");
                fs::create_dir(&nested).unwrap();
                fs::set_permissions(&nested, fs::Permissions::from_mode(0o700)).unwrap();
                let e = recipe(&nested, &f.repo, &f.pins, "recovery", &f.manifest).unwrap_err();
                assert!(e.contains("roots overlap"));
                proof(
                    &[65],
                    case,
                    &e,
                    &json!({"control":control,"mutant":f.facts(),"requested_root":nested}),
                );
                continue;
            }
            _ => unreachable!(),
        };
        let error = recipe(&f.root, &f.repo, &f.pins, workload, &f.manifest).unwrap_err();
        assert!(error.contains(message), "{case}: {error}");
        proof(
            &lines,
            case,
            &error,
            &json!({"control":control,"mutant":f.facts(),"workload":workload}),
        );
    }
}
#[test]
fn derived_tree_and_receipt_direct_catching_controls() {
    for case in [
        "module-mode",
        "nested-mode",
        "symlink",
        "extra-directory",
        "extra-file",
        "changed-bytes",
        "missing-file",
        "foreign-manifest",
        "changed-receipt",
        "invalid-receipt",
        "target",
        "collision",
    ] {
        let f = Fixture::new();
        let r = f.stage();
        f.verify(&r).unwrap();
        let control = f.facts();
        let (lines, message) = match case {
            "module-mode" => {
                fs::set_permissions(&r.working_dir, fs::Permissions::from_mode(0o755)).unwrap();
                (vec![138], "derived directory mode changed")
            }
            "nested-mode" => {
                fs::set_permissions(
                    &r.working_dir.join("tools"),
                    fs::Permissions::from_mode(0o755),
                )
                .unwrap();
                (vec![150], "derived directory mode changed")
            }
            "symlink" => {
                let p = r.working_dir.join("go.sum");
                fs::remove_file(&p).unwrap();
                symlink(f.repo.join("go.sum"), &p).unwrap();
                (vec![146], "derived symlink")
            }
            "extra-directory" => {
                fs::create_dir(r.working_dir.join("foreign")).unwrap();
                (vec![148], "unexpected derived directory")
            }
            "extra-file" => {
                fs::write(r.working_dir.join("foreign"), b"changed").unwrap();
                (vec![154], "unexpected derived source")
            }
            "changed-bytes" => {
                fs::write(r.working_dir.join("go.sum"), b"changed").unwrap();
                (vec![156], "derived source changed")
            }
            "missing-file" => {
                fs::remove_file(r.working_dir.join("go.sum")).unwrap();
                (vec![158], "derived inventory changed")
            }
            "foreign-manifest" => {
                let other = f.base.join("other.manifest");
                fs::write(&other, fs::read(&f.manifest).unwrap()).unwrap();
                let e = verify(
                    &f.root,
                    &f.repo,
                    &f.pins,
                    "recovery",
                    &r.source,
                    &r.working_dir,
                    Some(&other),
                )
                .unwrap_err();
                assert!(e.contains("foreign source manifest"));
                proof(
                    &[168],
                    case,
                    &e,
                    &json!({"control":control,"mutant":f.facts(),"requested_manifest":other}),
                );
                continue;
            }
            "changed-receipt" => {
                let p = f.root.join("source-custody.json");
                let mut v = serde_json::to_value(&r).unwrap();
                v["workload"] = json!("retry");
                let mut body = canonical::encode(&v).unwrap();
                body.push(b'\n');
                fs::write(p, body).unwrap();
                (vec![170], "custody recipe changed")
            }
            "invalid-receipt" => {
                fs::write(f.root.join("source-custody.json"), b"[]").unwrap();
                (vec![166], "invalid custody receipt")
            }
            "target" => {
                let e = verify(
                    &f.root,
                    &f.repo,
                    &f.pins,
                    "recovery",
                    "./foreign",
                    &r.working_dir,
                    Some(&f.manifest),
                )
                .unwrap_err();
                assert!(e.contains("target custody differs"));
                proof(
                    &[171],
                    case,
                    &e,
                    &json!({"control":control,"mutant":f.facts(),"requested_source":"./foreign","requested_working_dir":r.working_dir}),
                );
                continue;
            }
            "collision" => {
                let p = f.root.join("source-custody.json");
                fs::remove_file(p).unwrap();
                let e = stage(&f.root, &f.repo, &f.pins, "recovery", &f.manifest).unwrap_err();
                assert!(e.contains("source module collision"));
                proof(
                    &[184],
                    case,
                    &e,
                    &json!({"control":control,"mutant":f.facts(),"requested_source":r.source,"requested_working_dir":r.working_dir}),
                );
                continue;
            }
            _ => unreachable!(),
        };
        let error = f.verify(&r).unwrap_err();
        assert!(error.contains(message), "{case}: {error}");
        proof(
            &lines,
            case,
            &error,
            &json!({"control":control,"mutant":f.facts(),"requested_source":r.source,"requested_working_dir":r.working_dir}),
        );
    }
}

#[test]
fn descriptor_metadata_and_byte_bounds_direct_catching_controls() {
    use super::super::{byte_bound, canonical_path, regular_metadata, same_file};
    let f = Fixture::new();
    let p = f.repo.join("go.sum");
    canonical_path(&p).unwrap();
    let alias = f.base.join("alias");
    symlink(&p, &alias).unwrap();
    let error = canonical_path(&alias).unwrap_err();
    assert!(error.contains("noncanonical"));
    proof(
        &[30],
        "readable symlink alias",
        &error,
        &json!({"control":{"path":p,"bytes":fs::read(&p).unwrap()},"mutant":{"path":alias,"symlink":fs::read_link(&alias).unwrap()}}),
    );
    let before = fs::metadata(&p).unwrap();
    regular_metadata(&before, MAX_BYTES, None).unwrap();
    let link = f.base.join("linked");
    fs::hard_link(&p, &link).unwrap();
    let error = regular_metadata(&fs::metadata(&p).unwrap(), MAX_BYTES, None).unwrap_err();
    assert!(error.contains("nonregular or linked"));
    proof(
        &[37],
        "actual hardlink nlink2",
        &error,
        &json!({"control":stat(&before),"mutant":stat(&fs::metadata(&p).unwrap())}),
    );
    fs::remove_file(&link).unwrap();
    let body = fs::read(&p).unwrap();
    byte_bound(body.len() as u64, body.len() as u64).unwrap();
    let error = byte_bound(body.len() as u64, body.len() as u64 - 1).unwrap_err();
    assert!(error.contains("input byte bound"));
    proof(
        &[38, 52],
        "actual4byte body maximum3; before-length and after-read same validator",
        &error,
        &json!({"control":{"bytes":body,"maximum":body.len()},"mutant":{"bytes":body,"maximum":body.len()-1}}),
    );
    let mode_file = f.base.join("mode");
    fs::write(&mode_file, b"same source").unwrap();
    fs::set_permissions(&mode_file, fs::Permissions::from_mode(0o600)).unwrap();
    regular_metadata(&fs::metadata(&mode_file).unwrap(), MAX_BYTES, Some(0o600)).unwrap();
    let mode_control = fs::metadata(&mode_file).unwrap();
    fs::set_permissions(&mode_file, fs::Permissions::from_mode(0o644)).unwrap();
    let error =
        regular_metadata(&fs::metadata(&mode_file).unwrap(), MAX_BYTES, Some(0o600)).unwrap_err();
    assert!(error.contains("derived file mode"));
    proof(
        &[41],
        "actual file mode0644 instead of0600",
        &error,
        &json!({"control":{"stat":stat(&mode_control),"required_mode":384},"mutant":{"stat":stat(&fs::metadata(&mode_file).unwrap()),"required_mode":384}}),
    );
    let system = fs::metadata("/etc/passwd").unwrap();
    assert_ne!(system.uid(), rustix::process::geteuid().as_raw());
    let error = regular_metadata(&system, MAX_BYTES, Some(system.mode() & 0o7777)).unwrap_err();
    assert!(error.contains("derived file owner"));
    proof(
        &[40],
        "actual root-owned passwd metadata; no bytes read",
        &error,
        &json!({"control":stat(&mode_control),"mutant":stat(&system),"actual_euid":rustix::process::geteuid().as_raw()}),
    );
    let opened = File::open(&p).unwrap();
    let first = opened.metadata().unwrap();
    assert!(same_file(&first, &opened.metadata().unwrap()));
    fs::remove_file(&p).unwrap();
    fs::write(&p, b"sum\n").unwrap();
    let replacement = File::open(&p).unwrap().metadata().unwrap();
    let matched = same_file(&first, &replacement);
    assert!(!matched);
    for (line, indent) in [(49, 8), (53, 4)] {
        eprintln!(
            "SOURCE_PROOF {}",
            json!({"id":format!("workload-callback:source_custody:require:{line}:{indent}"),"input_mutation":"actual original-openfd versus replaced-path-openfd identicalbytes differentinode","current_validator":"same_file","observed_return":matched,"control_return":true,"inputs":{"control":{"before":stat(&first),"after":stat(&first)},"mutant":{"before":stat(&first),"after":stat(&replacement)},"actual_bytes":fs::read(&p).unwrap()},"test_source_sha256":digest(include_bytes!("source_tests.rs")),"source_sha256":digest(include_bytes!("source.rs")),"shared_source_sha256":digest(include_bytes!("mod.rs"))})
        );
    }
    private(&f.root).unwrap();
    let directory_control = fs::metadata(&f.root).unwrap();
    fs::set_permissions(&f.root, fs::Permissions::from_mode(0o755)).unwrap();
    let error = private(&f.root).unwrap_err();
    assert!(error.contains("derived directory mode"));
    proof(
        &[62],
        "actual producer root mode0755",
        &error,
        &json!({"control":stat(&directory_control),"mutant":stat(&fs::metadata(&f.root).unwrap())}),
    );
    fs::set_permissions(&f.root, fs::Permissions::from_mode(0o700)).unwrap();
    private(&f.root).unwrap();
    let foreign = Path::new("/root");
    let metadata = fs::metadata(foreign).unwrap();
    assert_eq!(metadata.mode() & 0o7777, 0o700);
    assert_ne!(metadata.uid(), rustix::process::geteuid().as_raw());
    let error = private(foreign).unwrap_err();
    assert!(error.contains("derived directory owner"));
    proof(
        &[64, 137, 149],
        "actual private root-owned directory metadata; shared root/module/child validator",
        &error,
        &json!({"control":stat(&fs::metadata(&f.root).unwrap()),"mutant":stat(&metadata),"actual_euid":rustix::process::geteuid().as_raw()}),
    );
}

#[test]
fn wrapper_collision_and_copy_limits_direct_catching_controls() {
    assert_eq!(MAX_FILES, 50_000);
    assert_eq!(MAX_BYTES, 536_870_912);
    let f = Fixture::new();
    let (record, copied) = f.recipe().unwrap();
    let names = vec![Path::new(BASE).join("temporal_storage/recovery_test.go")];
    let stage = record.source.strip_prefix("./").unwrap();
    let mut originals = copied.clone();
    originals.retain(|path, _| !path.starts_with(".tmp"));
    assert_eq!(
        add_wrappers(&mut originals, names.clone(), stage).unwrap(),
        record.original_wrappers
    );
    let collision_inputs =
        json!({"control":originals,"mutant":copied,"wrapper_names":names,"relative_stage":stage});
    let mut repeated = copied;
    let error = add_wrappers(&mut repeated, names, stage).unwrap_err();
    assert!(error.contains("wrapper collision"));
    proof(
        &[112],
        "actual already-derived copied map wrapper path collision; prefilter excludes earlierinstance",
        &error,
        &collision_inputs,
    );
    for (line, count, complete, control) in [
        (87, MAX_FILES, false, MAX_FILES - 1),
        (115, MAX_FILES + 1, true, MAX_FILES),
    ] {
        copied_file_bound(control, complete).unwrap();
        let error = copied_file_bound(count, complete).unwrap_err();
        assert!(error.contains("copied file bound"));
        proof(
            &[line],
            &format!("actual bound operand count={count} complete={complete}"),
            &error,
            &json!({"control":{"count":control,"complete":complete},"mutant":{"count":count,"complete":complete},"MAX_FILES":MAX_FILES}),
        );
    }
    copied_byte_bound(MAX_BYTES).unwrap();
    let error = copied_byte_bound(MAX_BYTES + 1).unwrap_err();
    assert!(error.contains("copied byte bound"));
    proof(
        &[89, 117],
        "exact original512MiB aggregate operand incremented1; same validator before and after wrappers",
        &error,
        &json!({"control":MAX_BYTES,"mutant":MAX_BYTES+1,"MAX_BYTES":MAX_BYTES}),
    );
}
