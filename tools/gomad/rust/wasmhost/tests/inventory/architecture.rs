use super::*;
use std::{fs, path::Path};

// These are modeled-state owners. Transport is a named seam, never an
// exemption for a filename merely containing "process" or "measurement".
fn pure_model_file(file: &str) -> bool {
    matches!(
        file,
        "abi.rs" | "clock.rs" | "local_model.rs" | "model.rs" | "transcript.rs"
    ) || ["model/", "namespace/", "network/", "volume/", "world/"]
        .iter()
        .any(|root| file.starts_with(root))
        && file != "namespace/transport.rs"
}

fn pure_imports(source: &str) -> Result<(), String> {
    let (lines, _) = production_lines(source)?;
    let source = lines
        .iter()
        .map(|(_, line)| strip_literals(line))
        .collect::<Vec<_>>()
        .join("\n");
    let mut tokens = Vec::new();
    let bytes = source.as_bytes();
    let mut index = 0;
    while index < bytes.len() {
        let start = index;
        if bytes[index].is_ascii_alphanumeric() || bytes[index] == b'_' {
            while index < bytes.len()
                && (bytes[index].is_ascii_alphanumeric() || bytes[index] == b'_')
            {
                index += 1;
            }
        } else if bytes[index..].starts_with(b"::") {
            index += 2;
        } else {
            index += source[index..].chars().next().unwrap().len_utf8();
        }
        let token = &source[start..index];
        if !token.chars().all(char::is_whitespace) {
            tokens.push(token);
        }
    }
    for (index, token) in tokens.iter().enumerate() {
        if matches!(*token, "rustix" | "libc") {
            return Err(format!("modeled state imports host {token}"));
        }
        if *token != "std" {
            continue;
        }
        if tokens.get(index + 1) == Some(&"::") {
            let module = *tokens.get(index + 2).ok_or("incomplete std module path")?;
            if !matches!(
                module,
                "array"
                    | "cell"
                    | "cmp"
                    | "collections"
                    | "convert"
                    | "error"
                    | "fmt"
                    | "hash"
                    | "hint"
                    | "io"
                    | "iter"
                    | "marker"
                    | "mem"
                    | "num"
                    | "ops"
                    | "option"
                    | "result"
                    | "slice"
                    | "str"
                    | "string"
                    | "sync"
                    | "vec"
            ) {
                return Err(format!("modeled state imports host std::{module}"));
            }
        } else if index > 0 && matches!(tokens[index - 1], "use" | "crate") {
            return Err("modeled state requires explicit std module paths".into());
        }
    }
    Ok(())
}

#[test]
fn native_modeled_state_rejects_host_imports_on_both_source_platforms() {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    let (_, files) = source_scan(&root, &["lib.rs"]).unwrap();
    // Conservative source checks inspect all reachable cfg branches for both
    // supported platforms. They do not claim a cross-compiled executable ran.
    for platform in ["darwin/arm64", "linux/amd64"] {
        let mut checked = 0;
        for file in &files {
            if pure_model_file(file) {
                checked += 1;
                pure_imports(&fs::read_to_string(root.join(file)).unwrap())
                    .unwrap_or_else(|error| panic!("{platform} {file}: {error}"));
            }
        }
        assert!(
            checked >= 20,
            "{platform}: modeled-state owners disappeared"
        );
        for name in [
            "clock.rs",
            "debug_output.rs",
            "measurement.rs",
            "process.rs",
            "cluster.rs",
            "moved_process.rs",
        ] {
            assert!(pure_model_file(&format!("namespace/{name}")), "{name}");
            for source in [
                "use std::env; fn read() { env::var(\"HOST\"); }",
                "use std::{collections::BTreeMap, fs};",
                "fn read() { std :: time :: Instant::now(); }",
                "use rustix::fs as host;",
                "extern crate std as host;",
                "extern crate libc;",
                "use std::*;",
            ] {
                assert!(pure_imports(source).is_err(), "{platform} {name}: {source}");
            }
        }
        pure_imports("struct Descriptor { std: u32 } let values = std::array::from_fn::<_, 2, _>(|i| i); use std::collections::BTreeMap; // std::env is documentation\n#[cfg(test)]\nmod tests {\n    use std::fs;\n}\n").unwrap();
    }
    assert!(!pure_model_file("namespace/transport.rs"));
}
