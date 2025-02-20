// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package proptest

import (
	"fmt"
	"iter"
	"unicode"

	"pgregory.net/rapid"
)

type (
	Gen[V varTag[V]] struct {
		label    string
		gen      *rapid.Generator[V]
		variants []V
	}
	ActionGen[V varTag[V]] interface{}
	seeder                 interface {
		Seed() int
	}
)

func Just[T any](val T) *Gen[T] {
	return &Gen[T]{
		label: "just",
		gen:   rapid.Just(val),
	}
}

func GenRange(s seeder, min, max int) iter.Seq[int] {
	return func(yield func(int) bool) {
		times := GenInt(min, max).Next(s)
		for i := 0; i < times; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func GenInt(min, max int) *Gen[int] {
	return &Gen[int]{
		label: fmt.Sprintf("%d,%d", min, max),
		gen:   rapid.IntRange(min, max),
	}
}

func GenId(prefix string) *Gen[string] {
	return &Gen[string]{
		label: prefix,
		gen: rapid.Custom(func(t *rapid.T) string {
			return prefix + "-" + rapid.StringOfN(rapid.RuneFrom(nil, unicode.ASCII_Hex_Digit), 12, 12, -1).Draw(t, "")
		})}
}

//func GenPick[T any](variants []T) *Gen[T] {
//	return &Gen[T]{
//		label:    getTypeName[T](),
//		gen:      rapid.SampledFrom(variants),
//		variants: variants,
//	}
//}

func (g *Gen[T]) Next(s seeder) T {
	return g.gen.Example(s.Seed())
}

func (g *Gen[T]) Variants() []any {
	var variants []any
	for _, v := range g.variants {
		variants = append(variants, v)
	}
	return variants
}
