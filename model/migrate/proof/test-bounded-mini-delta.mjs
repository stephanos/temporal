import assert from 'node:assert/strict';
import fs from 'node:fs';
import { checkMini } from './check-bounded-mini-delta.mjs';

const [beforePath, afterPath] = process.argv.slice(2);
const before = JSON.parse(fs.readFileSync(beforePath));
const after = JSON.parse(fs.readFileSync(afterPath));
checkMini(before, after);
const corruptions = [
  ir => { ir.types[2].enum.cases[1].fields[0].type.intRange.high = '3'; },
  ir => { ir.types[0].name = 'wrong ordinary identity'; },
  ir => { ir.types[2].position.line += 1; },
  ir => { ir.functions[59].body.match.cases[0].body.construct.type = 'wrong optional identity'; },
  ir => { ir.queries[0].limits.steps += 1; },
  ir => { ir.queries[0].total = '1'; },
  ir => { ir.functions[0].body.corrupt = true; }
];
for (const corrupt of corruptions) {
  const changed = structuredClone(after);
  corrupt(changed);
  assert.throws(() => checkMini(before, changed));
}
assert.throws(() => checkMini(before, before));
console.log('PASS: eight independent range, identity, reference, position, budget, total, body and no-op corruptions refused');
