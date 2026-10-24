use super::*;
use crate::cycle::model::{Adapter, Bounds, Mode, StageContract};
use std::cell::Cell;

fn stage(label: &str) -> Stage {
    Stage {
        contract: StageContract {
            label: label.into(),
            dependencies: vec![],
            isolation_class: Isolation::Normal,
            cores: 1.0,
            mem_gib: 1.0,
            estimated_seconds: 2.0,
            locks: vec![],
            scratch: vec![],
            flags: vec![],
            tier: 1,
            adapter: Adapter::CheckedCommand,
            bounds: Bounds::default(),
            required_subgates: vec![],
            repetitions: 1,
        },
        dependencies: vec![],
        isolation_class: Isolation::Normal,
        argv: vec!["fixture".into(), label.into()],
    }
}

fn policy() -> Policy {
    Policy {
        cores: 8.0,
        mem_gib: 20.0,
        allow_background: true,
        deadline: None,
        stop_at_deadline: false,
        margin_gib: 2.0,
    }
}

fn graph(stages: Vec<Stage>) -> Graph {
    Graph::new(&Plan {
        mode: Mode::Parallel,
        stages,
    })
    .unwrap()
}

#[test]
fn graph_validation_and_rank() {
    let mut a = stage("a");
    a.contract.estimated_seconds = 3.0;
    let mut b = stage("b");
    b.dependencies = vec!["a".into()];
    b.isolation_class = Isolation::Exclusive;
    let mut c = stage("c");
    c.dependencies = vec!["b".into(), "a".into()];
    c.contract.estimated_seconds = 5.0;
    let g = graph(vec![c.clone(), b.clone(), a.clone()]);
    assert_eq!(g.order, ["a", "b", "c"]);
    assert_eq!(
        g.ranks,
        BTreeMap::from([("a".into(), 8.0), ("b".into(), 5.0), ("c".into(), 5.0)])
    );
    assert_eq!(
        g.static_critical_path(),
        (10.0, vec!["a".into(), "b".into(), "c".into()])
    );
    let mut invalid = a.clone();
    invalid.contract.cores = 0.0;
    assert!(
        Graph::new(&Plan {
            mode: Mode::Parallel,
            stages: vec![invalid]
        })
        .unwrap_err()
        .contains("invalid resources")
    );
    for (bad, error) in [
        (vec![a.clone(), a.clone()], "duplicate stage"),
        (vec![b.clone()], "unknown dependency"),
        (
            {
                a.dependencies = vec!["c".into()];
                vec![a, b, c]
            },
            "dependency cycle",
        ),
    ] {
        assert!(
            Graph::new(&Plan {
                mode: Mode::Parallel,
                stages: bad
            })
            .unwrap_err()
            .contains(error)
        );
    }
}

#[test]
fn dag_rows_refuse_each_invalid_boundary() {
    let header = "label\tdeps\tclass\tcores\tmem_gib\test_s\tlocks\tscratch\tflags\ttier\n";
    let row = "a\t-\tnormal\t1\t0\t0\t-\t-\t-\t1\n";
    assert_eq!(parse_dag(&format!("{header}{row}")).unwrap()[0].label, "a");
    for (name, input, error) in [
        ("header", format!("wrong\n{row}"), "DAG header"),
        ("row", format!("{header}a\t-\n"), "bad DAG row"),
        (
            "class",
            format!("{header}{}", row.replace("normal", "unknown")),
            "unknown class",
        ),
        (
            "flag",
            format!("{header}{}", row.replace("-\t1\n", "unknown\t1\n")),
            "unknown flag",
        ),
        (
            "cores",
            format!("{header}{}", row.replace("normal\t1", "normal\t0")),
            "invalid resources",
        ),
        (
            "memory",
            format!("{header}{}", row.replace("1\t0\t0", "1\t-1\t0")),
            "invalid resources",
        ),
        (
            "estimate",
            format!("{header}{}", row.replace("1\t0\t0", "1\t0\t-1")),
            "invalid resources",
        ),
        (
            "nan",
            format!("{header}{}", row.replace("normal\t1", "normal\tNaN")),
            "invalid resources",
        ),
        (
            "infinity",
            format!("{header}{}", row.replace("1\t0\t0", "1\tinf\t0")),
            "invalid resources",
        ),
    ] {
        assert!(parse_dag(&input).unwrap_err().contains(error), "{name}");
    }
}

