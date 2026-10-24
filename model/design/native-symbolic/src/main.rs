//! Spike driver: Kettle -> BTOR2 -> rIC3 -> native replay, per property, with timings.
//!
//! Runs: the baseline Kettle (every engine in ENGINES), two controls that mutate the admitted
//! tree so the cross-check must flag them, and a scaled Kettle (boiling at 1 000 000) that
//! compares native exploration with IC3 proof time.
#![allow(clippy::print_stdout, clippy::print_stderr)]
mod btor2;
mod smt;
mod witness;

#[allow(dead_code, missing_docs)]
mod sample_kettle;
#[allow(dead_code, missing_docs)]
mod kettle_big;

use std::fmt::Write as _;
use std::path::Path;
use std::process::Command;
use std::time::Instant;
use umpire_dsl::__lifted::Lifts;
use umpire_dsl::{CheckOutcome, Machine, Values};
use umpire_ir::expr::{Expr, ExprKind};
use umpire_ir::{Lifted, Type};

fn main() {
    if let Err(error) = run() {
        eprintln!("error: {error}");
        std::process::exit(1);
    }
}

type Fallible<T> = Result<T, Box<dyn std::error::Error>>;

struct Settings {
    ric3: String,
    out: std::path::PathBuf,
    timeout: String,
}

fn run() -> Fallible<()> {
    let settings = Settings {
        ric3: std::env::var("RIC3").map_err(|_| "set RIC3 to the ric3 binary")?,
        out: std::env::var("OUT").unwrap_or_else(|_| "out".into()).into(),
        timeout: std::env::var("TIMEOUT").unwrap_or_else(|_| "300".into()),
    };
    let engines: Vec<String> = std::env::var("ENGINES").unwrap_or_else(|_| "ic3".into()).split(',').map(str::to_owned).collect();
    let which = std::env::var("RUNS").unwrap_or_else(|_| "baseline,claim99,heat2e9,big".into());
    let mut report = String::new();
    for name in which.split(',') {
        let section = match name {
            "baseline" => check("baseline", &sample_kettle::kettle::world(), |_| {}, &engines, umpire_checker::Checker::BUDGET, &settings)?,
            "claim99" => check(
                "control claim99 (postcondition literal 100 -> 99 in the admitted tree only; expect: unsafe, replay NOT confirmed, DISAGREES)",
                &sample_kettle::kettle::world(),
                |model| {
                    for post in &mut model.postconditions {
                        if post.name == "fails_over_at_boiling" {
                            rewrite_ints(&mut post.expr, 100, 99);
                        }
                    }
                },
                &["ic3".to_owned()],
                umpire_checker::Checker::BUDGET,
                &settings,
            )?,
            "heat2e9" => check(
                "control heat2e9 (Heater::Heat guard 100 -> i32::MAX and adds 2e9, admitted tree only; expect: arithmetic unsafe, witnesses rejected)",
                &sample_kettle::kettle::world(),
                |model| {
                    for action in &mut model.actions {
                        if action.name == "Heater::Heat" {
                            for branch in &mut action.branches {
                                rewrite_ints(&mut branch.guard, 100, i128::from(i32::MAX));
                                for (_, e) in &mut branch.definitions {
                                    rewrite_ints(e, 1, 2_000_000_000);
                                }
                                for e in branch.updates.values_mut() {
                                    rewrite_ints(e, 1, 2_000_000_000);
                                }
                            }
                        }
                    }
                },
                &["ic3".to_owned()],
                umpire_checker::Checker::BUDGET,
                &settings,
            )?,
            "big" => check("big (BOILING_TEMPERATURE = 1_000_000, a real second Model)", &kettle_big::kettle::world(), |_| {}, &["ic3".to_owned()], 1 << 23, &settings)?,
            other => return Err(format!("unknown run {other}").into()),
        };
        report.push_str(&section);
        report.push('\n');
    }
    print!("{report}");
    std::fs::write(settings.out.join("report.txt"), report)?;
    Ok(())
}

