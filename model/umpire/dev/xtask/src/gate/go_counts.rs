//! Go list and test JSON observations for the existing stage count authority.
use serde::{Deserialize, Deserializer};
use std::{
    collections::{BTreeMap, BTreeSet},
    io,
};

const MAX_RECORD_BYTES: usize = 1024 * 1024;
const MAX_STREAM_BYTES: usize = 64 * 1024 * 1024;
const MAX_EVENTS: usize = 1_000_000;

#[derive(Debug, Eq, PartialEq)]
pub(super) struct Totals {
    pub(super) discovered: u64,
    pub(super) executed: u64,
    pub(super) ignored: u64,
    pub(super) skipped: u64,
}

#[derive(Debug, Eq, PartialEq)]
pub(super) struct Observation {
    pub(super) totals: Totals,
    pub(super) failed: bool,
}

fn invalid(message: &str) -> io::Error {
    io::Error::other(message)
}

fn present<'de, D: Deserializer<'de>, T: Deserialize<'de>>(
    input: D,
) -> Result<Option<T>, D::Error> {
    T::deserialize(input).map(Some)
}

#[derive(Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
struct Event {
    action: String,
    #[serde(default, deserialize_with = "present")]
    package: Option<String>,
    #[serde(default, deserialize_with = "present")]
    test: Option<String>,
    #[serde(default, deserialize_with = "present")]
    import_path: Option<String>,
    #[serde(default, deserialize_with = "present")]
    output: Option<String>,
    #[serde(default, deserialize_with = "present")]
    output_type: Option<String>,
    #[serde(default, deserialize_with = "present")]
    failed_build: Option<String>,
    #[serde(default, deserialize_with = "present")]
    elapsed: Option<f64>,
    #[serde(default, deserialize_with = "present")]
    time: Option<String>,
}

impl Event {
    fn build(&self) -> bool {
        self.action == "build-output" || self.action == "build-fail"
    }

    fn validate(&self) -> io::Result<()> {
        if self.elapsed.is_some_and(|elapsed| !elapsed.is_finite() || elapsed < 0.0) {
            return Err(invalid("invalid Go event elapsed time"));
        }
        if self.build() {
            if self.import_path.as_ref().is_none_or(String::is_empty)
                || self.package.is_some()
                || self.test.is_some()
                || self.failed_build.is_some()
                || self.elapsed.is_some()
                || self.time.is_some()
                || self.output_type.is_some()
                || (self.action == "build-output") != self.output.is_some()
            {
                return Err(invalid("invalid Go build event"));
            }
            return Ok(());
        }
        if !["start", "run", "pause", "cont", "pass", "fail", "output", "skip", "bench"]
            .contains(&self.action.as_str())
            || self.package.as_ref().is_none_or(String::is_empty)
            || self.import_path.is_some()
            || self.test.as_ref().is_some_and(String::is_empty)
            || (self.action == "output") != self.output.is_some()
            || self.output_type.as_deref().is_some_and(|kind| {
                self.action != "output" || !["", "frame", "error", "error-continue"].contains(&kind)
            })
            || self
                .failed_build
                .as_ref()
                .is_some_and(|name| name.is_empty() || self.action != "fail" || self.test.is_some())
            || (self.action == "start" && self.test.is_some())
            || (["run", "pause", "cont", "bench"].contains(&self.action.as_str())
                && self.test.is_none())
        {
            return Err(invalid("invalid Go test event"));
        }
        Ok(())
    }
}

fn events(bytes: &[u8]) -> io::Result<Vec<Event>> {
    if bytes.is_empty() || bytes.len() > MAX_STREAM_BYTES || bytes.last() != Some(&b'\n') {
        return Err(invalid("empty, oversized or truncated Go JSON stream"));
    }
    bytes
        .split_inclusive(|byte| *byte == b'\n')
        .enumerate()
        .map(|(index, line)| {
            if index >= MAX_EVENTS {
                return Err(invalid("Go JSON event count exceeds bound"));
            }
            if line.len() > MAX_RECORD_BYTES {
                return Err(invalid("Go JSON record exceeds bound"));
            }
            let event: Event = serde_json::from_slice(line)?;
            event.validate()?;
            Ok(event)
        })
        .collect()
}

