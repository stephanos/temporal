// Copyright 2016, 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by the BSD-style license in GO-LICENSE.
// Adapted from Go 1.27.1 compress/flate/{deflatefast,level6}.go.

use super::tokens::Tokens;

const HISTORY: usize = 65535 * 5;
const WINDOW: i32 = 32768;

pub(super) struct Matcher {
    history: Vec<u8>,
    cur: i32,
    short: Vec<i32>,
    long: Vec<[i32; 2]>,
}

fn load(src: &[u8], at: i32) -> u64 {
    u64::from_le_bytes(src[at as usize..at as usize + 8].try_into().unwrap())
}
fn load32(src: &[u8], at: i32) -> u32 {
    u32::from_le_bytes(src[at as usize..at as usize + 4].try_into().unwrap())
}
fn short_hash(v: u64) -> usize {
    (v as u32).wrapping_mul(2_654_435_761).wrapping_shr(17) as usize
}
fn long_hash(v: u64) -> usize {
    (v.wrapping_shl(8).wrapping_mul(58_295_818_150_454_627) >> 49) as usize
}
fn match_len(src: &[u8], s: i32, t: i32, limit: usize) -> i32 {
    let s = s as usize;
    let t = t as usize;
    let max = (src.len() - s).min(limit);
    let mut n = 0;
    while n + 8 <= max {
        let diff = load(src, (s + n) as i32) ^ load(src, (t + n) as i32);
        if diff != 0 {
            return (n + diff.trailing_zeros() as usize / 8) as i32;
        }
        n += 8;
    }
    while n < max && src[s + n] == src[t + n] {
        n += 1;
    }
    n as i32
}

