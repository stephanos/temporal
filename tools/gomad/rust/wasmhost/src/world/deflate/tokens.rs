// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by the BSD-style license in GO-LICENSE.
// Adapted from Go 1.27.1 compress/flate/token.go.

pub(super) const LENGTH_BASE: [usize; 29] = [
    0, 1, 2, 3, 4, 5, 6, 7, 8, 10, 12, 14, 16, 20, 24, 28, 32, 40, 48, 56, 64, 80, 96, 112, 128,
    160, 192, 224, 255,
];
pub(super) const LENGTH_EXTRA: [u8; 29] = [
    0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0,
];
pub(super) const OFFSET_BASE: [usize; 30] = [
    0, 1, 2, 3, 4, 6, 8, 12, 16, 24, 32, 48, 64, 96, 128, 192, 256, 384, 512, 768, 1024, 1536,
    2048, 3072, 4096, 6144, 8192, 12288, 16384, 24576,
];
pub(super) const OFFSET_EXTRA: [u8; 30] = [
    0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13,
    13,
];

#[derive(Clone, Copy)]
pub(super) enum Token {
    Literal(u8),
    Match { length: usize, offset: usize },
    End,
}

pub(super) fn length_code(length: usize) -> usize {
    LENGTH_BASE.iter().rposition(|&b| b <= length).unwrap()
}
pub(super) fn offset_code(offset: usize) -> usize {
    OFFSET_BASE.iter().rposition(|&b| b <= offset).unwrap()
}

pub(super) struct Tokens {
    pub data: Vec<Token>,
    pub literals: [u16; 286],
    pub offsets: [u16; 30],
}

impl Tokens {
    pub fn new() -> Self {
        Self {
            data: Vec::with_capacity(65536),
            literals: [0; 286],
            offsets: [0; 30],
        }
    }
    pub fn literal(&mut self, b: u8) {
        self.data.push(Token::Literal(b));
        self.literals[usize::from(b)] += 1;
    }
    pub fn matches(&mut self, mut length: usize, offset: usize) {
        while length > 0 {
            let n = if length > 261 {
                258
            } else if length > 258 {
                255
            } else {
                length
            };
            length -= n;
            self.data.push(Token::Match {
                length: n - 3,
                offset,
            });
            self.literals[257 + length_code(n - 3)] += 1;
            self.offsets[offset_code(offset)] += 1;
        }
    }
    pub fn end(&mut self) {
        self.data.push(Token::End);
        self.literals[256] += 1;
    }
    pub fn estimated_bits(&self) -> usize {
        fn log2(v: f32) -> f32 {
            let mut ux = v.to_bits() as i32;
            let mut log = (((ux >> 23) & 255) - 128) as f32;
            ux &= -0x7f80_0001;
            ux += 127 << 23;
            let u = f32::from_bits(ux as u32);
            log += ((-0.344_848_43_f32) * u + 2.024_665_8_f32) * u - 0.674_877_6_f32;
            log
        }
        let mut shannon = 0f32;
        let mut extra = 0;
        let mut matches = 0usize;
        let total = self.data.len();
        if total > 0 {
            let inv = 1.0 / total as f32;
            for &v in &self.literals[..256] {
                if v > 0 {
                    let n = f32::from(v);
                    shannon += (-log2(n * inv)).clamp(1.0, 15.0) * n;
                }
            }
            shannon += 15.0;
            for (i, &v) in self.literals[257..].iter().enumerate() {
                if v > 0 {
                    let n = f32::from(v);
                    shannon += (-log2(n * inv)).clamp(1.0, 15.0) * n;
                    // Go's estimate indexes extraHist[1:] from zero.
                    extra += usize::from(LENGTH_EXTRA[i]) * usize::from(v);
                    matches += usize::from(v);
                }
            }
        }
        if matches > 0 {
            let inv = 1.0 / matches as f32;
            for (i, &v) in self.offsets.iter().enumerate() {
                if v > 0 {
                    let n = f32::from(v);
                    shannon += (-log2(n * inv)).clamp(1.0, 15.0) * n;
                    extra += usize::from(OFFSET_EXTRA[i]) * usize::from(v);
                }
            }
        }
        shannon as usize + extra
    }
}
