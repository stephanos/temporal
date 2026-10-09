import {spawnSync} from 'node:child_process';
import path from 'node:path';

const runner = path.join(path.dirname(new URL(import.meta.url).pathname), 'run.mjs');
const commands = [
  ['task40-generated-serialized', 'GOFLAGS=-tags=test_dep make -C tools/gomad3 validate'],
  ['task40-source-linux', 'cd tools/gomad3 && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go list -json -tags test_dep ./target/... ./internal/hostexec'],
  ['task40-source-darwin', 'cd tools/gomad3 && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go list -json -tags test_dep ./target/... ./internal/hostexec'],
  ['task40-format', 'bash .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/source-acceptance-20261008/format-gate.sh gofmt -l tools/gomad3/target/internal/gocommand/command_test.go && git diff --check'],
  ['task40-lint-scoped', 'cd tools/gomad3 && /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./internal/hostexec ./target/internal/gocommand ./target/internal/capabilityreview'],
  ['task40-lint-task9-scope', 'cd tools/gomad3 && /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./target/... ./internal/hostexec'],
  ['task40-lint-integrated', 'make lint-code-gomad3 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype GOLANGCI_LINT_BASE_REV=d635e23f00d926a43b942f25a9d05bd0ccb72025'],
  ['task40-lint-fast', 'make lint-code-fast GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype GOLANGCI_LINT_BASE_REV=4ce2d847af']
];
for (const [name, command] of commands) {
  const result = spawnSync(process.execPath, [runner, name, command], {stdio: 'inherit'});
  if (result.error || result.signal || ![0, 1].includes(result.status)) {
    throw new Error(`${name}: runner failed: ${result.error ?? result.signal ?? result.status}`);
  }
}
