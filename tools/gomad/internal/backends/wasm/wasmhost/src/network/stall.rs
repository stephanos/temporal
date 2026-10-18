impl super::Model {
    pub(super) fn expire_data_stalls(&mut self) -> Result<(), String> {
        if self.finished {
            return Ok(());
        }
        let identities: Vec<_> = self.connections.keys().copied().collect();
        for identity in identities {
            let pair = self.connections[&identity].clone();
            if pair.closed
                || pair.reset
                || self.states[&(identity, true)].close_called
                || self.states[&(identity, false)].close_called
                || !self.reservations.contains(&identity)
                || self.nodes[&pair.client.node].last_incarnation > pair.client.incarnation
                || self.nodes[&pair.server.node].last_incarnation > pair.server.incarnation
            {
                continue;
            }
            for client in [true, false] {
                let state = &self.states[&(identity, client)];
                if !state.stalled || self.now < state.stall_due {
                    continue;
                }
                self.commit_batch(
                    &[super::history::Transition {
                        kind: "reset".into(),
                        source: pair.client.clone(),
                        destination: pair.server.clone(),
                        connection: identity,
                        bytes: self.states[&(identity, true)].retained_bytes
                            + self.states[&(identity, false)].retained_bytes,
                        outcome: "timeout".into(),
                        ..Default::default()
                    }],
                    0,
                    1,
                )?;
                self.reservations.remove(&identity);
                let pair = self.connections.get_mut(&identity).unwrap();
                pair.closed = true;
                pair.reset = true;
                for client in [true, false] {
                    self.drop_bytes(identity, client, false);
                    self.states.get_mut(&(identity, client)).unwrap().terminal =
                        Some("connection timed out".into());
                }
                *self.pair_generation.get_mut(&identity).unwrap() += 1;
                self.network_generation += 1;
                break;
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::super::{Config, Domain, Model, OperationResult, Request, history};
    use serde_json::{Value, json};

    #[expect(
        clippy::needless_pass_by_value,
        reason = "Fixture configuration owns the optional replay record for serde_json"
    )]
    fn config(replay: Option<history::Record>, transitions: u64) -> Config {
        serde_json::from_value(json!({
            "nodes": [{"node": "a", "address": "a"}, {"node": "b", "address": "b"}],
            "links": [
                {"from": "a", "to": "b", "enabled": true, "delay_nanos": 0},
                {"from": "b", "to": "a", "enabled": true, "delay_nanos": 0}
            ],
            "limits": {"listeners": 4, "connections": 4, "bytes": 32,
                "connection_bytes": 4, "stall_nanos": 100, "transitions": transitions},
            "replay": replay
        }))
        .unwrap()
    }

    fn model(replay: Option<history::Record>, transitions: u64) -> Model {
        let mut model = Model::new(1, config(replay, transitions)).unwrap();
        for node in ["a", "b"] {
            model
                .register(Domain {
                    run: 1,
                    node: node.into(),
                    incarnation: 1,
                    address: node.into(),
                })
                .unwrap();
        }
        model
    }

    fn dispatch(model: &mut Model, node: &str, id: u64, mut operation: Value) -> OperationResult {
        operation["domain"] = json!({"run": 1, "node": node, "incarnation": 1, "address": node});
        operation["request_id"] = id.into();
        let request: Request = serde_json::from_value(operation).unwrap();
        model.dispatch(request)
    }

    fn connect(model: &mut Model, id: u64) -> (u64, u64) {
        let listener = dispatch(model, "b", id, json!({"op": "listen", "port": 0}));
        assert_eq!(listener.error, "");
        let client = dispatch(
            model,
            "a",
            id + 1,
            json!({"op": "dial", "host": "b", "port": listener.local.port}),
        );
        assert_eq!(client.error, "");
        let server = dispatch(
            model,
            "b",
            id + 2,
            json!({"op": "accept", "handle": listener.handle}),
        );
        assert_eq!(server.error, "");
        (client.handle, server.handle)
    }

    fn write(model: &mut Model, node: &str, id: u64, handle: u64, bytes: &[u8]) -> OperationResult {
        dispatch(
            model,
            node,
            id,
            json!({"op": "write", "handle": handle, "bytes": bytes}),
        )
    }

    fn held_pair(model: &mut Model) -> (u64, u64) {
        let (client, server) = connect(model, 1);
        model.partition("a", "b", false).unwrap();
        assert_eq!(write(model, "a", 4, client, b"held").count, 4);
        (client, server)
    }

    #[test]
    fn reserved_expiry_drops_received_and_held_bytes_at_history_ceiling() {
        let mut model = model(None, 8);
        let (client, server) = connect(&mut model, 1);
        assert_eq!(write(&mut model, "b", 4, server, b"ab").count, 2);
        assert_eq!(
            dispatch(
                &mut model,
                "a",
                5,
                json!({"op": "read", "handle": client, "count": 1})
            )
            .bytes,
            b"a"
        );
        model.partition("a", "b", true).unwrap();
        assert_eq!(write(&mut model, "a", 6, client, b"held").count, 4);
        assert_eq!(write(&mut model, "b", 7, server, b"cd").count, 2);
        assert_eq!(model.transitions.len(), 7);
        assert_eq!(model.history_reserved, 1);
        model.advance(100).unwrap();
        assert!(!model.connections[&1].reset);
        model.advance(101).unwrap();
        assert_eq!(model.pending_bytes, 0);
        assert_eq!(model.history_reserved, 0);
        assert_eq!(model.next_deadline(), 0);
        assert_eq!(
            model.transitions.last().unwrap(),
            &history::Transition {
                kind: "reset".into(),
                source: history::Endpoint {
                    node: "a".into(),
                    incarnation: 1,
                    address: "a".into(),
                    port: 40000
                },
                destination: history::Endpoint {
                    node: "b".into(),
                    incarnation: 1,
                    address: "b".into(),
                    port: 20000
                },
                connection: 1,
                bytes: 7,
                outcome: "timeout".into(),
                ..Default::default()
            }
        );
        for (node, handle) in [("a", client), ("b", server)] {
            assert_eq!(
                dispatch(
                    &mut model,
                    node,
                    8,
                    json!({"op": "read", "handle": handle, "count": 1})
                )
                .error,
                "connection timed out"
            );
            assert_eq!(
                write(&mut model, node, 9, handle, b"x").error,
                "connection timed out"
            );
        }
        model.advance(1000).unwrap();
        assert_eq!(model.transitions.len(), 8);
    }

    #[test]
    fn expiry_wakes_partial_writer_and_blocked_reader_with_terminal_timeout() {
        let mut model = model(None, 32);
        let (client, server) = connect(&mut model, 1);
        model.partition("a", "b", false).unwrap();
        let writer = write(&mut model, "a", 4, client, b"heldmore");
        assert_eq!(writer.state, "pending");
        assert_eq!(writer.count, 4);
        assert_eq!(
            dispatch(
                &mut model,
                "b",
                5,
                json!({"op": "read", "handle": server, "count": 1})
            )
            .state,
            "pending"
        );
        assert_eq!(model.advance(101).unwrap(), Some(vec![4, 5]));
        let writer = dispatch(&mut model, "a", 4, json!({"op": "poll"}));
        assert_eq!(writer.count, 4);
        assert_eq!(writer.error, "connection timed out");
        let reader = dispatch(&mut model, "b", 5, json!({"op": "poll"}));
        assert_eq!(reader.error, "connection timed out");
        assert_eq!(reader.bytes.len(), 0);
    }

    #[test]
    fn healed_stall_epoch_survives_old_due_then_expires_replacement_epoch() {
        let mut model = model(None, 32);
        held_pair(&mut model);
        model.advance(100).unwrap();
        model.heal("a", "b", false).unwrap();
        model.partition("a", "b", false).unwrap();
        model.advance(101).unwrap();
        assert!(!model.connections[&1].reset);
        assert_eq!(model.next_deadline(), 201);
        model.advance(201).unwrap();
        assert!(model.connections[&1].reset);
        assert_eq!(model.history_reserved, 0);
    }

    #[test]
    fn equal_time_heal_first_invalidates_expiry_while_expiry_first_is_permanent() {
        for heal_first in [true, false] {
            let mut model = model(None, 32);
            let (_, server) = held_pair(&mut model);
            if heal_first {
                model.advance_before(101).unwrap();
                model.heal("a", "b", false).unwrap();
                model.advance(101).unwrap();
            } else {
                model.advance(101).unwrap();
                model.heal("a", "b", false).unwrap();
            }
            let result = dispatch(
                &mut model,
                "b",
                5,
                json!({"op": "read", "handle": server, "count": 4}),
            );
            if heal_first {
                assert_eq!(result.bytes, b"held");
                assert_eq!(result.error, "");
            } else {
                assert_eq!(result.error, "connection timed out");
            }
        }
    }

    #[test]
    fn simultaneous_pair_expiry_commits_in_identity_order() {
        let mut model = model(None, 32);
        let (first, _) = connect(&mut model, 1);
        let (second, _) = connect(&mut model, 4);
        model.partition("a", "b", false).unwrap();
        assert_eq!(write(&mut model, "a", 7, second, b"two").count, 3);
        assert_eq!(write(&mut model, "a", 8, first, b"one").count, 3);
        model.advance(101).unwrap();
        let identities: Vec<_> = model
            .transitions
            .iter()
            .filter(|transition| transition.kind == "reset")
            .map(|transition| transition.connection)
            .collect();
        assert_eq!(identities, vec![1, 2]);
        assert_eq!(model.history_reserved, 0);
    }

    #[test]
    fn replay_mismatch_preserves_timeout_reservation_and_queued_bytes() {
        let mut model = model(None, 32);
        held_pair(&mut model);
        let mut record = model.record().unwrap();
        let mut transitions = model.transitions.clone();
        transitions.push(history::Transition {
            kind: "reset".into(),
            source: model.connections[&1].client.clone(),
            destination: model.connections[&1].server.clone(),
            connection: 1,
            bytes: 5,
            outcome: "timeout".into(),
            ..Default::default()
        });
        history::canonicalize(&mut transitions);
        record.snapshot.transition_sha256 = history::transitions_digest(&transitions).unwrap();
        let pair = &mut record.snapshot.connections.as_mut().unwrap()[0];
        pair.closed = true;
        pair.reset = true;
        pair.client_bytes = 0;
        pair.server_bytes = 0;
        record.snapshot.identity = history::snapshot_digest(&record.snapshot).unwrap();
        record.transitions = Some(transitions);
        let mut model = self::model(Some(record), 32);
        let (client, _) = held_pair(&mut model);
        let before = model.snapshot().unwrap();
        assert_eq!(
            model.advance(101).unwrap_err(),
            "simulation network replay diverged"
        );
        assert_eq!(model.snapshot().unwrap(), before);
        assert_eq!(model.pending_bytes, 4);
        assert_eq!(model.history_reserved, 1);
        assert_eq!(model.next_deadline(), 101);
        assert_eq!(
            write(&mut model, "a", 5, client, b"x").error,
            "simulation network replay diverged"
        );
        assert_eq!(
            model.finish().unwrap_err(),
            "simulation network replay diverged"
        );
    }
}
