//! Replays the Go-generated World conformance corpus
//! (`internal/backends/wasm/cluster/world_conformance_test.go`).
use serde_json::Value;
use std::path::PathBuf;

fn corpora() -> Vec<(String, Value)> {
    let directory =
        PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/world_conformance");
    let pin = std::fs::read_to_string(directory.join("PINNED_GO_WORLD_SHA256")).unwrap();
    let mut files: Vec<_> = std::fs::read_dir(&directory)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .filter(|path| path.extension().is_some_and(|extension| extension == "json"))
        .collect();
    files.sort();
    files
        .into_iter()
        .map(|path| {
            let name = path.file_stem().unwrap().to_str().unwrap().to_owned();
            let value: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
            assert_eq!(value["schema"], "gomad3.world-conformance/v1", "{name}");
            assert_eq!(value["name"], name.as_str());
            assert_eq!(value["world_implementation_sha256"], pin.trim(), "{name}");
            (name, value)
        })
        .collect()
}

#[test]
fn world_corpus_headers_match_the_go_pin() {
    let corpora = corpora();
    for source in ["scripted", "generated", "recorded"] {
        assert!(
            corpora.iter().any(|(_, corpus)| corpus["source"] == source),
            "missing {source} World corpus"
        );
    }
}

#[test]
#[ignore = "fn-178.3: the Rust World model is not ported yet"]
fn world_corpus_replays_exactly() {
    unimplemented!("fn-178.3")
}
