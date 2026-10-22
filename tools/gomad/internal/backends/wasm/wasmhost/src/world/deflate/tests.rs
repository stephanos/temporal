use super::*;
use std::io::Write;
use std::process::{Command, Stdio};

fn oracle(input: &[u8]) -> Vec<u8> {
    oracle_level(input, -1)
}

fn oracle_level(input: &[u8], level: i32) -> Vec<u8> {
    let executable =
        std::env::var("GOMAD_GO_GZIP_ORACLE").expect("set pinned Go gzip oracle executable");
    let mut child = Command::new(executable)
        .arg(level.to_string())
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .spawn()
        .unwrap();
    let mut stdin = child.stdin.take().unwrap();
    let output = std::thread::scope(|scope| {
        let writer = scope.spawn(move || stdin.write_all(input).unwrap());
        let output = child.wait_with_output().unwrap();
        writer.join().unwrap();
        output
    });
    assert!(output.status.success());
    output.stdout
}

#[test]
#[ignore = "requires pinned Go oracle; see deflate/README.md"]
fn default_compression_matches_pinned_go() {
    let mut random = 1u64;
    for len in [
        0, 1, 32, 33, 127, 128, 257, 1024, 65534, 65535, 65536, 131_070, 393_211, 1_048_576,
    ] {
        for pattern in 0..4 {
            let input: Vec<u8> = (0..len)
                .map(|i| match pattern {
                    0 => b'a',
                    1 => {
                        b"GMWORLD2/GMWSNAP1/requests/events/transitions"
                            [i % b"GMWORLD2/GMWSNAP1/requests/events/transitions".len()]
                    }
                    2 => {
                        random ^= random << 13;
                        random ^= random >> 7;
                        random ^= random << 17;
                        random as u8
                    }
                    _ => ((i % 521) ^ ((i / 521) % 7)) as u8,
                })
                .collect();
            let expected = oracle(&input);
            let actual = encode(&input, 64 << 20).unwrap();
            assert_eq!(
                actual,
                expected[10..expected.len() - 8],
                "length={len} pattern={pattern}"
            );
            assert_eq!(
                super::super::gzip::encode(&input, 64 << 20).unwrap(),
                expected
            );
            let (decoded, consumed) = super::super::gzip::decode(&expected, input.len()).unwrap();
            assert_eq!(decoded, input, "decode length={len} pattern={pattern}");
            assert_eq!(consumed, expected.len());
        }
    }
}

#[test]
#[ignore = "requires pinned Go oracle; see deflate/README.md"]
fn decode_all_go_block_types_with_bounded_output() {
    let input = vec![b'a'; 200_000];
    let gzip = oracle(&input);
    let body = &gzip[10..gzip.len() - 8];
    let (decoded, consumed) = decode(body, input.len()).unwrap();
    assert_eq!(decoded, input);
    assert_eq!(consumed, body.len());
    assert!(decode(body, input.len() - 1).is_err());
    for end in 0..body.len() {
        assert!(decode(&body[..end], input.len()).is_err());
    }
    assert_eq!(decode(&[3, 0], 0).unwrap(), (Vec::new(), 2));
    assert!(decode(&[7], 0).is_err());
    assert!(decode(&[1, 1, 0, 1, 0, b'x'], 1).is_err());
}

#[test]
#[ignore = "requires pinned Go oracle; see deflate/README.md"]
fn accepts_independent_go_compression_levels() {
    let input: Vec<u8> = (0..100_003)
        .map(|i| {
            b"World-boundary-entropy-and-repetition"
                [i % b"World-boundary-entropy-and-repetition".len()]
        })
        .collect();
    let mut kinds = std::collections::BTreeSet::new();
    for level in [-2, 0, 1, 6, 9] {
        let gzip = oracle_level(&input, level);
        kinds.insert((gzip[10] >> 1) & 3);
        let (decoded, consumed) = super::super::gzip::decode(&gzip, input.len()).unwrap();
        assert_eq!(decoded, input, "level={level}");
        assert_eq!(consumed, gzip.len());
    }
    let fixed = oracle_level(b"hello", 9);
    assert_eq!(super::super::gzip::decode(&fixed, 5).unwrap().0, b"hello");
    kinds.insert((fixed[10] >> 1) & 3);
    assert_eq!(kinds, [0, 1, 2].into_iter().collect());
}