#[test]
fn admission_each_disjunct_and_valid_control() {
    let p = policy();
    let ordinary = stage("running");
    assert!(can_start(&stage("next"), &[], &p, None).unwrap());
    assert!(can_start(&stage("next"), &[&ordinary], &p, None).unwrap());
    let mut over = stage("over");
    over.contract.cores = 100.0;
    over.contract.mem_gib = 100.0;
    over.contract.flags.push(Flag::Memguard);
    let calls = Cell::new(0);
    let live = || {
        calls.set(calls.get() + 1);
        Ok(0.0)
    };
    assert!(can_start(&over, &[], &p, Some(&live)).unwrap());
    assert_eq!(calls.get(), 0);
    // Each row changes one exclusion and includes a nearby admitted control.
    for name in [
        "solo_candidate",
        "solo_running",
        "budgeted_exclusive",
        "budgeted_heavy",
        "heavy_budgeted",
        "exclusive_exclusive",
        "exclusive_policy",
        "exclusive_bgok",
        "exclusive_running_bg",
        "running_exclusive_policy",
        "running_exclusive_bg",
        "running_exclusive_bgok",
        "clean_candidate",
        "clean_running",
        "lock",
        "cores",
        "memory",
        "guard_candidate",
        "guard_running",
        "live",
    ] {
        let mut s = stage("next");
        let mut r = ordinary.clone();
        let mut pol = p;
        match name {
            "solo_candidate" => s.isolation_class = Isolation::Solo,
            "solo_running" => r.isolation_class = Isolation::Solo,
            "budgeted_exclusive" => {
                s.isolation_class = Isolation::Budgeted;
                r.isolation_class = Isolation::Exclusive;
                r.contract.flags.push(Flag::Bgok);
                s.contract.flags.push(Flag::Bg);
            }
            "budgeted_heavy" => {
                s.isolation_class = Isolation::Budgeted;
                r.contract.flags.push(Flag::Heavy);
            }
            "heavy_budgeted" => {
                s.contract.flags.push(Flag::Heavy);
                r.isolation_class = Isolation::Budgeted;
            }
            "exclusive_exclusive" => {
                s.isolation_class = Isolation::Exclusive;
                r.isolation_class = Isolation::Exclusive;
                s.contract.flags.extend([Flag::Bg, Flag::Bgok]);
                r.contract.flags.extend([Flag::Bg, Flag::Bgok]);
            }
            "exclusive_policy" | "exclusive_bgok" | "exclusive_running_bg" => {
                s.isolation_class = Isolation::Exclusive;
                s.contract.flags.push(Flag::Bgok);
                r.contract.flags.push(Flag::Bg);
                if name == "exclusive_policy" {
                    pol.allow_background = false;
                }
                if name == "exclusive_bgok" {
                    s.contract.flags.clear();
                }
                if name == "exclusive_running_bg" {
                    r.contract.flags.clear();
                }
            }
            "running_exclusive_policy" | "running_exclusive_bg" | "running_exclusive_bgok" => {
                r.isolation_class = Isolation::Exclusive;
                r.contract.flags.push(Flag::Bgok);
                s.contract.flags.push(Flag::Bg);
                if name == "running_exclusive_policy" {
                    pol.allow_background = false;
                }
                if name == "running_exclusive_bg" {
                    s.contract.flags.clear();
                }
                if name == "running_exclusive_bgok" {
                    r.contract.flags.clear();
                }
            }
            "clean_candidate" => {
                s.contract.flags.push(Flag::Cleantree);
                r.contract.scratch.push("src".into());
            }
            "clean_running" => {
                r.contract.flags.push(Flag::Cleantree);
                s.contract.scratch.push("src".into());
            }
            "lock" => {
                s.contract.locks.push("lock".into());
                r.contract.locks.push("lock".into());
            }
            "cores" => {
                pol.cores = 1.5;
            }
            "memory" => {
                pol.mem_gib = 1.5;
            }
            "guard_candidate" => {
                s.contract.flags.push(Flag::Memguard);
                r.contract.mem_gib = 13.0;
            }
            "guard_running" => {
                r.contract.flags.push(Flag::Memguard);
                s.contract.mem_gib = 13.0;
            }
            "live" => {
                s.contract.flags.push(Flag::Memguard);
            }
            _ => unreachable!(),
        }
        let low = || Ok(10.999);
        let high = || Ok(11.0);
        assert!(
            !can_start(
                &s,
                &[&r],
                &pol,
                if name == "live" { Some(&low) } else { None }
            )
            .unwrap(),
            "refusal {name}"
        );
        match name {
            "solo_candidate" => s.isolation_class = Isolation::Normal,
            "solo_running" => r.isolation_class = Isolation::Normal,
            "budgeted_exclusive" => r.isolation_class = Isolation::Normal,
            "budgeted_heavy" => r.contract.flags.clear(),
            "heavy_budgeted" => s.contract.flags.clear(),
            "exclusive_exclusive" => r.isolation_class = Isolation::Normal,
            "exclusive_policy" | "running_exclusive_policy" => pol.allow_background = true,
            "exclusive_bgok" => s.contract.flags.push(Flag::Bgok),
            "exclusive_running_bg" => r.contract.flags.push(Flag::Bg),
            "running_exclusive_bg" => s.contract.flags.push(Flag::Bg),
            "running_exclusive_bgok" => r.contract.flags.push(Flag::Bgok),
            "clean_candidate" => r.contract.scratch.clear(),
            "clean_running" => s.contract.scratch.clear(),
            "lock" => r.contract.locks.clear(),
            "cores" => pol.cores = 2.0,
            "memory" => pol.mem_gib = 2.0,
            "guard_candidate" => r.contract.mem_gib = 12.0,
            "guard_running" => s.contract.mem_gib = 12.0,
            "live" => {}
            _ => unreachable!(),
        }
        assert!(
            can_start(
                &s,
                &[&r],
                &pol,
                if name == "live" { Some(&high) } else { None }
            )
            .unwrap(),
            "control {name}"
        );
        println!("admission pair {name}: refused=false control=true");
    }
    let mut guarded = stage("guard");
    guarded.contract.flags.push(Flag::Memguard);
    assert!(
        can_start(
            &guarded,
            &[&ordinary],
            &p,
            Some(&|| Err("sample failed".into()))
        )
        .unwrap_err()
        .contains("sample failed")
    );
}

