//! Bounded diagnostic commit ordering, never part of canonical replay evidence.
//! The helper serializes modeled effects; the mutex also serializes their
//! diagnostic receipts. Missing context or dropped frames cannot complete a log.
use serde::Serialize;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::cell::RefCell;
use std::collections::BTreeMap;
use std::io::Write;
use std::sync::{Arc, Mutex, MutexGuard, OnceLock};

const PREFIX: &str = "gomad-wasm-event-order-v1 ";
const BYTE_LIMIT: usize = 32 * 1024 * 1024;
const EVENT_LIMIT: u64 = 131_072;

#[derive(Clone)]
pub struct Context {
    pub instance: String,
    pub incarnation: u64,
    pub trace: u64,
    pub decision: u64,
    pub time_nanos: i64,
}

thread_local! { static CURRENT: RefCell<Option<Context>> = const { RefCell::new(None) }; }

#[cfg(feature = "determinism-fixture")]
pub(crate) fn current_owner() -> Option<crate::model::inspection::Handle> {
    CURRENT.with(|current| {
        current
            .borrow()
            .as_ref()
            .map(|context| crate::model::inspection::Handle {
                node: context.instance.clone(),
                incarnation: context.incarnation,
            })
    })
}

pub struct Scope(Option<Context>);
impl Drop for Scope {
    fn drop(&mut self) {
        CURRENT.with(|current| {
            current.replace(self.0.take());
        });
    }
}
pub fn enter(context: Context) -> Scope {
    let previous = CURRENT.with(|current| current.replace(Some(context.clone())));
    with_recorder(|recorder| {
        recorder
            .owners
            .insert((context.instance.clone(), context.incarnation), context);
    });
    Scope(previous)
}

#[derive(PartialEq, Eq)]
enum State {
    Disabled,
    Pending,
    Recording,
    Failed,
    Finished,
}

struct Recorder<W> {
    output: W,
    state: State,
    bytes: usize,
    events: u64,
    byte_limit: usize,
    event_limit: u64,
    owners: BTreeMap<(String, u64), Context>,
    linked_attachments: usize,
    linked_closed: bool,
}
impl<W: Write> Recorder<W> {
    fn new(output: W, enabled: bool, byte_limit: usize, event_limit: u64) -> Self {
        Self {
            output,
            state: if enabled {
                State::Pending
            } else {
                State::Disabled
            },
            bytes: 0,
            events: 0,
            byte_limit,
            event_limit,
            owners: BTreeMap::new(),
            linked_attachments: 0,
            linked_closed: false,
        }
    }
    fn write(&mut self, frame: &impl Serialize) -> bool {
        let Ok(json) = serde_json::to_vec(frame) else {
            self.state = State::Failed;
            return false;
        };
        let mut line = PREFIX.as_bytes().to_vec();
        line.extend(json);
        line.push(b'\n');
        if line.len() > self.byte_limit.saturating_sub(self.bytes) {
            self.incomplete("event-order byte capacity");
            return false;
        }
        if self
            .output
            .write_all(&line)
            .and_then(|()| self.output.flush())
            .is_err()
        {
            self.state = State::Failed;
            return false;
        }
        self.bytes += line.len();
        true
    }
    fn start(&mut self) -> bool {
        if !matches!(self.state, State::Pending | State::Recording) {
            return false;
        }
        if self.state == State::Pending {
            self.state = State::Recording;
            return self.write(&json!({"type":"header","schema":"gomad.wasm-event-order/v1"}));
        }
        true
    }
    fn incomplete(&mut self, reason: &str) {
        if !matches!(self.state, State::Pending | State::Recording) {
            return;
        }
        self.state = State::Failed;
        let line = format!("{PREFIX}{}\n", json!({"type":"incomplete","reason":reason}));
        let _ = self
            .output
            .write_all(line.as_bytes())
            .and_then(|()| self.output.flush());
    }
    fn event(&mut self, context: &Context, kind: &str, ordinal: u64, value: Value) {
        if !self.start() {
            return;
        }
        if self.events >= self.event_limit {
            self.incomplete("event-order event capacity");
            return;
        }
        let order = self.events + 1;
        let mut frame = json!({"type":"event","order":order,"kind":kind,"instance":context.instance,"incarnation":context.incarnation,"ordinal":ordinal,"trace":context.trace,"decision":context.decision,"time_nanos":context.time_nanos});
        frame["value"] = value;
        if self.write(&frame) {
            self.events = order;
        }
    }
    fn finish(&mut self) {
        if !self.start() {
            return;
        }
        self.write(&json!({"type":"complete","events":self.events,"bytes":self.bytes}));
        if self.state != State::Failed {
            self.state = State::Finished;
        }
    }
    fn boundary(&mut self, context: &Context, inspection_order: u64) {
        if !self.start() {
            return;
        }
        let order = self.events;
        self.write(&json!({"type":"boundary","order":order,"instance":context.instance,"incarnation":context.incarnation,"trace":context.trace,"decision":context.decision,"time_nanos":context.time_nanos,"inspection_order":inspection_order}));
    }
}

