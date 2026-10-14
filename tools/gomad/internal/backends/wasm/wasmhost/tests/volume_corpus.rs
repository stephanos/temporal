#![cfg(feature = "conformance")]

use base64::{Engine, engine::general_purpose::STANDARD};
use serde_json::Value;
use std::fs;
use std::io::Write;
use std::path::Path;
use std::process::{Command, Stdio};

#[test]
#[ignore = "Rust volume owner awaits fn-176.2; unsupported is not a differential pass"]
fn volume_corpus_matches_go_reference() {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../wasi/testdata/volume-conformance");
    let mut paths: Vec<_> = fs::read_dir(root)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .filter(|path| {
            path.extension()
                .is_some_and(|extension| extension == "json")
        })
        .collect();
    paths.sort();
    assert!(!paths.is_empty(), "Go volume corpus is missing");
    for path in paths {
        let corpus: Value = serde_json::from_slice(&fs::read(&path).unwrap()).unwrap();
        let seed = &corpus["seed"];
        let mut child = Command::new(env!("CARGO_BIN_EXE_gomad3-wasmhost"))
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .unwrap();
        let mut input = child.stdin.take().unwrap();
        writeln!(input, "{}", corpus["session"]).unwrap();
        for operation in corpus["operations"]
            .as_array()
            .map(Vec::as_slice)
            .unwrap_or(&[])
        {
            writeln!(input, "{operation}").unwrap();
        }
        drop(input);
        let output = child.wait_with_output().unwrap();
        assert!(
            output.status.success(),
            "{:?}: {}",
            path,
            String::from_utf8_lossy(&output.stderr)
        );
        let replies: Vec<Value> = output
            .stdout
            .split(|byte| *byte == b'\n')
            .filter(|line| !line.is_empty())
            .map(|line| serde_json::from_slice(line).unwrap())
            .collect();
        assert!(!replies.is_empty(), "{:?}: no session reply", path);
        assert_ne!(
            replies[0]["unsupported"], true,
            "{:?}: Rust volume owner awaits fn-176.2",
            path
        );
        let initial = STANDARD
            .decode(corpus["initial_reply_base64"].as_str().unwrap())
            .unwrap();
        assert_eq!(
            replies[0],
            serde_json::from_slice::<Value>(&initial).unwrap(),
            "{:?} seed={seed}: session divergence",
            path
        );
        let observations = corpus["observations"]
            .as_array()
            .map(Vec::as_slice)
            .unwrap_or(&[]);
        assert_eq!(
            replies.len(),
            observations.len() + 1,
            "{:?} seed={seed}: reply count",
            path
        );
        for (step, expected) in observations.iter().enumerate() {
            assert_eq!(
                &replies[step + 1],
                expected,
                "{:?} seed={seed} first divergence at step={step} op={}",
                path,
                corpus["operations"][step]["op"]
            );
        }
    }
}
