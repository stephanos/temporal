//! Replays the Go-generated clock conformance corpus
//! (`internal/backends/wasm/cluster/clock_conformance_test.go`).
use serde_json::Value;
use std::path::PathBuf;

fn corpora() -> Vec<(String, Value)> {
    let directory =
        PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/clock_conformance");
    let pin = std::fs::read_to_string(directory.join("PINNED_GO_CLOCK_SHA256")).unwrap();
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
            assert_eq!(value["schema"], "gomad3.clock-conformance/v1", "{name}");
            assert_eq!(value["name"], name.as_str());
            assert_eq!(value["clock_implementation_sha256"], pin.trim(), "{name}");
            (name, value)
        })
        .collect()
}

#[test]
fn clock_corpus_headers_match_the_go_pin() {
    let corpora = corpora();
    for source in ["scripted", "generated", "recorded"] {
        assert!(
            corpora.iter().any(|(_, corpus)| corpus["source"] == source),
            "missing {source} clock corpus"
        );
    }
}

#[test]
#[ignore = "fn-178.5: the Rust simulation clock is not ported yet"]
fn clock_corpus_replays_exactly() {
    unimplemented!("fn-178.5")
}
