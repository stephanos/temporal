use super::*;

#[test]
fn receiver_bound_is_not_the_engine_inspection_ceiling() {
    assert_eq!(STORAGE_LIMIT, 64 << 20);
    assert!(bounded(&vec![0; STORAGE_LIMIT + 1]).is_err());
    assert!(bounded(&[]).is_ok());
}

#[test]
fn request_identity_refuses_each_changed_execution_input() {
    let expected = engine_input();
    assert!(validate_input(&expected, &expected).is_ok());
    for field in [
        "module",
        "limits",
        "namespace",
        "configuration",
        "arguments",
        "environment",
        "storage",
        "module_sha256",
        "replay_trace",
    ] {
        let mut changed = expected.clone();
        changed[field] = serde_json::json!("receiver changed");
        assert_eq!(
            validate_input(&changed, &expected).unwrap_err(),
            "engine input changed",
            "{field}"
        );
    }
}

#[test]
fn genuine_engine_matrix_and_receiver_owned_semantic_mutations() {
    let expected = engine_input();
    let mut rows = Vec::new();
    for fresh in 0..2 {
        let baseline = run_case(&expected, fresh, Case::Baseline);
        let admitted = admit(&baseline).unwrap();
        let states = admitted.state.enumerations()["data"]["States"]
            .as_array()
            .unwrap()
            .clone();
        let context = admitted.state.contexts().unwrap().clone();
        rows.push(baseline);
        for (index, state) in states.iter().enumerate() {
            let mut input = expected.clone();
            configure_selection(&mut input, &context, &state["SelectedOperations"]);
            let selected = run_case(&input, fresh, Case::Selected(index));
            probe_recovered(
                &expected,
                selected.image.as_ref().unwrap(),
                state["Entries"]
                    .as_array()
                    .unwrap()
                    .iter()
                    .any(|entry| entry["Path"] == "/db"),
            );
            rows.push(selected);
        }
        for variant in 0..3 {
            let mut forged = context.clone();
            match variant {
                0 => {
                    forged["data"]["PendingSHA256"] =
                        serde_json::json!(format!("sha256:{}", "0".repeat(64)))
                }
                1 => forged["data"]["Operations"][0]["Kind"] = serde_json::json!("forged"),
                _ => forged["data"]["Operations"] = serde_json::json!([]),
            }
            let mut input = expected.clone();
            let empty_selection = serde_json::json!([]);
            configure_selection(
                &mut input,
                &forged,
                if variant == 0 {
                    &states.last().unwrap()["SelectedOperations"]
                } else {
                    &empty_selection
                },
            );
            rows.push(run_case(&input, fresh, Case::Divergence(variant)));
        }
    }
    validate_matrix(&expected, &rows).unwrap();
    retain_rows("matrix-control", &rows);
    inner_state_framing(&rows[0]);
    for mutation in [
        "missing",
        "missing-baseline",
        "missing-selected",
        "unknown-index",
        "peak-upper",
        "outcome",
        "selected-outcome",
        "prior-log",
        "determinism",
        "duplicate",
        "fresh",
        "peak",
        "input",
        "foreign-baseline",
        "reorder",
        "effect",
        "selected-image",
        "divergence-stage",
        "raw",
    ] {
        let mut changed = rows.clone();
        match mutation {
            "missing" => {
                changed.pop();
            }
            "missing-baseline" => {
                changed.remove(0);
            }
            "missing-selected" => {
                changed.remove(1);
            }
            "unknown-index" => {
                let mut row = changed[1].clone();
                row.case = Case::Selected(999);
                changed.push(row);
            }
            "peak-upper" => changed[0].peak_memory = 65537,
            "outcome" => changed[0].termination = "trap".into(),
            "selected-outcome" => changed[1].exit_code = Some(1),
            "prior-log" => {
                use sha2::Digest;
                let admitted = admit(&changed[1]).unwrap();
                let version = admitted.state.version();
                let mut sections: Vec<Vec<u8>> = admitted
                    .state
                    .sections()
                    .iter()
                    .map(|x| x.to_vec())
                    .collect();
                let log = b"receiver-owned log".to_vec();
                let mut binding = (log.len() as u64).to_le_bytes().to_vec();
                binding.extend_from_slice(&sha2::Sha256::digest(&log));
                sections[4] = binding;
                changed[1].state = envelope(version, &sections);
                changed[1].log = log;
            }
            "determinism" => {
                for row in changed.iter_mut().filter(|row| row.fresh == 1) {
                    let admitted = admit(row).unwrap();
                    let version = admitted.state.version();
                    let mut sections: Vec<Vec<u8>> = admitted
                        .state
                        .sections()
                        .iter()
                        .map(|x| x.to_vec())
                        .collect();
                    sections[0].push(b' ');
                    row.state = envelope(version, &sections);
                }
            }
            "duplicate" => changed.push(changed[0].clone()),
            "fresh" => changed[0].fresh = 2,
            "peak" => changed[0].peak_memory = 0,
            "input" => changed[0].input["configuration"]["seed"] = serde_json::json!(8),
            "foreign-baseline" => changed[1].input["storage"]["context"] = serde_json::json!([]),
            "reorder" => {
                let state = admit(&changed[0]).unwrap();
                let mut sections: Vec<Vec<u8>> =
                    state.state.sections().iter().map(|x| x.to_vec()).collect();
                let mut page = state.state.enumerations().clone();
                page["data"]["States"].as_array_mut().unwrap().reverse();
                sections[2] = serde_json::to_vec(&page).unwrap();
                changed[0].state = envelope(state.state.version(), &sections);
            }
            "effect" => {
                let state = admit(&changed[0]).unwrap();
                let mut sections: Vec<Vec<u8>> =
                    state.state.sections().iter().map(|x| x.to_vec()).collect();
                let mut page = state.state.enumerations().clone();
                page["data"]["States"][0]["Entries"][0]["Mode"] = serde_json::json!(123);
                let mut hash = sha2::Sha256::new();
                use sha2::Digest;
                for bytes in [b"gomad-volume-crash-state/v1".as_slice(), b"data"] {
                    hash.update((bytes.len() as u64).to_le_bytes());
                    hash.update(bytes);
                }
                let entry = &page["data"]["States"][0]["Entries"][0];
                let path = entry["Path"].as_str().unwrap().as_bytes();
                hash.update((path.len() as u64).to_le_bytes());
                hash.update(path);
                for field in ["Mode", "Kind"] {
                    hash.update(entry[field].as_u64().unwrap().to_le_bytes());
                }
                hash.update(
                    entry["ModTime"]
                        .as_i64()
                        .unwrap()
                        .cast_unsigned()
                        .to_le_bytes(),
                );
                hash.update(0u64.to_le_bytes());
                page["data"]["States"][0]["Identity"] =
                    serde_json::json!(format!("sha256:{:x}", hash.finalize()));
                sections[2] = serde_json::to_vec(&page).unwrap();
                changed[0].state = envelope(state.state.version(), &sections);
            }
            "selected-image" => {
                use sha2::Digest;
                let index = changed
                    .iter()
                    .position(|r| r.fresh == 0 && matches!(r.case, Case::Divergence(_)))
                    .unwrap()
                    - 1;
                let selected = admit(&changed[index]).unwrap();
                let mut sections: Vec<Vec<u8>> = selected
                    .state
                    .sections()
                    .iter()
                    .map(|x| x.to_vec())
                    .collect();
                sections[3] = format!(
                    "{:x}",
                    sha2::Sha256::digest(changed[0].image.as_ref().unwrap())
                )
                .into_bytes();
                let version = selected.state.version();
                changed[index].state = envelope(version, &sections);
                changed[index].image = changed[0].image.clone();
            }
            "divergence-stage" => {
                let index = changed
                    .iter()
                    .position(|r| r.fresh == 0 && matches!(r.case, Case::Divergence(_)))
                    .unwrap();
                let state = admit(&changed[index]).unwrap();
                let mut sections: Vec<Vec<u8>> =
                    state.state.sections().iter().map(|x| x.to_vec()).collect();
                sections[5] = vec![1];
                changed[index].state = envelope(state.state.version(), &sections);
            }
            _ => changed[0].state.push(0),
        }
        let error = validate_matrix(&expected, &changed).unwrap_err();
        retain_rows(mutation, &changed);
        if matches!(mutation, "effect" | "reorder") {
            assert_eq!(error, "enumeration persistence effects/order/coverage");
        }
        if mutation == "selected-image" {
            assert_eq!(error, "selected image durable effects");
        }
        println!("core-catcher {mutation}: {error}");
    }
    let admitted = admit(&rows[0]).unwrap();
    for (name, image, state, log) in [
        (
            "image-trailing",
            [rows[0].image.as_deref().unwrap(), &[0]].concat(),
            rows[0].state.clone(),
            rows[0].log.clone(),
        ),
        (
            "image-malformed",
            vec![0; 56],
            rows[0].state.clone(),
            rows[0].log.clone(),
        ),
        (
            "state-malformed",
            rows[0].image.clone().unwrap(),
            vec![0; 56],
            rows[0].log.clone(),
        ),
        (
            "log-binding",
            rows[0].image.clone().unwrap(),
            rows[0].state.clone(),
            vec![1],
        ),
    ] {
        let mut row = rows[0].clone();
        row.image = Some(image);
        row.state = state;
        row.log = log;
        if name.starts_with("image-") {
            use sha2::Digest;
            let mut sections: Vec<Vec<u8>> = admitted
                .state
                .sections()
                .iter()
                .map(|x| x.to_vec())
                .collect();
            sections[3] =
                format!("{:x}", sha2::Sha256::digest(row.image.as_ref().unwrap())).into_bytes();
            row.state = envelope(admitted.state.version(), &sections);
        }
        assert!(admit(&row).is_err(), "{name}");
    }
    assert!(admitted.image.is_some());
}

