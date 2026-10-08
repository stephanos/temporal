import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { out, worker, root, stock, read } from './capture.mjs';

assert.equal(process.env.GOCACHE, '/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r');
const go = stock + '/go';
const commands = [];
for (const [label, original, path, count] of [
  ['independent-portable-deterministicio', 'sealed-final-deterministicio', './deterministicio', 36],
  ['independent-portable-cli', 'sealed-final-cli', './cmd/gomad/internal/cli', 90],
  ['independent-portable-runner', 'sealed-final-runner-focused', './runner', 6],
]) {
  const receipt = JSON.parse(read(worker + '/' + original + '.json'));
  const selected = receipt.tests.filter(t => t.action === 'pass' && !t.test.includes('/')).map(t => t.test);
  assert.equal(selected.length, count);
  commands.push([label, go, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-timeout=2m', '-json', '-run', '^(' + selected.join('|') + ')$', path]);
}
commands.push(
  ['independent-empty-cache-normal', go, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-timeout=2m', '-json', '-run', '^TestRewrittenModulesRejectChangedIdentity/empty$', './deterministicio'],
  ['independent-empty-cache-mutant', go, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-timeout=2m', '-json', '-overlay', worker + '/control-adapter-current-overlay.json', '-run', '^TestRewrittenModulesRejectChangedIdentity/empty$', './deterministicio'],
  ['independent-completion-cause-mutant', go, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-timeout=2m', '-json', '-overlay', worker + '/control-completion-current-mutant-overlay.json', '-run', '^TestAssessCompletionProjectsCoverageInOrderAndClassifies$', './runner'],
  ['independent-world-seed-mutant', go, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-timeout=2m', '-json', '-overlay', worker + '/control-world-current-mutant-overlay.json', '-run', '^TestAssessWorldValidatesTheRecordAgainstItsSeed$', './runner'],
);
for (const [label, ...argv] of commands) {
  const result = spawnSync('node', [out + '/capture.mjs', label, ...argv], { cwd: root, env: process.env, stdio: 'inherit', timeout: 610000 });
  assert.equal(result.signal, null, label);
  assert.notEqual(result.status, null, label);
}
