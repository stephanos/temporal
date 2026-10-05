import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const directory = dirname(fileURLToPath(import.meta.url));
const previous = resolve(directory, '../../task-9/adapter-integration-20261005/root-integrated-lint/stdout.txt');
const current = resolve(directory, 'root-integrated-lint/stdout.log');
const blocks = file => {
  const text = readFileSync(file, 'utf8');
  const matches = [...text.matchAll(/^tools\/gomad3\/.+\.go:\d+:\d+: .+$/gm)];
  const footer = text.search(/^\d+ issues:$/m);
  assert.ok(footer > 0);
  return matches.map((match, index) => text.slice(match.index, matches[index + 1]?.index ?? footer));
};
const before = blocks(previous);
const after = blocks(current);
const removed = before.filter(block => !after.includes(block));
const introduced = after.filter(block => !before.includes(block));
assert.equal(before.length, 319);
assert.equal(after.length, 318);
assert.equal(removed.length, 1);
assert.match(removed[0], /^tools\/gomad3\/internal\/sourceinventory\/inventory.go:82:10: QF1012:/);
assert.deepEqual(introduced, []);
assert.deepEqual(before.filter(block => block !== removed[0]), after);
const receipt = JSON.parse(readFileSync(resolve(directory, 'root-integrated-lint/receipt.json')));
assert.equal(receipt.exit_code, 2);
assert.equal(receipt.signal, null);
assert.equal(receipt.source_and_tool_inputs_stable, true);
const output = readFileSync(current, 'utf8');
assert.match(output, /Lint module tools\/gomad3: 55 host packages/);
assert.match(output, /--fix=false/);
assert.match(output, /--new-from-rev=951c5516e9e7b3066e7e069adda9565cfd68844c/);
assert.doesNotMatch(output, /go vet| -vettool=/);
const counts = {};
for (const block of after) {
  const linter = block.split('\n')[0].match(/\(([^()]*)\)$/)[1];
  counts[linter] = (counts[linter] ?? 0) + 1;
}
console.log(JSON.stringify({ previous, current, before: before.length, after: after.length, removed: removed[0], introduced, unchanged_complete_blocks: after.length, counts, integrated_errortype: 'UNREACHED', make_exit: receipt.exit_code, elapsed_seconds: receipt.elapsed_seconds }, null, 2));
