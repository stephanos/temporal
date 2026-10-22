// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by the BSD-style license in GO-LICENSE.
// RFC 1951 decoder, following Go 1.27.1 compress/flate tree validity rules.

use super::tokens::{LENGTH_BASE, LENGTH_EXTRA, OFFSET_BASE, OFFSET_EXTRA};

struct BitReader<'a> {
    input: &'a [u8],
    at: usize,
    bits: u32,
    count: u8,
}
impl BitReader<'_> {
    fn read(&mut self, n: u8) -> Result<usize, String> {
        while self.count < n {
            let b = *self
                .input
                .get(self.at)
                .ok_or("World gzip deflate truncated")?;
            self.at += 1;
            self.bits |= u32::from(b) << self.count;
            self.count += 8;
        }
        let value = self.bits & ((1 << n) - 1);
        self.bits >>= n;
        self.count -= n;
        Ok(value as usize)
    }
    fn align(&mut self) {
        self.bits = 0;
        self.count = 0;
    }
}
struct Tree {
    counts: [usize; 16],
    symbols: Vec<usize>,
    max: usize,
}
impl Tree {
    fn new(lengths: &[usize]) -> Result<Self, String> {
        let mut counts = [0usize; 16];
        let mut max = 0;
        for &n in lengths {
            if n > 15 {
                return Err("World gzip invalid Huffman code length".into());
            }
            if n > 0 {
                counts[n] += 1;
                max = max.max(n);
            }
        }
        let mut code = 0usize;
        for &count in &counts[1..=max] {
            code = (code << 1) + count;
        }
        if max != 0 && code != 1 << max && !(code == 1 && max == 1) {
            return Err("World gzip invalid Huffman tree".into());
        }
        let mut symbols = Vec::with_capacity(lengths.len());
        for n in 1..=max {
            for (symbol, &len) in lengths.iter().enumerate() {
                if len == n {
                    symbols.push(symbol);
                }
            }
        }
        Ok(Self {
            counts,
            symbols,
            max,
        })
    }
    fn symbol(&self, bits: &mut BitReader<'_>) -> Result<usize, String> {
        let mut code = 0;
        let mut first = 0;
        let mut index = 0;
        for n in 1..=self.max {
            code = (code << 1) | bits.read(1)?;
            if code >= first && code - first < self.counts[n] {
                return Ok(self.symbols[index + code - first]);
            }
            index += self.counts[n];
            first = (first + self.counts[n]) << 1;
        }
        Err("World gzip invalid Huffman symbol".into())
    }
}
fn dynamic(bits: &mut BitReader<'_>) -> Result<(Tree, Tree), String> {
    const ORDER: [usize; 19] = [
        16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15,
    ];
    let nl = bits.read(5)? + 257;
    let nd = bits.read(5)? + 1;
    let nc = bits.read(4)? + 4;
    if nl > 286 {
        return Err("World gzip invalid literal code count".into());
    }
    if nd > 30 {
        return Err("World gzip invalid distance code count".into());
    }
    let mut codes = [0usize; 19];
    for &order in &ORDER[..nc] {
        codes[order] = bits.read(3)?;
    }
    let tree = Tree::new(&codes)?;
    let mut lengths = Vec::with_capacity(nl + nd);
    while lengths.len() < nl + nd {
        let symbol = tree.symbol(bits)?;
        let (value, count) = match symbol {
            0..=15 => (symbol, 1),
            16 => (
                *lengths.last().ok_or("World gzip missing repeated code")?,
                bits.read(2)? + 3,
            ),
            17 => (0, bits.read(3)? + 3),
            18 => (0, bits.read(7)? + 11),
            _ => return Err("World gzip invalid code-length symbol".into()),
        };
        if count > nl + nd - lengths.len() {
            return Err("World gzip code-length run exceeds tree".into());
        }
        lengths.resize(lengths.len() + count, value);
    }
    Ok((Tree::new(&lengths[..nl])?, Tree::new(&lengths[nl..])?))
}

fn reserve(output: &mut Vec<u8>, count: usize, limit: usize) -> Result<(), String> {
    let needed = output
        .len()
        .checked_add(count)
        .ok_or("World expanded size overflow")?;
    if needed > limit {
        return Err("World expanded byte bound".into());
    }
    if needed > output.capacity() {
        let target = needed
            .max(output.capacity().saturating_mul(2))
            .max(4096)
            .min(limit);
        output
            .try_reserve_exact(target - output.len())
            .map_err(|_| "World expanded allocation failed")?;
    }
    Ok(())
}

pub fn decode(input: &[u8], limit: usize) -> Result<(Vec<u8>, usize), String> {
    let mut bits = BitReader {
        input,
        at: 0,
        bits: 0,
        count: 0,
    };
    let mut output = Vec::new();
    loop {
        let final_block = bits.read(1)? != 0;
        match bits.read(2)? {
            0 => {
                bits.align();
                let n = bits.read(16)?;
                if n != (!bits.read(16)? & 65535) {
                    return Err("World gzip stored length mismatch".into());
                }
                if n > limit - output.len() {
                    return Err("World expanded byte bound".into());
                }
                let end = bits.at.checked_add(n).ok_or("World gzip size overflow")?;
                let data = input
                    .get(bits.at..end)
                    .ok_or("World gzip stored block truncated")?;
                reserve(&mut output, n, limit)?;
                output.extend_from_slice(data);
                bits.at = end;
            }
            kind @ (1 | 2) => {
                let (lit, dist) = if kind == 1 {
                    let lengths: Vec<usize> = (0..288)
                        .map(|c| match c {
                            144..256 => 9,
                            256..280 => 7,
                            _ => 8,
                        })
                        .collect();
                    (Tree::new(&lengths)?, Tree::new(&[5; 32])?)
                } else {
                    dynamic(&mut bits)?
                };
                loop {
                    match lit.symbol(&mut bits)? {
                        byte @ 0..=255 => {
                            if output.len() == limit {
                                return Err("World expanded byte bound".into());
                            }
                            reserve(&mut output, 1, limit)?;
                            output.push(byte as u8);
                        }
                        256 => break,
                        symbol @ 257..=285 => {
                            let lc = symbol - 257;
                            let length = LENGTH_BASE[lc] + 3 + bits.read(LENGTH_EXTRA[lc])?;
                            let dc = dist.symbol(&mut bits)?;
                            if dc >= 30 {
                                return Err("World gzip invalid distance code".into());
                            }
                            let offset = OFFSET_BASE[dc] + 1 + bits.read(OFFSET_EXTRA[dc])?;
                            if offset > output.len() {
                                return Err("World gzip distance exceeds history".into());
                            }
                            if length > limit - output.len() {
                                return Err("World expanded byte bound".into());
                            }
                            reserve(&mut output, length, limit)?;
                            for _ in 0..length {
                                output.push(output[output.len() - offset]);
                            }
                        }
                        _ => return Err("World gzip invalid literal code".into()),
                    }
                }
            }
            _ => return Err("World gzip invalid block type".into()),
        }
        if final_block {
            return Ok((output, bits.at));
        }
    }
}
