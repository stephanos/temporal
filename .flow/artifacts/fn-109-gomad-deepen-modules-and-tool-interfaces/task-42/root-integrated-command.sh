#!/usr/bin/env bash
set -u
set -o pipefail
cd /Users/stephan/Workspace/skunkworks/gomad/temporal || exit 90
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
export GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS= GOMAXPROCS=2
unset GOMADSEED GOMAD3_CHILD_SEED GOMAD3_SEED
scratch=/tmp/fn109-policy-integrated.hwEkegMn
sha256sum -c .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-41/root-integrated-inputs.sha256 > "$scratch/input-check.log" || exit 91
sed '\@  tools/gomad3/internal/compatibilitypack/policy.go$@d' .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42/admission-base/source.sha256 > "$scratch/protected.sha256"
sha256sum -c "$scratch/protected.sha256" > "$scratch/protected-check.log" || exit 92
sha256sum tools/gomad3/internal/compatibilitypack/policy.go tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go > "$scratch/candidate.sha256"
git rev-parse HEAD > "$scratch/admission-head.txt"
date -u '+%Y-%m-%dT%H:%M:%SZ' > "$scratch/start.txt"
started_ns=$(date +%s%N)
make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype > "$scratch/gate.log" 2>&1
gate_status=$?
ended_ns=$(date +%s%N)
date -u '+%Y-%m-%dT%H:%M:%SZ' > "$scratch/end.txt"
printf 'exit_code=%s\nelapsed_ns=%s\n' "$gate_status" "$((ended_ns-started_ns))" > "$scratch/result.txt"
sha256sum -c .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-41/root-integrated-inputs.sha256 >> "$scratch/input-check.log" || exit 93
sha256sum -c "$scratch/protected.sha256" >> "$scratch/protected-check.log" || exit 94
sha256sum -c "$scratch/candidate.sha256" > "$scratch/candidate-check.log" || exit 95
cat "$scratch/result.txt"
tail -n 13 "$scratch/gate.log"
exit "$gate_status"
