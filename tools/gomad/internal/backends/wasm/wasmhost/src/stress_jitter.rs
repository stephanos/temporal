//! Host-only perturbation, excluded entirely from the release helper.
use std::io::Read;
use std::sync::OnceLock;

pub fn delay() {
    static ENABLED: OnceLock<bool> = OnceLock::new();
    if !*ENABLED.get_or_init(|| std::env::var_os("GOMAD_WASM_STRESS_JITTER").is_some()) {
        return;
    }
    let mut bytes = [0; 2];
    std::fs::File::open("/dev/urandom")
        .and_then(|mut file| file.read_exact(&mut bytes))
        .expect("stress jitter requires host entropy");
    std::thread::sleep(std::time::Duration::from_micros(u64::from(
        u16::from_ne_bytes(bytes) % 1001,
    )));
}

#[cfg(test)]
mod tests {
    #[test]
    fn stress_jitter_is_a_distinct_engine_identity() {
        assert!(crate::engine::CONFIGURATION_REVISION.ends_with("-stress-jitter"));
    }
}
