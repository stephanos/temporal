use super::*;

fn image(headers: usize) -> Vec<u8> {
    let mut raw = vec![0; 64 + headers * 56];
    raw[..6].copy_from_slice(b"\x7fELF\x02\x01");
    raw[18..20].copy_from_slice(&62u16.to_le_bytes());
    raw[32..40].copy_from_slice(&64u64.to_le_bytes());
    raw[54..56].copy_from_slice(&56u16.to_le_bytes());
    raw[56..58].copy_from_slice(&(headers as u16).to_le_bytes());
    raw
}

#[test]
fn elf_accepts_plain_files_and_static_native_images() {
    assert_eq!(elf(b"plain").unwrap(), None);
    assert_eq!(
        elf(&image(0)).unwrap(),
        Some(Elf {
            machine: 62,
            interpreter: vec![],
            needed: vec![],
            search: vec![]
        })
    );
}

#[test]
fn elf_rejects_malformed_bounds_and_foreign_architecture() {
    let raw = image(1);
    let mut cases = vec![raw[..63].to_vec()];
    for (offset, bytes) in [
        (4, vec![1]),
        (18, 3u16.to_le_bytes().to_vec()),
        (32, u64::MAX.to_le_bytes().to_vec()),
        (54, 55u16.to_le_bytes().to_vec()),
        (56, 1025u16.to_le_bytes().to_vec()),
    ] {
        let mut value = raw.clone();
        value[offset..offset + bytes.len()].copy_from_slice(&bytes);
        cases.push(value);
    }
    for value in cases {
        assert!(elf(&value).is_err());
    }
}

#[test]
fn elf_rejects_invalid_dynamic_and_interpreter_segments() {
    for kind in [2u32, 3] {
        let mut raw = image(1);
        raw[64..68].copy_from_slice(&kind.to_le_bytes());
        raw[72..80].copy_from_slice(&u64::MAX.to_le_bytes());
        raw[104..112].copy_from_slice(&1u64.to_le_bytes());
        assert!(elf(&raw).is_err());
    }
}

#[test]
fn probes_preserve_exact_arguments_and_environment() {
    let probes = probes("/gcc", "/ldconfig");
    assert_eq!(probes.len(), 21);
    assert_eq!(
        probes["include-cxx"],
        ["/gcc", "-E", "-x", "c++", "/dev/null", "-v"]
    );
    assert_eq!(
        probe_environment(Path::new("/private"))["HOME"],
        "/private/home"
    );
    assert!(aliases(Path::new("relative"), &mut BTreeMap::new()).is_err());
}

fn dynamic_image() -> Vec<u8> {
    let mut raw = image(2);
    raw.resize(272, 0);
    raw[64..68].copy_from_slice(&1u32.to_le_bytes());
    raw[72..80].copy_from_slice(&240u64.to_le_bytes());
    raw[80..88].copy_from_slice(&0x1000u64.to_le_bytes());
    raw[104..112].copy_from_slice(&32u64.to_le_bytes());
    raw[120..124].copy_from_slice(&2u32.to_le_bytes());
    raw[128..136].copy_from_slice(&176u64.to_le_bytes());
    raw[160..168].copy_from_slice(&64u64.to_le_bytes());
    for (index, (tag, value)) in [(5u64, 0x1000u64), (10, 32), (1, 0), (29, 9)]
        .iter()
        .enumerate()
    {
        raw[176 + index * 16..184 + index * 16].copy_from_slice(&tag.to_le_bytes());
        raw[184 + index * 16..192 + index * 16].copy_from_slice(&value.to_le_bytes());
    }
    raw[240..249].copy_from_slice(b"libc.so\0\0");
    raw[249..257].copy_from_slice(b"/usr/lib");
    raw
}

#[test]
fn elf_extracts_and_bounds_dynamic_strings() {
    let raw = dynamic_image();
    assert_eq!(elf(&raw).unwrap().unwrap().needed, ["libc.so"]);
    assert_eq!(elf(&raw).unwrap().unwrap().search, ["/usr/lib"]);
    for (offset, value) in [
        (184usize, u64::MAX),
        (200, 64 << 20 | 1),
        (216, 32),
        (232, 32),
    ] {
        let mut changed = raw.clone();
        changed[offset..offset + 8].copy_from_slice(&value.to_le_bytes());
        assert!(elf(&changed).is_err());
    }
    let mut unterminated = raw;
    unterminated[240..].fill(b'x');
    assert!(elf(&unterminated).is_err());
}