#[test]
fn selection_preserves_priority_and_reservation() {
    let mut a = stage("a");
    a.contract.flags.push(Flag::Heavy);
    let mut b = stage("b");
    b.isolation_class = Isolation::Budgeted;
    b.contract.tier = 0;
    let mut c = stage("c");
    c.isolation_class = Isolation::Exclusive;
    c.contract.tier = 0;
    let g = graph(vec![a, b, c]);
    assert_eq!(
        g.select(&BTreeSet::new(), &[], &BTreeSet::new(), &policy(), None)
            .unwrap(),
        vec!["a"]
    );
    let done = BTreeSet::from(["a".into()]);
    assert_eq!(
        g.select(&done, &[], &done, &policy(), None).unwrap(),
        vec!["b"]
    );
    let a = stage("a");
    let mut b = stage("b");
    b.contract.estimated_seconds = 10.0;
    let mut z = stage("z");
    z.contract.tier = 0;
    let ranked = graph(vec![a, b, z]);
    assert_eq!(
        ranked
            .select(&BTreeSet::new(), &[], &BTreeSet::new(), &policy(), None)
            .unwrap(),
        vec!["z", "b", "a"]
    );
    let mut first = stage("a");
    first.isolation_class = Isolation::Solo;
    let mut second = stage("b");
    second.isolation_class = Isolation::Exclusive;
    second.contract.flags.push(Flag::Bgok);
    let mut r = stage("r");
    r.contract.flags.push(Flag::Bg);
    let g = graph(vec![first, second, r]);
    assert_eq!(
        g.select(
            &BTreeSet::new(),
            &["r".into()],
            &BTreeSet::from(["r".into()]),
            &policy(),
            None
        )
        .unwrap(),
        Vec::<String>::new()
    );
    assert_eq!(
        g.select(
            &BTreeSet::from(["r".into()]),
            &[],
            &BTreeSet::from(["r".into()]),
            &policy(),
            None
        )
        .unwrap(),
        vec!["a"]
    );
    let mut a = stage("a");
    a.contract.estimated_seconds = 10.0;
    let mut b = stage("b");
    b.contract.tier = 0;
    let mut c = stage("c");
    c.dependencies = vec!["a".into()];
    let g = graph(vec![a, b, c]);
    assert_eq!(
        g.select(&BTreeSet::new(), &[], &BTreeSet::new(), &policy(), None)
            .unwrap(),
        vec!["b", "a"]
    );
    assert!(
        g.select(
            &BTreeSet::new(),
            &[],
            &BTreeSet::from(["a".into(), "b".into()]),
            &policy(),
            None
        )
        .unwrap()
        .is_empty()
    );
}