fn run_case(input: &serde_json::Value, fresh: u8, case: Case) -> EngineCase {
    let request = request(input).unwrap();
    let mut prepared = gomad_wasmhost::linked::prepare(request).unwrap();
    prepared.conditions(gomad_wasmhost::linked::Conditions::compiled());
    let result = gomad_wasmhost::linked::execute_prepared(prepared).unwrap();
    assert!(
        result.evidence.contains_key("storage_state"),
        "missing retained storage state: {} {} keys={:?}",
        result.outcome.termination,
        result.outcome.message,
        result.evidence.keys().collect::<Vec<_>>()
    );
    EngineCase {
        fresh,
        case,
        input: input.clone(),
        termination: result.outcome.termination.into(),
        message: result.outcome.message,
        exit_code: result.outcome.exit_code,
        peak_memory: result.outcome.peak_memory_bytes as u64,
        state: result.evidence["storage_state"].clone(),
        log: result.evidence.get("storage").cloned().unwrap_or_default(),
        image: result.evidence.get("storage_image").cloned(),
    }
}

fn probe_recovered(expected: &Value, image: &[u8], present: bool) {
    use sha2::{Digest, Sha256};
    let module=format!(r#"(module
        (import "wasi_snapshot_preview1" "path_filestat_get" (func $stat(param i32 i32 i32 i32 i32)(result i32)))
        (import "gomad_wasm_v1" "config" (func $config(param i32 i32)(result i32)))
        (import "gomad_wasm_v1" "finish" (func $finish(param i32 i32)(result i32)))
        (memory(export "memory")1)(data(i32.const 0)"db")
        (func(export "_start") i32.const 512 i32.const 16 call $config drop
        i32.const 4 i32.const 0 i32.const 0 i32.const 2 i32.const 2048 call $stat
        i32.const {} i32.ne if unreachable end
        i32.const 995 i32.const 1 i32.store8 i32.const 992 i32.const 16 call $finish drop))"#,if present{0}else{44}).into_bytes();
    let mut input = expected.clone();
    input["module_sha256"] = serde_json::json!(format!("{:x}", Sha256::digest(&module)));
    input["module"] = serde_json::json!(module);
    input["storage"]["descriptor"]["crash_limits"] = Value::Null;
    input["storage"]["descriptor"]["image"] = serde_json::json!({"bytes":image.len(),"chunks":image.len().div_ceil(65536),"sha256":format!("{:x}",Sha256::digest(image))});
    input["storage"]["image"] = serde_json::json!(image);
    let result = run_case(&input, 0, Case::Baseline);
    assert_eq!(
        result.termination, "exit",
        "recovery namespace: {}",
        result.message
    );
    assert_eq!(result.exit_code, Some(0));
    let observed = admit(&result).unwrap();
    let storage = serde_json::from_value(input["storage"].clone()).unwrap();
    let original = recovery(&storage, image, &format!("{:x}", Sha256::digest(image))).unwrap();
    effects::recovered(&original, &observed.state).unwrap();
}

