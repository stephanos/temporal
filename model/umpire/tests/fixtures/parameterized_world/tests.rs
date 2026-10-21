use crate::parameterized_world::{counter, enumerations, reset_enumerations};
use counter::World as SelectedWorld;
use umpire_dsl::Claim;

std::thread_local! {
    static SELECTION: std::cell::Cell<u8> = const { std::cell::Cell::new(0) };
    static CONSTRUCTIONS: std::cell::Cell<usize> = const { std::cell::Cell::new(0) };
}

fn choose_world() -> SelectedWorld {
    CONSTRUCTIONS.set(CONSTRUCTIONS.get() + 1);
    match SELECTION.get() {
        0 => SelectedWorld::new(1, 3),
        1 => SelectedWorld::new(0, 1),
        _ => SelectedWorld::new(2, 1),
    }
}

umpire_testing::fixture!(
    choose_world(),
    64,
    Claim::new("never_three", |_, state: &counter::Model| (state.value < 3).into())
);

fn selected<T>(selection: u8, operation: impl FnOnce() -> T) -> T {
    let previous = SELECTION.replace(selection);
    CONSTRUCTIONS.set(0);
    reset_enumerations();
    let result = operation();
    SELECTION.set(previous);
    result
}

fn selected_once() {
    assert_eq!((CONSTRUCTIONS.get(), enumerations()), (1, 1));
}

#[test]
fn each_native_execution_selects_one_world_and_enumerates_once() -> umpire_testing::TestResult {
    for (selection, states, edges, verdict) in
        [(0, 5, 3, "Refuted"), (1, 3, 1, "Verified"), (2, 0, 0, "Verified")]
    {
        selected(selection, || -> umpire_testing::TestResult {
            let files = run()?;
            selected_once();
            assert_eq!(files["states.txt"].lines().count(), states);
            assert_eq!(files["rows.txt"].lines().count(), edges);
            assert!(
                files["negative_report.txt"].contains(&format!("claim never_three: {verdict}")),
                "{}",
                files["negative_report.txt"]
            );
            if selection == 0 {
                assert!(files["paths/never_three.txt"].contains("User::Advance"));
            }
            Ok(())
        })?;
    }
    Ok(())
}

#[test]
fn each_source_binding_preserves_selection_and_empty_starts() -> umpire_testing::TestResult {
    let mut modules = Vec::new();
    for selection in 0..3 {
        modules.push(selected(selection, || -> Result<_, Box<dyn std::error::Error>> {
            let module = crate::lifted("parameterized_world")?;
            selected_once();
            Ok(module)
        })?);
    }
    assert_ne!(modules[0].cfg, modules[1].cfg);
    assert_ne!(modules[0].tla, modules[1].tla);
    assert!(modules[2].tla.contains("FALSE"));
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn actual_tlc_replays_the_false_claim_in_the_same_selected_world() -> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!("World replay control skipped: no TLC evidence (set UMPIRE_BACKENDS=require)");
        return Ok(());
    }
    let folder =
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/parameterized_world");
    for (selection, verdict, replayed) in [(0, "Refuted", 1), (1, "Verified", 0)] {
        selected(selection, || -> umpire_testing::TestResult {
            let files = crate::render_tlc(run, &folder)?;
            selected_once();
            let report = &files["tlc.txt"];
            assert!(report.contains(&format!("never_three: {verdict}")), "{report}");
            assert!(report.contains(&format!("counterexamples replayed: {replayed}")), "{report}");
            Ok(())
        })?;
    }
    Ok(())
}
