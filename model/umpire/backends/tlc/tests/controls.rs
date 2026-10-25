//! Real TLC controls for native maximal emitted-edge semantics.

#![forbid(unsafe_code)]

use std::{collections::BTreeMap, fs, path::Path};
use umpire_tla::observers::{ACTION, CHOICE, EDGE_TICK, FACTS, OUTCOME, PENDING, STEPS};

use umpire_tla::Type;
use umpire_tlc::{Error, Refusal, Schema, Tlc, Value, Verdict};

fn schema() -> Schema {
    Schema {
        fields: BTreeMap::from([("x".into(), Type::Int { min: 0, max: 3 })]),
        observers: BTreeMap::from([
            (EDGE_TICK.into(), Type::Bool),
            (PENDING.into(), Type::Int { min: 0, max: 2 }),
            (STEPS.into(), Type::Int { min: 0, max: 4 }),
            (ACTION.into(), Type::Text),
            (CHOICE.into(), Type::Int { min: 0, max: 0 }),
            (OUTCOME.into(), Type::Bool),
            (FACTS.into(), Type::Seq(Box::new(Type::Text))),
        ]),
    }
}

fn module(
    edges: &[(u8, u8, &str)],
    fair: Option<&str>,
    bound: Option<u8>,
    trigger: u8,
    target: u8,
    allowed: Option<&str>,
) -> String {
    let actions: Vec<_> = edges.iter().enumerate().map(|(index, (from, to, action))| {
        let distance = bound.map_or(STEPS.into(), |bound| format!("IF {STEPS} < {} THEN {STEPS} + 1 ELSE {STEPS}", bound + 1));
        let outcome = if *action == "Reject" { "FALSE" } else { "TRUE" };
        format!("Edge{index} == /\\ x = {from} /\\ x' = {to}\n /\\ {PENDING} # 2 /\\ {PENDING}' = (IF {PENDING} = 1 /\\ x' = {target} THEN 2 ELSE {PENDING})\n /\\ {EDGE_TICK}' = ~{EDGE_TICK} /\\ {STEPS}' = (IF {PENDING} = 1 THEN ({distance}) ELSE {STEPS})\n /\\ {ACTION}' = \"{action}\" /\\ {CHOICE}' = 0 /\\ {OUTCOME}' = {outcome} /\\ {FACTS}' = <<\"{action}\">>\n")
    }).collect();
    let next =
        (0..edges.len()).map(|index| format!("Edge{index}")).collect::<Vec<_>>().join(" \\/ ");
    let scoped = edges
        .iter()
        .enumerate()
        .filter(|(_, (_, _, action))| allowed.is_none_or(|allowed| allowed == *action))
        .map(|(index, _)| format!("Edge{index}"))
        .collect::<Vec<_>>()
        .join(" \\/ ");
    let fairness = fair.map_or(String::new(), |action| {
        let relation = edges
            .iter()
            .enumerate()
            .filter(|(_, (_, _, name))| *name == action)
            .map(|(index, _)| format!("Edge{index}"))
            .collect::<Vec<_>>()
            .join(" \\/ ");
        format!(" /\\ WF_vars(({PENDING} = 1) /\\ ({relation}))")
    });
    let bounded = bound.map_or(String::new(), |bound| format!("Bound == []({STEPS} <= {bound})\n"));
    format!(
        r#"---- MODULE Control ----
EXTENDS Integers
VARIABLES x, {EDGE_TICK}, {PENDING}, {STEPS}, {ACTION}, {CHOICE}, {OUTCOME}, {FACTS}
vars == <<x, {EDGE_TICK}, {PENDING}, {STEPS}, {ACTION}, {CHOICE}, {OUTCOME}, {FACTS}>>
Init == x = 0 /\ {EDGE_TICK} = FALSE /\ {PENDING} = 0 /\ {STEPS} = 0 /\ {ACTION} = "" /\ {CHOICE} = 0 /\ {OUTCOME} = TRUE /\ {FACTS} = <<>>
{actions}
ModelNext == {next}
ScopedNext == {scoped}
RealNext == (({PENDING} = 0) /\ ModelNext) \/ (({PENDING} = 1) /\ ScopedNext)
Activate == /\ x = {trigger} /\ {PENDING} = 0 /\ {PENDING}' = 1
 /\ UNCHANGED <<x, {EDGE_TICK}, {STEPS}, {ACTION}, {CHOICE}, {OUTCOME}, {FACTS}>>
Complete == ({PENDING} = 2 \/ ~ENABLED RealNext) /\ UNCHANGED vars
Next == RealNext \/ Activate \/ Complete
Spec == Init /\ [][Next]_vars{fairness}
Maximal == ([]<>{EDGE_TICK} /\ []<>(~{EDGE_TICK})) \/ <>[]({PENDING} = 1 /\ ~ENABLED ScopedNext)
Progress == Maximal => ({PENDING} = 1 ~> {PENDING} = 2)
Naive == {PENDING} = 1 ~> {PENDING} = 2
Post == [][~({EDGE_TICK}' # {EDGE_TICK} /\ {ACTION}' = "A") \/ x' >= x]_vars
InitialBad == x # 0
{bounded}
====
"#,
        actions = actions.join("\n")
    )
}

type Control = (
    &'static str,
    Vec<(u8, u8, &'static str)>,
    Option<&'static str>,
    Option<u8>,
    &'static str,
    Verdict,
);

fn controls() -> [Control; 14] {
    [
        ("acyclic", vec![(0, 1, "A"), (1, 2, "A")], None, None, "Progress", Verdict::Verified),
        ("idle_is_not_maximal", vec![(0, 2, "A")], None, None, "Progress", Verdict::Verified),
        ("naive_idle", vec![(0, 2, "A")], None, None, "Naive", Verdict::Refuted),
        ("finite_dead_end", vec![(0, 1, "A")], None, None, "Progress", Verdict::Refuted),
        (
            "authored_self_loop",
            vec![(0, 0, "A"), (0, 2, "B")],
            None,
            None,
            "Progress",
            Verdict::Refuted,
        ),
        (
            "rejecting_self_loop",
            vec![(0, 0, "Reject"), (0, 2, "B")],
            None,
            None,
            "Progress",
            Verdict::Refuted,
        ),
        (
            "fair_exit",
            vec![(0, 0, "A"), (0, 2, "B")],
            Some("B"),
            None,
            "Progress",
            Verdict::Verified,
        ),
        (
            "intermittently_enabled",
            vec![(0, 1, "A"), (1, 0, "B"), (1, 2, "C")],
            Some("C"),
            None,
            "Progress",
            Verdict::Refuted,
        ),
        ("exact_bound", vec![(0, 1, "A"), (1, 2, "A")], None, Some(2), "Bound", Verdict::Verified),
        ("missed_bound", vec![(0, 1, "A"), (1, 2, "A")], None, Some(1), "Bound", Verdict::Refuted),
        ("postcondition", vec![(0, 1, "A"), (1, 2, "A")], None, None, "Post", Verdict::Verified),
        (
            "postcondition_refuted",
            vec![(0, 1, "A"), (1, 0, "A")],
            None,
            None,
            "Post",
            Verdict::Refuted,
        ),
        (
            "unrestricted_prefix",
            vec![(0, 1, "A"), (1, 2, "B")],
            None,
            None,
            "Progress",
            Verdict::Refuted,
        ),
        (
            "bound_excludes_incoming_edge",
            vec![(0, 1, "A"), (1, 2, "B")],
            None,
            Some(1),
            "Bound",
            Verdict::Verified,
        ),
    ]
}

#[test]
#[allow(clippy::print_stderr)]
fn real_tlc_maximality_fairness_bounds_and_action_properties() -> Result<(), Error> {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!("TLC controls skipped: no TLC evidence (set UMPIRE_BACKENDS=require)");
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = Tlc::discover(&root)?;
    let evidence = root.join(".flow/tmp/tlc-controls");
    fs::create_dir_all(&evidence)?;
    for (name, edges, fair, bound, property, expected) in controls() {
        let (trigger, target, allowed) = match name {
            "unrestricted_prefix" => (1, 3, Some("B")),
            "bound_excludes_incoming_edge" => (1, 2, None),
            _ => (0, 2, None),
        };
        let result = tlc.check(
            "Control",
            &module(&edges, fair, bound, trigger, target, allowed),
            &format!("SPECIFICATION Spec\nPROPERTY {property}\n"),
            &schema(),
        )?;
        fs::write(
            evidence.join(format!("{name}.txt")),
            format!(
                "control: {name}\nverdict: {:?}\nprojected states: {}\nraw states: {}\n{}",
                result.properties[property].verdict,
                result.states.len(),
                result.raw_state_count,
                result.output
            ),
        )?;
        assert_eq!(result.properties[property].verdict, expected, "{name}: {}", result.output);
        assert!(!result.states.is_empty(), "{name}");
        assert_eq!(
            result.properties[property].trace.is_some(),
            expected == Verdict::Refuted,
            "{name}"
        );
        if name == "rejecting_self_loop" {
            assert!(result.properties[property].trace.as_ref().is_some_and(|trace| {
                trace.states.iter().any(|state| {
                    state.observers[OUTCOME] == Value::Bool(false)
                        && state.observers[FACTS] == Value::Seq(vec![Value::Text("Reject".into())])
                })
            }));
        }
    }
    Ok(())
}

#[test]
fn real_tlc_initial_invariant_has_a_typed_counterexample() -> Result<(), Error> {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = Tlc::discover(&root)?;
    let result = tlc.check(
        "Control",
        &module(&[(0, 1, "A")], None, None, 0, 2, None),
        "SPECIFICATION Spec\nINVARIANT InitialBad\n",
        &schema(),
    )?;
    assert_eq!(result.properties["InitialBad"].verdict, Verdict::Refuted);
    assert_eq!(
        result.properties["InitialBad"].trace.as_ref().map(|trace| trace.states.len()),
        Some(1)
    );
    Ok(())
}

#[test]
fn real_tlc_keeps_deadlock_checks_and_refuses_tool_errors() -> Result<(), Error> {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = Tlc::discover(&root)?;
    let source = module(&[(0, 1, "A")], None, None, 0, 2, None);
    let deadlock = source
        .replace("Next == RealNext \\/ Activate \\/ Complete", "Next == RealNext \\/ Activate");
    let error = tlc
        .check("Control", &deadlock, "SPECIFICATION Spec\n", &schema())
        .err()
        .ok_or_else(|| Refusal("TLC admitted a deadlocked completion relation".into()))?;
    assert!(error.to_string().contains("Deadlock reached"), "{error}");
    let error = tlc
        .check(
            "Control",
            "---- MODULE Control ----\nthis is invalid\n====\n",
            "SPECIFICATION Spec\n",
            &schema(),
        )
        .err()
        .ok_or_else(|| Refusal("TLC admitted invalid syntax".into()))?;
    let error = error.to_string();
    assert!(error.contains("error") || error.contains("failed"), "{error}");
    Ok(())
}

/// TLC's own evaluation failures, an integer overflow past its 32-bit integers and a failed
/// `Assert`, surface as typed errors naming the property and the state.
#[test]
fn real_tlc_evaluation_errors_are_typed_and_located() -> Result<(), Error> {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = Tlc::discover(&root)?;
    let schema = Schema {
        fields: BTreeMap::from([("x".into(), Type::Int { min: 0, max: 3 })]),
        observers: BTreeMap::new(),
    };
    let module = |init: &str, next: &str| {
        format!(
            "---- MODULE Evaluation ----\nEXTENDS Integers, TLC\nVARIABLE x\nInit == {init}\nNext == {next}\nSpec == Init /\\ [][Next]_x\nComplete == UNCHANGED x\nWide == x * 1000000000 # 7\n====\n"
        )
    };
    let counting = module("x = 0", "(x < 3 /\\ x' = x + 1) \\/ (x = 3 /\\ UNCHANGED x)");
    let evaluation = |tla: &str, cfg: &str| match tlc.check("Evaluation", tla, cfg, &schema) {
        Err(Error::Evaluation(evaluation)) => Ok(evaluation),
        other => Err(Refusal(format!("not an evaluation error: {other:?}"))),
    };
    let wide = evaluation(&counting, "SPECIFICATION Spec\nINVARIANT Wide\n")?;
    assert_eq!(wide.property.as_deref(), Some("Wide"), "{wide}");
    assert_eq!(wide.message, "Overflow when computing 3*1000000000", "{wide}");
    assert_eq!(wide.state.as_deref(), Some("x = 3"), "{wide}");
    assert_eq!(
        wide.trace.as_ref().and_then(|trace| trace.states.last()).map(|state| &state.state["x"]),
        Some(&Value::Int(3))
    );
    let asserted = module(
        "x = 0",
        "x < 3 /\\ x' = (IF Assert(x < 1, \"checked Rust integer overflow\") THEN x + 1 ELSE x)",
    );
    let next = evaluation(&asserted, "SPECIFICATION Spec\n")?;
    assert_eq!(next.property, None, "{next}");
    assert!(next.message.contains("checked Rust integer overflow"), "{next}");
    assert_eq!(next.state.as_deref(), Some("x = 1"), "{next}");
    let initial = evaluation(&module("x = 3 * 1000000000", "x' = x"), "SPECIFICATION Spec\n")?;
    assert!(initial.trace.is_none(), "{initial}");
    assert!(initial.to_string().starts_with("TLC evaluation error in the initial predicate"));
    let evidence = root.join(".flow/tmp/tlc-controls");
    fs::create_dir_all(&evidence)?;
    fs::write(evidence.join("evaluation_errors.txt"), format!("{wide}\n\n{next}\n\n{initial}\n"))?;
    Ok(())
}