#[test]
#[ignore = "requires pinned Go oracle; see deflate/README.md"]
fn actual_world_snapshots_match_raw_gzip() {
    let executable = std::env::var("GOMAD_GO_GZIP_ORACLE").unwrap();
    for count in [0, 1, 10, 1000] {
        let output = Command::new(&executable)
            .args(["world", &count.to_string()])
            .output()
            .unwrap();
        assert!(
            output.status.success(),
            "{}",
            String::from_utf8_lossy(&output.stderr)
        );
        let envelope = output.stdout;
        assert_eq!(&envelope[..8], b"GMWORLD2");
        let expanded = u64::from_be_bytes(envelope[16..24].try_into().unwrap()) as usize;
        let (raw, consumed) = super::super::gzip::decode(&envelope[64..], expanded).unwrap();
        assert_eq!(&raw[..8], b"GMWSNAP1");
        assert_eq!(raw.len(), expanded);
        assert_eq!(consumed + 64, envelope.len());
        assert_eq!(
            super::super::gzip::encode(&raw, 64 << 20).unwrap(),
            envelope[64..]
        );
    }
}

#[test]
#[ignore = "requires pinned Go oracle; see deflate/README.md"]
fn mutated_stream_admission_matches_go() {
    let input: Vec<u8> = (0..5000).map(|i| (i % 13) as u8).collect();
    let base = oracle(&input);
    let executable = std::env::var("GOMAD_GO_GZIP_ORACLE").unwrap();
    for at in 0..base.len() {
        for bit in [0, 3, 7] {
            let mut candidate = base.clone();
            candidate[at] ^= 1 << bit;
            let mut child = Command::new(&executable)
                .args(["decode", "1048576"])
                .stdin(Stdio::piped())
                .stdout(Stdio::piped())
                .stderr(Stdio::null())
                .spawn()
                .unwrap();
            child.stdin.take().unwrap().write_all(&candidate).unwrap();
            let output = child.wait_with_output().unwrap();
            let actual = super::super::gzip::decode(&candidate, 1 << 20);
            assert_eq!(
                actual.is_ok(),
                output.status.success(),
                "offset={at} bit={bit}"
            );
            if let Ok((raw, _)) = actual {
                assert_eq!(raw, output.stdout);
            }
        }
    }
}

#[test]
#[ignore = "384MiB resource window plus pinned Go oracle; see deflate/README.md"]
fn original_full_profile_bounds() {
    const MAX: usize = 384 << 20;
    let raw = vec![b'a'; MAX + 1];
    let expected = oracle(&raw[..MAX]);
    let actual = super::super::gzip::encode(&raw[..MAX], 64 << 20).unwrap();
    assert!(actual == expected, "exact gzip mismatch");
    if let Ok(directory) = std::env::var("GOMAD_CODEC_EVIDENCE_DIR") {
        let directory = std::path::Path::new(&directory);
        std::fs::write(directory.join("world-codec-max-rust.gz"), &actual).unwrap();
        std::fs::write(directory.join("world-codec-max-go.gz"), &expected).unwrap();
    }
    println!(
        "expanded_bytes={} compressed_bytes={} crc32={:08x}",
        MAX,
        actual.len(),
        crc32fast::hash(&actual)
    );
    assert!(super::super::gzip::encode(&raw, 64 << 20).is_err());
    let (decoded, consumed) = super::super::gzip::decode(&actual, MAX).unwrap();
    assert_eq!(decoded, raw[..MAX]);
    assert!(decoded.capacity() <= MAX);
    assert_eq!(consumed, actual.len());
    drop(decoded);
    drop(raw);
    assert!(super::super::gzip::decode(&actual, MAX - 1).is_err());
    let mut state = 1u64;
    let random: Vec<u8> = (0..(64 << 20))
        .map(|_| {
            state ^= state << 13;
            state ^= state >> 7;
            state ^= state << 17;
            state as u8
        })
        .collect();
    assert!(super::super::gzip::encode(&random, 64 << 20).is_err());
}

