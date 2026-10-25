use crate::phased_rejection::job;
use job::World;
use umpire_tla::observers::{ACTION, CHOICE, FACTS, INPUT, OUTCOME, PENDING};

// Two attempts, so every Queued state has one current and one stale claim.
umpire_testing::fixture!(World = World::new(2), 64);

fn folder() -> std::path::PathBuf {
    std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/phased_rejection")
}

fn required() -> bool {
    std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require")
}

/// The emitted module with each `from` replaced by `to`, every target present exactly once.
fn edited(edits: &[(&str, &str)]) -> Result<umpire_tla::Module, Box<dyn std::error::Error>> {
    let mut module = bound().lift(folder().join("model.rs"))?;
    for (from, to) in edits {
        if module.tla.matches(from).count() != 1 {
            return Err(format!("mutation target is not unique: {from}").into());
        }
        module.tla = module.tla.replace(from, to);
    }
    Ok(module)
}

#[test]
fn closure_laws_keep_source_order_and_the_stock_outcome() -> umpire_testing::TestResult {
    let module = bound().lift(folder().join("model.rs"))?;
    let names: Vec<_> = module.metadata.postconditions.iter().map(|claim| &claim.name).collect();
    assert_eq!(
        names,
        [
            "terminal_remains",
            "closed_rejects",
            "drop_rejects",
            "reopen_remains",
            "touch_remains",
            "touch_rejects",
            "recheck_rejects",
            "steps_accept"
        ]
    );
    assert_eq!(module.metadata.outcome_type, umpire_tla::Type::outcome());
    let law = |name: &str| {
        module.tla.lines().find(|line| line.starts_with(&format!("Post_{name} =="))).unwrap_or("")
    };
    // closed_rejects judges the edge's outcome and whole state; terminal_remains only the phase.
    assert!(
        law("closed_rejects")
            .contains(format!("({OUTCOME}') = (\"Outcome::Rejected(NotFound)\")").as_str())
    );
    assert!(law("closed_rejects").contains("(attempt') = (attempt)"));
    assert!(
        !law("terminal_remains").contains(OUTCOME) && !law("terminal_remains").contains("attempt")
    );
    Ok(())
}

/// Source-only changes to an emitted outcome or action identity disagree with the native graph.
#[test]
fn emitted_outcome_and_action_mutations_fail_agreement() -> umpire_testing::TestResult {
    if !required() {
        return Ok(());
    }
    for (edits, reason) in [
        // The wrong rejection becomes the expected one: recheck_rejects would verify.
        (
            vec![("\"Outcome::Rejected(Unavailable)\"", "\"Outcome::Rejected(NotFound)\"")],
            "verdict difference recheck_rejects",
        ),
        // The accepted closed step is relabelled as a rejection.
        (
            vec![(
                format!("({OUTCOME}' = \"Outcome::Accepted\") /\\ ({FACTS}' = <<>>) /\\ ({INPUT}' = 5)) \\/").as_str(),
                format!("({OUTCOME}' = \"Outcome::Rejected(NotFound)\") /\\ ({FACTS}' = <<>>) /\\ ({INPUT}' = 5)) \\/").as_str(),
            )],
            "verdict difference touch_rejects",
        ),
        // A rejecting Recheck edge is relabelled as the accepted Touch.
        (
            vec![(
                format!("({ACTION}' = \"Client::Recheck\") /\\ ({CHOICE}' = 0) /\\ ({OUTCOME}' = \"Outcome::Rejected(Unavailable)\")").as_str(),
                format!("({ACTION}' = \"Client::Touch\") /\\ ({CHOICE}' = 0) /\\ ({OUTCOME}' = \"Outcome::Rejected(Unavailable)\")").as_str(),
            )],
            "verdict difference recheck_rejects",
        ),
    ] {
        let error = bound()
            .check_module(&folder(), &edited(&edits)?)
            .err()
            .ok_or_else(|| format!("mutant agreed: {edits:?}"))?;
        assert!(error.to_string().contains(reason), "{error}");
    }
    Ok(())
}