fn envelope(version: u32, sections: &[Vec<u8>]) -> Vec<u8> {
    use sha2::{Digest, Sha256};
    let mut raw = b"GMSTORE\x01".to_vec();
    raw.extend_from_slice(&version.to_le_bytes());
    raw.extend_from_slice(&(sections.len() as u32).to_le_bytes());
    for (index, section) in sections.iter().enumerate() {
        raw.extend_from_slice(&(index as u32 + 1).to_le_bytes());
        raw.extend_from_slice(&(section.len() as u64).to_le_bytes());
        raw.extend_from_slice(section);
    }
    let length = raw.len() as u64;
    let hash = Sha256::digest(&raw);
    raw.extend_from_slice(&length.to_le_bytes());
    raw.extend_from_slice(&hash);
    raw
}

fn retain_rows(label: &str, rows: &[EngineCase]) {
    let Some(root) = std::env::var_os("FN242_CORE_PROOF_DIRECTORY") else {
        return;
    };
    let root = std::path::PathBuf::from(root);
    std::fs::create_dir_all(&root).unwrap();
    for (index, row) in rows.iter().enumerate() {
        let raw=serde_json::to_vec(&serde_json::json!({"fresh":row.fresh,"case":format!("{:?}",row.case),"input":row.input,"termination":row.termination,"message":row.message,"exit_code":row.exit_code,"peak_memory":row.peak_memory,"state":row.state,"log":row.log,"image":row.image})).unwrap();
        use sha2::{Digest, Sha256};
        let name = format!("{label}-{index}.json");
        std::fs::write(root.join(&name), &raw).unwrap();
        println!(
            "core-proof-input {name} sha256={:x} bytes={}",
            Sha256::digest(&raw),
            raw.len()
        );
    }
}

