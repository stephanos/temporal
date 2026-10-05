import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const directory = dirname(fileURLToPath(import.meta.url));
const previous = readFileSync(resolve(directory, '../refresh-exhaustive-20261005/root-integrated-lint.stdout.log'), 'utf8');
const current = readFileSync(resolve(directory, 'root-integrated-lint.stdout.log'), 'utf8');
const header = /^tools\/gomad3\/.+\.go:\d+:\d+: .+$/;
const footer = /^\d+ issues:$/;
const blocks = log => {
  const lines = log.split('\n');
  const result = [];
  for (let index = 0; index < lines.length; index++) {
    if (!header.test(lines[index])) continue;
    let end = index + 1;
    while (end < lines.length && !header.test(lines[end]) && !footer.test(lines[end])) end++;
    result.push(lines.slice(index, end).join('\n').trimEnd());
    index = end - 1;
  }
  return result.sort();
};
const before = blocks(previous);
const after = blocks(current);
assert.equal(before.length, 317);
assert.equal(after.length, 317);
assert.equal(current.match(/^\d+ issues:$/m)?.[0], '317 issues:');
assert.deepEqual(after, before);
assert.match(current, /Lint module tools\/gomad3: 55 host packages/);
assert.match(current, /--fix=false/);
assert.match(current, /951c5516e9e7b3066e7e069adda9565cfd68844c/);
const categories = {};
for (const block of after) {
  const category = block.split('\n')[0].match(/\(([^()]*)\)$/)?.[1];
  assert.ok(category);
  categories[category] = (categories[category] || 0) + 1;
}
assert.deepEqual(categories, { forbidigo: 11, staticcheck: 52, errcheck: 252, exhaustive: 2 });
console.log(JSON.stringify({ diagnostic_blocks: after.length, categories, complete_blocks_equal: true, host_packages: 55, integrated_errortype: 'UNREACHED' }, null, 2));
