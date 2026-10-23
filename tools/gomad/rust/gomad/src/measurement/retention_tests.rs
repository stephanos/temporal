use super::*;
use std::os::unix::fs::MetadataExt;
use std::os::unix::fs::symlink;

struct Scratch(PathBuf);
impl Scratch {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!("gomad-measure-retention-{}", unique()));
        std::fs::create_dir(&path).unwrap();
        Self(std::fs::canonicalize(path).unwrap())
    }
    fn begin(&self, producer: &str, keep: bool) -> RetainedRun {
        RetainedRun::begin(&self.0, producer, &BTreeMap::new(), keep, &|| false).unwrap()
    }
    fn latest(&self, producer: &str) -> PathBuf {
        let store = self.0.join(".gomad-measure").join(producer);
        let name: String =
            serde_json::from_slice(&std::fs::read(store.join("latest.json")).unwrap()).unwrap();
        store.join(name)
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}

#[test]
fn success_publishes_and_superseded_success_is_collected() {
    let scratch = Scratch::new();
    let first = scratch.begin("prototype", false);
    let first_path = first.path().to_path_buf();
    std::fs::write(first.path().join("result.json"), b"result").unwrap();
    first.finish(true, &|| false).unwrap();
    assert_eq!(scratch.latest("prototype"), first_path);
    let second = scratch.begin("prototype", false);
    let second_path = second.path().to_path_buf();
    second.finish(true, &|| false).unwrap();
    assert_eq!(scratch.latest("prototype"), second_path);
    assert!(!first_path.exists());
    assert!(second_path.join(".complete").is_file());
}

#[test]
fn failed_and_cancelled_runs_do_not_advance_latest() {
    let scratch = Scratch::new();
    let good = scratch.begin("prototype", false);
    let good_path = good.path().to_path_buf();
    good.finish(true, &|| false).unwrap();
    let failed = scratch.begin("prototype", false);
    failed.finish(false, &|| false).unwrap();
    assert_eq!(scratch.latest("prototype"), good_path);
    let cancelled = scratch.begin("prototype", false);
    let cancelled_path = cancelled.path().to_path_buf();
    assert!(
        cancelled
            .finish(true, &|| true)
            .unwrap_err()
            .contains("cancelled")
    );
    assert_eq!(scratch.latest("prototype"), good_path);
    assert!(!cancelled_path.join(".complete").exists());
}

#[test]
fn overlapping_active_and_explicit_keep_runs_survive_collection() {
    let scratch = Scratch::new();
    let active = scratch.begin("prototype", false);
    let active_path = active.path().to_path_buf();
    let kept = scratch.begin("prototype", true);
    let kept_path = kept.path().to_path_buf();
    kept.finish(true, &|| false).unwrap();
    scratch
        .begin("prototype", false)
        .finish(true, &|| false)
        .unwrap();
    assert!(active_path.exists());
    assert!(kept_path.exists());
    active.finish(true, &|| false).unwrap();
    assert_eq!(scratch.latest("prototype"), active_path);
    scratch
        .begin("prototype", false)
        .finish(true, &|| false)
        .unwrap();
    assert!(!active_path.exists());
    assert!(kept_path.exists());
}

#[test]
fn only_newest_unpinned_failed_run_survives() {
    let scratch = Scratch::new();
    let first = scratch.begin("prototype", false);
    let first_path = first.path().to_path_buf();
    first.finish(false, &|| false).unwrap();
    let second = scratch.begin("prototype", false);
    let second_path = second.path().to_path_buf();
    // Force a strict ordering without relying on filesystem timestamp resolution.
    second
        .root
        .set_times(
            std::fs::FileTimes::new().set_modified(UNIX_EPOCH + Duration::from_secs(2_000_000_000)),
        )
        .unwrap();
    second.finish(false, &|| false).unwrap();
    assert!(!first_path.exists());
    assert!(second_path.exists());
}

#[test]
fn pins_hold_input_identity_and_release_dependencies_transitively() {
    let scratch = Scratch::new();
    let input = scratch.begin("prototype", false);
    let input_path = input.path().to_path_buf();
    input.finish(true, &|| false).unwrap();
    let middle = RetainedRun::begin(
        &scratch.0,
        "namespace",
        &BTreeMap::from([(
            "SOURCE".into(),
            input_path.join("index.json").to_str().unwrap().into(),
        )]),
        false,
        &|| false,
    )
    .unwrap();
    let middle_path = middle.path().to_path_buf();
    middle.finish(true, &|| false).unwrap();
    let consumer = RetainedRun::begin(
        &scratch.0,
        "matched",
        &BTreeMap::from([("SOURCE".into(), middle_path.to_str().unwrap().into())]),
        false,
        &|| false,
    )
    .unwrap();
    consumer.finish(true, &|| false).unwrap();
    scratch
        .begin("prototype", false)
        .finish(true, &|| false)
        .unwrap();
    scratch
        .begin("namespace", false)
        .finish(true, &|| false)
        .unwrap();
    assert!(input_path.exists());
    assert!(middle_path.exists());
    let input_pin = std::fs::metadata(input_path.join(".pin")).unwrap();
    assert_eq!(input_pin.nlink(), 2);
    scratch
        .begin("matched", false)
        .finish(true, &|| false)
        .unwrap();
    assert!(!middle_path.exists());
    assert!(!input_path.exists());
}