enum Output {
    Stderr(std::io::Stderr),
    Retained(Vec<u8>),
}

impl Write for Output {
    fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
        match self {
            Self::Stderr(output) => output.write(bytes),
            Self::Retained(output) => output.write(bytes),
        }
    }

    fn flush(&mut self) -> std::io::Result<()> {
        match self {
            Self::Stderr(output) => output.flush(),
            Self::Retained(output) => output.flush(),
        }
    }
}

struct LinkedRecorder {
    session: LinkedSession,
    previous: Option<Context>,
}

#[derive(Clone)]
pub(crate) struct LinkedSession(Arc<Mutex<Recorder<Output>>>);

impl LinkedSession {
    pub(crate) fn new() -> Self {
        Self(Arc::new(Mutex::new(Recorder::new(
            Output::Retained(Vec::new()),
            true,
            BYTE_LIMIT,
            EVENT_LIMIT,
        ))))
    }

    fn lock(&self) -> MutexGuard<'_, Recorder<Output>> {
        self.0.lock().unwrap_or_else(|poisoned| {
            let mut recorder = poisoned.into_inner();
            recorder.incomplete("linked event-order recorder poisoned");
            recorder
        })
    }
}

thread_local! {
    static LINKED: RefCell<Option<LinkedRecorder>> = const { RefCell::new(None) };
}

pub(crate) fn start_linked(session: &LinkedSession) -> Result<(), &'static str> {
    LINKED.with(|linked| {
        let mut linked = linked.borrow_mut();
        if linked.is_some() {
            return Err("overlapping linked event-order recorders");
        }
        let mut recorder = session.lock();
        if recorder.linked_closed {
            return Err("closed linked event-order session");
        }
        recorder.linked_attachments = recorder
            .linked_attachments
            .checked_add(1)
            .ok_or("linked event-order attachment overflow")?;
        *linked = Some(LinkedRecorder {
            session: session.clone(),
            previous: CURRENT.with(RefCell::take),
        });
        Ok(())
    })
}

pub(crate) fn finish_linked(complete: bool) -> Option<Vec<u8>> {
    let linked = LINKED.with(|linked| linked.borrow_mut().take())?;
    let mut recorder = linked.session.lock();
    if complete {
        if recorder.linked_attachments == 1 {
            recorder.finish();
        } else {
            recorder.incomplete("linked execution finalized with attached participants");
        }
    } else {
        recorder.incomplete("linked execution abandoned");
    }
    recorder.linked_attachments = recorder.linked_attachments.saturating_sub(1);
    recorder.linked_closed = true;
    recorder.state = State::Finished;
    CURRENT.with(|current| current.replace(linked.previous));
    match &mut recorder.output {
        Output::Retained(bytes) => Some(std::mem::take(bytes)),
        Output::Stderr(_) => None,
    }
}

pub(crate) fn detach_linked() {
    if let Some(linked) = LINKED.with(|linked| linked.borrow_mut().take()) {
        let mut recorder = linked.session.lock();
        recorder.linked_attachments = recorder.linked_attachments.saturating_sub(1);
        CURRENT.with(|current| current.replace(linked.previous));
    }
}

fn recorder() -> &'static Mutex<Recorder<Output>> {
    static RECORDER: OnceLock<Mutex<Recorder<Output>>> = OnceLock::new();
    RECORDER.get_or_init(|| {
        Mutex::new(Recorder::new(
            Output::Stderr(std::io::stderr()),
            std::env::var("GOMAD_WASM_EVENT_ORDER").is_ok_and(|value| value == "1"),
            BYTE_LIMIT,
            EVENT_LIMIT,
        ))
    })
}

fn with_recorder<T>(action: impl FnOnce(&mut Recorder<Output>) -> T) -> T {
    LINKED.with(|linked| {
        if let Some(linked) = linked.borrow_mut().as_mut() {
            return action(&mut linked.session.lock());
        }
        action(
            &mut recorder()
                .lock()
                .unwrap_or_else(std::sync::PoisonError::into_inner),
        )
    })
}