fn rewrite_ints(e: &mut Expr, from: i128, to: i128) {
    match &mut e.kind {
        ExprKind::Int(v) if *v == from => *v = to,
        ExprKind::Unary(_, a) | ExprKind::Length(a) => rewrite_ints(a, from, to),
        ExprKind::Binary(_, a, b) | ExprKind::Append(a, b) => {
            rewrite_ints(a, from, to);
            rewrite_ints(b, from, to);
        }
        ExprKind::If(a, b, c) | ExprKind::GetOr(a, b, c) => {
            rewrite_ints(a, from, to);
            rewrite_ints(b, from, to);
            rewrite_ints(c, from, to);
        }
        ExprKind::Tuple(vs) | ExprKind::Sequence(vs) | ExprKind::Checks(vs) => vs.iter_mut().for_each(|v| rewrite_ints(v, from, to)),
        _ => {}
    }
}

/// One Model through the whole pipeline. `M::State` must expose `beeping` for the negative
/// control, which both Kettles do; the closure below is the fixture's `never_beeps`.
fn check<M>(label: &str, world: &M, mutate: impl FnOnce(&mut Lifted), engines: &[String], budget: usize, settings: &Settings) -> Fallible<String>
where
    M: Machine + Lifts,
    M::State: Values + Beeping,
{
    let out = settings.out.join(label.split(' ').next().unwrap_or("run"));
    std::fs::create_dir_all(&out)?;
    let embedded = M::__umpire_lifted();
    let mut model = embedded.model.map_err(|d| format!("{}:{}:{}: {}", embedded.file, d.line, d.column, d.message))?;
    mutate(&mut model);
    let never_beeps = ("never_beeps".to_owned(), Expr::var("beeping", Type::Bool, false).not());
    let started = Instant::now();
    let system = btor2::emit(&model, std::slice::from_ref(&never_beeps))?;
    let emit_time = started.elapsed();

    let started = Instant::now();
    let report = umpire_checker::Checker::explore(world, budget)?.verify_all()?;
    let native_time = started.elapsed();
    let mut expected: Vec<(String, Option<bool>)> = report
        .claims
        .iter()
        .map(|c| {
            (c.name.to_owned(), match c.outcome {
                CheckOutcome::Refuted => Some(true),
                CheckOutcome::Verified => Some(false),
                CheckOutcome::Unknown => None,
            })
        })
        .collect();
    expected.push(("never_beeps".into(), Some(true)));
    expected.push(("arithmetic".into(), Some(false)));
    let judge = |state: &M::State| !state.beeping();
    let negatives: [(&str, &dyn Fn(&M::State) -> bool); 1] = [("never_beeps", &judge)];

    let mut lines = String::new();
    writeln!(lines, "== {label}")?;
    writeln!(
        lines,
        "emit: {} shared BTOR2 lines in {emit_time:.1?}; native: {} states, depth {}, cut={}, {native_time:.2?}",
        system.shared_lines(),
        report.counts.reached,
        report.counts.depth,
        report.counts.cut
    )?;
    writeln!(lines, "{:<24} {:<14} {:<8} {:<8} {:>8}  {:<6} {}", "property", "kind", "engine", "verdict", "time", "steps", "replay; agreement")?;
    let mut all_agree = true;
    for property in &system.properties {
        let file = out.join(format!("{}.btor", property.name));
        std::fs::write(&file, system.file(property))?;
        for engine in engines {
            let mut command = Command::new("timeout");
            command.arg(&settings.timeout).arg(&settings.ric3).arg("check").arg(&file).arg("--cex").arg("--ui=false").arg(engine);
            if engine == "bmc" || engine == "wl-bmc" {
                command.arg("--end").arg((report.counts.depth + 1).to_string());
            }
            let started = Instant::now();
            let output = command.output()?;
            let elapsed = started.elapsed();
            let stdout = String::from_utf8_lossy(&output.stdout).into_owned();
            let stderr = String::from_utf8_lossy(&output.stderr).into_owned();
            std::fs::write(out.join(format!("{}.{engine}.log", property.name)), format!("{stdout}\n--- stderr ---\n{stderr}"))?;
            let bounded = engine == "bmc" || engine == "wl-bmc";
            let verdict = if stdout.contains("UNSAT") {
                "safe"
            } else if stdout.contains("SAT") {
                "unsafe"
            } else if bounded && stdout.contains("UNKNOWN") {
                "bounded"
            } else if output.status.code() == Some(124) {
                "timeout"
            } else {
                "unknown"
            };
            let native_refuted = expected.iter().find(|(n, _)| *n == property.name).and_then(|(_, r)| *r);
            let (steps, detail) = match verdict {
                "unsafe" => match witness::parse(&stdout).and_then(|frames| witness::replay(world, &system.layout, &frames, property, &negatives)) {
                    Ok(confirmed) => {
                        std::fs::write(out.join(format!("{}.path.txt", property.name)), &confirmed.path)?;
                        match confirmed.confirmed {
                            Ok(()) => (confirmed.steps.to_string(), "replayed, native violation confirmed".to_owned()),
                            Err(why) => (confirmed.steps.to_string(), format!("replayed, NOT confirmed: {why}")),
                        }
                    }
                    Err(why) => ("-".into(), format!("witness rejected: {why}")),
                },
                "safe" => ("-".into(), "unbounded proof".to_owned()),
                "bounded" => ("-".into(), format!("no violation to depth {}", report.counts.depth + 1)),
                _ => ("-".into(), String::new()),
            };
            let agreement = match (native_refuted, verdict) {
                (Some(true), "unsafe") | (Some(false), "safe" | "bounded") => "agrees",
                (Some(_), "unsafe" | "safe" | "bounded") => {
                    all_agree = false;
                    "DISAGREES"
                }
                (None, _) => "native unknown",
                _ => "no verdict",
            };
            writeln!(lines, "{:<24} {:<14} {:<8} {:<8} {:>7.2}s  {:<6} {detail}; {agreement}", property.name, format!("{:?}", property.kind), engine, verdict, elapsed.as_secs_f64(), steps)?;
        }
    }
    if let Ok(solver) = std::env::var("SMT_SOLVER") {
        writeln!(lines, "SMT-LIB bounded check through a pipe to {solver} (QF_BV, bound = native depth + 1 = {}):", report.counts.depth + 1)?;
        for property in &system.properties {
            let (depth, elapsed) = smt::bmc(&solver, &system.file(property), report.counts.depth + 1)?;
            let native_refuted = expected.iter().find(|(n, _)| *n == property.name).and_then(|(_, r)| *r);
            let agreement = match (native_refuted, depth) {
                (Some(true), Some(_)) | (Some(false), None) => "agrees",
                (Some(_), _) => "DISAGREES",
                (None, _) => "native unknown",
            };
            writeln!(lines, "{:<24} {:<8} {:>7.2}s  {}; {agreement}", property.name, "smt-bmc", elapsed.as_secs_f64(), depth.map_or("no violation within the bound".to_owned(), |d| format!("violation at depth {d} (no replay on this path)")))?;
        }
    }
    writeln!(lines, "all verdicts agree with the native checker: {all_agree}")?;
    let _ = Path::new("");
    Ok(lines)
}

/// The one field the negative control reads.
trait Beeping {
    fn beeping(&self) -> bool;
}
impl Beeping for sample_kettle::kettle::Model {
    fn beeping(&self) -> bool {
        self.beeping
    }
}
impl Beeping for kettle_big::kettle::Model {
    fn beeping(&self) -> bool {
        self.beeping
    }
}