#[test]
fn active_input_is_refused_and_identity_tampering_is_refused() {
    let scratch = Scratch::new();
    let input = scratch.begin("prototype", false);
    let input_path = input.path().to_path_buf();
    let inputs = BTreeMap::from([("SOURCE".into(), input_path.to_str().unwrap().into())]);
    assert!(RetainedRun::begin(&scratch.0, "namespace", &inputs, false, &|| false).is_err());
    input.finish(true, &|| false).unwrap();
    std::fs::write(input_path.join(".pin"), b"substituted").unwrap();
    assert!(RetainedRun::begin(&scratch.0, "namespace", &inputs, false, &|| false).is_err());
    scratch
        .begin("prototype", false)
        .finish(true, &|| false)
        .unwrap();
    assert!(input_path.exists());
}

#[test]
fn nested_store_roots_symlinks_and_unclean_roots_are_refused() {
    let scratch = Scratch::new();
    let active = scratch.begin("prototype", false);
    for path in [
        PathBuf::from("/"),
        scratch.0.join("a/../b"),
        active.path().join("nested"),
        scratch.0.join(".gomad-measure/other"),
    ] {
        assert!(
            RetainedRun::begin(&path, "prototype", &BTreeMap::new(), false, &|| false).is_err()
        );
    }
    let link = scratch.0.join("link");
    symlink(&scratch.0, &link).unwrap();
    assert!(RetainedRun::begin(&link, "prototype", &BTreeMap::new(), false, &|| false).is_err());
    let aliased = RetainedRun::begin(
        &link.join("nested"),
        "prototype",
        &BTreeMap::new(),
        false,
        &|| false,
    )
    .unwrap();
    assert!(aliased.path().starts_with(scratch.0.join("nested")));
    aliased.finish(true, &|| false).unwrap();
    let retained_alias = scratch.0.join("retained-alias");
    symlink(active.path(), &retained_alias).unwrap();
    assert!(
        RetainedRun::begin(
            &retained_alias.join("child"),
            "prototype",
            &BTreeMap::new(),
            false,
            &|| false
        )
        .is_err()
    );
    assert!(!active.path().join("child").exists());
    for producer in ["", "..", "../outside", "a/b"] {
        assert!(
            RetainedRun::begin(&scratch.0, producer, &BTreeMap::new(), false, &|| false).is_err()
        );
    }
}

#[test]
fn abandoned_stages_trash_and_atomic_write_stages_are_recovered() {
    let scratch = Scratch::new();
    let run = scratch.begin("prototype", false);
    let store = run.path().parent().unwrap().to_path_buf();
    for name in [".stage-run-abandoned", ".trash-abandoned"] {
        std::fs::create_dir(store.join(name)).unwrap();
        std::fs::write(store.join(name).join("partial"), b"partial").unwrap();
    }
    std::fs::write(store.join(".stage-abandoned"), b"partial").unwrap();
    run.finish(true, &|| false).unwrap();
    for entry in std::fs::read_dir(store).unwrap() {
        let name = entry.unwrap().file_name();
        assert!(!name.to_str().unwrap().starts_with(".stage-"));
        assert!(!name.to_str().unwrap().starts_with(".trash-"));
    }
}

#[test]
fn malformed_latest_blocks_publication_without_collecting_existing_runs() {
    let scratch = Scratch::new();
    let run = scratch.begin("prototype", false);
    let path = run.path().to_path_buf();
    std::fs::write(path.parent().unwrap().join("latest.json"), b"\"../escape\"").unwrap();
    assert!(run.finish(true, &|| false).is_err());
    assert!(path.exists());
    assert!(!path.join(".complete").exists());
}

#[test]
fn cancelled_store_lock_wait_returns_while_lock_remains_owned() {
    let scratch = Scratch::new();
    let run = scratch.begin("prototype", false);
    let lock = store_lock(&run.store, &|| false).unwrap();
    let calls = std::cell::Cell::new(0);
    let result = store_lock(&run.store, &|| {
        calls.set(calls.get() + 1);
        calls.get() > 1
    });
    assert!(result.unwrap_err().contains("cancelled"));
    drop(lock);
}

#[test]
fn symlink_outputs_are_not_followed_and_special_outputs_refuse_publication() {
    let scratch = Scratch::new();
    let outside = scratch.0.join("outside");
    std::fs::write(&outside, b"untouched").unwrap();
    let run = scratch.begin("prototype", false);
    symlink(&outside, run.path().join("link")).unwrap();
    run.finish(true, &|| false).unwrap();
    assert_eq!(std::fs::read(outside).unwrap(), b"untouched");
    let run = scratch.begin("prototype", false);
    rustix::fs::mkfifoat(
        rustix::fs::CWD,
        run.path().join("fifo"),
        rustix::fs::Mode::RUSR | rustix::fs::Mode::WUSR,
    )
    .unwrap();
    assert!(
        run.finish(true, &|| false)
            .unwrap_err()
            .contains("nonregular")
    );
}