pub fn emit(context: &Context, kind: &str, ordinal: u64, value: Value) {
    with_recorder(|recorder| {
        recorder.owners.insert(
            (context.instance.clone(), context.incarnation),
            context.clone(),
        );
        recorder.event(context, kind, ordinal, value);
    });
}
pub fn latest(context: &Context) -> Context {
    with_recorder(|recorder| {
        recorder
            .owners
            .get(&(context.instance.clone(), context.incarnation))
            .cloned()
            .unwrap_or_else(|| context.clone())
    })
}
pub fn emit_current(kind: &str, ordinal: u64, time_nanos: i64, value: Value) {
    let context = CURRENT.with(|current| current.borrow().clone());
    let Some(mut context) = context else {
        incomplete("model commit without owner context");
        return;
    };
    context.time_nanos = time_nanos;
    emit(&context, kind, ordinal, value);
}
pub fn emit_current_context(kind: &str, ordinal: u64, value: Value) {
    let context = CURRENT.with(|current| current.borrow().clone());
    let Some(context) = context else {
        incomplete("model commit without owner context");
        return;
    };
    emit(&context, kind, ordinal, value);
}
pub fn owner(instance: &str, incarnation: u64, time_nanos: i64) -> Context {
    with_recorder(|recorder| {
        let mut context = recorder
            .owners
            .get(&(instance.into(), incarnation))
            .cloned()
            .unwrap_or(Context {
                instance: instance.into(),
                incarnation,
                trace: 0,
                decision: 0,
                time_nanos,
            });
        context.time_nanos = time_nanos;
        context
    })
}
pub fn boundary(context: &Context, inspection_order: u64) {
    with_recorder(|recorder| recorder.boundary(context, inspection_order));
}
pub fn bytes_value(bytes: &[u8]) -> Value {
    json!({"bytes":bytes.len(),"sha256":format!("{:x}",Sha256::digest(bytes))})
}
pub fn finish() {
    with_recorder(Recorder::finish);
}
pub fn is_enabled() -> bool {
    with_recorder(|recorder| matches!(recorder.state, State::Pending | State::Recording))
}
pub fn has_context() -> bool {
    CURRENT.with(|current| current.borrow().is_some())
}
pub fn incomplete(reason: &str) {
    with_recorder(|recorder| recorder.incomplete(reason));
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    // A poisoned shared-recorder lock is internal state that no supported CLI input can manufacture.
    #[test]
    fn poisoned_linked_session_never_emits_complete() {
        let session = LinkedSession::new();
        start_linked(&session).unwrap();
        let poisoned = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            let _recorder = session.0.lock().unwrap();
            panic!("poison linked event recorder");
        }));
        assert!(poisoned.is_err());
        let output = String::from_utf8(finish_linked(true).unwrap()).unwrap();
        assert!(output.contains("linked event-order recorder poisoned"));
        assert!(!output.contains("\"type\":\"complete\""));
    }

    #[test]
    fn event_order_records_full_values_and_actual_coordinates_in_commit_order() {
        let mut recorder = Recorder::new(Vec::new(), true, 8192, 16);
        let context = Context {
            instance: "b".into(),
            incarnation: 2,
            trace: 13,
            decision: 7,
            time_nanos: 19,
        };
        recorder.event(&context, "output", 3, json!({"fd":1,"data":"before"}));
        recorder.boundary(&context, 91);
        recorder.event(&context, "clock", 8, json!({"timestamp":20}));
        recorder.finish();
        let frames: Vec<serde_json::Value> = std::str::from_utf8(&recorder.output)
            .unwrap()
            .lines()
            .map(|line| serde_json::from_str(line.strip_prefix(PREFIX).unwrap()).unwrap())
            .collect();
        assert_eq!(
            frames[0],
            json!({"type":"header","schema":"gomad.wasm-event-order/v1"})
        );
        assert_eq!(
            frames[1],
            json!({"type":"event","order":1,"kind":"output","instance":"b","incarnation":2,"ordinal":3,"trace":13,"decision":7,"time_nanos":19,"value":{"fd":1,"data":"before"}})
        );
        assert_eq!(
            frames[2],
            json!({"type":"boundary","order":1,"instance":"b","incarnation":2,"trace":13,"decision":7,"time_nanos":19,"inspection_order":91})
        );
        assert_eq!(frames[3]["order"], 2);
        assert_eq!(frames[4]["type"], "complete");
        assert_eq!(frames[4]["events"], 2);
        let counted: usize = recorder
            .output
            .split_inclusive(|byte| *byte == b'\n')
            .take(4)
            .map(<[u8]>::len)
            .sum();
        assert_eq!(frames[4]["bytes"], counted);
    }

    #[test]
    fn event_order_never_marks_a_dropped_or_disabled_stream_complete() {
        let context = Context {
            instance: "a".into(),
            incarnation: 1,
            trace: 0,
            decision: 0,
            time_nanos: 1,
        };
        let mut recorder = Recorder::new(Vec::new(), true, 8192, 1);
        recorder.event(&context, "output", 0, json!("first"));
        recorder.event(&context, "clock", 0, json!(2));
        recorder.finish();
        let output = String::from_utf8(recorder.output).unwrap();
        assert!(output.contains("\"type\":\"incomplete\""));
        assert!(!output.contains("\"type\":\"complete\""));
        let mut disabled = Recorder::new(Vec::new(), false, 8192, 1);
        disabled.event(&context, "clock", 0, json!(2));
        disabled.finish();
        assert_eq!(disabled.output, Vec::<u8>::new());
        let mut byte_bounded = Recorder::new(Vec::new(), true, 80, 16);
        byte_bounded.event(
            &context,
            "output",
            0,
            json!("large enough to exceed the header plus event bound"),
        );
        byte_bounded.finish();
        let output = String::from_utf8(byte_bounded.output).unwrap();
        assert!(output.contains("\"type\":\"incomplete\""));
        assert!(!output.contains("\"type\":\"complete\""));
    }
}