#[derive(Default)]
struct Fixture {
    initial_clock: Cell<Option<f64>>,
    fail_wait: bool,
    terminated: usize,
    clock: f64,
    events: Vec<(f64, String)>,
    outcomes: BTreeMap<String, i32>,
    starts: Vec<(String, usize, Vec<String>, Bounds)>,
}
impl Executor for Fixture {
    fn now(&self) -> f64 {
        self.initial_clock.take().unwrap_or(self.clock)
    }
    fn start(&mut self, s: &Stage, lane: usize) -> Result<(), String> {
        self.events.push((
            self.clock + s.contract.estimated_seconds,
            s.contract.label.clone(),
        ));
        self.starts.push((
            s.contract.label.clone(),
            lane,
            s.argv.clone(),
            s.contract.bounds.clone(),
        ));
        Ok(())
    }
    fn terminate(&mut self) -> Result<(), String> {
        self.terminated += 1;
        self.events.clear();
        Ok(())
    }
    fn wait_any(&mut self) -> Result<Completion, String> {
        if self.fail_wait {
            return Err("wait custody error".into());
        }
        self.events
            .sort_by(|a, b| a.0.total_cmp(&b.0).then(a.1.cmp(&b.1)));
        let (end, label) = self.events.remove(0);
        self.clock = end;
        Ok(Completion {
            exit: *self.outcomes.get(&label).unwrap_or(&0),
            label,
            end,
        })
    }
}

#[test]
fn executor_deadline_failure_drain_and_accounting() {
    let a = stage("a");
    let b = stage("b");
    let mut c = stage("c");
    c.dependencies = vec!["a".into()];
    let g = graph(vec![a, b, c]);
    let mut p = policy();
    p.deadline = Some(1.0);
    let mut e = Fixture::default();
    let result = run(&g, &p, &mut e, None).unwrap();
    assert_eq!(
        (
            result.status,
            result.not_run,
            result.wall_s,
            result.stages.len()
        ),
        (3, vec![], 4.0, 3)
    );
    assert_eq!(
        result
            .critical_path
            .iter()
            .map(|h| h.label.as_str())
            .collect::<Vec<_>>(),
        ["a", "c"]
    );
    p.stop_at_deadline = true;
    let mut e = Fixture::default();
    let result = run(&g, &p, &mut e, None).unwrap();
    assert_eq!(
        (result.status, result.not_run, result.stages.len()),
        (3, vec!["c".to_string()], 2)
    );
    p.stop_at_deadline = false;
    let mut e = Fixture {
        outcomes: BTreeMap::from([("a".into(), 7)]),
        ..Fixture::default()
    };
    let result = run(&g, &p, &mut e, None).unwrap();
    assert_eq!(
        (
            result.status,
            result.not_run,
            result.failed,
            result.stages.len()
        ),
        (1, vec!["c".into()], vec!["a".into()], 2)
    );
    assert!(e.events.is_empty());
    assert_eq!(result.stages["a"].exit, Some(7));
    assert_eq!(result.stages["b"].exit, Some(0));
    let mut independent = g.stages.values().cloned().collect::<Vec<_>>();
    independent.push(stage("d"));
    p.cores = 2.0;
    let mut e = Fixture {
        outcomes: BTreeMap::from([("a".into(), 7)]),
        ..Fixture::default()
    };
    let result = run(&graph(independent), &p, &mut e, None).unwrap();
    assert_eq!(
        (result.status, result.not_run, result.stages.len()),
        (1, vec!["c".into(), "d".into()], 2)
    );
    assert!(e.events.is_empty());
    let g = graph(vec![]);
    let result = run(&g, &policy(), &mut Fixture::default(), None).unwrap();
    assert_eq!(
        (
            result.status,
            result.stages.len(),
            result.critical_path.len()
        ),
        (0, 0, 0)
    );
}

