//! Bounded diagnostic commit ordering, never part of canonical replay evidence.
//! The helper serializes modeled effects; the mutex also serializes their
//! diagnostic receipts. Missing context or dropped frames cannot complete a log.
use serde::Serialize;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::cell::RefCell;
use std::collections::BTreeMap;
use std::io::Write;
use std::sync::{Mutex, OnceLock};

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
    recorder()
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
        .owners
        .insert((context.instance.clone(), context.incarnation), context);
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

fn recorder() -> &'static Mutex<Recorder<std::io::Stderr>> {
    static RECORDER: OnceLock<Mutex<Recorder<std::io::Stderr>>> = OnceLock::new();
    RECORDER.get_or_init(|| {
        Mutex::new(Recorder::new(
            std::io::stderr(),
            std::env::var("GOMAD_WASM_EVENT_ORDER").is_ok_and(|value| value == "1"),
            BYTE_LIMIT,
            EVENT_LIMIT,
        ))
    })
}

pub fn emit(context: &Context, kind: &str, ordinal: u64, value: Value) {
    let mut recorder = recorder()
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    recorder.owners.insert(
        (context.instance.clone(), context.incarnation),
        context.clone(),
    );
    recorder.event(context, kind, ordinal, value);
}
pub fn latest(context: &Context) -> Context {
    recorder()
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
        .owners
        .get(&(context.instance.clone(), context.incarnation))
        .cloned()
        .unwrap_or_else(|| context.clone())
}
pub fn emit_current(kind: &str, ordinal: u64, time_nanos: i64, value: Value) {
    let context = CURRENT.with(|current| current.borrow().clone());
    let Some(mut context) = context else {
        recorder()
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .incomplete("model commit without owner context");
        return;
    };
    context.time_nanos = time_nanos;
    emit(&context, kind, ordinal, value);
}
pub fn emit_current_context(kind: &str, ordinal: u64, value: Value) {
    let context = CURRENT.with(|current| current.borrow().clone());
    let Some(context) = context else {
        recorder()
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .incomplete("model commit without owner context");
        return;
    };
    emit(&context, kind, ordinal, value);
}
pub fn owner(instance: &str, incarnation: u64, time_nanos: i64) -> Context {
    let recorder = recorder()
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
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
}
pub fn boundary(context: &Context, inspection_order: u64) {
    let mut recorder = recorder()
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    recorder.boundary(context, inspection_order);
}
pub fn bytes_value(bytes: &[u8]) -> Value {
    json!({"bytes":bytes.len(),"sha256":format!("{:x}",Sha256::digest(bytes))})
}
pub fn finish() {
    recorder()
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
        .finish();
}
pub fn is_enabled() -> bool {
    matches!(
        recorder()
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .state,
        State::Pending | State::Recording
    )
}
pub fn has_context() -> bool {
    CURRENT.with(|current| current.borrow().is_some())
}
pub fn incomplete(reason: &str) {
    recorder()
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
        .incomplete(reason);
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

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
