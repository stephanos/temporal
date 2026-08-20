// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadchoicewire

import "encoding/binary"

type Hasher struct {
	state [8]uint32
	block [64]byte
	used  uint32
	bytes uint64
}

func NewHasher() *Hasher {
	return &Hasher{state: [8]uint32{0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19}}
}

func (hasher *Hasher) Write(input []byte) {
	hasher.bytes += uint64(len(input))
	for len(input) != 0 {
		count := min(len(input), len(hasher.block)-int(hasher.used))
		copy(hasher.block[hasher.used:], input[:count])
		hasher.used += uint32(count)
		input = input[count:]
		if hasher.used == uint32(len(hasher.block)) {
			hasher.compress()
			hasher.used = 0
		}
	}
}

func (hasher *Hasher) Sum() [DigestBytes]byte {
	cloned := *hasher
	length := cloned.bytes * 8
	cloned.block[cloned.used] = 0x80
	cloned.used++
	if cloned.used > 56 {
		clear(cloned.block[cloned.used:])
		cloned.compress()
		cloned.used = 0
	}
	clear(cloned.block[cloned.used:56])
	binary.BigEndian.PutUint64(cloned.block[56:64], length)
	cloned.compress()
	var result [DigestBytes]byte
	for index, value := range cloned.state {
		binary.BigEndian.PutUint32(result[index*4:index*4+4], value)
	}
	return result
}

func (hasher *Hasher) compress() {
	var words [64]uint32
	for index := 0; index < 16; index++ {
		words[index] = binary.BigEndian.Uint32(hasher.block[index*4 : index*4+4])
	}
	for index := 16; index < 64; index++ {
		s0 := rotateRight(words[index-15], 7) ^ rotateRight(words[index-15], 18) ^ words[index-15]>>3
		s1 := rotateRight(words[index-2], 17) ^ rotateRight(words[index-2], 19) ^ words[index-2]>>10
		words[index] = words[index-16] + s0 + words[index-7] + s1
	}
	a, b, c, d := hasher.state[0], hasher.state[1], hasher.state[2], hasher.state[3]
	e, f, g, h := hasher.state[4], hasher.state[5], hasher.state[6], hasher.state[7]
	for index := 0; index < 64; index++ {
		s1 := rotateRight(e, 6) ^ rotateRight(e, 11) ^ rotateRight(e, 25)
		choice := e&f ^ (^e)&g
		temporary1 := h + s1 + choice + sha256Constants[index] + words[index]
		s0 := rotateRight(a, 2) ^ rotateRight(a, 13) ^ rotateRight(a, 22)
		majority := a&b ^ a&c ^ b&c
		temporary2 := s0 + majority
		h, g, f, e, d, c, b, a = g, f, e, d+temporary1, c, b, a, temporary1+temporary2
	}
	hasher.state[0] += a
	hasher.state[1] += b
	hasher.state[2] += c
	hasher.state[3] += d
	hasher.state[4] += e
	hasher.state[5] += f
	hasher.state[6] += g
	hasher.state[7] += h
}
