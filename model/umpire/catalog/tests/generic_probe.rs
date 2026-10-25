//! The shared engine owns lazy metadata, current-context admission and correlation.
#![forbid(unsafe_code)]

use std::sync::{
    Arc,
    atomic::{AtomicBool, AtomicUsize, Ordering},
};
use umpire_catalog::generic_probe::{Engine, ProbeSystem, Request, Result};
use umpire_catalog::{Expectation, Expected, Violation, WireCase, WireField, WireKind};

struct System {
    calls: Arc<AtomicUsize>,
    registered: Arc<AtomicBool>,
}
impl ProbeSystem for System {
    type Method = ();
    type Context = u8;
    fn methods(&self) -> Vec<((), &'static str)> {
        vec![((), "ready")]
    }
    fn name(&self, (): ()) -> &'static str {
        "Inspect"
    }
    fn rpc_path(&self, (): ()) -> String {
        "/example.Service/Inspect".into()
    }
    fn fields(&self, (): ()) -> Vec<WireField> {
        vec![WireField {
            path: "example.Request.value".into(),
            declaration: "example.Request.value".into(),
            kind: WireKind::String,
            optional: false,
            refusal: None,
        }]
    }
    fn bounds(&self) -> Vec<u64> {
        vec![8]
    }
    fn canonical_context(&self) -> u8 {
        1
    }
    fn baseline(&self, (): (), context: &u8) -> Result<Vec<u8>> {
        self.calls.fetch_add(1, Ordering::Relaxed);
        Ok(vec![*context])
    }
    fn mutate(&self, (): (), _: &WireCase, _: &[u8]) -> Result<Vec<u8>> {
        Ok(vec![0])
    }
    fn prediction(&self, (): (), finding: Violation) -> Result<Violation> {
        if !self.registered.load(Ordering::Relaxed) {
            return Err("unregistered strict claim".into());
        }
        Ok(finding)
    }
    fn findings(&self, (): (), bytes: &[u8]) -> Result<Vec<Violation>> {
        Ok(if bytes == [0] || bytes == [4] {
            vec![Violation {
                path: "example.Request.value".into(),
                constraint: "required".into(),
                fields: vec![],
                expected: Some(Expected {
                    expectation: Expectation::RejectsCode(umpire_catalog::Code::InvalidArgument),
                }),
            }]
        } else {
            vec![]
        })
    }
}

#[test]
fn lazy_canonical_metadata_never_replaces_current_request_validation() -> Result<()> {
    let calls = Arc::new(AtomicUsize::new(0));
    let engine =
        Engine::new(System { calls: calls.clone(), registered: Arc::new(AtomicBool::new(true)) })?;
    assert_eq!(calls.load(Ordering::Relaxed), 0);
    let id = "generic::Inspect::example.Request.value:mutation.empty";
    for context in [2, 3] {
        let prepared = engine.run(&Request::Prepare { id, baseline: false, context: &context })?;
        assert_eq!(prepared.message, [0]);
        assert_eq!(prepared.cases[0].predictions.len(), 1);
    }
    assert_eq!(calls.load(Ordering::Relaxed), 3, "one canonical and two actual baselines");
    assert!(engine.run(&Request::Prepare { id, baseline: false, context: &4 }).is_err());
    assert!(engine.run(&Request::Prepare { id: "unknown", baseline: true, context: &1 }).is_err());
    assert!(engine.run(&Request::Classify(&[])).is_err());
    Ok(())
}

#[test]
fn failed_canonical_initialization_is_not_published() -> Result<()> {
    let calls = Arc::new(AtomicUsize::new(0));
    let registered = Arc::new(AtomicBool::new(false));
    let engine = Engine::new(System { calls: calls.clone(), registered: registered.clone() })?;
    assert_eq!(
        engine.run(&Request::Inventory).err().ok_or("missing registry refusal")?.to_string(),
        "unregistered strict claim"
    );
    assert_eq!(calls.load(Ordering::Relaxed), 1);
    registered.store(true, Ordering::Relaxed);
    let inventory = engine.run(&Request::Inventory)?;
    let prepared = calls.load(Ordering::Relaxed);
    assert_eq!(prepared, 1 + inventory.cases.iter().filter(|case| case.applicability == 0).count());
    assert_eq!(engine.run(&Request::Inventory)?, inventory);
    assert_eq!(calls.load(Ordering::Relaxed), prepared);
    Ok(())
}