#[test]
#[ignore = "requires pinned Go oracle; see deflate/README.md"]
fn changing_distributions_and_lazy_matches_agree_with_go() {
    let mut random = 29u64;
    let mut raw = Vec::new();
    for block in 0..32 {
        for i in 0..65535 {
            random ^= random << 13;
            random ^= random >> 7;
            random ^= random << 17;
            raw.push(match block % 4 {
                0 => (random % 2) as u8,
                1 => random as u8,
                2 => {
                    b"lazy-matches-0123456789-lazy-matches-012X456789"
                        [i % b"lazy-matches-0123456789-lazy-matches-012X456789".len()]
                }
                _ => ((i % 257) ^ (i / 257)) as u8,
            });
        }
    }
    let expected = oracle(&raw);
    assert!(
        super::super::gzip::encode(&raw, 64 << 20).unwrap() == expected,
        "exact mixed-distribution gzip mismatch"
    );
    assert_eq!(
        super::super::gzip::decode(&expected, raw.len()).unwrap().0,
        raw
    );
}

#[test]
fn fixed_empty_and_malformed_deflate() {
    assert_eq!(decode(&[3, 0], 0).unwrap(), (Vec::new(), 2));
    for bytes in [&[7][..], &[1, 1, 0, 1, 0, b'x'], &[3, 2, 0], &[5, 0, 0, 0]] {
        assert!(decode(bytes, 1024).is_err(), "{bytes:?}");
    }
}

fn empty_dynamic_with_distance_count(count: usize) -> Vec<u8> {
    let mut bits = Vec::new();
    let mut write = |value: usize, count: usize| {
        bits.extend((0..count).map(|bit| ((value >> bit) & 1) as u8));
    };
    write(5, 3);
    write(0, 5);
    write(count - 1, 5);
    write(14, 4);
    for order in [16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1] {
        write(usize::from(order == 0 || order == 1), 3);
    }
    for _ in 0..256 {
        write(0, 1);
    }
    write(1, 1);
    write(1, 1);
    for _ in 1..count {
        write(0, 1);
    }
    write(0, 1);
    let mut gzip = vec![31, 139, 8, 0, 0, 0, 0, 0, 0, 255];
    gzip.extend(bits.chunks(8).map(|chunk| {
        chunk
            .iter()
            .enumerate()
            .fold(0, |byte, (at, bit)| byte | (bit << at))
    }));
    gzip.extend_from_slice(&[0; 8]);
    gzip
}

#[test]
fn dynamic_distance_code_count_bound() {
    for count in [30, 31, 32] {
        let gzip = empty_dynamic_with_distance_count(count);
        let result = super::super::gzip::decode(&gzip, 0);
        if count == 30 {
            assert_eq!(result.unwrap(), (Vec::new(), gzip.len()));
        } else {
            assert!(result.is_err(), "reserved distance count {count} accepted");
        }
        if let Ok(executable) = std::env::var("GOMAD_GO_GZIP_ORACLE") {
            let mut child = Command::new(executable)
                .args(["decode", "0"])
                .stdin(Stdio::piped())
                .stdout(Stdio::piped())
                .stderr(Stdio::null())
                .spawn()
                .unwrap();
            child.stdin.take().unwrap().write_all(&gzip).unwrap();
            let output = child.wait_with_output().unwrap();
            assert_eq!(
                output.status.success(),
                count == 30,
                "Go distance count {count}"
            );
            if count == 30 {
                assert_eq!(output.stdout, Vec::<u8>::new());
            }
        }
    }
}
