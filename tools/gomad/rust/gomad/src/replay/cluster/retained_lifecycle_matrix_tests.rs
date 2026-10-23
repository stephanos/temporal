use super::*;
use crate::evidence::store::{Publication, Store};
use std::path::{Path, PathBuf};

const CASES: [&str; 21] = [
    "missing-action",
    "mutated-action",
    "missing-module",
    "foreign-module",
    "boot-module-differs",
    "changed-limits",
    "changed-environment",
    "changed-helper-budget",
    "changed-choice-budget",
    "malformed-storage-selection",
    "changed-imports",
    "missing-model-implementation",
    "changed-model-implementation",
    "changed-world-profile",
    "changed-world-encoding",
    "trailing-world-byte",
    "invalid-world-header",
    "corrupt-world-digest",
    "changed-runtime-binding",
    "changed-original-record",
    "changed-root-identity",
];

struct Fixture {
    root: PathBuf,
    path: PathBuf,
}
impl Fixture {
    fn new(mut publication: Publication) -> Self {
        for name in [FILE, "evidence/world"] {
            if let Some(raw) = publication.payloads.get(name) {
                let stream = publication.record.streams.get_mut(name).unwrap();
                stream.bytes = raw.len() as u64;
                stream.sha256 = canonical::digest(raw);
            }
        }
        publication
            .payloads
            .insert("record.json".into(), publication.record.bytes().unwrap());
        let root = std::env::temp_dir().join(format!(
            "gomad-retained-lifecycle-matrix-{}",
            crate::campaign::origin::identity().unwrap()
        ));
        let published = Store {
            root: root.clone(),
            maximum_bytes: 128 << 20,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        }
        .publish(publication)
        .unwrap();
        Self {
            root,
            path: published.path,
        }
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.root).unwrap();
    }
}

fn publication() -> Publication {
    let corpus = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../testdata/parity");
    let source = crate::fixture_cluster::publication(&corpus, 7).unwrap();
    let mut inputs: Inputs = serde_json::from_slice(&source.payloads[FILE]).unwrap();
    inputs.configuration.world_encoding = "gomad/world/compressed-snapshot/v2".into();
    inputs.configuration.world_profile = Some(inputs.configuration.world_encoding.clone());
    crate::fixture_cluster::execute_configuration(
        inputs.configuration,
        inputs.toolchain_sha256,
        BTreeMap::from([(
            "target/module.wasm".into(),
            source.payloads["target/module.wasm"].clone(),
        )]),
    )
    .unwrap()
}

fn copy(publication: &Publication) -> Publication {
    Publication {
        record: publication.record.clone(),
        payloads: publication.payloads.clone(),
    }
}

fn files(root: &Path) -> BTreeMap<String, Value> {
    fn walk(root: &Path, directory: &Path, output: &mut BTreeMap<String, Value>) {
        for item in std::fs::read_dir(directory).unwrap() {
            let path = item.unwrap().path();
            let name = path
                .strip_prefix(root)
                .unwrap()
                .to_str()
                .unwrap()
                .to_owned();
            if path.is_dir() {
                output.insert(name, json!({"directory":true}));
                walk(root, &path, output);
            } else {
                let raw = std::fs::read(&path).unwrap();
                output.insert(
                    name,
                    json!({"bytes":raw.len(),"sha256":canonical::digest(&raw)}),
                );
            }
        }
    }
    let mut output = BTreeMap::new();
    walk(root, root, &mut output);
    output
}

fn raw_artifact(fixture: &Fixture) -> Value {
    let manifest_raw = std::fs::read(fixture.path.join("manifest.json")).unwrap();
    let manifest: Value = serde_json::from_slice(&manifest_raw).unwrap();
    let payloads: BTreeMap<String, String> = manifest["files"]
        .as_object()
        .unwrap()
        .keys()
        .map(|name| {
            (
                name.clone(),
                crate::campaign::exploration::base64(
                    &std::fs::read(fixture.path.join(name)).unwrap(),
                ),
            )
        })
        .collect();
    json!({"manifest_base64":crate::campaign::exploration::base64(&manifest_raw),
        "record_base64":payloads["record.json"],"payloads_base64":payloads})
}

