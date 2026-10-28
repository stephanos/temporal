//! Admission and JSON transformations of the existing Go test leaves.
use std::{io, process::Command};

pub(super) fn is_go_test(_command: &Command) -> bool {
    false
}

pub(super) fn environment(_command: &Command) -> io::Result<Command> {
    Err(io::Error::other("Go environment command transform not implemented"))
}

pub(super) fn inventory(_command: &Command) -> io::Result<Command> {
    Err(io::Error::other("Go inventory command transform not implemented"))
}

pub(super) fn execution(_command: &Command) -> io::Result<Command> {
    Err(io::Error::other("Go execution command transform not implemented"))
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