#[test]
fn serial_parallel_preserve_argv_and_bounds() {
    let config = crate::cycle::model::PlanConfig {
        binary: "/fixture/gomad".into(),
        config_path: "/fixture/config".into(),
        config_sha256: "a".repeat(64),
    };
    let mut observed = vec![];
    for mode in [Mode::Serial, Mode::Parallel] {
        let plan = crate::cycle::plan::derive(&config, mode).unwrap();
        let g = Graph::new(&plan).unwrap();
        let mut e = Fixture::default();
        let r = run(&g, &policy(), &mut e, None).unwrap();
        assert_eq!(r.status, 0);
        assert_eq!(r.stages.len(), 96);
        assert!(r.not_run.is_empty());
        if mode == Mode::Serial {
            assert!(e.starts.iter().all(|(_, lane, _, _)| *lane == 1));
        }
        observed.push(
            e.starts
                .into_iter()
                .map(|(label, _, argv, bounds)| (label, (argv, bounds)))
                .collect::<BTreeMap<_, _>>(),
        );
    }
    assert_eq!(observed[0], observed[1]);
}

#[test]
fn memory_sampling_and_critical_resource_path() {
    assert_eq!(
        parse_mem_available("MemTotal: 22 kB\nMemAvailable: 12582912 kB\n").unwrap(),
        12.0
    );
    assert!(
        parse_mem_available("MemFree: 12582912 kB\n")
            .unwrap_err()
            .contains("MemAvailable missing")
    );
    assert!(parse_mem_available("MemAvailable: nope kB\n").is_err());
    for malformed in [
        "MemAvailable: 1 kB\nMemAvailable: 2 kB\n",
        "MemAvailable: +1 kB\n",
        "MemAvailable: -1 kB\n",
        "MemAvailable: 18446744073709551616 kB\n",
        "MemAvailable: 2 MB\n",
    ] {
        assert!(parse_mem_available(malformed).is_err(), "{malformed}");
    }
    assert_eq!(parse_mem_available("MemAvailable: 0 kB\n").unwrap(), 0.0);
    assert!(mem_available_gib().unwrap() > 0.0);
    let a = stage("a");
    let b = stage("b");
    let g = graph(vec![a, b]);
    let mut p = policy();
    p.cores = 1.0;
    let r = run(&g, &p, &mut Fixture::default(), None).unwrap();
    assert_eq!(
        r.critical_path
            .iter()
            .map(|h| (h.label.as_str(), h.via.as_str()))
            .collect::<Vec<_>>(),
        [("a", "start"), ("b", "resource")]
    );
}

#[test]
fn incomplete_and_executor_errors_do_not_fabricate_success() {
    let mut s = stage("blocked");
    s.isolation_class = Isolation::Budgeted;
    s.contract.flags.push(Flag::Heavy);
    let r = run(&graph(vec![s]), &policy(), &mut Fixture::default(), None).unwrap();
    assert_eq!(
        (r.status, r.not_run, r.stages.len()),
        (1, vec!["blocked".into()], 0)
    );
    let mut e = Fixture {
        fail_wait: true,
        ..Fixture::default()
    };
    assert_eq!(
        run(
            &graph(vec![stage("a"), stage("b")]),
            &policy(),
            &mut e,
            None
        )
        .unwrap_err(),
        "wait custody error"
    );
    assert!(e.events.is_empty());
    assert_eq!(e.terminated, 1);
    let mut e = Fixture {
        clock: 5.0,
        initial_clock: Cell::new(Some(0.0)),
        ..Fixture::default()
    };
    let r = run(&graph(vec![stage("a")]), &policy(), &mut e, None).unwrap();
    assert_eq!(
        (r.critical_path[0].start, r.critical_path[0].via.as_str()),
        (5.0, "start")
    );
}