#[test]
fn current_context_admission_preserves_selection_and_inventory_guards() {
    use serde_json::json;
    let input = engine_input();
    let baseline = run_case(&input, 0, Case::Baseline);
    let admitted = admit(&baseline).unwrap();
    let original = admitted.state.contexts().unwrap().clone();
    let valid_ids = json!([1, 2]);
    for name in [
        "valid",
        "zero",
        "duplicate",
        "unknown",
        "dependency",
        "volume",
        "missing",
        "hash",
        "null",
        "order",
        "future",
        "unknown-field",
        "context-bound",
        "context-shape",
        "prior-shape",
        "hash-type",
        "operations-shape",
        "operation-shape",
        "text-type",
        "dependencies-shape",
        "dependency-type",
        "unsigned-negative",
        "unsigned-bool",
        "unsigned-float",
        "unsigned-overflow",
        "selection-shape",
        "selected-shape",
        "selected-type",
        "context-volume",
        "duplicate-operation",
        "pending-bound",
    ] {
        let mut context = original.clone();
        let mut ids = valid_ids.clone();
        match name {
            "zero" => ids = json!([0]),
            "duplicate" => ids = json!([1, 1]),
            "unknown" => ids = json!([99]),
            "dependency" => ids = json!([2]),
            "hash" => context["data"]["PendingSHA256"] = json!("invalid"),
            "null" => context["data"]["Operations"] = Value::Null,
            "order" => context["data"]["Operations"]
                .as_array_mut()
                .unwrap()
                .reverse(),
            "future" => context["data"]["Operations"][0]["Dependencies"] = json!([2]),
            "unknown-field" => context["data"]["Operations"][0]["unexpected"] = json!(true),
            "context-shape" => context = json!([]),
            "prior-shape" => context["data"] = json!([]),
            "hash-type" => context["data"]["PendingSHA256"] = json!(1),
            "operations-shape" => context["data"]["Operations"] = json!({}),
            "operation-shape" => context["data"]["Operations"] = json!([null]),
            "text-type" => context["data"]["Operations"][0]["Kind"] = json!(1),
            "dependencies-shape" => context["data"]["Operations"][0]["Dependencies"] = json!({}),
            "dependency-type" => context["data"]["Operations"][0]["Dependencies"] = json!([true]),
            "unsigned-negative" => context["data"]["Operations"][0]["ID"] = json!(-1),
            "unsigned-bool" => context["data"]["Operations"][0]["ID"] = json!(true),
            "unsigned-float" => context["data"]["Operations"][0]["ID"] = json!(1.5),
            "unsigned-overflow" => {
                context["data"]["Operations"][0]["ID"] =
                    serde_json::from_str("18446744073709551616").unwrap()
            }
            "selected-shape" => ids = json!(false),
            "selected-type" => ids = json!([true]),
            "context-volume" => context = json!({"foreign":context["data"]}),
            "duplicate-operation" => context["data"]["Operations"][1]["ID"] = json!(1),
            "pending-bound" => {
                context["data"]["Operations"] =
                    json!(vec![context["data"]["Operations"][0].clone(); 1001])
            }
            _ => {}
        }
        if matches!(
            name,
            "hash"
                | "null"
                | "order"
                | "future"
                | "context-volume"
                | "duplicate-operation"
                | "pending-bound"
        ) {
            ids = json!([]);
        }
        let mut changed = input.clone();
        configure_selection(&mut changed, &context, &ids);
        if name == "selection-shape" {
            changed["storage"]["descriptor"]["crash_selection"] = json!([]);
        }
        if name == "volume" {
            changed["storage"]["descriptor"]["crash_selection"] = json!({"foreign":[1]});
        }
        if name == "missing" {
            changed["storage"]["context"] = Value::Null;
            changed["storage"]["descriptor"]["crash_context"] = Value::Null;
        }
        if name == "context-bound" {
            changed["storage"]["descriptor"]["crash_context"]["bytes"] = json!(STORAGE_LIMIT + 1);
        }
        let actual = request(&changed)
            .and_then(|request| gomad_wasmhost::linked::prepare(request).map_err(|e| e.to_string()))
            .and_then(|mut prepared| {
                prepared.conditions(gomad_wasmhost::linked::Conditions::compiled());
                gomad_wasmhost::linked::execute_prepared(prepared).map_err(|e| e.to_string())
            });
        if name == "valid" {
            let actual = actual.unwrap();
            assert_eq!(actual.outcome.termination, "exit");
            assert_eq!(actual.outcome.exit_code, Some(0));
        } else {
            let error = match actual {
                Err(e) => e,
                Ok(execution) => {
                    assert_eq!(execution.outcome.termination, "invalid", "{name}");
                    execution.outcome.message
                }
            };
            assert!(!error.is_empty(), "{name}");
            let guard = match name {
                "zero" => "zero operation",
                "duplicate" => "duplicate operation",
                "unknown" => "unknown operation",
                "dependency" => "dependency-closed",
                "volume" => "selection volume",
                "missing" => "requires prior observation",
                "hash" | "null" | "pending-bound" => "context inventory",
                "order" | "duplicate-operation" => "operation order",
                "future" => "context dependency",
                "context-volume" => "context volume",
                "context-bound" => "context descriptor",
                "unknown-field" => "unknown field",
                "hash-type" | "text-type" => "expected a string",
                "unsigned-negative" | "unsigned-bool" | "unsigned-float" | "unsigned-overflow"
                | "dependency-type" | "selected-type" => "expected u64",
                "context-shape" | "selection-shape" => "expected a map",
                "operations-shape" | "dependencies-shape" | "selected-shape" => {
                    "expected a sequence"
                }
                "prior-shape" => "StorageCrashContext",
                "operation-shape" => "expected struct Operation",
                _ => unreachable!(),
            };
            assert!(
                error.contains(guard),
                "{name}: intended {guard}, got {error}"
            );
            println!("context-catcher {name}: {error}");
        }
    }
}

