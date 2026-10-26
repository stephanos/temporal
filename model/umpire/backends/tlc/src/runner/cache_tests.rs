use super::*;
use std::{os::unix::fs::PermissionsExt, process::Command, sync::Arc, time::Duration};
use umpire_tla::Type;

fn owner() -> Result<Scratch, Refusal> {
    Scratch::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?)
}
fn bundle(root: &Path) -> Bundle {
    let id = cache::record_id("SPECIFICATION Spec\n");
    Bundle {
        key: structural_sha256(&"key"),
        schema: structural_sha256(&"schema"),
        root: root.to_owned(),
        expected: vec![id],
        replay: false,
        limits: Limits::default(),
        records: Mutex::new(BTreeMap::new()),
    }
}
fn fill(bundle: &Bundle) {
    bundle.records.lock().unwrap_or_else(|error| panic!("{error:?}")).insert(
        bundle.expected[0].clone(),
        Record { output: "output".into(), dump: "dump".into() },
    );
}

#[test]
fn manifest_integrity_limits_and_interrupted_publication() -> Result<(), Refusal> {
    let scratch = owner()?;
    let result = (|| {
        let cache = bundle(scratch.path());
        let abandoned = Scratch::new_in(scratch.path(), "quick-tlc-v1")?;
        fs::write(abandoned.path().join("manifest"), "partial")?;
        assert!(bundle(scratch.path()).load().is_err());
        fill(&cache);
        cache.publish()?;
        assert!(bundle(scratch.path()).load().is_ok());
        let directory = scratch.path().join("quick-tlc-v1").join(&cache.key);
        let manifest_path = directory.join("manifest");
        let original = fs::read_to_string(&manifest_path)?;
        for corrupt in [
            original.replace(VERSION_MARKER, "unknown-version"),
            original.replace(&cache.key, &structural_sha256(&"wrong-key")),
            original.replace(&cache.schema, &structural_sha256(&"wrong-schema")),
            original.replace("complete-success", "partial"),
            original.replace("\n1\n", "\n2\n"),
            original.replace(" 0\n", " 1\n"),
            original.replace(" 6 ", " 9999999999999999999 "),
            original[..original.len() / 2].to_owned(),
            format!("{original}extra\n"),
        ] {
            fs::write(&manifest_path, corrupt)?;
            assert!(bundle(scratch.path()).load().is_err());
        }
        fs::write(&manifest_path, &original)?;
        let mut limited = bundle(scratch.path());
        limited.limits.output_bytes = 1;
        assert!(limited.load().is_err());
        let mut limited = bundle(scratch.path());
        limited.limits.dump_bytes = 1;
        assert!(limited.load().is_err());
        fs::write(directory.join("extra"), "unexpected")?;
        assert!(bundle(scratch.path()).load().is_err());
        fs::remove_file(directory.join("extra"))?;
        fs::write(directory.join("0.out"), "tamper")?;
        assert!(bundle(scratch.path()).load().is_err());
        fs::remove_file(directory.join("0.out"))?;
        std::os::unix::fs::symlink(abandoned.path().join("manifest"), directory.join("0.out"))?;
        assert!(bundle(scratch.path()).load().is_err());
        abandoned.finish(Ok(()))
    })();
    scratch.finish(result)
}
const VERSION_MARKER: &str = cache::VERSION;

#[test]
fn concurrent_writers_and_readers_see_only_complete_bundles() -> Result<(), Refusal> {
    let scratch = owner()?;
    let result = {
        let barrier = Arc::new(std::sync::Barrier::new(5));
        std::thread::scope(|scope| {
            let mut handles = Vec::new();
            for _ in 0..4 {
                let barrier = barrier.clone();
                let root = scratch.path();
                handles.push(scope.spawn(move || -> Result<(), Refusal> {
                    let writer = bundle(root);
                    fill(&writer);
                    barrier.wait();
                    writer.publish()?;
                    let mut reader = bundle(root);
                    reader.load()?;
                    assert_eq!(
                        reader.records.lock().unwrap_or_else(|error| panic!("{error:?}"))
                            [&reader.expected[0]]
                            .output,
                        "output"
                    );
                    Ok(())
                }));
            }
            barrier.wait();
            for _ in 0..100 {
                let mut reader = bundle(scratch.path());
                if reader.load().is_ok() {
                    assert_eq!(
                        reader.records.lock().unwrap_or_else(|error| panic!("{error:?}")).len(),
                        1
                    );
                }
            }
            for handle in handles {
                handle.join().unwrap_or_else(|error| panic!("{error:?}"))?;
            }
            Ok(())
        })
    };
    scratch.finish(result)
}

