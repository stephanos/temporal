// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by the BSD-style license in GO-LICENSE.
// Adapted from Go 1.27.1 compress/flate/huffman_code.go.

#[derive(Clone, Copy, Default)]
pub(super) struct Code {
    pub bits: u16,
    pub len: u8,
}

#[derive(Clone, Copy, Default)]
struct Level {
    last: i32,
    next_char: i32,
    next_pair: i32,
    needed: i32,
}

pub(super) fn generate(freq: &[u16], max_bits: usize) -> Vec<Code> {
    let mut codes = vec![Code::default(); freq.len()];
    let mut list: Vec<(usize, u16)> = freq
        .iter()
        .copied()
        .enumerate()
        .filter(|&(_, f)| f != 0)
        .collect();
    let n = list.len();
    if n <= 2 {
        for (i, &(literal, _)) in list.iter().enumerate() {
            codes[literal] = Code {
                bits: i as u16,
                len: 1,
            };
        }
        return codes;
    }
    list.sort_unstable_by_key(|&(lit, f)| (f, lit));
    list.push((usize::MAX, u16::MAX));
    let max_bits = max_bits.min(n - 1);
    let mut levels = [Level::default(); 16];
    let mut leaves = [[0usize; 16]; 16];
    for level in 1..=max_bits {
        levels[level] = Level {
            last: i32::from(list[1].1),
            next_char: i32::from(list[2].1),
            next_pair: if level == 1 {
                i32::MAX
            } else {
                i32::from(list[0].1) + i32::from(list[1].1)
            },
            needed: 0,
        };
        leaves[level][level] = 2;
    }
    levels[max_bits].needed = (2 * n - 4) as i32;
    let mut level = max_bits;
    loop {
        if levels[level].next_pair == i32::MAX && levels[level].next_char == i32::MAX {
            levels[level].needed = 0;
            levels[level + 1].next_pair = i32::MAX;
            level += 1;
            continue;
        }
        let prev = levels[level].last;
        if levels[level].next_char < levels[level].next_pair {
            let k = leaves[level][level] + 1;
            levels[level].last = levels[level].next_char;
            leaves[level][level] = k;
            levels[level].next_char = if list[k].0 == usize::MAX {
                i32::MAX
            } else {
                i32::from(list[k].1)
            };
        } else {
            levels[level].last = levels[level].next_pair;
            let save = leaves[level][level];
            leaves[level] = leaves[level - 1];
            leaves[level][level] = save;
            levels[level - 1].needed = 2;
        }
        levels[level].needed -= 1;
        if levels[level].needed == 0 {
            if level == max_bits {
                break;
            }
            levels[level + 1].next_pair = prev + levels[level].last;
            level += 1;
        } else {
            while levels[level - 1].needed > 0 {
                level -= 1;
            }
        }
    }
    debug_assert_eq!(leaves[max_bits][max_bits], n);
    list.truncate(n);
    let mut code = 0u16;
    for bits in 1..=max_bits {
        code <<= 1;
        let count = leaves[max_bits][max_bits + 1 - bits] - leaves[max_bits][max_bits - bits];
        let start = list.len() - count;
        list[start..].sort_unstable_by_key(|&(lit, _)| lit);
        for &(lit, _) in &list[start..] {
            codes[lit] = Code {
                bits: code.reverse_bits() >> (16 - bits),
                len: bits as u8,
            };
            code += 1;
        }
        list.truncate(start);
    }
    codes
}

pub(super) fn bit_length(codes: &[Code], freq: &[u16]) -> usize {
    codes
        .iter()
        .zip(freq)
        .map(|(c, &f)| usize::from(c.len) * usize::from(f))
        .sum()
}

pub(super) fn can_encode(codes: &[Code], freq: &[u16]) -> usize {
    if codes.iter().zip(freq).any(|(c, &f)| f != 0 && c.len == 0) {
        i32::MAX as usize
    } else {
        bit_length(codes, freq)
    }
}

pub(super) fn fixed_literal() -> Vec<Code> {
    (0..286u16)
        .map(|c| {
            let (bits, len) = match c {
                0..144 => (c + 48, 8),
                144..256 => (c + 256, 9),
                256..280 => (c - 256, 7),
                _ => (c - 88, 8),
            };
            Code {
                bits: bits.reverse_bits() >> (16 - len),
                len,
            }
        })
        .collect()
}

pub(super) fn fixed_offset() -> Vec<Code> {
    (0..30u16)
        .map(|bits| Code {
            bits: bits.reverse_bits() >> 11,
            len: 5,
        })
        .collect()
}
