use std::{
    fs,
    path::PathBuf,
    sync::atomic::{AtomicU64, Ordering},
};
use umpire_guest_driver::support::{SidecarLease, canonical, hash, publish};

struct Scratch(PathBuf);
impl Scratch {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let p = std::env::temp_dir().join(format!(
            "xtask-publication-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir(&p).unwrap();
        Self(p)
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.0).unwrap();
    }
}

#[test]
fn canonical_hash_uses_sorted_json_and_known_sha256() {
    assert_eq!(
        canonical(&serde_json::json!({"z": {"b":2,"a":1},"a":3})).unwrap(),
        br#"{"a":3,"z":{"a":1,"b":2}}"#
    );
    assert_eq!(hash(b"abc"), "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad");
}

#[test]
fn publication_is_complete_under_concurrent_writers_and_cleans_rename_failure() {
    let scratch = Scratch::new();
    let p = scratch.0.join("record");
    let a = vec![b'a'; 65536];
    let b = vec![b'b'; 65536];
    publish(&p, &a).unwrap();
    std::thread::scope(|scope| {
        for bytes in [&a, &b] {
            let p = &p;
            scope.spawn(move || {
                for _ in 0..10 {
                    publish(p, bytes).unwrap();
                }
            });
        }
        for _ in 0..50 {
            let actual = fs::read(&p).unwrap();
            assert!(actual == a || actual == b);
        }
    });
    let refused = scratch.0.join("directory");
    fs::create_dir(&refused).unwrap();
    assert!(publish(&refused, b"bad").is_err());
    assert!(
        !fs::read_dir(&scratch.0).unwrap().any(|e| e
            .unwrap()
            .file_name()
            .as_encoded_bytes()
            .ends_with(b".tmp"))
    );
}

#[test]
fn sidecar_lease_serializes_failure_cleanup_and_refuses_symlink() {
    let scratch = Scratch::new();
    let p = scratch.0.join("record");
    let first = SidecarLease::acquire(&p).unwrap();
    publish(&p, b"old").unwrap();
    let (tx, rx) = std::sync::mpsc::channel();
    std::thread::scope(|scope| {
        let p = &p;
        scope.spawn(move || {
            tx.send("waiting").unwrap();
            let _second = SidecarLease::acquire(p).unwrap();
            publish(p, b"new").unwrap();
            tx.send("published").unwrap();
        });
        assert_eq!(rx.recv().unwrap(), "waiting");
        first.remove().unwrap();
        drop(first);
        assert_eq!(rx.recv().unwrap(), "published");
    });
    assert_eq!(fs::read(&p).unwrap(), b"new");
    #[cfg(unix)]
    {
        let fake = scratch.0.join("fake");
        std::os::unix::fs::symlink(scratch.0.join("record.lock"), scratch.0.join("fake.lock"))
            .unwrap();
        assert!(SidecarLease::acquire(&fake).is_err());
    }
}