fn backend(root: &Path) -> Tlc {
    Tlc {
        root: root.to_owned(),
        java: root.join("java"),
        jar: root.join("tlc.jar"),
        limits: Limits::default(),
        property_workers: 2,
    }
}
fn schema() -> Schema {
    Schema { fields: BTreeMap::from([("x".into(), Type::Bool)]), observers: BTreeMap::new() }
}

#[test]
#[allow(clippy::too_many_lines)]
fn every_effective_identity_input_invalidates() {
    let original = backend(Path::new("/unused"));
    let schema = schema();
    let key = |tlc: &Tlc, name, tla, cfg, schema: &Schema, inventory: &[&str], index, jar, java| {
        tlc.cache_key(name, tla, cfg, schema, inventory, index, jar, java).0
    };
    let baseline = key(
        &original,
        "M",
        "emitted",
        "cfg",
        &schema,
        &["cfg", "progress", "negative"],
        0,
        "jar",
        "java",
    );
    for changed in [
        key(
            &original,
            "N",
            "emitted",
            "cfg",
            &schema,
            &["cfg", "progress", "negative"],
            0,
            "jar",
            "java",
        ),
        key(
            &original,
            "M",
            "changed",
            "cfg",
            &schema,
            &["cfg", "progress", "negative"],
            0,
            "jar",
            "java",
        ),
        key(
            &original,
            "M",
            "emitted",
            "changed",
            &schema,
            &["cfg", "progress", "negative"],
            0,
            "jar",
            "java",
        ),
        key(
            &original,
            "M",
            "emitted",
            "cfg",
            &schema,
            &["changed", "progress", "negative"],
            0,
            "jar",
            "java",
        ),
        key(
            &original,
            "M",
            "emitted",
            "cfg",
            &schema,
            &["cfg", "changed", "negative"],
            0,
            "jar",
            "java",
        ),
        key(
            &original,
            "M",
            "emitted",
            "cfg",
            &schema,
            &["cfg", "progress", "changed"],
            0,
            "jar",
            "java",
        ),
        key(&original, "M", "emitted", "cfg", &schema, &["cfg", "progress"], 0, "jar", "java"),
        key(
            &original,
            "M",
            "emitted",
            "cfg",
            &schema,
            &["cfg", "progress", "negative"],
            1,
            "jar",
            "java",
        ),
        key(
            &original,
            "M",
            "emitted",
            "cfg",
            &schema,
            &["cfg", "progress", "negative"],
            0,
            "changed",
            "java",
        ),
        key(
            &original,
            "M",
            "emitted",
            "cfg",
            &schema,
            &["cfg", "progress", "negative"],
            0,
            "jar",
            "changed",
        ),
    ] {
        assert_ne!(baseline, changed);
    }
    for change in 0..6 {
        let mut tlc = original.clone();
        match change {
            0 => tlc.limits.timeout = Duration::from_secs(7),
            1 => tlc.limits.output_bytes = 17,
            2 => tlc.limits.dump_bytes = 17,
            3 => tlc.limits.heap_mb = 17,
            4 => tlc.property_workers = 1,
            _ => tlc.java = PathBuf::from("/other-java"),
        }
        assert_ne!(
            baseline,
            key(
                &tlc,
                "M",
                "emitted",
                "cfg",
                &schema,
                &["cfg", "progress", "negative"],
                0,
                "jar",
                "java"
            )
        );
    }
    for changed in [
        Schema {
            fields: BTreeMap::from([("renamed".into(), Type::Bool)]),
            observers: BTreeMap::new(),
        },
        Schema {
            fields: BTreeMap::from([("x".into(), Type::Int { min: 0, max: 1 })]),
            observers: BTreeMap::new(),
        },
        Schema {
            fields: schema.fields.clone(),
            observers: BTreeMap::from([("o".into(), Type::Bool)]),
        },
    ] {
        assert_ne!(
            baseline,
            key(
                &original,
                "M",
                "emitted",
                "cfg",
                &changed,
                &["cfg", "progress", "negative"],
                0,
                "jar",
                "java"
            )
        );
    }
    let bounded = umpire_tla::Bounded {
        name: "Ticks".into(),
        field: None,
        backing: "u8".into(),
        min: 0,
        max: 255,
        low: 1,
        high: 3,
    };
    let nominal = Schema {
        fields: BTreeMap::from([("x".into(), Type::Bounded(bounded.clone()))]),
        observers: BTreeMap::from([("o".into(), Type::Optional(bounded.clone()))]),
    };
    let nominal_key = key(&original, "M", "emitted", "cfg", &nominal, &["cfg"], 0, "jar", "java");
    for observer in [false, true] {
        for change in 0..7 {
            let mut changed = nominal.clone();
            let mut ty = bounded.clone();
            match change {
                0 => ty.name = "Other".into(),
                1 => ty.field = Some("value".into()),
                2 => ty.backing = "u16".into(),
                3 => ty.min = -1,
                4 => ty.max = 127,
                5 => ty.low = 0,
                _ => ty.high = 4,
            }
            if observer {
                changed.observers.insert("o".into(), Type::Optional(ty));
            } else {
                changed.fields.insert("x".into(), Type::Bounded(ty));
            }
            assert_ne!(
                nominal_key,
                key(&original, "M", "emitted", "cfg", &changed, &["cfg"], 0, "jar", "java")
            );
        }
    }
    assert!(!cache::enabled(Some("1"), Some("1")));
    assert!(!cache::enabled(Some("0"), None));
    assert!(!cache::enabled(None, None));
}

