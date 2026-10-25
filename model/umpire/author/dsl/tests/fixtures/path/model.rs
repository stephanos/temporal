use std::fmt::Write;

use umpire_dsl::{Gap, Path, PathNode, Waypoint, path};

#[derive(Clone, Debug, PartialEq)]
struct Kettle {
    temperature: i32,
    beeping: bool,
}

fn render(output: &mut String, name: &str, path: &Path<Kettle>, states: &[Kettle]) -> Result<(), Box<dyn std::error::Error>> {
    let judge = |waypoint: &Waypoint<Kettle>| {
        states.iter().map(|state| if waypoint.judge(state) { 'y' } else { 'n' }).collect::<String>()
    };
    let nodes = path.alternatives(1)?.remove(0).nodes;
    let waypoints = nodes.iter().filter(|node| matches!(node, PathNode::State(_))).count();
    write!(output, "{name}: {waypoints} waypoints;")?;
    for node in &nodes {
        match node {
            PathNode::State(waypoint) => write!(output, " {}", judge(waypoint))?,
            PathNode::Gap { gap: Gap::Any, .. } => output.push_str(" .."),
            PathNode::Gap { gap: Gap::AtMost(bound), .. } => write!(output, " ..={bound}")?,
            PathNode::Action { .. } => return Err("a state path has no actions".into()),
        }
    }
    output.push('\n');
    Ok(())
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let cold = Kettle { temperature: 20, beeping: false };
    let boiled = Kettle { temperature: 100, beeping: true };
    let states = [cold.clone(), Kettle { temperature: 60, beeping: false }, boiled.clone()];
    let steps = 3;
    let mut output = String::new();
    writeln!(output, "judged against: {states:?}")?;
    render(&mut output, "bools", &path![true, .., false], &states)?;
    let (warm, boiling, beeping) = (false, true, false);
    render(&mut output, "bool or", &path![warm | boiling | beeping, .., true], &states)?;
    render(&mut output, "closures", &path![|k: &Kettle| k.temperature < 50, ..=1, move |k: &Kettle| k.beeping], &states)?;
    render(&mut output, "inferred closure", &path![cold.clone(), .., |k| k.temperature >= 60], &states)?;
    render(&mut output, "destructured closure", &path![boiled.clone(), .., |&Kettle { beeping, .. }| !beeping], &states)?;
    render(&mut output, "states", &path![cold, ..=steps + 1, boiled], &states)?;
    let chain: Path<Kettle> = path![
        |k: &Kettle| !k.beeping, ..=2, |k: &Kettle| k.temperature == 60, .., |k: &Kettle| k.beeping,
    ];
    render(&mut output, "chain", &chain, &states)?;
    let claim = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| chain.claim()))
        .err()
        .and_then(|error| error.downcast::<String>().ok())
        .ok_or("a chain split as a claim")?;
    writeln!(output, "chain as a claim: {claim}")?;
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