#[test]
fn genuine_large_context_and_selected_image_replay_all_creation_effects() {
    use serde_json::json;
    use sha2::{Digest, Sha256};
    let mut input = engine_input();
    let mut body = String::new();
    for index in 0..50 {
        body.push_str(&format!("i32.const 3998 i32.const {} i32.store8 i32.const 3999 i32.const {} i32.store8\n i32.const 4 i32.const 0 i32.const 0 i32.const 4000 i32.const 1 i64.const 66 i64.const 0 i32.const 0 i32.const 8000 call $open if unreachable end i32.const 8000 i32.load call $close drop\n",b'0'+index/10,b'0'+index%10));
    }
    let module=format!(r#"(module
        (import "wasi_snapshot_preview1" "path_open" (func $open (param i32 i32 i32 i32 i32 i64 i64 i32 i32)(result i32)))
        (import "wasi_snapshot_preview1" "fd_close" (func $close(param i32)(result i32)))
        (import "gomad_wasm_v1" "config" (func $config(param i32 i32)(result i32)))
        (import "gomad_wasm_v1" "finish" (func $finish(param i32 i32)(result i32)))
        (memory(export "memory") 1)(data(i32.const 0) "{}00")
        (func(export "_start") i32.const 6000 i32.const 16 call $config drop {}
        i32.const 6995 i32.const 1 i32.store8 i32.const 6992 i32.const 16 call $finish drop))"#,r"\01".repeat(3998),body).into_bytes();
    input["module_sha256"] = json!(format!("{:x}", Sha256::digest(&module)));
    input["module"] = json!(module);
    input["storage"]["descriptor"]["crash_limits"] =
        json!({"States":1,"Operations":1,"Depth":1,"Bytes":1<<20,"Work":1});
    input["configuration"]["limits"]["transcript_bytes"] = json!(8 << 20);
    input["limits"]["evidence_bytes"] = json!(64 << 20);
    let baseline = run_case(&input, 0, Case::Baseline);
    assert_eq!(baseline.termination, "exit", "{}", baseline.message);
    let observed = admit(&baseline).unwrap();
    let context = observed.state.contexts().unwrap();
    assert_eq!(context["data"]["Operations"].as_array().unwrap().len(), 100);
    assert!(serde_json::to_vec(context).unwrap().len() > 1 << 20);
    let mut selected = input.clone();
    configure_selection(
        &mut selected,
        context,
        &json!((1..=100).collect::<Vec<u64>>()),
    );
    let selected = run_case(&selected, 0, Case::Selected(0));
    assert_eq!(selected.termination, "exit");
    let recovered = admit(&selected).unwrap();
    effects::large_selection(
        observed.image.as_ref().unwrap(),
        recovered.image.as_ref().unwrap(),
        &recovered.state,
    )
    .unwrap();
    let mut changed = selected.clone();
    let original = baseline.image.as_ref().unwrap();
    changed.image = Some(original.clone());
    let mut sections: Vec<Vec<u8>> = recovered
        .state
        .sections()
        .iter()
        .map(|x| x.to_vec())
        .collect();
    sections[3] = format!("{:x}", Sha256::digest(original)).into_bytes();
    changed.state = envelope(recovered.state.version(), &sections);
    let changed = admit(&changed).unwrap();
    assert_eq!(
        effects::large_selection(
            observed.image.as_ref().unwrap(),
            changed.image.as_ref().unwrap(),
            &changed.state
        )
        .unwrap_err(),
        "large selected durable entries"
    );
}

