import fs from 'node:fs';
import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import {pathToFileURL} from 'node:url';

export function checkDelta(before, after, beforeWaivers, afterWaivers) {
  assert.ok(before.equals(after), 'raw IR differs: restoration permits no declaration, ordering, value or position delta');
  assert.ok(beforeWaivers.equals(afterWaivers), 'raw waivers differ: reasons and subjects must remain byte-identical');
  const model = JSON.parse(before), waivers = JSON.parse(beforeWaivers);
  const sha256 = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
  return {
    allowedDelta: [],
    boundary: 'independently captured accepted source versus actual compiled-rule restoration; legacy lift is refused and is not an equivalence baseline',
    irSha256: sha256(before), waiversSha256: sha256(beforeWaivers),
    sections: Object.fromEntries(Object.entries(model).filter(([, value]) => Array.isArray(value))
      .map(([name, value]) => [name, value.length])),
    waivers: waivers.waivers.length
  };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const [before, after, beforeWaivers, afterWaivers, ledger] = process.argv.slice(2);
  const result = checkDelta(...[before, after, beforeWaivers, afterWaivers].map(p => fs.readFileSync(p)));
  fs.writeFileSync(ledger, JSON.stringify(result, null, 2) + '\n');
  console.log('PASS: complete raw IR and waivers, including positions; no allowed delta');
}
