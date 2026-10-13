import assert from 'node:assert/strict';
import fs from 'node:fs';
import crypto from 'node:crypto';
import { pathToFileURL } from 'node:url';

const beforeName = 'scala.Option[framework.Domain$package$.UpTo]';
const afterName = 'scala.Option[framework.Domain$package$.UpTo[2]]';
const allowed = new Set([
  '/types/2/name',
  '/functions/14/body/match/cases/0/pattern/literal/enum/type',
  '/functions/14/body/match/cases/1/pattern/case/type',
  '/functions/59/body/match/cases/0/body/construct/type',
  '/functions/59/body/match/cases/1/body/construct/type'
]);

export function checkMini(before, after) {
  const changes = [];
  function compare(a, b, path) {
    if (a && b && typeof a === 'object' && typeof b === 'object') {
      assert.deepEqual(Object.keys(a), Object.keys(b), `shape at ${path}`);
      for (const key of Object.keys(a)) compare(a[key], b[key], `${path}/${key}`);
    } else if (allowed.has(path)) {
      assert.equal(a, beforeName, `before identity at ${path}`);
      assert.equal(b, afterName, `after identity at ${path}`);
      changes.push({ path, before: a, after: b });
    } else assert.deepEqual(a, b, `unpermitted change at ${path}`);
  }
  compare(before, after, '');
  assert.deepEqual(new Set(changes.map(c => c.path)), allowed);
  assert.deepEqual(after.types[2].enum.cases[1].fields[0].type.intRange, { high: '2' });
  return changes;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const [beforePath, afterPath, ledgerPath] = process.argv.slice(2);
  const beforeBytes = fs.readFileSync(beforePath), afterBytes = fs.readFileSync(afterPath);
  const changes = checkMini(JSON.parse(beforeBytes), JSON.parse(afterBytes));
  const digest = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
  fs.writeFileSync(ledgerPath, JSON.stringify({ beforePath, afterPath,
    beforeSha256: digest(beforeBytes), afterSha256: digest(afterBytes), changes }, null, 2) + '\n');
  console.log('PASS: exactly five typed optional identity/reference changes; all ranges, bodies, ordinary identities, positions and Query metadata unchanged');
}
