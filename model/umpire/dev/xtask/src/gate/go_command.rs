//! Admission and JSON transformations of the existing Go test leaves.
use std::{
    env,
    ffi::{OsStr, OsString},
    io,
    os::unix::ffi::{OsStrExt, OsStringExt},
    path::Path,
    process::Command,
};

pub(super) fn is_go_test(command: &Command) -> bool {
    let arguments: Vec<_> = command.get_args().take(6).collect();
    Path::new(command.get_program()).file_name() == Some(OsStr::new("mise"))
        && arguments.len() == 6
        && arguments[0] == "exec"
        && arguments[1] == "--"
        && arguments[2] == "go"
        && arguments[3] == "-C"
        && arguments[5] == "test"
}

fn invalid(message: &str) -> io::Error {
    io::Error::other(message)
}

fn arguments(command: &Command) -> io::Result<Vec<OsString>> {
    if !is_go_test(command) {
        return Err(invalid("not a canonical Go stage command"));
    }
    let arguments: Vec<_> = command.get_args().map(ToOwned::to_owned).collect();
    if arguments[4].is_empty() || arguments[4].as_bytes().contains(&0) {
        return Err(invalid("invalid Go stage module"));
    }
    Ok(arguments)
}

fn copy_context(command: &Command) -> Command {
    let mut copied = Command::new(command.get_program());
    if let Some(directory) = command.get_current_dir() {
        copied.current_dir(directory);
    }
    for (name, value) in command.get_envs() {
        if let Some(value) = value {
            copied.env(name, value);
        } else {
            copied.env_remove(name);
        }
    }
    copied
}

pub(super) fn environment(command: &Command) -> io::Result<Command> {
    let arguments = arguments(command)?;
    let mut query = copy_context(command);
    query.args(&arguments[..5]).args(["env", "-json", "GOFLAGS"]);
    Ok(query)
}

pub(super) fn inventory(command: &Command) -> io::Result<Command> {
    transform(command, true)
}

pub(super) fn execution(command: &Command) -> io::Result<Command> {
    transform(command, false)
}

fn transform(command: &Command, list: bool) -> io::Result<Command> {
    let arguments = arguments(command)?;
    admit_arguments(&arguments[6..])?;
    let flags = command
        .get_envs()
        .find(|(name, _)| *name == "GOFLAGS")
        .map_or_else(|| env::var_os("GOFLAGS"), |(_, value)| value.map(ToOwned::to_owned));
    if let Some(flags) = flags {
        admit_environment(&split(flags.as_bytes())?)?;
    }
    let mut transformed = copy_context(command);
    transformed.args(&arguments[..6]).arg("-json");
    if list {
        transformed.args(["-list", "."]);
    }
    transformed.args(&arguments[6..]);
    Ok(transformed)
}

// Go's quoted.Split recognizes quotes only at a field's start and performs no unescaping.
fn split(mut bytes: &[u8]) -> io::Result<Vec<OsString>> {
    let mut fields = Vec::new();
    while !bytes.is_empty() {
        let start = bytes
            .iter()
            .position(|byte| !matches!(byte, b' ' | b'\t' | b'\n' | b'\r'))
            .unwrap_or(bytes.len());
        bytes = &bytes[start..];
        if bytes.is_empty() {
            break;
        }
        if bytes[0] == b'\'' || bytes[0] == b'"' {
            let quote = bytes[0];
            bytes = &bytes[1..];
            let end = bytes
                .iter()
                .position(|byte| *byte == quote)
                .ok_or_else(|| invalid("unterminated GOFLAGS quote"))?;
            fields.push(OsString::from_vec(bytes[..end].to_vec()));
            bytes = &bytes[end + 1..];
        } else {
            let end = bytes
                .iter()
                .position(|byte| matches!(byte, b' ' | b'\t' | b'\n' | b'\r'))
                .unwrap_or(bytes.len());
            fields.push(OsString::from_vec(bytes[..end].to_vec()));
            bytes = &bytes[end..];
        }
    }
    Ok(fields)
}