fn mixed_module() -> umpire_tla::Module {
    let cfg = "SPECIFICATION Spec\nINVARIANT A\n";
    umpire_tla::Module {
        name: "M".into(),
        tla: "---- MODULE M ----\nA == TRUE\nB == FALSE\n====\n".into(),
        cfg: cfg.into(),
        metadata: umpire_tla::Metadata {
            state_name: "Model".into(),
            world_path: "World".into(),
            parameters: vec![],
            binding: None,
            fields: vec![umpire_tla::Field { name: "x".into(), ty: Type::Bool }],
            actions: vec![],
            catalog: vec![],
            invariants: vec![umpire_tla::Property {
                name: "good".into(),
                operator: "A".into(),
                cfg: cfg.into(),
            }],
            postconditions: vec![],
            arithmetic: vec![],
            progress: vec![],
            negative: vec![umpire_tla::Property {
                name: "bad".into(),
                operator: "B".into(),
                cfg: "SPECIFICATION Spec\nINVARIANT B\n".into(),
            }],
            observer_fields: vec![],
            observer_types: BTreeMap::new(),
            outcome_type: Type::Bool,
        },
    }
}

#[test]
fn cache_child() -> Result<(), Box<dyn std::error::Error>> {
    let Some(root) = std::env::var_os("UMPIRE_CACHE_TEST_ROOT") else {
        return Ok(());
    };
    let root = PathBuf::from(root);
    if std::env::var_os("UMPIRE_CACHE_TEST_REAL_KETTLE").is_some() {
        let source = format!(
            "{}\n{}",
            include_str!("../../../../dev/fixtures/fixtures/sample_kettle/model.rs"),
            include_str!("../../../../dev/fixtures/fixtures/sample_kettle/tests.rs")
        );
        let module = umpire_tla::Tla::emit(&umpire_tla::Lift::parse("kettle.rs", &source)?)?;
        let checked = Tlc::discover(&root)?.check_module(&module)?;
        assert_eq!(checked.states.len(), 283);
        assert_eq!(checked.properties.len(), 5);
        assert_eq!(
            checked
                .properties
                .values()
                .filter(|property| property.verdict == Verdict::Verified)
                .count(),
            4
        );
        assert_eq!(module.metadata.negative.len(), 1);
        for property in &module.metadata.negative {
            assert_eq!(checked.properties[&property.operator].verdict, Verdict::Refuted);
        }
        for property in checked.properties.values() {
            if property.verdict == Verdict::Refuted {
                assert!(property.trace.is_some());
            }
        }
        return Ok(());
    }
    let tlc = backend(&root);
    let result = if fs::read_to_string(root.join("mode"))? == "mixed" {
        let module = mixed_module();
        if std::env::var("UMPIRE_CACHE_TEST_EXPECT").as_deref() == Ok("identity-refresh") {
            for change in [None, Some("jar"), Some("java")] {
                match change {
                    Some("jar") => fs::write(root.join("tlc.jar"), "changed jar")?,
                    Some("java") => fs::write(root.join("java-version"), "fake-java-v2")?,
                    _ => {}
                }
                let before = runs(&root);
                let checked = tlc.check_module(&module)?;
                assert_eq!(checked.properties["A"].verdict, Verdict::Verified);
                assert_eq!(checked.properties["B"].verdict, Verdict::Refuted);
                assert_eq!(runs(&root), before + 4);
            }
            assert_eq!(fs::read_to_string(root.join("identities"))?.lines().count(), 3);
            return Ok(());
        }
        tlc.check_module(&module)
    } else if std::env::var_os("UMPIRE_CACHE_TEST_MODULE").is_some() {
        let module = umpire_tla::Tla::emit(&umpire_tla::Lift::parse(
            "scaling.rs",
            include_str!("../../tests/fixtures/scaling_model.rs"),
        )?)?;
        tlc.check_module(&module)
    } else {
        tlc.check(
            "M",
            "---- MODULE M ----\nA == TRUE\n====\n",
            "SPECIFICATION Spec\nINVARIANT A\n",
            &schema(),
        )
    };
    match std::env::var("UMPIRE_CACHE_TEST_EXPECT").as_deref() {
        Ok("error") => assert!(result.is_err(), "{result:?}"),
        Ok("mixed") => {
            let result = result?;
            assert_eq!(result.properties["A"].verdict, Verdict::Verified);
            assert_eq!(result.properties["B"].verdict, Verdict::Refuted);
        }
        Ok("refuted") => assert_eq!(result?.properties["A"].verdict, Verdict::Refuted),
        _ => assert!(all_verified(&result?)),
    }
    Ok(())
}

