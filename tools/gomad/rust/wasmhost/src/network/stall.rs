impl super::Model {
    #[allow(
        dead_code,
        reason = "The network corpus retains the standalone stall-expiry entry point."
    )]
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
