use std::{
    collections::BTreeMap,
    ffi::{OsStr, OsString},
    ops::Range,
    os::unix::ffi::{OsStrExt, OsStringExt},
};

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(crate) enum Kind {
    Test,
    Run,
    Build,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(crate) enum BoundaryKind {
    Args,
    Terminator,
    Positional,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(crate) struct Boundary {
    pub index: usize,
    pub kind: BoundaryKind,
}

#[derive(Debug, Default)]
pub(crate) struct Invocation {
    pub driver_ranges: Vec<Range<usize>>,
    pub package_indices: Vec<usize>,
    pub guest_ranges: Vec<Range<usize>>,
    pub boundary: Option<Boundary>,
    pub count: Option<OsString>,
    pub exec: Option<OsString>,
    pub caller_exec: bool,
    pub tags: Vec<OsString>,
    pub effective: BTreeMap<String, OsString>,
    pub goflags: Vec<OsString>,
    pub remove_arguments: Vec<usize>,
    pub remove_goflags: Vec<usize>,
    build_flags: Vec<(String, OsString)>,
    goflags_raw: OsString,
    goflags_ranges: Vec<Range<usize>>,
    caller_overlay: bool,
}

impl Invocation {
    /// Keeps caller order and bytes while inserting flags before every package or guest boundary.
    pub fn rewritten_arguments(
        &self,
        original: &[OsString],
        injected: &[OsString],
    ) -> Vec<OsString> {
        injected
            .iter()
            .cloned()
            .chain(
                original
                    .iter()
                    .enumerate()
                    .filter(|(index, _)| self.remove_arguments.binary_search(index).is_err())
                    .map(|(_, value)| value.clone()),
            )
            .collect()
    }

    pub fn rewritten_goflags(&self) -> OsString {
        let raw = self.goflags_raw.as_bytes();
        let mut result = Vec::new();
        let mut cursor = 0;
        for &index in &self.remove_goflags {
            let range = &self.goflags_ranges[index];
            result.extend_from_slice(&raw[cursor..range.start]);
            cursor = range.end;
        }
        result.extend_from_slice(&raw[cursor..]);
        OsString::from_vec(result)
    }

    pub fn merged_tags(&self, required: &[String]) -> OsString {
        crate::install::tags::merged(&self.tags, required)
    }

    pub fn build_arguments(&self, _original: &[OsString]) -> Result<Vec<String>, String> {
        self.build_flags
            .iter()
            .map(|(name, value)| {
                let value = value
                    .to_str()
                    .ok_or_else(|| format!("gomad: build flag -{name} is not UTF-8"))?;
                Ok(format!("-{name}={value}"))
            })
            .collect()
    }
}

const MAXIMUM_ARGUMENTS: usize = 65536;
const MAXIMUM_BYTES: usize = 16 << 20;

/// Arguments exclude the Go subcommand. Host passthrough must bypass this parser.
pub(crate) fn inspect(
    arguments: &[OsString],
    goflags: &OsStr,
    kind: Kind,
) -> Result<Invocation, String> {
    let invocation = inspect_unchecked(arguments, goflags, kind)?;
    invocation.admit()?;
    Ok(invocation)
}

/// Parses boundaries without guest policy. Resolve run/build package opt-in before admission.
pub(crate) fn inspect_unchecked(
    arguments: &[OsString],
    goflags: &OsStr,
    kind: Kind,
) -> Result<Invocation, String> {
    let size = arguments
        .iter()
        .try_fold(goflags.len(), |total, value| total.checked_add(value.len()));
    if arguments.len() > MAXIMUM_ARGUMENTS || size.is_none_or(|size| size > MAXIMUM_BYTES) {
        return Err("gomad: Go argument capacity exceeded".into());
    }
    let (fields, ranges) = quoted_fields(goflags)?;
    let mut invocation = Invocation {
        goflags: fields,
        goflags_raw: goflags.to_owned(),
        goflags_ranges: ranges,
        ..Invocation::default()
    };
    let mut exec_arguments = Vec::new();
    let mut exec_goflags = Vec::new();
    for index in 0..invocation.goflags.len() {
        let raw = invocation.goflags[index].as_bytes();
        let Some((name, explicit)) = flag(raw) else {
            return Err("go: parsing $GOFLAGS: non-flag argument".into());
        };
        let Some(descriptor) = descriptor(name, kind) else {
            // GOFLAGS may contain flags registered only by a different Go command.
            continue;
        };
        let value = flag_value(name, explicit, None, descriptor)?;
        let name = name.to_owned();
        if removes(&name) {
            invocation.remove_goflags.push(index);
        }
        if name == "exec" {
            exec_goflags.push(index);
        }
        invocation.record(&name, value, descriptor.build)?;
    }
    match kind {
        Kind::Test => parse_test(arguments, &mut invocation, &mut exec_arguments)?,
        Kind::Run | Kind::Build => {
            parse_build(arguments, kind, &mut invocation, &mut exec_arguments)?;
        }
    }
    if !invocation.caller_exec {
        invocation.remove_arguments.extend(exec_arguments);
        invocation.remove_goflags.extend(exec_goflags);
    }
    invocation.remove_arguments.sort_unstable();
    invocation.remove_goflags.sort_unstable();
    Ok(invocation)
}

fn parse_test(
    arguments: &[OsString],
    invocation: &mut Invocation,
    exec_arguments: &mut Vec<usize>,
) -> Result<(), String> {
    let mut index = 0;
    let mut in_packages = false;
    let mut packages_complete = false;
    let mut after_unknown_without_value = false;
    while index < arguments.len() {
        let raw = arguments[index].as_bytes();
        let was_after_unknown = std::mem::take(&mut after_unknown_without_value);
        if raw == b"--" {
            invocation.suffix(index, index, arguments.len(), BoundaryKind::Terminator);
            break;
        }
        if help(raw) {
            return Err("go: help requested".into());
        }
        let Some((name, explicit)) = flag(raw) else {
            if !in_packages && packages_complete {
                if was_after_unknown {
                    invocation.guest_ranges.push(index..index + 1);
                    index += 1;
                    continue;
                }
                invocation.suffix(index, index, arguments.len(), BoundaryKind::Positional);
                break;
            }
            in_packages = true;
            invocation.package_indices.push(index);
            index += 1;
            continue;
        };
        if in_packages {
            in_packages = false;
            packages_complete = true;
        }
        let Some(descriptor) = descriptor(name, Kind::Test) else {
            packages_complete = true;
            if matches!(raw, b"-args" | b"--args") {
                invocation.suffix(index, index + 1, arguments.len(), BoundaryKind::Args);
                break;
            }
            invocation.guest_ranges.push(index..index + 1);
            after_unknown_without_value = explicit.is_none();
            index += 1;
            continue;
        };
        index = consume(
            arguments,
            index,
            name,
            explicit,
            descriptor,
            invocation,
            exec_arguments,
        )?;
    }
    Ok(())
}

fn parse_build(
    arguments: &[OsString],
    kind: Kind,
    invocation: &mut Invocation,
    exec_arguments: &mut Vec<usize>,
) -> Result<(), String> {
    let mut index = 0;
    while index < arguments.len() {
        let raw = arguments[index].as_bytes();
        if raw == b"--" {
            index += 1;
            break;
        }
        if matches!(raw, b"-h" | b"-help" | b"--h" | b"--help") {
            return Err("go: help requested".into());
        }
        let Some((name, explicit)) = flag(raw) else {
            if raw.len() > 1 && raw[0] == b'-' {
                return Err(format!(
                    "go: bad flag syntax: {}",
                    arguments[index].to_string_lossy()
                ));
            }
            break;
        };
        let descriptor = descriptor(name, kind)
            .ok_or_else(|| format!("go: flag provided but not defined: -{name}"))?;
        index = consume(
            arguments,
            index,
            name,
            explicit,
            descriptor,
            invocation,
            exec_arguments,
        )?;
    }
    if kind == Kind::Build {
        invocation.package_indices.extend(index..arguments.len());
    } else if index < arguments.len() {
        if arguments[index].as_bytes().ends_with(b".go") {
            while index < arguments.len() && arguments[index].as_bytes().ends_with(b".go") {
                invocation.package_indices.push(index);
                index += 1;
            }
        } else {
            invocation.package_indices.push(index);
            index += 1;
        }
        if index < arguments.len() {
            invocation.suffix(index, index, arguments.len(), BoundaryKind::Positional);
        }
    }
    Ok(())
}

#[derive(Clone, Copy)]
struct Descriptor {
    boolean: bool,
    build: bool,
    test_verbose: bool,
}

// Mirrors Go 1.27.1 AddBuildFlags, AddCoverFlags and testflag.go registrations.
fn descriptor(name: &str, kind: Kind) -> Option<Descriptor> {
    let (boolean, build) = match name {
        "a" | "asan" | "buildvcs" | "linkshared" | "modcacherw" | "msan" | "race" | "trimpath"
        | "cover" => (true, true),
        "n" | "x" | "work" | "json" => (true, false),
        "asmflags" | "compiler" | "buildmode" | "gcflags" | "gccgoflags" | "mod" | "modfile"
        | "overlay" | "installsuffix" | "ldflags" | "pgo" | "pkgdir" | "tags" | "toolexec"
        | "covermode" | "coverpkg" => (false, true),
        "debug-actiongraph" | "debug-runtime-trace" | "debug-trace" | "p" | "o" | "C" => {
            (false, false)
        }
        "v" if kind != Kind::Test => (true, false),
        "exec" if kind != Kind::Build => (false, false),
        "c" if kind == Kind::Test => (true, false),
        "vet" if kind == Kind::Test => (false, false),
        _ if kind == Kind::Test => {
            return test_descriptor(name.strip_prefix("test.").unwrap_or(name));
        }
        _ => return None,
    };
    Some(Descriptor {
        boolean,
        build,
        test_verbose: false,
    })
}

fn test_descriptor(name: &str) -> Option<Descriptor> {
    let boolean = match name {
        "artifacts" | "benchmem" | "failfast" | "fullpath" | "short" | "v" => true,
        "bench"
        | "benchtime"
        | "blockprofile"
        | "blockprofilerate"
        | "count"
        | "cpu"
        | "cpuprofile"
        | "fuzz"
        | "list"
        | "memprofile"
        | "memprofilerate"
        | "mutexprofile"
        | "mutexprofilefraction"
        | "outputdir"
        | "parallel"
        | "run"
        | "skip"
        | "timeout"
        | "fuzztime"
        | "fuzzminimizetime"
        | "trace"
        | "shuffle"
        | "coverprofile" => false,
        _ => return None,
    };
    Some(Descriptor {
        boolean,
        build: false,
        test_verbose: name == "v",
    })
}

fn consume(
    arguments: &[OsString],
    index: usize,
    name: &str,
    explicit: Option<&[u8]>,
    descriptor: Descriptor,
    invocation: &mut Invocation,
    exec_arguments: &mut Vec<usize>,
) -> Result<usize, String> {
    let count = if explicit.is_none() && !descriptor.boolean {
        2
    } else {
        1
    };
    let value = flag_value(name, explicit, arguments.get(index + 1), descriptor)?;
    let range = index..index + count;
    invocation.driver_ranges.push(range.clone());
    if removes(name) {
        invocation.remove_arguments.extend(range.clone());
    }
    if name == "exec" {
        exec_arguments.extend(range.clone());
    }
    invocation.record(name, value, descriptor.build)?;
    Ok(range.end)
}

fn flag(raw: &[u8]) -> Option<(&str, Option<&[u8]>)> {
    let raw = raw.strip_prefix(b"--").or_else(|| raw.strip_prefix(b"-"))?;
    if raw.is_empty() || matches!(raw[0], b'-' | b'=') {
        return None;
    }
    let split = raw.iter().position(|byte| *byte == b'=');
    let (name, value) = split.map_or((raw, None), |index| {
        (&raw[..index], Some(&raw[index + 1..]))
    });
    // Non-UTF-8 flag names are unrecognized, as opposed to positional arguments.
    Some((std::str::from_utf8(name).unwrap_or("\u{fffd}"), value))
}

fn help(raw: &[u8]) -> bool {
    matches!(raw, b"-?" | b"-h" | b"-help" | b"--?" | b"--h" | b"--help")
}

fn flag_value(
    name: &str,
    explicit: Option<&[u8]>,
    next: Option<&OsString>,
    descriptor: Descriptor,
) -> Result<OsString, String> {
    let boolean = descriptor.boolean;
    let value = if let Some(value) = explicit {
        OsString::from_vec(value.to_vec())
    } else if boolean {
        OsString::from("true")
    } else {
        next.cloned()
            .ok_or_else(|| format!("go: flag needs an argument: -{name}"))?
    };
    if boolean
        && !(name == "buildvcs" && matches!(value.as_bytes(), b"auto" | b""))
        && !(descriptor.test_verbose && value == "test2json")
        && parse_bool(&value).is_none()
    {
        return Err(format!("go: invalid boolean value for -{name}"));
    }
    Ok(value)
}

fn parse_bool(value: &OsStr) -> Option<bool> {
    match value.as_bytes() {
        b"1" | b"t" | b"T" | b"TRUE" | b"true" | b"True" => Some(true),
        b"0" | b"f" | b"F" | b"FALSE" | b"false" | b"False" => Some(false),
        _ => None,
    }
}

fn removes(name: &str) -> bool {
    matches!(name, "tags" | "overlay")
}

impl Invocation {
    fn record(&mut self, name: &str, value: OsString, build: bool) -> Result<(), String> {
        let name = name.strip_prefix("test.").unwrap_or(name);
        match name {
            "C" => return Err("go: -C flag must be first flag on command line".into()),
            "overlay" if !value.is_empty() => self.caller_overlay = true,
            "tags" => self.tags = parse_tags(&value)?,
            "exec" => {
                self.caller_exec = !split_quoted(&value)?.is_empty();
                self.exec = Some(value.clone());
            }
            "count" => self.count = Some(value.clone()),
            "covermode" | "coverpkg" | "coverprofile" => {
                self.effective
                    .insert("cover".into(), OsString::from("true"));
            }
            _ => {}
        }
        if build && !removes(name) {
            self.build_flags.push((name.into(), value.clone()));
        }
        self.effective.insert(name.into(), value);
        Ok(())
    }

    fn suffix(&mut self, index: usize, start: usize, end: usize, kind: BoundaryKind) {
        self.boundary = Some(Boundary { index, kind });
        if start != end {
            self.guest_ranges.push(start..end);
        }
    }

    pub fn admit(&self) -> Result<(), String> {
        if self.caller_overlay {
            return Err(
                "gomad: guest option -overlay conflicts with the authenticated build recipe".into(),
            );
        }
        for name in ["race", "msan", "asan", "cover"] {
            if self
                .effective
                .get(name)
                .is_some_and(|value| parse_bool(value) == Some(true))
            {
                return Err(format!("gomad: unsupported guest option -{name}"));
            }
        }
        for name in ["fuzz", "coverprofile", "coverpkg"] {
            if self
                .effective
                .get(name)
                .is_some_and(|value| !value.is_empty())
            {
                return Err(format!("gomad: unsupported guest option -{name}"));
            }
        }
        for (name, accepted) in [
            ("overlay", &[""].as_slice()),
            (
                "trimpath",
                &["true", "1", "t", "T", "TRUE", "True"].as_slice(),
            ),
            (
                "buildvcs",
                &["false", "0", "f", "F", "FALSE", "False"].as_slice(),
            ),
            ("pgo", &["off"].as_slice()),
            ("buildmode", &["exe", "default"].as_slice()),
            ("compiler", &["gc"].as_slice()),
        ] {
            if self.effective.get(name).is_some_and(|value| {
                !accepted
                    .iter()
                    .any(|accepted| value == OsStr::new(accepted))
            }) {
                return Err(format!(
                    "gomad: guest option -{name} conflicts with the authenticated build recipe"
                ));
            }
        }
        Ok(())
    }
}

fn parse_tags(value: &OsStr) -> Result<Vec<OsString>, String> {
    if value
        .as_bytes()
        .iter()
        .any(|byte| matches!(byte, b' ' | b'\''))
    {
        split_quoted(value)
    } else {
        Ok(value
            .as_bytes()
            .split(|byte| *byte == b',')
            .filter(|tag| !tag.is_empty())
            .map(|tag| OsString::from_vec(tag.to_vec()))
            .collect())
    }
}

// Go's cmd/internal/quoted.Split recognizes quotes only at the start of a field
// and performs no unescaping, including inside double quotes.
fn split_quoted(value: &OsStr) -> Result<Vec<OsString>, String> {
    quoted_fields(value).map(|(fields, _)| fields)
}

fn quoted_fields(value: &OsStr) -> Result<(Vec<OsString>, Vec<Range<usize>>), String> {
    let bytes = value.as_bytes();
    let mut cursor = 0;
    let mut fields = Vec::new();
    let mut ranges = Vec::new();
    while cursor < bytes.len() {
        while cursor < bytes.len() && space(bytes[cursor]) {
            cursor += 1;
        }
        if cursor == bytes.len() {
            break;
        }
        let start = cursor;
        if matches!(bytes[cursor], b'\'' | b'"') {
            let quote = bytes[cursor];
            cursor += 1;
            let field = cursor;
            while cursor < bytes.len() && bytes[cursor] != quote {
                cursor += 1;
            }
            if cursor == bytes.len() {
                return Err("go: unterminated quoted flag string".into());
            }
            fields.push(OsString::from_vec(bytes[field..cursor].to_vec()));
            cursor += 1;
        } else {
            while cursor < bytes.len() && !space(bytes[cursor]) {
                cursor += 1;
            }
            fields.push(OsString::from_vec(bytes[start..cursor].to_vec()));
        }
        ranges.push(start..cursor);
        if fields.len() > MAXIMUM_ARGUMENTS {
            return Err("gomad: Go argument capacity exceeded".into());
        }
    }
    Ok((fields, ranges))
}

fn space(byte: u8) -> bool {
    matches!(byte, b' ' | b'\t' | b'\n' | b'\r')
}

#[cfg(test)]
mod tests {
    use super::*;

    fn args(values: &[&str]) -> Vec<OsString> {
        values.iter().map(OsString::from).collect()
    }

    fn test(values: &[&str], goflags: &str) -> Result<Invocation, String> {
        inspect(&args(values), OsStr::new(goflags), Kind::Test)
    }

    #[test]
    fn known_test_flags_are_recognized_on_both_sides_of_package_list() {
        let invocation = test(&["-run", "Selected", "./a", "./b", "-v", "--count=3"], "").unwrap();
        assert_eq!(invocation.driver_ranges, [0..2, 4..5, 5..6]);
        assert_eq!(invocation.package_indices, [2, 3]);
        assert_eq!(invocation.count, Some("3".into()));
        assert_eq!(invocation.guest_ranges, Vec::<Range<usize>>::new());
        assert!(invocation.boundary.is_none());
    }

    #[test]
    fn terminators_keep_guest_flag_lookalikes_out_of_effective_flags() {
        for (terminator, kind, start) in [
            ("-args", BoundaryKind::Args, 2),
            ("--args", BoundaryKind::Args, 2),
            ("--", BoundaryKind::Terminator, 1),
        ] {
            let original = args(&[
                "./pkg",
                terminator,
                "-race",
                "-exec=true",
                "-count=0",
                "-tags=excluded",
            ]);
            let invocation = inspect(&original, OsStr::new(""), Kind::Test).unwrap();
            assert_eq!(invocation.package_indices, [0]);
            assert_eq!(
                invocation.guest_ranges,
                Vec::from_iter(std::iter::once(start..6))
            );
            assert_eq!(invocation.boundary, Some(Boundary { index: 1, kind }));
            assert!(!invocation.caller_exec);
            assert_eq!(invocation.count, None);
            assert_eq!(invocation.tags, Vec::<OsString>::new());
            let rewritten = invocation.rewritten_arguments(&original, &args(&["-count=1"]));
            assert_eq!(rewritten[1..], original);
        }
    }

    #[test]
    fn unknown_flags_preserve_historical_optional_value_heuristic() {
        let invocation = test(
            &[
                "./pkg",
                "-custom",
                "value",
                "-count=9",
                "positional",
                "-race",
            ],
            "",
        )
        .unwrap();
        assert_eq!(invocation.package_indices, [0]);
        assert_eq!(invocation.guest_ranges, [1..2, 2..3, 4..6]);
        assert_eq!(invocation.count, Some("9".into()));
        assert_eq!(
            invocation.boundary,
            Some(Boundary {
                index: 4,
                kind: BoundaryKind::Positional
            })
        );
        assert!(test(&["./pkg", "-custom", "value", "-race"], "").is_err());
    }

    #[test]
    fn unknown_equals_flag_completes_package_list_even_before_any_package() {
        let invocation = test(&["-custom=value", "./looks-like-package", "-race"], "").unwrap();
        assert_eq!(invocation.package_indices, Vec::<usize>::new());
        assert_eq!(invocation.guest_ranges, [0..1, 1..3]);
        assert!(invocation.effective.is_empty());
    }

    #[test]
    fn unknown_without_value_consumes_one_nonflag_and_keeps_looking_for_driver_flags() {
        let invocation = test(&["-custom", "./not-a-package", "-count=4"], "").unwrap();
        assert_eq!(invocation.package_indices, Vec::<usize>::new());
        assert_eq!(invocation.guest_ranges, [0..1, 1..2]);
        assert_eq!(invocation.count, Some("4".into()));
    }

    #[test]
    fn known_string_flag_consumes_a_value_that_looks_like_a_terminator() {
        let invocation = test(&["-run", "-args", "./pkg", "-count=4"], "").unwrap();
        assert_eq!(invocation.effective["run"], "-args");
        assert_eq!(invocation.count, Some("4".into()));
        assert!(invocation.boundary.is_none());
    }

    #[test]
    fn bool_flags_do_not_consume_separate_false_argument() {
        let invocation = test(&["-short", "false", "-count=4"], "").unwrap();
        assert_eq!(invocation.package_indices, [1]);
        assert_eq!(invocation.effective["short"], "true");
        assert!(test(&["-race", "false"], "").is_err());
    }

    #[test]
    fn explicit_values_override_goflags_and_test_aliases_share_values() {
        let invocation = test(
            &["--test.count", "7", "-test.fuzz=", "--race=FALSE"],
            "-count=2 -fuzz=Old -race",
        )
        .unwrap();
        assert_eq!(invocation.count, Some("7".into()));
        assert_eq!(invocation.effective["fuzz"], "");
        assert_eq!(invocation.effective["race"], "FALSE");
        assert!(test(&["-test.count=0"], "-race=0").is_ok());
    }

    #[test]
    fn goflags_count_is_preserved_when_not_overridden() {
        let invocation = test(&["./pkg"], "-test.count=0").unwrap();
        assert_eq!(invocation.count, Some("0".into()));
    }

    #[test]
    fn nonempty_caller_launchers_including_true_are_preserved() {
        for exec in [
            "true",
            "fixture-launcher --mode=test",
            "'path with spaces' arg",
        ] {
            let original = args(&["-exec", exec, "./pkg", "-count=0"]);
            let invocation = inspect(&original, OsStr::new(""), Kind::Test).unwrap();
            assert!(invocation.caller_exec);
            assert_eq!(invocation.exec, Some(exec.into()));
            assert_eq!(invocation.rewritten_arguments(&original, &[]), original);
        }
        assert!(
            test(&[], "'-exec=fixture-launcher argument'")
                .unwrap()
                .caller_exec
        );
    }

    #[test]
    fn empty_effective_exec_is_removed_before_injecting_launcher() {
        let original = args(&["-exec=true", "./pkg", "-exec", " \t"]);
        let invocation = inspect(&original, OsStr::new("-exec=old"), Kind::Test).unwrap();
        assert!(!invocation.caller_exec);
        assert_eq!(
            invocation.rewritten_arguments(&original, &args(&["-exec=gomad run-guest"])),
            args(&["-exec=gomad run-guest", "./pkg"])
        );
        assert_eq!(invocation.rewritten_goflags(), "");
    }

    #[test]
    fn effective_tags_replace_prior_values_and_merge_once_with_required_tags() {
        let original = args(&[
            "-tags=old",
            "./pkg",
            "--tags",
            "user,shared",
            "-args",
            "-tags=guest",
        ]);
        let invocation = inspect(
            &original,
            OsStr::new("-tags=environment -count=0"),
            Kind::Test,
        )
        .unwrap();
        assert_eq!(invocation.tags, args(&["user", "shared"]));
        let tags = invocation.merged_tags(&["test_dep".into(), "shared".into()]);
        assert_eq!(tags, "shared,test_dep,user");
        assert_eq!(
            invocation.rewritten_arguments(&original, &args(&["-tags=shared,test_dep,user"])),
            args(&[
                "-tags=shared,test_dep,user",
                "./pkg",
                "-args",
                "-tags=guest"
            ])
        );
        assert_eq!(invocation.rewritten_goflags(), " -count=0");
    }

    #[test]
    fn legacy_tags_follow_go_strings_flag_grammar() {
        assert_eq!(
            test(&["-tags", "'one tag' two"], "").unwrap().tags,
            args(&["one tag", "two"])
        );
        assert_eq!(
            test(&["-tags", "a,,b,"], "").unwrap().tags,
            args(&["a", "b"])
        );
        assert_eq!(test(&["-tags", "a\tb"], "").unwrap().tags, args(&["a\tb"]));
    }

    #[test]
    fn merging_legacy_tags_preserves_literal_commas_and_spaces() {
        for raw in ["'a,b' c", "'one tag' two", "'a,b'", "''", "a\tb", "a,b"] {
            let invocation = test(&["-tags", raw], "").unwrap();
            let mut expected = invocation.tags.clone();
            expected.push(OsString::from("required"));
            expected.sort();
            expected.dedup();
            assert_eq!(
                parse_tags(&invocation.merged_tags(&["required".into()])).unwrap(),
                expected
            );
        }
    }

    #[test]
    fn goflags_quoting_is_not_shell_quoting() {
        assert_eq!(
            split_quoted(OsStr::new("'one two'\"three four\" a\\b x'y\"z")).unwrap(),
            args(&["one two", "three four", "a\\b", "x'y\"z"])
        );
        assert!(test(&[], "-tags='one two'").is_err());
        let invocation = test(&[], "'-tags=one two' -count=2").unwrap();
        assert_eq!(invocation.tags, args(&["one", "two"]));
        assert_eq!(invocation.count, Some("2".into()));
        assert!(test(&[], "'-tags=unterminated").is_err());
        assert!(test(&[], "-count 2").is_err());
    }

    #[test]
    fn goflags_rewrite_preserves_unrelated_raw_bytes_and_quotes() {
        let raw = OsStr::new("\t'-tags=foo bar'  -ldflags=x'y\"z\r\n-count=0");
        let invocation = inspect(&[], raw, Kind::Test).unwrap();
        assert_eq!(
            invocation.rewritten_goflags(),
            "\t  -ldflags=x'y\"z\r\n-count=0"
        );
    }

    #[test]
    fn instrumentation_refusal_uses_effective_boolean_values() {
        for flag in ["race", "msan", "asan", "cover"] {
            for value in ["true", "TRUE", "True", "1", "t", "T"] {
                assert!(
                    test(&[&format!("-{flag}={value}")], "")
                        .unwrap_err()
                        .contains("unsupported guest option")
                );
            }
            for value in ["false", "FALSE", "False", "0", "f", "F"] {
                assert!(test(&[&format!("-{flag}={value}")], "").is_ok());
            }
            assert!(test(&[&format!("-{flag}=false")], &format!("-{flag}")).is_ok());
        }
        for flag in ["fuzz", "coverprofile", "coverpkg"] {
            assert!(test(&[&format!("-{flag}=nonempty")], "").is_err());
        }
        assert!(test(&["-race=invalid"], "").is_err());
    }

    #[test]
    fn coverage_subflags_imply_cover_even_when_value_is_empty() {
        for flag in ["coverprofile", "coverpkg", "covermode"] {
            assert!(test(&[&format!("-{flag}=")], "").is_err());
        }
        assert!(test(&["-covermode=set", "-cover=false"], "").is_ok());
        assert!(test(&["-cover=false", "-covermode=set"], "").is_err());
    }

    #[test]
    fn unsafe_recipe_overrides_are_rejected_and_safe_values_are_preserved() {
        for (name, unsafe_value, safe_value) in [
            ("overlay", "caller.json", ""),
            ("trimpath", "false", "true"),
            ("buildvcs", "auto", "false"),
            ("pgo", "auto", "off"),
            ("buildmode", "c-shared", "exe"),
            ("compiler", "gccgo", "gc"),
        ] {
            assert!(
                test(&[&format!("-{name}={unsafe_value}")], "")
                    .unwrap_err()
                    .contains("authenticated build recipe")
            );
            assert!(test(&[&format!("-{name}={safe_value}")], "").is_ok());
        }
    }

    #[test]
    fn empty_overlay_cannot_override_injected_authenticated_overlay() {
        let original = args(&["./pkg", "-overlay", ""]);
        let invocation = inspect(&original, OsStr::new("-overlay="), Kind::Test).unwrap();
        assert_eq!(
            invocation.rewritten_arguments(&original, &args(&["-overlay=prepared.json"])),
            args(&["-overlay=prepared.json", "./pkg"])
        );
        assert_eq!(invocation.rewritten_goflags(), "");
    }

    #[test]
    fn preparation_receives_ordered_compiler_settings_from_goflags_and_argv() {
        let original = args(&[
            "-gcflags",
            "example/...=-N",
            "./pkg",
            "-ldflags=-s",
            "-run=X",
            "-v",
            "-p=2",
            "-json",
            "-o=test",
            "-tags=caller",
        ]);
        let invocation = inspect(
            &original,
            OsStr::new(
                "'-gcflags=all=-l' -mod=vendor -modfile=custom.mod -asmflags=all=-trimpath=x",
            ),
            Kind::Test,
        )
        .unwrap();
        assert_eq!(
            invocation.build_arguments(&original).unwrap(),
            [
                "-gcflags=all=-l",
                "-mod=vendor",
                "-modfile=custom.mod",
                "-asmflags=all=-trimpath=x",
                "-gcflags=example/...=-N",
                "-ldflags=-s"
            ]
        );
    }

    #[test]
    fn run_flags_stop_at_first_package_and_go_file_lists_are_preserved() {
        for (values, packages, guest) in [
            (
                vec!["-tags=x", "./cmd", "-race", "-exec=true"],
                vec![1],
                Vec::from_iter(std::iter::once(2..4)),
            ),
            (
                vec!["main.go", "helper.go", "-race"],
                vec![0, 1],
                Vec::from_iter(std::iter::once(2..3)),
            ),
            (
                vec!["--", "./cmd", "-race"],
                vec![1],
                Vec::from_iter(std::iter::once(2..3)),
            ),
        ] {
            let invocation = inspect(&args(&values), OsStr::new(""), Kind::Run).unwrap();
            assert_eq!(invocation.package_indices, packages);
            assert_eq!(invocation.guest_ranges, guest);
            assert!(!invocation.caller_exec);
        }
    }

    #[test]
    fn build_flags_stop_at_first_package_as_go_flagset_does() {
        let invocation = inspect(
            &args(&["-o", "output", "./cmd", "-race"]),
            OsStr::new(""),
            Kind::Build,
        )
        .unwrap();
        assert_eq!(
            invocation.driver_ranges,
            Vec::from_iter(std::iter::once(0..2))
        );
        assert_eq!(invocation.package_indices, [2, 3]);
        assert_eq!(invocation.guest_ranges, Vec::<Range<usize>>::new());
        assert!(inspect(&args(&["-exec=true", "./cmd"]), OsStr::new(""), Kind::Build).is_err());
    }

    #[test]
    fn malformed_standard_build_flags_do_not_become_packages() {
        for kind in [Kind::Run, Kind::Build] {
            for raw in ["---race", "-=value", "--=value"] {
                assert!(inspect_unchecked(&args(&[raw]), OsStr::new(""), kind).is_err());
            }
        }
        let invocation = test(&["---race"], "").unwrap();
        assert_eq!(invocation.package_indices, [0]);
    }

    #[test]
    fn test_only_flags_from_goflags_do_not_become_build_or_run_flags() {
        let invocation = inspect(
            &args(&["./cmd"]),
            OsStr::new("-count=0 -test.run=X"),
            Kind::Run,
        )
        .unwrap();
        assert_eq!(invocation.count, None);
        assert_eq!(
            invocation.build_arguments(&[]).unwrap(),
            Vec::<String>::new()
        );
    }

    #[test]
    fn guest_and_unrelated_argument_bytes_survive_rewriting() {
        let raw = OsString::from_vec(vec![b'g', 0xff, b'x']);
        let original = vec![
            "-run".into(),
            raw.clone(),
            "./pkg".into(),
            "--args".into(),
            raw.clone(),
            "-tags=guest".into(),
        ];
        let invocation = inspect(&original, OsStr::new(""), Kind::Test).unwrap();
        let rewritten = invocation.rewritten_arguments(&original, &args(&["-count=1"]));
        assert_eq!(rewritten[1..], original);
        assert_eq!(invocation.effective["run"], raw);
    }

    #[test]
    fn host_run_and_build_can_be_classified_without_applying_guest_policy() {
        for kind in [Kind::Run, Kind::Build] {
            let invocation = inspect_unchecked(
                &args(&["-race", "-overlay=host.json", "./host"]),
                OsStr::new("-pgo=auto"),
                kind,
            )
            .unwrap();
            assert_eq!(invocation.package_indices, [2]);
            assert!(invocation.admit().is_err());
        }
    }

    #[test]
    fn oversized_inputs_are_refused_before_parsing() {
        let arguments = vec![OsString::from("-v"); MAXIMUM_ARGUMENTS + 1];
        assert!(inspect(&arguments, OsStr::new(""), Kind::Test).is_err());
        let goflags = OsString::from_vec(vec![b' '; MAXIMUM_BYTES + 1]);
        assert!(inspect(&[], &goflags, Kind::Test).is_err());
    }
}