fn increment(value: &mut Value) {
    *value = json!(value.as_u64().unwrap() + 1);
}

fn mutate(publication: &mut Publication, case: &str) -> &'static str {
    let mut inputs: Value = serde_json::from_slice(&publication.payloads[FILE]).unwrap();
    let zeros = "0".repeat(64);
    let mapping = match case {
        "missing-action" => {
            inputs["configuration"]["steps"]
                .as_array_mut()
                .unwrap()
                .pop();
            "configuration.steps: remove final terminal action"
        }
        "mutated-action" => {
            increment(&mut inputs["configuration"]["steps"][0]["handle"]["incarnation"]);
            "configuration.steps[0].handle.incarnation"
        }
        "missing-module" => {
            inputs["configuration"]["incarnations"][0]["module"]["sha256"] = json!("");
            "configuration.incarnations[0].module.sha256: absent primary identity"
        }
        "foreign-module" => {
            inputs["configuration"]["incarnations"][0]["module"]["sha256"] = json!(zeros);
            "configuration.incarnations[0].module.sha256: foreign primary identity"
        }
        "boot-module-differs" => {
            inputs["configuration"]["incarnations"][1]["module"]["sha256"] = json!(zeros);
            "configuration.incarnations[1].module.sha256: foreign boot identity"
        }
        "changed-limits" => {
            increment(&mut inputs["configuration"]["broker"]["limits"]["max_transitions"]);
            "configuration.broker.limits.max_transitions: native derived transition bound"
        }
        "changed-environment" => {
            increment(&mut inputs["configuration"]["incarnations"][0]["namespace"]["epoch_nanos"]);
            "configuration.incarnations[0].namespace.epoch_nanos: retained clock environment"
        }
        "changed-helper-budget" => {
            increment(&mut inputs["configuration"]["incarnations"][0]["limits"]["wall_millis"]);
            "configuration.incarnations[0].limits.wall_millis: native helper wall budget"
        }
        "changed-choice-budget" => {
            increment(&mut inputs["configuration"]["incarnations"][0]["model"]["limit"]);
            "configuration.incarnations[0].model.limit"
        }
        "malformed-storage-selection" => {
            inputs["configuration"]["incarnations"][0]["storage_crash_selection"] =
                json!({"unknown":[0,0]});
            "configuration.incarnations[0].storage_crash_selection: unknown volume and duplicate sequence"
        }
        "changed-imports" => {
            inputs["configuration"]["incarnations"][0]["imports"] = json!([]);
            "configuration.incarnations[0].imports: differs from actual module imports"
        }
        "missing-model-implementation" => {
            inputs["runtime_implementation_sha256"] = json!("");
            "runtime_implementation_sha256: missing native implementation binding"
        }
        "changed-model-implementation" => {
            inputs["runtime_implementation_sha256"] = json!(zeros);
            "runtime_implementation_sha256: foreign native implementation binding"
        }
        "changed-world-profile" => {
            inputs["configuration"]["world_profile"] = json!("unknown");
            "configuration.world_profile"
        }
        "changed-world-encoding" => {
            inputs["configuration"]["world_encoding"] = json!("unknown");
            "configuration.world_encoding: native encoding declaration is in bound replay configuration"
        }
        "trailing-world-byte" => {
            publication
                .payloads
                .get_mut("evidence/world")
                .unwrap()
                .push(0);
            "external evidence/world: trailing byte, record stream and manifest resealed"
        }
        "invalid-world-header" => {
            publication.payloads.get_mut("evidence/world").unwrap()[0] ^= 1;
            "external evidence/world: native codec header byte zero, record stream and manifest resealed"
        }
        "corrupt-world-digest" => {
            publication.payloads.get_mut("evidence/world").unwrap()[32] ^= 1;
            "external evidence/world: native codec byte 32, record stream and manifest resealed"
        }
        "changed-runtime-binding" => {
            inputs["configuration"]["incarnations"][0]["choice_identity"]["ToolchainBuildKey"] =
                json!("changed");
            "configuration.incarnations[0].choice_identity.ToolchainBuildKey"
        }
        "changed-original-record" => {
            "record.json: append newline and reseal outer file descriptor, canonical record remains unchanged"
        }
        "changed-root-identity" => {
            "manifest.record.execution_identity and record.json.execution_identity: foreign root identity, outer descriptors resealed"
        }
        _ => unreachable!(),
    };
    publication
        .payloads
        .insert(FILE.into(), canonical::encode(&inputs).unwrap());
    mapping
}

