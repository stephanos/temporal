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

fn pure_io_members(members: &[&str]) -> Result<(), String> {
    if members.first() != Some(&"::") {
        return Err("modeled state requires named std::io types".into());
    }
    let allowed = |name: &str| {
        matches!(
            name,
            "BufRead"
                | "BufReader"
                | "BufWriter"
                | "Cursor"
                | "Error"
                | "ErrorKind"
                | "Read"
                | "Result"
                | "Seek"
                | "SeekFrom"
                | "Write"
        )
    };
    if members.get(1) == Some(&"{") {
        let mut member = 2;
        loop {
            let name = members
                .get(member)
                .ok_or("incomplete std::io import group")?;
            if *name == "}" {
                break;
            }
            if !allowed(name) {
                return Err(format!("modeled state imports host std::io::{name}"));
            }
            member += 1;
            if members.get(member) == Some(&"as") {
                member += 2;
            }
            match members.get(member) {
                Some(&",") => member += 1,
                Some(&"}") => break,
                _ => return Err("unrecognized std::io import group".into()),
            }
        }
    } else if !members.get(1).is_some_and(|name| allowed(name)) {
        return Err("modeled state requires named std::io types".into());
    }
    Ok(())
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
        if *token == "last_os_error" {
            return Err("modeled state reads the host OS error".into());
        }
        if matches!(*token, "rustix" | "libc") {
            return Err(format!("modeled state imports host {token}"));
        }
        if *token != "std" {
            continue;
        }
        if tokens.get(index + 1) == Some(&"::") {
            let module = *tokens.get(index + 2).ok_or("incomplete std module path")?;
            if module == "io" {
                pure_io_members(&tokens[index + 3..])?;
            }
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

#[test]
fn modeled_state_io_admits_named_types_and_refuses_host_streams() {
    for source in [
        "fn read() { std::io::stdin(); }",
        "fn write() { std :: io :: stdout(); }",
        "use std::io::{Read, stdin as input};",
        "use std::io::{self, Write};",
        "use std::io as host;",
        "use std::io::*;",
        "fn write() { std::io::stderr(); }",
        "fn read() { std::io::Error::last_os_error(); }",
        "use std::io::Error as HostError; fn read() { HostError::last_os_error(); }",
    ] {
        assert!(pure_imports(source).is_err(), "{source}");
    }
    for source in [
        "use std::io::{Read, Write, Error, ErrorKind, Result};",
        "use std::io::Read as Input;",
        "fn decode() { std::io::Cursor::new(Vec::new()); }",
    ] {
        pure_imports(source).unwrap();
    }
}
