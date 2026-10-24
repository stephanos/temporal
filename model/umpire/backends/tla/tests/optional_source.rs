//! Source grammar for stored bounded optional values, independent of native enumeration.
use umpire_tla::Lift;

const SOURCE: &str = r"
#[machine] mod optional_source {
    use umpire_dsl::{Action, Finite, State, World, check};
    #[derive(Finite, Clone, Copy, Debug, Default)]
    struct Remaining(#[inputs(0..=3)] u8);
    #[derive(Finite, Clone, Copy, Debug)]
    struct Ticks { #[inputs(1..=3)] value: u8 }
    const LIMIT: Option<Ticks> = Some(Ticks { value: 3 });
    #[derive(World)] struct World { timeout: Option<Ticks> }
    #[derive(State)] struct Model { remaining: Option<Remaining> }
    #[derive(Action)] enum User { Tick }
    impl Model {
        #[starts] fn starts(w: &World) -> impl IntoIterator<Item = Self> {
            [Self { remaining: None }]
        }
        #[on(User)] fn user(&mut self, action: User, w: &World) {
            match action {
                User::Tick if w.timeout.is_some() => {
                    let fallback = Remaining(0);
                    let value: Option<Remaining> = self.remaining;
                    if value.is_none() {
                        self.remaining = Some(Remaining(w.timeout.unwrap_or(Ticks { value: 1 }).value));
                    } else {
                        self.remaining = Some(Remaining(value.unwrap_or(fallback).0));
                    }
                }
                _ => {}
            }
        }
        #[invariant] fn bounded(&self) {
            check!(Remaining::contains(self.remaining.unwrap_or(Remaining(0)).0));
            check!(None == self.remaining || self.remaining.is_some());
            check!(LIMIT.is_some());
        }
        #[postcondition] fn unchanged(&self, before: &Self, w: &World) {
            check!(before.remaining.unwrap_or(Remaining(0)).0 <= w.timeout.unwrap_or(Ticks { value: 3 }).value);
        }
    }
}
";

#[test]
fn canonical_optional_paths_and_nominal_operations() {
    for path in [
        "Option",
        "core::option::Option",
        "std::option::Option",
        "::core::option::Option",
        "::std::option::Option",
    ] {
        let source = SOURCE.replace("Option<", &format!("{path}<"));
        Lift::parse("optional_source.rs", &source)
            .unwrap_or_else(|error| panic!("{path}: {error}"));
    }
    for prefix in ["Option::", "core::option::Option::", "std::option::Option::"] {
        let source = SOURCE
            .replace("Some(", &format!("{prefix}Some("))
            .replace("= None", &format!("= {prefix}None"));
        Lift::parse("optional_source.rs", &source)
            .unwrap_or_else(|error| panic!("{prefix}: {error}"));
    }
}

#[test]
fn unsupported_optional_sources_refuse_at_source_locations() {
    let cases = [
        ("Option<Remaining>", "Option<u8>", "bounded newtype"),
        ("Option<Remaining>", "Option<Option<Remaining>>", "bounded newtype"),
        ("Option<Remaining>", "Option<Remaining, Ticks>", "exactly one"),
        ("Option<Remaining>", "Option", "exactly one"),
        (
            "self.remaining = Some(Remaining(value.unwrap_or(fallback).0))",
            "self.remaining = Some(Ticks { value: 1 })",
            "type mismatch",
        ),
        ("let fallback = Remaining(0);", "let fallback = None;", "type context"),
        ("value.unwrap_or(fallback).0", "value.unwrap().0", "unwrap"),
        ("value.unwrap_or(fallback).0", "value.expect(\"present\").0", "optional method"),
        ("value.unwrap_or(fallback).0", "value.as_mut().unwrap_or(fallback).0", "optional method"),
        (
            "let fallback = Remaining(0);",
            "let Some = true; let fallback = Remaining(0);",
            "canonical constructor",
        ),
        ("let fallback = Remaining(0);", "let Option = true; let fallback = Remaining(0);", ""),
        ("core::option::Option", "other::Option", "unsupported type"),
    ];
    for (from, to, diagnostic) in cases {
        let base = if from == "core::option::Option" {
            SOURCE.replace("Option<", "core::option::Option<")
        } else {
            SOURCE.to_owned()
        };
        let source = base.replace(from, to);
        if to.starts_with("let Option") {
            // The prelude Some constructor remains unshadowed by an unrelated local Option.
            Lift::parse("optional_source.rs", &source).unwrap_or_else(|error| panic!("{error}"));
            continue;
        }
        let error = match Lift::parse("optional_source.rs", &source) {
            Ok(_) => panic!("admitted {to}"),
            Err(error) => error.to_string(),
        };
        assert!(error.contains(diagnostic), "{to}: {error}");
        assert!(error.contains("optional_source.rs:"), "{error}");
    }
}

#[test]
fn canonical_optional_imports_preserve_builtin_identity() {
    for import in [
        "use core::option::Option;",
        "use std::option::Option;",
        "use core::option::Option::{Some, None};",
        "use std::option::{Option, Some, None};",
        "use ::core::option::Option; use ::std::option::Option::{Some, None};",
    ] {
        let source = SOURCE.replace("use umpire_dsl::{", &format!("{import}\nuse umpire_dsl::{{"));
        Lift::parse("optional_import.rs", &source)
            .unwrap_or_else(|error| panic!("{import}: {error}"));
    }
    for import in [
        "use core::option::Option as Option;",
        "use core::option::Option::*;",
        "use other::Option;",
        "mod core {} use core::option::Option;",
    ] {
        let source = SOURCE.replace("use umpire_dsl::{", &format!("{import}\nuse umpire_dsl::{{"));
        let error = Lift::parse("optional_import.rs", &source)
            .err()
            .unwrap_or_else(|| panic!("admitted {import}"));
        assert!(error.line > 1 && error.column > 0, "{error}");
    }
}
