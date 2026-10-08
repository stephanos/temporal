import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { root, worker, out, read } from './capture.mjs';
const json = p => JSON.parse(read(p));
const seal = json(resolve(worker, 'terminal-seal.json'));
assert.equal(seal.files.length, 150);
const testLabels = ['portable-private-operations', 'portable-api-external', 'portable-execution-controls', 'portable-minimizer-plan'];
const entries = testLabels.map(label => {
  const receipt = json(resolve(worker, label + '.json')), argv = [...receipt.argv];
  if (receipt.exit !== 0) {
    const names = receipt.tests.filter(e => e.action === 'pass' && !e.test.includes('/')).map(e => e.test);
    assert(names.length > 0);
    argv[argv.indexOf('-run') + 1] = '^(' + names.map(n => n.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|') + ')$';
  }
  return { label: 'independent-' + label, argv, expected: 0 };
});
for (const [label, expected] of [['external-negative-field', 1], ['external-negative-internal', 1], ['configured-generators', 0], ['configured-vet', 0], ['configured-errortype', 0], ['unfiltered-scoped-lint', 2], ['configured-fast-lint', 2], ['public-api-ast', 0], ['current-godoc', 0]]) entries.push({ label: 'independent-' + label, argv: json(resolve(worker, label + '.json')).argv, expected });
entries.push({ label: 'independent-whole-format', argv: json(resolve(worker, 'fmt-check.json')).argv, expected: 0 });
entries.push({ label: 'independent-task6-format', argv: json(resolve(worker, 'focused-format-check.json')).argv, expected: 0 });
for (const entry of entries) {
  if (existsSync(resolve(out, entry.label + '.json'))) {
    assert.equal(json(resolve(out, entry.label + '.json')).exit, entry.expected);
    continue;
  }
  const result = spawnSync('node', [fileURLToPath(new URL('./capture.mjs', import.meta.url)), entry.label, ...entry.argv], { cwd: root, stdio: 'inherit', timeout: 650000 });
  assert.equal(result.status, entry.expected, entry.label);
  assert.equal(json(resolve(out, entry.label + '.json')).exit, entry.expected);
}
