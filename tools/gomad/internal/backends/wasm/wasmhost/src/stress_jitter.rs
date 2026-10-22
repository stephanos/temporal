//! Host-only perturbation, excluded entirely from the release helper.
use std::io::Read;
use std::sync::{
    OnceLock,
    atomic::{AtomicU64, Ordering},
};

static CALLS: AtomicU64 = AtomicU64::new(0);
static MICROS: AtomicU64 = AtomicU64::new(0);

#[must_use]
pub fn statistics() -> (u64, u64) {
    (
        CALLS.load(Ordering::Relaxed),
        MICROS.load(Ordering::Relaxed),
    )
}

fn micros(bytes: [u8; 2]) -> u64 {
    u64::from(u16::from_ne_bytes(bytes) % 1001)
}

pub fn delay() {
    static ENABLED: OnceLock<bool> = OnceLock::new();
    if !*ENABLED
        .get_or_init(|| std::env::var("GOMAD_WASM_STRESS_JITTER").is_ok_and(|value| value == "1"))
    {
        return;
    }
    let mut bytes = [0; 2];
    std::fs::File::open("/dev/urandom")
        .and_then(|mut file| file.read_exact(&mut bytes))
        .expect("stress jitter requires host entropy");
    let delay = micros(bytes);
    CALLS.fetch_add(1, Ordering::Relaxed);
    MICROS.fetch_add(delay, Ordering::Relaxed);
    std::thread::sleep(std::time::Duration::from_micros(delay));
}

#[cfg(test)]
mod tests {
    #[test]
    fn stress_jitter_delay_is_bounded() {
        for value in 0..=u16::MAX {
            assert!(super::micros(value.to_ne_bytes()) <= 1000);
        }
    }

    #[test]
    fn stress_jitter_is_a_distinct_engine_identity() {
        assert!(crate::engine::CONFIGURATION_REVISION.ends_with("-stress-jitter"));
    }
}

#[cfg(test)]
mod linked_tests {
    #[test]
    fn stress_jitter_runs_inside_the_linked_role() {
        if std::env::var("GOMAD_STRESS_JITTER_TEST_CHILD").as_deref() != Ok("1") {
            let output = std::process::Command::new(std::env::current_exe().unwrap())
                .args([
                    "--exact",
                    "stress_jitter::linked_tests::stress_jitter_runs_inside_the_linked_role",
                ])
                .env("GOMAD_WASM_STRESS_JITTER", "1")
                .env("GOMAD_STRESS_JITTER_TEST_CHILD", "1")
                .output()
                .unwrap();
            assert!(
                output.status.success(),
                "{} {}",
                String::from_utf8_lossy(&output.stdout),
                String::from_utf8_lossy(&output.stderr)
            );
            return;
        }
        let _scope = crate::linked_conditions::Conditions::compiled()
            .enter()
            .unwrap();
        let before = super::statistics();
        super::delay();
        let after = super::statistics();
        assert_eq!(after.0, before.0 + 1);
        assert!(after.1 >= before.1 && after.1 - before.1 <= 1000);
    }
}