#[test]
fn cancellation_after_complete_marker_rolls_back_publication() {
    let scratch = Scratch::new();
    let good = scratch.begin("prototype", false);
    let good_path = good.path().to_path_buf();
    good.finish(true, &|| false).unwrap();
    let run = scratch.begin("prototype", false);
    let path = run.path().to_path_buf();
    assert!(
        run.finish(true, &|| path.join(".complete").exists())
            .is_err()
    );
    assert_eq!(scratch.latest("prototype"), good_path);
    assert!(!path.join(".complete").exists());
}

#[test]
fn symlink_store_and_identity_files_are_refused() {
    let scratch = Scratch::new();
    let outside = Scratch::new();
    symlink(&outside.0, scratch.0.join(".gomad-measure")).unwrap();
    assert!(
        RetainedRun::begin(&scratch.0, "prototype", &BTreeMap::new(), false, &|| false).is_err()
    );
    std::fs::remove_file(scratch.0.join(".gomad-measure")).unwrap();
    let run = scratch.begin("prototype", false);
    let path = run.path().to_path_buf();
    run.finish(true, &|| false).unwrap();
    let identity = outside.0.join("identity");
    std::fs::write(&identity, path.as_os_str().as_encoded_bytes()).unwrap();
    std::fs::remove_file(path.join(".pin")).unwrap();
    symlink(identity, path.join(".pin")).unwrap();
    let inputs = BTreeMap::from([("SOURCE".into(), path.to_str().unwrap().into())]);
    assert!(RetainedRun::begin(&scratch.0, "namespace", &inputs, false, &|| false).is_err());
}

#[test]
fn dropped_run_lease_is_recovered_as_failure() {
    let scratch = Scratch::new();
    let abandoned = scratch.begin("prototype", false);
    let abandoned_path = abandoned.path().to_path_buf();
    drop(abandoned);
    scratch
        .begin("prototype", false)
        .finish(true, &|| false)
        .unwrap();
    assert!(abandoned_path.exists());
    let newer = scratch.begin("prototype", false);
    newer
        .root
        .set_times(
            std::fs::FileTimes::new().set_modified(UNIX_EPOCH + Duration::from_secs(2_000_000_000)),
        )
        .unwrap();
    newer.finish(false, &|| false).unwrap();
    assert!(!abandoned_path.exists());
}

#[test]
fn shared_input_lease_prevents_collection_until_released() {
    let scratch = Scratch::new();
    let input = scratch.begin("prototype", false);
    let path = input.path().to_path_buf();
    input.finish(true, &|| false).unwrap();
    let root = directory(&path, false).unwrap();
    let lease = open(&root, OsStr::new(".lease"), false).unwrap();
    fs::flock(&lease, FlockOperation::NonBlockingLockShared).unwrap();
    scratch
        .begin("prototype", false)
        .finish(true, &|| false)
        .unwrap();
    assert!(path.exists());
    drop(lease);
    reclaim(path.parent().unwrap()).unwrap();
    assert!(!path.exists());
}

#[test]
fn child_lease_outlives_parent_and_defers_reclamation_without_reopening_inputs() {
    let scratch = Scratch::new();
    let parent = scratch.begin("stress", false);
    let path = parent.path().to_owned();
    let child = ChildLease::for_path(&path.join("child-evidence"))
        .unwrap()
        .unwrap();
    parent.finish(true, &|| false).unwrap();
    let newer = scratch.begin("stress", false);
    newer.finish(true, &|| false).unwrap();
    assert!(path.exists());
    drop(child);
    let newer = scratch.begin("stress", false);
    newer.finish(true, &|| false).unwrap();
    assert!(!path.exists());
    assert!(ChildLease::for_path(&scratch.0).unwrap().is_none());
}

#[test]
fn hardlinked_publication_pointer_preserves_previous_result_and_recovers() {
    let scratch = Scratch::new();
    let produce = || -> Result<(), String> {
        let run = RetainedRun::begin(&scratch.0, "prototype", &BTreeMap::new(), false, &|| false)?;
        std::fs::write(run.path().join("result"), b"complete").map_err(|e| e.to_string())?;
        run.finish(true, &|| false)
    };
    produce().unwrap();
    let previous = scratch.latest("prototype");
    let pointer = previous.parent().unwrap().join("latest.json");
    let linked = scratch.0.join("linked-pointer");
    std::fs::hard_link(&pointer, &linked).unwrap();
    assert!(
        produce().is_err(),
        "hardlinked publication pointer was replaced"
    );
    assert_eq!(scratch.latest("prototype"), previous);
    assert_eq!(std::fs::read(previous.join("result")).unwrap(), b"complete");
    std::fs::remove_file(linked).unwrap();
    produce().unwrap();
    assert_ne!(scratch.latest("prototype"), previous);
}
