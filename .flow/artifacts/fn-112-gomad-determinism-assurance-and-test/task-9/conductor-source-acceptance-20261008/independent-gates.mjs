import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { out, worker, root, stock, read } from './capture.mjs';

assert.equal(process.env.GOCACHE, '/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r');
const proof = JSON.parse(read(worker + '/sealed-source-proof.json'));
const runner = [...new Set(proof.current_shared_assertion_bodies.filter(f => f.path.includes('/runner/')).map(f => f.name))];
runner.push('TestAssessWorldValidatesTheRecordAgainstItsSeed', 'TestAssessCompletionProjectsCoverageInOrderAndClassifies', 'TestIsolatedRunnerDrainsFastCoordinatorBeforeWaitClosesOutput', 'TestRunRejectsSuccessfulRetentionWithoutReplayTranscript');
const lint = '/tmp/fn109-lint-tools.ZdNe1t50', go = stock + '/go';
const common = ['ALL_TEST_TAGS=test_dep', 'GOLANGCI_LINT_FIX=false', 'GOLANGCI_LINT=' + lint + '/golangci-lint-v2.13.0', 'ERRORTYPE=' + lint + '/errortype'];
const commands = [
  ['independent-deterministicio', go, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-timeout=3m', '-json', './deterministicio'],
  ['independent-cli', go, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-timeout=2m', '-json', './cmd/gomad/internal/cli'],
  ['independent-runner-mapped', go, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-timeout=2m', '-json', '-run', '^(' + [...new Set(runner)].join('|') + ')$', './runner'],
  ['independent-validate', 'make', '-C', 'tools/gomad3', 'validate'],
  ['independent-scoped-vet', go, '-C', 'tools/gomad3', 'vet', '-tags', 'test_dep', '.', './deterministicio', './cmd/gomad/internal/cli', './runner'],
  ['independent-scoped-errortype', go, '-C', 'tools/gomad3', 'vet', '-tags', 'test_dep', '-vettool=' + lint + '/errortype', '-style-check=false', '.', './deterministicio', './cmd/gomad/internal/cli', './runner'],
  ['independent-fast-lint', 'make', 'lint-code-fast', ...common, 'GOLANGCI_LINT_BASE_REV=59ca3d17395be5501906586006e2539d33bea28c'],
  ['independent-scoped-lint', 'make', 'lint-code', ...common, 'GOLANGCI_LINT_BASE_REV=', 'LINT_CODE_DIR=' + root + '/tools/gomad3', 'LINT_CODE_TARGETS=. ./deterministicio/... ./runner/... ./cmd/gomad/...'],
  ['independent-focused-format', 'node', worker + '/format-check.mjs', 'focused'],
  ['independent-nested-format', 'node', worker + '/format-check.mjs', 'nested'],
  ['independent-checker-controls', 'python3', '-B', worker + '/checker-controls.py'],
];
for (const [label, ...argv] of commands) {
  const result = spawnSync('node', [out + '/capture.mjs', label, ...argv], { cwd: root, env: process.env, stdio: 'inherit', timeout: 610000 });
  assert.equal(result.signal, null, label);
  assert.notEqual(result.status, null, label);
}
