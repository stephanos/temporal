package adapterregen

import (
	"fmt"
	"strings"
)

const diffContext = 3

type diffOp struct {
	kind byte // ' ', '-', or '+'
	line string
}

// unifiedDiff renders the line changes from previous to candidate as a
// unified diff with three lines of context, or "" when they are equal.
func unifiedDiff(name string, previous, candidate []byte) string {
	if string(previous) == string(candidate) {
		return ""
	}
	ops := diffLines(splitLines(string(previous)), splitLines(string(candidate)))
	var out strings.Builder
	fmt.Fprintf(&out, "--- previous/%s\n+++ candidate/%s\n", name, name)
	for start := 0; start < len(ops); {
		for start < len(ops) && ops[start].kind == ' ' {
			start++
		}
		if start == len(ops) {
			break
		}
		first := max(start-diffContext, 0)
		last := start
		for index := start; index < len(ops); index++ {
			if ops[index].kind != ' ' {
				last = index
			} else if index-last > 2*diffContext {
				break
			}
		}
		end := min(last+diffContext+1, len(ops))
		previousLine, candidateLine := 1, 1
		for _, op := range ops[:first] {
			if op.kind != '+' {
				previousLine++
			}
			if op.kind != '-' {
				candidateLine++
			}
		}
		previousCount, candidateCount := 0, 0
		for _, op := range ops[first:end] {
			if op.kind != '+' {
				previousCount++
			}
			if op.kind != '-' {
				candidateCount++
			}
		}
		fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n", previousLine, previousCount, candidateLine, candidateCount)
		for _, op := range ops[first:end] {
			out.WriteByte(op.kind)
			out.WriteString(op.line)
			if !strings.HasSuffix(op.line, "\n") {
				out.WriteString("\n\\ No newline at end of file\n")
			}
		}
		start = end
	}
	return out.String()
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// diffLines computes a shortest edit script with Myers' algorithm.
func diffLines(a, b []string) []diffOp {
	n, m := len(a), len(b)
	limit := n + m
	offset := limit + 1
	frontier := make([]int, 2*limit+3)
	var trace [][]int
	for depth := 0; depth <= limit; depth++ {
		trace = append(trace, append([]int(nil), frontier...))
		for diagonal := -depth; diagonal <= depth; diagonal += 2 {
			var x int
			if diagonal == -depth || diagonal != depth && frontier[offset+diagonal-1] < frontier[offset+diagonal+1] {
				x = frontier[offset+diagonal+1]
			} else {
				x = frontier[offset+diagonal-1] + 1
			}
			y := x - diagonal
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			frontier[offset+diagonal] = x
			if x >= n && y >= m {
				return backtrack(trace, a, b, depth, offset)
			}
		}
	}
	return nil
}

func backtrack(trace [][]int, a, b []string, depth, offset int) []diffOp {
	x, y := len(a), len(b)
	var reversed []diffOp
	for ; depth > 0; depth-- {
		frontier := trace[depth]
		diagonal := x - y
		var previousDiagonal int
		if diagonal == -depth || diagonal != depth && frontier[offset+diagonal-1] < frontier[offset+diagonal+1] {
			previousDiagonal = diagonal + 1
		} else {
			previousDiagonal = diagonal - 1
		}
		previousX := frontier[offset+previousDiagonal]
		previousY := previousX - previousDiagonal
		for x > previousX && y > previousY {
			x--
			y--
			reversed = append(reversed, diffOp{' ', a[x]})
		}
		if x == previousX {
			y--
			reversed = append(reversed, diffOp{'+', b[y]})
		} else {
			x--
			reversed = append(reversed, diffOp{'-', a[x]})
		}
	}
	for x > 0 && y > 0 {
		x--
		y--
		reversed = append(reversed, diffOp{' ', a[x]})
	}
	ops := make([]diffOp, len(reversed))
	for index, op := range reversed {
		ops[len(reversed)-1-index] = op
	}
	return ops
}