fn mutate_record_envelope(fixture: &Fixture, case: &str) {
    if !matches!(case, "changed-original-record" | "changed-root-identity") {
        return;
    }
    let path = fixture.path.join("manifest.json");
    let mut manifest: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
    let mut record = std::fs::read(fixture.path.join("record.json")).unwrap();
    if case == "changed-original-record" {
        record.push(b'\n');
    } else {
        manifest["record"]["execution_identity"] = json!("0".repeat(64));
        record = canonical::encode(&manifest["record"]).unwrap();
        manifest["record_sha256"] = json!(canonical::digest(&record));
    }
    manifest["files"]["record.json"]["bytes"] = json!(record.len());
    manifest["files"]["record.json"]["sha256"] = json!(canonical::digest(&record));
    std::fs::write(fixture.path.join("record.json"), record).unwrap();
    std::fs::write(path, canonical::encode(&manifest).unwrap()).unwrap();
}

fn boundary(fixture: &Fixture, name: &str) -> Result<Value, String> {
    if name == "public-cli-replay" {
        let mut output = Vec::new();
        let status = crate::cli::replay::run(&[fixture.path.to_str().unwrap().into()], &mut output)
            .map_err(|error| format!("{}: {}", error.status, error.message))?;
        return Ok(
            json!({"status":status,"output_base64":crate::campaign::exploration::base64(&output)}),
        );
    }
    let opened = Opened::open(&fixture.path)?;
    match name {
        "verify" => verify_retained(&opened).map(|identity| json!({"execution_identity":identity})),
        "storage-projection" => {
            let selection = storage_selections(&opened)?;
            let expected = canonical::encode(
                &selection
                    .iter()
                    .map(|(handle, values)| json!({"handle":handle,"selection":values}))
                    .collect::<Vec<_>>(),
            )?;
            let mut detached = selection;
            detached.insert(
                helper::Handle {
                    node: "caller-mutation".into(),
                    incarnation: 1,
                },
                BTreeMap::from([("unknown".into(), vec![0, 0])]),
            );
            assert_eq!(
                canonical::encode(
                    &storage_selections(&opened)?
                        .iter()
                        .map(|(handle, values)| json!({"handle":handle,"selection":values}))
                        .collect::<Vec<_>>()
                )?,
                expected
            );
            Ok(
                json!({"selection_entries":detached.len()-1,"selection_bytes_base64":crate::campaign::exploration::base64(&expected)}),
            )
        }
        "prepare-reopen" => {
            let prepared = prepare(opened, None)?;
            Ok(json!({"exact":prepared.exact,"available":prepared.available}))
        }
        _ => unreachable!(),
    }
}