fn flag(argument: &OsStr) -> io::Result<(&str, Option<&OsStr>)> {
    let bytes = argument.as_bytes();
    let bytes = bytes
        .strip_prefix(b"--")
        .or_else(|| bytes.strip_prefix(b"-"))
        .ok_or_else(|| invalid("unexpected Go stage package or non-flag GOFLAGS field"))?;
    let (name, value) = bytes
        .iter()
        .position(|byte| *byte == b'=')
        .map_or((bytes, None), |end| (&bytes[..end], Some(OsStr::from_bytes(&bytes[end + 1..]))));
    if name.is_empty() || name[0] == b'-' {
        return Err(invalid("invalid Go flag spelling"));
    }
    let name = std::str::from_utf8(name).map_err(|_| invalid("invalid Go flag name"))?;
    Ok((name, value))
}

#[derive(Clone, Copy)]
enum Kind {
    Boolean,
    Value,
    Parallelism,
    Compiler,
    ModuleMode,
    BuildMode,
    CoverMode,
    BuildVcs,
}

fn kind(name: &str) -> io::Result<Kind> {
    match name {
        "a" | "n" | "x" | "v" | "asan" | "msan" | "race" | "linkshared" | "modcacherw"
        | "trimpath" | "work" | "cover" => Ok(Kind::Boolean),
        "asmflags" | "gcflags" | "gccgoflags" | "ldflags" | "modfile" | "overlay"
        | "installsuffix" | "pgo" | "pkgdir" | "tags" | "toolexec" | "coverpkg" => Ok(Kind::Value),
        "p" => Ok(Kind::Parallelism),
        "compiler" => Ok(Kind::Compiler),
        "mod" => Ok(Kind::ModuleMode),
        "buildmode" => Ok(Kind::BuildMode),
        "covermode" => Ok(Kind::CoverMode),
        "buildvcs" => Ok(Kind::BuildVcs),
        _ => Err(invalid("unsupported Go selection, repetition or ambiguous flag")),
    }
}

