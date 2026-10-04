# World preimage preservation oracle

Production Go sources copied read-only from base `0dd05b313acd0986312da7fd3159520e6a21f1bf`
using `git show`; no checkout, worktree, commit, or source-tree mutation. The
re-anchor baseline had no World production edits. Scratch module is self-contained
World plus canonicaljson, with stock Go 1.27.1 and GOWORK off.

Executed once after the corrective production edits, not claimed as pre-edit
TDD. It independently reconstructs the original implementation and compares
all eight default category/detail values plus complete capacity/replay fields,
escaped/non-ASCII/8192-byte detail, snapshots/digests, complete recording bytes
and decode values. The retained original sources and preservation_test.go are
the executable substrate; observed SHA256 values are pinned in the current
production preservation test.

Command: `GOWORK=off PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH go test -v -count=1 -tags test_dep ./world`

Start/end: 2026-10-04T05:01:42Z. Exit 0; World package 0.004s, 10 subtests passed.
Actual host Linux/arm64. No native qualification or formal review claim.
