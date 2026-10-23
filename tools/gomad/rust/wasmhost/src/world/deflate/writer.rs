// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by the BSD-style license in GO-LICENSE.
// Adapted from Go 1.27.1 compress/flate/huffman_bit_writer.go.

use super::huffman::{self, Code};
use super::tokens::{
    LENGTH_BASE, LENGTH_EXTRA, OFFSET_BASE, OFFSET_EXTRA, Token, Tokens, length_code, offset_code,
};

const ORDER: [usize; 19] = [
    16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15,
];

pub(super) struct Writer {
    output: Vec<u8>,
    bits: u64,
    nbits: u8,
    limit: usize,
    overflow: bool,
    prev_header: usize,
    wrote_huffman: bool,
    literal: Vec<Code>,
    offset: Vec<Code>,
    fixed_literal: Vec<Code>,
    fixed_offset: Vec<Code>,
    codegen: Vec<(usize, usize)>,
    code_freq: [u16; 19],
    code_codes: Vec<Code>,
}
impl Writer {
    pub fn new(limit: usize) -> Self {
        Self {
            output: Vec::new(),
            bits: 0,
            nbits: 0,
            limit,
            overflow: false,
            prev_header: 0,
            wrote_huffman: false,
            literal: vec![Code::default(); 286],
            offset: vec![Code::default(); 30],
            fixed_literal: huffman::fixed_literal(),
            fixed_offset: huffman::fixed_offset(),
            codegen: Vec::new(),
            code_freq: [0; 19],
            code_codes: Vec::new(),
        }
    }
    fn byte(&mut self, byte: u8) {
        if self.output.len() < self.limit {
            self.output.push(byte);
        } else {
            self.overflow = true;
        }
    }
    fn bits(&mut self, bits: usize, n: u8) {
        self.bits |= (bits as u64) << self.nbits;
        self.nbits += n;
        while self.nbits >= 8 {
            self.byte(self.bits as u8);
            self.bits >>= 8;
            self.nbits -= 8;
        }
    }
    fn code(&mut self, code: Code) {
        self.bits(usize::from(code.bits), code.len);
    }
    fn align(&mut self) {
        if self.nbits != 0 {
            self.byte(self.bits as u8);
            self.bits = 0;
            self.nbits = 0;
        }
    }
    fn end_previous(&mut self) {
        if self.prev_header > 0 {
            self.code(self.literal[256]);
            self.prev_header = 0;
        }
    }
    pub fn stored(&mut self, input: &[u8]) {
        self.end_previous();
        self.bits(0, 3);
        self.align();
        let n = input.len() as u16;
        for byte in n.to_le_bytes().into_iter().chain((!n).to_le_bytes()) {
            self.byte(byte);
        }
        for &byte in input {
            self.byte(byte);
        }
    }
    fn fixed(&mut self, tokens: &mut Tokens, sync: bool) {
        self.end_previous();
        self.bits(2, 3);
        if !sync {
            tokens.end();
        }
        let lit = self.fixed_literal.clone();
        let off = self.fixed_offset.clone();
        self.tokens(tokens, &lit, &off);
    }
    fn tokens(&mut self, tokens: &Tokens, lit: &[Code], off: &[Code]) {
        for &token in &tokens.data {
            match token {
                Token::Literal(b) => self.code(lit[usize::from(b)]),
                Token::End => self.code(lit[256]),
                Token::Match { length, offset } => {
                    let lc = length_code(length);
                    self.code(lit[257 + lc]);
                    self.bits(length - LENGTH_BASE[lc], LENGTH_EXTRA[lc]);
                    let oc = offset_code(offset);
                    self.code(off[oc]);
                    self.bits(offset - OFFSET_BASE[oc], OFFSET_EXTRA[oc]);
                }
            }
        }
    }
    fn generate_codegen(&mut self, nl: usize, no: usize, lit: &[Code], off: &[Code]) {
        self.codegen.clear();
        self.code_freq.fill(0);
        let sizes: Vec<u8> = lit[..nl].iter().chain(&off[..no]).map(|c| c.len).collect();
        let mut at = 0;
        while at < sizes.len() {
            let size = sizes[at];
            let mut end = at + 1;
            while end < sizes.len() && sizes[end] == size {
                end += 1;
            }
            let mut count = end - at;
            if size != 0 {
                self.add_codegen(usize::from(size), 0);
                count -= 1;
                while count >= 3 {
                    let n = count.min(6);
                    self.add_codegen(16, n - 3);
                    count -= n;
                }
            } else {
                while count >= 11 {
                    let n = count.min(138);
                    self.add_codegen(18, n - 11);
                    count -= n;
                }
                if count >= 3 {
                    self.add_codegen(17, count - 3);
                    count = 0;
                }
            }
            for _ in 0..count {
                self.add_codegen(usize::from(size), 0);
            }
            at = end;
        }
        self.code_codes = huffman::generate(&self.code_freq, 7);
    }
    fn add_codegen(&mut self, code: usize, extra: usize) {
        self.codegen.push((code, extra));
        self.code_freq[code] += 1;
    }
    fn codegens(&self) -> usize {
        let mut n = 19;
        while n > 4 && self.code_freq[ORDER[n - 1]] == 0 {
            n -= 1;
        }
        n
    }
    fn header_size(&self) -> usize {
        17 + 3 * self.codegens()
            + huffman::bit_length(&self.code_codes, &self.code_freq)
            + usize::from(self.code_freq[16]) * 2
            + usize::from(self.code_freq[17]) * 3
            + usize::from(self.code_freq[18]) * 7
    }
    fn dynamic_header(&mut self, nl: usize, no: usize) {
        self.bits(4, 3);
        self.bits(nl - 257, 5);
        self.bits(no - 1, 5);
        let ng = self.codegens();
        self.bits(ng - 4, 4);
        for &order in &ORDER[..ng] {
            self.bits(usize::from(self.code_codes[order].len), 3);
        }
        for i in 0..self.codegen.len() {
            let (code, extra) = self.codegen[i];
            self.code(self.code_codes[code]);
            match code {
                16 => self.bits(extra, 2),
                17 => self.bits(extra, 3),
                18 => self.bits(extra, 7),
                _ => (),
            }
        }
    }
    pub fn huffman(&mut self, input: &[u8], sync: bool) {
        let mut freq = [0u16; 286];
        for &b in input {
            freq[usize::from(b)] += 1;
        }
        let stored = (input.len() + 5) * 8;
        if input.len() > 1024 {
            let avg = input.len() as f64 / 256.0;
            let max = (input.len() * 2) as f64;
            let mut abs = 0.0;
            for &v in &freq[..256] {
                let diff = f64::from(v) - avg;
                abs += diff * diff;
                if abs >= max {
                    break;
                }
            }
            if abs < max {
                self.stored(input);
                return;
            }
        }
        freq[256] = 1;
        let temp = huffman::generate(&freq[..257], 15);
        let mut estimate = huffman::can_encode(&temp, &freq[..257]);
        estimate += self.prev_header;
        if self.prev_header == 0 {
            estimate += 560;
        }
        estimate += estimate >> 7;
        if stored <= estimate {
            self.stored(input);
            return;
        }
        if self.prev_header > 0 {
            let reuse = huffman::can_encode(&self.literal, &freq[..256]);
            if estimate < reuse {
                self.end_previous();
            }
        }
        if self.prev_header == 0 {
            self.literal = temp;
            let mut off = vec![Code::default(); 30];
            off[0] = Code { bits: 0, len: 1 };
            let lit = self.literal.clone();
            self.generate_codegen(257, 1, &lit, &off);
            self.dynamic_header(257, 1);
            self.wrote_huffman = true;
            self.prev_header = self.header_size();
        }
        for &b in input {
            self.code(self.literal[usize::from(b)]);
        }
        if sync {
            self.end_previous();
            self.wrote_huffman = false;
        }
    }
    pub fn dynamic(&mut self, tokens: &mut Tokens, input: &[u8], sync: bool) {
        if sync {
            tokens.end();
        } else {
            tokens.literals[256] = 1;
        }
        if self.wrote_huffman && self.prev_header > 0 {
            self.end_previous();
            self.wrote_huffman = false;
        }
        if self.prev_header > 0
            && (huffman::can_encode(&self.literal, &tokens.literals) == i32::MAX as usize
                || huffman::can_encode(&self.offset, &tokens.offsets) == i32::MAX as usize)
        {
            self.end_previous();
        }
        let nl = tokens.literals.iter().rposition(|&f| f != 0).unwrap() + 1;
        let mut no = tokens
            .offsets
            .iter()
            .rposition(|&f| f != 0)
            .map_or(0, |p| p + 1);
        if no == 0 {
            tokens.offsets[0] = 1;
            no = 1;
        }
        let extra: usize = tokens.literals[257..]
            .iter()
            .zip(LENGTH_EXTRA)
            .map(|(&f, e)| usize::from(f) * usize::from(e))
            .sum::<usize>()
            + tokens
                .offsets
                .iter()
                .zip(OFFSET_EXTRA)
                .map(|(&f, e)| usize::from(f) * usize::from(e))
                .sum::<usize>();
        let stored = (input.len() + 5) * 8;
        let fixed_size = 3
            + huffman::bit_length(&self.fixed_literal, &tokens.literals)
            + huffman::bit_length(&self.fixed_offset, &tokens.offsets)
            + extra;
        if self.prev_header > 0 {
            let mut new_size = self.prev_header + tokens.estimated_bits();
            new_size += usize::from(self.literal[256].len) + (new_size >> 7);
            let reuse_size = huffman::bit_length(&self.literal, &tokens.literals)
                + huffman::bit_length(&self.offset, &tokens.offsets)
                + extra;
            let size = if new_size < reuse_size {
                self.end_previous();
                new_size
            } else {
                reuse_size
            };
            if tokens.data.len() < 250 && fixed_size + 7 < size {
                if stored <= size {
                    self.stored(input);
                } else {
                    self.fixed(tokens, sync);
                }
                return;
            }
            if stored <= size {
                self.stored(input);
                return;
            }
        }
        if self.prev_header == 0 {
            tokens.literals[256] = 1;
            self.literal = huffman::generate(&tokens.literals, 15);
            self.offset = huffman::generate(&tokens.offsets, 15);
            let lit = self.literal.clone();
            let off = self.offset.clone();
            self.generate_codegen(nl, no, &lit, &off);
            let size = self.header_size()
                + huffman::bit_length(&lit, &tokens.literals)
                + huffman::bit_length(&off, &tokens.offsets)
                + extra;
            if tokens.data.len() < 250 && fixed_size <= size {
                if stored <= fixed_size {
                    self.stored(input);
                } else {
                    self.fixed(tokens, sync);
                }
                return;
            }
            if stored <= size {
                self.stored(input);
                return;
            }
            self.dynamic_header(nl, no);
            if !sync {
                self.prev_header = self.header_size();
            }
            self.wrote_huffman = false;
        }
        if sync {
            self.prev_header = 0;
        }
        let lit = self.literal.clone();
        let off = self.offset.clone();
        self.tokens(tokens, &lit, &off);
    }
    pub fn finish(mut self) -> Result<Vec<u8>, String> {
        self.end_previous();
        self.bits(3, 3);
        self.bits(0, 7);
        self.align();
        if self.overflow {
            Err("World compressed byte bound".into())
        } else {
            Ok(self.output)
        }
    }
    pub fn exceeded(&self) -> bool {
        self.overflow
    }
}