/// One fairness obligation over every Claim is discharged by a stale claim's rejecting
/// self-loop; the exact current-attempt obligations are not.
#[test]
fn a_stale_rejecting_claim_cannot_discharge_current_claim_fairness() -> umpire_testing::TestResult {
    use umpire_tlc::Verdict::{Refuted, Verified};
    if !required() {
        return Ok(());
    }
    let union = " /\\ WF_vars(Fair_queued_job_runs_Worker_Claim_1) /\\ WF_vars(Fair_queued_job_runs_Worker_Claim_2)";
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    for (edits, verdict) in [
        (vec![], Verified),
        (
            vec![(
                union,
                format!(
                    " /\\ WF_vars(({PENDING} = 1) /\\ Worker_Claim /\\ Monitor_queued_job_runs)"
                )
                .as_str(),
            )],
            Refuted,
        ),
    ] {
        let mut module = edited(&edits)?;
        module.metadata.invariants.clear();
        module.metadata.postconditions.clear();
        module.metadata.negative.clear();
        module.cfg = module
            .cfg
            .lines()
            .filter(|line| !line.starts_with("INVARIANT") && !line.starts_with("PROPERTY"))
            .flat_map(|line| [line, "\n"])
            .collect();
        let result = umpire_tlc::Tlc::discover(&root)?.check_module(&module)?;
        assert_eq!(result.properties["Progress_queued_job_runs"].verdict, verdict, "{edits:?}");
    }
    Ok(())
}

/// A handler that rejects after changing its state is a Model error in TLC, as natively.
#[test]
fn rejecting_after_a_mutation_fails_in_tlc() -> umpire_testing::TestResult {
    let source = std::fs::read_to_string(folder().join("model.rs"))?.replace(
        "Client::Recheck if self.closed() => reject!(Rejection::Unavailable),",
        "Client::Recheck if self.closed() => {\n                    self.phase = Phase::Queued;\n                    reject!(Rejection::Unavailable)\n                }",
    );
    let module = bound().lift_source("model.rs", &source)?;
    assert!(module.tla.contains("\"rejecting step changed state\""));
    if !required() {
        return Ok(());
    }
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let error = umpire_tlc::Tlc::discover(&root)?
        .check_module(&module)
        .err()
        .ok_or("TLC admitted a rejection after a mutation")?;
    assert!(error.to_string().contains("rejecting step changed state"), "{error}");
    Ok(())
}

/// A rejection computed from the state lowers to a choice over the stock rejections, with the
/// same outcomes as the literal it replaces in every closed state.
#[test]
fn a_computed_rejection_agrees_with_the_literal() -> umpire_testing::TestResult {
    let source = std::fs::read_to_string(folder().join("model.rs"))?.replace(
        "Client::Recheck if self.closed() => reject!(Rejection::Unavailable),",
        "Client::Recheck if self.closed() => reject!(if self.phase == Phase::Queued {\n                    Rejection::NotFound\n                } else {\n                    Rejection::Unavailable\n                }),",
    );
    let module = bound().lift_source("model.rs", &source)?;
    let recheck =
        module.tla.lines().find(|line| line.starts_with("Client_Recheck ==")).ok_or("Recheck")?;
    assert!(
        recheck.contains("= (\"Rejection::NotFound\")) THEN \"Outcome::Rejected(NotFound)\""),
        "{recheck}"
    );
    if required() {
        bound().check_module(&folder(), &module)?;
    }
    Ok(())
}

/// TLC's counterexample to `steps_accept` is the stale claim's rejected payload edge, which the
/// fixture's replay then checks against the native step.
#[test]
fn the_steps_accept_counterexample_is_a_rejected_payload_edge() -> umpire_testing::TestResult {
    use umpire_tlc::Value::{Int, Text};
    if !required() {
        return Ok(());
    }
    let mut module = bound().lift(folder().join("model.rs"))?;
    module.metadata.postconditions.retain(|claim| claim.name == "steps_accept");
    module.metadata.progress.clear();
    module.cfg = module
        .cfg
        .lines()
        .filter(|line| !line.starts_with("PROPERTY") || *line == "PROPERTY Post_steps_accept")
        .flat_map(|line| [line, "\n"])
        .collect();
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let result = umpire_tlc::Tlc::discover(&root)?.check_module(&module)?;
    let trace = result.properties["Post_steps_accept"].trace.as_ref().ok_or("no counterexample")?;
    let [_, edge] = &trace.states[..] else { return Err("expected one edge".into()) };
    assert_eq!(edge.observers[ACTION], Text("Worker::Claim".into()));
    assert_eq!(edge.observers[INPUT], Int(1));
    assert_eq!(edge.observers[OUTCOME], Text("Outcome::Rejected(FailedPrecondition)".into()));
    Ok(())
}
