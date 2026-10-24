use super::{BTreeMap, BTreeSet, Flag, Isolation, LiveMemory, Plan, Policy, Stage, can_start};
use crate::cycle::model::{Adapter, Bounds, StageContract};

#[derive(Debug)]
pub(crate) struct Graph {
    pub(super) stages: BTreeMap<String, Stage>,
    pub(super) order: Vec<String>,
    pub(super) ranks: BTreeMap<String, f64>,
}

fn validate_resources(s: &StageContract) -> Result<(), String> {
    if !s.cores.is_finite()
        || !s.mem_gib.is_finite()
        || !s.estimated_seconds.is_finite()
        || s.cores <= 0.0
        || s.mem_gib < 0.0
        || s.estimated_seconds < 0.0
    {
        return Err(format!("{}: invalid resources", s.label));
    }
    Ok(())
}

fn visit(
    label: &str,
    stages: &BTreeMap<String, Stage>,
    state: &mut BTreeMap<String, bool>,
    order: &mut Vec<String>,
) -> Result<(), String> {
    if state.get(label) == Some(&true) {
        return Ok(());
    }
    if state.get(label) == Some(&false) {
        return Err(format!("dependency cycle at {label}"));
    }
    state.insert(label.into(), false);
    for dep in &stages[label].dependencies {
        visit(dep, stages, state, order)?;
    }
    state.insert(label.into(), true);
    order.push(label.into());
    Ok(())
}
impl Graph {
    pub(crate) fn new(plan: &Plan) -> Result<Self, String> {
        let mut stages = BTreeMap::new();
        for s in &plan.stages {
            validate_resources(&s.contract)?;
            if s.contract.label.is_empty() {
                return Err("empty stage label".into());
            }
            if stages.insert(s.contract.label.clone(), s.clone()).is_some() {
                return Err(format!("duplicate stage {}", s.contract.label));
            }
        }
        for s in stages.values() {
            for dep in &s.dependencies {
                if !stages.contains_key(dep) {
                    return Err(format!("{}: unknown dependency {dep}", s.contract.label));
                }
            }
        }
        let mut order = vec![];
        let mut state = BTreeMap::new();
        for label in stages.keys() {
            visit(label, &stages, &mut state, &mut order)?;
        }
        let mut ranks = BTreeMap::<String, f64>::new();
        for label in order.iter().rev() {
            let s = &stages[label];
            let successor = stages
                .values()
                .filter(|s| s.dependencies.contains(label))
                .map(|s| ranks[&s.contract.label])
                .fold(0.0, f64::max);
            let own = if matches!(s.isolation_class, Isolation::Normal | Isolation::Budgeted) {
                s.contract.estimated_seconds
            } else {
                0.0
            };
            ranks.insert(label.clone(), own + successor);
        }
        Ok(Self {
            stages,
            order,
            ranks,
        })
    }

    pub(crate) fn static_critical_path(&self) -> (f64, Vec<String>) {
        let mut finish = BTreeMap::<String, f64>::new();
        let mut via = BTreeMap::<String, Option<String>>::new();
        let mut last = None;
        let mut longest = -1.0;
        for label in &self.order {
            let s = &self.stages[label];
            let mut best = None;
            let mut end = -1.0;
            for dep in &s.dependencies {
                if finish[dep] > end {
                    best = Some(dep.clone());
                    end = finish[dep];
                }
            }
            let duration = s.contract.estimated_seconds + end.max(0.0);
            finish.insert(label.clone(), duration);
            via.insert(label.clone(), best);
            if duration > longest {
                last = Some(label.clone());
                longest = duration;
            }
        }
        let mut path = vec![];
        while let Some(label) = last.take() {
            last.clone_from(&via[&label]);
            path.push(label);
        }
        path.reverse();
        (longest.max(0.0), path)
    }

