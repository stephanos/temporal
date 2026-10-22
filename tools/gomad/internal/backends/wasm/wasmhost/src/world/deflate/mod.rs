// Copyright 2009, 2016, 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by the BSD-style license in GO-LICENSE.
// Go 1.27.1 default DEFLATE algorithm; provenance is recorded in README.md.

#![expect(
    clippy::cast_possible_truncation,
    clippy::cast_possible_wrap,
    clippy::cast_sign_loss,
    clippy::cast_precision_loss,
    clippy::many_single_char_names,
    clippy::similar_names,
    clippy::too_many_lines,
    reason = "Preserve the pinned Go matcher, integer bit operations, and f32 entropy estimate; the caller caps input at 384 MiB."
)]

mod huffman;
mod matcher;
mod reader;
mod tokens;
mod writer;
pub(super) use reader::decode;

pub(super) fn encode(raw: &[u8], limit: usize) -> Result<Vec<u8>, String> {
    let mut writer = writer::Writer::new(limit);
    let mut matcher = matcher::Matcher::new();
    let mut chunks = raw.chunks(65535).peekable();
    while let Some(input) = chunks.next() {
        let sync = chunks.peek().is_none();
        if input.len() <= 32 {
            writer.stored(input);
        } else if input.len() < 128 {
            writer.huffman(input, true);
        } else {
            let mut tokens = matcher.encode(input);
            if tokens.data.is_empty() {
                writer.stored(input);
            } else if tokens.data.len() > input.len() - (input.len() >> 4) {
                writer.huffman(input, sync);
            } else {
                writer.dynamic(&mut tokens, input, sync);
            }
        }
        if writer.exceeded() {
            return Err("World compressed byte bound".into());
        }
    }
    writer.finish()
}
#[cfg(test)]
mod tests;
