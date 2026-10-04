#!/usr/bin/env bash
set -eu
cd /Users/stephan/Workspace/skunkworks/gomad/temporal
artifact=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-24
git show ce80d2425cf34da103939b5aa23f90bde1c2092f:.github/.golangci.yml > "$artifact/config-before.yml"
cp .github/.golangci.yml "$artifact/config-after.yml"
awk '/path:|path-except:|^      - \^/ {print NR ":" $0}' "$artifact/config-before.yml" | od -An -tx1 -c > "$artifact/path-expression-bytes.log"
git diff -- .github/.golangci.yml > "$artifact/config-change.diff"
git diff --exit-code ce80d2425cf34da103939b5aa23f90bde1c2092f -- '*.go' go.mod go.sum tools/gomad3/go.mod tools/gomad3/go.sum tests/mixedbrain/go.mod tests/mixedbrain/go.sum Makefile tools/gomad3/Makefile .github/workflows .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-23 > "$artifact/preservation.log"
git blame -L 157,174 -- tools/gomad3sim/controller.go > "$artifact/root-finding-blame.log"
jq -Rn '[inputs | select(test("^tools/gomad3/[^:]+:[0-9]+:[0-9]+:")) | capture("^(?<path>[^:]+):(?<line>[0-9]+):(?<column>[0-9]+): (?<message>.*) \\((?<linter>[^()]*)\\)$") | .line |= tonumber | .column |= tonumber | . + {owner:(.path|split("/")|.[0:-1]|join("/"))}]' "$artifact/gomad-lint.log" > "$artifact/gomad-findings.json"
jq 'group_by(.owner) | map({owner:.[0].owner,count:length,linters:(group_by(.linter)|map({linter:.[0].linter,count:length})),files:(map(.path)|unique)})' "$artifact/gomad-findings.json" > "$artifact/gomad-finding-owners.json"
jq -e 'length == 419' "$artifact/gomad-findings.json"
sha256sum "$artifact/config-before.yml" "$artifact/config-after.yml" cmd/tools/lintcode/lint_policy_test.go > "$artifact/final-policy.sha256"
git diff --check -- .github/.golangci.yml > "$artifact/scoped-diff-check.log"
/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt -l cmd/tools/lintcode/lint_policy_test.go >> "$artifact/scoped-diff-check.log"
test ! -s "$artifact/scoped-diff-check.log"
