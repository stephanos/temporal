use super::*;

fn probe(ids: &[u64]) -> Vec<u8> {
    let payload: Vec<_> = ids.iter().flat_map(|id| id.to_be_bytes()).collect();
    let mut data = buffer(14, 0, 0, 0, &payload);
    set_u32(&mut data, 36, u32::try_from(ids.len()).unwrap());
    data
}

fn pending_accept(endpoint: &mut Endpoint, id: u64, listener: u64) {
    let mut accept = buffer(3, id, listener, 0, &[]);
    endpoint.serve(&mut accept, 1, true).unwrap();
    assert_eq!(get32(&accept, 40), 1);
}

fn listener(endpoint: &mut Endpoint) -> (u64, u32) {
    let mut listen = buffer(1, 1, 0, 0, &[]);
    endpoint.serve(&mut listen, 1, true).unwrap();
    (
        u64::from_be_bytes(listen[16..24].try_into().unwrap()),
        get32(&listen, 48),
    )
}

fn results(data: &[u8]) -> Vec<u64> {
    data[HEADER..]
        .chunks_exact(8)
        .map(|chunk| u64::from_be_bytes(chunk.try_into().unwrap()))
        .collect()
}

#[test]
fn readiness_probe_empty_full_and_repeated_do_not_consume_completions() {
    let shared = Arc::new(Mutex::new(Shared::new(&config(), &ids()).unwrap()));
    let mut a = endpoint(&shared, "a", 1, 1 << 20);
    let mut b = endpoint(&shared, "b", 2, 1 << 20);
    let mut empty = probe(&[]);
    let expected = empty.clone();
    b.serve(&mut empty, 1, true).unwrap();
    assert_eq!(empty, expected);
    let (handle, port) = listener(&mut b);
    let ids: Vec<_> = (2..130).rev().collect();
    for &id in &ids {
        pending_accept(&mut b, id, handle);
    }
    let mut data = probe(&ids);
    b.serve(&mut data, 1, true).unwrap();
    assert_eq!(results(&data), vec![0; 128]);
    assert_eq!(get32(&data, 36), 128);
    let mut dial = buffer(2, 1, 0, port, &[]);
    set_u32(&mut dial, 56, u32::from_be_bytes([10, 0, 0, 3]));
    a.serve(&mut dial, 1, true).unwrap();
    let before = b.world.transitions().len();
    for repetition in 0..2 {
        let mut data = probe(&ids);
        b.serve(&mut data, 1, true).unwrap();
        let ready = results(&data);
        assert_eq!(ready.iter().filter(|id| **id != 0).count(), 1);
        assert_eq!(ready.last(), Some(&2));
        assert_eq!(get32(&data, 36), 128);
        if repetition == 0 {
            assert!(b.world.transitions().len() > before);
        }
    }
    let mut poll = buffer(12, 2, 0, 0, &[]);
    b.serve(&mut poll, 1, true).unwrap();
    assert_eq!(get32(&poll, 40), 0);
    assert_eq!(get32(&poll, 44), 0);
    assert_ne!(u64::from_be_bytes(poll[16..24].try_into().unwrap()), 0);
    assert!(matches!(
        b.serve(&mut probe(&[2]), 1, true),
        Err(Failure::Invalid(_))
    ));
}

#[test]
fn readiness_probe_rejects_malformed_duplicate_foreign_stale_and_inactive_frames() {
    let shared = Arc::new(Mutex::new(Shared::new(&config(), &ids()).unwrap()));
    let mut a = endpoint(&shared, "a", 1, 1 << 20);
    let mut b = endpoint(&shared, "b", 2, 1 << 20);
    let (listener_a, _) = listener(&mut a);
    let (listener_b, _) = listener(&mut b);
    pending_accept(&mut a, 2, listener_a);
    pending_accept(&mut b, 3, listener_b);
    let mut malformed = vec![
        probe(&[0]),
        probe(&[1 << 48]),
        probe(&[3, 3]),
        probe(&[2]),
        probe(&[99]),
        probe(&[3; 129]),
    ];
    for offset in [8, 16, 24, 32, 40, 44, 48, 52, 56, 60] {
        let mut data = probe(&[3]);
        data[offset] = 1;
        malformed.push(data);
    }
    let mut truncated = probe(&[3]);
    truncated.pop();
    malformed.push(truncated);
    let mut extra = probe(&[3]);
    extra.push(0);
    malformed.push(extra);
    let mut bad_version = probe(&[3]);
    set_u32(&mut bad_version, 0, 1);
    malformed.push(bad_version);
    for mut data in malformed {
        let original = data.clone();
        assert!(
            matches!(b.serve(&mut data, 1, true), Err(Failure::Invalid(_))),
            "{original:?}"
        );
        assert_eq!(data, original);
    }
    assert!(matches!(
        b.serve(&mut probe(&[]), 1, false),
        Err(Failure::Invalid(_))
    ));
    let mut cancel = buffer(13, 3, 0, 0, &[]);
    b.serve(&mut cancel, 1, true).unwrap();
    assert!(matches!(
        b.serve(&mut probe(&[3]), 1, true),
        Err(Failure::Invalid(_))
    ));
    pending_accept(&mut b, 4, listener_b);
    lock(&shared).revoke("b", 1, false).unwrap();
    assert!(matches!(
        b.serve(&mut probe(&[4]), 1, true),
        Err(Failure::Invalid(_))
    ));
    lock(&shared).register("b", 2).unwrap();
    let mut restarted = Endpoint::new(shared, "b", 2, 3, &profile(), 1000, 1 << 20).unwrap();
    assert!(matches!(
        restarted.serve(&mut probe(&[4]), 1, true),
        Err(Failure::Invalid(_))
    ));
}

