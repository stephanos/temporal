// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by the BSD-style license in deflate/GO-LICENSE.
// Adapted from Go 1.27.1 compress/gzip/{gzip,gunzip}.go.

use super::deflate;

const MAX_EXPANDED: usize = 384 << 20;
const MAX_COMPRESSED: usize = 64 << 20;

pub(super) fn encode(raw: &[u8], limit: usize) -> Result<Vec<u8>, String> {
    if raw.len() > MAX_EXPANDED {
        return Err("World expanded byte bound".into());
    }
    if limit > MAX_COMPRESSED {
        return Err("World compressed byte bound".into());
    }
    let body_limit = limit.checked_sub(18).ok_or("World compressed byte bound")?;
    let body = deflate::encode(raw, body_limit)?;
    let mut gzip = Vec::with_capacity(body.len() + 18);
    gzip.extend_from_slice(&[31, 139, 8, 0, 0, 0, 0, 0, 0, 255]);
    gzip.extend_from_slice(&body);
    gzip.extend_from_slice(&crc32fast::hash(raw).to_le_bytes());
    gzip.extend_from_slice(
        &u32::try_from(raw.len())
            .map_err(|_| "World expanded size overflow")?
            .to_le_bytes(),
    );
    Ok(gzip)
}

pub(super) fn decode(bytes: &[u8], limit: usize) -> Result<(Vec<u8>, usize), String> {
    if limit > MAX_EXPANDED {
        return Err("World expanded byte bound".into());
    }
    if bytes.len() > MAX_COMPRESSED {
        return Err("World compressed byte bound".into());
    }
    let header = bytes.get(..10).ok_or("World gzip header truncated")?;
    if header[..3] != [31, 139, 8] {
        return Err("World gzip invalid header".into());
    }
    let flags = header[3];
    let mut at = 10usize;
    if flags & 4 != 0 {
        let len = bytes
            .get(at..at + 2)
            .ok_or("World gzip extra length truncated")?;
        let len = usize::from(u16::from_le_bytes(len.try_into().unwrap()));
        at += 2;
        bytes
            .get(at..at + len)
            .ok_or("World gzip extra truncated")?;
        at += len;
    }
    for flag in [8, 16] {
        if flags & flag != 0 {
            let tail = bytes.get(at..).ok_or("World gzip header truncated")?;
            let count = tail
                .iter()
                .take(512)
                .position(|&b| b == 0)
                .ok_or("World gzip invalid header string")?
                + 1;
            at += count;
        }
    }
    if flags & 2 != 0 {
        let expected = bytes
            .get(at..at + 2)
            .ok_or("World gzip header checksum truncated")?;
        if crc32fast::hash(&bytes[..at]).to_le_bytes()[..2] != *expected {
            return Err("World gzip invalid header checksum".into());
        }
        at += 2;
    }
    let (raw, consumed) = deflate::decode(&bytes[at..], limit)?;
    at += consumed;
    let footer = bytes.get(at..at + 8).ok_or("World gzip footer truncated")?;
    if crc32fast::hash(&raw) != u32::from_le_bytes(footer[..4].try_into().unwrap())
        || usize::try_from(u32::from_le_bytes(footer[4..].try_into().unwrap()))
            .map_err(|_| "World gzip size overflow")?
            != raw.len()
    {
        return Err("World gzip invalid checksum".into());
    }
    Ok((raw, at + 8))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn canonical_empty_and_small_streams() {
        let empty = [
            31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        ];
        assert_eq!(encode(&[], 20).unwrap(), empty);
        assert_eq!(decode(&empty, 0).unwrap(), (Vec::new(), empty.len()));
        assert!(encode(&[], 19).is_err());
        for input in [b"a".as_slice(), b"hello world", &[0; 256]] {
            let gzip = encode(input, 4096).unwrap();
            assert_eq!(
                decode(&gzip, input.len()).unwrap(),
                (input.to_vec(), gzip.len())
            );
            assert!(decode(&gzip, input.len() - 1).is_err());
            for end in 0..gzip.len() {
                assert!(decode(&gzip[..end], input.len()).is_err(), "prefix={end}");
            }
            for offset in [0, 1, 2, gzip.len() - 8, gzip.len() - 4] {
                let mut corrupt = gzip.clone();
                corrupt[offset] ^= 1;
                assert!(decode(&corrupt, input.len()).is_err(), "offset={offset}");
            }
        }
    }

    #[test]
    fn optional_headers_single_member_and_header_crc() {
        let raw = b"bounded World wire";
        let canonical = encode(raw, 4096).unwrap();
        let mut gzip = canonical[..10].to_vec();
        gzip[3] = 2 | 4 | 8 | 16 | 0xe0;
        gzip[4..10].copy_from_slice(&[1, 2, 3, 4, 5, 6]);
        gzip.extend_from_slice(&[3, 0, 1, 2, 3]);
        gzip.extend_from_slice(b"name\0comment\0");
        let crc = crc32fast::hash(&gzip).to_le_bytes();
        gzip.extend_from_slice(&crc[..2]);
        gzip.extend_from_slice(&canonical[10..]);
        let member_len = gzip.len();
        gzip.extend_from_slice(&canonical);
        assert_eq!(
            decode(&gzip, raw.len()).unwrap(),
            (raw.to_vec(), member_len)
        );
        gzip[20] ^= 1;
        assert!(
            decode(&gzip, raw.len())
                .unwrap_err()
                .contains("header checksum")
        );
        let mut long_name = canonical[..10].to_vec();
        long_name[3] = 8;
        long_name.extend_from_slice(&[b'a'; 512]);
        long_name.push(0);
        long_name.extend_from_slice(&canonical[10..]);
        assert!(decode(&long_name, raw.len()).is_err());
    }
}
