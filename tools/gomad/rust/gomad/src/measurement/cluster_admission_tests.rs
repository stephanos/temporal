use super::*;
use std::cell::Cell;

struct Temporary(PathBuf);
impl Temporary {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-cluster-admission-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir(&root).unwrap();
        Self(root)
    }
}
impl Drop for Temporary {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

#[test]
fn original_four_phases_by_four_drifts_stop_dispatch_or_observation_at_boundary() {
    for replay in [false, true] {
        for incarnation in [1, 2] {
            let phase = LifecyclePhase {
                incarnation,
                handles: ["active", "standby"]
                    .map(|node| Handle {
                        node: node.into(),
                        incarnation,
                    })
                    .into(),
            };
            let label = phase_label(&phase, replay).unwrap();
            assert_eq!(
                label,
                match (incarnation, replay) {
                    (1, false) => "cluster-crash",
                    (2, false) => "cluster-reopened",
                    (1, true) => "cluster-prefix-replay",
                    _ => "cluster-full-replay",
                }
            );
            for drift in ["none", "before", "after", "resource"] {
                let directory = Temporary::new();
                let input = directory.0.join("module");
                let memory = directory.0.join("meminfo");
                std::fs::write(&input, b"admitted").unwrap();
                std::fs::write(&memory, b"MemAvailable: 8388608 kB\n").unwrap();
                let pins = BTreeMap::from([(input.clone(), canonical::digest(b"admitted"))]);
                if drift == "before" {
                    std::fs::write(&input, b"changed").unwrap();
                }
                if drift == "resource" {
                    std::fs::write(&memory, b"MemAvailable: 8388607 kB\n").unwrap();
                }
                let dispatched = Cell::new(false);
                let observed = Cell::new(false);
                let verify = || module::verify_inputs(&pins);
                let result = boundary(
                    true,
                    verify,
                    || {
                        let raw =
                            std::fs::read_to_string(&memory).map_err(|error| error.to_string())?;
                        let available = host::linux_available_memory(&raw)?;
                        if available < 8_u64 << 30 {
                            Err("at least 8GiB MemAvailable".into())
                        } else {
                            Ok(())
                        }
                    },
                    || Ok(()),
                )
                .and_then(|()| {
                    dispatched.set(true);
                    if drift == "after" {
                        std::fs::write(&input, b"changed").unwrap();
                    }
                    boundary(
                        false,
                        verify,
                        || panic!("post-boundary must not re-admit resource"),
                        || {
                            observed.set(true);
                            Ok(())
                        },
                    )
                });
                assert_eq!(result.is_ok(), drift == "none", "{label}/{drift}");
                assert_eq!(
                    dispatched.get(),
                    matches!(drift, "none" | "after"),
                    "{label}/{drift}"
                );
                assert_eq!(observed.get(), drift == "none", "{label}/{drift}");
                if let Err(error) = result {
                    assert!(
                        error.contains(if drift == "resource" {
                            "8GiB"
                        } else {
                            "changed"
                        }),
                        "{error}"
                    );
                }
            }
        }
    }
}
