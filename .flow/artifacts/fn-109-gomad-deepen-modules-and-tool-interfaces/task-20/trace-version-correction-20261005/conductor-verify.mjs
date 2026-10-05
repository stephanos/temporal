import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const area = dirname(fileURLToPath(import.meta.url));
const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const json = file => JSON.parse(readFileSync(resolve(area, file)));
const sha = bytes => createHash('sha256').update(bytes).digest('hex');
const before = json('before.json');
const final = json('final-source.json');
const paths = Object.keys(before.closure.files).sort();
assert.equal(paths.length, 1108);
assert.deepEqual(Object.keys(final.files).sort(), paths);
const batch = spawnSync('git', ['cat-file', '--batch'], { cwd: root, input: paths.map(path => before.base_commit + ':' + path).join('\n') + '\n', maxBuffer: 64 << 20 });
assert.equal(batch.status, 0);
let offset = 0;
const actual = {};
const changed = [];
for (const path of paths) {
  const end = batch.stdout.indexOf(10, offset);
  const fields = batch.stdout.subarray(offset, end).toString().split(' ');
  assert.equal(fields[1], 'blob', path);
  const size = Number(fields[2]);
  const baseBytes = batch.stdout.subarray(end + 1, end + 1 + size);
  assert.equal(sha(baseBytes), before.closure.files[path], path);
  offset = end + 2 + size;
  actual[path] = sha(readFileSync(resolve(root, path)));
  assert.equal(actual[path], final.files[path], path);
  if (actual[path] !== before.closure.files[path]) changed.push(path);
}
assert.deepEqual(changed, ['tools/gomad3/ARCHITECTURE.md', 'tools/gomad3/README.md', 'tools/gomad3/TUTORIAL.md']);
assert.equal(sha(JSON.stringify(actual)), final.sha256);
assert.equal(sha(JSON.stringify(Object.fromEntries(paths.map(path => [path, before.closure.files[path]])))), before.closure.sha256);
let commands = 0;
for (const phase of ['baseline', 'final', 'root']) {
  for (const receipt of json(phase + '-receipts.json')) {
    assert.equal(receipt.exit_code, 0);
    assert.equal(sha(readFileSync(resolve(area, receipt.log))), receipt.log_sha256);
    assert.equal(receipt.source_before, phase === 'baseline' ? before.closure.sha256 : final.sha256);
    assert.equal(receipt.source_after, receipt.source_before);
    assert.deepEqual([...receipt.expected_tests].sort(), [...receipt.passed_expected_tests].sort());
    assert.ok(Date.parse(receipt.finished_at) >= Date.parse(receipt.started_at));
    const events = readFileSync(resolve(area, receipt.log), 'utf8').split('\n').flatMap(line => {
      if (!line.startsWith('{')) return [];
      const event = JSON.parse(line);
      return event.Action ? [event] : [];
    });
    assert.equal(events.filter(event => event.Action === 'fail' || event.Action === 'skip').length, 0);
    for (const test of receipt.expected_tests) assert.equal(events.filter(event => event.Action === 'pass' && event.Test === test).length, 1);
    commands++;
  }
}
for (const [path, hash] of Object.entries(before.user_files)) assert.equal(sha(readFileSync(resolve(root, path))), hash);
const oldChecks = json('baseline-doc-check.json').checks;
assert.equal(oldChecks.filter(check => !check.passed).length, 6);
assert.ok(oldChecks.filter(check => !check.passed).every(check => check.name.startsWith('current-claim:')));
assert.ok(json('final-doc-check.json').checks.every(check => check.passed));
console.log(JSON.stringify({ base_git_blobs: paths.length, unchanged_product_paths: paths.length - changed.length, changed, final_sha256: final.sha256, verified_command_receipts: commands, baseline_stale_claims: 6, final_document_checks: json('final-doc-check.json').checks.length }, null, 2));
