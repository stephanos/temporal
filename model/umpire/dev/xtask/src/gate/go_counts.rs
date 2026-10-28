//! Go list and test JSON observations for the existing stage count authority.
use std::io;

#[derive(Debug, Eq, PartialEq)]
pub(super) struct Totals {
    pub(super) discovered: u64,
    pub(super) executed: u64,
    pub(super) ignored: u64,
    pub(super) skipped: u64,
}

#[derive(Debug)]
pub(super) struct Inventory;

impl Inventory {
    pub(super) fn parse(_bytes: &[u8]) -> io::Result<Self> {
        Err(io::Error::other("Go inventory parser not implemented"))
    }

    pub(super) fn observe(&self, _bytes: &[u8]) -> io::Result<Totals> {
        Err(io::Error::other("Go execution parser not implemented"))
    }
}

pub(super) fn output(_bytes: &[u8]) -> io::Result<Vec<u8>> {
    Err(io::Error::other("Go output decoder not implemented"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::{Value, json};

    fn stream(events: &[Value]) -> Vec<u8> {
        events
            .iter()
            .flat_map(|event| {
                let mut bytes = serde_json::to_vec(event).unwrap_or_default();
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
        assert_eq!(
            inventory.observe(&stream(&run()))?,
            Totals { discovered: 4, executed: 3, ignored: 0, skipped: 1 }
        );
        Ok(())
    }

    #[test]
    fn explicit_no_test_package_proves_zero_but_empty_stream_does_not() -> io::Result<()> {
        let empty = stream(&[
            json!({"Action":"start","Package":"empty"}),
            json!({"Action":"skip","Package":"empty"}),
        ]);
        assert_eq!(
            Inventory::parse(&empty)?.observe(&empty)?,
            Totals { discovered: 0, executed: 0, ignored: 0, skipped: 0 }
        );
        assert!(Inventory::parse(b"").is_err());
        Ok(())
    }

    #[test]
    fn completed_failed_tests_retain_exact_totals() -> io::Result<()> {
        let inventory = Inventory::parse(&stream(&list()))?;
        let mut failed = run();
        failed[11]["Action"] = "fail".into();
        failed[18]["Action"] = "fail".into();
        assert_eq!(
            inventory.observe(&stream(&failed))?,
            Totals { discovered: 4, executed: 3, ignored: 0, skipped: 1 }
        );
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
                    events[8]["Test"] = "TestSame/absent/grandchild".into();
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
        assert!(inventory.observe(&stream(&events)).is_ok());
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
        assert_eq!(inventory.observe(&stream(&events))?.discovered, 4);
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
        }
        let oversized =
            stream(&[json!({"Action":"output","Package":"p","Output":"x".repeat(1024*1024+1)})]);
        assert!(Inventory::parse(&oversized).is_err());
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
