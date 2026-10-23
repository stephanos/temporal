use super::*;
use serde::Deserialize;
use serde_json::{Value, json};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Action {
    op: String,
    #[serde(default)]
    index: usize,
    #[serde(default = "one")]
    incarnation: u64,
    #[serde(default)]
    kind: String,
    #[serde(default)]
    value: String,
    #[serde(default)]
    deadline: u64,
    #[serde(default)]
    nodes: Vec<String>,
    #[serde(default)]
    deadlines: Vec<u64>,
}

fn one() -> u64 {
    1
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Step {
    input: Action,
    expected: Value,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    schema: String,
    name: String,
    nodes: Vec<String>,
    #[serde(default)]
    network: bool,
    #[serde(default)]
    control: Option<String>,
    steps: Vec<Step>,
}

#[derive(Clone, Default)]
struct Capture(Arc<Mutex<Vec<u8>>>);

impl std::io::Write for Capture {
    fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
        lock(&self.0).extend_from_slice(data);
        Ok(data.len())
    }
    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

fn boundary(kind: &str) -> &'static str {
    match kind {
        "crash" => "crash",
        "stop" => "stop",
        "inspected_prefix" => "inspected_prefix",
        "infrastructure" => "infrastructure",
        other => panic!("unknown corpus boundary {other}"),
    }
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The corpus validator preserves its complete ordered literal and observation checks."
)]
fn broker_lifecycle_scripts() {
    let directory =
        PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/broker_lifecycle_rust");
    let mut files: Vec<_> = std::fs::read_dir(directory)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .collect();
    files.sort();
    assert_eq!(files.len(), 10);
    for path in files {
        let case: Case = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
        assert_eq!(case.schema, "gomad.broker-lifecycle-corpus/v1");
        assert_eq!(path.extension().unwrap(), "json");
        assert_eq!(path.file_stem().unwrap().to_str().unwrap(), case.name);
        assert!(!case.steps.is_empty());
        let shared = case.network.then(|| {
            let (config, _) = network_config(&json!({"network":{"profile":net::PROFILE,"limits":{
                "listeners":8,"connections":8,"bytes":1024,"connection_bytes":1024,"stall_nanos":1000,"transitions":64
            }}})).unwrap();
            Arc::new(Mutex::new(Shared::new(&config, &case.nodes).unwrap()))
        });
        let capture = Capture::default();
        let mut config = broker_config();
        config.pause_on_settle = case.control.is_some();
        let controller = case.control.map(|control| {
            let input: SharedInput = Arc::new(Mutex::new(Box::new(std::io::Cursor::new(
                control.into_bytes(),
            ))));
            let output: SharedOutput = Arc::new(Mutex::new(Box::new(capture.clone())));
            (input, output)
        });
        let broker =
            Arc::new(Broker::new(&config, &case.nodes, shared.clone(), controller).unwrap());
        let mut acknowledgment = None;
        let mut command_thread = None;
        for (ordinal, step) in case.steps.into_iter().enumerate() {
            let a = step.input;
            let actual = match a.op.as_str() {
                "defer_starts" => {
                    broker.defer_starts();
                    Value::Null
                }
                "start" => {
                    broker.start();
                    Value::Null
                }
                "activate" => json!(broker.activate(a.index).is_ok()),
                "run" => json!(broker.run(a.index, a.incarnation).is_ok()),
                "acquire" => json!(broker.acquire(a.index)),
                "finish" => {
                    broker.finish(a.index);
                    Value::Null
                }
                "restart" => match broker.restart(a.index, a.incarnation) {
                    Ok(pair) => json!(pair),
                    Err(_) => Value::Null,
                },
                "check" => json!(
                    broker
                        .check(a.index, a.incarnation)
                        .err()
                        .map(|error| error.termination())
                        .unwrap_or_default()
                ),
                "active" => json!(lock(&broker.inner).active),
                "hold" => json!(lock(&broker.inner).sessions[a.index].hold.is_some()),
                "current" => json!(broker.current()),
                "wait" => json!(broker.wait()),
                "boundary" => json!(
                    broker
                        .boundary(
                            a.index,
                            a.incarnation,
                            boundary(&a.kind),
                            a.value.into_bytes()
                        )
                        .is_ok()
                ),
                "output" => json!(
                    broker
                        .output(a.index, a.value.as_bytes())
                        .err()
                        .map(|error| error.termination())
                        .unwrap_or_default()
                ),
                "barrier" => json!(broker.barrier(&a.nodes, a.value.into_bytes()).is_ok()),
                "idle" => match broker.idle(a.index, a.deadline) {
                    Ok(now) => json!(now),
                    Err(error) => json!(error.termination()),
                },
                "abort" => {
                    broker.abort(boundary(&a.kind), a.value);
                    Value::Null
                }
                "failure" => json!(broker.failure()),
                "captured" => json!(String::from_utf8(lock(&capture.0).clone()).unwrap()),
                "idle_all" => {
                    let threads: Vec<_> = (0..case.nodes.len())
                        .map(|index| {
                            let broker = broker.clone();
                            let deadlines = a.deadlines.clone();
                            std::thread::spawn(move || {
                                let mut member = Member {
                                    broker: broker.clone(),
                                    incarnation: 1,
                                    index,
                                    network: None,
                                };
                                assert!(broker.acquire(index));
                                let (mut now, mut advances) = (1, 0);
                                for deadline in deadlines {
                                    let mut data = vec![0u8; 16];
                                    data[..4].copy_from_slice(&1u32.to_be_bytes());
                                    data[8..].copy_from_slice(&deadline.to_be_bytes());
                                    member
                                        .idle(
                                            &mut data,
                                            &mut Clockwork {
                                                now: &mut now,
                                                advances: &mut advances,
                                                runtime_active: true,
                                            },
                                        )
                                        .unwrap();
                                }
                                broker.finish(index);
                            })
                        })
                        .collect();
                    broker.start();
                    let complete = broker.wait();
                    for thread in threads {
                        thread.join().unwrap();
                    }
                    json!(complete)
                }
                "revoke" => json!(
                    broker
                        .command(Command::Revoke {
                            index: a.index,
                            incarnation: a.incarnation,
                            kind: boundary(&a.kind)
                        })
                        .is_ok()
                ),
                "queue_revoke" | "queue_link" | "queue_restart" => {
                    assert!(
                        acknowledgment.is_none(),
                        "unconsumed command acknowledgment"
                    );
                    let command = match a.op.as_str() {
                        "queue_revoke" => Command::Revoke {
                            index: a.index,
                            incarnation: a.incarnation,
                            kind: boundary(&a.kind),
                        },
                        "queue_restart" => Command::Restart {
                            index: a.index,
                            incarnation: a.incarnation,
                        },
                        _ => Command::Link {
                            action: "partition".into(),
                            from: a.nodes[0].clone(),
                            to: a.nodes[1].clone(),
                            delay: 0,
                            symmetric: true,
                        },
                    };
                    let (send, reply) = mpsc::sync_channel(1);
                    if a.op == "queue_restart" {
                        {
                            let inner = lock(&broker.inner);
                            assert!(inner.active.is_some(), "restart requires an active store");
                            assert!(inner.commands.is_empty(), "restart queue must start empty");
                        }
                        let peer = broker.clone();
                        command_thread = Some(std::thread::spawn(move || {
                            send.send(peer.command(command)).unwrap();
                        }));
                        let watchdog = Instant::now();
                        loop {
                            if !lock(&broker.inner).commands.is_empty() {
                                break;
                            }
                            assert!(
                                matches!(reply.try_recv(), Err(mpsc::TryRecvError::Empty)),
                                "restart completed before the active store yielded"
                            );
                            assert!(
                                watchdog.elapsed() < std::time::Duration::from_secs(10),
                                "restart worker did not reach the command queue"
                            );
                            std::thread::yield_now();
                        }
                        assert!(
                            matches!(reply.try_recv(), Err(mpsc::TryRecvError::Empty)),
                            "queued restart acknowledged before the active store yielded"
                        );
                    } else {
                        lock(&broker.inner).commands.push_back((command, send));
                    }
                    acknowledgment = Some(reply);
                    Value::Null
                }
                "ack" => {
                    let result = acknowledgment.take().unwrap().recv().unwrap();
                    if let Some(thread) = command_thread.take() {
                        thread.join().unwrap();
                    }
                    json!(result.is_ok())
                }
                "queued_commands" => json!(lock(&broker.inner).commands.len()),
                "session" => {
                    let inner = lock(&broker.inner);
                    json!([
                        inner.sessions[a.index].incarnation,
                        inner.sessions[a.index].slot
                    ])
                }
                "network_incarnations" => {
                    let record = lock(shared.as_ref().unwrap()).record().unwrap();
                    json!(
                        record
                            .snapshot
                            .nodes
                            .unwrap()
                            .into_iter()
                            .filter(|node| node.node.starts_with(&a.value))
                            .map(|node| node.last_incarnation)
                            .collect::<Vec<_>>()
                    )
                }
                "cross_links" => {
                    let record = lock(shared.as_ref().unwrap()).record().unwrap();
                    json!(
                        record
                            .snapshot
                            .links
                            .unwrap()
                            .into_iter()
                            .filter(|link| link.from[..1] != link.to[..1])
                            .map(|link| link.enabled)
                            .collect::<Vec<_>>()
                    )
                }
                "park_manual_idle" => {
                    let mut inner = lock(&broker.inner);
                    inner.active = None;
                    inner.sessions[a.index].state = State::Idle;
                    Value::Null
                }
                "idle_with_restarted_peer" => {
                    let peer = broker.clone();
                    let index = a.index;
                    let incarnation = a.incarnation;
                    let thread = std::thread::spawn(move || {
                        let acquired = peer.acquire(index);
                        let current = peer.check(index, incarnation).is_ok();
                        peer.finish(index);
                        (acquired, current)
                    });
                    let now = broker.idle(0, a.deadline).unwrap();
                    let (acquired, current) = thread.join().unwrap();
                    json!({"now":now,"peer_acquired":acquired,"peer_current":current})
                }
                "output_barrier_handoff" => {
                    let peer = broker.clone();
                    let marker = a.value.clone();
                    let thread = std::thread::spawn(move || {
                        assert!(peer.acquire(0));
                        peer.output(0, marker.as_bytes()).unwrap();
                        let kind = peer.idle(0, 1000).unwrap_err().termination();
                        peer.finish(0);
                        kind
                    });
                    assert!(broker.acquire(1));
                    let kind = broker
                        .output(1, a.value.as_bytes())
                        .unwrap_err()
                        .termination();
                    broker.finish(1);
                    json!([thread.join().unwrap(), kind])
                }
                op => panic!("unknown broker corpus action {op}"),
            };
            assert_eq!(
                actual, step.expected,
                "{} step {ordinal} {}",
                case.name, a.op
            );
        }
        assert!(
            acknowledgment.is_none(),
            "{} left a command pending",
            case.name
        );
        assert!(
            command_thread.is_none(),
            "{} left a worker pending",
            case.name
        );
    }
}