#[test]
fn readiness_probe_advances_deadline_and_preserves_error_until_poll() {
    let shared = Arc::new(Mutex::new(Shared::new(&config(), &ids()).unwrap()));
    let mut b = endpoint(&shared, "b", 2, 1 << 20);
    let (listener, _) = listener(&mut b);
    let mut accept = buffer(3, 2, listener, 0, &[]);
    set_u64(&mut accept, 24, 5);
    b.serve(&mut accept, 1, true).unwrap();
    for (now, expected) in [(1, 0), (5, 2), (5, 2)] {
        let mut data = probe(&[2]);
        b.serve(&mut data, now, true).unwrap();
        assert_eq!(results(&data), [expected]);
    }
    let mut poll = buffer(12, 2, 0, 0, &[]);
    b.serve(&mut poll, 5, true).unwrap();
    assert_eq!(get32(&poll, 44), 73);
}

#[test]
#[expect(
    clippy::too_many_lines,
    reason = "The differential table keeps identical legacy and probed network scripts together"
)]
fn readiness_probe_matches_legacy_scans_across_stream_mutations() {
    fn scan(
        endpoint: &mut Endpoint,
        pending: &mut Vec<u64>,
        now: u64,
        batched: bool,
    ) -> Vec<(u64, u64, Vec<u8>)> {
        let mut completed = Vec::new();
        let mut index = 0;
        while index < pending.len() {
            let ready = if batched {
                let mut data = probe(&pending[index..]);
                endpoint.serve(&mut data, now, true).unwrap();
                results(&data)
            } else {
                pending[index..].to_vec()
            };
            let mut consumed = false;
            for id in ready {
                if id == 0 {
                    index += 1;
                    continue;
                }
                let mut poll = buffer(12, id, 0, 0, &[0]);
                endpoint.serve(&mut poll, now, true).unwrap();
                if get32(&poll, 40) == 0 {
                    assert_eq!(pending.remove(index), id);
                    completed.push((now, id, poll));
                    consumed = true;
                    break;
                }
                index += 1;
            }
            if !consumed {
                break;
            }
        }
        completed
    }

    let run = |case: &str, batched| {
        let shared = Arc::new(Mutex::new(Shared::new(&config(), &ids()).unwrap()));
        let mut a = endpoint(&shared, "a", 1, 1 << 20);
        let mut b = endpoint(&shared, "b", 2, 1 << 20);
        let (listener, port) = listener(&mut b);
        pending_accept(&mut b, 2, listener);
        let mut dial = buffer(2, 1, 0, port, &[]);
        set_u32(&mut dial, 56, u32::from_be_bytes([10, 0, 0, 3]));
        a.serve(&mut dial, 1, true).unwrap();
        let client = u64::from_be_bytes(dial[16..24].try_into().unwrap());
        let mut accept = buffer(12, 2, 0, 0, &[]);
        b.serve(&mut accept, 1, true).unwrap();
        let server = u64::from_be_bytes(accept[16..24].try_into().unwrap());
        let mut pending = vec![11, 10];
        for &id in &pending {
            let mut read = buffer(4, id, server, 0, &[0]);
            b.serve(&mut read, 1, true).unwrap();
            assert_eq!(get32(&read, 40), 1);
        }
        let mut observed = scan(&mut b, &mut pending, 1, batched);
        assert!(observed.is_empty());
        match case {
            "write" | "delay" | "partition_heal" => {
                if case != "write" {
                    lock(&shared)
                        .change_link("delay", "a", "b", 3, true)
                        .unwrap();
                }
                let mut write = buffer(5, 2, client, 0, b"ab");
                a.serve(&mut write, 1, true).unwrap();
                if case == "partition_heal" {
                    lock(&shared)
                        .change_link("partition", "a", "b", 0, true)
                        .unwrap();
                }
                observed.extend(scan(&mut b, &mut pending, 1, batched));
                if case == "partition_heal" {
                    observed.extend(scan(&mut b, &mut pending, 4, batched));
                    assert_eq!(pending.len(), 2);
                    lock(&shared)
                        .change_link("heal", "a", "b", 0, true)
                        .unwrap();
                }
            }
            "close" | "close_write" | "close_read" => {
                let (target, op, handle) = if case == "close_read" {
                    (&mut b, 7, server)
                } else {
                    (&mut a, if case == "close" { 6 } else { 8 }, client)
                };
                target
                    .serve(&mut buffer(op, 3, handle, 0, &[]), 1, true)
                    .unwrap();
            }
            "deadline" => {
                for deadline in [2, 0, 4] {
                    let mut data = buffer(10, 3, server, 0, &[]);
                    set_u64(&mut data, 24, deadline);
                    b.serve(&mut data, 1, true).unwrap();
                }
            }
            "revoke_restart" => {
                lock(&shared).revoke("a", 1, false).unwrap();
                lock(&shared).register("a", 2).unwrap();
                let mut fresh =
                    Endpoint::new(shared.clone(), "a", 2, 3, &profile(), 1000, 1 << 20).unwrap();
                fresh
                    .serve(&mut buffer(1, 1, 0, 4501, &[]), 1, true)
                    .unwrap();
            }
            _ => unreachable!(),
        }
        observed.extend(scan(
            &mut b,
            &mut pending,
            if case == "partition_heal" { 7 } else { 4 },
            batched,
        ));
        assert!(pending.is_empty(), "{case}: {pending:?}");
        assert_eq!(
            observed.iter().map(|(_, id, _)| *id).collect::<Vec<_>>(),
            [11, 10]
        );
        observed
    };
    for case in [
        "write",
        "delay",
        "partition_heal",
        "close",
        "close_write",
        "close_read",
        "deadline",
        "revoke_restart",
    ] {
        assert_eq!(run(case, false), run(case, true), "{case}");
    }
}