#[derive(Debug, Default)]
struct PackageInventory {
    units: BTreeSet<String>,
    listed: BTreeSet<String>,
    pending: String,
    finished: bool,
}

impl PackageInventory {
    fn output(&mut self, package: &str, output: &str) -> io::Result<()> {
        self.pending.push_str(output);
        while let Some(end) = self.pending.find('\n') {
            if end > MAX_RECORD_BYTES {
                return Err(invalid("Go listing line exceeds bound"));
            }
            let line: String = self.pending.drain(..=end).collect();
            let line = line.trim_end_matches(['\r', '\n']);
            if let Some(kind) = ["Test", "Example", "Fuzz", "Benchmark"]
                .into_iter()
                .find(|prefix| line.starts_with(prefix))
            {
                if line[kind.len()..].chars().next().is_some_and(char::is_lowercase)
                    || !line
                        .chars()
                        .all(|character| character.is_alphanumeric() || character == '_')
                    || !self.listed.insert(line.to_owned())
                {
                    return Err(invalid("invalid or duplicate Go listing identity"));
                }
                if kind != "Benchmark" {
                    self.units.insert(line.to_owned());
                }
            } else {
                let fields: Vec<_> = line.split_whitespace().collect();
                if fields.len() < 3
                    || !["ok", "?"].contains(&fields[0])
                    || fields[1] != package
                    || fields.contains(&"(cached)")
                {
                    return Err(invalid("unexpected Go listing output"));
                }
            }
        }
        if self.pending.len() > MAX_RECORD_BYTES {
            return Err(invalid("Go listing line exceeds bound"));
        }
        Ok(())
    }
}

#[derive(Debug)]
pub(super) struct Inventory {
    packages: BTreeMap<String, PackageInventory>,
}

impl Inventory {
    pub(super) fn parse(bytes: &[u8]) -> io::Result<Self> {
        let mut packages = BTreeMap::<String, PackageInventory>::new();
        for event in events(bytes)? {
            if event.action == "build-output" {
                continue;
            }
            if event.build() || event.failed_build.is_some() {
                return Err(invalid("Go inventory build failed"));
            }
            let name = event.package.as_deref().ok_or_else(|| invalid("missing Go package"))?;
            if event.action == "start" {
                if packages.insert(name.to_owned(), PackageInventory::default()).is_some() {
                    return Err(invalid("duplicate Go inventory package"));
                }
                continue;
            }
            let package = packages
                .get_mut(name)
                .ok_or_else(|| invalid("Go inventory event before package start"))?;
            if package.finished || event.test.is_some() {
                return Err(invalid("unexpected Go inventory lifecycle"));
            }
            match event.action.as_str() {
                "output" => package.output(
                    name,
                    event.output.as_deref().ok_or_else(|| invalid("missing Go listing output"))?,
                )?,
                "pass" | "skip" => {
                    if !package.pending.is_empty()
                        || (event.action == "skip" && !package.listed.is_empty())
                    {
                        return Err(invalid("incomplete Go inventory package"));
                    }
                    package.finished = true;
                }
                _ => return Err(invalid("Go inventory did not complete listing")),
            }
        }
        if packages.is_empty() || packages.values().any(|package| !package.finished) {
            return Err(invalid("missing Go inventory package completion"));
        }
        Ok(Self { packages })
    }

