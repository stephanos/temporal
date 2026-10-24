use super::{BTreeMap, BTreeSet, Graph, LiveMemory, Policy, Stage};
use serde::Serialize;

pub(crate) struct Completion {
    pub(crate) label: String,
    pub(crate) exit: i32,
    pub(crate) end: f64,
}

pub(crate) trait Executor {
    fn now(&self) -> f64;
    fn start(&mut self, stage: &Stage, lane: usize) -> Result<(), String>;
    fn wait_any(&mut self) -> Result<Completion, String>;
    fn terminate(&mut self) -> Result<(), String>;
}

#[derive(Clone, Debug, PartialEq, Serialize)]
pub(crate) struct Record {
    pub(crate) label: String,
    pub(crate) lane: usize,
    pub(crate) start: f64,
    pub(crate) end: Option<f64>,
    pub(crate) exit: Option<i32>,
    #[serde(skip)]
    order: usize,
}

#[derive(Clone, Debug, PartialEq, Serialize)]
pub(crate) struct Hop {
    pub(crate) label: String,
    pub(crate) start: f64,
    pub(crate) end: f64,
    pub(crate) duration: f64,
    pub(crate) via: String,
}

#[derive(Debug, Serialize)]
pub(crate) struct RunResult {
    pub(crate) status: i32,
    pub(crate) wall_s: f64,
    pub(crate) failed: Vec<String>,
    pub(crate) not_run: Vec<String>,
    pub(crate) stages: BTreeMap<String, Record>,
    pub(crate) critical_path: Vec<Hop>,
}

pub(crate) fn run(
    graph: &Graph,
    policy: &Policy,
    executor: &mut impl Executor,
    live_memory: LiveMemory<'_>,
) -> Result<RunResult, String> {
    policy.validate()?;
    let result = run_inner(graph, policy, executor, live_memory);
    if result.is_err()
        && let Err(cleanup) = executor.terminate()
    {
        return Err(format!(
            "{}; executor cleanup: {cleanup}",
            result.unwrap_err()
        ));
    }
    result
}

fn run_inner(
    graph: &Graph,
    p: &Policy,
    e: &mut impl Executor,
    live_memory: LiveMemory<'_>,
) -> Result<RunResult, String> {
    let mut done = BTreeSet::new();
    let mut started = BTreeSet::new();
    let mut running = Vec::<String>::new();
    let mut records = BTreeMap::<String, Record>::new();
    let mut failed = vec![];
    let mut wall_exceeded = false;
    let origin = e.now();
    if !origin.is_finite() {
        return Err("invalid executor clock".into());
    }
    let mut previous = origin;
    loop {
        let now = e.now();
        if !now.is_finite() || now < previous {
            return Err("executor clock moved backwards or is nonfinite".into());
        }
        previous = now;
        if !wall_exceeded && p.deadline.is_some_and(|deadline| now > deadline) {
            wall_exceeded = true;
        }
        if failed.is_empty() && !(wall_exceeded && p.stop_at_deadline) {
            for label in graph.select(&done, &running, &started, p, live_memory)? {
                let lane = (1..=running.len() + 1)
                    .find(|lane| running.iter().all(|l| records[l].lane != *lane))
                    .ok_or("no free scheduler lane")?;
                let start = e.now();
                if !start.is_finite() || start < previous {
                    return Err("invalid stage start clock".into());
                }
                previous = start;
                records.insert(
                    label.clone(),
                    Record {
                        label: label.clone(),
                        lane,
                        start: start - origin,
                        end: None,
                        exit: None,
                        order: started.len(),
                    },
                );
                started.insert(label.clone());
                e.start(&graph.stages[&label], lane)?;
                running.push(label);
            }
        }
        if running.is_empty() {
            break;
        }
        let completed = e.wait_any()?;
        let Some(index) = running.iter().position(|label| *label == completed.label) else {
            return Err(format!(
                "completion for nonrunning stage {}",
                completed.label
            ));
        };
        let record = records
            .get_mut(&completed.label)
            .ok_or("missing stage start")?;
        if !completed.end.is_finite()
            || completed.end < origin + record.start
            || completed.end < previous
            || completed.end > e.now()
        {
            return Err("invalid stage completion clock".into());
        }
        previous = completed.end;
        running.remove(index);
        record.end = Some(completed.end - origin);
        record.exit = Some(completed.exit);
        if completed.exit == 0 {
            done.insert(completed.label);
        } else {
            failed.push(completed.label);
        }
    }
    let now = e.now();
    let not_run = graph
        .stages
        .keys()
        .filter(|label| !started.contains(*label))
        .cloned()
        .collect::<Vec<_>>();
    let status = if !failed.is_empty() {
        1
    } else if wall_exceeded || p.deadline.is_some_and(|deadline| now > deadline) {
        3
    } else {
        i32::from(!not_run.is_empty())
    };
    let critical_path = actual_critical_path(graph, &records);
    Ok(RunResult {
        status,
        wall_s: now - origin,
        failed,
        not_run,
        stages: records,
        critical_path,
    })
}

fn latest<'a>(records: impl Iterator<Item = &'a Record>) -> Option<&'a Record> {
    records.max_by(|a, b| {
        a.end
            .unwrap()
            .total_cmp(&b.end.unwrap())
            .then_with(|| b.order.cmp(&a.order))
    })
}
fn actual_critical_path(graph: &Graph, records: &BTreeMap<String, Record>) -> Vec<Hop> {
    let finished: Vec<_> = records.values().filter(|r| r.end.is_some()).collect();
    let mut current = latest(finished.iter().copied());
    let mut path = vec![];
    while let Some(record) = current {
        let deps = &graph.stages[&record.label].dependencies;
        let mut previous = deps
            .iter()
            .filter_map(|dep| records.get(dep))
            .filter(|r| r.end.is_some())
            .max_by(|a, b| {
                a.end.unwrap().total_cmp(&b.end.unwrap()).then_with(|| {
                    let ai = deps.iter().position(|d| d == &a.label).unwrap();
                    let bi = deps.iter().position(|d| d == &b.label).unwrap();
                    bi.cmp(&ai)
                })
            });
        let mut reason = if previous.is_some() { "dep" } else { "start" };
        let dep_end = previous.and_then(|r| r.end).unwrap_or(0.0);
        if record.start - dep_end > 1.0
            && let Some(blocker) = latest(
                finished
                    .iter()
                    .copied()
                    .filter(|r| r.order < record.order && r.end.unwrap() <= record.start + 1e-6),
            )
        {
            previous = Some(blocker);
            reason = "resource";
        }
        let rounded = |v: f64| (v * 10.0).round() / 10.0;
        path.push(Hop {
            label: record.label.clone(),
            start: rounded(record.start),
            end: rounded(record.end.unwrap()),
            duration: rounded(record.end.unwrap() - record.start),
            via: reason.into(),
        });
        current = previous;
    }
    path.reverse();
    path
}
