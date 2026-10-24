use super::*;
use crate::network::stream_abi::{set_u32, set_u64};

fn frame(op: u32) -> Vec<u8> {
    let mut data = vec![0; 64];
    set_u32(&mut data, 0, 2);
    set_u32(&mut data, 4, op);
    data
}

#[test]
fn readiness_probe_keeps_three_transitions_and_observes_queued_controls_at_next_boundary() {
    let model: ModelV2 = serde_json::from_value(serde_json::json!({
        "profile":"gomad.simulation-network-stream/v3",
        "limits":{"listeners":128,"connections":128,"bytes":8<<20,
            "connection_bytes":4<<20,"stall_nanos":10_000_000_000_u64,"transitions":10000}
    }))
    .unwrap();
    let config = net::Config {
        model: model.clone(),
        addresses: BTreeMap::new(),
        links: vec![],
        strict_topology: false,
        faults: vec![],
    };
    for next in ["probe", "poll", "idle"] {
        let ids = vec!["a".into(), "b".into()];
        let shared = Arc::new(Mutex::new(Shared::new(&config, &ids).unwrap()));
        let mut broker_config = broker_config();
        broker_config.transition_bytes = Some(1 << 20);
        let broker = Arc::new(
            Broker::new(&broker_config, &["a".into()], Some(shared.clone()), None).unwrap(),
        );
        let endpoint = Endpoint::new(shared.clone(), "a", 1, 1, &model, 1000, 1 << 20).unwrap();
        let mut member = Member {
            broker: broker.clone(),
            incarnation: 1,
            index: 0,
            network: Some(endpoint),
        };
        broker.start();
        assert!(broker.acquire(0));
        let (mut now, mut advances) = (1, 0);
        let mut clock = Clockwork {
            now: &mut now,
            advances: &mut advances,
            runtime_active: true,
        };
        member.net_stream(&mut frame(14), &mut clock).unwrap();
        {
            let inner = lock(&broker.inner);
            assert_eq!(
                inner
                    .world
                    .transitions()
                    .iter()
                    .map(|transition| transition.body.kind())
                    .collect::<Vec<_>>(),
                ["register", "ready", "quiesce"]
            );
        }
        let before = lock(&shared).record().unwrap();
        let (send, receive) = mpsc::sync_channel(1);
        lock(&broker.inner).commands.push_back((
            Command::Link {
                action: "partition".into(),
                from: "a".into(),
                to: "b".into(),
                delay: 0,
                symmetric: true,
            },
            send,
        ));
        assert!(matches!(receive.try_recv(), Err(mpsc::TryRecvError::Empty)));
        assert_eq!(lock(&shared).record().unwrap(), before);
        match next {
            "probe" => member.net_stream(&mut frame(14), &mut clock).unwrap(),
            "poll" => {
                let mut data = frame(12);
                set_u64(&mut data, 8, 99);
                member.net_stream(&mut data, &mut clock).unwrap();
            }
            _ => {
                let mut data = vec![0; 16];
                set_u32(&mut data, 0, 1);
                set_u64(&mut data, 8, 10);
                member.idle(&mut data, &mut clock).unwrap();
            }
        }
        receive.try_recv().unwrap().unwrap();
        assert_ne!(lock(&shared).record().unwrap(), before);
        let (send, receive) = mpsc::sync_channel(1);
        lock(&broker.inner).commands.push_back((
            Command::Revoke {
                index: 0,
                incarnation: 1,
                kind: "stop",
            },
            send,
        ));
        assert!(matches!(receive.try_recv(), Err(mpsc::TryRecvError::Empty)));
        assert!(matches!(
            member.net_stream(&mut frame(14), &mut clock),
            Err(Failure::Boundary("stop", _))
        ));
        receive.try_recv().unwrap().unwrap();
        broker.finish(0);
    }
}