    pub(crate) fn select(
        &self,
        done: &BTreeSet<String>,
        running: &[String],
        started: &BTreeSet<String>,
        policy: &Policy,
        live_memory: LiveMemory<'_>,
    ) -> Result<Vec<String>, String> {
        let mut ready: Vec<_> = self
            .stages
            .values()
            .filter(|s| {
                !started.contains(&s.contract.label)
                    && s.dependencies.iter().all(|d| done.contains(d))
            })
            .collect();
        ready.sort_by(|a, b| {
            (a.contract.tier != 0)
                .cmp(&(b.contract.tier != 0))
                .then_with(|| {
                    self.ranks[&b.contract.label].total_cmp(&self.ranks[&a.contract.label])
                })
                .then(a.contract.label.cmp(&b.contract.label))
        });
        let mut active = running
            .iter()
            .map(|label| {
                self.stages
                    .get(label)
                    .ok_or_else(|| format!("unknown running stage {label}"))
            })
            .collect::<Result<Vec<_>, _>>()?;
        let shared_ready = ready
            .iter()
            .any(|s| matches!(s.isolation_class, Isolation::Normal | Isolation::Budgeted));
        let heavy_pending = ready
            .iter()
            .chain(active.iter())
            .any(|s| s.contract.flags.contains(&Flag::Heavy));
        let mut chosen = vec![];
        for s in ready {
            if s.isolation_class == Isolation::Budgeted && heavy_pending {
                continue;
            }
            if matches!(s.isolation_class, Isolation::Exclusive | Isolation::Solo) && shared_ready {
                continue;
            }
            if can_start(s, &active, policy, live_memory)? {
                chosen.push(s.contract.label.clone());
                active.push(s);
            } else if matches!(s.isolation_class, Isolation::Exclusive | Isolation::Solo) {
                break;
            }
        }
        Ok(chosen)
    }
}

fn list(value: &str) -> Vec<String> {
    let value = value.trim();
    if value == "-" {
        vec![]
    } else {
        value
            .split(',')
            .filter(|s| !s.is_empty())
            .map(str::to_owned)
            .collect()
    }
}
pub(crate) fn parse_dag(text: &str) -> Result<Vec<StageContract>, String> {
    let mut rows = text
        .lines()
        .filter(|line| !line.trim().is_empty() && !line.starts_with('#'));
    if rows.next() != Some("label\tdeps\tclass\tcores\tmem_gib\test_s\tlocks\tscratch\tflags\ttier")
    {
        return Err("DAG header must name the ten scheduler columns".into());
    }
    rows.map(|row| {
        let f: Vec<_> = row.split('\t').collect();
        if f.len() != 10 {
            return Err("bad DAG row".into());
        }
        let isolation_class = match f[2] {
            "normal" => Isolation::Normal,
            "budgeted" => Isolation::Budgeted,
            "exclusive" => Isolation::Exclusive,
            "solo" => Isolation::Solo,
            _ => return Err(format!("{}: unknown class {}", f[0], f[2])),
        };
        let flags = list(f[8])
            .iter()
            .map(|s| match s.as_str() {
                "bg" => Ok(Flag::Bg),
                "bgok" => Ok(Flag::Bgok),
                "cleantree" => Ok(Flag::Cleantree),
                "heavy" => Ok(Flag::Heavy),
                "memguard" => Ok(Flag::Memguard),
                _ => Err(format!("{}: unknown flag {s}", f[0])),
            })
            .collect::<Result<_, _>>()?;
        let number = |i: usize| {
            f[i].parse::<f64>()
                .map_err(|_| format!("{}: invalid resources", f[0]))
        };
        let s = StageContract {
            label: f[0].into(),
            dependencies: list(f[1]),
            isolation_class,
            cores: number(3)?,
            mem_gib: number(4)?,
            estimated_seconds: number(5)?,
            locks: list(f[6]),
            scratch: list(f[7]),
            flags,
            tier: f[9].parse().map_err(|_| "invalid tier")?,
            adapter: Adapter::CheckedCommand,
            bounds: Bounds::default(),
            required_subgates: vec![],
            repetitions: 1,
        };
        validate_resources(&s)?;
        Ok(s)
    })
    .collect()
}
