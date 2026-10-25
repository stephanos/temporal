//! Native fixture hooks require no symbolic value projection.
#![forbid(unsafe_code)]

use std::path::Path;
use umpire_dsl::machine;

mod native_only {
    include!(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../../backends/native/tests/fixtures/format_values/model.rs"
    ));
}

fn unselected_native_run() -> umpire_testing::RunResult {
    panic!("an unsupported backend must refuse before running the native producer")
}

#[test]
fn manual_values_keep_native_formatting_and_refuse_symbolic_hooks() -> umpire_testing::TestResult {
    let files = native_only::run()?;
    assert!(files["rows.txt"].contains("Accepted"));
    assert!(files["rows.txt"].contains("Changed { amount: 1 }"));
    assert!(files["states.txt"].contains("State { value: Some((1, true)) }"));
    let folder = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../backends/native/tests/fixtures/format_values");
    let source = folder.join("model.rs");
    let unsupported = native_only::lifted(&source).err().ok_or("manual Model was lifted")?;
    let checked = native_only::check(&folder, &source, unselected_native_run)
        .err()
        .ok_or("manual Model reached typed backend replay")?;
    assert_eq!(checked.to_string(), unsupported);
    assert!(unsupported.contains("implements Machine by hand"), "{unsupported}");
    Ok(())
}

#[machine]
mod typed {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub(crate) struct Model {
        done: bool,
    }
    #[derive(Action, Copy)]
    pub(crate) enum User {
        Set,
    }
    impl Model {
        #[on(User)]
        fn step(&mut self, action: User) {
            match action {
                User::Set => self.done = true,
            }
        }
    }
}
umpire_testing::fixture!(typed::World, 2);

fn selected_native_run() -> umpire_testing::RunResult {
    Err("typed native producer selected".into())
}

#[test]
fn lifted_values_select_typed_backend_hook() -> umpire_testing::TestResult {
    if std::env::var_os("UMPIRE_TYPED_BACKEND_CHILD").is_none() {
        let output = std::process::Command::new(std::env::current_exe()?)
            .current_dir(Path::new(env!("CARGO_MANIFEST_DIR")).join("../../.."))
            .env("UMPIRE_TYPED_BACKEND_CHILD", "1")
            .args(["--exact", "lifted_values_select_typed_backend_hook"])
            .output()?;
        assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stdout));
        return Ok(());
    }
    let files = run()?;
    assert_eq!(files["states.txt"].lines().count(), 2);
    let folder = Path::new(env!("CARGO_MANIFEST_DIR")).join("tests");
    let source = folder.join("native_backend.rs");
    lifted(&source)?;
    let error = check(&folder, &source, selected_native_run)
        .err()
        .ok_or("typed backend did not run its native producer")?;
    assert_eq!(error.to_string(), "typed native producer selected");
    Ok(())
}