#[test]
fn original_lifecycle_twenty_one_mutations_refuse_all_four_native_readers() {
    let baseline = publication();
    let original_identity = baseline.record.identity.clone();
    let original_record = baseline.record.bytes().unwrap();
    let before_fixture = Fixture::new(copy(&baseline));
    let before = raw_artifact(&before_fixture);
    for case in std::iter::once("valid").chain(CASES) {
        let mut changed = copy(&baseline);
        let mapping = if case == "valid" {
            "unchanged actual three-incarnation publication"
        } else {
            mutate(&mut changed, case)
        };
        assert_eq!(changed.record.identity, original_identity);
        let fixture = Fixture::new(changed);
        mutate_record_envelope(&fixture, case);
        let after = raw_artifact(&fixture);
        let mut results = BTreeMap::new();
        for name in [
            "verify",
            "storage-projection",
            "prepare-reopen",
            "public-cli-replay",
        ] {
            let entries_before = files(&fixture.root);
            let result = boundary(&fixture, name);
            let entries_after = files(&fixture.root);
            assert_eq!(
                entries_before, entries_after,
                "{case}/{name}: wrote retained files"
            );
            results.insert(
                name,
                json!({"error":result.as_ref().err(),"value":result.as_ref().ok(),
                "entries_before":entries_before,"entries_after":entries_after,
                "entry_count_before":entries_before.len(),"entry_count_after":entries_after.len()}),
            );
            assert_eq!(result.is_ok(), case == "valid", "{case}/{name}: {result:?}");
        }
        println!(
            "GOMAD_RETAINED_LIFECYCLE_MATRIX {}",
            json!({
            "schema":"gomad.retained-lifecycle-matrix/v1","case":case,"native_mapping":mapping,
            "native_identity":original_identity,"original_record_base64":crate::campaign::exploration::base64(&original_record),
            "before":before,"after":after,"boundaries":results})
        );
    }
}

fn number(mut value: u32) -> Vec<u8> {
    let mut output = Vec::new();
    loop {
        let byte = (value & 127) as u8;
        value >>= 7;
        output.push(byte | if value == 0 { 0 } else { 128 });
        if value == 0 {
            return output;
        }
    }
}

fn section(module: &mut Vec<u8>, id: u8, bytes: &[u8]) {
    module.push(id);
    module.extend(number(u32::try_from(bytes.len()).unwrap()));
    module.extend(bytes);
}

fn large_binary_module() -> Vec<u8> {
    let mut module = b"\0asm\x01\0\0\0".to_vec();
    section(
        &mut module,
        1,
        &[2, 0x60, 2, 0x7f, 0x7f, 1, 0x7f, 0x60, 0, 0],
    );
    let mut imports = vec![5];
    for name in ["config", "decision", "observation", "finish", "idle"] {
        imports.extend(number(13));
        imports.extend(b"gomad_wasm_v1");
        imports.extend(number(u32::try_from(name.len()).unwrap()));
        imports.extend(name.as_bytes());
        imports.extend([0, 0]);
    }
    section(&mut module, 2, &imports);
    section(&mut module, 3, &[1, 1]);
    section(&mut module, 5, &[1, 0, 1]);
    section(
        &mut module,
        7,
        &[
            2, 6, b'm', b'e', b'm', b'o', b'r', b'y', 2, 0, 6, b'_', b's', b't', b'a', b'r', b't',
            0, 5,
        ],
    );
    let mut code = vec![0];
    for (address, length, function) in [(0, 16, 0), (64, 96, 1), (16, 16, 3)] {
        code.push(0x41);
        code.extend(number(address));
        // Positive signed LEB values need a zero sign group when bit six is set.
        if address & 64 != 0 {
            *code.last_mut().unwrap() |= 128;
            code.push(0);
        }
        code.push(0x41);
        code.extend(number(length));
        if length & 64 != 0 {
            *code.last_mut().unwrap() |= 128;
            code.push(0);
        }
        code.extend([0x10, function, 0x04, 0x40, 0x00, 0x0b]);
    }
    code.push(0x0b);
    let mut body = vec![1];
    body.extend(number(u32::try_from(code.len()).unwrap()));
    body.extend(code);
    section(&mut module, 10, &body);
    let source = std::fs::read_to_string(
        Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../testdata/parity/targets/inspect-choice.wat"),
    )
    .unwrap();
    let mut data = vec![2];
    for (line, address) in source
        .lines()
        .filter(|line| line.trim_start().starts_with("(data "))
        .zip([16, 64])
    {
        let escaped = line.split('"').nth(1).unwrap();
        let bytes: Vec<u8> = escaped
            .split('\\')
            .skip(1)
            .map(|byte| u8::from_str_radix(byte, 16).unwrap())
            .collect();
        data.extend([0, 0x41]);
        data.extend(number(address));
        if address & 64 != 0 {
            *data.last_mut().unwrap() |= 128;
            data.push(0);
        }
        data.push(0x0b);
        data.extend(number(u32::try_from(bytes.len()).unwrap()));
        data.extend(bytes);
    }
    section(&mut module, 11, &data);
    let mut custom = vec![26];
    custom.extend(b"gomad-retention-padding-v1");
    custom.resize(1 << 20, 0);
    section(&mut module, 0, &custom);
    module_imports(&module).unwrap();
    module
}