fn run_child(
    root: &Path,
    cache_root: &Path,
    enable: &str,
    disable: &str,
    expect: &str,
    module: bool,
) -> Result<String, Refusal> {
    let mut command = Command::new(std::env::current_exe()?);
    command
        .args(["--exact", "runner::cache_tests::cache_child", "--nocapture"])
        .env("UMPIRE_CACHE_TEST_ROOT", root)
        .env("UMPIRE_QUICK_CACHE_ROOT", cache_root)
        .env("UMPIRE_QUICK_CACHE", enable)
        .env("UMPIRE_CACHE_DISABLE", disable)
        .env("UMPIRE_CACHE_TEST_EXPECT", expect);
    if module {
        command.env("UMPIRE_CACHE_TEST_MODULE", "1");
    } else {
        command.env_remove("UMPIRE_CACHE_TEST_MODULE");
    }
    let output = command.output()?;
    assert!(
        output.status.success(),
        "{}{}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    Ok(String::from_utf8_lossy(&output.stderr).into_owned())
}
fn runs(root: &Path) -> usize {
    fs::read_to_string(root.join("runs")).unwrap_or_default().lines().count()
}
fn fake_backend(root: &Path) -> Result<(), Refusal> {
    fs::write(root.join("tlc.jar"), "jar identity")?;
    fs::write(root.join("mode"), "verified")?;
    fs::write(root.join("java-version"), "fake-java-v1")?;
    fs::write(
        root.join("java"),
        r#"#!/bin/bash
set -eu
base=$(dirname "$0")
if [ "${1:-}" = '-version' ]; then
 if [ "$(cat "$base/mode")" = 'identity-failure' ]; then exit 1; fi
 cat "$base/java-version"; echo version >> "$base/identities"; exit 0
fi
config=''
dump=''
while [ "$#" -gt 0 ]; do
 case "$1" in
 -dump) shift; dump=$1 ;;
 -config) shift; config=$1 ;;
 esac
 shift
done
echo run >> "$base/runs"
mode=$(cat "$base/mode")
if [ "$mode" = 'failure' ]; then exit 1; fi
if [ "$mode" = 'module-failure' ] && /usr/bin/grep -q 'Progress_' "$config"; then exit 1; fi
message() { printf '@!@!@STARTMSG %s:%s @!@!@\n%s\n@!@!@ENDMSG %s @!@!@\n' "$1" "$2" "$3" "$1"; }
if [ "$mode" = 'module-failure' ]; then
 : > "$dump"
 message 2199 0 '0 states generated, 0 distinct states found, 0 states left on queue.'
else
 printf 'State 1:\nx = TRUE\n' > "$dump"
 message 2199 0 '1 states generated, 1 distinct states found, 0 states left on queue.'
fi
if [ "$mode" = 'bad-dump' ]; then printf 'State 1:\nx = 99\n' > "$dump"; fi
if [ "$mode" = 'evaluation' ] && /usr/bin/grep -q 'INVARIANT A' "$config"; then
 message 2111 1 $'Evaluating invariant A failed.\nOverflow'
 exit 1
fi
if [ "$mode" = 'refuted' ] && /usr/bin/grep -q 'INVARIANT A' "$config"; then
 message 2110 1 'Invariant A is violated.'
 message 2217 4 $'1: <Initial predicate>\nx = TRUE'
 message 2186 0 Finished
 exit 12
fi
if [ "$mode" = 'mixed' ] && /usr/bin/grep -q 'INVARIANT B' "$config"; then
 message 2110 1 'Invariant B is violated.'
 message 2217 4 $'1: <Initial predicate>\nx = TRUE'
 message 2186 0 Finished
 exit 12
fi
message 2193 0 'Model checking completed. No error has been found.'
message 2186 0 Finished
"#,
    )?;
    fs::set_permissions(root.join("java"), fs::Permissions::from_mode(0o700))?;
    Ok(())
}

