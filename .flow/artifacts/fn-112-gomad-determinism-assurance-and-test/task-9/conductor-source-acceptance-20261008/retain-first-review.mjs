import assert from 'node:assert/strict';
import { writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { out, read, sha } from './capture.mjs';

const path = '/tmp/impl-review-receipt-657da2bc4466-fn-112-gomad-determinism-assurance-and-test.9.json';
const receipt = JSON.parse(read(path));
assert.equal(receipt.verdict, 'NEEDS_WORK');
assert.equal(receipt.base, 'a7d99f4e64990839f81df4acd9bc150b842f8ea6');
assert.equal(receipt.findings.headSha, '7fee2ee6ba7e0e60212be65e600ff515cd533fa8');
assert.equal(receipt.rid, '551ca2a6ed8a4550bff5ce8f340d6aaa');
assert.equal(receipt.draws.length, 3);
assert.equal(receipt.introduced_count, 1);
const directory = '.flow/review-fanout/' + receipt.rid;
const paths = [path, directory + '/meta.json', ...['correctness', 'contracts', 'integration'].flatMap(axis => [directory + '/' + axis + '.json', directory + '/' + axis + '.review.md'])];
const records = paths.map(path => {
  const bytes = read(path), data = bytes.toString('base64');
  assert.deepEqual(Buffer.from(data, 'base64'), bytes);
  return { path, bytes: bytes.length, sha256: sha(bytes), encoding: 'base64', data };
});
writeFileSync(resolve(out, 'first-source-review.json'), JSON.stringify({ task: receipt.id, rid: receipt.rid, reviewed_base: receipt.base, reviewed_head: receipt.findings.headSha, verdict: receipt.verdict, records }, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify({ rid: receipt.rid, verdict: receipt.verdict, records: records.length }));