impl Matcher {
    pub fn new() -> Self {
        Self {
            history: Vec::with_capacity(HISTORY),
            cur: 65535,
            short: vec![0; 32768],
            long: vec![[0; 2]; 32768],
        }
    }
    pub fn encode(&mut self, input: &[u8]) -> Tokens {
        if self.history.len() + input.len() > HISTORY {
            let offset = self.history.len() - WINDOW as usize;
            self.history.copy_within(offset.., 0);
            self.history.truncate(WINDOW as usize);
            self.cur += offset as i32;
        }
        let mut s = self.history.len() as i32;
        self.history.extend_from_slice(input);
        let src = &self.history;
        let mut dst = Tokens::new();
        // The caller handles final blocks below 128 bytes without match searching.
        let mut next_emit = s;
        let limit = src.len() as i32 - 11;
        let mut cv = load(src, s);
        let mut repeat = 1;
        'search: loop {
            let mut next_s = s;
            let mut l = 0;
            let mut t;
            loop {
                let hs = short_hash(cv);
                let hl = long_hash(cv);
                s = next_s;
                next_s = s + 1 + ((s - next_emit) >> 7);
                if next_s > limit {
                    break 'search;
                }
                let sc = self.short[hs];
                let lc = self.long[hl];
                let next = load(src, next_s);
                self.short[hs] = s + self.cur;
                self.long[hl] = [s + self.cur, self.long[hl][0]];
                let hs_next = short_hash(next);
                let hl_next = long_hash(next);
                t = lc[0] - self.cur;
                if s - t < WINDOW {
                    if cv as u32 == load32(src, t) {
                        self.short[hs_next] = next_s + self.cur;
                        self.long[hl_next] = [next_s + self.cur, self.long[hl_next][0]];
                        let t2 = lc[1] - self.cur;
                        if s - t2 < WINDOW && cv as u32 == load32(src, t2) {
                            l = match_len(src, s + 4, t + 4, 254) + 4;
                            let ml = match_len(src, s + 4, t2 + 4, 254) + 4;
                            if ml > l {
                                t = t2;
                                l = ml;
                            }
                        }
                        break;
                    }
                    t = lc[1] - self.cur;
                    if s - t < WINDOW && cv as u32 == load32(src, t) {
                        self.short[hs_next] = next_s + self.cur;
                        self.long[hl_next] = [next_s + self.cur, self.long[hl_next][0]];
                        break;
                    }
                }
                t = sc - self.cur;
                if s - t < WINDOW && cv as u32 == load32(src, t) {
                    l = match_len(src, s + 4, t + 4, 254) + 4;
                    let lc = self.long[hl_next];
                    self.short[hs_next] = next_s + self.cur;
                    self.long[hl_next] = [next_s + self.cur, self.long[hl_next][0]];
                    let t2 = s - repeat + 1;
                    if load32(src, t2) == (cv >> 8) as u32 {
                        let ml = match_len(src, s + 5, t2 + 4, 254) + 4;
                        if ml > l {
                            t = t2;
                            l = ml;
                            s += 1;
                            break;
                        }
                    }
                    let t2 = lc[0] - self.cur;
                    if next_s - t2 < WINDOW {
                        if load32(src, t2) == next as u32 {
                            let ml = match_len(src, next_s + 4, t2 + 4, 254) + 4;
                            if ml > l {
                                t = t2;
                                s = next_s;
                                l = ml;
                            }
                        }
                        let t2 = lc[1] - self.cur;
                        if next_s - t2 < WINDOW && load32(src, t2) == next as u32 {
                            let ml = match_len(src, next_s + 4, t2 + 4, 254) + 4;
                            if ml > l {
                                t = t2;
                                s = next_s;
                                l = ml;
                                break;
                            }
                        }
                    }
                    break;
                }
                cv = next;
            }
            if l == 0 {
                l = match_len(src, s + 4, t + 4, usize::MAX) + 4;
            } else if l == 258 {
                l += match_len(src, s + l, t + l, usize::MAX);
            }
            if s + l < limit {
                let entry = self.long[long_hash(load(src, s + l))];
                let mut t2 = entry[0] - self.cur - l + 2;
                let s2 = s + 2;
                let off = s2 - t2;
                if off < WINDOW {
                    if off > 0 && t2 >= 0 {
                        let l2 = match_len(src, s2, t2, usize::MAX);
                        if l2 > l {
                            t = t2;
                            l = l2;
                            s = s2;
                        }
                    }
                    t2 = entry[1] - self.cur - l + 2;
                    let off = s2 - t2;
                    if off > 0 && off < WINDOW && t2 >= 0 {
                        let l2 = match_len(src, s2, t2, usize::MAX);
                        if l2 > l {
                            t = t2;
                            l = l2;
                            s = s2;
                        }
                    }
                }
            }
            while t > 0 && s > next_emit && src[t as usize - 1] == src[s as usize - 1] {
                s -= 1;
                t -= 1;
                l += 1;
            }
            for &v in &src[next_emit as usize..s as usize] {
                dst.literal(v);
            }
            dst.matches(l as usize, (s - t - 1) as usize);
            repeat = s - t;
            s += l;
            next_emit = s;
            if next_s >= s {
                s = next_s + 1;
            }
            if s >= limit {
                let mut i = next_s + 1;
                while i < src.len() as i32 - 8 {
                    let value = load(src, i);
                    self.short[short_hash(value)] = i + self.cur;
                    let e = &mut self.long[long_hash(value)];
                    *e = [i + self.cur, e[0]];
                    i += 2;
                }
                break;
            }
            let mut i = next_s + 1;
            while i < s - 1 {
                let value = load(src, i);
                self.short[short_hash(value)] = i + self.cur;
                let e = &mut self.long[long_hash(value)];
                *e = [i + self.cur, e[0]];
                let e2 = &mut self.long[long_hash(value >> 8)];
                *e2 = [i + self.cur + 1, e2[0]];
                i += 2;
            }
            cv = load(src, s);
        }
        if !dst.data.is_empty() {
            for &v in &src[next_emit as usize..] {
                dst.literal(v);
            }
        }
        dst
    }
}
