//! Executable controls for typed annotation expansion.

#![allow(dead_code)]

use std::collections::BTreeMap;
use std::fmt::Write;
use umpire_catalog::{Consequence, Rng, Tag, Timestamp, fill, validate};
use umpire_temporal_annotate::annotate;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Status {
    Ready = 0,
    Done = 2,
}

impl TryFrom<i32> for Status {
    type Error = i32;

    fn try_from(value: i32) -> Result<Self, Self::Error> {
        match value {
            0 => Ok(Self::Ready),
            2 => Ok(Self::Done),
            other => Err(other),
        }
    }
}

#[derive(Clone, Default)]
struct Request {
    id: String,
    count: i32,
    status: i32,
}

#[derive(Clone, Default, Debug, PartialEq, Eq)]
struct WireTime {
    seconds: i64,
    nanos: i32,
}

#[derive(Clone, Default)]
struct Opaque {
    nested: bool,
}

#[derive(Clone)]
enum Choice {
    Text(String),
    Flag(bool),
}

#[derive(Clone, Default)]
struct Response {
    id: String,
    timestamp: Option<WireTime>,
    flag: bool,
    labels: BTreeMap<String, String>,
    opaque: Option<Opaque>,
    choice: Option<Choice>,
}

enum Action {
    Client(Client),
    Worker(Worker),
}

enum Client {
    Start,
    Cancel { id: String },
}

enum Worker {
    Poll,
}

annotate! {
    catalog controls;
    tag Visibility { kinds: [Timestamp]; consequence: EventuallyConsistent(3); }
    enum status: Status = "test.Status" {
        Status::Ready = "READY",
        Status::Done = "DONE",
    }
    message request request: Request = "test.Request" {
        id: Uuid + Decided => Accepts;
        count: Counter[gte = 1] + Decided => Rejects(3, "bad");
        status: Enum(status) + Decided => Accepts;
        rules { required(id) => Accepts; }
    }
    message response response: Response = "test.Response" {
        id: Uuid;
        timestamp: Timestamp[gte = 1] + Visibility via Timestamp (
            |m: &Response| m.timestamp.as_ref().map(|v| Timestamp { seconds: v.seconds, nanos: v.nanos }).unwrap_or_default(),
            |m: &mut Response, v: Timestamp| m.timestamp = Some(WireTime { seconds: v.seconds, nanos: v.nanos })
        );
        flag: Bytes + Ignored("boolean outside the model") opaque;
        labels: Bytes + Ignored("map outside the model") opaque;
        opaque: Bytes + Ignored("nested message outside the model") opaque;
        choice: Bytes + Ignored("oneof outside the model") opaque variants {
            Choice::Text = "text",
            Choice::Flag = "choice_flag",
        };
    }
    service "test.Service" {
        Start: Action("Client::Start");
        Poll: Observed("poll");
        Other: OutOfScope("not represented by this model");
    }
    actions bind: Action {
        Action::Client => {
            Client::Start => Rpc("test.Service.Start"),
            Client::Cancel { .. } => Environmental("cancellation"),
        },
        Action::Worker => { Worker::Poll => Observed("poll"), },
    }
}

#[test]
fn unknown_enum_numbers_are_refused() -> Result<(), Box<dyn std::error::Error>> {
    assert_eq!(controls::status(0)?, Status::Ready);
    assert_eq!(controls::status(2)?, Status::Done);
    assert!(controls::status(1).is_err());
    assert!(controls::status(-1).is_err());
    assert_eq!(controls::status_values(), vec![0, 2]);
    let catalog = controls::request()?;
    let message =
        Request { id: "a13cd9da-521c-48d4-ab9c-6ab174876f25".into(), count: 1, status: 1 };
    assert_eq!(validate(&message, &catalog)[0].constraint, "enum.defined_only");
    Ok(())
}

#[test]
fn typed_projection_fills_and_validates_actual_wrapper_fields()
-> Result<(), Box<dyn std::error::Error>> {
    let catalog = controls::response()?;
    let mut message = Response::default();
    assert_ne!(validate(&message, &catalog), [] as [umpire_catalog::Violation; 0]);
    fill(&mut message, &catalog, &mut Rng::new(5))?;
    assert_eq!(validate(&message, &catalog), []);
    assert!(message.timestamp.is_some());
    message.timestamp = Some(WireTime { seconds: 0, nanos: -1 });
    assert!(validate(&message, &catalog).iter().any(|v| v.constraint == "timestamp.valid"));
    assert_eq!(controls::Visibility.consequence(), Consequence::EventuallyConsistent { polls: 3 });
    Ok(())
}

#[test]
fn ignored_rust_shapes_preserve_their_values() -> Result<(), Box<dyn std::error::Error>> {
    let catalog = controls::response()?;
    let mut message = Response {
        flag: true,
        labels: BTreeMap::from([("key".into(), "value".into())]),
        opaque: Some(Opaque { nested: true }),
        choice: Some(Choice::Flag(true)),
        ..Response::default()
    };
    fill(&mut message, &catalog, &mut Rng::new(9))?;
    assert_eq!(validate(&message, &catalog), []);
    assert!(message.flag);
    assert_eq!(message.labels.get("key").map(String::as_str), Some("value"));
    assert!(message.opaque.as_ref().is_some_and(|opaque| opaque.nested));
    assert!(matches!(message.choice, Some(Choice::Flag(true))));
    for (path, rust_type) in [
        ("test.Response.text", std::any::type_name::<Option<String>>()),
        ("test.Response.choice_flag", std::any::type_name::<Option<bool>>()),
    ] {
        let field = catalog.fields().find(|field| field.path == path).ok_or(path)?;
        assert_eq!(field.rust_type, rust_type);
    }
    Ok(())
}

#[test]
fn generated_catalogs_retain_executable_declaration_metadata()
-> Result<(), Box<dyn std::error::Error>> {
    let request = controls::request()?;
    let response = controls::response()?;
    let mut output = String::new();
    for field in request.fields().chain(response.fields()) {
        writeln!(output, "{}: {:?} {:?}", field.path, field.kind, field.tags)?;
        for (constraint, expected) in &field.expectations {
            writeln!(output, "  {constraint}: {expected:?}")?;
        }
    }
    for rule in request.rules() {
        writeln!(output, "rule: {rule:?}")?;
    }
    for rpc in controls::RPC_BINDINGS {
        writeln!(output, "rpc: {rpc:?}")?;
    }
    for action in [
        Action::Client(Client::Start),
        Action::Client(Client::Cancel { id: "cancel".into() }),
        Action::Worker(Worker::Poll),
    ] {
        writeln!(output, "binding: {:?}", controls::bind(&action))?;
    }
    insta::assert_snapshot!("generated-catalog", output);
    Ok(())
}
