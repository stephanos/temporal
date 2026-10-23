use crate::kettle_progress::kettle;
use std::fmt::Write;
use umpire_checker::{Checker, ClaimResult, ProgressVerdict};
use umpire_dsl::{Assumption, Machine, CheckOutcome};
use umpire_testing::Files;

#[path = "../../support/progress_starts.rs"]
mod progress_starts;

use progress_starts::ProgressStarts;

umpire_testing::lifted!(kettle::world());

fn world<M: Machine<State = kettle::Model>>(
    world: M,
    temperature: i32,
) -> Result<ProgressStarts<M>, Box<dyn std::error::Error>> {
    let claim = world
        .progress()
        .into_iter()
        .find(|claim| claim.name == "boils")
        .ok_or("Kettle has no registered boils claim")?;
    Ok(ProgressStarts {
        world,
        starts: vec![kettle::Model { temperature, heating_on: false, beeping: false }],
        claim,
    })
}

fn render<S: std::fmt::Display, A: std::fmt::Display>(
    files: &mut Files,
    report: &mut String,
    question: &str,
    result: ClaimResult<S, A>,
    expected: &(CheckOutcome, ProgressVerdict),
) -> umpire_testing::TestResult {
    assert_eq!(
        (result.outcome, result.verdict.as_ref()),
        (expected.0, Some(&expected.1)),
        "{question}"
    );
    if matches!(question, "room_unrestricted" | "room_fair_heat") {
        let path = result.path.as_ref().ok_or("Kettle cycle has no witness")?;
        assert_eq!(path.cycle_start, Some(1), "{question}");
        let actions: Vec<_> = path
            .steps
            .iter()
            .map(|step| step.action.to_string())
            .collect();
        assert_eq!(
            actions,
            ["User::PressButton", "User::DepressButton", "User::PressButton"],
            "{question}"
        );
    }
    write!(report, "{question}: ")?;
    write!(report, "{:?}", result.outcome)?;
    report.push_str("; ");
    write!(report, "{}", result.verdict.as_ref().ok_or("Kettle progress has no progress verdict")?)?;
    report.push('\n');
    if let Some(hint) = result.hint {
        writeln!(report, "hint: {hint}")?;
    }
    if let Some(path) = result.path {
        let text = path.to_string();
        files.insert(format!("paths/{question}.txt"), text);
    }
    Ok(())
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let room = world(kettle::world(), 20)?;
    let graph = Checker::explore(&room, 1 << 20)?;
    let mut files = Files::new();
    let mut report = String::new();
    for (question, claim, expected) in [
        (
            "room_never_switched_off",
            room.progress().remove(0),
            (CheckOutcome::Verified, ProgressVerdict::Verified { longest: Some(81) }),
        ),
        (
            "room_unrestricted",
            room.progress().remove(0).assuming(Assumption::default()),
            (CheckOutcome::Refuted, ProgressVerdict::Cycle),
        ),
        (
            "room_press_only",
            room.progress().remove(0).assuming(Assumption::only(&[kettle::User::PressButton])),
            (CheckOutcome::Refuted, ProgressVerdict::Deadlock),
        ),
        (
            "room_within_80",
            room.progress().remove(0).within_steps(80),
            (CheckOutcome::Refuted, ProgressVerdict::MissedDeadline),
        ),
        (
            "room_fair_heat",
            room.progress().remove(0).assuming(Assumption::fair(&[kettle::Heater::Heat])),
            (CheckOutcome::Refuted, ProgressVerdict::Cycle),
        ),
    ] {
        render(&mut files, &mut report, question, graph.verify_progress(&room, &claim)?, &expected)?;
    }
    let cold = world(kettle::world(), -40)?;
    let cold_graph = Checker::explore(&cold, 1 << 20)?;
    render(
        &mut files,
        &mut report,
        "cold_never_switched_off",
        cold_graph.verify_progress(&cold, &cold.progress().remove(0))?,
        &(CheckOutcome::Verified, ProgressVerdict::Verified { longest: Some(141) }),
    )?;
    let cut = Checker::explore(&room, 10)?;
    render(
        &mut files,
        &mut report,
        "room_budget_10",
        cut.verify_progress(&room, &room.progress().remove(0))?,
        &(CheckOutcome::Unknown, ProgressVerdict::Unknown),
    )?;
    files.insert("report.txt".into(), report);
    Ok(files)
}
