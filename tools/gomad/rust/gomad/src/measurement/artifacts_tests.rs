use super::*;
use std::os::unix::fs::DirBuilderExt;

struct Scratch(std::path::PathBuf);
impl Scratch {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-variant-custody-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&root)
            .unwrap();
        Self(root)
    }
    fn native(&self) -> std::path::PathBuf {
        let module = self.0.join("module.wat");
        std::fs::write(
            &module,
            b"(module (memory (export \"memory\") 1) (func (export \"_start\")))",
        )
        .unwrap();
        let input = self.0.join("input.json");
        std::fs::write(&input,serde_json::to_vec(&json!({"kind":"standalone","guest":{
            "module":module,"module_sha256":canonical::digest(&std::fs::read(module).unwrap()),
            "limits":{"memory_bytes":65536,"fuel":500_000_000,"wall_millis":5000,"output_bytes":1024,"evidence_bytes":1<<20},
            "model":null,"namespace":null,"storage":null},"network":null})).unwrap()).unwrap();
        let output = self.0.join("native");
        super::super::execution::run(&input, &output).unwrap();
        output
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}
fn read(path: &Path) -> Value {
    serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
}

#[test]
fn real_native_payload_mutation_cannot_become_available_canonical_evidence() {
    let scratch = Scratch::new();
    let output = scratch.native();
    let prior = read(&output.join("variant-artifacts.json"));
    assert_eq!(prior["artifacts"]["guest-stdout"]["status"], "available");
    std::fs::remove_file(output.join("variant-artifacts.json")).unwrap();
    std::fs::write(output.join("stdout"), b"forged after finalization").unwrap();
    let error = retain_execution(&output, Some("original collection failure")).unwrap_err();
    assert!(
        error.contains("size or digest mismatch") || error.contains("bound"),
        "{error}"
    );
    let catalog = read(&output.join("variant-artifacts.json"));
    assert_eq!(catalog["failure"], "original collection failure");
    for (meaning, _, _) in MEANINGS {
        assert_eq!(catalog["artifacts"][meaning]["status"], "unavailable");
        assert!(catalog["artifacts"][meaning].get("sha256").is_none());
    }
    assert_eq!(
        std::fs::read(output.join("stdout")).unwrap(),
        b"forged after finalization"
    );
}