fn boolean(value: &str) -> bool {
    ["1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "false", "False"]
        .contains(&value)
}

fn parallelism(value: &str) -> bool {
    let value = value.strip_prefix('+').unwrap_or(value);
    let (radix, digits) =
        if let Some(digits) = value.strip_prefix("0x").or_else(|| value.strip_prefix("0X")) {
            (16, digits)
        } else if let Some(digits) = value.strip_prefix("0o").or_else(|| value.strip_prefix("0O")) {
            (8, digits)
        } else if let Some(digits) = value.strip_prefix("0b").or_else(|| value.strip_prefix("0B")) {
            (2, digits)
        } else if value.len() > 1 && value.starts_with('0') {
            (8, &value[1..])
        } else {
            (10, value)
        };
    i64::from_str_radix(digits, radix).is_ok_and(|value| value > 0)
}

fn check(kind: Kind, value: Option<&OsStr>) -> io::Result<()> {
    if value.is_some_and(|value| value.as_bytes().contains(&0)) {
        return Err(invalid("NUL in Go flag value"));
    }
    if matches!(kind, Kind::Value) {
        return if value.is_some() { Ok(()) } else { Err(invalid("Go build flag needs a value")) };
    }
    let text = match value {
        Some(value) => Some(value.to_str().ok_or_else(|| invalid("non-Unicode Go scalar flag"))?),
        None => None,
    };
    let valid = match kind {
        Kind::Boolean => text.is_none_or(boolean),
        Kind::BuildVcs => text.is_none_or(|value| value == "auto" || boolean(value)),
        Kind::Parallelism => text.is_some_and(parallelism),
        Kind::Compiler => text.is_some_and(|value| ["gc", "gccgo"].contains(&value)),
        Kind::ModuleMode => {
            text.is_some_and(|value| ["readonly", "vendor", "mod"].contains(&value))
        }
        Kind::BuildMode => text.is_some_and(|value| {
            ["archive", "c-archive", "c-shared", "default", "exe", "pie", "plugin", "shared"]
                .contains(&value)
        }),
        Kind::CoverMode => text.is_some_and(|value| ["set", "count", "atomic"].contains(&value)),
        Kind::Value => true,
    };
    if valid { Ok(()) } else { Err(invalid("invalid Go build flag value")) }
}

fn admit_environment(flags: &[OsString]) -> io::Result<()> {
    for argument in flags {
        let (name, value) = flag(argument)?;
        check(kind(name)?, value)?;
    }
    Ok(())
}

fn admit_arguments(arguments: &[OsString]) -> io::Result<()> {
    let mut index = 0;
    let mut packages = 0;
    let mut counts = 0;
    while index < arguments.len() {
        let argument = &arguments[index];
        index += 1;
        if argument == "./..." {
            packages += 1;
            continue;
        }
        let (name, inline) = flag(argument)?;
        let mut value = inline;
        let kind = if name == "count" { None } else { Some(kind(name)?) };
        if inline.is_none() && !matches!(kind, Some(Kind::Boolean | Kind::BuildVcs)) {
            value = Some(
                arguments.get(index).ok_or_else(|| invalid("missing Go flag value"))?.as_os_str(),
            );
            index += 1;
        }
        if let Some(kind) = kind {
            check(kind, value)?;
        } else {
            if value != Some(OsStr::new("1")) {
                return Err(invalid("Go stage requires count=1"));
            }
            counts += 1;
        }
    }
    if packages != 1 || counts != 1 {
        return Err(invalid("Go stage requires exactly one ./... and explicit count=1"));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{ffi::OsString, os::unix::ffi::OsStringExt, path::Path};

    fn command(program: &str, module: &str) -> Command {
        let mut command = Command::new(program);
        command
            .args([
                "exec", "--", "go", "-C", module, "test", "-tags", "test_dep", "-count=1", "./...",
            ])
            .current_dir("/caller with spaces")
            .env("CGO_ENABLED", "0")
            .env("UMPIRE_WASM", "/guest with spaces/control.wasm")
            .env("CARGO_HOME", "/cargo home")
            .env_remove("GOFLAGS")
            .env_remove("BASH_ENV");
        command
    }

    #[test]
    fn three_canonical_stages_preserve_context_and_add_only_json_inventory_flags() -> io::Result<()>
    {
        for module in
            ["/model/umpire/hosts/go", "/model/umpire/reference/kettle", "/model/temporal/hosts/go"]
        {
            let original = command("/fixture/bin/mise", module);
            let observed = execution(&original)?;
            let listed = inventory(&original)?;
            assert!(is_go_test(&original));
            for transformed in [&observed, &listed] {
                assert_eq!(transformed.get_program(), original.get_program());
                assert_eq!(transformed.get_current_dir(), original.get_current_dir());
                assert_eq!(
                    transformed.get_envs().collect::<Vec<_>>(),
                    original.get_envs().collect::<Vec<_>>()
                );
            }
            assert_eq!(
                observed.get_args().collect::<Vec<_>>(),
                [
                    "exec", "--", "go", "-C", module, "test", "-json", "-tags", "test_dep",
                    "-count=1", "./..."
                ]
            );
            assert_eq!(
                listed.get_args().collect::<Vec<_>>(),
                [
                    "exec", "--", "go", "-C", module, "test", "-json", "-list", ".", "-tags",
                    "test_dep", "-count=1", "./..."
                ]
            );
        }
        Ok(())
    }

    #[test]
    fn environment_query_preserves_module_and_complete_command_context() -> io::Result<()> {
        let original = command("/fixture/bin/mise", "/module with spaces");
        let query = environment(&original)?;
        assert_eq!(query.get_program(), original.get_program());
        assert_eq!(query.get_current_dir(), original.get_current_dir());
        assert_eq!(query.get_envs().collect::<Vec<_>>(), original.get_envs().collect::<Vec<_>>());
        assert_eq!(
            query.get_args().collect::<Vec<_>>(),
            ["exec", "--", "go", "-C", "/module with spaces", "env", "-json", "GOFLAGS"]
        );
        assert!(environment(&Command::new("/bin/sh")).is_err());
        Ok(())
    }

    #[test]
    fn recognition_is_structural_and_does_not_hide_rejected_go_flags() {
        assert!(is_go_test(&command("mise", "/module")));
        assert!(is_go_test(command("mise", "/module").arg("-run=Hidden")));
        for (program, arguments) in [
            ("/bin/sh", vec!["exec", "--", "go", "-C", "/module", "test"]),
            ("arbitrary-mise", vec!["exec", "--", "go", "-C", "/module", "test"]),
            ("mise", vec!["exec", "--", "cargo", "test"]),
            ("mise", vec!["exec", "--", "go", "-C", "/module", "list"]),
            ("mise", vec!["exec", "--", "go", "test"]),
        ] {
            let mut unrelated = Command::new(program);
            unrelated.args(arguments);
            assert!(!is_go_test(&unrelated));
            assert!(inventory(&unrelated).is_err());
            assert!(execution(&unrelated).is_err());
        }
    }

    #[test]
    fn selection_repetition_cache_and_forwarding_controls_are_rejected() {
        for flag in [
            "-run=TestHidden",
            "--run=TestHidden",
            "-test.run=TestHidden",
            "-skip=TestHidden",
            "-cpu=1,2",
            "-bench=.",
            "-fuzz=FuzzX",
            "-fuzztime=1s",
            "-count=2",
            "-count=1",
            "-list=.",
            "-args",
            "-c",
            "-exec=wrapper",
            "-failfast",
            "-short",
            "-json=false",
            "-unknown=1",
        ] {
            let mut original = command("mise", "/module");
            original.arg(flag);
            assert!(inventory(&original).is_err(), "argv {flag}");
            assert!(execution(&original).is_err(), "argv {flag}");
            let mut original = command("mise", "/module");
            original.env("GOFLAGS", flag);
            assert!(inventory(&original).is_err(), "GOFLAGS {flag}");
            assert!(execution(&original).is_err(), "GOFLAGS {flag}");
        }
        let mut missing_count = Command::new("mise");
        missing_count
            .args(["exec", "--", "go", "-C", "/module", "test", "-tags=test_dep", "./..."])
            .env_remove("GOFLAGS");
        assert!(inventory(&missing_count).is_err());
    }

    #[test]
    fn build_flags_and_go_quoted_split_are_preserved_without_shell_interpretation() -> io::Result<()>
    {
        for flags in [
            " ",
            "-tags=control -p=9 -p=1",
            "'-tags=control extra' -p=1",
            "\"-tags=control extra\" -p=1",
            "'-tags=control'-p=9 -p=1",
            "--tags=control --p=1",
            "-gcflags=all=-N",
            "'-ldflags=-X=example.Value=with spaces'",
            "-mod=readonly -trimpath -buildvcs=false",
        ] {
            let mut original = command("mise", "/module");
            original.env("GOFLAGS", flags);
            for transformed in [inventory(&original)?, execution(&original)?] {
                assert_eq!(
                    transformed.get_envs().collect::<Vec<_>>(),
                    original.get_envs().collect::<Vec<_>>()
                );
            }
        }
        for flags in [
            "'unterminated",
            "\"unterminated",
            "''",
            "-tags='control extra'",
            "-p 1",
            "---p=1",
            "--",
            "-",
            "-unknown=1",
            "'-run=Hidden'",
            "'-tags=control' '-skip=Hidden'",
            "-p=oops",
        ] {
            let mut original = command("mise", "/module");
            original.env("GOFLAGS", flags);
            assert!(inventory(&original).is_err(), "{flags}");
        }
        Ok(())
    }

    #[test]
    fn raw_os_values_and_explicit_go_flags_removal_survive_transformation() -> io::Result<()> {
        let raw = OsString::from_vec(b"/module-\xff".to_vec());
        let mut original = Command::new("mise");
        original
            .args(["exec", "--", "go", "-C"])
            .arg(&raw)
            .args(["test", "-tags=control", "-p=9", "-p=1", "-count=1", "./..."])
            .current_dir(Path::new("/caller"))
            .env("RAW_PATH", OsString::from_vec(b"value-\xfe".to_vec()))
            .env_remove("GOFLAGS");
        let transformed = inventory(&original)?;
        assert_eq!(transformed.get_args().nth(4), Some(raw.as_os_str()));
        assert_eq!(
            transformed.get_envs().collect::<Vec<_>>(),
            original.get_envs().collect::<Vec<_>>()
        );
        Ok(())
    }
}