#[test]
fn current_large_context_admits_exact_five_thousand_operations_and_authenticates_raw_bytes() {
    use serde_json::json;
    use sha2::{Digest, Sha256};
    let mut input = engine_input();
    let model_bytes=br#"{"volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1048576}],"limits":{"PendingOperations":5000,"Transitions":10000}}"#;
    input["storage"]["descriptor"]["model"] = serde_json::from_slice(model_bytes).unwrap();
    input["storage"]["descriptor"]["model_identity"] =
        json!(format!("sha256:{:x}", Sha256::digest(model_bytes)));
    let operations:Vec<_>=(1..=5000).map(|id|json!({"ID":id,"Kind":"write","Dependencies":null,"Inode":0,"Offset":0,"Bytes":0,"PayloadSHA256":"","EffectSHA256":"","Path":"x".repeat(256),"Destination":""})).collect();
    let context = json!({"data":{"PendingSHA256":format!("sha256:{}","a".repeat(64)),"Operations":operations}});
    configure_selection(&mut input, &context, &json!([1]));
    assert!(input["storage"]["context"].as_array().unwrap().len() > 1 << 20);
    assert!(
        input["storage"]["descriptor"]["crash_context"]["chunks"]
            .as_u64()
            .unwrap()
            >= 17
    );
    assert!(
        serde_json::to_vec(&input["storage"]["descriptor"])
            .unwrap()
            .len()
            <= 8192
    );
    gomad_wasmhost::linked::validate(request(&input).unwrap()).unwrap();
    for mutation in [
        "raw",
        "count",
        "digest",
        "chunk-count",
        "shape",
        "oversized",
    ] {
        let mut changed = input.clone();
        match mutation {
            "raw" => changed["storage"]["context"][0] = json!(0),
            "count" => changed["storage"]["descriptor"]["crash_context"]["bytes"] = json!(1),
            "digest" => {
                changed["storage"]["descriptor"]["crash_context"]["sha256"] = json!("a".repeat(64))
            }
            "chunk-count" => changed["storage"]["descriptor"]["crash_context"]["chunks"] = json!(1),
            "shape" => {
                changed["storage"]["descriptor"]["crash_context"]["unexpected"] = json!(true)
            }
            _ => {
                changed["storage"]["descriptor"]["crash_context"]["bytes"] =
                    json!(STORAGE_LIMIT + 1)
            }
        }
        let error = request(&changed)
            .and_then(|r| gomad_wasmhost::linked::validate(r).map_err(|e| e.to_string()))
            .unwrap_err();
        println!("large-context-catcher {mutation}: {error}");
    }
}

