use super::{Config, Domain, Model, Request, history};

#[test]
fn implicit_listen_exhaustion_records_zero_port_binary_identity() {
    let config: Config = serde_json::from_value(serde_json::json!({
        "nodes":[{"node":"a","address":"a"}],
        "links":[],
        "limits":{"listeners":4,"connections":4,"bytes":64,"connection_bytes":64,"stall_nanos":10,"transitions":8}
    })).unwrap();
    let mut model = Model::new(1, config).unwrap();
    let domain = Domain {
        run: 1,
        node: "a".into(),
        incarnation: 1,
        address: "a".into(),
    };
    model.register(domain.clone()).unwrap();
    model.nodes.get_mut("a").unwrap().next_listener_port = 65535;
    let listen = |id| Request {
        domain: domain.clone(),
        op: "listen".into(),
        handle: 0,
        host: String::new(),
        port: 0,
        bytes: Vec::new(),
        count: 0,
        deadline: 0,
        request_id: id,
    };
    let last = model.dispatch(listen(1));
    assert_eq!(last.error, "");
    assert_eq!(last.local.port, 65535);
    let exhausted = model.dispatch(listen(2));
    assert_eq!(exhausted.error, "network resources exhausted");
    let failure = model.transitions.last().unwrap();
    assert_eq!(
        failure,
        &history::Transition {
            kind: "listen".into(),
            source: history::Endpoint {
                node: "a".into(),
                incarnation: 1,
                address: "a".into(),
                port: 0
            },
            outcome: "capacity".into(),
            ..Default::default()
        }
    );
    assert_eq!(
        history::lane(failure),
        concat!("listener\0a\0", "00000000000000000001", "\0a\0", "00000")
    );
    assert_eq!(
        history::transitions_digest(std::slice::from_ref(failure)).unwrap(),
        "sha256:4bd4c13af7629a8cf6f8b35992963f1d5b44c43be9d206040c3ecc5e26b8116f"
    );
    let record = model.record().unwrap();
    assert_eq!(
        record.snapshot.transition_sha256,
        "sha256:55b912a7e4d21194a609401aeaf271f491b2637ac1ecb09e05d4eba519ec9fcc"
    );
    assert_eq!(
        record.snapshot.identity,
        "sha256:10cff68de9f5ea8ab8b6527ba17003828fcff6305a3f4690993f30997d71110a"
    );
    assert_eq!(record.snapshot.nodes.unwrap()[0].next_listener_port, 65536);
}