#[test]
fn aliases_capture_each_hop_and_refuse_cycles() {
    let root = std::env::temp_dir().join(format!("gomad-gnu-alias-{}", std::process::id()));
    std::fs::create_dir(&root).unwrap();
    std::fs::create_dir(root.join("dir")).unwrap();
    std::os::unix::fs::symlink("dir", root.join("first")).unwrap();
    std::os::unix::fs::symlink("first", root.join("second")).unwrap();
    let mut table = BTreeMap::new();
    assert_eq!(
        aliases(&root.join("second/file"), &mut table).unwrap(),
        root.join("dir/file")
    );
    assert_eq!(table.len(), 2);
    std::os::unix::fs::symlink("loop", root.join("loop")).unwrap();
    assert_eq!(
        aliases(&root.join("loop"), &mut table).unwrap_err(),
        "GNU input alias cycle"
    );
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn probe_journal_requires_exact_producer_environment_and_lifecycle() {
    let environment = probe_environment(Path::new("/private"));
    let owner =
        serde_json::json!({"pid": 10, "start_monotonic_nanos": 100, "end_monotonic_nanos": 200});
    let row = serde_json::json!({"argv": ["/gcc", "--version"], "cwd": "/repo", "pid": 12,
        "parent_pid": 11, "stage_pid": 10, "supervisor_pid": 11, "supervisor_parent_pid": 10,
        "start_ns": 1, "end_ns": 2, "exit": 0, "observation": "completed", "active": false,
        "clock": "CLOCK_MONOTONIC", "started_monotonic_nanos": 110, "reaped_monotonic_nanos": 120,
        "environment_mode": "exact", "environment": environment, "effective_environment": environment,
        "cleanup_complete": true, "failure": null});
    let argv = vec!["/gcc".to_string(), "--version".to_string()];
    validate_probe(
        &row,
        &row,
        &argv,
        "/repo",
        &owner,
        &environment,
        &mut BTreeSet::new(),
    )
    .unwrap();
    for (field, value) in [
        ("pid", serde_json::json!(true)),
        ("exit", serde_json::json!(1)),
        ("stage_pid", serde_json::json!(9)),
        ("reaped_monotonic_nanos", serde_json::json!(210)),
        ("clock", serde_json::json!("CLOCK_REALTIME")),
        ("environment_mode", serde_json::json!("clean")),
    ] {
        let mut changed = row.clone();
        changed[field] = value;
        assert!(
            validate_probe(
                &changed,
                &changed,
                &argv,
                "/repo",
                &owner,
                &environment,
                &mut BTreeSet::new()
            )
            .is_err()
        );
    }
    let mut pids = BTreeSet::new();
    validate_probe(&row, &row, &argv, "/repo", &owner, &environment, &mut pids).unwrap();
    assert!(validate_probe(&row, &row, &argv, "/repo", &owner, &environment, &mut pids).is_err());
}

struct Proof {
    observations: Vec<Value>,
}
impl Proof {
    fn raw(
        &mut self,
        id: &str,
        boundary: &str,
        control: &[u8],
        mutations: Vec<(&str, Vec<u8>)>,
        expected: &str,
        mut validate: impl FnMut(&[u8]) -> Result<(), String>,
    ) -> Result<(), String> {
        validate(control).map_err(|e| format!("{id} valid control failed: {e}"))?;
        let mut observed = Vec::new();
        for (name, bytes) in mutations {
            let refusal = validate(&bytes)
                .err()
                .ok_or_else(|| format!("{id}/{name} accepted mutation"))?;
            require(
                refusal == expected,
                &format!("{id}/{name} wrong predicate: {refusal}; expected {expected}"),
            )?;
            observed.push(json!({"case":name,"raw_bytes":bytes,"sha256":digest(&bytes),"function_return":"Err","refusal":refusal}));
        }
        require(!observed.is_empty(), "GNU original proof lacks mutation")?;
        self.observations.push(json!({"original_id":format!("reader:native_gnu:{id}"),
            "original_source":"tools/gomad/rust/driver/native_gnu.py",
            "original_source_sha256":"00d1fc7fbf0dbba800ff3bf84c6e1fcfe3d846ec72c59441ce3dbb31af0e22e8",
            "authoritative_boundary":boundary,"intended_predicate":expected,
            "control":{"raw_bytes":control,"sha256":digest(control),"normal_return":true,"function_return":"Ok"},
            "mutations":observed}));
        Ok(())
    }
    fn json(
        &mut self,
        id: &str,
        boundary: &str,
        control: &Value,
        mutations: Vec<(&str, Value)>,
        expected: &str,
        mut validate: impl FnMut(&Value) -> Result<(), String>,
    ) -> Result<(), String> {
        let mutations = mutations
            .into_iter()
            .map(|(name, value)| Ok((name, canonical(&value)?)))
            .collect::<Result<Vec<_>, String>>()?;
        self.raw(
            id,
            boundary,
            &canonical(control)?,
            mutations,
            expected,
            |bytes| validate(&strict_json(bytes)?),
        )
    }
}
fn replace(raw: &[u8], offset: usize, value: &[u8]) -> Vec<u8> {
    let mut changed = raw.to_vec();
    changed[offset..offset + value.len()].copy_from_slice(value);
    changed
}
fn variants(control: &Value, fields: &[(&str, Value)]) -> Vec<(&'static str, Value)> {
    fields
        .iter()
        .map(|(field, value)| {
            let mut changed = control.clone();
            changed[*field] = value.clone();
            ("changed-field", changed)
        })
        .collect()
}

pub(super) fn original_cases(
    base: &Path,
    root: &Path,
    tools: &Path,
    linker: &str,
    valid: &Value,
) -> Result<Vec<Value>, String> {
    let mut proof = Proof {
        observations: Vec::new(),
    };
    proof.json(
        "native_machine:26:4",
        "host_machine",
        &json!(["linux", "x86_64"]),
        vec![
            ("foreign-os", json!(["darwin", "x86_64"])),
            ("foreign-machine", json!(["linux", "riscv64"])),
        ],
        "GNU identity source-build envelope requires native Linux aarch64/x86_64",
        |v| host_machine(v[0].as_str().unwrap(), v[1].as_str().unwrap()).map(|_| ()),
    )?;
    let raw = image(0);
    proof.raw(
        "elf:34:4",
        "elf",
        &raw,
        vec![
            ("short", raw[..63].to_vec()),
            ("elf32", replace(&raw, 4, &[1])),
            ("big-endian", replace(&raw, 5, &[2])),
        ],
        "GNU envelope supports ELF64 little-endian only",
        |b| elf(b).map(|_| ()),
    )?;
    proof.raw(
        "elf:36:4",
        "elf",
        &raw,
        vec![("foreign-machine", replace(&raw, 18, &3u16.to_le_bytes()))],
        "GNU envelope supports native x86_64/aarch64 only",
        |b| elf(b).map(|_| ()),
    )?;
    let raw = image(1);
    proof.raw(
        "elf:39:4",
        "elf",
        &raw,
        vec![
            ("entry-size", replace(&raw, 54, &55u16.to_le_bytes())),
            ("count-limit", replace(&raw, 56, &1025u16.to_le_bytes())),
            (
                "offset-overflow",
                replace(&raw, 32, &u64::MAX.to_le_bytes()),
            ),
            ("truncated-table", replace(&raw, 32, &65u64.to_le_bytes())),
        ],
        "malformed compiler ELF program headers",
        |b| elf(b).map(|_| ()),
    )?;
    let mut segment = raw.clone();
    segment[64..68].copy_from_slice(&3u32.to_le_bytes());
    proof.raw(
        "elf.segment:43:8",
        "elf::segment",
        &segment,
        vec![
            (
                "out-of-range",
                replace(&segment, 104, &121u64.to_le_bytes()),
            ),
            (
                "overflow",
                replace(
                    &replace(&segment, 72, &u64::MAX.to_le_bytes()),
                    104,
                    &1u64.to_le_bytes(),
                ),
            ),
        ],
        "malformed compiler ELF segment",
        |b| elf(b).map(|_| ()),
    )?;
    let mut two = image(2);
    two[64..68].copy_from_slice(&3u32.to_le_bytes());
    proof.raw(
        "elf:46:4",
        "elf",
        &two,
        vec![(
            "duplicate-interpreter",
            replace(&two, 120, &3u32.to_le_bytes()),
        )],
        "multiple compiler ELF interpreters",
        |b| elf(b).map(|_| ()),
    )?;
    let raw = dynamic_image();
    // Replace PT_LOAD by a second PT_DYNAMIC; count rejects before parsing entries.
    proof.raw(
        "elf:48:4",
        "elf",
        &raw,
        vec![("duplicate-dynamic", replace(&raw, 64, &2u32.to_le_bytes()))],
        "multiple compiler ELF dynamic tables",
        |b| elf(b).map(|_| ()),
    )?;
    proof.raw(
        "elf:52:4",
        "elf",
        &raw,
        vec![(
            "unaligned-dynamic",
            replace(&raw, 160, &63u64.to_le_bytes()),
        )],
        "malformed compiler ELF dynamic entries",
        |b| elf(b).map(|_| ()),
    )?;
    proof.raw(
        "elf:61:4",
        "elf",
        &raw,
        vec![
            ("missing-strtab", replace(&raw, 176, &4u64.to_le_bytes())),
            ("missing-strsz", replace(&raw, 192, &9u64.to_le_bytes())),
            ("duplicate-strtab", replace(&raw, 208, &5u64.to_le_bytes())),
            ("duplicate-strsz", replace(&raw, 208, &10u64.to_le_bytes())),
        ],
        "compiler ELF string table missing",
        |b| elf(b).map(|_| ()),
    )?;
    let mut large = raw.clone();
    large[104..112].copy_from_slice(&(65u64 << 20).to_le_bytes());
    proof.raw(
        "elf:64:4",
        "elf",
        &raw,
        vec![
            ("outside-load", replace(&raw, 184, &0x2000u64.to_le_bytes())),
            (
                "table-size-limit",
                replace(&large, 200, &((64u64 << 20) + 1).to_le_bytes()),
            ),
        ],
        "compiler ELF string table outside load segments",
        |b| elf(b).map(|_| ()),
    )?;
    let mut unterminated = raw.clone();
    unterminated[240..].fill(b'x');
    proof.raw(
        "elf.text:67:8",
        "elf::text",
        &raw,
        vec![
            ("index-at-end", replace(&raw, 216, &32u64.to_le_bytes())),
            ("missing-nul", unterminated),
        ],
        "unterminated compiler ELF string",
        |b| elf(b).map(|_| ()),
    )?;
    let fixtures = base.join("original-cases");
    fs::create_dir(&fixtures).map_err(|e| e.to_string())?;
    let directory = fixtures.join("directory");
    fs::create_dir(&directory).map_err(|e| e.to_string())?;
    let file = fixtures.join("file");
    fs::write(&file, b"literal GNU boundary file\n").map_err(|e| e.to_string())?;
    let cycle = fixtures.join("cycle");
    std::os::unix::fs::symlink("cycle", &cycle).map_err(|e| e.to_string())?;
    proof.json(
        "aliases:76:4",
        "aliases",
        &json!(directory),
        vec![("relative", json!("relative/input"))],
        "GNU input must be absolute",
        |v| aliases(Path::new(v.as_str().unwrap()), &mut BTreeMap::new()).map(|_| ()),
    )?;
    proof.json(
        "aliases:92:4",
        "aliases",
        &json!(directory),
        vec![("cycle", json!(cycle))],
        "GNU input alias cycle",
        |v| aliases(Path::new(v.as_str().unwrap()), &mut BTreeMap::new()).map(|_| ()),
    )?;
    proof.json(
        "validate_linker:110:4",
        "linker_path",
        &json!(linker),
        vec![
            ("symlink-driver", json!("/usr/bin/gcc")),
            ("foreign-placement", json!(file)),
            ("relative", json!("gcc")),
        ],
        "GNU envelope requires its supported canonical system GCC driver",
        |v| linker_path(Path::new(v.as_str().unwrap())),
    )?;
    let expected = native_machine()?;
    let native = replace(&image(0), 18, &expected.to_le_bytes());
    let foreign = replace(
        &native,
        18,
        &(if expected == 62 { 183u16 } else { 62 }).to_le_bytes(),
    );
    for (id, message) in [
        (
            "validate_linker:114:4",
            "GNU driver is not a native supported ELF executable",
        ),
        ("derive:160:4", "GNU compiler is not native"),
    ] {
        proof.raw(
            id,
            "native_elf",
            &native,
            vec![
                ("nonelf", b"ordinary text".to_vec()),
                ("foreign-native-elf", foreign.clone()),
            ],
            message,
            |raw| native_elf(raw, expected, message),
        )?;
    }
    let root_info = fs::metadata(root).map_err(|e| e.to_string())?;
    let attributes =
        json!({"directory":root_info.is_dir(),"mode":root_info.mode(),"uid":root_info.uid()});
    proof.json(
        "derive:118:4",
        "directory_attributes",
        &attributes,
        variants(
            &attributes,
            &[
                ("directory", json!(false)),
                ("mode", json!(0o755)),
                ("uid", json!(root_info.uid().wrapping_add(1))),
            ],
        ),
        "GNU prefreeze root is not its private actual owner directory",
        |v| {
            directory_attributes(
                v["directory"].as_bool().unwrap(),
                v["mode"].as_u64().unwrap() as u32,
                v["uid"].as_u64().unwrap() as u32,
                "GNU prefreeze root is not its private actual owner directory",
            )
        },
    )?;
    let linked_root = fixtures.join("linked-root");
    std::os::unix::fs::symlink(root, &linked_root).map_err(|e| e.to_string())?;
    proof.json(
        "derive:118:4",
        "private_directory",
        &json!(root),
        vec![
            ("symlink-root", json!(linked_root)),
            ("file-root", json!(file)),
        ],
        "GNU prefreeze root is not its private actual owner directory",
        |v| private_directory(Path::new(v.as_str().unwrap())),
    )?;
    let tool = document(tools)?;
    proof.json(
        "derive:122:4",
        "static_plan",
        &tool,
        variants(&tool, &[("schema", json!("foreign/v1"))]),
        "GNU envelope requires the actual static tool plan",
        static_plan,
    )?;
    let owner = document(&root.join("producer.json"))?;
    let pins = source_pins(Path::new(tool["repo"].as_str().unwrap()))?;
    let mut extra = owner.clone();
    extra["foreign"] = json!(true);
    let mut source = owner.clone();
    let first = source["sources"]
        .as_object()
        .unwrap()
        .keys()
        .next()
        .unwrap()
        .clone();
    source["sources"][first] = json!("changed");
    let mut owner_mutations = variants(
        &owner,
        &[
            ("pid", json!(true)),
            ("pid", json!(0)),
            ("start_monotonic_nanos", json!(0)),
            ("start_monotonic_nanos", json!(true)),
            (
                "end_monotonic_nanos",
                owner["start_monotonic_nanos"].clone(),
            ),
        ],
    );
    owner_mutations.extend([("extra-owner-key", extra), ("changed-source", source)]);
    proof.json(
        "derive:132:4",
        "producer_identity",
        &owner,
        owner_mutations,
        "GNU prefreeze producer identity changed",
        |v| producer_identity(v, &pins),
    )?;
    let row = document(&root.join("machine.command.json"))?;
    let process = document(&root.join("machine.process.json"))?;
    let env = probe_environment(root);
    let argv: Vec<String> =
        serde_json::from_value(row["argv"].clone()).map_err(|e| e.to_string())?;
    let journal = json!({"row":row,"process":process,"seen":[]});
    let check_journal = |v: &Value| {
        let mut seen: BTreeSet<u64> = v["seen"]
            .as_array()
            .unwrap()
            .iter()
            .map(|n| n.as_u64().unwrap())
            .collect();
        validate_probe(
            &v["row"],
            &v["process"],
            &argv,
            tool["repo"].as_str().unwrap(),
            &owner,
            &env,
            &mut seen,
        )
    };
    let mut environment = Vec::new();
    for field in ["environment_mode", "environment", "effective_environment"] {
        let mut changed = journal.clone();
        changed["row"][field] = if field == "environment_mode" {
            json!("clean")
        } else {
            json!({})
        };
        environment.push(("substituted-environment", changed));
    }
    proof.json(
        "derive:143:8",
        "validate_probe",
        &journal,
        environment,
        "GNU probe substituted its environment",
        check_journal,
    )?;
    let mut lifecycles = Vec::new();
    for (field, value) in [
        ("parent_pid", json!(true)),
        ("stage_pid", json!(1)),
        ("supervisor_parent_pid", json!(1)),
        ("clock", json!("CLOCK_REALTIME")),
        ("started_monotonic_nanos", json!(0)),
        (
            "started_monotonic_nanos",
            row["reaped_monotonic_nanos"].clone(),
        ),
        (
            "reaped_monotonic_nanos",
            json!(owner["end_monotonic_nanos"].as_u64().unwrap() + 1),
        ),
    ] {
        let mut changed = journal.clone();
        changed["row"][field] = value;
        lifecycles.push(("borrowed-lifecycle", changed));
    }
    let mut reused = journal.clone();
    reused["seen"] = json!([row["pid"].clone()]);
    lifecycles.push(("reused-pid", reused));
    proof.json(
        "derive:145:8",
        "validate_probe",
        &journal,
        lifecycles,
        "GNU probe borrowed another producer/lifecycle",
        check_journal,
    )?;
    let mut differs = Vec::new();
    for field in [
        "parent_pid",
        "started_monotonic_nanos",
        "reaped_monotonic_nanos",
    ] {
        let mut changed = journal.clone();
        changed["process"][field] = json!(process[field].as_u64().unwrap() + 1);
        differs.push(("process-command-differ", changed));
    }
    proof.json(
        "derive:151:12",
        "validate_probe",
        &journal,
        differs,
        "GNU probe process/command lifecycle differs",
        check_journal,
    )?;
    let identity = json!({"machine":output(root,"machine")?,"linker":linker,"version":output(root,"version")?,"gcc":output(root,"gcc-version")?,"specs":output(root,"specs")?,"sysroot":output(root,"sysroot")?});
    proof.json(
        "derive:155:4",
        "compiler_identity",
        &identity,
        variants(
            &identity,
            &[
                ("machine", json!("foreign")),
                ("linker", json!("/foreign/gcc")),
                ("version", json!("clang")),
                ("gcc", json!("")),
                ("gcc", json!("1..2")),
                ("gcc", json!("1.a")),
                ("specs", json!("")),
                ("sysroot", json!("/foreign")),
            ],
        ),
        "unsupported GNU compiler family/machine/specs/sysroot",
        |v| {
            compiler_identity(
                v["machine"].as_str().unwrap(),
                v["linker"].as_str().unwrap(),
                v["version"].as_str().unwrap(),
                v["gcc"].as_str().unwrap(),
                v["specs"].as_str().unwrap(),
                v["sysroot"].as_str().unwrap(),
            )
        },
    )?;
    proof.json(
        "derive:159:4",
        "host_matches",
        &json!(expected),
        vec![("foreign-host", json!(if expected == 62 { 183 } else { 62 }))],
        "GNU toolchain does not match this actual native source-build host",
        |v| host_matches(v.as_u64().unwrap() as u16, expected),
    )?;
    proof.json(
        "derive:165:8",
        "which",
        &json!("git"),
        vec![("missing-delegate", json!("gomad-no-such-gnu-delegate-242"))],
        "missing GNU delegate gomad-no-such-gnu-delegate-242",
        |v| which(v.as_str().unwrap()).map(|_| ()),
    )?;
    let delegated = json!({"path":valid["programs"]["cc1"],"elf":native});
    let mut changed = delegated.clone();
    changed["elf"] = json!(foreign);
    let mut placement = delegated.clone();
    placement["path"] = json!(file);
    proof.json(
        "derive:167:8",
        "delegate",
        &delegated,
        vec![("foreign-elf", changed), ("foreign-placement", placement)],
        "unsupported GNU delegate placement or architecture",
        |v| {
            let raw: Vec<u8> =
                serde_json::from_value(v["elf"].clone()).map_err(|e| e.to_string())?;
            delegate(
                Path::new(v["path"].as_str().unwrap()),
                &raw,
                identity["machine"].as_str().unwrap(),
                expected,
            )
        },
    )?;
    proof.json(
        "derive.supported:172:8",
        "Closure::supported",
        &json!("/usr/include"),
        vec![("foreign-root", json!(directory))],
        "GNU search path escapes the supported system envelope",
        |v| {
            Closure::default()
                .supported(
                    Path::new(v.as_str().unwrap()),
                    identity["machine"].as_str().unwrap(),
                )
                .map(|_| ())
        },
    )?;
    let walk = fixtures.join("walk");
    fs::create_dir(&walk).map_err(|e| e.to_string())?;
    let escaped = fixtures.join("escaped-walk");
    fs::create_dir(&escaped).map_err(|e| e.to_string())?;
    std::os::unix::fs::symlink(&directory, escaped.join("foreign")).map_err(|e| e.to_string())?;
    proof.json(
        "derive.walk:187:20",
        "Closure::walk",
        &json!(walk),
        vec![("directory-alias-escape", json!(escaped))],
        "GNU directory alias escapes supported compiler/include roots",
        |v| Closure::default().walk(Path::new(v.as_str().unwrap())),
    )?;
    let searches = output(root, "search")?;
    for (kind, id, message) in [
        (
            "programs",
            "derive:191:4",
            "GNU executable search probe missing",
        ),
        (
            "libraries",
            "derive:206:4",
            "GNU library search probe missing",
        ),
    ] {
        proof.json(
            id,
            "search_roots",
            &json!(searches),
            vec![
                ("missing-search", json!("foreign: =/usr/lib")),
                (
                    "duplicate-search",
                    json!(format!("{searches}\n{kind}: =/usr/lib\n")),
                ),
            ],
            message,
            |v| search_roots(v.as_str().unwrap(), kind).map(|_| ()),
        )?;
    }
    proof.json(
        "derive:195:8",
        "Closure::program_root",
        &json!("/usr/bin"),
        vec![("foreign-program-search", json!(directory))],
        "GNU executable search root escapes its supported prefix",
        |v| {
            Closure::default()
                .program_root(
                    Path::new(v.as_str().unwrap()),
                    identity["machine"].as_str().unwrap(),
                )
                .map(|_| ())
        },
    )?;
    let cc1 = Path::new(valid["programs"]["cc1"].as_str().unwrap())
        .parent()
        .unwrap();
    proof.json(
        "derive:203:4",
        "compiler_root",
        &json!(cc1),
        vec![("foreign-internal-root", json!("/usr/bin"))],
        "GNU internal compiler root missing",
        |v| compiler_root(Path::new(v.as_str().unwrap())),
    )?;
    for (id, header, message) in [
        (
            "derive:212:8",
            false,
            "GNU library search root is not a directory",
        ),
        ("derive:227:12", true, "GNU header search input missing"),
    ] {
        proof.json(
            id,
            "search_directory",
            &json!(directory),
            vec![
                ("regular-file", json!(file)),
                ("missing-directory", json!(fixtures.join("absent"))),
            ],
            message,
            |v| search_directory(Path::new(v.as_str().unwrap()), header),
        )?;
    }
    let headers = String::from_utf8(regular(&root.join("include-c.stderr"), 128 << 20, None)?)
        .map_err(|e| e.to_string())?;
    proof.json(
        "derive:224:8",
        "header_roots",
        &json!(headers),
        vec![
            ("missing-start", json!("End of search list.")),
            (
                "missing-end",
                json!("#include <...> search starts here:\n /usr/include\n"),
            ),
        ],
        "GNU actual header search probe missing",
        |v| header_roots(v.as_str().unwrap()).map(|_| ()),
    )?;
    proof.json(
        "derive:229:4",
        "no_preload",
        &json!(fixtures.join("absent-preload")),
        vec![("actual-preload-file", json!(file))],
        "GNU envelope refuses a global loader preload",
        |v| no_preload(Path::new(v.as_str().unwrap())),
    )?;
    proof.json(
        "derive:255:8",
        "closure_machine",
        &json!(expected),
        vec![(
            "foreign-dynamic-machine",
            json!(if expected == 62 { 183 } else { 62 }),
        )],
        "GNU closure contains a foreign dynamic ELF machine",
        |v| closure_machine(v.as_u64().unwrap() as u16, expected),
    )?;
    proof.json(
        "derive:260:12",
        "needed_name",
        &json!("libc.so.6"),
        vec![
            ("absolute-needed", json!("/lib/libc.so.6")),
            ("relative-needed-path", json!("subdir/libc.so.6")),
        ],
        "GNU closure has an unsupported absolute DT_NEEDED entry",
        |v| needed_name(v.as_str().unwrap()),
    )?;
    proof.json(
        "derive:263:16",
        "loader_search",
        &json!("$ORIGIN/lib"),
        vec![
            ("empty-search", json!("")),
            ("unknown-expansion", json!("$LIB/lib")),
        ],
        "unsupported GNU dynamic-loader expansion",
        |v| loader_search(v.as_str().unwrap(), "/usr/lib").map(|_| ()),
    )?;
    proof.json(
        "derive:265:16",
        "loader_search",
        &json!("${ORIGIN}/lib"),
        vec![("relative-search", json!("relative/lib"))],
        "relative GNU ELF loader search",
        |v| loader_search(v.as_str().unwrap(), "/usr/lib").map(|_| ()),
    )?;
    proof.json(
        "derive:276:12",
        "dependency_found",
        &json!(true),
        vec![("unresolved-needed", json!(false))],
        "GNU compiler ELF dependency is not captured: libc.so.6",
        |v| dependency_found(v.as_bool().unwrap(), "libc.so.6"),
    )?;
    let links: BTreeMap<String, String> = fs::read_dir(root.join("bin"))
        .map_err(|e| e.to_string())?
        .map(|e| {
            let e = e.map_err(|e| e.to_string())?;
            Ok((
                e.file_name()
                    .into_string()
                    .map_err(|_| "GNU fixture UTF8")?,
                fs::read_link(e.path())
                    .map_err(|e| e.to_string())?
                    .to_string_lossy()
                    .into_owned(),
            ))
        })
        .collect::<Result<_, String>>()?;
    let empty_bin = fixtures.join("empty-bin");
    {
        use std::os::unix::fs::DirBuilderExt as _;
        fs::DirBuilder::new()
            .mode(0o700)
            .create(&empty_bin)
            .map_err(|e| e.to_string())?;
    }
    proof.json(
        "derive:280:4",
        "lookup_directory",
        &json!(root.join("bin")),
        vec![("missing-entries", json!(empty_bin))],
        "GNU executable lookup contains foreign or missing entries",
        |v| lookup_directory(Path::new(v.as_str().unwrap()), &links),
    )?;
    let wrong = fixtures.join("wrong-link");
    std::os::unix::fs::symlink("/foreign/compiler", &wrong).map_err(|e| e.to_string())?;
    proof.json(
        "derive:285:8",
        "lookup_link",
        &json!(root.join("bin/cc")),
        vec![
            ("wrong-target", json!(wrong)),
            ("regular-entry", json!(file)),
        ],
        "GNU closed executable lookup changed",
        |v| lookup_link(Path::new(v.as_str().unwrap()), linker),
    )?;
    proof.json(
        "derive:287:4",
        "envelope_bounds",
        &json!([100_000u64, 16u64 << 30]),
        vec![
            ("file-count", json!([100_001u64, 16u64 << 30])),
            ("total-bytes", json!([100_000u64, (16u64 << 30) + 1])),
        ],
        "GNU compiler envelope exceeds bounded native toolchain inputs",
        |v| envelope_bounds(v[0].as_u64().unwrap() as usize, v[1].as_u64().unwrap()),
    )?;
    let alias_parent = fixtures.join("alias-parent");
    std::os::unix::fs::symlink(&directory, &alias_parent).map_err(|e| e.to_string())?;
    proof.json(
        "prepare:296:4",
        "fresh_root",
        &json!(fixtures.join("fresh-root")),
        vec![
            ("already-exists", json!(directory)),
            ("relative", json!("relative/root")),
            ("linked-parent", json!(alias_parent.join("new"))),
        ],
        "GNU prefreeze root must be fresh canonical absolute",
        |v| fresh_root(Path::new(v.as_str().unwrap())),
    )?;
    let null_info = fs::metadata("/dev/null").map_err(|e| e.to_string())?;
    proof.json(
        "prepare:301:4",
        "null_device",
        &json!([null_info.mode() as u64, null_info.rdev()]),
        vec![
            ("ordinary-file", json!([0o100_600u64, null_info.rdev()])),
            (
                "different-device",
                json!([null_info.mode() as u64, rustix::fs::makedev(1, 5)]),
            ),
        ],
        "GNU header probes require actual Linux /dev/null",
        |v| null_device(v[0].as_u64().unwrap() as u32, v[1].as_u64().unwrap()),
    )?;
    proof.json(
        "prepare:308:12",
        "gcc_version",
        &json!(output(root, "version")?),
        vec![
            ("other-family", json!("clang version 19")),
            ("empty", json!("")),
        ],
        "GNU envelope requires actual GNU GCC",
        |v| gcc_version(v.as_str().unwrap()),
    )?;
    let binding = json!({"toolchain_plan":root.join("gnu-toolchain.json")});
    let mut extra = binding.clone();
    extra["foreign"] = json!(true);
    proof.json(
        "inspect:325:4",
        "harness_binding",
        &binding,
        vec![("missing-binding", json!({})), ("extra-binding", extra)],
        "identity harness requires its actual prepared GNU toolchain plan",
        harness_binding,
    )?;
    let declared = json!({"plan":valid,"path":root.join("gnu-toolchain.json")});
    let mut foreign_path = declared.clone();
    foreign_path["path"] = json!(fixtures.join("gnu-toolchain.json"));
    let mut foreign_schema = declared.clone();
    foreign_schema["plan"]["schema"] = json!("foreign/v1");
    let mut foreign_tools = declared.clone();
    foreign_tools["plan"]["tools_plan"] = json!(fixtures.join("foreign-tools"));
    proof.json(
        "inspect:328:4",
        "declared_roots",
        &declared,
        vec![
            ("foreign-root", foreign_path),
            ("foreign-schema", foreign_schema),
            ("foreign-tools", foreign_tools),
        ],
        "GNU toolchain plan has foreign roots/static tools",
        |v| {
            declared_roots(
                &v["plan"],
                tools,
                Path::new(v["path"].as_str().unwrap()),
                root,
            )
        },
    )?;
    let mut changed = valid.clone();
    changed["machine"] = json!("foreign");
    proof.json(
        "inspect:331:4",
        "closure_equal",
        valid,
        vec![("changed-reopened-closure", changed)],
        "GNU toolchain probes/delegates/search/loader closure changed",
        |v| closure_equal(v, valid),
    )?;
    let ids: BTreeSet<_> = proof
        .observations
        .iter()
        .filter_map(|v| v["original_id"].as_str())
        .collect();
    require(
        ids.len() == 50,
        &format!("GNU original proof selects {} IDs instead of50", ids.len()),
    )?;
    let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("src/cycle/preparation/inputs/gnu.rs");
    let tests = source.with_file_name("gnu_tests.rs");
    let source_hash = digest(&read(&source, 512 << 20)?);
    let tests_hash = digest(&read(&tests, 512 << 20)?);
    let literal = read(&file, 512 << 20)?;
    let file_info = fs::metadata(&file).map_err(|e| e.to_string())?;
    let primitive_files = json!([
        {"path":file,"raw_bytes":literal,"sha256":digest(&literal),"mode":file_info.mode(),"uid":file_info.uid()},
        {"path":cycle,"symlink_target_bytes":fs::read_link(&cycle).map_err(|e|e.to_string())?.as_os_str().as_encoded_bytes(),"sha256":digest(b"cycle")},
        {"path":wrong,"symlink_target_bytes":b"/foreign/compiler","sha256":digest(b"/foreign/compiler")},
        {"path":linked_root,"symlink_target_bytes":root.as_os_str().as_encoded_bytes(),"sha256":digest(root.as_os_str().as_encoded_bytes())}
    ]);
    for row in &mut proof.observations {
        row["current_source"] = json!(source);
        row["current_source_sha256"] = json!(source_hash);
        row["test_source_sha256"] = json!(tests_hash);
        row["primitive_files"] = primitive_files.clone();
        row["genuine_control"] = json!({"root":root,"tool_plan":tools,"commands":valid["commands"],"control":"normal-return-prepare-and-independent-inspect"});
    }
    Ok(proof.observations)
}