#[test]
fn admitted_current_effects_replay_and_refuse_receiver_owned_semantic_corruption() {
    use sha2::{Digest, Sha256};
    let input: linked::StorageInput = serde_json::from_str(testdata::EFFECT_INPUT).unwrap();
    let before = recovery(
        &input,
        testdata::EFFECT_BASELINE,
        &format!("{:x}", Sha256::digest(testdata::EFFECT_BASELINE)),
    )
    .unwrap();
    let after = recovery(
        &input,
        testdata::EFFECT_SELECTED,
        &format!("{:x}", Sha256::digest(testdata::EFFECT_SELECTED)),
    )
    .unwrap();
    effects::pending_effects(&before, &after).unwrap();
    effects::semantic_guard_cases(&before);
    assert_eq!(
        effects::pending_effects(&before, &before).unwrap_err(),
        "effect replay history provenance"
    );
    println!(
        "core-effect-control baseline={:x} selected={:x}",
        Sha256::digest(testdata::EFFECT_BASELINE),
        Sha256::digest(testdata::EFFECT_SELECTED)
    );
}

#[test]
fn partial_durability_over_capacity_is_admitted_without_weakening_current_bounds() {
    use sha2::{Digest, Sha256};
    let input: linked::StorageInput = serde_json::from_str(testdata::OVER_CAPACITY_INPUT).unwrap();
    let expected = format!("{:x}", Sha256::digest(testdata::OVER_CAPACITY));
    let image = recovery(&input, testdata::OVER_CAPACITY, &expected).unwrap();
    let volume = &image.volumes()[0];
    let durable = volume
        .objects()
        .values()
        .map(|o| o.data().len() as u64)
        .sum::<u64>();
    assert_eq!(volume.capacity_bytes(), 8);
    assert_eq!(durable, 16);
    assert!(
        volume
            .objects()
            .keys()
            .all(|inode| *inode < image.next_inode())
    );
    assert!(
        recovery(&input, testdata::OVER_CAPACITY, &"0".repeat(64))
            .unwrap_err()
            .contains("digest mismatch")
    );
    let mut trailing = testdata::OVER_CAPACITY.to_vec();
    trailing.push(0);
    assert!(
        recovery(
            &input,
            &trailing,
            &format!("{:x}", Sha256::digest(&trailing))
        )
        .unwrap_err()
        .contains("trailing bytes")
    );
    let mut forged = serde_json::to_value(input).unwrap();
    forged["descriptor"]["model_identity"] =
        serde_json::json!(format!("sha256:{}", "0".repeat(64)));
    let forged = serde_json::from_value(forged).unwrap();
    assert!(recovery(&forged, testdata::OVER_CAPACITY, &expected).is_err());
    println!("core-over-capacity-control image={expected} durable={durable} capacity=8");
}