#[test]
fn populated_cache_hits_and_authoritative_disable_runs_fresh() -> Result<(), Refusal> {
    if cfg!(feature = "strategy-equivalence") {
        return Ok(());
    }
    let scratch = owner()?;
    let result = (|| {
        fake_backend(scratch.path())?;
        let cache_root = scratch.path().join("cache");
        assert!(
            run_child(scratch.path(), &cache_root, "1", "0", "verified", false)?
                .contains("cache miss")
        );
        assert_eq!(runs(scratch.path()), 2);
        assert!(
            run_child(scratch.path(), &cache_root, "1", "0", "verified", false)?
                .contains("cache hit")
        );
        assert_eq!(runs(scratch.path()), 2);
        let directory = fs::read_dir(cache_root.join("quick-tlc-v1"))?
            .next()
            .ok_or_else(|| Refusal("missing cache entry".into()))??
            .path();
        let manifest_path = directory.join("manifest");
        let manifest = fs::read_to_string(&manifest_path)?;
        for (file, malformed) in [
            ("0.out", "missing completion"),
            ("0.dump", "State 1:\nx = FALSE\n/\\ undeclared = TRUE\n"),
        ] {
            let path = directory.join(file);
            let original = fs::read_to_string(&path)?;
            // Preserve manifest integrity, forcing the current execution/schema validators to refuse.
            let modified_manifest = manifest
                .replace(&bytes_sha256(original.as_bytes()), &bytes_sha256(malformed.as_bytes()))
                .replace(&format!(" {} ", original.len()), &format!(" {} ", malformed.len()));
            fs::write(&path, malformed)?;
            fs::write(&manifest_path, modified_manifest)?;
            let before = runs(scratch.path());
            assert!(
                run_child(scratch.path(), &cache_root, "1", "0", "verified", false)?
                    .contains("cache miss")
            );
            assert_eq!(runs(scratch.path()), before + 2);
            fs::write(&path, original)?;
            fs::write(&manifest_path, &manifest)?;
        }
        let before = runs(scratch.path());
        let disabled = run_child(scratch.path(), &cache_root, "1", "1", "verified", false)?;
        assert!(disabled.contains("TLC execute"));
        assert!(!disabled.contains("cache hit"));
        assert_eq!(runs(scratch.path()), before + 2);
        run_child(scratch.path(), &cache_root, "0", "0", "verified", false)?;
        assert_eq!(runs(scratch.path()), before + 4);
        assert_eq!(fs::read_to_string(scratch.path().join("identities"))?.lines().count(), 4);
        Ok(())
    })();
    scratch.finish(result)
}

#[test]
fn verified_configuration_publishes_despite_refuted_sibling() -> Result<(), Refusal> {
    if cfg!(feature = "strategy-equivalence") {
        return Ok(());
    }
    let scratch = owner()?;
    let result = (|| {
        fake_backend(scratch.path())?;
        fs::write(scratch.path().join("mode"), "mixed")?;
        let cache_root = scratch.path().join("cache");
        let first = run_child(scratch.path(), &cache_root, "1", "0", "mixed", true)?;
        assert!(!first.contains("cache hit"));
        assert_eq!(runs(scratch.path()), 4);
        let second = run_child(scratch.path(), &cache_root, "1", "0", "mixed", true)?;
        assert!(second.contains("TLC cache hit M configuration 0"), "{second}");
        assert!(second.contains("TLC cache miss M configuration 1"), "{second}");
        assert_eq!(runs(scratch.path()), 6);
        assert_eq!(fs::read_to_string(scratch.path().join("identities"))?.lines().count(), 2);
        assert_eq!(fs::read_dir(cache_root.join("quick-tlc-v1"))?.count(), 1);
        Ok(())
    })();
    scratch.finish(result)
}