    pub(super) fn observe(&self, bytes: &[u8]) -> io::Result<Observation> {
        let mut failed = false;
        let mut packages = BTreeMap::<String, PackageRun>::new();
        for event in events(bytes)? {
            if event.action == "build-output" {
                continue;
            }
            if event.build() || event.failed_build.is_some() {
                return Err(invalid("Go execution build failed"));
            }
            failed |= event.action == "fail";
            let name = event.package.as_deref().ok_or_else(|| invalid("missing Go package"))?;
            let inventory =
                self.packages.get(name).ok_or_else(|| invalid("unknown Go execution package"))?;
            if event.action == "start" {
                if packages.insert(name.to_owned(), PackageRun::default()).is_some() {
                    return Err(invalid("duplicate Go execution package"));
                }
                continue;
            }
            let package = packages
                .get_mut(name)
                .ok_or_else(|| invalid("Go execution event before package start"))?;
            if package.finished {
                return Err(invalid("Go event after package completion"));
            }
            if let Some(test) = &event.test {
                let root = test.split_once('/').map_or(test.as_str(), |(root, _)| root);
                if !inventory.units.contains(root) {
                    return Err(invalid("unknown Go execution test identity"));
                }
                package.test(test, &event.action)?;
            } else {
                match event.action.as_str() {
                    "output" => (),
                    "pass" | "fail" | "skip" => {
                        if inventory
                            .units
                            .iter()
                            .any(|name| !matches!(package.tests.get(name), Some(Phase::Done(_))))
                            || package.tests.values().any(|phase| !matches!(phase, Phase::Done(_)))
                            || (event.action == "skip" && !package.tests.is_empty())
                            || (event.action != "fail"
                                && package
                                    .tests
                                    .values()
                                    .any(|phase| *phase == Phase::Done(Terminal::Fail)))
                        {
                            return Err(invalid(
                                "incomplete or inconsistent Go package completion",
                            ));
                        }
                        package.finished = true;
                    }
                    _ => return Err(invalid("invalid Go package execution action")),
                }
            }
        }
        if packages.len() != self.packages.len()
            || packages.values().any(|package| !package.finished)
        {
            return Err(invalid("missing Go execution package completion"));
        }
        let mut totals = Totals { discovered: 0, executed: 0, ignored: 0, skipped: 0 };
        for (name, inventory) in &self.packages {
            let package =
                packages.get(name).ok_or_else(|| invalid("missing Go execution package"))?;
            for unit in &inventory.units {
                increment(&mut totals.discovered)?;
                match package.tests.get(unit) {
                    Some(Phase::Done(Terminal::Pass | Terminal::Fail)) => {
                        increment(&mut totals.executed)?
                    }
                    Some(Phase::Done(Terminal::Skip)) => increment(&mut totals.skipped)?,
                    _ => return Err(invalid("missing Go test completion")),
                }
            }
        }
        Ok(Observation { totals, failed })
    }
}