fn inner_state_framing(baseline: &EngineCase) {
    for (label, reason) in [
        ("state-version", "version/sections"),
        ("state-count", "version/sections"),
        (
            "state-truncated-header",
            "truncated retained storage section",
        ),
        ("state-tag-order", "unordered retained storage sections"),
        ("state-section-size", "retained storage section capacity"),
        ("state-trailing-body", "trailing retained storage state"),
    ] {
        let mut changed = baseline.clone();
        changed.state.truncate(changed.state.len() - 40);
        match label {
            "state-version" => changed.state[8..12].copy_from_slice(&99_u32.to_le_bytes()),
            "state-count" => changed.state[12..16].copy_from_slice(&99_u32.to_le_bytes()),
            "state-truncated-header" => changed.state.truncate(24),
            "state-tag-order" => changed.state[16..20].copy_from_slice(&2_u32.to_le_bytes()),
            "state-section-size" => changed.state[20..28].copy_from_slice(&u64::MAX.to_le_bytes()),
            "state-trailing-body" => changed.state.push(0),
            _ => unreachable!(),
        }
        let body = changed.state.len() as u64;
        let hash = Sha256::digest(&changed.state);
        changed.state.extend(body.to_le_bytes());
        changed.state.extend(hash);
        let error = admit(&changed)
            .err()
            .expect("inner state mutation admitted");
        assert!(error.contains(reason), "{label}: {error}");
        retain_rows(label, &[changed]);
        println!("core-proof-rejection {label}: {error}");
    }
}