#[test]
fn identity_refreshes_between_module_calls_and_is_shared_between_configurations()
-> Result<(), Refusal> {
    if cfg!(feature = "strategy-equivalence") {
        return Ok(());
    }
    let scratch = owner()?;
    let result = (|| {
        fake_backend(scratch.path())?;
        fs::write(scratch.path().join("mode"), "mixed")?;
        run_child(
            scratch.path(),
            &scratch.path().join("cache"),
            "1",
            "0",
            "identity-refresh",
            true,
        )?;
        Ok(())
    })();
    scratch.finish(result)
}

#[test]
fn cache_root_refusal_reports_its_cause_without_following_symlinks() -> Result<(), Refusal> {
    if cfg!(feature = "strategy-equivalence") {
        return Ok(());
    }
    let scratch = owner()?;
    let result = (|| {
        fake_backend(scratch.path())?;
        let real = scratch.path().join("real");
        fs::create_dir(&real)?;
        std::os::unix::fs::symlink(&real, scratch.path().join("link"))?;
        let log = run_child(
            scratch.path(),
            &scratch.path().join("link/cache"),
            "1",
            "0",
            "verified",
            false,
        )?;
        assert!(log.contains("unsafe quick cache root"), "{log}");
        assert!(!real.join("cache").exists());
        assert_eq!(runs(scratch.path()), 2);
        Ok(())
    })();
    scratch.finish(result)
}

#[test]
fn failures_refutations_and_incomplete_modules_never_publish() -> Result<(), Refusal> {
    if cfg!(feature = "strategy-equivalence") {
        return Ok(());
    }
    let scratch = owner()?;
    let result = (|| {
        fake_backend(scratch.path())?;
        for (mode, expected, module) in [
            ("refuted", "refuted", false),
            ("failure", "error", false),
            ("evaluation", "error", false),
            ("bad-dump", "error", false),
            ("identity-failure", "verified", false),
            ("module-failure", "error", true),
        ] {
            fs::write(scratch.path().join("mode"), mode)?;
            let cache_root = scratch.path().join(mode);
            for _ in 0..2 {
                run_child(scratch.path(), &cache_root, "1", "0", expected, module)?;
            }
            assert!(
                fs::read_dir(cache_root.join("quick-tlc-v1"))?.next().is_none(),
                "published {mode}"
            );
        }
        Ok(())
    })();
    scratch.finish(result)
}

#[test]
fn invalid_cached_batch_refuses_before_any_singleton_recovery() -> Result<(), Refusal> {
    let tlc = backend(Path::new("/missing-backend"));
    let cfg = "SPECIFICATION Spec\nINVARIANT A\nINVARIANT B\n";
    let (base, properties) = split_cfg(cfg)?;
    let message = |code, error, body| {
        format!("@!@!@STARTMSG {code}:{error} @!@!@\n{body}\n@!@!@ENDMSG {code} @!@!@\n")
    };
    let success = format!(
        "{}{}{}",
        message(2199, 0, "0 states generated, 0 distinct states found, 0 states left on queue."),
        message(2193, 0, "No error has been found."),
        message(2186, 0, "Finished")
    );
    for invalid in [
        message(2110, 1, "Invariant A is violated."),
        message(2111, 1, "Evaluating invariant A failed.\nOverflow"),
    ] {
        let mut cached = bundle(Path::new("/unused"));
        cached.replay = true;
        cached.records = Mutex::new(BTreeMap::from([
            (cache::record_id(&base), Record { output: success.clone(), dump: String::new() }),
            (
                cache::record_id(&property_cfg(&base, &properties)),
                Record { output: format!("{success}{invalid}"), dump: String::new() },
            ),
        ]));
        let error = tlc
            .check_strategy_cached("M", "", cfg, &schema(), true, Some(&cached))
            .err()
            .ok_or_else(|| Refusal("invalid cache batch accepted".into()))?;
        assert!(
            matches!(error, Error::Evaluation(_))
                || error.to_string().contains("cached property batch is not Verified"),
            "{error}"
        );
    }
    Ok(())
}