#[test]
fn original_retained_nonembedding_bound_uses_actual_large_external_module() {
    let initial = publication();
    let small_fixture = Fixture::new(copy(&initial));
    let small = raw_artifact(&small_fixture);
    let mut inputs: Inputs = serde_json::from_slice(&initial.payloads[FILE]).unwrap();
    let module = large_binary_module();
    assert!(module.len() > 1 << 20);
    for item in &mut inputs.configuration.incarnations {
        item.module.bytes = module.len() as u64;
        item.module.sha256 = canonical::digest(&module);
        item.choice_identity.target_sha256 = sha2::Sha256::digest(&module).into();
        item.imports = Some(module_imports(&module).unwrap());
    }
    let publication = crate::fixture_cluster::execute_configuration(
        inputs.configuration,
        inputs.toolchain_sha256,
        BTreeMap::from([("target/module.wasm".into(), module.clone())]),
    )
    .unwrap();
    let retained = publication.payloads[FILE].clone();
    let record = publication.record.bytes().unwrap();
    let nonmodule_bytes: usize = publication
        .payloads
        .iter()
        .filter(|(name, _)| name.as_str() != "target/module.wasm" && name.as_str() != "record.json")
        .map(|(_, raw)| raw.len())
        .sum();
    let bound = record.len() * 4 / 3 + (64 << 10);
    assert!(nonmodule_bytes <= bound);
    let fixture = Fixture::new(copy(&publication));
    let before = raw_artifact(&fixture);
    let mut boundaries = BTreeMap::new();
    for name in [
        "verify",
        "storage-projection",
        "prepare-reopen",
        "public-cli-replay",
    ] {
        let entries_before = files(&fixture.root);
        let result = boundary(&fixture, name).unwrap();
        let entries_after = files(&fixture.root);
        assert_eq!(entries_before, entries_after);
        boundaries.insert(
            name,
            json!({"error":Value::Null,"value":result,
            "entries_before":entries_before,"entries_after":entries_after,
            "entry_count_before":entries_before.len(),"entry_count_after":entries_after.len()}),
        );
    }
    let mut detached = copy(&publication);
    detached.payloads.remove("target/module.wasm");
    assert_eq!(detached.payloads[FILE], retained);
    let detached_fixture = Fixture::new(detached);
    let opened = Opened::open(&detached_fixture.path).unwrap();
    assert_eq!(
        super::inputs(&opened)
            .and_then(|inputs| canonical::encode(&inputs))
            .unwrap(),
        retained
    );
    println!(
        "GOMAD_RETAINED_LIFECYCLE_MATRIX {}",
        json!({
        "schema":"gomad.retained-lifecycle-matrix/v1","case":"module-nonembedding",
        "native_identity":publication.record.identity,"small":small,"before":before,"after":raw_artifact(&detached_fixture),"boundaries":boundaries,
        "retained_bytes":retained.len(),"nonmodule_payload_bytes":nonmodule_bytes,
        "original_record_bytes":record.len(),"module_bytes":module.len(),"bound_bytes":bound,
        "bound_mapping":"sum of external retained payload bytes excluding module and canonical record <= original canonical record bytes * 4 / 3 + 64 KiB"})
    );
}