fn increment(total: &mut u64) -> io::Result<()> {
    *total = total.checked_add(1).ok_or_else(|| invalid("Go test total overflow"))?;
    Ok(())
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum Terminal {
    Pass,
    Fail,
    Skip,
}
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum Phase {
    Running,
    Paused,
    Done(Terminal),
}
#[derive(Default)]
struct PackageRun {
    tests: BTreeMap<String, Phase>,
    finished: bool,
}

impl PackageRun {
    fn test(&mut self, name: &str, action: &str) -> io::Result<()> {
        if action == "run" {
            if let Some((root, _)) = name.split_once('/')
                && !matches!(self.tests.get(root), Some(Phase::Running | Phase::Paused))
            {
                return Err(invalid("Go subtest has no active root"));
            }
            if self.tests.insert(name.to_owned(), Phase::Running).is_some() {
                return Err(invalid("duplicate Go test run"));
            }
            return Ok(());
        }
        let current = *self.tests.get(name).ok_or_else(|| invalid("Go test event before run"))?;
        let next = match (action, current) {
            ("output", Phase::Running | Phase::Paused) => current,
            ("pause", Phase::Running) => Phase::Paused,
            ("cont", Phase::Paused) => Phase::Running,
            ("pass" | "fail" | "skip", Phase::Running) => {
                let prefix = format!("{name}/");
                for (child, phase) in &self.tests {
                    if !name.contains('/')
                        && child.starts_with(&prefix)
                        && (!matches!(phase, Phase::Done(_))
                            || (action != "fail" && *phase == Phase::Done(Terminal::Fail)))
                    {
                        return Err(invalid("incomplete or inconsistent Go child test"));
                    }
                }
                Phase::Done(match action {
                    "pass" => Terminal::Pass,
                    "fail" => Terminal::Fail,
                    _ => Terminal::Skip,
                })
            }
            _ => return Err(invalid("invalid or duplicate Go test transition")),
        };
        self.tests.insert(name.to_owned(), next);
        Ok(())
    }
}

/// Decode diagnostic payloads independently of count lifecycle completion.
pub(super) fn output(bytes: &[u8]) -> io::Result<Vec<u8>> {
    let mut output = Vec::new();
    for event in events(bytes)? {
        if let Some(payload) = event.output {
            output.extend_from_slice(payload.as_bytes());
        }
    }
    Ok(output)
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::{Value, json};

    fn stream(events: &[Value]) -> Vec<u8> {
        events
            .iter()
            .flat_map(|event| {
                let mut bytes = serde_json::to_vec(event)
                    .unwrap_or_else(|error| panic!("fixture serialization failed: {error}"));
                bytes.push(b'\n');
                bytes
            })
            .collect()
    }

    fn list() -> Vec<Value> {
        vec![
            json!({"Action":"start","Package":"p/a"}),
            json!({"Action":"start","Package":"p/b"}),
            json!({"Action":"output","Package":"p/a","Output":"TestSa"}),
            json!({"Action":"output","Package":"p/b","Output":"TestSame\n"}),
            json!({"Action":"output","Package":"p/a","Output":"me\nExampleValue\nFuzzDecode\nBenchmarkRows\n"}),
            json!({"Action":"output","Package":"p/a","Output":"ok  \tp/a\t0.001s\n"}),
            json!({"Action":"pass","Package":"p/a","Elapsed":0.001}),
            json!({"Action":"pass","Package":"p/b","Elapsed":0.002}),
            json!({"Action":"start","Package":"p/empty"}),
            json!({"Action":"output","Package":"p/empty","Output":"?   \tp/empty\t[no test files]\n"}),
            json!({"Action":"skip","Package":"p/empty"}),
        ]
    }

    fn run() -> Vec<Value> {
        vec![
            json!({"Action":"start","Package":"p/b"}),
            json!({"Action":"start","Package":"p/a"}),
            json!({"Action":"run","Package":"p/b","Test":"TestSame"}),
            json!({"Action":"skip","Package":"p/b","Test":"TestSame"}),
            json!({"Action":"run","Package":"p/a","Test":"TestSame"}),
            json!({"Action":"run","Package":"p/a","Test":"TestSame/child"}),
            json!({"Action":"pause","Package":"p/a","Test":"TestSame/child"}),
            json!({"Action":"cont","Package":"p/a","Test":"TestSame/child"}),
            json!({"Action":"run","Package":"p/a","Test":"TestSame/child/grandchild"}),
            json!({"Action":"pass","Package":"p/a","Test":"TestSame/child/grandchild"}),
            json!({"Action":"pass","Package":"p/a","Test":"TestSame/child"}),
            json!({"Action":"pass","Package":"p/a","Test":"TestSame"}),
            json!({"Action":"run","Package":"p/a","Test":"ExampleValue"}),
            json!({"Action":"pass","Package":"p/a","Test":"ExampleValue"}),
            json!({"Action":"run","Package":"p/a","Test":"FuzzDecode"}),
            json!({"Action":"run","Package":"p/a","Test":"FuzzDecode/seed#0"}),
            json!({"Action":"pass","Package":"p/a","Test":"FuzzDecode/seed#0"}),
            json!({"Action":"pass","Package":"p/a","Test":"FuzzDecode"}),
            json!({"Action":"pass","Package":"p/a"}),
            json!({"Action":"pass","Package":"p/b"}),
            json!({"Action":"start","Package":"p/empty"}),
            json!({"Action":"skip","Package":"p/empty"}),
        ]
    }

    #[test]
    fn top_level_units_match_packages_and_ignore_nested_seed_work() -> io::Result<()> {
        let inventory = Inventory::parse(&stream(&list()))?;
        let observed = inventory.observe(&stream(&run()))?;
        assert!(!observed.failed);
        assert_eq!(observed.totals, Totals { discovered: 4, executed: 3, ignored: 0, skipped: 1 });
        Ok(())
    }

    #[test]
    fn slash_in_a_subtest_name_does_not_invent_an_intermediate_parent() -> io::Result<()> {
        let inventory = Inventory::parse(&stream(&list()))?;
        let mut events = run();
        for event in &mut events {
            let renamed = event
                .get("Test")
                .and_then(Value::as_str)
                .filter(|name| name.starts_with("TestSame/child"))
                .map(|name| name.replacen("TestSame/child", "TestSame/direct//slash", 1));
            if let Some(renamed) = renamed {
                event["Test"] = renamed.into();
            }
        }
        assert_eq!(inventory.observe(&stream(&events))?.totals.executed, 3);
        Ok(())
    }

    #[test]
    fn explicit_no_test_package_proves_zero_but_empty_stream_does_not() -> io::Result<()> {
        let empty = stream(&[
            json!({"Action":"start","Package":"empty"}),
            json!({"Action":"output","Package":"empty","Output":"?   \tempty\t[no test files]\n"}),
            json!({"Action":"skip","Package":"empty"}),
        ]);
        let observed = Inventory::parse(&empty)?.observe(&empty)?;
        assert!(!observed.failed);
        assert_eq!(observed.totals, Totals { discovered: 0, executed: 0, ignored: 0, skipped: 0 });
        assert!(Inventory::parse(b"").is_err());
        Ok(())
    }

    #[test]
    fn completed_failed_tests_retain_exact_totals() -> io::Result<()> {
        let inventory = Inventory::parse(&stream(&list()))?;
        let mut failed = run();
        failed[11]["Action"] = "fail".into();
        failed[18]["Action"] = "fail".into();
        let observed = inventory.observe(&stream(&failed))?;
        assert!(observed.failed);
        assert_eq!(observed.totals, Totals { discovered: 4, executed: 3, ignored: 0, skipped: 1 });
        Ok(())
    }

    #[test]
    fn inventory_refuses_duplicate_partial_failed_and_executing_lists() {
        for mutation in 0..8 {
            let mut events = list();
            match mutation {
                0 => {
                    events.insert(1, events[0].clone());
                }
                1 => {
                    events.insert(
                        5,
                        json!({"Action":"output","Package":"p/a","Output":"TestSame\n"}),
                    );
                }
                2 => {
                    events.remove(6);
                }
                3 => {
                    events[6]["Action"] = "fail".into();
                }
                4 => {
                    events.insert(6, json!({"Action":"run","Package":"p/a","Test":"TestSame"}));
                }
                5 => {
                    events.insert(7, events[6].clone());
                }
                6 => {
                    events[4]["Output"] = "me\nExampleValue\nFuzzDecode\nBenchmarkRows".into();
                }
                _ => {
                    events[0]["Package"] = "".into();
                }
            }
            assert!(Inventory::parse(&stream(&events)).is_err(), "mutation {mutation}");
        }
    }

    #[test]
    fn execution_refuses_missing_unknown_duplicate_and_misordered_events() -> io::Result<()> {
        let inventory = Inventory::parse(&stream(&list()))?;
        for mutation in 0..14 {
            let mut events = run();
            match mutation {
                0 => {
                    events.remove(3);
                }
                1 => {
                    events.insert(4, events[3].clone());
                }
                2 => {
                    events[2]["Test"] = "TestUnknown".into();
                }
                3 => {
                    events[14]["Test"] = "FuzzUnknown".into();
                }
                4 => {
                    events[2]["Package"] = "p/unknown".into();
                }
                5 => {
                    events.insert(2, events[0].clone());
                }
                6 => {
                    events.pop();
                }
                7 => {
                    events.insert(20, events[19].clone());
                }
                8 => {
                    events[5]["Test"] = "TestUnknown/child".into();
                }
                9 => {
                    events
                        .insert(12, json!({"Action":"run","Package":"p/a","Test":"TestSame/late"}));
                }
                10 => {
                    events[6]["Action"] = "cont".into();
                }
                11 => {
                    events[11]["Action"] = "pause".into();
                }
                12 => {
                    events[2]["Test"] = "BenchmarkRows".into();
                }
                _ => {
                    events[18]["Action"] = "skip".into();
                }
            }
            assert!(inventory.observe(&stream(&events)).is_err(), "mutation {mutation}");
        }
        assert!(inventory.observe(b"").is_err());
        Ok(())
    }

    #[test]
    fn package_completion_requires_all_tests_and_children_terminal() -> io::Result<()> {
        let inventory = Inventory::parse(&stream(&list()))?;
        let mut events = run();
        events.swap(9, 11);
        assert!(inventory.observe(&stream(&events)).is_err());
        let mut events = run();
        events[18]["Action"] = "fail".into();
        assert!(inventory.observe(&stream(&events))?.failed);
        let mut events = run();
        events[11]["Action"] = "fail".into();
        assert!(inventory.observe(&stream(&events)).is_err());
        Ok(())
    }

    #[test]
    fn build_events_do_not_invent_test_work_and_failures_refuse_completion() -> io::Result<()> {
        let inventory = Inventory::parse(&stream(&list()))?;
        let mut events = run();
        events.insert(0, json!({"ImportPath":"dependency","Action":"build-output","Output":"compiler diagnostic\n"}));
        assert_eq!(inventory.observe(&stream(&events))?.totals.discovered, 4);
        events.insert(1, json!({"ImportPath":"dependency","Action":"build-fail"}));
        assert!(inventory.observe(&stream(&events)).is_err());
        let mut events = run();
        events[18]["Action"] = "fail".into();
        events[18]["FailedBuild"] = "dependency".into();
        assert!(inventory.observe(&stream(&events)).is_err());
        Ok(())
    }

    #[test]
    fn json_records_must_be_typed_complete_and_bounded() {
        for bytes in [
            b"{\"Action\":\"start\",\"Package\":\"p\"}".as_slice(),
            b"{\n",
            b"\n",
            b"[]\n",
            b"null\n",
            b"{\"Action\":null,\"Package\":\"p\"}\n",
            b"{\"Action\":\"start\",\"Package\":17}\n",
            b"{\"Action\":\"start\",\"Package\":null}\n",
            b"{\"Action\":\"start\",\"Action\":\"start\",\"Package\":\"p\"}\n",
            b"{\"Action\":\"unknown\",\"Package\":\"p\"}\n",
            b"{\"Action\":\"start\",\"Package\":\"p\",\"Elapsed\":-1}\n",
        ] {
            assert!(Inventory::parse(bytes).is_err(), "{bytes:?}");
            assert!(output(bytes).is_err(), "decoder accepted malformed record: {bytes:?}");
        }
        let oversized =
            stream(&[json!({"Action":"output","Package":"p","Output":"x".repeat(1024*1024+1)})]);
        assert!(Inventory::parse(&oversized).is_err());
        assert!(output(&oversized).is_err());
    }

    #[test]
    fn human_output_is_separate_from_completion_metadata() -> io::Result<()> {
        let bytes = stream(&[
            json!({"ImportPath":"dep","Action":"build-output","Output":"compiler\n"}),
            json!({"Action":"start","Package":"p"}),
            json!({"Action":"output","Package":"p","Test":"TestFail","Output":"diagnostic\n","OutputType":"error"}),
            json!({"Action":"fail","Package":"p","FailedBuild":"dep"}),
        ]);
        assert_eq!(output(&bytes)?, b"compiler\ndiagnostic\n");
        Ok(())
    }
}