fn guest_raw_artifact(fixture: &Fixture) -> Value {
    let manifest_raw = std::fs::read(fixture.path.join("manifest.json")).unwrap();
    let manifest: Value = serde_json::from_slice(&manifest_raw).unwrap();
    let mut payloads = BTreeMap::new();
    let mut modules = BTreeMap::new();
    for (name, descriptor) in manifest["files"].as_object().unwrap() {
        let raw = std::fs::read(fixture.path.join(name)).unwrap();
        if name == "target/module.wasm" {
            assert_eq!(descriptor["sha256"], canonical::digest(&raw));
            modules.insert(name.clone(), json!({"bytes":raw.len(),"sha256":canonical::digest(&raw),"source":"original-artifact"}));
        } else {
            payloads.insert(name.clone(), crate::campaign::exploration::base64(&raw));
        }
    }
    json!({"manifest_base64":crate::campaign::exploration::base64(&manifest_raw),
        "record_base64":payloads["record.json"],"payloads_base64":payloads,"module_preimages":modules})
}

fn guest_boundaries(fixture: &Fixture, admitted: bool) -> Value {
    let mut results = BTreeMap::new();
    for name in [
        "verify",
        "storage-projection",
        "prepare-reopen",
        "public-cli-replay",
    ] {
        let entries_before = files(&fixture.root);
        let result = boundary(fixture, name);
        let entries_after = files(&fixture.root);
        assert_eq!(entries_before, entries_after);
        assert_eq!(result.is_ok(), admitted, "{name}: {result:?}");
        results.insert(
            name,
            json!({"error":result.as_ref().err(),"value":result.as_ref().ok(),
            "entries_before":entries_before,"entries_after":entries_after,
            "entry_count_before":entries_before.len(),"entry_count_after":entries_after.len()}),
        );
    }
    json!(results)
}

fn guest_mutation(publication: &mut Publication, case: &str) {
    if matches!(case, "changed-imports" | "changed-runtime-binding") {
        let mut inputs: Value = serde_json::from_slice(&publication.payloads[FILE]).unwrap();
        let item = &mut inputs["configuration"]["incarnations"][0];
        let field = if case == "changed-imports" {
            &mut item["imports"][0]["name"]
        } else {
            &mut item["choice_identity"]["ToolchainBuildKey"]
        };
        *field = json!(format!("{}-changed", field.as_str().unwrap()));
        publication
            .payloads
            .insert(FILE.into(), canonical::encode(&inputs).unwrap());
    } else if case == "foreign-linked-helper" {
        let mut inputs: Value = serde_json::from_slice(&publication.payloads[FILE]).unwrap();
        inputs["helper_sha256"] = json!("f".repeat(64));
        publication
            .payloads
            .insert(FILE.into(), canonical::encode(&inputs).unwrap());
    } else {
        mutate(publication, case);
    }
}

fn guest_primary_envelope(fixture: &Fixture, empty: bool) {
    let manifest_path = fixture.path.join("manifest.json");
    let mut manifest: Value =
        serde_json::from_slice(&std::fs::read(&manifest_path).unwrap()).unwrap();
    manifest["record"]["identity"]["target"] =
        json!(if empty { String::new() } else { "0".repeat(64) });
    manifest["record"]["execution_identity"] = json!(canonical::digest(
        &canonical::encode(&manifest["record"]["identity"]).unwrap()
    ));
    let record = canonical::encode(&manifest["record"]).unwrap();
    manifest["record_sha256"] = json!(canonical::digest(&record));
    manifest["files"]["record.json"]["bytes"] = json!(record.len());
    manifest["files"]["record.json"]["sha256"] = json!(canonical::digest(&record));
    std::fs::write(fixture.path.join("record.json"), record).unwrap();
    std::fs::write(manifest_path, canonical::encode(&manifest).unwrap()).unwrap();
}

pub(super) fn actual_guest_retention(
    path: &Path,
    original_root: &str,
    case: &str,
    producer_root: &Path,
) {
    let opened = Opened::open(path).unwrap();
    opened.authenticate_all().unwrap();
    let baseline = Publication {
        record: opened.manifest().record.clone(),
        payloads: opened
            .manifest()
            .files
            .keys()
            .map(|name| (name.clone(), opened.payload(name, 128 << 20).unwrap()))
            .collect(),
    };
    let original_record = baseline.record.bytes().unwrap();
    let original_inputs = baseline.payloads[FILE].clone();
    let original_module = &baseline.payloads["target/module.wasm"];
    assert!(!producer_root.join("prepared-module.wasm").exists());
    assert!(!producer_root.join("installation.json").exists());
    let original_snapshot = files(path);
    let fixture = Fixture::new(copy(&baseline));
    let before = guest_raw_artifact(&fixture);
    let emit = |variant: &str, value: Value| {
        println!(
            "GOMAD_GUEST_RETENTION {}",
            json!({"schema":"gomad.guest-retention-matrix/v1",
            "original_root":original_root,"case":case,"variant":variant,"original_artifact":path,
            "record_sha256":canonical::digest(&original_record),"module_sha256":canonical::digest(original_module),
            "observed":value})
        );
    };
    emit(
        "valid",
        json!({"artifact":before,"boundaries":guest_boundaries(&fixture,true)}),
    );
    for variant in
        CASES
            .into_iter()
            .chain(["foreign-linked-helper", "primary-empty", "primary-foreign"])
    {
        let mut changed = copy(&baseline);
        if !variant.starts_with("primary-") {
            guest_mutation(&mut changed, variant);
        }
        let changed = Fixture::new(changed);
        if variant.starts_with("primary-") {
            guest_primary_envelope(&changed, variant == "primary-empty");
        } else {
            mutate_record_envelope(&changed, variant);
        }
        emit(
            variant,
            json!({"artifact":guest_raw_artifact(&changed),"boundaries":guest_boundaries(&changed,false)}),
        );
    }
    let mut unavailable = copy(&baseline);
    unavailable.payloads.remove("target/module.wasm");
    let unavailable = Fixture::new(unavailable);
    let without = Opened::open(&unavailable.path).unwrap();
    let without_inputs = super::inputs(&without).unwrap();
    assert_eq!(canonical::encode(&without_inputs).unwrap(), original_inputs);
    let metadata_bound = original_record.len() * 4 / 3 + (64 << 10);
    assert!(original_inputs.len() <= metadata_bound);
    emit(
        "module-independent",
        json!({"artifact":guest_raw_artifact(&unavailable),
        "read_inputs_base64":crate::campaign::exploration::base64(&canonical::encode(&without_inputs).unwrap()),
        "bound_bytes":metadata_bound,"original_module_absent":!producer_root.join("prepared-module.wasm").exists(),
        "original_installation_absent":!producer_root.join("installation.json").exists()}),
    );
    let prepared = super::prepare(opened.try_clone().unwrap(), None).unwrap();
    let projection = prepared.execution.projection();
    let replayed = super::execute(prepared.execution).unwrap();
    assert_eq!(replayed.incarnations.len(), 4);
    let summaries: Vec<_> = replayed
        .incarnations
        .iter()
        .map(|item| {
            json!({"handle":item.handle,
        "interrupted":item.interrupted,"outcome":item.execution.outcome,
        "evidence":item.execution.evidence})
        })
        .collect();
    let replay_steps = canonical::encode(&replayed.executed_steps).unwrap();
    let projected = projection.project(replayed).unwrap();
    let divergence = crate::replay::divergence::first(&baseline.record, &projected);
    assert_eq!(divergence, None);
    assert_eq!(files(path), original_snapshot);
    emit(
        "source-less-replay",
        json!({"original_record_base64":crate::campaign::exploration::base64(&original_record),
        "summaries":summaries,"executed_steps_base64":crate::campaign::exploration::base64(&replay_steps),
        "outcome":projected.outcome,"projected_evidence":projected.evidence,"divergence":divergence,
        "entries_before":original_snapshot,"entries_after":files(path)}),
    );
}
